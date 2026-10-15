// SPDX-License-Identifier: GPL-3.0-or-later
// The spike's engine runner: the miniature of 05's `engine run --real` and `engine record`
// (docs/design/07-spike-plan.md, "Every run" and "The code: spike/").
//
//   node run.js --out <dir> --prompt <text> [flags]      one engine process, one turn
//   node run.js --out <dir> --resume last|<id> ...        resume (last: the session id of the latest turn in <dir>)
//   node run.js --replay <turn.jsonl | fixture dir>       parse a recording, no engine
//   node run.js --version-only                            print `claude --version` and the pin check
//   node run.js ... -- <args>                             engine args replace $FLAGS verbatim (stand-in tests)
//
// Writes under --out: turn-<n>.jsonl (stdout verbatim: the fixture), events-<n>.jsonl (EngineEvent),
// stderr-<n>.txt, meta-<n>.json (version, env names, outcome, time, usage delta, kill record).
// Prints EngineEvent JSONL on stdout and `run: ...` summary lines on stderr.

import { spawn, type ChildProcess } from 'node:child_process';
import { createWriteStream, existsSync, readFileSync, readdirSync, statSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import {
  BF, EXPECTED_PIN, IS_WIN, SPIKE_IDENTITY, childEnv, defaultEngine, die, dropletPaths, ensureDir, envNames, fwd,
  parseFlags, run, spikeRoot, str,
} from './common.js';

// ---------------------------------------------------------------------------------------------
// EngineEvent, as 03-architecture.md "The engine adapter" types it.

type ModelUsage = Record<string, unknown>;
type Block = { kind: 'text'; text: string } | { kind: 'tool-use'; id: string; name: string; input: unknown };
export type EngineEvent =
  | { type: 'init'; engineSessionId: string; model: string; tools: string[]; capabilities: string[] }
  | {
      type: 'assistant'; messageId: string; model: string; parentToolUseId?: string; blocks: Block[];
      usage: { input: number; cacheRead: number; cacheWrite: number };
    }
  | { type: 'tool-result'; toolUseId: string; isError: boolean; text: string }
  | {
      type: 'result'; stopReason: string; isError: boolean; numTurns: number;
      deferred?: { toolUseId: string; name: string; input: unknown };
      permissionDenials: unknown[]; modelUsage: Record<string, ModelUsage>; usage: unknown;
    }
  | { type: 'exit'; code: number | null; signal: string | null; interrupted: boolean }
  | { type: 'unknown'; raw: string };

/** What a replay or a run noticed beyond the events: absent fields 03 reads, unknown kinds, dropped blocks. */
export interface ParseNotes {
  missing: Set<string>; // "<kind>.<field>"
  unknownKinds: Map<string, number>; // "<type>/<subtype>" -> count
  droppedBlocks: Map<string, number>; // assistant content block types not in 03's subset
  controlRequests: number;
}

export function newNotes(): ParseNotes {
  return { missing: new Set(), unknownKinds: new Map(), droppedBlocks: new Map(), controlRequests: 0 };
}

type J = Record<string, unknown>;
const isObj = (v: unknown): v is J => typeof v === 'object' && v !== null && !Array.isArray(v);
const num = (v: unknown): number => (typeof v === 'number' ? v : 0);
const bump = (m: Map<string, number>, k: string) => m.set(k, (m.get(k) ?? 0) + 1);

function need(o: J, field: string, kind: string, notes: ParseNotes): unknown {
  const path = field.split('.');
  let cur: unknown = o;
  for (const p of path) cur = isObj(cur) ? cur[p] : undefined;
  if (cur === undefined) notes.missing.add(`${kind}.${field}`);
  return cur;
}

function unknown(line: string, o: J | null, notes: ParseNotes): EngineEvent {
  const kind = o ? `${String(o.type)}${o.subtype !== undefined ? '/' + String(o.subtype) : ''}` : '(not JSON)';
  bump(notes.unknownKinds, kind);
  return { type: 'unknown', raw: line };
}

/** One stream-json stdout line to zero or more EngineEvents (03's subset; everything else `unknown`). */
export function parseLine(line: string, notes: ParseNotes): EngineEvent[] {
  if (line.trim() === '') return [];
  let o: J;
  try {
    const v: unknown = JSON.parse(line);
    if (!isObj(v)) return [unknown(line, null, notes)];
    o = v;
  } catch {
    return [unknown(line, null, notes)];
  }
  if (o.type === 'control_request') notes.controlRequests++;

  if (o.type === 'system' && o.subtype === 'init') {
    const k = 'init';
    return [{
      type: 'init',
      engineSessionId: String(need(o, 'session_id', k, notes) ?? ''),
      model: String(need(o, 'model', k, notes) ?? ''),
      tools: (need(o, 'tools', k, notes) as string[] | undefined) ?? [],
      capabilities: (need(o, 'capabilities', k, notes) as string[] | undefined) ?? [],
    }];
  }

  if (o.type === 'assistant') {
    const k = 'assistant';
    const msg = isObj(o.message) ? o.message : {};
    const id = need(o, 'message.id', k, notes);
    const model = need(o, 'message.model', k, notes);
    const u = need(o, 'message.usage', k, notes);
    const usage = isObj(u) ? u : {};
    const blocks: Block[] = [];
    for (const b of Array.isArray(msg.content) ? msg.content : []) {
      if (!isObj(b)) continue;
      if (b.type === 'text') blocks.push({ kind: 'text', text: String(b.text ?? '') });
      else if (b.type === 'tool_use') blocks.push({ kind: 'tool-use', id: String(b.id ?? ''), name: String(b.name ?? ''), input: b.input });
      else bump(notes.droppedBlocks, String(b.type));
    }
    const ev: EngineEvent = {
      type: 'assistant',
      messageId: String(id ?? ''),
      model: String(model ?? ''),
      blocks,
      usage: {
        input: num(usage.input_tokens),
        cacheRead: num(usage.cache_read_input_tokens),
        cacheWrite: num(usage.cache_creation_input_tokens),
      },
    };
    if (typeof o.parent_tool_use_id === 'string') ev.parentToolUseId = o.parent_tool_use_id;
    return [ev];
  }

  if (o.type === 'user') {
    const msg = isObj(o.message) ? o.message : {};
    const out: EngineEvent[] = [];
    for (const b of Array.isArray(msg.content) ? msg.content : []) {
      if (!isObj(b) || b.type !== 'tool_result') continue;
      const c = b.content;
      const text = typeof c === 'string'
        ? c
        : Array.isArray(c) ? c.filter((x) => isObj(x) && x.type === 'text').map((x) => String((x as J).text ?? '')).join('\n') : '';
      if (b.tool_use_id === undefined) notes.missing.add('tool-result.tool_use_id');
      out.push({ type: 'tool-result', toolUseId: String(b.tool_use_id ?? ''), isError: b.is_error === true, text });
    }
    return out.length ? out : [unknown(line, o, notes)];
  }

  if (o.type === 'result') {
    const k = 'result';
    const stopReason = String(need(o, 'stop_reason', k, notes) ?? '');
    const ev: EngineEvent = {
      type: 'result',
      stopReason,
      isError: need(o, 'is_error', k, notes) === true,
      numTurns: num(need(o, 'num_turns', k, notes)),
      permissionDenials: (need(o, 'permission_denials', k, notes) as unknown[] | undefined) ?? [],
      modelUsage: (need(o, 'modelUsage', k, notes) as Record<string, ModelUsage> | undefined) ?? {},
      usage: need(o, 'usage', k, notes),
    };
    const d = o.deferred_tool_use;
    if (isObj(d)) ev.deferred = { toolUseId: String(d.id ?? d.tool_use_id ?? ''), name: String(d.name ?? ''), input: d.input };
    else if (stopReason === 'tool_deferred') notes.missing.add('result.deferred_tool_use');
    return [ev];
  }

  return [unknown(line, o, notes)];
}

// ---------------------------------------------------------------------------------------------
// Usage delta (03, "Usage delta"): cumulative per engine session; the turn's record is the
// cumulative value minus the rows already written for the session, clamped at zero. The session is
// found by the `--resume` id, never by the new init's `session_id`, which may differ.

const NOT_SUMMED = new Set(['contextWindow', 'maxOutputTokens']);

export function usageDelta(cur: Record<string, ModelUsage>, prev: Record<string, ModelUsage> | undefined) {
  const delta: Record<string, Record<string, number>> = {};
  for (const [model, u] of Object.entries(cur)) {
    delta[model] = {};
    for (const [f, v] of Object.entries(u)) {
      if (typeof v !== 'number' || NOT_SUMMED.has(f)) continue;
      delta[model][f] = Math.max(0, v - num(prev?.[model]?.[f]));
    }
  }
  // cumulative: no field of any model went down, and something went up
  let grew = false;
  let shrank = false;
  for (const [model, u] of Object.entries(prev ?? {})) {
    for (const [f, v] of Object.entries(u)) {
      if (typeof v !== 'number' || NOT_SUMMED.has(f)) continue;
      const c = num(cur[model]?.[f]);
      if (c > v) grew = true;
      if (c < v) shrank = true;
    }
  }
  const cumulative = prev ? (shrank ? 'no (a field went down)' : grew ? 'consistent' : 'no growth') : 'no previous result';
  return { delta, cumulative };
}

type Row = Record<string, Record<string, number>>;
type Seen = { model: string; usage: { input: number; cacheRead: number; cacheWrite: number } };

/** The row a turn without `result` writes (03): input and cache counts per model, deduplicated by message id; output unknown. */
export function interruptedRow(seen: Iterable<Seen>): Row {
  const row: Row = {};
  for (const { model, usage } of seen) {
    const r = (row[model || '(no model)'] ??= {});
    r.inputTokens = (r.inputTokens ?? 0) + usage.input;
    r.cacheReadInputTokens = (r.cacheReadInputTokens ?? 0) + usage.cacheRead;
    r.cacheCreationInputTokens = (r.cacheCreationInputTokens ?? 0) + usage.cacheWrite;
  }
  return row;
}

function sumRows(rows: Row[]): Row {
  const sum: Row = {};
  for (const row of rows) {
    for (const [model, u] of Object.entries(row)) {
      const s = (sum[model] ??= {});
      for (const [f, v] of Object.entries(u)) s[f] = (s[f] ?? 0) + v;
    }
  }
  return sum;
}

/** One earlier turn as the delta reads it; `cumulative` is its result's `modelUsage`, absent when it had none. */
interface ChainTurn { turn: number; sessionId?: string; resumedFrom?: string; row: Row; cumulative?: Record<string, ModelUsage> }

/** The turns of the session `resume` names, oldest first: each turn joins the chain of the id it resumed. */
function chainOf(prior: ChainTurn[], resume: string | undefined): ChainTurn[] {
  const byId = new Map<string, ChainTurn[]>();
  for (const t of [...prior].sort((a, b) => a.turn - b.turn)) {
    const chain = [...((t.resumedFrom && byId.get(t.resumedFrom)) || []), t];
    if (t.sessionId) byId.set(t.sessionId, chain);
  }
  return resume ? byId.get(resume) ?? [] : [];
}

/**
 * The turn's delta and the rule that produced it: the previous result's cumulative value when the
 * session's last turn ended in a result; the sum of the rows written (03) when it had none.
 */
export function turnDelta(cur: Record<string, ModelUsage>, chain: ChainTurn[]) {
  const last = chain[chain.length - 1];
  if (!last) return { ...usageDelta(cur, undefined), rule: 'first turn of the session: the cumulative value', against: null };
  if (last.cumulative) {
    return { ...usageDelta(cur, last.cumulative), rule: `previous result's cumulative (turn ${last.turn})`, against: last.turn };
  }
  const rule = `sum of the ${chain.length} rows written for the session (turn ${last.turn} had no result)`;
  return { ...usageDelta(cur, sumRows(chain.map((c) => c.row))), rule, against: last.turn };
}

// ---------------------------------------------------------------------------------------------
// Replay: a recorded turn file or fixture directory through the parser, deltas across turns.

function turnFiles(path: string): string[] {
  if (!statSync(path).isDirectory()) return [path];
  return readdirSync(path)
    .filter((f) => /^turn-\d+\.jsonl$/.test(f))
    .sort((a, b) => Number(a.match(/\d+/)![0]) - Number(b.match(/\d+/)![0]))
    .map((f) => join(path, f));
}

function printNotes(notes: ParseNotes): void {
  const fmt = (m: Map<string, number>) => [...m].map(([k, v]) => `${k} x${v}`).join(', ') || 'none';
  process.stderr.write(`run: unknown kinds: ${fmt(notes.unknownKinds)}\n`);
  process.stderr.write(`run: missing fields 03 reads: ${[...notes.missing].join(', ') || 'none'}\n`);
  process.stderr.write(`run: dropped assistant blocks: ${fmt(notes.droppedBlocks)}\n`);
}

/**
 * The id a recorded turn resumed: `resumedFrom` or `--resume` in its meta; no meta means the
 * fixture is one chain, so the previous turn's session; a meta without either is a fresh session.
 */
function replayResume(file: string, previous: string | undefined): string | undefined {
  const n = file.match(/turn-(\d+)\.jsonl$/)?.[1];
  const metaFile = n ? join(file, '..', `meta-${n}.json`) : '';
  if (!metaFile || !existsSync(metaFile)) return previous;
  const m = JSON.parse(readFileSync(metaFile, 'utf8')) as { resumedFrom?: string | null; command?: string[] };
  if (m.resumedFrom) return m.resumedFrom;
  const i = m.command?.indexOf('--resume') ?? -1;
  return i >= 0 ? m.command![i + 1] : undefined;
}

function replay(path: string): void {
  const prior: ChainTurn[] = [];
  let previous: string | undefined;
  for (const [i, file] of turnFiles(path).entries()) {
    const notes = newNotes();
    const turn = Number(file.match(/turn-(\d+)\.jsonl$/)?.[1] ?? i + 1);
    const resumedFrom = replayResume(file, previous);
    const chain = chainOf(prior, resumedFrom);
    let session: string | undefined;
    let result: Extract<EngineEvent, { type: 'result' }> | undefined;
    const seen = new Map<string, Seen>();
    process.stderr.write(`run: replay ${fwd(file)}${resumedFrom ? ` (resumes ${resumedFrom})` : ''}\n`);
    for (const line of readFileSync(file, 'utf8').split(/\r?\n/)) {
      for (const ev of parseLine(line, notes)) {
        process.stdout.write(JSON.stringify(ev) + '\n');
        if (ev.type === 'init') session = ev.engineSessionId;
        if (ev.type === 'assistant') seen.set(ev.messageId, { model: ev.model, usage: ev.usage });
        if (ev.type === 'result') result = ev;
      }
    }
    let row: Row;
    if (result) {
      const u = turnDelta(result.modelUsage, chain);
      process.stderr.write(`run: modelUsage delta ${JSON.stringify(u.delta)} (rule: ${u.rule}; cumulative: ${u.cumulative})\n`);
      row = u.delta;
    } else {
      row = interruptedRow(seen.values());
      process.stderr.write(`run: no result: interrupted row ${JSON.stringify(row)}\n`);
    }
    prior.push({ turn, sessionId: session, resumedFrom, row, cumulative: result?.modelUsage });
    previous = session ?? previous;
    printNotes(notes);
  }
}

// ---------------------------------------------------------------------------------------------
// Process trees and kills, per host.

interface WinProc { pid: number; ppid: number; name: string; created: string }

function ps(script: string): string {
  return run('powershell.exe', ['-NoProfile', '-NonInteractive', '-Command', script], { timeout: 30000 });
}

function winSnapshot(): WinProc[] {
  const out = ps(
    "Get-CimInstance Win32_Process | ForEach-Object { [pscustomobject]@{ pid = $_.ProcessId; ppid = $_.ParentProcessId; " +
      "name = $_.Name; created = $(if ($_.CreationDate) { $_.CreationDate.ToUniversalTime().ToString('o') } else { '' }) } } " +
      '| ConvertTo-Json -Compress',
  );
  return JSON.parse(out) as WinProc[];
}

/** The engine's tree, walked by ParentProcessId from its pid; a child must not predate its parent (pid reuse). */
function winTree(root: number): WinProc[] {
  const all = winSnapshot();
  const top = all.find((p) => p.pid === root);
  if (!top) return [];
  const tree = [top];
  for (let i = 0; i < tree.length; i++) {
    for (const p of all) {
      if (p.ppid === tree[i].pid && p.created >= tree[i].created && !tree.some((t) => t.pid === p.pid)) tree.push(p);
    }
  }
  return tree;
}

/** Two seconds after the kill: which recorded pids still exist (Get-Process -Id), and which are the same process. */
async function winSurvivors(tree: WinProc[]): Promise<{ pidExists: number[]; sameProcess: number[] }> {
  await new Promise((r) => setTimeout(r, 2000));
  if (tree.length === 0) return { pidExists: [], sameProcess: [] };
  const out = ps(
    `Get-Process -Id ${tree.map((p) => p.pid).join(',')} -ErrorAction SilentlyContinue | ForEach-Object { ` +
      // Get-Process sets a failing status when no pid exists, even with SilentlyContinue
      "\"$($_.Id) $(try { $_.StartTime.ToUniversalTime().ToString('o') } catch { '' })\" }; exit 0",
  );
  const pidExists: number[] = [];
  const sameProcess: number[] = [];
  for (const line of out.split(/\r?\n/).filter(Boolean)) {
    const [pid, started] = line.trim().split(' ');
    const rec = tree.find((p) => p.pid === Number(pid));
    pidExists.push(Number(pid));
    // CIM CreationDate and StartTime agree to the microsecond; compare to the second
    if (rec && started && rec.created.slice(0, 19) === started.slice(0, 19)) sameProcess.push(Number(pid));
  }
  return { pidExists, sameProcess };
}

/** Linux: breadth-first over `pgrep -P`, returning the chain from `root` to the first `target` (`claude`). */
function linuxChain(root: number, target = 'claude'): { pid: number; comm: string }[] {
  const comm = (pid: number) => {
    try {
      return readFileSync(`/proc/${pid}/comm`, 'utf8').trim();
    } catch {
      return '?';
    }
  };
  const parent = new Map<number, number>();
  const queue = [root];
  while (queue.length) {
    const pid = queue.shift()!;
    if (pid !== root && comm(pid) === target) {
      const chain = [pid];
      while (chain[0] !== root) chain.unshift(parent.get(chain[0])!);
      return chain.map((p) => ({ pid: p, comm: comm(p) }));
    }
    let kids: number[] = [];
    try {
      kids = run('pgrep', ['-P', String(pid)]).split('\n').filter(Boolean).map(Number);
    } catch {
      kids = []; // pgrep exits 1 when there are none
    }
    for (const k of kids) {
      parent.set(k, pid);
      queue.push(k);
    }
  }
  return [{ pid: root, comm: comm(root) }];
}

// ---------------------------------------------------------------------------------------------
// One run.

function nextTurn(out: string): number {
  const ns = existsSync(out) ? readdirSync(out).map((f) => f.match(/^turn-(\d+)\.jsonl$/)?.[1]).filter(Boolean).map(Number) : [];
  return ns.length ? Math.max(...ns) + 1 : 1;
}

interface Meta {
  turn: number; engineSessionId?: string; resumedFrom?: string | null; usageRow?: Row;
  resultModelUsage?: Record<string, ModelUsage>;
}

function priorMetas(out: string, before: number): Meta[] {
  if (!existsSync(out)) return [];
  return readdirSync(out)
    .map((f) => f.match(/^meta-(\d+)\.json$/))
    .filter((m): m is RegExpMatchArray => !!m && Number(m[1]) < before)
    .map((m) => JSON.parse(readFileSync(join(out, m[0]), 'utf8')) as Meta)
    .sort((a, b) => a.turn - b.turn);
}

/** `--version` with the spike's config directory, so the owner's `~/.claude` is never read. */
function engineVersion(engine: string, cfg: string): { raw: string; version: string | null; pin: string } {
  let raw: string;
  try {
    raw = run(engine, ['--version'], { env: childEnv({ CLAUDE_CONFIG_DIR: cfg, DISABLE_AUTOUPDATER: '1' }), timeout: 30000 }).trim();
  } catch (e) {
    raw = `unavailable: ${(e as Error).message}`;
  }
  const version = raw.match(/(\d+\.\d+\.\d+)/)?.[1] ?? null;
  return { raw, version, pin: version === EXPECTED_PIN ? `match ${EXPECTED_PIN}` : `differs from ${EXPECTED_PIN}` };
}

const say = (s: string) => process.stderr.write(`run: ${s}\n`);

/** Seconds the bwrap hard kill waits for the process to go before it records `kill failed`. */
const KILL_WAIT_S = 10;

/** Whether `p` settles within `ms`. */
function within(p: Promise<unknown>, ms: number): Promise<boolean> {
  return new Promise((r) => {
    const t = setTimeout(() => r(false), ms);
    void p.then(() => {
      clearTimeout(t);
      r(true);
    });
  });
}

async function main(): Promise<void> {
  const f = parseFlags(process.argv.slice(2), {
    values: [
      'out', 'prompt', 'resume', 'cfg', 'cwd', 'settings', 'setting-sources', 'model', 'permission-mode',
      'permission-prompts', 'kill-after', 'deadline', 'spawn', 'reach', 'cred', 'engine', 'mode', 'hook-log',
      'canonical', 'turn', 'replay', 'prompt-wait', 'interrupt-timeout', 'stdin-json',
    ],
    booleans: ['strace', 'version-only', 'empty-line'],
    multi: ['env', 'engine-arg'],
  });

  const replayPath = str(f, 'replay');
  if (replayPath) return replay(replayPath);

  const spawnKind = str(f, 'spawn') ?? 'same';
  if (spawnKind !== 'same' && spawnKind !== 'bwrap') die('run: --spawn same|bwrap');
  const bw = spawnKind === 'bwrap';
  const d = dropletPaths();
  const S = spikeRoot(bw);
  const engine = fwd(str(f, 'engine') ?? defaultEngine());
  const cred = str(f, 'cred') ?? 'sub';
  if (cred !== 'sub' && cred !== 'key') die('run: --cred sub|key');
  if (cred === 'key' && IS_WIN) die('run: --cred key is refused on the PC: without CLAUDE_CONFIG_DIR the engine would load the owner\'s ~/.claude');
  const cfg = fwd(str(f, 'cfg') ?? (bw ? d.cfg : `${S}/cfg`));

  const ver = engineVersion(engine, cfg);
  say(`engine ${engine}`);
  say(`version ${ver.raw} (pin: ${ver.pin})`);
  if (f['version-only'] === true) return;

  const out = str(f, 'out') ?? die('run: --out <dir> is required');
  ensureDir(out);
  const turn = Number(str(f, 'turn') ?? nextTurn(out));
  const reach = str(f, 'reach') ?? 'code';
  const mode = str(f, 'mode') ?? 'policy';
  const cwd = fwd(str(f, 'cwd') ?? (bw ? d.tree : `${S}/tree`));
  const hookLog = fwd(str(f, 'hook-log') ?? (bw ? d.hookLog : `${S}/hook.log`));
  const canonical = fwd(str(f, 'canonical') ?? (bw ? d.canon : `${S}/canon`));
  const deadline = Number(str(f, 'deadline') ?? 120);
  const killAfter = str(f, 'kill-after') !== undefined ? Number(str(f, 'kill-after')) : undefined;
  const promptWait = Number(str(f, 'prompt-wait') ?? 10);
  const interruptTimeout = Number(str(f, 'interrupt-timeout') ?? 20);

  let resume = str(f, 'resume');
  if (resume === 'last') {
    resume = priorMetas(out, turn).reverse().find((m) => m.engineSessionId)?.engineSessionId ?? die('run: no earlier turn in --out has a session id');
  }

  // Engine args: $FLAGS, or everything after `--` verbatim.
  let args: string[];
  if (f.rest.length) {
    args = f.rest;
  } else {
    args = [
      '-p', '--output-format', 'stream-json', '--verbose', '--input-format', 'stream-json',
      '--permission-mode', str(f, 'permission-mode') ?? 'default',
      '--setting-sources', str(f, 'setting-sources') ?? 'user',
      '--strict-mcp-config', '--mcp-config', `${BF}/share/empty/mcp.json`,
    ];
    const model = str(f, 'model') ?? 'haiku';
    if (model !== 'none') args.push('--model', model);
    if (str(f, 'permission-prompts')) args.push('--permission-prompts', str(f, 'permission-prompts')!);
    if (str(f, 'settings')) args.push('--settings', str(f, 'settings')!);
    if (resume) args.push('--resume', resume);
    args.push(...((f['engine-arg'] as string[] | undefined) ?? []));
  }

  // Environment: inherited CLAUDE* and ANTHROPIC_* stripped, then the plan's names.
  const set: Record<string, string> = {
    BROODFORGE_SESSION: bw ? d.session : 'spike',
    BROODFORGE_MODE: mode,
    BROODFORGE_REACH: reach,
    BROODFORGE_TREE: cwd,
    BROODFORGE_CANONICAL: canonical,
    BROODFORGE_HOOK_LOG: hookLog,
    CLAUDE_CODE_SUBPROCESS_ENV_SCRUB: '1',
    DISABLE_AUTOUPDATER: '1',
    ...SPIKE_IDENTITY,
  };
  if (cred === 'sub') set.CLAUDE_CONFIG_DIR = cfg;
  else if (!process.env.ANTHROPIC_API_KEY) die('run: --cred key needs ANTHROPIC_API_KEY in this environment');
  for (const kv of (f.env as string[] | undefined) ?? []) {
    const i = kv.indexOf('=');
    if (i < 1) die(`run: --env ${kv}: want NAME=value`);
    set[kv.slice(0, i)] = kv.slice(i + 1);
  }
  const env = childEnv(set, cred === 'key');

  let cmd = engine;
  let cmdArgs = args;
  if (bw) {
    if (IS_WIN) die('run: --spawn bwrap is the droplet route');
    cmd = 'sh';
    cmdArgs = [`${BF}/bwrap.sh`, '--reach', reach, '--cred', cred,
      ...(f.strace === true ? ['--strace', `${d.sess}/home/strace.log`] : []), '--', engine, ...args];
  } else if (f.strace === true) {
    die('run: --strace needs --spawn bwrap');
  }

  const files = {
    turn: join(out, `turn-${turn}.jsonl`),
    events: join(out, `events-${turn}.jsonl`),
    stderr: join(out, `stderr-${turn}.txt`),
    meta: join(out, `meta-${turn}.json`),
  };
  const rawOut = createWriteStream(files.turn);
  const evOut = createWriteStream(files.events);
  const errOut = createWriteStream(files.stderr);
  const notes = newNotes();

  // The fields every meta carries, the spawn failure's included.
  const metaBase = {
    turn,
    engine,
    engineVersion: ver.version,
    engineVersionRaw: ver.raw,
    pin: ver.pin,
    node: process.version,
    spawn: spawnKind,
    command: [cmd, ...cmdArgs],
    cwd,
    cfg: cred === 'sub' ? cfg : null,
    cred,
    resumedFrom: resume ?? null,
    envNames: envNames(env),
  };

  say(`turn ${turn} -> ${fwd(files.turn)}`);
  const started = Date.now();
  const child: ChildProcess = spawn(cmd, cmdArgs, {
    cwd: bw ? undefined : cwd,
    env,
    stdio: ['pipe', 'pipe', 'pipe'],
    windowsHide: true,
    detached: !IS_WIN, // own process group on Linux, so the deadline can kill the group
  });
  const spawnError = new Promise<Error>((r) => child.once('error', r));
  const pid = child.pid;
  if (pid === undefined) {
    // No process (a bad --engine or --cwd): the error arrives on the next tick; record it and stop.
    const err = await spawnError;
    await Promise.all([rawOut, evOut, errOut].map((s) => new Promise((r) => s.end(r))));
    const meta = { ...metaBase, startedAt: new Date(started).toISOString(), outcome: 'spawn failed', spawnError: err.message };
    writeFileSync(files.meta, JSON.stringify(meta, null, 2) + '\n');
    say(`spawn failed: ${err.message}`);
    say(`meta ${fwd(files.meta)}`);
    process.exit(1);
  }
  say(`pid ${pid}`);

  // Stdin: one stream-json user line with one text block, then closed at once (ADR 0020).
  const prompt = str(f, 'prompt');
  if (str(f, 'stdin-json') !== undefined) child.stdin!.write(str(f, 'stdin-json') + '\n');
  else if (f['empty-line'] === true) child.stdin!.write('\n');
  else if (prompt !== undefined) {
    child.stdin!.write(JSON.stringify({ type: 'user', message: { role: 'user', content: [{ type: 'text', text: prompt }] } }) + '\n');
  }
  child.stdin!.on('error', () => undefined); // a stand-in that ignores stdin may close it first
  child.stdin!.end();

  let lastLine = '';
  let buf = '';
  let sessionId: string | undefined;
  let result: Extract<EngineEvent, { type: 'result' }> | undefined;
  const models = new Set<string>();
  const assistantSeen = new Map<string, Seen>();
  const emit = (ev: EngineEvent) => {
    const s = JSON.stringify(ev) + '\n';
    evOut.write(s);
    process.stdout.write(s);
  };
  let promptTimer: NodeJS.Timeout | undefined;
  const handleLine = (line: string) => {
    if (line.trim() === '') return;
    lastLine = line;
    const before = notes.controlRequests;
    for (const ev of parseLine(line, notes)) {
      emit(ev);
      if (ev.type === 'init') {
        sessionId = ev.engineSessionId;
        say(`session ${sessionId}`);
      }
      if (ev.type === 'assistant') {
        if (ev.model) models.add(ev.model);
        assistantSeen.set(ev.messageId, { model: ev.model, usage: ev.usage });
      }
      if (ev.type === 'result') result = ev;
    }
    if (notes.controlRequests > before && !promptTimer && !result) {
      say(`control_request seen; killing in ${promptWait} s unless a result arrives`);
      promptTimer = setTimeout(() => void kill('prompt'), promptWait * 1000);
    }
  };
  child.stdout!.on('data', (c: Buffer) => {
    rawOut.write(c);
    buf += c.toString('utf8');
    let i: number;
    while ((i = buf.indexOf('\n')) >= 0) {
      handleLine(buf.slice(0, i).replace(/\r$/, ''));
      buf = buf.slice(i + 1);
    }
  });
  child.stderr!.on('data', (c: Buffer) => {
    errOut.write(c);
    process.stderr.write(c);
  });

  // Kills. Windows: record the tree by ParentProcessId, then taskkill /T /F (S2). Linux same-account:
  // SIGINT, then the group. Droplet: the companion's Interrupt steps (2-3 for --kill-after, 4 at the deadline).
  let killedBy: 'deadline' | 'kill-after' | 'prompt' | undefined;
  const killRecord: Record<string, unknown> = {};
  let killing: Promise<void> | undefined;
  const exited = new Promise<{ code: number | null; signal: string | null }>((r) =>
    child.on('close', (code, signal) => r({ code, signal })),
  );
  let closed = false;
  void exited.then(() => (closed = true));
  // Resolved when a hard kill could not end the process within KILL_WAIT_S; the run then records
  // `kill failed` and exits instead of waiting on `exited` for ever.
  let killFailed = false;
  let giveUp!: () => void;
  const gaveUp = new Promise<void>((r) => (giveUp = r));

  const hardKill = async () => {
    if (IS_WIN) {
      let tree: WinProc[] = [];
      try {
        tree = winTree(pid!);
      } catch (e) {
        killRecord.treeError = (e as Error).message; // still kill; the record says the walk failed
      }
      killRecord.tree = tree;
      say(`engine tree: ${tree.map((p) => `${p.pid} ${p.name}`).join(', ') || '(not recorded)'}`);
      try {
        killRecord.taskkill = run('taskkill', ['/T', '/F', '/PID', String(pid)]).trim();
      } catch (e) {
        killRecord.taskkill = (e as Error).message;
      }
      try {
        const s = await winSurvivors(tree);
        killRecord.survivors = s;
        say(`after 2 s: recorded pids still present ${JSON.stringify(s.pidExists)}, same process ${JSON.stringify(s.sameProcess)}`);
      } catch (e) {
        killRecord.survivors = `check failed: ${(e as Error).message}`;
        say(`survivor check failed: ${(e as Error).message}`);
      }
    } else if (bw) {
      // Step 4. sudo runs as root, so the bot's SIGKILL to the sudo pid fails (EPERM); then the outer
      // bwrap gets it as broodforge-engine (the sudoers kill line), and --die-with-parent ends the rest.
      let step4: string;
      let failed = false;
      try {
        process.kill(pid!, 'SIGKILL');
        step4 = 'SIGKILL to the sudo child';
      } catch (e) {
        const direct = `direct SIGKILL to sudo ${pid}: ${(e as Error).message}`;
        const chain = linuxChain(pid!, 'bwrap');
        killRecord.step4Chain = chain;
        const outer = chain.length > 1 ? chain[chain.length - 1] : undefined;
        if (!outer) {
          step4 = `${direct}; no bwrap under the sudo pid`;
          failed = true;
        } else {
          try {
            run('sudo', ['-n', '-u', 'broodforge-engine', '/usr/bin/kill', '-KILL', String(outer.pid)]);
            step4 = `${direct}; SIGKILL to outer bwrap ${outer.pid} as broodforge-engine`;
          } catch (e2) {
            step4 = `${direct}; sudo kill -KILL ${outer.pid}: ${(e2 as Error).message}`;
            failed = true;
          }
        }
      }
      say(`step 4: ${step4}`);
      if (!(await within(exited, KILL_WAIT_S * 1000))) {
        const chain = (killRecord.step4Chain as { pid: number }[] | undefined) ?? [{ pid: pid! }];
        const present = chain.map((c) => c.pid).filter((p) => existsSync(`/proc/${p}`));
        killRecord.step4 = `kill failed: ${step4}; still running after ${KILL_WAIT_S} s (present: ${present.join(', ') || 'none'})`;
        say(String(killRecord.step4));
        killFailed = true;
        giveUp();
      } else {
        killRecord.step4 = failed ? `${step4}; exited anyway` : step4;
      }
    } else {
      try {
        process.kill(-pid!, 'SIGKILL');
      } catch (e) {
        killRecord.groupKill = `failed: ${(e as Error).message}`;
      }
    }
  };

  const kill = (why: 'deadline' | 'kill-after' | 'prompt'): Promise<void> => {
    if (closed || killing) return killing ?? Promise.resolve();
    killedBy = why;
    say(`${why}: killing engine pid ${pid}`);
    killing = (async () => {
      if (why === 'kill-after' && !IS_WIN) {
        if (bw) {
          const chain = linuxChain(pid!);
          killRecord.chain = chain;
          say(`comm chain: ${chain.map((c) => `${c.pid} ${c.comm}`).join(' > ')}`);
          const target = chain[chain.length - 1];
          if (target.comm === 'claude') {
            try {
              run('sudo', ['-n', '-u', 'broodforge-engine', '/usr/bin/kill', '-INT', String(target.pid)]);
              killRecord.sigint = target.pid;
            } catch (e) {
              killRecord.sigint = `failed: ${(e as Error).message}`;
            }
          }
        } else {
          process.kill(pid!, 'SIGINT');
          killRecord.sigint = pid;
        }
        const t = setTimeout(() => void hardKill(), interruptTimeout * 1000);
        await Promise.race([exited, gaveUp]);
        clearTimeout(t);
        return;
      }
      await hardKill();
    })();
    return killing;
  };

  const deadlineTimer = setTimeout(() => void kill('deadline'), deadline * 1000);
  const killTimer = killAfter !== undefined ? setTimeout(() => void kill('kill-after'), killAfter) : undefined;

  const ex = await Promise.race([exited, gaveUp.then(() => ({ code: null, signal: null }))]);
  clearTimeout(deadlineTimer);
  if (killTimer) clearTimeout(killTimer);
  if (promptTimer) clearTimeout(promptTimer);
  if (killing) await killing;
  if (killFailed) {
    // The process is still there: stop reading it so the files can close.
    for (const s of [child.stdout!, child.stderr!]) {
      s.removeAllListeners('data');
      s.destroy();
    }
    child.unref();
  }
  if (buf.trim()) handleLine(buf);
  const wall = (Date.now() - started) / 1000;
  emit({ type: 'exit', code: ex.code, signal: ex.signal, interrupted: killedBy !== undefined });
  await Promise.all([rawOut, evOut, errOut].map((s) => new Promise((r) => s.end(r))));

  let outcome: string;
  if (killFailed) outcome = `kill failed (${killedBy})`;
  else if (killedBy === 'prompt' || (killedBy === 'deadline' && notes.controlRequests > 0 && !result)) outcome = 'waits for a prompt';
  else if (killedBy === 'deadline') outcome = 'deadline';
  else if (killedBy === 'kill-after') outcome = result ? 'result after interrupt' : 'killed, no result';
  else outcome = result ? 'result' : 'exit without result';

  // The session is the one --resume named; its own init may carry a new id.
  const chain = chainOf(
    priorMetas(out, turn).map((m) => ({
      turn: m.turn, sessionId: m.engineSessionId, resumedFrom: m.resumedFrom ?? undefined,
      row: m.usageRow ?? {}, cumulative: m.resultModelUsage,
    })),
    resume,
  );
  const usage = result ? turnDelta(result.modelUsage, chain) : undefined;
  const usageRow = usage ? usage.delta : interruptedRow(assistantSeen.values());

  const meta = {
    ...metaBase,
    stdin: str(f, 'stdin-json') !== undefined ? 'stdin-json' : f['empty-line'] === true ? 'empty line' : prompt !== undefined ? 'prompt' : 'closed, nothing written',
    prompt,
    startedAt: new Date(started).toISOString(),
    wallSeconds: wall,
    deadlineSeconds: deadline,
    killAfterMs: killAfter ?? null,
    exit: ex,
    outcome,
    killedBy: killedBy ?? null,
    kill: killRecord,
    lastStdoutLine: lastLine,
    controlRequests: notes.controlRequests,
    engineSessionId: sessionId,
    models: result ? Object.keys(result.modelUsage) : [...models],
    stopReason: result?.stopReason,
    deferred: result?.deferred,
    permissionDenials: result?.permissionDenials,
    resultModelUsage: result?.modelUsage,
    resultUsage: result?.usage,
    usageDelta: usage?.delta,
    usageRule: usage ? usage.rule : 'no result: the interrupted row (assistant counts, output unknown)',
    usageDeltaAgainstTurn: usage?.against ?? null,
    usageRow,
    cumulative: usage?.cumulative,
    interruptedAssistantUsage: result ? undefined : Object.fromEntries(assistantSeen),
    unknownKinds: Object.fromEntries(notes.unknownKinds),
    missingFields: [...notes.missing],
    droppedBlocks: Object.fromEntries(notes.droppedBlocks),
  };
  writeFileSync(files.meta, JSON.stringify(meta, null, 2) + '\n');

  say(`outcome ${outcome}; exit ${ex.code ?? ''}${ex.signal ? ' ' + ex.signal : ''}; time ${wall.toFixed(1)} s`);
  if (killedBy) say(`last stdout line: ${lastLine.slice(0, 300) || '(none)'}`);
  if (result) say(`stop_reason ${result.stopReason}; models ${meta.models.join(', ')}`);
  if (usage) say(`modelUsage delta ${JSON.stringify(usage.delta)} (rule: ${usage.rule}; cumulative: ${usage.cumulative})`);
  else say(`no result: interrupted row ${JSON.stringify(usageRow)}`);
  printNotes(notes);
  say(`meta ${fwd(files.meta)}`);
  if (sessionId) say(`resume with: --out ${fwd(out)} --resume ${sessionId}`);
  if (killFailed) {
    say('kill failed: the engine may still be running; see kill.step4 in the meta');
    process.exit(1);
  }
}

main().catch((e: unknown) => die(`run: ${(e as Error).stack ?? String(e)}`));
