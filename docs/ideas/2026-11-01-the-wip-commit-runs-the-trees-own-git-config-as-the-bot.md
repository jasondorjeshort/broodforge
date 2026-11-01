---
date: 2026-11-01
kind: bug
---

# The WIP commit runs the tree's own git config as the bot

The reviewer found, on 2026-11-01, that the session-end WIP commit (`internal/spawn/worktree_wip.go`,
landed in 5891886) runs `git status` and `git add` in the session tree as the bot's account. A tree's
own `.git/config` can set `core.fsmonitor` and `filter.<name>.clean`, and both name a program git runs
during exactly those two commands. The agent can write that file, so whatever it names runs as the bot.

On the PC the account is the session's own (ADR 0022, Windows), so nothing escalates: the program
runs as the user who already owns the tree. The bwrap backend is different. Its `WorktreeWIP` is a
stub (`internal/spawn/bwrap.go`), and it must not get a real one while the commit runs as the bot.

Fix, either of: run the commit as the session's user, or disable the settings on the command line
with `-c core.fsmonitor=false` and an empty `filter` override, so the tree's config cannot name a program.

Anchor: `internal/spawn/worktree_wip.go` (the `git status` and `git add` calls) and
`internal/spawn/bwrap.go` (the `WorktreeWIP` stub that stays a stub until then).

## Open

- Which settings need disabling on the command: `core.fsmonitor`, `filter.*.clean` and `.smudge`,
  `core.hooksPath`, `diff.*.textconv`; or whether running as the session's user is the simpler rule.
- A test: a tree whose config sets `core.fsmonitor` to a script that writes a marker; the WIP commit
  leaves no marker.
