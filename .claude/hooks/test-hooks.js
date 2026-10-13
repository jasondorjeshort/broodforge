#!/usr/bin/env node
// SPDX-License-Identifier: GPL-3.0-or-later
// Test harness and debug interface for the hooks in this directory.
// Spec: docs/design/decisions/0002-hook-specs.md. Each case runs a hook as a child
// process with JSON on stdin, exactly as Claude Code does, and checks exit code and stdout.
//
// Usage:
//   node .claude/hooks/test-hooks.js                  run every case; exit 1 on any failure
//   node .claude/hooks/test-hooks.js --only G0        run cases whose id starts with G0
//   node .claude/hooks/test-hooks.js --verbose        also print each hook's raw stdout
//   node .claude/hooks/test-hooks.js --keep           keep the scratch docs tree (path printed)
//   node .claude/hooks/test-hooks.js --input <file> <git-guard|post-edit|usage-gate>
//       run one hook on the raw contents of <file> and print exit code, stdout, stderr.
//       Use this to reproduce a bug: write the hook input JSON to a file with the Write
//       tool (a command line containing e.g. a force push would itself trip git-guard).
//
// Isolation: post-edit cases run against a scratch tree in the OS temp dir (deleted at
// the end); git-guard cases run in two scratch repositories (one on main, one on task/T1);
// usage-gate and statusline cases point HOME/USERPROFILE at a scratch dir. Every case names
// its policy file through HOOK_POLICY_FILE: the Hatchery fixture, an example, none, or a scratch
// file. No file of the repository is read except to check that policy.json loads, and ~/.claude is never touched. Exit codes: 0 all pass, 1 any
// failure, 2 usage error.
'use strict';
const fs = require('fs');
const os = require('os');
const path = require('path');
const { spawnSync } = require('child_process');

const HOOKS = __dirname;
const NODE = process.execPath;

// ---------- CLI ----------
const argv = process.argv.slice(2);
const opt = { only: null, verbose: false, keep: false, input: null, hook: null };
for (let i = 0; i < argv.length; i++) {
  const a = argv[i];
  if (a === '--verbose') opt.verbose = true;
  else if (a === '--keep') opt.keep = true;
  else if (a === '--only' && argv[i + 1]) opt.only = argv[++i];
  else if (a === '--input' && argv[i + 1] && argv[i + 2]) { opt.input = argv[++i]; opt.hook = argv[++i]; }
  else { process.stderr.write(`test-hooks: unknown or incomplete argument: ${a}\n`); process.exit(2); }
}

function run(hook, stdin, env) {
  const r = spawnSync(NODE, [path.join(HOOKS, `${hook}.js`)], {
    input: stdin, encoding: 'utf8', timeout: 30000, env: env || process.env,
  });
  return { code: r.status, stdout: r.stdout || '', stderr: r.stderr || '', error: r.error };
}

if (opt.input) {
  if (!['git-guard', 'post-edit', 'usage-gate'].includes(opt.hook)) {
    process.stderr.write(`test-hooks: unknown hook ${opt.hook}\n`);
    process.exit(2);
  }
  const r = run(opt.hook, fs.readFileSync(opt.input, 'utf8'));
  process.stdout.write(`exit ${r.code}\nstdout: ${r.stdout}\nstderr: ${r.stderr}\n`);
  process.exit(0);
}

// ---------- result bookkeeping ----------
let pass = 0;
let fail = 0;
function report(id, label, ok, detail, stdout) {
  if (opt.only && !id.startsWith(opt.only)) return;
  if (ok) pass++; else fail++;
  process.stdout.write(`${ok ? 'PASS' : 'FAIL'} ${id} ${label}\n`);
  if (!ok) process.stdout.write(`     ${detail}\n`);
  if ((opt.verbose || !ok) && stdout !== undefined) process.stdout.write(`     stdout: ${stdout || '(empty)'}\n`);
}
const selected = (id) => !opt.only || id.startsWith(opt.only);

function parseDecision(stdout) {
  try {
    const o = JSON.parse(stdout);
    const h = o.hookSpecificOutput || {};
    return { event: h.hookEventName, decision: h.permissionDecision, reason: h.permissionDecisionReason, context: h.additionalContext };
  } catch (_) { return null; }
}

const scratch = fs.mkdtempSync(path.join(os.tmpdir(), 'hooks-test-'));
const fwd = (p) => p.split(path.sep).join('/');

// ---------- policies ----------
// Every hook case names the policy file it runs under (HOOK_POLICY_FILE): the Hatchery fixture
// below, the two examples, none at all (the defaults), or a scratch file written for the case. The
// repository's own policy.json is never a fixture, so the suite passes unchanged in a project that
// copied it and set its own policy; LIVE01 only checks that the project's file loads.
const HATCHERY_POLICY = {
  git: {
    commit: { subagents: ['runner'], protectedBranches: [] },
    push: { subagents: ['pusher'], branches: ['*'], mainSession: true, forceWithLease: [] },
    branchDelete: { branches: [], requireMergedPr: false },
    hooksPath: null,
  },
  usage: { hold: 50, deny: 80, staleSeconds: 7200, whenStale: 'allow' },
  edit: { capExempt: [], anchorExempt: ['docs/design/proposals/*', 'docs/design/decisions/*', 'docs/findings/*'], formatters: {} },
};
fs.writeFileSync(path.join(scratch, 'policy-hatchery.json'), JSON.stringify(HATCHERY_POLICY));
const POLICIES = {
  hatchery: path.join(scratch, 'policy-hatchery.json'),
  taskBranches: path.join(HOOKS, 'policy.examples', 'task-branches.json'),
  singleBranch: path.join(HOOKS, 'policy.examples', 'single-branch.json'),
  none: path.join(scratch, 'no-such-policy.json'),
};
function scratchPolicy(name, obj) {
  const f = path.join(scratch, `policy-${name}.json`);
  fs.writeFileSync(f, typeof obj === 'string' ? obj : JSON.stringify(obj));
  return f;
}
const envWith = (extra) => {
  const env = Object.assign({}, process.env, extra);
  for (const k of Object.keys(env)) if (env[k] === undefined) delete env[k];
  return env;
};

// ---------- scratch git repositories (git-guard reads the current branch) ----------
function git(dir, ...args) {
  const r = spawnSync('git', ['-C', dir, '-c', 'user.name=hooks-test', '-c', 'user.email=hooks-test@example.invalid',
    '-c', 'commit.gpgsign=false', ...args], { encoding: 'utf8' });
  if (r.status !== 0) { process.stderr.write(`test-hooks: git ${args.join(' ')} failed: ${r.stderr}\n`); process.exit(2); }
}
const repoMain = path.join(scratch, 'repo-main');
const repoTask = path.join(scratch, 'repo-task');
for (const [dir, branch] of [[repoMain, 'main'], [repoTask, 'task/T1']]) {
  fs.mkdirSync(dir, { recursive: true });
  git(dir, 'init', '-q');
  git(dir, 'checkout', '-q', '-b', branch);
  git(dir, 'commit', '-q', '--allow-empty', '-m', 'init');
}

