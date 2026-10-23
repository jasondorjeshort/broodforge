# Implementation plan: the session's workflow (ADR 0025)

This file continues `08-implementation-plan.md` under the same conventions: the pass format, the size
words, the standing rules and the `[no-check: ...]` rule. It holds the passes that follow ADR 0025,
P15 to P19, which are not held on P14.

The order is ADR 0025's Consequences, after `cb046d4`. Each pass reads the ADR (Decision) and its
companion `decisions/0025-refusals-and-checks.md` before its Implements line; the companion's texts
are the owner's and carry "wording: owner to confirm" until the handoff's Waiting on user clears
them. A check id (C1 to C6) is the companion's Open checks row the pass closes.

## P15. The session never pushes: `pre-push` and `worktree check` (small)

Implements: ADR 0025 rule 4; companion (Refusals and lifts: the `pre-push` row; Open checks: C1);
`05-debug-interface-git.md` (Git handoff: `worktree check`; the hooks under `share/hooks/`). Files
[no-check: `share/hooks/pre-push` and `internal/cli/git.go` are created by this pass]:
`share/hooks/pre-push` (shebang, SPDX line, then: when `bin/broodforge` or `.exe` beside `share/` is
executable run the hidden `broodforge git-pre-push`, else print the refusal line itself; `exit 1` on
every path, unlike `commit-msg`, which is fail-open by design; the clone's `core.hooksPath` already
names the directory, so every tree P8 creates gets it); `internal/cli/git.go` (`git-pre-push`:
prints to stderr, exit 1, the line `Broodforge: session <id> does not push. Commit in this tree;
Broodforge pushes at the handoff when the session ends.` (wording: owner to confirm), `<id>` from
`BROODFORGE_SESSION`, `this tree` when it is unset); `internal/spawn/` (a `Backend` method reading
the tree's `core.hooksPath` and the file's presence, since the bot's git never opens the session
tree, ADR 0022); `internal/cli/worktree.go` (`worktree check` gains, as its first row before the
bundle and the per-commit lines, `ok  pre-push` when the hooks path is the bot's directory and the
file exists, else `missing  pre-push` and exit 1 with `error: pre-push-missing`; `handoff` is not
refused for it, the tree's configuration being the session's). `git push --no-verify` and `git -c
core.hooksPath= push` skip the hook (the companion's lift column): a refusal with a message, never
the boundary, which stays the droplet's read-only canonical bind and the missing credential (the
ADR). Debug surface: `git -C <tree> push origin HEAD` from a terminal inside the tree, and
`worktree check <id>`. Proof: P8's test pushes from the tree to `origin` and to the canonical's
path (through `os/exec`); both exit 1, stderr's first line the text above, git's own `error:
failed to push some refs` after it, the canonical unchanged; with `bin/` renamed away the push
still exits 1 and the script prints the same line itself; `git push --no-verify origin HEAD` to the
test remote succeeds (the lift, recorded); `worktree check s0001` prints `ok  pre-push` first;
after `git config --unset core.hooksPath` in the tree it prints `missing  pre-push` and exits 1.
Closes C1 (PC half; the droplet repeat is P14's). Does not: set `pushurl`; parse git commands in
the bot's hook; change the handoff. Depends on: P8.

## P16. The push scan (small)

Implements: ADR 0025 rule 7 (the scan; the review fact is P19's); companion (Refusals and lifts: the
push-scan row); ADR 0018 through the base's `.claude/hooks/privacy-scan.js` (the classes and the
report shape, ported, never run); `05-debug-interface-git.md` (Git handoff: refusals). Files
[no-check: `internal/git/privacy.go` is created by this pass]: `internal/git/privacy.go` (over
each commit of `<base>..refs/sessions/<id>`: message, author, committer and added lines; the base's
five classes: a home path with a user segment in any spelling, an email, a name outside the
identities, a host name, a session link; identities for a session's scan are `privacy.identities`
plus the session's registered participants' names and emails and the model co-author address with
the `policy` display names, read from the store, since every session commit carries them
(`internal/git/attribution.go`); the `Claude-Session:` trailer is exempt as in the base; the never
list and allow list are the instance's; reports `privacy: <sha>  <class>  <location>` with
`<location>` `<path>:<line>` for an added line, `message:<line>` for the message, `author` or
`committer` for those fields, never the value); `internal/config/types.go` (`Privacy{Never, Allow,
Identities []string}` on `Instance`, `config check` refusing an empty `never` entry);
`internal/cli/worktree.go` (`worktree check` and `worktree handoff` run the scan after the
attribution check: a hit prints the `privacy:` lines, then `error: handoff-refused-privacy: <n> of
<m> commits carry personal data`, exit 1, the tree kept, the venue line the companion's text,
04's handoff-refused flow). Debug surface: `worktree check <id>` with its `privacy:` lines. Proof:
P8's test adds a commit whose message carries an address and a commit adding a file line with a
placeholder home path: `worktree check s0001` prints `privacy: <sha>  email  message:1` and
`privacy: <sha>  home-path  <path>:<line>`, then the `handoff-refused-privacy` error, exit 1; a
commit whose author and `Co-Authored-By` lines are the seeded driver and model passes; `handoff`
refuses the same; the path's user segment added to `privacy.allow` lifts that one; after an amend
`worktree check` is `ok`. Closes no companion check; its fixture is the check. Does not: run the
repository's `pre-push`; read the copy's `policy.json` (P17 adds that section once the copy
exists); scan session messages. Depends on: P4, P8.

## P17. The workflow copy, the `workflow` key, the runtime probe (medium)

Implements: ADR 0025 rules 5 and 6; companion (The copy procedure; The runtime probe; Refusals and
lifts: the probe-failed row); `03-architecture.md` (Components: a `workflow` row this pass adds,
and `.claude/rules/go.md`'s `cli` exception gains it; Configuration: the `workflow` key);
`05-debug-interface.md` (The workflow copy: `workflow render|probe`). Files [no-check:
`internal/workflow/`, `internal/cli/workflow.go` and `fixtures/workflow/` are created by this pass]:
`internal/workflow/` (`git ls-tree`, then `git archive` of what it listed, extracted with
`archive/tar`; the `@` import resolver as the base's `imports.js`; `instructions.md` without the
header, which is P19's; `agents.json` from the frontmatter fields the ADR names; `hooks.json` with
`{event, matcher, command, name, timeout}` filtered by `workflow.hooks`, `name` the base name of
the first word of `command` that names a file (a path separator or a script suffix, after the
`${CLAUDE_PROJECT_DIR}` forms are read as the copy), else the first word, a repeat suffixed `-2`,
`-3`; the probe per first word); `internal/config/types.go` (`Workflow{Inject string; Hooks
[]string}` on `Project`; `config check`: `inject` is `repository` or `none`, `hooks` only the two
events); `internal/git/privacy.go` reads the copy's `.claude/hooks/policy.json` `privacy` section
when the copy exists; `internal/cli/workflow.go` (`workflow render --project <id> --out <dir>
(file)` and `workflow probe --project <id> (file)` as 05 describes; `config check` runs the same
probe over each project that lists hooks, rendered to a temporary directory, a failure an error
naming the command and the word, rule 6); `fixtures/workflow/` (a
canonical tree: a `CLAUDE.md` with one `@` import and a missing one, two agents, three rules, a
`.claude/hooks/policy.json` whose `privacy.identities` lists one placeholder identity, a
`.claude/hooks/git-guard.js` and `post-edit.js` that print one line each, and a `settings.json`
with `PreToolUse` `Bash|PowerShell` running `node "${CLAUDE_PROJECT_DIR:-.}/.claude/hooks/git-guard.js"`,
`PostToolUse` `Write|Edit` running the same form of `post-edit.js`, and a `Stop` entry). Debug
surface: the two `workflow` commands and `fixtures/workflow/`. Proof: on a scratch root whose
canonical is the fixture, `workflow render` prints `workflow: <commit> <date>` and writes
`instructions.md` (the import's text, a `note:` line for the missing one, the rules in name order),
`agents.json` (two entries, bare names) and `hooks.json` (two entries named `git-guard.js` and
`post-edit.js`, the `Stop` entry absent); with `"hooks": []` no `hooks.json`; a canonical with
neither file prints the `none` line and writes `instructions.md` alone; `config check` with
`"hooks": ["Stop"]` exits 1 naming it; `workflow probe` prints `ok  git-guard.js  node` and `ok
post-edit.js  node` on the PC, and with a command `nosuchruntime x.js` the not-started line, exit
1; P16's test, given a copy rendered from the fixture, passes a commit whose `Co-Authored-By`
names the fixture's identity and refuses it with no copy. Closes no companion check. Does not:
spawn anything, run a child, set `BROODFORGE_WORKFLOW`, use `--add-dir` or `--plugin-dir`, copy per
turn. Depends on: P4, P8, P16.

## P18. The hook child chain and `hook post-tool-use` (medium; one real turn: the owner's)

Implements: ADR 0025 rule 2; companion (Refusals and lifts: child deny, over budget, cannot start;
Open checks: C2, C3); `03-hook-contract.md` (a chain section and the `BROODFORGE_WORKFLOW` row
this pass adds); `05-debug-interface.md` (The PreToolUse hook: `hook try --workflow`, `hook cases`;
`engine run --workflow`). Files [no-check: `internal/hook/chain.go`, `internal/hook/post.go` and
`tests/hook/chain/` are created by this pass]: `internal/hook/chain.go` (reads
`$BROODFORGE_WORKFLOW/hooks.json`; code reach only, after the bot's own non-deny decision; matches
`tool_name` against each `matcher` as the engine's hooks documentation defines it, a case-sensitive
regular expression against the whole tool name, empty or `*` matching every tool, an invalid
pattern never matching and logged; runs children serially through the shell named by
`CLAUDE_CODE_GIT_BASH_PATH` on Windows, which the spawn exports from `engine.gitBashPath`
(`internal/spawn/sameaccount.go`), unset there a cannot-start deny naming the shell, and `/bin/sh`
elsewhere; cwd the tree, `CLAUDE_PROJECT_DIR` the copy, the engine's stdin bytes replayed; budget
the smaller of the entry's `timeout` and ten seconds, twenty for the chain; takes exit 2 and the
`deny` field only, everything else to `BROODFORGE_HOOK_LOG`; the three companion texts verbatim);
`internal/hook/post.go` and `hook post-tool-use` in `internal/cli/hook.go` (the companion's
`additionalContext` pass-through, exit 0 always); `internal/spawn/` (the settings template registers
`post-tool-use`, empty matcher, thirty seconds); `internal/cli/engine.go` (`engine run --workflow
<dir>` adds `BROODFORGE_WORKFLOW` to the standalone hook-variable list and exports the git-bash
name, so a real turn can run the chain without a session); `tests/hook/chain/` (`sh` children, one
per case: deny by exit 2, deny by JSON, allow with `updatedInput`, sleep past budget, a
context-printing `PostToolUse` child, and a `hooks.json` naming them plus a command that does not
exist); `tests/hook/cases.jsonl` gains a `workflow` field. Debug surface:
`BROODFORGE_WORKFLOW=tests/hook/chain BROODFORGE_REACH=code ... | broodforge hook pre-tool-use`,
`hook try --tool Bash --input '{"command":"x"}' --workflow tests/hook/chain --expect deny`, `hook
cases`, `engine run --real --workflow <dir>`. Proof: `hook cases tests/hook/cases.jsonl` all `ok`:
exit-2 deny and JSON deny return `Denied by Broodforge: refused by the project's hook <name>: ...
The project's rules on its default branch decide this.`; `updatedInput` is dropped, the bot's
decision returned unchanged, the line in `hook.log`; the sleeper yields the over-budget text naming
it and ten seconds; the missing command the cannot-start text; read-only reach and an unset
`BROODFORGE_WORKFLOW` run no child; `hook post-tool-use` emits the child's context, exit 0; a timed
test runs this repository's `.claude/hooks/git-guard.js` and `post-edit.js` as children on the PC,
each under ten seconds, the whole hook under thirty (C3). Then, on the owner's word, one real turn
through `engine run --real --workflow tests/hook/chain --cwd <tmp tree>` asking for an edit: the
`PostToolUse` child's context appears in the transcript and the stream (C2). Closes C2, C3. Does
not: run any event but the two; take `allow`, `ask`, `defer` or `updatedInput`; merge anything into
the engine's hook list; wire a session (P19). Depends on: P6, P7, P17.

## P19. Injection in a session: the header, the flags, the `session show` lines (medium; owner sees the header; real turns: the owner's)

Implements: ADR 0025 rules 1, 3, 5 (the spawn half), 6 (the start-time probe), 7 (the review fact),
9 and 10; companion (Refusals and lifts: the probe-failed row; Open checks: C5, C6); `04-ux.md`
(the header; the not-started flow); `05-debug-interface-session.md` (`session show`: `workflow:`,
`hooks:`, `review:`; `session start`'s refusal). Files [no-check: P17's and P18's files, extended
here]: `internal/session/prompt.go` (the header: the precedence sentence verbatim, session,
project, reach, driver, one call at a time, the trailer rule of rule 10 with no list, the
taken-from line; `SystemPromptAppend` loses the trailer list); `internal/session/service.go` (at
start: the copy through `workflow`, then the probe, a failure refusing the start with `error:
workflow-probe-failed: <command> needs <word>`, exit 1, the venue line the companion's not-started
text, audit `action-refused`; `BROODFORGE_WORKFLOW` and the two flags on the spawn,
`--append-system-prompt` retired, the file flag passed always, the header alone under `inject:
none`; `workflow:` and `hooks:` from the copy; the review fact: a `reviewer` `Agent` call's result
row against the committer time of the last commit as of the last bundle, refreshed by `worktree
check`); `internal/session/events.go` (the `session tail` indentation from `parent_tool_use_id`,
`internal/engine/parse.go`); `internal/engine/claudecode.go` (`Args`: the file flag and
`--agents`); `internal/spawn/` (`BROODFORGE_WORKFLOW` in the environment array, so the 0022
companion's `env_keep` follows); `tests/repro/full-session.sh` gains the lines. Debug surface:
`session show <id>`, `received-<n>.json`, `session tail` (a subagent's calls indented under their
`Agent` line), `session start` against a project whose fixture names `nosuchruntime`. Proof: a
scratch session on the P17 fixture: `received-1.json` carries `--append-system-prompt-file` and
`--agents`, no `--append-system-prompt`; `session show s0001` prints `workflow: <commit> <date>`,
`hooks: git-guard.js, post-edit.js` and `review: none since <commit>`; with `inject: none` it
prints `workflow: none`, `received-1.json` carries the file flag (the header alone) and no
`--agents`; a project whose `settings.json` names `nosuchruntime x.js` with `hooks` listing
`PreToolUse`: `session start` exits 1 with `workflow-probe-failed`, `session outbox` holds the
not-started line, `audit` the `action-refused` row; the fake's `full-session.sh` exits 0. Then, on
the owner's word, in this repository's channel with the real engine: the driver launches
`reviewer` and `session tail` shows its calls indented with `decided by hook` (C6); an agent given
`tools: Read` fails to write (C5); after a commit and `worktree check`, `session show` prints
`review: <tool_use_id> after <commit>`. Closes C5, C6. Does not: land on `main` (the ideas wait on
this pass); add `--plugin-dir`; change the chain. Depends on: P10, P12, P17, P18.
