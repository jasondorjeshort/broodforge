#!/usr/bin/env node
// SPDX-License-Identifier: GPL-3.0-or-later
// PreToolUse hook for the Bash tool ("onFailure": "block" in .claude/settings.json).
// Denies destructive git commands, pushes from any subagent other than pusher, and
// commits from any subagent other than runner. The main session (no agent_type) is
// not restricted by caller. Spec: docs/design/decisions/0002-hook-specs.md section 1.
// Test harness: node .claude/hooks/test-hooks.js
//
// Allow: exit 0, no stdout. Deny: exit 0 with permissionDecision "deny" JSON.
// Fails closed: unparseable input or an exception also produces a deny.
'use strict';

// git options that come before the subcommand and take a separate value.
const GLOBAL_OPTS_WITH_VALUE = new Set([
  '-C', '-c', '--git-dir', '--work-tree', '--namespace', '--config-env', '--super-prefix',
]);
const SEPARATORS = new Set(['&&', '||', ';', '|', '&']);
const WHOLE_TREE = new Set(['.', './', '*', ':/']);

function deny(reason) {
  process.stdout.write(JSON.stringify({
    hookSpecificOutput: {
      hookEventName: 'PreToolUse',
      permissionDecision: 'deny',
      permissionDecisionReason: reason,
    },
  }));
  process.exit(0);
}
function allow() { process.exit(0); }