// ---------- git-guard ----------
const PREFIX = {
  1: 'force push',
  2: 'remote branch deletion',
  3: 'push not permitted for this caller',
  4: 'commit not permitted for this caller',
  5: 'destructive reset',
  6: 'git clean deletes untracked files',
  7: 'whole-tree checkout discards edits',
  8: 'whole-tree restore discards edits',
  9: 'forced branch delete',
  10: 'interactive rebase is unsupported here',
  11: 'stash drop/clear is unrecoverable',
  12: 'history rewrite',
  13: 'hooks bypass',
  14: 'hooks path override',
  15: 'git alias definition',
  16: 'forced checkout discards edits',
  17: 'forced worktree removal',
  18: 'forced branch reset',
  19: 'unexpanded argument',
  policy: 'policy file is malformed (',
};
const NOTE_MARK = 'git-guard reads the command line, not what runs';
const withNote = (d) => d.reason.includes(NOTE_MARK) || 'reason lacks the note about command text';
const noNote = (d) => !d.reason.includes(NOTE_MARK) || 'reason carries the note, but git is the command here';

// Runs git-guard once. o: { command, agent, tool, cwd, policy, raw, merged }. expect: 'allow',
// 'internal' (exit 2, reason on stderr), a rule number, or 'policy' (malformed policy deny).
function guardCase(id, o, expect, extra) {
  if (!selected(id)) return;
  const input = { hook_event_name: 'PreToolUse', tool_name: o.tool || 'Bash', cwd: o.cwd || repoMain, tool_input: { command: o.command } };
  if (o.agent !== null && o.agent !== undefined) input.agent_type = o.agent;
  const stdin = o.raw !== undefined && o.raw !== null ? o.raw : JSON.stringify(input);
  // task/T1 has a merged pull request, task/T2 does not (git-guard's GIT_GUARD_MERGED seam).
  const r = run('git-guard', stdin, envWith({ HOOK_POLICY_FILE: o.policy || POLICIES.hatchery, GIT_GUARD_MERGED: 'task/T1' }));
  const shown = JSON.stringify(String(o.command).split(fwd(scratch)).join('<scratch>'));
  const where = `${o.tool === 'PowerShell' ? ' ps' : ''}${o.cwd === repoTask ? ' on task/T1' : ''}${o.policyName ? ` (${o.policyName})` : ''}`;
  const label = `${shown} [${o.agent == null ? '-' : JSON.stringify(o.agent)}]${where} -> ${typeof expect === 'number' ? `deny rule ${expect}` : expect}`;
  let ok = false;
  let detail = '';
  if (expect === 'internal') {
    // The guard's own failure exits 2 with the reason on stderr, which blocks even without onFailure.
    ok = r.code === 2 && r.stdout === '' && r.stderr.startsWith('git-guard: internal error (');
    detail = `expected exit 2 and an internal-error reason on stderr; got exit ${r.code}, stderr ${JSON.stringify(r.stderr)}`;
  } else if (r.code !== 0) detail = `exit ${r.code}, expected 0; stderr: ${r.stderr.trim()}`;
  else if (expect === 'allow') { ok = r.stdout === ''; detail = 'expected no stdout'; }
  else {
    const d = parseDecision(r.stdout);
    if (!d || d.event !== 'PreToolUse' || d.decision !== 'deny' || typeof d.reason !== 'string') detail = 'expected deny JSON';
    else {
      const want = expect === 'policy' ? `git-guard: ${PREFIX.policy}` : `git-guard: ${PREFIX[expect]}: \``;
      ok = d.reason.startsWith(want);
      detail = `reason should start with ${JSON.stringify(want)}; got ${JSON.stringify(d.reason)}`;
      if (ok && extra) { const e = extra(d); if (e !== true) { ok = false; detail = e; } }
    }
  }
  report(id, label, ok, detail, r.stdout);
}

// [id, command, agent_type or null, expect: 'allow' | rule number | 'internal', raw stdin override, extra check]
const G = [
  // The 25-row table from the spec, in order.
  ['G01', 'git status', null, 'allow'],
  ['G02', 'git push', null, 'allow'],
  ['G03', 'git push', 'pusher', 'allow'],
  ['G04', 'git push', 'runner', 3],
  ['G05', 'git push origin main --force', 'pusher', 1],
  ['G06', 'git push -uf origin main', null, 1],
  ['G07', 'git push origin :old', null, 2],
  ['G08', 'git commit -m x', 'coder', 4],
  ['G09', 'git commit -m x', 'runner', 'allow'],
  ['G10', 'git commit -m x', null, 'allow'],
  ['G11', 'cd sub && git reset --hard HEAD~1', null, 5],
  ['G12', 'git reset --soft HEAD~1', null, 'allow'],
  ['G13', 'git clean -fd', null, 6],
  ['G14', 'git clean -n', null, 'allow'],
  ['G15', 'git checkout -- .', null, 7],
  ['G16', 'git checkout -- docs/HANDOFF.md', null, 'allow'],
  ['G17', 'git restore --staged .', null, 'allow'],
  ['G18', 'git restore .', null, 8],
  ['G19', 'git branch -D topic', null, 9],
  ['G20', 'git branch -d topic', null, 'allow'],
  ['G21', 'git rebase -i HEAD~3', null, 10],
  ['G22', 'git stash list', null, 'allow'],
  ['G23', 'git stash drop', null, 11],
  ['G24', 'echo "git push --force"', null, 'allow'], // quoted text is data since the quote-aware parser
  ['G25', 'not json', null, 'internal', 'not json'],
  // Extra cases: implementation choices and edge cases beyond the spec table.
  ['GX01', 'git push origin +main', null, 1],
  ['GX02', 'git push --force-with-lease', 'pusher', 1],
  ['GX03', 'git -C sub push --force', null, 1],
  ['GX04', "git push origin 'a|b' -f", null, 1],
  ['GX05', 'git status\ngit push -f', null, 1],
  ['GX06', 'git push --prune origin', null, 2],
  ['GX07', 'git push origin --delete old', 'pusher', 2],
  ['GX08', 'git commit -m "fix; then push"', 'coder', 4],
  ['GX09', 'git log --grep commit', 'coder', 'allow'],
  ['GX10', 'git push origin main && echo --force', null, 'allow'],
  ['GX11', 'git reset --merge', null, 5],
  ['GX12', 'git clean -nfd', null, 'allow'],
  ['GX13', 'git checkout .', null, 7],
  ['GX14', 'git checkout main', null, 'allow'],
  ['GX15', 'git restore --staged --worktree .', null, 8],
  ['GX16', 'git restore -- :/', null, 8],
  ['GX17', 'git branch --delete --force topic', null, 9],
  ['GX18', 'git rebase --interactive main', null, 10],
  ['GX19', 'git stash clear', null, 11],
  ['GX20', 'git filter-branch --tree-filter x HEAD', null, 12],
  ['GX21', 'git push', '', 'allow'],
  ['GX22', 'git status', 'coder', 'allow'],
  ['GX23', 'ls .git', 'coder', 'allow'],
  ['GX24', '(missing tool_input.command)', null, 'allow', JSON.stringify({ tool_name: 'Bash', tool_input: {} })],
  ['GX25', '(empty stdin)', null, 'internal', ''],
];

