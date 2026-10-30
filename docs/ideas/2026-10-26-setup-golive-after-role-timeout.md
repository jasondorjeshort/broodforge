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

Plan: tester, test first. `internal/setup/fakediscord_test.go`: `TestRole` gains "held on the second
run goes live" (roles hold `Trusted`, owner lacks it, `--create-role no`; run one writes
`roleGroups.trusted`, refuses `timeout-role`; `f.edit` gives the role; run two prints `ok  role
Trusted  <id>  held by <owner>; discord.mode live`, file reads `live`). `internal/setup/discord.go`:
`goLive` drops the `s.wrote` gate and calls `markWrote` for the role step when it switches, since
`passed()` (74-79) prints `ok` only for a step in `s.wrote`, else `skip ...; discord.mode live`.
"The run that fills the last Discord id" in `../design/decisions/0027-setup-commands.md` and
`../design/05-debug-interface-setup.md` becomes "the first run in which every Discord id is present
and the role is held". Verify: `go test ./internal/setup -run
'TestRole|TestRerun|TestDiscordSteps|TestDiscordGuards'`, check line. Not `next` or P20 W6.