// Strip shell quoting and grouping characters from the ends of a token.
function clean(tok) {
  return tok.replace(/^[\s'"`$(){}]+/, '').replace(/[\s'"`(){};]+$/, '');
}
const isGit = (tok) => /(^|[\/\\])git(\.exe)?$/i.test(tok);
const isEnvAssign = (tok) => /^[A-Za-z_][A-Za-z0-9_]*=/.test(tok);
// A single-dash cluster of short options (-uf, -nd), not a long option or a bare "-".
const isShortCluster = (a) => /^-[A-Za-z0-9]+$/.test(a);
const clusterHas = (a, ch) => isShortCluster(a) && a.includes(ch);

// Every git invocation in a unit: { sub, args, first }. args stop at a standalone
// separator token; first is true when git is the unit's command (not an argument).
function invocations(unit) {
  const raw = unit.split(' ').filter(Boolean);
  const toks = raw.map(clean);
  const out = [];
  for (let i = 0; i < toks.length; i++) {
    if (!isGit(toks[i])) continue;
    let j = i + 1;
    while (j < toks.length && toks[j].startsWith('-') && !SEPARATORS.has(raw[j])) {
      if (GLOBAL_OPTS_WITH_VALUE.has(toks[j])) j++;
      j++;
    }
    if (j >= toks.length || SEPARATORS.has(raw[j])) continue;
    const sub = toks[j];
    const args = [];
    for (let k = j + 1; k < toks.length && !SEPARATORS.has(raw[k]); k++) args.push(toks[k]);
    let lead = 0;
    while (lead < i && isEnvAssign(toks[lead])) lead++;
    out.push({ sub, args, first: lead === i });
  }
  return out;
}

function pathspecs(args) {
  const dd = args.indexOf('--');
  return dd >= 0 ? args.slice(dd + 1) : args.filter((a) => !a.startsWith('-'));
}

// Ordered rules. test(inv, agent) returns true when the invocation must be denied.
const RULES = [
  {
    n: 1, prefix: 'force push',
    advice: 'The project never force-pushes; integrate the remote changes and push normally, or ask the user.',
    test: ({ sub, args }) => sub === 'push' && args.some((a) =>
      a.startsWith('--force') || a === '-f' || clusterHas(a, 'f') ||
      (a.startsWith('+') && a.length > 1)),
  },
  {
    n: 2, prefix: 'remote branch deletion',
    advice: 'Deleting or mirroring remote refs is for the user; ask the user to do it.',
    test: ({ sub, args }) => sub === 'push' && args.some((a) =>
      a === '--delete' || a === '-d' || clusterHas(a, 'd') || a === '--mirror' ||
      a === '--prune' || (a.startsWith(':') && a.length > 1)),
  },
  {
    n: 3, prefix: 'only the pusher agent pushes',
    advice: 'Leave the work committed and report; the coordinator launches the pusher.',
    test: ({ sub }, agent) => sub === 'push' && agent !== null && agent !== 'pusher',
  },
  {
    n: 4, prefix: 'only the runner agent commits; leave the tree for it',
    advice: 'Describe the change in your report so the runner can commit it.',
    test: ({ sub }, agent) => sub === 'commit' && agent !== null && agent !== 'runner',
  },
  {
    n: 5, prefix: 'destructive reset',
    advice: 'Use git reset --soft or --mixed, or git stash, so the work stays recoverable; ask the user if a hard reset is needed.',
    test: ({ sub, args }) => sub === 'reset' && args.some((a) => a === '--hard' || a === '--merge'),
  },
  {
    n: 6, prefix: 'git clean deletes untracked files',
    advice: 'Run git clean -n to list what would go, then delete named files or ask the user.',
    test: ({ sub, args }) => sub === 'clean' &&
      !args.some((a) => a === '-n' || a === '--dry-run' || clusterHas(a, 'n')),
  },
  {
    n: 7, prefix: 'whole-tree checkout discards edits',
    advice: 'Name the files to discard explicitly, or ask the user.',
    test: ({ sub, args }) => {
      if (sub !== 'checkout') return false;
      const dd = args.indexOf('--');
      const specs = dd >= 0 ? args.slice(dd + 1) : args.slice(-1);
      return specs.some((a) => WHOLE_TREE.has(a));
    },
  },
  {
    n: 8, prefix: 'whole-tree restore discards edits',
    advice: 'Name the files to restore explicitly, or use git restore --staged to unstage only.',
    test: ({ sub, args }) => {
      if (sub !== 'restore') return false;
      if (!pathspecs(args).some((a) => WHOLE_TREE.has(a))) return false;
      const staged = args.some((a) => a === '--staged' || clusterHas(a, 'S'));
      const worktree = args.some((a) => a === '--worktree' || clusterHas(a, 'W'));
      return !(staged && !worktree);
    },
  },
  {
    n: 9, prefix: 'forced branch delete',
    advice: 'Use git branch -d, which refuses to delete unmerged work, or ask the user.',
    test: ({ sub, args }) => {
      if (sub !== 'branch') return false;
      if (args.some((a) => clusterHas(a, 'D'))) return true;
      const del = args.some((a) => a === '--delete' || clusterHas(a, 'd'));
      const force = args.some((a) => a === '--force' || clusterHas(a, 'f'));
      return del && force;
    },
  },
  {
    n: 10, prefix: 'interactive rebase is unsupported here',
    advice: 'Interactive commands cannot run in this environment; use a non-interactive rebase or ask the user.',
    test: ({ sub, args }) => sub === 'rebase' &&
      args.some((a) => a === '-i' || a === '--interactive' || a.startsWith('--interactive=')),
  },
  {
    n: 11, prefix: 'stash drop/clear is unrecoverable',
    advice: 'Use git stash pop or git stash apply, or ask the user before discarding a stash.',
    test: ({ sub, args }) => {
      if (sub !== 'stash') return false;
      const action = args.find((a) => !a.startsWith('-'));
      return action === 'drop' || action === 'clear';
    },
  },
  {
    n: 12, prefix: 'history rewrite',
    advice: 'History rewrites are for the user; ask first.',
    test: ({ sub }) => sub === 'filter-branch' || sub === 'filter-repo' || sub === 'replace',
  },
];

const FALSE_POSITIVE_NOTE = ' The guard matches command text, so it also fires when a git command is only quoted (echo, a message); if nothing here runs git, rephrase the text.';

function decide(command, agent) {
  const norm = command.replace(/\r/g, '').replace(/[\t ]+/g, ' ');
  const segments = norm.split(/&&|\|\||;|\||\n/).map((s) => s.trim()).filter(Boolean);
  const whole = norm.replace(/\n/g, ' ; ').trim();
  // Segments first so the reported text is as narrow as possible.
  const units = segments.concat([whole]).map((u) => ({ text: u, invs: invocations(u) }));
  for (const rule of RULES) {
    for (const unit of units) {
      for (const inv of unit.invs) {
        if (rule.test(inv, agent)) {
          let reason = `git-guard: ${rule.prefix}: \`${unit.text}\`. ${rule.advice}`;
          if (!inv.first) reason += FALSE_POSITIVE_NOTE;
          return { rule: rule.n, reason };
        }
      }
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
    if (typeof command !== 'string') return allow();
    const agent = typeof input.agent_type === 'string' && input.agent_type !== '' ? input.agent_type : null;
    verdict = decide(command, agent);
  } catch (e) {
    const msg = e && e.message ? e.message : String(e);
    return deny(`git-guard: internal error (${msg}); command not run. Fix the hook or ask the user to run it.`);
  }
  if (verdict) return deny(verdict.reason);
  allow();
});