for (const [id, command, agent, expect, raw, extra] of G) guardCase(id, { command, agent, raw }, expect, extra);

// New rules (13 to 17, and the additions to 12), under the Hatchery policy.
// [id, command, agent, expect, extra]
const GN = [
  ['GN01', 'git commit --no-verify -m x', null, 13],
  ['GN02', 'git commit -nm x', 'runner', 13],
  ['GN03', 'git push --no-verify', 'pusher', 13],
  ['GN04', 'git merge --no-verify topic', null, 13],
  ['GN05', 'git push -n', 'pusher', 'allow'], // push -n is --dry-run, not --no-verify
  ['GN06', 'git -c core.hooksPath=/dev/null commit -m x', null, 14],
  ['GN07', 'git --config-env=core.hooksPath=HP commit -m x', null, 14],
  ['GN08', 'git config --unset core.hooksPath', null, 14],
  ['GN09', 'git config core.hooksPath .git/hooks', null, 14],
  ['GN10', 'git config core.hooksPath', 'runner', 'allow'], // a read
  ['GN11', 'GIT_CONFIG_COUNT=1 GIT_CONFIG_KEY_0=core.hooksPath GIT_CONFIG_VALUE_0=x git commit -m x', null, 14],
  ['GN12', 'git -c alias.p=\'!git push -f\' p', null, 15],
  ['GN13', 'git config --global alias.p "push --force"', null, 15],
  ['GN14', 'git config --get alias.st', null, 'allow'],
  ['GN15', 'git checkout -f main', null, 16],
  ['GN16', 'git checkout --force -- src', null, 16],
  ['GN17', 'git switch --discard-changes main', null, 16],
  ['GN18', 'git switch -f main', null, 16],
  ['GN19', 'git switch main', null, 'allow'],
  ['GN20', 'git worktree remove --force ../wt', null, 17],
  ['GN21', 'git worktree remove -f ../wt', null, 17],
  ['GN22', 'git worktree remove ../wt', null, 'allow'],
  ['GN23', 'git update-ref -d refs/heads/topic', null, 12],
  ['GN24', 'git reflog expire --expire=now --all', null, 12],
  ['GN25', 'git reflog show', null, 'allow'],
  ['GN26', 'git push --force-with-lease origin main', null, 1], // lease is force unless the policy names branches
  ['GN27', 'git -C . -c x.y=z --no-pager push --force', null, 1],
  ['GN28', 'git --git-dir=.git push origin +main', null, 1],
];
for (const [id, command, agent, expect, extra] of GN) guardCase(id, { command, agent }, expect, extra);

// Quote-aware parsing: quoted text and heredoc bodies are data unless handed to a shell.
// [id, command, tool, agent, expect, extra]
const HEREDOC_MSG = "git commit -m \"$(cat <<'EOF'\nDon't git push --force (yet); git reset --hard\nEOF\n)\"";
const GQ = [
  ['GQ01', 'git commit -m "fix git push -n and --no-verify"', 'Bash', 'runner', 'allow'],
  ['GQ02', HEREDOC_MSG, 'Bash', null, 'allow'],
  ['GQ03', 'git commit -m "-n"', 'Bash', null, 'allow'],
  ['GQ04', 'git commit -mnote', 'Bash', null, 'allow'], // -m with its value attached
  ['GQ05', 'git commit -m x -n', 'Bash', null, 13, noNote],
  ['GQ06', "echo 'git reset --hard' > notes.txt", 'Bash', null, 'allow'],
  ['GQ07', 'cat > notes.md <<EOF\ngit push --force\nEOF', 'Bash', null, 'allow'],
  ['GQ08', 'git status # then git push --force', 'Bash', null, 'allow'],
  ['GQ09', 'echo git push -f', 'Bash', null, 1, withNote], // unquoted: still read as a command
  ['GQ10', 'bash -c "git reset --hard"', 'Bash', null, 5, withNote],
  ['GQ11', 'echo "git push -f" | sh', 'Bash', null, 1, withNote],
  ['GQ12', 'bash <<EOF\ngit clean -fdx\nEOF', 'Bash', null, 6, withNote],
  ['GQ13', 'x=$(git push -f)', 'Bash', null, 1],
  ['GQ14', 'echo `git stash drop`', 'Bash', null, 11],
  ['GQ15', 'cat <<EOF\n$(git reset --hard)\nEOF', 'Bash', null, 5], // an unquoted heredoc expands $(...)
  ['GQ16', 'cat <<\'EOF\'\n$(git reset --hard)\nEOF', 'Bash', null, 'allow'],
  ['GQ17', '{ git push --force; }', 'Bash', null, 1, noNote],
  ['GQ18', 'git push "--force"', 'Bash', null, 1],
  ['GQ19', 'git status 2>&1 && git push --force > out.txt', 'Bash', null, 1],
  ['GQ20', "git commit -m 'it''s git push -f'", 'PowerShell', null, 'allow'],
  ['GQ21', "& 'C:\\Program Files\\Git\\cmd\\git.exe' push --force", 'PowerShell', null, 1],
  ['GQ22', "git commit -m @'\ngit push --force\n'@", 'PowerShell', null, 'allow'],
  ['GQ23', 'Invoke-Command { git push -f } | Out-Null', 'PowerShell', null, 1],
  ['GQ24', 'powershell -NoProfile -Command "git reset --hard"', 'PowerShell', null, 5, withNote],
  ['GQ25', "$env:GIT_CONFIG_KEY_0 = 'core.hooksPath'; git commit -m x", 'PowerShell', null, 14],
  ['GQ26', 'git status; git push --force 2>&1', 'PowerShell', null, 1],
  ['GQ27', 'Write-Output "git push --force"', 'PowerShell', null, 'allow'],
  ['GQ28', 'git commit -m "$(git rev-parse HEAD) fix"', 'PowerShell', 'coder', 4],
];
for (const [id, command, tool, agent, expect, extra] of GQ) guardCase(id, { command, tool, agent }, expect, extra);

