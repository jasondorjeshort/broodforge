// SPDX-License-Identifier: GPL-3.0-or-later
// The spike's PreToolUse hook (docs/design/07-spike-plan.md, "The hook").
//
//   node hook.js                         the engine's call: JSON on stdin, decision on stdout
//   node hook.js mode <mode> [<id>:allow|deny]
//                                        writes spike-hook.json beside BROODFORGE_HOOK_LOG
//                                        (PC only; the droplet uses hookmode.sh)
//
// The behaviour comes from spike-hook.json in the directory of BROODFORGE_HOOK_LOG, never from a
// variable. Every error the hook can see prints `hook: <what>` on stderr and exits 2 (fail closed,
// 03-hook-contract.md "Failure"). The log line carries names, never values.

import { appendFileSync, readFileSync, writeFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fwd, spikeRoot } from './common.js';

const MODES = ['allow', 'deny', 'defer', 'exit1', 'exit2', 'sleep'] as const;
type Mode = (typeof MODES)[number];
const DENY_REASON = 'Denied by Broodforge: spike deny';
const SLEEP_SECONDS = 30;

interface ModeFile {
  mode: Mode;
  answer?: string;
}

class HookError extends Error {}

function logPath(): string {
  const p = process.env.BROODFORGE_HOOK_LOG;
  if (!p) throw new HookError('BROODFORGE_HOOK_LOG missing');
  return p;
}

function readMode(log: string): ModeFile {
  const file = join(dirname(log), 'spike-hook.json');
  let raw: string;
  try {
    raw = readFileSync(file, 'utf8');
  } catch {
    throw new HookError(`spike-hook.json missing beside BROODFORGE_HOOK_LOG`);
  }
  let m: ModeFile;
  try {
    m = JSON.parse(raw) as ModeFile;
  } catch {
    throw new HookError('spike-hook.json is not JSON');
  }
  if (!MODES.includes(m.mode)) throw new HookError(`spike-hook.json mode not one of ${MODES.join('|')}`);
  if (m.answer !== undefined && !/^[^:]+:(allow|deny)$/.test(m.answer)) {
    throw new HookError('spike-hook.json answer is not <tool_use_id>:allow|deny');
  }
  return m;
}

function readStdin(): Promise<string> {
  return new Promise((resolve, reject) => {
    const chunks: Buffer[] = [];
    process.stdin.on('data', (c: Buffer) => chunks.push(c));
    process.stdin.on('end', () => resolve(Buffer.concat(chunks).toString('utf8')));
    process.stdin.on('error', reject);
  });
}

function decision(d: 'allow' | 'deny' | 'defer', reason: string): string {
  return JSON.stringify({
    hookSpecificOutput: { hookEventName: 'PreToolUse', permissionDecision: d, permissionDecisionReason: reason },
  });
}

function writeLog(log: string, fields: Record<string, unknown>): void {
  const names = Object.keys(process.env)
    .filter((k) => /^(BROODFORGE_|CLAUDE_|ANTHROPIC_)/.test(k))
    .sort();
  try {
    appendFileSync(log, JSON.stringify({ t: new Date().toISOString(), pid: process.pid, ...fields, env: names }) + '\n');
  } catch {
    throw new HookError('cannot append to BROODFORGE_HOOK_LOG');
  }
}

async function hook(): Promise<number> {
  const log = logPath();
  const m = readMode(log);
  const raw = await readStdin();
  let input: { hook_event_name?: unknown; tool_name?: unknown; tool_use_id?: unknown };
  try {
    input = JSON.parse(raw) as typeof input;
  } catch {
    throw new HookError('stdin is not JSON');
  }
  if (typeof input.tool_name !== 'string') throw new HookError('stdin has no tool_name');
  const id = typeof input.tool_use_id === 'string' ? input.tool_use_id : '';

  let out: { stdout?: string; stderr?: string; code: number; decided: string; sleep?: number };
  const answer = m.answer?.split(':');
  if ((m.mode === 'allow' || m.mode === 'deny' || m.mode === 'defer') && answer && answer[0] === id) {
    out = answer[1] === 'allow'
      ? { stdout: decision('allow', 'spike answer allow'), code: 0, decided: 'answer:allow' }
      : { stdout: decision('deny', DENY_REASON), code: 0, decided: 'answer:deny' };
  } else if (m.mode === 'allow') {
    out = { stdout: decision('allow', 'spike allow'), code: 0, decided: 'allow' };
  } else if (m.mode === 'deny') {
    out = { stdout: decision('deny', DENY_REASON), code: 0, decided: 'deny' };
  } else if (m.mode === 'defer') {
    out = { stdout: decision('defer', 'spike defer'), code: 0, decided: 'defer' };
  } else if (m.mode === 'exit1') {
    out = { stderr: 'hook: spike exit 1', code: 1, decided: 'exit1' };
  } else if (m.mode === 'exit2') {
    out = { stderr: 'hook: spike exit 2', code: 2, decided: 'exit2' };
  } else {
    out = { stdout: decision('allow', 'spike sleep then allow'), code: 0, decided: 'sleep', sleep: SLEEP_SECONDS };
  }

  writeLog(log, {
    hook_event_name: typeof input.hook_event_name === 'string' ? input.hook_event_name : null,
    tool_name: input.tool_name,
    tool_use_id: id || null,
    mode: m.mode,
    decided: out.decided,
  });
  if (out.sleep) await new Promise((r) => setTimeout(r, out.sleep! * 1000));
  if (out.stdout) process.stdout.write(out.stdout + '\n');
  if (out.stderr) process.stderr.write(out.stderr + '\n');
  return out.code;
}

function setMode(args: string[]): number {
  const [mode, answer] = args;
  if (!MODES.includes(mode as Mode)) {
    process.stderr.write(`usage: node hook.js mode ${MODES.join('|')} [<tool_use_id>:allow|deny]\n`);
    return 1;
  }
  if (answer !== undefined && !/^[^:]+:(allow|deny)$/.test(answer)) {
    process.stderr.write('answer must be <tool_use_id>:allow|deny\n');
    return 1;
  }
  const log = process.env.BROODFORGE_HOOK_LOG ?? `${spikeRoot()}/hook.log`;
  const file = fwd(join(dirname(log), 'spike-hook.json'));
  const body: ModeFile = answer ? { mode: mode as Mode, answer } : { mode: mode as Mode };
  writeFileSync(file, JSON.stringify(body) + '\n');
  process.stdout.write(`${file}: ${JSON.stringify(body)}\n`);
  return 0;
}

async function main(): Promise<void> {
  const args = process.argv.slice(2);
  if (args[0] === 'mode') {
    process.exitCode = setMode(args.slice(1));
    return;
  }
  try {
    process.exitCode = await hook();
  } catch (e) {
    const what = e instanceof HookError ? e.message : `unexpected ${e instanceof Error ? e.name : 'error'}`;
    process.stderr.write(`hook: ${what}\n`);
    process.exitCode = 2;
  }
}

void main();
