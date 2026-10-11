# ADR 0002 companion: hook specifications

Behaviour of the three scripts under `.claude/hooks/`, as implemented and tested on 2026-10-11
(`node .claude/hooks/test-hooks.js`, 68 cases). The code is the reference; this document describes
it exactly enough to re-implement or extend it. Both new scripts: Node, `'use strict'`, no
dependencies outside the standard library, SPDX header on line 2 after the shebang, read all of stdin
as JSON, run under Git Bash on Windows and bash on Linux. Hook contract:
https://code.claude.com/docs/en/hooks (input fields, output JSON, exit codes).

Facts the design relies on (from that page): exit 0 with JSON on stdout means "the fields decide";
exit 1 without valid JSON is a non-blocking error, so a policy hook must never rely on exit 1; exit 2
is a blocking error with stderr as the reason; `hookSpecificOutput` must wrap `additionalContext`, a
top-level field is ignored; `agent_type` is present only when the hook fires inside a subagent (or
with `--agent`); `onFailure: "block"` turns a crash, timeout, or missing interpreter into a block.

## 1. `git-guard.js` (PreToolUse, matcher `Bash`, `"onFailure": "block"`)

`onFailure: "block"` was added to `.claude/settings.json` only after the test table passed, because
with that flag a missing script blocks every Bash call in the repository (observed 2026-10-11).

Input: `tool_input.command`, `agent_type` (optional). Caller: `agent_type` when it is a non-empty
string, otherwise the main session. Decision: an empty string counts as absent, because it carries no
caller identity and must not lock the main session out of its own commits and pushes. A missing or
non-string `tool_input.command` allows: there is nothing to inspect, and the Bash tool rejects it.

Parsing. The command is normalised (`\r` removed, tabs and runs of spaces collapsed to one space) and
split into units: every segment between `&&`, `||`, `;`, `|`, or newline (trimmed, non-empty), then
the whole string with newlines replaced by ` ; `. In each unit the words are cleaned (leading quotes,
backticks, `$`, parens, braces stripped; trailing quotes, backticks, parens, braces, `;` stripped)
and every git word is located: a cleaned word matching `(^|[/\\])git(\.exe)?$`, case-insensitive, so
`/usr/bin/git`, `git.exe`, and `"git` all count. Words after it that start with `-` are git global
options and are skipped; `-C`, `-c`, `--git-dir`, `--work-tree`, `--namespace`, `--config-env`, and
`--super-prefix` also consume the following word (`--git-dir=x` is one word and is simply skipped).
The next word is the subcommand; its arguments run to the next standalone raw word `&&`, `||`, `;`,
`|`, or `&`, or to the end of the unit. A git word with nothing after it is not an invocation. This
catches `git -C sub push --force` and does not count `git log --grep commit` as a commit.

Decision: the whole-string unit stays as a backstop, because segments are split without regard to
quoting, so a separator glued inside a quoted word (`git push "origin;" --force`) is not a split point
and only the whole string sees the full argument list. Consequence, accepted: `git push origin main
&& echo --force` is allowed because arguments stop at `&&`. Rules are tried in order; within a rule,
segments before the whole string, so the deny message quotes the narrowest text that trips it.

A "cluster" below is a single-dash short-option bundle, `^-[A-Za-z0-9]+$`, such as `-uf` or `-nfd`.

