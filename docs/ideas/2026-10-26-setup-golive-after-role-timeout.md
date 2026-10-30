---
date: 2026-10-26
kind: bug
---

# `setup` stays dry-run after a role timeout, while `next` says the bot is running

`goLive` switches `discord.mode` from `dry-run` to `live` only when this run wrote a Discord key.
If a run writes `roleGroups.trusted` and then times out waiting for the owner to hold the role
(`internal/setup/discord.go`, the role step's timeout), the next run finds every key present and
writes nothing, so the mode stays `dry-run`, while `next` prints "the bot is running".

Found by review from reading, not reproduced; needs a test first (a scripted Discord whose owner
does not hold the role on the first run, then does on the second; assert `discord.mode` after the
second run).

Searched `docs/ideas/`, `docs/findings/` and `docs/HANDOFF.md` for overlap: none.

Plan: tester, one pass; the test first. Files: `internal/setup/fakediscord_test.go` (`TestRole`
gains "held on the second run goes live": a scripted guild whose roles already hold `Trusted`,
`ownerRoles` empty, the bot without Manage Roles, `--create-role no`; the first run writes
`roleGroups.trusted` and refuses `timeout-role`; `f.edit` gives the owner the role; the second
run's line is `ok  role  Trusted  <id>  held by <owner>; discord.mode live` and the instance
file's `discord.mode` reads `live`; this fails on the current tree); `internal/setup/discord.go`
(`goLive`: drop the `s.wrote` gate; the switch is made in the role step of any run but `--check`
in which the four ids are present, the role is held and the mode is `dry-run`; an owner who wants
a configured instance dry for one run has `serve --discord dry-run`); the sentence "the run that
fills the last Discord id" in `../design/decisions/0027-setup-commands.md` (a dated `## Amendments`
line, `adr` skill) and under Drive in `../design/05-debug-interface-setup.md` becomes "the first
run in which every Discord id is present and the role is held". Verify: `go test ./internal/setup
-run 'TestRole|TestRerun|TestDiscordSteps|TestDiscordGuards'`, then the check line. Out of scope:
`next`'s text, right once the mode is live; the P20 W6 scheduled-task work.