// Shapes an independent review reproduced as bypasses (2026-10-13), and the cases it found
// untested. [id, command, tool, agent, expect, cwd, policy]
const mainNoPush = scratchPolicy('main-no-push', { git: { push: { mainSession: false } } });
const GR = [
  // Abbreviated long options read as the options they start.
  ['GR01', 'git reset --har HEAD~1', 'Bash', null, 5],
  ['GR02', 'git checkout --forc main', 'Bash', null, 16],
  ['GR03', 'git push --mirr origin', 'Bash', null, 2],
  ['GR04', 'git push --delet origin x', 'Bash', null, 2],
  ['GR05', 'git commit --no-verif -m x', 'Bash', null, 13],
  ['GR06', 'git switch --discard main', 'Bash', null, 16],
  ['GR07', 'git worktree remove --forc wt', 'Bash', null, 17],
  ['GR08', 'git clean --dry', 'Bash', null, 'allow'], // --dry is --dry-run
  ['GR09', 'git push --force-if-includes', 'Bash', 'pusher', 1],
  // Arguments the reader cannot see, on a subcommand that can do harm.
  ['GR10', 'X=--force; git push $X', 'Bash', null, 19],
  ['GR11', 'git push {-f,}', 'Bash', null, 19],
  ['GR12', 'git push ${F:---force}', 'Bash', null, 19],
  ['GR13', 'git push $(echo --force)', 'Bash', null, 19],
  ['GR14', 'G=git; $G push -f', 'Bash', null, 19],
  ['GR15', 'echo --force | xargs git push', 'Bash', null, 19],
  ['GR16', 'git checkout "$BRANCH"', 'Bash', null, 19],
  ['GR17', 'git reset --soft $(git commit-tree HEAD^{tree} -p HEAD -m x)', 'Bash', null, 19],
  ['GR18', 'git log --oneline $(git merge-base main HEAD)..HEAD', 'Bash', null, 'allow'],
  ['GR19', 'git commit -m "add $f" --file="$TMP"', 'Bash', 'runner', 'allow'], // values of options
  ['GR20', '$(git rev-parse --show-toplevel)/scripts/check.sh', 'Bash', null, 'allow'],
  ['GR21', 'git -C "$REPO" status && git diff {a,b}.txt', 'Bash', null, 'allow'],
  ['GR22', '$g="git"; & $g push -f', 'PowerShell', null, 19],
  ['GR23', '$a="push","-f"; git @a', 'PowerShell', null, 19],
  ['GR24', '$h = git rev-parse HEAD', 'PowerShell', null, 'allow'],
  // PowerShell launchers.
  ['GR25', 'Start-Process git -ArgumentList "push --force"', 'PowerShell', null, 1],
  ['GR26', 'Start-Process git -ArgumentList "push","--force"', 'PowerShell', null, 1],
  ['GR27', 'Start-Process -FilePath git -ArgumentList $rest', 'PowerShell', null, 19],
  ['GR28', '& ([scriptblock]::Create("git push -f"))', 'PowerShell', null, 1],
  ['GR29', 'iex "git reset --hard"', 'PowerShell', null, 5],
  ['GR30', 'cmd /c "git push -f"', 'PowerShell', null, 1],
  ['GR31', 'cmd /c git push -f', 'Bash', null, 1],
  ['GR32', 'echo "git push -f" | xargs -I{} sh -c "{}"', 'Bash', null, 1],
  ['GR33', 'sudo -u x bash -c "git reset --hard"', 'Bash', null, 5],
  ['GR34', "t 'git -C x push' '' PowerShell", 'Bash', 'coder', 'allow'], // a runner name as an argument runs nothing
  ['GR35', 'node -e "require(\'child_process\').execSync(\'git push -f\')"', 'Bash', null, 'allow'], // not covered, by design
  // Every command that makes a commit is rule 4's.
  ['GR36', 'git merge --no-ff feature', 'Bash', 'coder', 4],
  ['GR37', 'git cherry-pick abc', 'Bash', 'coder', 4],
  ['GR38', 'git revert HEAD', 'Bash', 'coder', 4],
  ['GR39', 'git am patch.mbox', 'Bash', 'coder', 4],
  ['GR40', 'git rebase main', 'Bash', 'coder', 4],
  ['GR41', 'git pull', 'Bash', 'coder', 4],
  ['GR42', 'git rebase --abort', 'Bash', 'coder', 'allow'],
  ['GR43', 'git merge feature', 'Bash', 'general-purpose', 4, repoMain, POLICIES.taskBranches],
  ['GR44', 'git rebase origin/main', 'Bash', 'general-purpose', 'allow', repoTask, POLICIES.taskBranches],
  // Config that redirects hooks by another route.
  ['GR45', 'GIT_CONFIG_GLOBAL=/tmp/evil git commit -m x', 'Bash', null, 14],
  ['GR46', 'GIT_CONFIG_SYSTEM=/tmp/evil git status', 'Bash', null, 14],
  ['GR47', "GIT_CONFIG_PARAMETERS=\"'core.hooksPath'='x'\" git commit -m x", 'Bash', null, 14],
  ['GR48', 'git -c include.path=/tmp/evil commit -m x', 'Bash', null, 14],
  ['GR49', 'git config include.path /tmp/evil', 'Bash', null, 14],
  ['GR50', 'git config includeIf.gitdir:~/w/.path /tmp/evil', 'Bash', null, 14],
  ['GR51', "$env:GIT_CONFIG_GLOBAL = 'x'; git status", 'PowerShell', null, 14],
  ['GR52', 'git config --get include.path', 'Bash', null, 'allow'],
  // Forced branch resets, send-pack, and whole-tree spellings.
  ['GR53', 'git checkout -B main HEAD~3', 'Bash', null, 18], // main is the current branch
  ['GR54', 'git switch -C main HEAD~3', 'Bash', null, 18],
  ['GR55', 'git branch -f main HEAD~3', 'Bash', null, 'allow', repoTask], // not current, not protected
  ['GR56', 'git branch -f main HEAD~3', 'Bash', null, 18, repoTask, POLICIES.taskBranches],
  ['GR57', 'git checkout -B topic origin/topic', 'Bash', null, 'allow'],
  ['GR58', 'git send-pack --force origin main', 'Bash', null, 1],
  ['GR59', 'git checkout -- ./.', 'Bash', null, 7],
  ['GR60', 'git checkout -- src/..', 'Bash', null, 7],
  ['GR61', 'git restore -- ./docs/..', 'Bash', null, 8],
  ['GR62', 'git checkout -- ./src', 'Bash', null, 'allow'],
  // Untested before.
  ['GR63', 'git push', 'Bash', null, 3, null, mainNoPush],
  ['GR64', 'git reflog delete HEAD@{1}', 'Bash', null, 12],
];
for (const [id, command, tool, agent, expect, cwd, policy] of GR) {
  guardCase(id, { command, tool, agent, cwd: cwd || undefined, policy: policy || undefined }, expect);
}
// A reassembled Start-Process command carries the note: git ran from text, not as the command word.
guardCase('GR65', { command: 'Start-Process git -ArgumentList "push --force"', tool: 'PowerShell' }, 1, withNote);

// Policy file: malformed denies every git command (and nothing else); absent means the defaults.
const badJson = scratchPolicy('badjson', '{ "git": ');
const badKey = scratchPolicy('badkey', { git: { push: { subagent: 'any' } } });
const badType = scratchPolicy('badtype', { usage: { hold: 'fifty' } });
guardCase('GF01', { command: 'git status', policy: badJson, policyName: 'not JSON' }, 'policy');
guardCase('GF02', { command: 'git status', policy: badKey, policyName: 'unknown key' }, 'policy');
guardCase('GF03', { command: 'git log', policy: badType, policyName: 'wrong type' }, 'policy');
guardCase('GF04', { command: 'ls -la', policy: badJson, policyName: 'not JSON' }, 'allow');
guardCase('GF05', { command: 'git commit -m x', agent: 'coder', policy: POLICIES.none, policyName: 'absent' }, 4);
guardCase('GF06', { command: 'git commit -m x', agent: 'runner', policy: POLICIES.none, policyName: 'absent' }, 'allow');