| # | Rule | Denies an invocation with subcommand ... | Reason prefix |
|---|---|---|---|
| 1 | force push | `push` and an argument `--force` or any `--force-*`, `-f`, a cluster containing `f`, or `+<refspec>` | `force push` |
| 2 | remote delete, mirror, prune | `push` and `--delete`, `-d`, a cluster containing `d`, `--mirror`, `--prune`, or `:<ref>` | `remote branch deletion` |
| 3 | push from the wrong caller | `push` when the caller is a subagent other than `pusher` | `only the pusher agent pushes` |
| 4 | commit from the wrong caller | `commit` when the caller is a subagent other than `runner` | `only the runner agent commits; leave the tree for it` |
| 5 | hard reset | `reset` and `--hard` or `--merge` | `destructive reset` |
| 6 | clean | `clean` without `-n`, `--dry-run`, or a cluster containing `n` (`-nfd` allowed) | `git clean deletes untracked files` |
| 7 | whole-tree checkout | `checkout` whose pathspec (words after `--`, otherwise the last argument only) includes `.`, `./`, `*`, or `:/` | `whole-tree checkout discards edits` |
| 8 | whole-tree restore | `restore` whose pathspec (words after `--`, otherwise every non-dash argument) includes one of those four, unless staged (`--staged` or a cluster containing `S`) and not worktree (`--worktree` or a cluster containing `W`) | `whole-tree restore discards edits` |
| 9 | force branch delete | `branch` with a cluster containing `D`, or both a delete flag (`--delete`, cluster containing `d`) and a force flag (`--force`, cluster containing `f`) | `forced branch delete` |
| 10 | interactive rebase | `rebase` and exactly `-i`, `--interactive`, or `--interactive=...` (`-i` inside a cluster is not matched) | `interactive rebase is unsupported here` |
| 11 | stash loss | `stash` whose first non-dash argument is `drop` or `clear` | `stash drop/clear is unrecoverable` |
| 12 | history rewrite | `filter-branch`, `filter-repo`, or `replace` | `history rewrite` |

Anything else: allow (exit 0, no stdout). `--force-with-lease` is denied on purpose; the project never
force-pushes. The main session may push only on the user's instruction (`CLAUDE.md`) and may commit
wrap-ups at any usage level (ADR 0002). Rules 3 and 4 enforce the prose in the four agent files.

Deny output, exit 0, where the advice sentence per rule is the `advice` field of `RULES` in the code:

```json
{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"deny",
 "permissionDecisionReason":"git-guard: <reason prefix>: `<unit text>`. <advice sentence>[ <note>]"}}
```

The note is appended when the git word is not the first command word of its unit (leading
`NAME=value` words are skipped), exactly: `The guard matches command text, so it also fires when a
git command is only quoted (echo, a message); if nothing here runs git, rephrase the text.` Decision:
that position test is the only cheap signal separating `echo "git push -f"` from a real push, and it
also fires for real cases such as `sh -c "git push -f"`, so the note says "also fires" rather than
asserting a false positive.

Failure handling: one try/catch around parsing and deciding. Unparseable stdin (including empty) or an
exception produces the same deny JSON with reason `git-guard: internal error (<message>); command not
run. Fix the hook or ask the user to run it.` The guard fails closed because `onFailure: "block"`
already makes a crash block; the JSON only makes the reason readable.

Test table, harness cases G01-G25 (caller in brackets, main session when omitted): `git status`,
`git push`, `git push` [pusher], `git commit -m x`, `git commit -m x` [runner], `git reset --soft
HEAD~1`, `git clean -n`, `git checkout -- docs/HANDOFF.md`, `git restore --staged .`,
`git branch -d topic`, `git stash list` allow with no output. `git push` [runner] denies rule 3;
`git push origin main --force` [pusher] and `git push -uf origin main` rule 1; `git push origin :old`
rule 2; `git commit -m x` [coder] rule 4; `cd sub && git reset --hard HEAD~1` rule 5; `git clean -fd`
rule 6; `git checkout -- .` rule 7; `git restore .` rule 8; `git branch -D topic` rule 9;
`git rebase -i HEAD~3` rule 10; `git stash drop` rule 11; `echo "git push --force"` rule 1 with the
note appended; stdin `not json` the internal-error deny.

Edge cases GX01-GX25 pin the decisions above: `+main`, `--prune`, `git -C sub push --force`,
`git status\ngit push -f`, `--staged --worktree .`, and `--delete --force` denied; `git log --grep
commit`, `git push origin main && echo --force`, `-nfd`, empty `agent_type`, and no command allowed.

Debug interface: `node .claude/hooks/test-hooks.js` runs every case for all three hooks, PASS/FAIL
per case plus a summary; exit 1 on any failure, 2 on a usage error. Flags: `--only <id-prefix>`,
`--verbose`, `--keep` (keep the scratch docs tree), `--input <file> <hook>` (reproduce one input from
a file; a command line containing a denied git command trips the guard itself).

## 2. `post-edit.js` (PostToolUse, matcher `Write|Edit`, no `onFailure`)

