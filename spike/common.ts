// SPDX-License-Identifier: GPL-3.0-or-later
// Shared names for the spike harness (docs/design/07-spike-plan.md, "Shared setup").
// Throwaway: deleted in the commit that lands `engine run --real`.

import { execFileSync, type ExecFileSyncOptions } from 'node:child_process';
import { existsSync, mkdirSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

export const IS_WIN = process.platform === 'win32';

/** The engine version the spike was written against; `run.js` reports match or differs. */
export const EXPECTED_PIN = '2.1.296';

/** `<bf>`: the directory holding the compiled spike (this file's directory). */
export const BF = fwd(dirname(fileURLToPath(import.meta.url)));

/** Forward slashes on both hosts (settings files, hook command, printed paths). */
export function fwd(p: string): string {
  return p.replace(/\\/g, '/');
}

/** `$S`: `$TEMP/bf-spike` on the PC, `<root>/sessions/<session>` on the droplet. `BF_SPIKE` overrides. */
export function spikeRoot(droplet = false): string {
  if (process.env.BF_SPIKE) return fwd(process.env.BF_SPIKE);
  if (droplet) return dropletPaths().sess;
  const tmp = process.env.TEMP ?? (process.env.LOCALAPPDATA ? join(process.env.LOCALAPPDATA, 'Temp') : '/tmp');
  return fwd(join(tmp, 'bf-spike'));
}

/** The companion's names on the droplet; `BF_ROOT`, `BF_SESSION`, `BF_ENGINE_DIR`, `BF_NODE_DIR` override. */
export function dropletPaths() {
  const root = process.env.BF_ROOT ?? '/srv/broodforge';
  const session = process.env.BF_SESSION ?? 'spike';
  const sess = `${root}/sessions/${session}`;
  return {
    root,
    session,
    sess,
    cfg: `${root}/harness/spike`,
    canon: `${root}/projects/spike`,
    settings: `${root}/settings/spike.json`,
    tree: `${sess}/tree`,
    hookLog: `${sess}/home/hook.log`,
    engineDir: process.env.BF_ENGINE_DIR ?? '/opt/broodforge/engine',
    nodeDir: process.env.BF_NODE_DIR ?? '/opt/broodforge/node',
  };
}

/** The spike identity the four `GIT_*` names carry on every engine process and fixture commit. */
export const SPIKE_IDENTITY: Record<string, string> = {
  GIT_AUTHOR_NAME: 'Broodforge Spike',
  GIT_AUTHOR_EMAIL: 'spike@broodforge.invalid',
  GIT_COMMITTER_NAME: 'Broodforge Spike',
  GIT_COMMITTER_EMAIL: 'spike@broodforge.invalid',
};

/**
 * The default engine: the native `claude.exe` the global npm package ships, never the `.cmd` shim
 * (ADR 0020, Mechanism). On Linux the droplet's `<engine>/claude`.
 */
export function defaultEngine(): string {
  if (process.env.BF_ENGINE) return fwd(process.env.BF_ENGINE);
  if (IS_WIN) {
    const appData = process.env.APPDATA ?? join(process.env.USERPROFILE ?? 'C:/', 'AppData', 'Roaming');
    return fwd(join(appData, 'npm', 'node_modules', '@anthropic-ai', 'claude-code', 'bin', 'claude.exe'));
  }
  return `${dropletPaths().engineDir}/claude`;
}

/** Inherited names that survive the strip; everything else matching `CLAUDE*` or `ANTHROPIC_*` goes. */
const KEEP_INHERITED = new Set(['CLAUDE_CODE_GIT_BASH_PATH']);

/**
 * Names an engine session sets for its own children (a coder's Bash tool, the hook): a spike run
 * started from inside an agent session must not hand them on. Dropped unless the plan sets them.
 */
const SESSION_NAMES = new Set([
  'AI_AGENT', 'GIT_EDITOR', 'TRACEPARENT', 'GIT_CONFIG_PARAMETERS', 'TMPDIR', 'COREPACK_ENABLE_AUTO_PIN',
  'WT_SESSION', 'MSYSTEM', 'SHELL',
]);

/**
 * The child environment: the parent's, minus every inherited `CLAUDE*` and `ANTHROPIC_*` name except
 * `CLAUDE_CODE_GIT_BASH_PATH` (and `ANTHROPIC_API_KEY` when `keepApiKey`), minus `SESSION_NAMES`,
 * plus `set` (which may put any stripped name back).
 */
export function childEnv(set: Record<string, string>, keepApiKey = false): NodeJS.ProcessEnv {
  const env: NodeJS.ProcessEnv = {};
  for (const [k, v] of Object.entries(process.env)) {
    if (/^(CLAUDE|ANTHROPIC_)/i.test(k) && !KEEP_INHERITED.has(k) && !(keepApiKey && k === 'ANTHROPIC_API_KEY')) continue;
    if (SESSION_NAMES.has(k.toUpperCase())) continue; // Windows names are case-insensitive
    env[k] = v;
  }
  return { ...env, ...set };
}

/** Names only, sorted, for the record: never values. Every name unless `re` narrows it. */
export function envNames(env: NodeJS.ProcessEnv, re?: RegExp): string[] {
  return Object.keys(env).filter((k) => env[k] !== undefined && (!re || re.test(k))).sort();
}

/** Run a program without a shell; throws with stderr on a non-zero exit. */
export function run(cmd: string, args: string[], opts: ExecFileSyncOptions = {}): string {
  try {
    return String(execFileSync(cmd, args, { encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'], ...opts }));
  } catch (e) {
    const err = e as { status?: number; stderr?: Buffer | string; message: string };
    throw new Error(`${cmd} ${args.join(' ')}: exit ${err.status ?? '?'}: ${String(err.stderr ?? err.message).trim()}`);
  }
}

export function ensureDir(d: string): void {
  if (!existsSync(d)) mkdirSync(d, { recursive: true });
}

export interface Flags {
  _: string[];
  /** Everything after a bare `--`. */
  rest: string[];
  [k: string]: string | boolean | string[] | undefined;
}

/**
 * `--name value` and `--name` (boolean when listed in `booleans`); repeatable names in `multi`
 * collect into arrays. Unknown flags are an error, so a typo never silently changes a run.
 */
export function parseFlags(argv: string[], known: { values: string[]; booleans?: string[]; multi?: string[] }): Flags {
  const out: Flags = { _: [], rest: [] };
  const bools = new Set(known.booleans ?? []);
  const multi = new Set(known.multi ?? []);
  const vals = new Set([...known.values, ...multi]);
  for (let i = 0; i < argv.length; i++) {
    const a = argv[i];
    if (a === '--') {
      out.rest = argv.slice(i + 1);
      break;
    }
    if (a.startsWith('--')) {
      const name = a.slice(2);
      if (bools.has(name)) {
        out[name] = true;
      } else if (vals.has(name)) {
        if (i + 1 >= argv.length) throw new Error(`--${name} needs a value`);
        const v = argv[++i];
        if (multi.has(name)) out[name] = [...((out[name] as string[] | undefined) ?? []), v];
        else out[name] = v;
      } else {
        throw new Error(`unknown flag --${name}`);
      }
    } else {
      out._.push(a);
    }
  }
  return out;
}

export function str(f: Flags, name: string): string | undefined {
  const v = f[name];
  return typeof v === 'string' ? v : undefined;
}

export function die(msg: string, code = 1): never {
  process.stderr.write(`${msg}\n`);
  process.exit(code);
}
