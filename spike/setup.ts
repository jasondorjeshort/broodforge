// SPDX-License-Identifier: GPL-3.0-or-later
// Fixture setup for the spike (docs/design/07-spike-plan.md, "Fixture commits").
//
// Every fixture commit is made here through child_process (execFileSync, no shell), never from an
// agent's Bash tool and never inside `sh -c`: the git-guard hook denies `git commit` from agents,
// and these commits are scratch fixtures, not project commits.
//
//   node setup.js pc                       $S/tree, $S/canon, $S/cfg, $CFG/settings.json, hook mode allow
//   node setup.js tree                     $S/tree: git init -b main, one committed hello.txt
//   node setup.js canon [--droplet]        $S/canon (droplet: <root>/projects/spike): git init -b base, hello.txt
//   node setup.js commit <dir>             a change to hello.txt, then git commit -am t under the spike identity
//   node setup.js commit --droplet [<dir>] the same in <sess>/tree, through bwrap.sh --reach code
//   node setup.js settings [--out <file>] [--no-hook] [--sandbox] [--droplet]
//                                          settings.json from settings.template.json
//   node setup.js append <file>            appends one line (run inside the sandbox by commit --droplet)
//
// $S is BF_SPIKE when set, else $TEMP/bf-spike (PC) or <root>/sessions/<session> (droplet).

import { appendFileSync, existsSync, readFileSync, renameSync, writeFileSync } from 'node:fs';
import { dirname } from 'node:path';
import { BF, SPIKE_IDENTITY, die, dropletPaths, ensureDir, fwd, parseFlags, run, spikeRoot, str } from './common.js';

const gitEnv = { ...process.env, ...SPIKE_IDENTITY };
// `<bf>/share/empty/dir` must be truly empty (bound over the loaded-list directories), so git
// cannot carry it; every setup step creates it, and bwrap.sh does the same before it binds it.
const EMPTY_DIR = `${BF}/share/empty/dir`;
// The owner's global git config may sign commits or run hooks; neither belongs on a fixture.
const gitCfg = ['-c', 'commit.gpgsign=false', '-c', `core.hooksPath=${EMPTY_DIR}`];

function git(dir: string, args: string[]): string {
  return run('git', ['-C', dir, ...gitCfg, ...args], { env: gitEnv });
}

function initRepo(dir: string, branch: string): void {
  if (existsSync(dir)) die(`setup: ${dir} exists; delete it first (fixtures are made once)`);
  ensureDir(dir);
  git(dir, ['init', '-q', '-b', branch]);
  writeFileSync(`${dir}/hello.txt`, 'hello\n');
  git(dir, ['add', 'hello.txt']);
  git(dir, ['commit', '-q', '-m', 'spike fixture: hello.txt']);
  process.stdout.write(`setup: ${dir} on ${branch}: ${git(dir, ['log', '--oneline', '-1']).trim()}\n`);
}

function settings(f: ReturnType<typeof parseFlags>): void {
  const droplet = f.droplet === true;
  const cfg = droplet ? dropletPaths().cfg : `${spikeRoot()}/cfg`;
  const out = str(f, 'out') ?? (droplet ? dropletPaths().settings : `${cfg}/settings.json`);
  const tpl = JSON.parse(readFileSync(`${BF}/settings.template.json`, 'utf8').replace(/<bf>/g, BF)) as Record<string, unknown>;
  for (const k of Object.keys(tpl)) if (k.startsWith('//')) delete tpl[k];
  if (f['no-hook'] === true) delete tpl.hooks;
  if (f.sandbox === true) {
    // ADR 0022's block, written once C33 is run. The shape, `credentials.files` entries included,
    // is the one the Claude Code settings reference documents; the engine's schema validation
    // reports a wrong shape all the same.
    tpl.sandbox = {
      enabled: true,
      failIfUnavailable: true,
      allowUnsandboxedCommands: false,
      credentials: { files: [{ path: `${cfg}/.credentials.json`, mode: 'deny' }] },
    };
  }
  ensureDir(dirname(out));
  const tmp = `${out}.tmp`;
  writeFileSync(tmp, JSON.stringify(tpl, null, 2) + '\n');
  renameSync(tmp, out); // temp file and rename, as the companion writes <settings>
  process.stdout.write(`setup: wrote ${fwd(out)}${f['no-hook'] === true ? ' (no hooks)' : ''}${f.sandbox === true ? ' (sandbox block)' : ''}\n`);
}

function commitPc(dir: string): void {
  appendFileSync(`${dir}/hello.txt`, 'spike\n');
  git(dir, ['commit', '-q', '-am', 't']);
  process.stdout.write(`setup: ${dir}: ${git(dir, ['log', '-1', '--format=%h %an <%ae> / %cn <%ce> %s']).trim()}\n`);
}

function commitDroplet(dir: string): void {
  const d = dropletPaths();
  const bw = (args: string[]) =>
    run('sh', [`${BF}/bwrap.sh`, '--reach', 'code', '--', ...args], { env: gitEnv, stdio: ['ignore', 'pipe', 'pipe'] });
  bw([`${d.nodeDir}/bin/node`, `${BF}/setup.js`, 'append', `${dir}/hello.txt`]);
  bw(['git', '-C', dir, ...gitCfg, 'commit', '-q', '-am', 't']);
  process.stdout.write(bw(['git', '-C', dir, 'log', '-1', '--format=%h %an <%ae> / %cn <%ce> %s']));
}

function main(): void {
  const f = parseFlags(process.argv.slice(2), { values: ['out'], booleans: ['droplet', 'no-hook', 'sandbox'] });
  const [step, arg] = f._;
  const S = spikeRoot(f.droplet === true);
  ensureDir(EMPTY_DIR);
  switch (step) {
    case 'pc': {
      ensureDir(S);
      initRepo(`${S}/tree`, 'main');
      initRepo(`${S}/canon`, 'base');
      ensureDir(`${S}/cfg`);
      settings(f);
      const mode = `${S}/spike-hook.json`;
      if (!existsSync(mode)) writeFileSync(mode, JSON.stringify({ mode: 'allow' }) + '\n');
      process.stdout.write(`setup: $S is ${S}; hook mode file ${mode}\n`);
      break;
    }
    case 'tree':
      if (f.droplet === true) die('setup: on the droplet the tree is worktree-create (bwrap.sh --maintenance)');
      ensureDir(S);
      initRepo(`${S}/tree`, 'main');
      break;
    case 'canon':
      if (f.droplet === true) {
        const canon = dropletPaths().canon;
        initRepo(canon, 'base');
        run('chgrp', ['-R', 'broodforge-engine', canon]);
        run('chmod', ['-R', 'u+rwX,g+rX,g-w,o-rwx', canon]);
      } else {
        ensureDir(S);
        initRepo(`${S}/canon`, 'base');
      }
      break;
    case 'commit':
      if (f.droplet === true) commitDroplet(arg ?? dropletPaths().tree);
      else if (arg) commitPc(arg);
      else die('usage: node setup.js commit <dir> | commit --droplet [<dir>]');
      break;
    case 'settings':
      settings(f);
      break;
    case 'append':
      if (!arg) die('usage: node setup.js append <file>');
      appendFileSync(arg, 'spike\n');
      break;
    default:
      die('usage: node setup.js pc|tree|canon|commit|settings|append (see the header of setup.ts)');
  }
}

try {
  main();
} catch (e) {
  die(`setup: ${(e as Error).message}`);
}