Input: `tool_input.file_path` (non-empty string, else nothing happens), `cwd`. Decision: the project
root is the input `cwd`, falling back to `process.cwd()`; `CLAUDE_PROJECT_DIR` is not used, because
`cwd` is what a relative `file_path` resolves against and the harness drives a scratch tree through
it. The path is resolved against the root; "inside" means the relative path does not start with `..`.
The docs checks run only when the file is inside, under `docs/`, and ends in `.md`. The formatter runs
for any file, inside or not, whose lower-cased extension is in the table. Never exits non-zero; any
exception is swallowed and the hook exits 0 with no output. On Windows the `cwd` must be a Windows
path (`C:/...`); a Git Bash `/c/...` path resolves to a non-existent file and the hook stays silent.

Checks on `docs/**/*.md` (`<path>` is the forward-slash path from the root; line count as `wc -l`,
the number of newline characters):

| Check | Message (one line each, joined with newlines) |
|---|---|
| any file over 300 lines, except under `docs/design/proposals/` | `post-edit: <path> is N lines; cap is 300 (.claude/rules/docs.md). Split by topic; the pusher will refuse to push.` |
| `docs/HANDOFF.md` over 40 lines | `post-edit: docs/HANDOFF.md is N lines; cap is 40. Move detail to docs/backlog.md or a finding.` |
| `docs/findings/*.md` (not README) missing frontmatter or any of `date:`, `status:`, `area:` | `post-edit: <path> lacks finding frontmatter (date, status, area). See the finding skill.` |
| `docs/reference/*.md` (not README) missing `verified:` | `post-edit: <path> lacks verified: frontmatter. See .claude/rules/docs.md.` |
| index: a file at least one directory below `docs/`, basename not `README.md`, `HANDOFF.md`, or `backlog.md`, whose basename appears as a substring in no `README.md` from its own folder up to `docs/<top>/README.md` | `post-edit: <path> is not listed in <folder>/README.md. Add one line.` where `<folder>` is the file's own folder relative to `docs/` |

The walk is why ADRs listed only in `docs/design/README.md` pass. Consequences: files directly in
`docs/` are not index-checked; a proposal named like its ADR passes through the ADR line anyway.

Frontmatter means the file starts with a line `---`, has a closing `---` within the first 20 lines,
and each named key appears at line start between them. Lines are split on LF or CRLF.

Formatter slot: a constant `FORMATTERS = {}` near the top, keyed by lower-case extension with the dot,
value an argv array with the file's absolute path appended at run time, for example
`'.rs': ['rustfmt', '--edition', '2021']`. Empty in the base; a comment tells the decanted project to
fill it. Test seam: a JSON object in the env var `POST_EDIT_FORMATTERS_JSON` is merged over
`FORMATTERS`. A matching entry runs as `spawnSync(argv[0], [...argv.slice(1), abs], { timeout: 20000,
encoding: 'utf8' })`, no shell, so on Windows an npm `.cmd` shim cannot be the command; name the real
executable or `node` plus the script path. Failure is a spawn error or a non-zero status; the message
is `post-edit: formatter <argv[0]> failed on <path>: <detail>`, where detail is the first 5 non-empty
stderr lines, else the spawn error message, else `killed by <signal> (timeout 20 s?)`.

Output when there is at least one message, exit 0:

```json
{"hookSpecificOutput":{"hookEventName":"PostToolUse","additionalContext":"<messages joined by \n>"}}
```

No messages: exit 0, no stdout. Harness cases P01-P04 cover a 301-line reference file, a 41-line
copy of the real `docs/HANDOFF.md`, a finding without frontmatter, and a clean file; PX01-PX11 a
reference file missing from its index, an ADR listed only in `docs/design/README.md`, the proposals
cap exemption, a relative `file_path`, files outside `docs/`, malformed stdin, and a formatter that
succeeds and one that fails. All run in a scratch `docs/` tree under the OS temp dir.

## 3. `usage-gate.js` (exists; text change only)

HOLD reason becomes: `Usage gate: <detail>. At or above 50%: do not start new subagents. Finish
in-flight work in this session, run the handoff skill, commit from the main session, then report.`
STOP reason becomes: `Usage gate: <detail>. At or above 80%: stop all work, run the handoff skill,
commit WIP from the main session, end the turn.` Thresholds, inputs, and allow paths are unchanged.
