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
// the end); usage-gate cases point HOME/USERPROFILE at a scratch dir. The real docs/ and
// ~/.claude are never touched. Exit codes: 0 all pass, 1 any failure, 2 usage error.
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
};

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

for (const [id, command, agent, expect, rawOverride, extra] of G) {
  if (!selected(id)) continue;
  const input = { hook_event_name: 'PreToolUse', tool_name: 'Bash', cwd: process.cwd(), tool_input: { command } };
  if (agent !== null) input.agent_type = agent;
  const stdin = rawOverride !== undefined && rawOverride !== null ? rawOverride : JSON.stringify(input);
  const r = run('git-guard', stdin);
  const label = `${JSON.stringify(command)} [${agent === null ? '-' : JSON.stringify(agent)}] -> ${expect === 'allow' || expect === 'internal' ? expect : `deny rule ${expect}`}`;
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
      const want = `git-guard: ${PREFIX[expect]}: \``;
      ok = d.reason.startsWith(want);
      detail = `reason should start with ${JSON.stringify(want)}; got ${JSON.stringify(d.reason)}`;
      if (ok && extra) { const e = extra(d); if (e !== true) { ok = false; detail = e; } }
    }
  }
  report(id, label, ok, detail, r.stdout);
}

// ---------- post-edit ----------
const scratch = fs.mkdtempSync(path.join(os.tmpdir(), 'hooks-test-'));
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
const realHandoff = fs.readFileSync(path.join(HOOKS, '..', '..', 'docs', 'HANDOFF.md'), 'utf8');
const handoffLines = realHandoff.split('\n').filter((_, i, a) => i < a.length - 1 || a[i] !== '');
while (handoffLines.length < 41) handoffLines.push('padding');
write('docs/HANDOFF.md', handoffLines.slice(0, 41).join('\n') + '\n'); // 41-line copy of the real one
write('docs/findings/README.md', '# Findings\n\n`2026-01-01-nofm.md` `2026-01-02-ok.md`\n');
write('docs/findings/2026-01-01-nofm.md', '# A finding without frontmatter\n\n## Question\n');
write('docs/findings/2026-01-02-ok.md', '---\ndate: 2026-01-02\nstatus: open\narea: test\n---\n# OK\n');
write('docs/design/README.md', '# Design\n\n- [0009 Something](decisions/0009-something.md)\n');
write('docs/design/decisions/README.md', '# Decision records\n');
write('docs/design/decisions/0009-something.md', '# 0009. Something\n');
write('docs/design/proposals/README.md', '# Proposals\n\n`0009-long.md`\n');
write('docs/design/proposals/0009-long.md', lines(320));
write('src/app.txt', 'not a doc\n');
write('README.md', lines(400));

const absOf = (rel) => path.join(scratch, ...rel.split('/'));
const fmtOk = { '.txt': [NODE, '-e', 'process.exit(0)'] };
const fmtBad = { '.txt': [NODE, '-e', 'console.error("bad formatter output");process.exit(3)'] };

// [id, label, file_path (absolute unless noted), expected additionalContext or null, raw stdin override, formatter env]
const P = [
  // The four cases the spec asks for.
  ['P01', '301-line docs/reference file', absOf('docs/reference/big.md'),
    'post-edit: docs/reference/big.md is 301 lines; cap is 300 (.claude/rules/docs.md). Split by topic; the pusher will refuse to push.'],
  ['P02', '41-line docs/HANDOFF.md copy', absOf('docs/HANDOFF.md'),
    'post-edit: docs/HANDOFF.md is 41 lines; cap is 40. Move detail to docs/backlog.md or a finding.'],
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
  ['PX11', 'formatter that fails', absOf('src/app.txt'),
    `post-edit: formatter ${NODE} failed on src/app.txt: bad formatter output`, null, fmtBad],
];

for (const [id, label, filePath, expect, rawOverride, fmt] of P) {
  if (!selected(id)) continue;
  const stdin = rawOverride != null ? rawOverride : JSON.stringify({
    hook_event_name: 'PostToolUse', tool_name: 'Write', cwd: scratch, tool_input: { file_path: filePath },
  });
  const env = Object.assign({}, process.env);
  delete env.POST_EDIT_FORMATTERS_JSON;
  if (fmt) env.POST_EDIT_FORMATTERS_JSON = JSON.stringify(fmt);
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
const HOLD_MSG = 'Usage gate: 5h 55%, 7d 10%. At or above 50%: do not start new subagents. Let in-flight agents finish, run the handoff skill, commit from the main session, then report. On the user\'s explicit instruction only, write ~/.claude/usage-gate-override.json {"until": <epoch>, "reason": "..."} to lift HOLD.';
const STOP_MSG = 'Usage gate: 5h 85%, 7d 10%. At or above 80%: stop all work, run the handoff skill, commit WIP from the main session, end the turn.';
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
    updated_at: Math.floor(Date.now() / 1000),
    five_hour: { used_percentage: pct },
    seven_day: { used_percentage: 10 },
  }));
  const ovPath = path.join(home, '.claude', 'usage-gate-override.json');
  if (override === undefined) fs.rmSync(ovPath, { force: true });
  else fs.writeFileSync(ovPath, typeof override === 'string' ? override : JSON.stringify(override));
  const env = Object.assign({}, process.env, { HOME: home, USERPROFILE: home });
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

if (opt.keep) process.stdout.write(`scratch tree kept at ${scratch}\n`);
else fs.rmSync(scratch, { recursive: true, force: true });

process.stdout.write(`\n${pass} passed, ${fail} failed, ${pass + fail} total\n`);
process.exit(fail > 0 ? 1 : 0);
