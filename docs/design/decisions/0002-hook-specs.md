# ADR 0002 companion: hook specifications

Behaviour for the three scripts under `.claude/hooks/`, exact enough to implement and test without
asking. Both new scripts: Node, `'use strict'`, no dependencies outside the standard library, SPDX
header on line 2 after the shebang, read all of stdin as JSON, run under Git Bash on Windows and under
bash on Linux. Reference: https://code.claude.com/docs/en/hooks (input fields, output JSON, exit codes).

Facts the design relies on (from that page): exit 0 with JSON on stdout means "the fields decide";
exit 1 without valid JSON is a non-blocking error, so a policy hook must never rely on exit 1; exit 2
is a blocking error with stderr as the reason; `hookSpecificOutput` must wrap `additionalContext`, a
top-level field is ignored; `agent_type` is present only when the hook fires inside a subagent (or
with `--agent`); `onFailure: "block"` turns a crash, timeout, or missing interpreter into a block.

## 1. `git-guard.js` (PreToolUse, matcher `Bash`, `"onFailure": "block"`)

Staging note: `.claude/settings.json` points at this script without `onFailure` until it exists,
because with `onFailure: "block"` a missing script blocks every Bash call in the repository
(observed 2026-10-11). Last step of implementing this hook: run the test table, then add
`"onFailure": "block"` to the git-guard entry in `.claude/settings.json` and run `git status` once
to confirm Bash still works.

Input: `tool_input.command` (string), `agent_type` (string, optional), `cwd`.

Decision rules, evaluated in order against the normalised command (tabs and runs of spaces collapsed
to one space; evaluated over the whole string, and additionally over each segment split on `&&`,
`||`, `;`, `|`, and newlines, so that `cd x && git push -f` is caught):

| # | Rule | Regex sketch (applied per segment, `\bgit\b` must precede) | Deny reason prefix |
|---|---|---|---|
| 1 | force push | `push` segment containing `--force`, `--force-with-lease`, `--force-if-includes`, `-f`, or a short cluster containing `f` (`-uf`) | `force push` |
| 2 | remote delete or mirror | `push` segment containing `--delete`, `-d`, `--mirror`, or a refspec starting with `:` | `remote branch deletion` |
| 3 | push from the wrong caller | `push` (any form) when `agent_type` is present and is not `pusher` | `only the pusher agent pushes` |
| 4 | commit from the wrong caller | `commit` when `agent_type` is present and is not `runner` | `only the runner agent commits; leave the tree for it` |
| 5 | hard reset | `reset` segment with `--hard` or `--merge` | `destructive reset` |
| 6 | clean | `clean` segment without `-n` or `--dry-run` | `git clean deletes untracked files` |
| 7 | whole-tree checkout | `checkout` whose pathspec (after `--`, or last arg) is `.`, `*`, or `:/` | `whole-tree checkout discards edits` |
| 8 | whole-tree restore | `restore` with pathspec `.`, `*`, or `:/`, unless `--staged` present and `--worktree` absent | `whole-tree restore discards edits` |
| 9 | force branch delete | `branch` segment with `-D`, or both `--delete` and `--force` | `forced branch delete` |
| 10 | interactive rebase | `rebase` segment with `-i` or `--interactive` | `interactive rebase is unsupported here` |
| 11 | stash loss | `stash drop` or `stash clear` | `stash drop/clear is unrecoverable` |
| 12 | history rewrite | `filter-branch`, `filter-repo`, or `replace` | `history rewrite` |

Anything else: allow (exit 0, no stdout). `git push --force-with-lease` is denied on purpose; the
project never force-pushes. Rule 3 and 4 identify the caller by `agent_type`; when the field is absent
the caller is the main session, which may push only by the user's instruction (rule in `CLAUDE.md`)
and may commit wrap-ups at any usage level (ADR 0002). Rules 3 and 4 are the enforceable form of the
prose in `coder.md`, `tester.md`, `runner.md`, and `pusher.md`.

Deny output, exit 0:

```json
{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"deny",
 "permissionDecisionReason":"git-guard: <reason prefix>: `<offending segment>`. <one sentence on what to do instead>"}}
```

Failure handling: wrap everything in one try/catch. Unparseable stdin, or an exception, produces the
same deny JSON with reason `git-guard: internal error (<message>); command not run. Fix the hook or ask
the user to run it.` A missing `tool_input.command` allows. The guard fails closed because
`onFailure: "block"` already makes a crash block; the JSON only makes the reason readable.

Test table (run as `printf '%s' '<json>' | node .claude/hooks/git-guard.js; echo "exit $?"`; the
`agent_type` column is the top-level input field):