// One command set under every policy. The absent file must behave as the Hatchery one.
// [id, command, agent, cwd, { hatchery, taskBranches, singleBranch }]
const H = 'hatchery';
const GP = [
  ['01', 'git push', 'pusher', repoMain, { hatchery: 'allow', taskBranches:3, singleBranch:'allow' }],
  ['02', 'git push', 'general-purpose', repoTask, { hatchery: 3, taskBranches:'allow', singleBranch:3 }],
  ['03', 'git push', null, repoMain, { hatchery: 'allow', taskBranches:'allow', singleBranch:'allow' }],
  ['04', 'git commit -m x', 'runner', repoMain, { hatchery: 'allow', taskBranches:4, singleBranch:'allow' }],
  ['05', 'git commit -m x', 'general-purpose', repoMain, { hatchery: 4, taskBranches:4, singleBranch:'allow' }],
  ['06', 'git commit -m x', 'general-purpose', repoTask, { hatchery: 4, taskBranches:'allow', singleBranch:'allow' }],
  ['07', 'git push --force-with-lease origin task/T1', 'general-purpose', repoTask, { hatchery: 1, taskBranches:'allow', singleBranch:1 }],
  ['08', 'git push --force-with-lease origin main', null, repoMain, { hatchery: 1, taskBranches:1, singleBranch:1 }],
  ['09', 'git branch -D task/T1', null, repoMain, { hatchery: 9, taskBranches:'allow', singleBranch:9 }],
  ['10', 'git branch -D task/T2', null, repoMain, { hatchery: 9, taskBranches:9, singleBranch:9 }],
  ['11', 'git push origin --delete task/T1', null, repoMain, { hatchery: 2, taskBranches:'allow', singleBranch:2 }],
  ['12', 'git push origin :task/T2', 'general-purpose', repoMain, { hatchery: 2, taskBranches:2, singleBranch:2 }],
  ['13', 'git config core.hooksPath tools/git-hooks', null, repoMain, { hatchery: 14, taskBranches:'allow', singleBranch:14 }],
  ['14', 'git push --tags', 'general-purpose', repoTask, { hatchery: 3, taskBranches:3, singleBranch:3 }],
  ['15', 'git push --tags', 'pusher', repoMain, { hatchery: 'allow', taskBranches:3, singleBranch:'allow' }],
  ['16', 'git push origin v1.0', null, repoMain, { hatchery: 'allow', taskBranches:'allow', singleBranch:'allow' }],
  ['17', 'git push origin HEAD:main', 'general-purpose', repoTask, { hatchery: 3, taskBranches:3, singleBranch:3 }],
  ['18', `git -C ${fwd(repoTask)} commit -m x`, 'general-purpose', repoMain, { hatchery: 4, taskBranches:'allow', singleBranch:'allow' }],
  ['19', `cd ${fwd(repoMain)} && git commit -m x`, 'general-purpose', repoTask, { hatchery: 4, taskBranches:4, singleBranch:'allow' }],
  ['20', 'git reset --hard', null, repoMain, { hatchery: 5, taskBranches:5, singleBranch:5 }],
  ['21', 'git commit --no-verify -m x', 'general-purpose', repoTask, { hatchery: 4, taskBranches:13, singleBranch:13 }],
];
for (const [n, command, agent, cwd, want] of GP) {
  for (const p of [H, 'none', 'taskBranches', 'singleBranch']) {
    const tag = { hatchery: 'H', none: 'N', taskBranches: 'T', singleBranch: 'S' }[p];
    guardCase(`GP${n}${tag}`, { command, agent, cwd, policy: POLICIES[p], policyName: p }, want[p === 'none' ? H : p]);
  }
}

// ---------- post-edit ----------
const write = (rel, text) => {
  const f = path.join(scratch, ...rel.split('/'));
  fs.mkdirSync(path.dirname(f), { recursive: true });
  fs.writeFileSync(f, text);
};
const lines = (n, prefix) => Array.from({ length: n }, (_, i) => `${prefix || 'line'} ${i + 1}\n`).join('');

write('docs/reference/README.md', '# Reference\n\n| `big.md` | `clean.md` | `noverified.md` |\n');
write('docs/reference/big.md', `---\nverified: 2026-10-11\nsource: observed\n---\n${lines(297)}`); // 301 newlines
write('docs/reference/clean.md', '---\nverified: 2026-10-11\nsource: observed\n---\n\n# Clean\n');
write('docs/reference/noverified.md', '---\nsource: observed\n---\n# No verified\n');
write('docs/reference/unlisted.md', '---\nverified: 2026-10-11\n---\n# Unlisted\n');
// 41 lines, synthetic: the suite reads no file of the repository it runs in.
write('docs/HANDOFF.md', `# Handoff\n\n${lines(39)}`);
write('docs/findings/README.md', '# Findings\n\n`2026-01-01-nofm.md` `2026-01-02-ok.md` `2026-01-03-gone.md`\n');
write('docs/findings/2026-01-01-nofm.md', '# A finding without frontmatter\n\n## Question\n');
write('docs/findings/2026-01-02-ok.md', '---\ndate: 2026-01-02\nstatus: open\narea: test\n---\n# OK\n');
write('docs/design/README.md', '# Design\n\n- [0009 Something](decisions/0009-something.md)\n');
write('docs/design/decisions/README.md', '# Decision records\n');
write('docs/design/decisions/0009-something.md', '# 0009. Something\n');
write('docs/design/proposals/README.md', '# Proposals\n\n`0009-long.md`\n');
write('docs/design/proposals/0009-long.md', lines(320));
write('src/app.txt', 'not a doc\n');
write('README.md', lines(400));
write('CLAUDE.md', lines(151));
write('src/CLAUDE.md', lines(151));
write('root150/CLAUDE.md', lines(150));

const absOf = (rel) => path.join(scratch, ...rel.split('/'));
const fmtOk = { '.txt': [NODE, '-e', 'process.exit(0)'] };
const fmtBad = { '.txt': [NODE, '-e', 'console.error("bad formatter output");process.exit(3)'] };

