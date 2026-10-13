#!/usr/bin/env node
// SPDX-License-Identifier: GPL-3.0-or-later
// PreToolUse hook for the Bash and PowerShell tools ("onFailure": "block" in .claude/settings.json).
// Denies destructive git commands and hook bypasses for every caller, and limits who commits,
// pushes and deletes branches by the project's policy (.claude/hooks/policy.json, read through
// policy.js). The main session (no agent_type) is limited only where the policy says so.
// Spec: docs/design/decisions/0002-hook-specs.md section 1; policy: ADR 0006.
// Test harness: node .claude/hooks/test-hooks.js
//
// Allow: exit 0, no stdout. Deny: exit 0 with permissionDecision "deny" JSON.
// Its own failure (unparseable input, an exception, a missing module): exit 2 with the reason on
// stderr, which blocks with or without onFailure. A malformed policy file denies every git command.
'use strict';

function fail(e) {
  const msg = e && e.message ? e.message : String(e);
  process.stderr.write(`git-guard: internal error (${msg}); command not run. Fix the hook or ask the user to run it.\n`);
  process.exit(2);
}
process.on('uncaughtException', fail);

const path = require('path');
const { spawnSync } = require('child_process');
const { parse, SUBST } = require('./shell-parse');
const { load, matches } = require('./policy');

// git options that come before the subcommand and take a separate value.
const GLOBAL_OPTS_WITH_VALUE = new Set([
  '-C', '-c', '--git-dir', '--work-tree', '--namespace', '--config-env', '--super-prefix',
]);
// Per subcommand, the options that take a value: short letters, and long names taking the next
// word. Their values are dropped from args, so a message such as -m "-n" is not read as a flag.
const VALUE_OPTS = {
  commit: ['mFCct', ['--message', '--file', '--author', '--date', '--template', '--reuse-message',
    '--reedit-message', '--fixup', '--squash', '--cleanup', '--trailer', '--pathspec-from-file']],
  push: ['o', ['--push-option', '--repo', '--receive-pack', '--exec']],
  branch: ['u', ['--set-upstream-to', '--contains', '--no-contains', '--points-at', '--format', '--sort']],
  checkout: ['bB', ['--orphan']],
  switch: ['cC', ['--orphan']],
  restore: ['s', ['--source']],
  clean: ['e', ['--exclude']],
  stash: ['m', ['--message']],
  merge: ['msXF', ['--message', '--strategy', '--strategy-option', '--file']],
  rebase: ['xsX', ['--exec', '--onto', '--strategy', '--strategy-option']],
  worktree: ['bB', ['--reason']],
  config: ['f', ['--file', '--blob', '--type', '--default', '--comment']],
};
const WHOLE_TREE = new Set(['.', './', '*', ':/']);
// Words that run their arguments (or stdin) as a script. When one appears, quoted text in the same
// command line is read as commands too.
const RUNNERS = new Set(['sh', 'bash', 'zsh', 'dash', 'ksh', 'fish', 'pwsh', 'powershell', 'cmd', 'wsl',
  'eval', 'iex', 'invoke-expression', 'invoke-command', 'start-process', 'xargs', 'source', '.']);
// Words that may precede a command without making it an argument.
const PREFIX_WORDS = new Set(['sudo', 'command', 'exec', 'time', 'nohup', 'env', 'nice', '!', 'then', 'do',
  'else', 'if', 'elif', 'while', 'until']);
const NOTE = ' git-guard reads the command line, not what runs: it counted this git word as a command because' +
  ' it is unquoted or sits in text handed to a shell. If nothing here runs git, tell the user it is a false positive.';

