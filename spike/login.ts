// SPDX-License-Identifier: GPL-3.0-or-later
// C25: the engine's login with CLAUDE_CONFIG_DIR set and no TTY, in the relay's shape
// (docs/design/07-spike-plan.md, C25; the owner runs it, since the code is theirs).
//
//   CLAUDE_CONFIG_DIR=<cfg> node login.js [--console] [--slash] [--spawn bwrap] [--engine <path>] [-- <args>]
//
// `-- <args>` replaces `auth login` verbatim (a stand-in engine for testing the relay plumbing).
// Default: spawns `<engine> auth login` with stdin, stdout and stderr piped, prints what the engine
// prints, then reads the pasted code from its own prompt and writes it to the engine's stdin.
// --console  the retry when the pipe needs a TTY: stdout and stderr inherit this console, only
//            stdin is piped. This is the spike's flag; it is never passed to the engine, whose own
//            `auth login --console` means Anthropic Console (API billing) instead.
// --slash    `<engine> /login` instead of `auth login` (for an engine without that subcommand).
// --spawn bwrap  the droplet relay shape: through bwrap.sh --cred sub.
// After the engine exits: its exit code, `auth status --text` against the same directory, and the
// directory's listing (names and sizes only). Never prints a file's contents.

import { spawn, spawnSync } from 'node:child_process';
import { readdirSync, statSync } from 'node:fs';
import { homedir } from 'node:os';
import { join, resolve } from 'node:path';
import { createInterface } from 'node:readline';
import { BF, childEnv, defaultEngine, die, fwd, parseFlags, str } from './common.js';

async function main(): Promise<void> {
  const f = parseFlags(process.argv.slice(2), { values: ['engine', 'spawn'], booleans: ['console', 'slash'] });
  const cfg = process.env.CLAUDE_CONFIG_DIR;
  if (!cfg) die('login: set CLAUDE_CONFIG_DIR to the spike config directory ($S/cfg); the owner\'s ~/.claude is never used');
  if (resolve(cfg).toLowerCase() === resolve(homedir(), '.claude').toLowerCase()) die('login: CLAUDE_CONFIG_DIR is ~/.claude; refusing');
  const engine = fwd(str(f, 'engine') ?? defaultEngine());
  const sub = f.rest.length ? f.rest : f.slash === true ? ['/login'] : ['auth', 'login'];
  const bw = str(f, 'spawn') === 'bwrap';
  const [cmd, args] = bw ? ['sh', [`${BF}/bwrap.sh`, '--cred', 'sub', '--', engine, ...sub]] : [engine, sub];
  const env = childEnv({ CLAUDE_CONFIG_DIR: fwd(cfg), DISABLE_AUTOUPDATER: '1' });
  const consoleMode = f.console === true;

  process.stderr.write(`login: ${[cmd, ...args].join(' ')} (CLAUDE_CONFIG_DIR=${fwd(cfg)}, stdio ${consoleMode ? 'stdin piped, stdout/stderr this console' : 'all piped'})\n`);
  const child = spawn(cmd, args, {
    env,
    stdio: consoleMode ? ['pipe', 'inherit', 'inherit'] : ['pipe', 'pipe', 'pipe'],
    windowsHide: true,
  });

  let prompted = false;
  let exited = false; // no paste prompt once the engine is gone
  const rl = createInterface({ input: process.stdin });
  const ask = () => {
    if (prompted || exited) return;
    prompted = true;
    process.stderr.write('\nlogin: paste the code from the browser here and press Enter: ');
  };
  rl.on('line', (line) => {
    if (child.stdin?.writable) child.stdin.write(line + '\n');
    process.stderr.write('login: code written to the engine\'s stdin\n');
  });
  rl.on('close', () => child.stdin?.end());

  const seenUrl = (c: Buffer) => /https?:\/\//.test(c.toString('utf8'));
  child.stdout?.on('data', (c: Buffer) => {
    process.stdout.write(c);
    if (seenUrl(c)) setTimeout(ask, 500);
  });
  child.stderr?.on('data', (c: Buffer) => {
    process.stderr.write(c);
    if (seenUrl(c)) setTimeout(ask, 500);
  });
  // In --console mode the URL goes straight to the console; ask after a moment either way.
  const askTimer = setTimeout(ask, consoleMode ? 3000 : 15000);

  const code = await new Promise<number | null>((r) => child.on('close', (c) => r(c)));
  exited = true;
  clearTimeout(askTimer);
  rl.close();
  process.stdin.destroy();
  process.stderr.write(`\nlogin: engine exited ${code}\n`);

  if (!bw) {
    // Not logged in is exit 1 with the reason on stdout, so print both streams whatever the code.
    const st = spawnSync(engine, ['auth', 'status', '--text'], { env, timeout: 30000, encoding: 'utf8', windowsHide: true });
    if (st.error) process.stderr.write(`login: auth status failed: ${st.error.message}\n`);
    else {
      process.stderr.write(`login: auth status exit ${st.status ?? st.signal}\n`);
      if (st.stdout.trim()) process.stderr.write(`login: auth status stdout: ${st.stdout.trim()}\n`);
      if (st.stderr.trim()) process.stderr.write(`login: auth status stderr: ${st.stderr.trim()}\n`);
    }
    try {
      for (const name of readdirSync(cfg)) {
        const s = statSync(join(cfg, name));
        process.stderr.write(`login: ${s.isDirectory() ? 'dir ' : 'file'} ${name}${s.isDirectory() ? '' : ` ${s.size} bytes`}\n`);
      }
    } catch (e) {
      process.stderr.write(`login: cannot list ${fwd(cfg)}: ${(e as Error).message}\n`);
    }
  }
  process.exitCode = code ?? 1;
}

main().catch((e: unknown) => die(`login: ${(e as Error).message}`));