// [id, label, file_path (absolute unless noted), expected additionalContext or null, raw stdin override, formatter env,
//  policy file (the Hatchery fixture when omitted), project root (the scratch tree when omitted)]
const P = [
  // The four cases the spec asks for.
  ['P01', '301-line docs/reference file', absOf('docs/reference/big.md'),
    'post-edit: docs/reference/big.md is 301 lines; cap is 300 (.claude/rules/docs.md). Split by topic; the pusher will refuse to push.'],
  ['P02', '41-line docs/HANDOFF.md', absOf('docs/HANDOFF.md'),
    'post-edit: docs/HANDOFF.md is 41 lines; cap is 40. Move detail to docs/ideas/ or a finding.'],
  ['P03', 'findings file without frontmatter', absOf('docs/findings/2026-01-01-nofm.md'),
    'post-edit: docs/findings/2026-01-01-nofm.md lacks finding frontmatter (date, status, area). See the finding skill.'],
  ['P04', 'clean reference file', absOf('docs/reference/clean.md'), null],
  // Extra cases.
  ['PX01', 'reference file without verified:', absOf('docs/reference/noverified.md'),
    'post-edit: docs/reference/noverified.md lacks verified: frontmatter. See .claude/rules/docs.md.'],
  ['PX02', 'reference file missing from its index', absOf('docs/reference/unlisted.md'),
    'post-edit: docs/reference/unlisted.md is not listed in reference/README.md. Add one line.'],
  ['PX03', 'finding with full frontmatter', absOf('docs/findings/2026-01-02-ok.md'), null],
  ['PX04', 'ADR listed only in docs/design/README.md', absOf('docs/design/decisions/0009-something.md'), null],
  ['PX05', '320-line proposal (exempt from cap)', absOf('docs/design/proposals/0009-long.md'), null],
  ['PX06', 'relative file_path resolved against cwd', 'docs/reference/big.md',
    'post-edit: docs/reference/big.md is 301 lines; cap is 300 (.claude/rules/docs.md). Split by topic; the pusher will refuse to push.'],
  ['PX07', '400-line .md outside docs/', absOf('README.md'), null],
  ['PX08', 'non-doc file, empty formatter table', absOf('src/app.txt'), null],
  ['PX09', 'malformed stdin', '(n/a)', null, 'not json'],
  ['PX10', 'formatter that succeeds', absOf('src/app.txt'), null, null, fmtOk],
  ['PX12', '151-line CLAUDE.md', absOf('CLAUDE.md'),
    'post-edit: CLAUDE.md is 151 lines; cap is 150 (.claude/rules/docs.md). Move facts to docs/reference/ and path-scoped rules to .claude/rules/.'],
  ['PX13', '150-line CLAUDE.md at another root', absOf('root150/CLAUDE.md'), null, null, null, null, absOf('root150')],
  ['PX14', '151-line CLAUDE.md below the root', absOf('src/CLAUDE.md'), null],
  ['PX11', 'formatter that fails', absOf('src/app.txt'),
    `post-edit: formatter ${NODE} failed on src/app.txt: bad formatter output`, null, fmtBad],
  // Path anchors: backtick-quoted repo paths must exist.
  ['PA01', 'doc naming a missing path', absOf('docs/notes/anchors.md'),
    'post-edit: docs/notes/anchors.md names paths that do not exist: `docs/reference/gone.md`, `src/old/`. Fix them, or mark the paragraph [no-check: <reason>] (.claude/rules/docs.md).'],
  ['PA02', 'missing path in a [no-check] paragraph', absOf('docs/notes/nocheck.md'), null],
  ['PA03', 'missing path in a fenced block', absOf('docs/notes/fenced.md'), null],
  ['PA04', 'placeholders, globs, refs, other repos', absOf('docs/notes/placeholders.md'), null],
  // Policy: cap exemptions and formatters come from the edit block.
  ['PC01', 'cap exemption by path', absOf('docs/reference/big.md'), null, null, null,
    scratchPolicy('capexempt', { edit: { capExempt: ['docs/reference/big.md'] } })],
  ['PC02', 'cap exemption by pattern', absOf('docs/reference/big.md'), null, null, null,
    scratchPolicy('capglob', { edit: { capExempt: ['docs/reference/*'] } })],
  ['PC03', 'formatter from the policy', absOf('src/app.txt'),
    `post-edit: formatter ${NODE} failed on src/app.txt: bad formatter output`, null, null,
    scratchPolicy('formatter', { edit: { formatters: fmtBad } })],
  ['PC04', 'malformed policy: base defaults', absOf('docs/reference/big.md'),
    'post-edit: docs/reference/big.md is 301 lines; cap is 300 (.claude/rules/docs.md). Split by topic; the pusher will refuse to push.',
    null, null, scratchPolicy('pe-bad', '{ nope')],
  ['PC05', 'task-branches example exempts docs/TASKS.md', absOf('docs/TASKS.md'), null, null, null, POLICIES.taskBranches],
  ['PC06', 'docs/PLANNING.md under the Hatchery policy', absOf('docs/PLANNING.md'),
    'post-edit: docs/PLANNING.md is 320 lines; cap is 300 (.claude/rules/docs.md). Split by topic; the pusher will refuse to push.'],
  // Closed records are exempt from the path check by default; list items scope [no-check:].
  ['PA05', 'a finding naming a gone path (closed record)', absOf('docs/findings/2026-01-03-gone.md'), null],
  ['PA06', 'the same finding with anchorExempt emptied', absOf('docs/findings/2026-01-03-gone.md'),
    'post-edit: docs/findings/2026-01-03-gone.md names paths that do not exist: `docs/reference/gone.md`. Fix them, or mark the paragraph [no-check: <reason>] (.claude/rules/docs.md).',
    null, null, scratchPolicy('anchor-all', { edit: { anchorExempt: [] } })],
  ['PA07', 'an indented fenced block', absOf('docs/notes/indented.md'), null],
  ['PA08', '[no-check:] covers its own list item only', absOf('docs/notes/list.md'),
    'post-edit: docs/notes/list.md names paths that do not exist: `docs/reference/also-gone.md`. Fix them, or mark the paragraph [no-check: <reason>] (.claude/rules/docs.md).'],
];
write('docs/findings/2026-01-03-gone.md', '---\ndate: 2026-01-03\nstatus: resolved\narea: test\n---\n# Gone\n\nIt lived in `docs/reference/gone.md`.\n');
write('docs/notes/indented.md', '# Indented\n\n- step:\n\n    ```\n    cat `docs/reference/gone.md`\n    ```\n');
write('docs/notes/list.md', '# List\n\n- planned `docs/reference/gone.md` [no-check: not yet]\n- also `docs/reference/also-gone.md`\n- fine `docs/reference/clean.md`\n');
write('docs/notes/README.md', '# Notes\n\n`anchors.md` `nocheck.md` `fenced.md` `placeholders.md` `indented.md` `list.md`\n');
write('docs/notes/anchors.md', '# Anchors\n\nSee `docs/reference/clean.md`, `docs/reference/gone.md`, `src/app.txt` and `src/old/`.\n\nAgain `docs/reference/gone.md`.\n');
write('docs/notes/nocheck.md', '# No check\n\nPlanned: `docs/reference/gone.md`. [no-check: not written yet]\n');
write('docs/notes/fenced.md', '# Fenced\n\n```\nls `docs/reference/gone.md`\n```\n\nFine: `docs/reference/clean.md`.\n');
write('docs/notes/placeholders.md', '# Placeholders\n\n`docs/design/decisions/NNNN-title.md`, `docs/findings/<slug>.md`, `docs/*.md`,\n' +
  '`origin/main`, `../decisions/x.md`, `task/T1`, `docs/reference/clean.md#section`, `docs/reference/clean.md:3`.\n');
write('docs/PLANNING.md', lines(320));