| command | agent_type | expect |
|---|---|---|
| `git status` | - | allow, no output |
| `git push` | - | allow |
| `git push` | `pusher` | allow |
| `git push` | `runner` | deny rule 3 |
| `git push origin main --force` | `pusher` | deny rule 1 |
| `git push -uf origin main` | - | deny rule 1 |
| `git push origin :old` | - | deny rule 2 |
| `git commit -m x` | `coder` | deny rule 4 |
| `git commit -m x` | `runner` | allow |
| `git commit -m x` | - | allow |
| `cd sub && git reset --hard HEAD~1` | - | deny rule 5 |
| `git reset --soft HEAD~1` | - | allow |
| `git clean -fd` | - | deny rule 6 |
| `git clean -n` | - | allow |
| `git checkout -- .` | - | deny rule 7 |
| `git checkout -- docs/HANDOFF.md` | - | allow |
| `git restore --staged .` | - | allow |
| `git restore .` | - | deny rule 8 |
| `git branch -D topic` | - | deny rule 9 |
| `git branch -d topic` | - | allow |
| `git rebase -i HEAD~3` | - | deny rule 10 |
| `git stash list` | - | allow |
| `git stash drop` | - | deny rule 11 |
| `echo "git push --force"` | - | deny rule 1 (false positive accepted; the message says so) |
| `not json` | - | deny, internal error |

Debug interface: `node .claude/hooks/test-hooks.js` runs every row above, the post-edit and
usage-gate cases, and extra edge cases, printing PASS/FAIL per case and a summary; exit 1 on any
failure. `--only <id-prefix>`, `--verbose`, and `--input <file> <hook>` (reproduce one input from a
file, because a command line containing a denied git command trips the guard itself).

## 2. `post-edit.js` (PostToolUse, matcher `Write|Edit`, no `onFailure`)

Input: `tool_input.file_path`, `cwd`. Resolve the path relative to `cwd`; act only when it is inside
`docs/` and ends in `.md`, or when its extension is in the formatter table. Never exit non-zero; any
exception is swallowed and the hook exits 0 with no output.

Checks on `docs/**/*.md` (line count as `wc -l`: number of newline characters):

| Check | Message (one line each, joined with newlines) |
|---|---|
| any file over 300 lines, except under `docs/design/proposals/` | `post-edit: <path> is N lines; cap is 300 (.claude/rules/docs.md). Split by topic; the pusher will refuse to push.` |
| `docs/HANDOFF.md` over 40 lines | `post-edit: docs/HANDOFF.md is N lines; cap is 40. Move detail to docs/backlog.md or a finding.` |
| `docs/findings/*.md` (not README) missing frontmatter or any of `date:`, `status:`, `area:` | `post-edit: <path> lacks finding frontmatter (date, status, area). See the finding skill.` |
| `docs/reference/*.md` (not README) missing `verified:` | `post-edit: <path> lacks verified: frontmatter. See .claude/rules/docs.md.` |
| file under `docs/<dir>/` whose basename does not appear in `docs/<dir>/README.md` (skip README itself and `HANDOFF.md`, `backlog.md`) | `post-edit: <path> is not listed in <dir>/README.md. Add one line.` |

Frontmatter means the file starts with a line `---`, has a closing `---` within the first 20 lines,
and the named keys appear between them at line start.

Formatter slot: a constant `FORMATTERS = {}` near the top, keyed by extension, value an argv array with
the file path appended at run time, e.g. `['.rs']: ['rustfmt', '--edition', '2021']`. Empty in the
base; a comment tells the decanted project to fill it. When an entry matches, run it with
`child_process.spawnSync(argv[0], [...argv.slice(1), file], { timeout: 20000 })`; on non-zero exit or
timeout append `post-edit: formatter <argv[0]> failed on <path>: <first 5 lines of stderr>`.

Output when there is at least one message, exit 0:

```json
{"hookSpecificOutput":{"hookEventName":"PostToolUse","additionalContext":"<messages joined by \n>"}}
```

No messages: exit 0, no stdout. Test with a temporary 301-line file under `docs/reference/`, a
41-line `docs/HANDOFF.md` copy, a findings file without frontmatter, and a clean file; delete the
temporaries afterwards and show the four outputs in the report. `test-hooks.js` (cases P01-P04)
does this in a scratch `docs/` tree under the OS temp dir and prints each output.

## 3. `usage-gate.js` (exists; text change only)

HOLD reason becomes: `Usage gate: <detail>. At or above 50%: do not start new subagents. Finish
in-flight work in this session, run the handoff skill, commit from the main session, then report.`
STOP reason becomes: `Usage gate: <detail>. At or above 80%: stop all work, run the handoff skill,
commit WIP from the main session, end the turn.` Thresholds, inputs, and allow paths are unchanged.