const isGit = (w) => /(^|[\/\\])git(\.exe)?$/i.test(w);
const isEnvAssign = (w) => /^[A-Za-z_][A-Za-z0-9_]*=/.test(w);
const base = (w) => w.replace(/^.*[\/\\]/, '').replace(/\.exe$/i, '').toLowerCase();
const has = (args, ...flags) => args.some((a) => flags.includes(a));
const positionals = (args) => args.filter((a) => !a.startsWith('-'));
const refName = (r) => r.replace(/^refs\/heads\//, '');

function deny(reason) {
  process.stdout.write(JSON.stringify({
    hookSpecificOutput: { hookEventName: 'PreToolUse', permissionDecision: 'deny', permissionDecisionReason: reason },
  }));
  process.exit(0);
}

// Splits short clusters into single flags (-nm x becomes -n -m) and drops option values.
function normalize(sub, rest) {
  const [shortV, longV] = VALUE_OPTS[sub] || ['', []];
  const out = [];
  for (let k = 0; k < rest.length; k++) {
    const a = rest[k];
    if (a === '--') { out.push(...rest.slice(k)); break; }
    if (/^-[A-Za-z0-9]+$/.test(a)) {
      for (let c = 1; c < a.length; c++) {
        out.push(`-${a[c]}`);
        if (shortV.includes(a[c])) { if (c === a.length - 1) k++; break; }
      }
    } else {
      out.push(a);
      if (longV.includes(a)) k++;
    }
  }
  return out;
}

// Resolves a cd or -C target against dir; Git Bash /c/... becomes C:/... on Windows.
function resolveDir(dir, rel) {
  if (!rel || rel.includes(SUBST)) return dir;
  const p = rel.replace(/^~(?=$|[\/\\])/, process.env.HOME || process.env.USERPROFILE || '~');
  return path.resolve(dir, process.platform === 'win32' ? p.replace(/^\/([a-zA-Z])(\/|$)/, '$1:/') : p);
}

// The git invocations in one simple command: { sub, args, globals, dir, first, text }.
function invocations(cmd, dir) {
  const w = cmd.words;
  const out = [];
  for (let i = 0; i < w.length; i++) {
    if (!isGit(w[i])) continue;
    const globals = [];
    let d = dir;
    let j = i + 1;
    while (j < w.length && w[j].startsWith('-')) {
      if (GLOBAL_OPTS_WITH_VALUE.has(w[j])) {
        globals.push([w[j], w[j + 1] || '']);
        if (w[j] === '-C') d = resolveDir(d, w[j + 1]);
        j += 2;
      } else {
        const eq = w[j].indexOf('=');
        globals.push(eq > 0 ? [w[j].slice(0, eq), w[j].slice(eq + 1)] : [w[j], null]);
        j++;
      }
    }
    if (j >= w.length) continue;
    const lead = w.slice(0, i);
    out.push({
      sub: w[j], args: normalize(w[j], w.slice(j + 1)), globals, dir: d, text: cmd.text,
      first: !cmd.fromData && lead.every((x) => isEnvAssign(x) || PREFIX_WORDS.has(x)),
    });
  }
  return out;
}

// Current branch of the repository at dir, or null when unreadable ("HEAD" when detached).
const branchCache = new Map();
function currentBranch(dir) {
  if (!branchCache.has(dir)) {
    const r = spawnSync('git', ['-C', dir, 'rev-parse', '--abbrev-ref', 'HEAD'], { encoding: 'utf8', timeout: 5000 });
    branchCache.set(dir, !r.error && r.status === 0 ? r.stdout.trim() || null : null);
  }
  return branchCache.get(dir);
}

// The branches a push updates: { refs, all, unknown }. No refspec pushes the current branch.
function pushTargets(inv) {
  const words = [];
  let all = false;
  for (const a of inv.args) {
    if (has([a], '--all', '--branches', '--tags', '--mirror', '--follow-tags')) all = true;
    else if (!a.startsWith('-')) words.push(a);
  }
  const refs = [];
  let unknown = false;
  const here = () => {
    const b = currentBranch(inv.dir);
    if (!b || b === 'HEAD') unknown = true; else refs.push(b);
  };
  if (words.length <= 1) here();
  for (const s of words.slice(1)) {
    const spec = s.replace(/^\+/, '');
    const dst = spec.includes(':') ? spec.slice(spec.indexOf(':') + 1) : spec;
    if (dst === '') continue; // a deletion, rule 2
    if (dst === 'HEAD') here(); else refs.push(refName(dst));
  }
  return { refs, all, unknown };
}
const within = (t, patterns) => !t.all && !t.unknown && t.refs.length > 0 && t.refs.every((r) => matches(patterns, r));

// Remote branches a push deletes (--delete arguments or :ref).
function pushDeletions(inv) {
  const del = has(inv.args, '--delete', '-d');
  return positionals(inv.args).slice(1)
    .filter((w) => del || (w.startsWith(':') && w.length > 1))
    .map((w) => refName(w.replace(/^:/, '')));
}

// True when GitHub shows a merged pull request from this branch. Fails closed (false) when gh is
// missing, offline or not logged in. Test seam: GIT_GUARD_MERGED, comma-separated branch names,
// replaces the gh call (test-hooks.js sets it; unset in normal use).
const mergedCache = new Map();
function mergedOnGitHub(branch, dir) {
  if (process.env.GIT_GUARD_MERGED !== undefined) return process.env.GIT_GUARD_MERGED.split(',').map((x) => x.trim()).includes(branch);
  if (!mergedCache.has(branch)) {
    let merged = false;
    try {
      const r = spawnSync('gh', ['pr', 'list', '--head', branch, '--state', 'merged', '--json', 'number', '--limit', '1'],
        { cwd: dir, encoding: 'utf8', timeout: 20000 });
      if (!r.error && r.status === 0) { const l = JSON.parse(r.stdout || '[]'); merged = Array.isArray(l) && l.length > 0; }
    } catch (_) { merged = false; }
    mergedCache.set(branch, merged);
  }
  return mergedCache.get(branch);
}

// Rules 2 and 9: deletion is allowed only when every branch matches git.branchDelete.branches
// and, when the policy asks, GitHub shows its pull request merged. Returns false (allow) or a deny.
function deletable(names, dir, { policy }) {
  const d = policy.git.branchDelete;
  if (d.branches.length === 0) return true;
  if (names.length > 0 && names.every((b) => matches(d.branches, b) && (!d.requireMergedPr || mergedOnGitHub(b, dir)))) return false;
  return `Only branches matching ${d.branches.join(', ')} are deleted` +
    `${d.requireMergedPr ? ', and only after GitHub shows their pull request merged; a failed or pending merge keeps the branch' : ''}.`;
}

const isLease = (a) => a.startsWith('--force-with-lease') || a === '--force-if-includes';
const isPlainForce = (a) => a === '--force' || a.startsWith('--force=') || a === '-f' || (a.startsWith('+') && a.length > 1);
const listOf = (who) => (who === 'any' ? 'any subagent' : who === 'none' ? 'none' : who.join(', '));
const mayAct = (who, agent) => who === 'any' || (Array.isArray(who) && who.includes(agent));

// Config keys an invocation sets through -c, --config-env, or GIT_CONFIG_* in the command line.
function configKeys(inv, ctx) {
  const keys = [];
  for (const [o, v] of inv.globals) if ((o === '-c' || o === '--config-env') && v) keys.push(v);
  for (const e of ctx.envConfig) {
    const m = /^(?:\$env:)?GIT_CONFIG_(?:KEY_\d+|PARAMETERS)=(.*)$/i.exec(e);
    if (m) keys.push(...m[1].replace(/['"]/g, '').split(/\s+/));
  }
  return keys;
}
// For git config: { unset, value } of the first positional key matching re, or null.
function configWrite(inv, re) {
  const pos = positionals(inv.args);
  if (pos[0] === 'unset') return re.test(pos[1] || '') ? { unset: true } : null;
  if (pos[0] === 'set') pos.shift();
  const k = pos.findIndex((a) => re.test(a));
  if (k < 0) return null;
  const unset = has(inv.args, '--unset', '--unset-all', '--remove-section');
  return { unset, value: pos[k].includes('=') ? pos[k].slice(pos[k].indexOf('=') + 1) : pos[k + 1] };
}
const HOOKS_KEY = /^core\.hookspath(=|$)/i;
const ALIAS_KEY = /^alias\./i;

// Ordered rules. test(inv, ctx) returns false to pass, true to deny with the rule's advice, or a
// string to deny with that advice instead. ctx: { agent, policy, envConfig }.
const RULES = [
  {
    n: 1, prefix: 'force push',
    advice: 'The project does not force-push; integrate the remote changes and push normally, or ask the user.',
    test: (inv, { policy }) => {
      if (inv.sub !== 'push') return false;
      if (inv.args.some(isPlainForce)) return true;
      if (!inv.args.some(isLease)) return false;
      const ok = policy.git.push.forceWithLease;
      if (ok.length === 0) return true;
      return within(pushTargets(inv), ok) ? false : `--force-with-lease is allowed only onto branches matching ${ok.join(', ')}; ask the user for anything else.`;
    },
  },
  {
    n: 2, prefix: 'remote branch deletion',
    advice: 'Deleting or mirroring remote refs is for the user; ask the user to do it.',
    test: (inv, ctx) => {
      if (inv.sub !== 'push') return false;
      if (has(inv.args, '--mirror', '--prune')) return true;
      const dels = pushDeletions(inv);
      if (dels.length === 0 && !has(inv.args, '--delete', '-d')) return false;
      return deletable(dels, inv.dir, ctx);
    },
  },
  {
    n: 3, prefix: 'push not permitted for this caller',
    test: (inv, { agent, policy }) => {
      if (inv.sub !== 'push') return false;
      const p = policy.git.push;
      if (agent === null) return p.mainSession ? false : 'Under .claude/hooks/policy.json the main session does not push; ask the user.';
      if (!mayAct(p.subagents, agent)) return `Under .claude/hooks/policy.json the subagents that push are: ${listOf(p.subagents)}. Leave the work committed and report.`;
      if (p.branches.includes('*') || within(pushTargets(inv), p.branches)) return false;
      return `Subagents push only branches matching ${p.branches.join(', ')}; tags and other branches go through the main session.`;
    },
  },
  {
    n: 4, prefix: 'commit not permitted for this caller',
    test: (inv, { agent, policy }) => {
      if (inv.sub !== 'commit' || agent === null) return false;
      const c = policy.git.commit;
      if (!mayAct(c.subagents, agent)) return `Under .claude/hooks/policy.json the subagents that commit are: ${listOf(c.subagents)}. Describe the change in your report so it can be committed.`;
      if (c.protectedBranches.length === 0) return false;
      const b = currentBranch(inv.dir);
      return b !== null && matches(c.protectedBranches, b) ? `Subagents do not commit on ${b}; work on your own branch.` : false;
    },
  },
  {
    n: 5, prefix: 'destructive reset',
    advice: 'Use git reset --soft or --mixed, or git stash, so the work stays recoverable; ask the user if a hard reset is needed.',
    test: ({ sub, args }) => sub === 'reset' && has(args, '--hard', '--merge'),
  },
  {
    n: 6, prefix: 'git clean deletes untracked files',
    advice: 'List what would go with a dry run, then delete named files or ask the user.',
    test: ({ sub, args }) => sub === 'clean' && !has(args, '-n', '--dry-run'),
  },
  {
    n: 7, prefix: 'whole-tree checkout discards edits',
    advice: 'Name the files to discard explicitly, or ask the user.',
    test: ({ sub, args }) => {
      if (sub !== 'checkout') return false;
      const dd = args.indexOf('--');
      return (dd >= 0 ? args.slice(dd + 1) : args.slice(-1)).some((a) => WHOLE_TREE.has(a));
    },
  },
  {
    n: 8, prefix: 'whole-tree restore discards edits',
    advice: 'Name the files to restore explicitly, or use git restore --staged to unstage only.',
    test: ({ sub, args }) => {
      if (sub !== 'restore') return false;
      const dd = args.indexOf('--');
      if (!(dd >= 0 ? args.slice(dd + 1) : positionals(args)).some((a) => WHOLE_TREE.has(a))) return false;
      return !(has(args, '--staged', '-S') && !has(args, '--worktree', '-W'));
    },
  },
  {
    n: 9, prefix: 'forced branch delete',
    advice: 'Use git branch -d, which refuses to delete unmerged work, or ask the user.',
    test: (inv, ctx) => {
      if (inv.sub !== 'branch') return false;
      const forced = has(inv.args, '-D') || (has(inv.args, '--delete', '-d') && has(inv.args, '--force', '-f'));
      return forced ? deletable(positionals(inv.args), inv.dir, ctx) : false;
    },
  },
  {
    n: 10, prefix: 'interactive rebase is unsupported here',
    advice: 'Interactive commands cannot run in this environment; use a non-interactive rebase or ask the user.',
    test: ({ sub, args }) => sub === 'rebase' && args.some((a) => a === '-i' || a === '--interactive' || a.startsWith('--interactive=')),
  },
  {
    n: 11, prefix: 'stash drop/clear is unrecoverable',
    advice: 'Use git stash pop or git stash apply, or ask the user before discarding a stash.',
    test: ({ sub, args }) => sub === 'stash' && ['drop', 'clear'].includes(positionals(args)[0]),
  },
  {
    n: 12, prefix: 'history rewrite',
    advice: 'History rewrites and ref surgery are for the user; ask first.',
    test: ({ sub, args }) => ['filter-branch', 'filter-repo', 'replace', 'update-ref'].includes(sub) ||
      (sub === 'reflog' && ['expire', 'delete'].includes(positionals(args)[0])),
  },
  {
    n: 13, prefix: 'hooks bypass',
    advice: 'Never --no-verify. If a git hook fires, fix what it reports.',
    test: ({ sub, args }) => (has(args, '--no-verify') &&
      ['commit', 'push', 'merge', 'pull', 'rebase', 'am', 'cherry-pick', 'revert'].includes(sub)) ||
      (sub === 'commit' && has(args, '-n')),
  },
  {
    n: 14, prefix: 'hooks path override',
    test: (inv, ctx) => {
      const want = ctx.policy.git.hooksPath;
      const advice = want ? `core.hooksPath stays ${want}; the project's git hooks must run.` : 'core.hooksPath is not changed here; the git hooks must run.';
      if (configKeys(inv, ctx).some((k) => HOOKS_KEY.test(k))) return advice;
      if (inv.sub !== 'config') return false;
      const c = configWrite(inv, HOOKS_KEY);
      if (!c || (!c.unset && c.value === undefined)) return false; // a read
      if (!c.unset && want && c.value.replace(/\\/g, '/').replace(/\/$/, '') === want) return false;
      return advice;
    },
  },
  {
    n: 15, prefix: 'git alias definition',
    advice: 'An alias can wrap any command, so agents do not define one; run the git command itself.',
    test: (inv, ctx) => {
      if (configKeys(inv, ctx).some((k) => ALIAS_KEY.test(k))) return true;
      if (inv.sub !== 'config') return false;
      const c = configWrite(inv, ALIAS_KEY);
      return !!c && !c.unset && c.value !== undefined;
    },
  },
  {
    n: 16, prefix: 'forced checkout discards edits',
    advice: 'Commit or stash the edits first, or ask the user.',
    test: ({ sub, args }) => (sub === 'checkout' && has(args, '-f', '--force')) ||
      (sub === 'switch' && has(args, '-f', '--force', '--discard-changes')),
  },
  {
    n: 17, prefix: 'forced worktree removal',
    advice: 'Remove a clean worktree without --force, or ask the user; a forced removal deletes uncommitted work.',
    test: ({ sub, args }) => sub === 'worktree' && positionals(args)[0] === 'remove' && has(args, '-f', '--force'),
  },
];

// Every command the line runs, with the directory each runs in. Quoted text and heredoc bodies
// are read as commands too when a RUNNERS word appears (sh -c, eval, | bash, powershell -Command).
function commandsOf(command, dialect, cwd) {
  const top = parse(command, dialect);
  const cmds = top.commands.slice();
  const runner = (list) => list.some((c) => c.words.some((w) => RUNNERS.has(base(w))));
  let data = runner(cmds) ? top.data : [];
  for (let depth = 0; depth < 3 && data.length > 0; depth++) {
    const next = [];
    for (const text of data) {
      for (const d of ['bash', 'powershell']) {
        const p = parse(text, d);
        for (const c of p.commands) cmds.push(Object.assign(c, { fromData: true }));
        if (runner(p.commands)) next.push(...p.data);
      }
    }
    data = next;
  }
  let dir = cwd;
  return cmds.map((c) => {
    const at = dir;
    if (!c.fromData && ['cd', 'set-location', 'sl', 'chdir', 'pushd'].includes(c.words[0].toLowerCase())) dir = resolveDir(dir, c.words[1]);
    return { cmd: c, dir: at };
  });
}

function decide(command, dialect, agent, cwd, loaded) {
  const cmds = commandsOf(command, dialect, cwd);
  const invs = [];
  for (const { cmd, dir } of cmds) invs.push(...invocations(cmd, dir));
  if (invs.length === 0) return null;
  const clip = (t) => (t.length > 200 ? `${t.slice(0, 200)}...` : t);
  if (loaded.error) {
    return `git-guard: policy file is malformed (${loaded.error}): \`${clip(invs[0].text)}\`. Every git command is denied until it is fixed; ask the user.`;
  }
  const envConfig = [];
  for (const { cmd } of cmds) {
    cmd.words.forEach((w, k) => {
      if (/^(?:\$env:)?GIT_CONFIG_(KEY_\d+|PARAMETERS)/i.test(w)) envConfig.push(w.includes('=') ? w : `${w}=${cmd.words[k + 2] || ''}`);
    });
  }
  const ctx = { agent, policy: loaded.policy, envConfig };
  for (const rule of RULES) {
    for (const inv of invs) {
      const r = rule.test(inv, ctx);
      if (r) return `git-guard: ${rule.prefix}: \`${clip(inv.text)}\`. ${typeof r === 'string' ? r : rule.advice}${inv.first ? '' : NOTE}`;
    }
  }
  return null;
}

let raw = '';
process.stdin.setEncoding('utf8');
process.stdin.on('data', (c) => (raw += c));
process.stdin.on('end', () => {
  let verdict;
  try {
    const input = JSON.parse(raw);
    const command = input && input.tool_input && input.tool_input.command;
    if (typeof command !== 'string') process.exit(0);
    const agent = typeof input.agent_type === 'string' && input.agent_type !== '' ? input.agent_type : null;
    const cwd = typeof input.cwd === 'string' && input.cwd ? input.cwd : process.cwd();
    const dialect = input.tool_name === 'PowerShell' ? 'powershell' : 'bash';
    verdict = decide(command, dialect, agent, cwd, load());
  } catch (e) {
    fail(e);
  }
  if (verdict) deny(verdict);
  process.exit(0);
});