for (const [id, label, filePath, expect, rawOverride, fmt, policy, root] of P) {
  if (!selected(id)) continue;
  const stdin = rawOverride != null ? rawOverride : JSON.stringify({
    hook_event_name: 'PostToolUse', tool_name: 'Write', cwd: root || scratch, tool_input: { file_path: filePath },
  });
  const env = envWith({ HOOK_POLICY_FILE: policy || POLICIES.hatchery, POST_EDIT_FORMATTERS_JSON: fmt ? JSON.stringify(fmt) : undefined });
  const r = run('post-edit', stdin, env);
  let ok = false;
  let detail = '';
  if (r.code !== 0) detail = `exit ${r.code}, expected 0; stderr: ${r.stderr.trim()}`;
  else if (expect === null) { ok = r.stdout === ''; detail = 'expected no stdout'; }
  else {
    const d = parseDecision(r.stdout);
    ok = !!d && d.event === 'PostToolUse' && d.context === expect;
    detail = `expected additionalContext ${JSON.stringify(expect)}`;
  }
  report(id, `${label} -> ${expect === null ? 'no output' : 'message'}`, ok, detail, r.stdout);
  if (ok && !opt.verbose) process.stdout.write(`     exit ${r.code}, stdout: ${r.stdout || '(empty)'}\n`);
}

// ---------- usage-gate (section 3 text) ----------
const home = path.join(scratch, 'home');
fs.mkdirSync(path.join(home, '.claude'), { recursive: true });
const HOLD_MSG = 'Usage gate: 5h 55%, 7d 10%. At or above 50%: Do not start new subagents. Let in-flight agents finish, run the handoff skill, commit from the main session, then report. Lifting this early is the user\'s decision (docs/design/decisions/0006-hook-policy-file.md).';
const STOP_MSG = 'Usage gate: 5h 85%, 7d 10%. At or above 80%: Stop all work, run the handoff skill, commit WIP from the main session, end the turn.';
const nowSec = Math.floor(Date.now() / 1000);
// [id, pct, expected deny reason or null for allow, override file contents or undefined]
const U = [
  ['U01', 55, HOLD_MSG],
  ['U02', 85, STOP_MSG],
  ['U03', 20, null],
  ['U04', 55, null, { until: nowSec + 3600, reason: 'test: user instruction' }],
  ['U05', 55, HOLD_MSG, { until: nowSec - 1, reason: 'test: expired' }],
  ['U06', 85, STOP_MSG, { until: nowSec + 3600, reason: 'test: override does not lift STOP' }],
  ['U07', 55, HOLD_MSG, 'not json'],
];
for (const [id, pct, expect, override] of U) {
  if (!selected(id)) continue;
  fs.writeFileSync(path.join(home, '.claude', 'usage-state.json'), JSON.stringify({
    updatedAt: Math.floor(Date.now() / 1000),
    five_hour: { used_percentage: pct },
    seven_day: { used_percentage: 10 },
  }));
  const ovPath = path.join(home, '.claude', 'usage-gate-override.json');
  if (override === undefined) fs.rmSync(ovPath, { force: true });
  else fs.writeFileSync(ovPath, typeof override === 'string' ? override : JSON.stringify(override));
  const env = envWith({ HOME: home, USERPROFILE: home, HOOK_POLICY_FILE: POLICIES.hatchery });
  const r = run('usage-gate', JSON.stringify({ hook_event_name: 'PreToolUse', tool_name: 'Agent', tool_input: {} }), env);
  let ok;
  let detail;
  if (expect === null) { ok = r.code === 0 && r.stdout === ''; detail = 'expected allow (exit 0, no stdout)'; }
  else {
    const d = parseDecision(r.stdout);
    ok = r.code === 0 && !!d && d.decision === 'deny' && d.reason === expect;
    detail = `expected reason ${JSON.stringify(expect)}`;
  }
  report(id, `usage ${pct}% -> ${expect === null ? 'allow' : 'deny'}`, ok, detail, r.stdout);
}

// Staleness, reset windows and the example policies. state null means no file.
// [id, label, state, policy file, expected deny-reason prefix or null for allow, stderr must include]
const stateFile = path.join(home, '.claude', 'usage-state.json');
const fresh = (five, seven, extra) => Object.assign({ updatedAt: nowSec, five_hour: { used_percentage: five }, seven_day: { used_percentage: seven } }, extra);
const staleDeny = scratchPolicy('stale-deny', { usage: { whenStale: 'deny' } });
const UX = [
  ['UX01', 'stale data, whenStale allow', fresh(95, 10, { updatedAt: nowSec - 3 * 3600 }), POLICIES.hatchery, null, '180 min old; allowing'],
  ['UX02', 'stale data, whenStale deny', fresh(10, 10, { updatedAt: nowSec - 3 * 3600 }), staleDeny,
    'Usage gate: usage data is 180 min old, and the policy denies delegation without fresh data. Check usage with /usage'],
  ['UX03', 'no data, whenStale deny', null, staleDeny, 'Usage gate: no usage data'],
  ['UX04', 'no data, whenStale allow', null, POLICIES.hatchery, null, 'no usage data'],
  ['UX05', 'a window past its reset no longer counts', fresh(95, 10, { five_hour: { used_percentage: 95, resets_at: nowSec - 60 } }), POLICIES.hatchery, null],
  ['UX06', 'a window before its reset counts', fresh(95, 10, { five_hour: { used_percentage: 95, resets_at: nowSec + 60 } }), POLICIES.hatchery, 'Usage gate: 5h 95%, 7d 10%. At or above 80%'],
  ['UX07', 'single-branch example, 85%', fresh(85, 10), POLICIES.singleBranch, 'Usage gate: 5h 85%, 7d 10%. At or above 80%: Do not start new subagents. Let in-flight agents finish and commit'],
  ['UX08', 'single-branch example, 92%', fresh(92, 10), POLICIES.singleBranch, 'Usage gate: 5h 92%, 7d 10%. At or above 90%: Stop all work; commit what is green'],
  ['UX09', 'single-branch example, 60%', fresh(60, 10), POLICIES.singleBranch, null],
  ['UX10', 'task-branches example, 80%', fresh(10, 80), POLICIES.taskBranches, 'Usage gate: 5h 10%, 7d 80%. At or above 75%: No new implementers'],
  ['UX11', 'malformed policy: base defaults', fresh(55, 10), scratchPolicy('ug-bad', '[]'), HOLD_MSG, 'using the base defaults'],
  ['UX12', 'pre-merge file with updated_at only', { updated_at: nowSec, five_hour: { used_percentage: 95 } }, POLICIES.hatchery, null, 'no usage data'],
];
fs.rmSync(path.join(home, '.claude', 'usage-gate-override.json'), { force: true });
for (const [id, label, state, policy, expect, stderrHas] of UX) {
  if (!selected(id)) continue;
  if (state === null) fs.rmSync(stateFile, { force: true }); else fs.writeFileSync(stateFile, JSON.stringify(state));
  const r = run('usage-gate', JSON.stringify({ hook_event_name: 'PreToolUse', tool_name: 'Agent', tool_input: {} }),
    envWith({ HOME: home, USERPROFILE: home, HOOK_POLICY_FILE: policy }));
  let ok;
  let detail;
  if (expect === null) { ok = r.code === 0 && r.stdout === ''; detail = 'expected allow (exit 0, no stdout)'; }
  else {
    const d = parseDecision(r.stdout);
    ok = r.code === 0 && !!d && d.decision === 'deny' && d.reason.startsWith(expect);
    detail = `expected a reason starting ${JSON.stringify(expect)}`;
  }
  if (ok && stderrHas && !r.stderr.includes(stderrHas)) { ok = false; detail = `stderr lacks ${JSON.stringify(stderrHas)}: ${r.stderr}`; }
  report(id, `${label} -> ${expect === null ? 'allow' : 'deny'}`, ok, detail, r.stdout);
}

// ---------- statusline ----------
// Runs .claude/statusline.js with a scratch HOME. The cache must merge per window and never lose
// numbers to a refresh that carries no rate_limits.
const slHome = path.join(scratch, 'sl-home');
fs.mkdirSync(path.join(slHome, '.claude'), { recursive: true });
const slFile = path.join(slHome, '.claude', 'usage-state.json');
const readSl = () => { try { return fs.readFileSync(slFile, 'utf8'); } catch (_) { return null; } };
function statusline(stdin, policy) {
  const r = spawnSync(NODE, [path.join(HOOKS, '..', 'statusline.js')], {
    input: stdin, encoding: 'utf8', timeout: 30000,
    env: envWith({ HOME: slHome, USERPROFILE: slHome, HOOK_POLICY_FILE: policy || POLICIES.hatchery }),
  });
  return { code: r.status, stdout: r.stdout || '' };
}
const rl = (five, seven) => JSON.stringify({ model: { display_name: 'M' }, rate_limits: Object.assign({},
  five === null ? {} : { five_hour: { used_percentage: five, resets_at: nowSec + 3600 } },
  seven === null ? {} : { seven_day: { used_percentage: seven, resets_at: nowSec + 86400 * 3 } }) });
// [id, label, stdin, policy, check(result, fileBefore, fileAfter) -> true or a failure detail]
const S = [
  ['S01', 'both windows written', rl(55, 20), null, (r, b, a) => {
    const j = JSON.parse(a);
    return (j.five_hour.used_percentage === 55 && j.five_hour.resets_at === nowSec + 3600 && j.seven_day.used_percentage === 20 &&
      typeof j.updatedAt === 'number' && r.stdout.includes('5h 55%') && r.stdout.endsWith(' HOLD')) || `file ${a}, line ${r.stdout}`;
  }],
  ['S02', 'no rate_limits: file untouched, cached numbers shown', JSON.stringify({ model: { display_name: 'M' } }), null,
    (r, b, a) => (b === a && r.stdout.includes('5h 55%') && r.stdout.includes('7d 20%')) || `before ${b}, after ${a}, line ${r.stdout}`],
  ['S03', 'one window: the other is kept', rl(null, 30), null, (r, b, a) => {
    const j = JSON.parse(a);
    return (j.five_hour.used_percentage === 55 && j.seven_day.used_percentage === 30) || `file ${a}`;
  }],
  ['S04', 'malformed stdin: prints, writes nothing', 'not json', null,
    (r, b, a) => (r.code === 0 && b === a && r.stdout.startsWith('? |')) || `exit ${r.code}, line ${r.stdout}`],
  ['S05', 'flags follow the policy (single-branch: 85 is HOLD)', rl(85, 10), POLICIES.singleBranch, (r) => r.stdout.endsWith(' HOLD') || r.stdout],
  ['S06', 'flags follow the policy (single-branch: 92 is STOP)', rl(92, 10), POLICIES.singleBranch, (r) => r.stdout.endsWith(' STOP') || r.stdout],
  ['S07', 'flags follow the policy (Hatchery: 85 is STOP)', rl(85, 10), null, (r) => r.stdout.endsWith(' STOP') || r.stdout],
];
fs.rmSync(slFile, { force: true });
for (const [id, label, stdin, policy, check] of S) {
  if (!selected(id)) continue;
  const before = readSl();
  const r = statusline(stdin, policy);
  const after = readSl();
  let res;
  try { res = check(r, before, after); } catch (e) { res = e.message; }
  report(id, label, res === true, res, r.stdout);
}
// A cached window past its reset reads as n/a rather than its old number.
if (selected('S08')) {
  fs.writeFileSync(slFile, JSON.stringify({ updatedAt: nowSec, five_hour: { used_percentage: 90, resets_at: nowSec - 1 }, seven_day: { used_percentage: 5 } }));
  const r = statusline(JSON.stringify({ model: { display_name: 'M' } }));
  report('S08', 'cached window past its reset shows n/a', r.stdout.includes('5h n/a') && !r.stdout.includes('STOP'), r.stdout, r.stdout);
}
// A rename that fails (the cache path is a directory here) leaves no temporary file behind.
if (selected('S09')) {
  fs.rmSync(slFile, { force: true });
  fs.mkdirSync(slFile);
  const r = statusline(rl(40, 10));
  const left = fs.readdirSync(path.dirname(slFile)).filter((f) => f.endsWith('.tmp'));
  report('S09', 'failed rename: no .tmp left, line still prints', r.code === 0 && left.length === 0 && r.stdout.includes('5h 40%'),
    `exit ${r.code}, leftovers ${left.join(', ')}`, r.stdout);
  fs.rmSync(slFile, { recursive: true, force: true });
}

// ---------- policy files themselves ----------
// The repository's own policy.json (whatever the project chose) and the examples must load; the
// Hatchery fixture must equal the loader's defaults, since an absent file means those.
{
  const { load, DEFAULTS } = require('./policy');
  const files = [['LIVE01', 'this repository\'s policy.json loads (or is absent)', path.join(HOOKS, 'policy.json')],
    ['LIVE02', 'policy.examples/task-branches.json loads', POLICIES.taskBranches],
    ['LIVE03', 'policy.examples/single-branch.json loads', POLICIES.singleBranch]];
  for (const [id, label, file] of files) {
    if (!selected(id)) continue;
    const r = load(file);
    report(id, label, r.error === null, r.error);
  }
  if (selected('LIVE04')) {
    const merged = load(POLICIES.hatchery).policy;
    const strip = (p) => JSON.stringify(Object.assign({}, p, { usage: Object.assign({}, p.usage, { holdAdvice: '', denyAdvice: '' }) }));
    report('LIVE04', 'the Hatchery fixture equals the loader defaults', strip(merged) === strip(DEFAULTS), `fixture ${strip(merged)} vs defaults ${strip(DEFAULTS)}`);
  }
}

if (opt.keep) process.stdout.write(`scratch tree kept at ${scratch}\n`);
else fs.rmSync(scratch, { recursive: true, force: true });

process.stdout.write(`\n${pass} passed, ${fail} failed, ${pass + fail} total\n`);
process.exit(fail > 0 ? 1 : 0);
