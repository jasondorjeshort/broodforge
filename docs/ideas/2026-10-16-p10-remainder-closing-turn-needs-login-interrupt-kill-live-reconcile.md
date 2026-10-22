---
date: 2026-10-16
kind: feature
---

# P10 remainder: closing turn, needs-login, interrupt kill, live audit and reconcile

P10 (`docs/design/08-implementation-plan.md`) landed without these four pieces.

## Closing turn at session end

Anchor: `docs/design/03-architecture.md` (End of session); `internal/session/actions.go` (the
closing-turn comment). Today the fallback title always stands. Done: a `closing` fixture under
`fixtures/engine/`, recorded with the real engine, drives the fake and `session end` uses its
title and body. It waits on an owner turn through `engine record`, so nothing is plannable yet.

2026-10-17 (coordinator): the first closing recording, with a prompt that told the engine to use
`git log` and `git diff`, ended `tool_deferred` on its first Bash, because the standalone run takes
`defaults.mode` and the owner root is in prompt mode; a session-end turn has nobody to click, and
under read-only reach Bash is denied outright. Decision: the closing turn uses no tools. `session`
runs `git log` and `git diff --stat` over base..HEAD itself and inlines both in the fixed prompt,
which also says not to run tools; the turn is then mode- and reach-independent and one process. The
`closing` fixture is recorded with that prompt shape.

## Auth error marks the credential needs-login

Anchor: `docs/design/03-architecture.md` (Failure modes, `engine` row); `internal/credentials/service.go`
(`StatusNeedsLogin`); `internal/session/turn.go`. Done: an auth-error `result` sets `needs-login`
(an API key keeps its status) and ends the session through the normal sequence, no closing turn.

Plan: coder, one pass. Auth-error fixture; set the status from the turn's result handler; reuse
the end sequence; test with `-race`.

## Interrupt timeout kill

Anchor: `spawn.interruptTimeoutSeconds` (`internal/spawn/spawn.go`, `DefaultInterrupt`);
`internal/session/actions.go` (`Interrupt`). The fake interrupts at once, so this is untested.
Done: an engine still alive that long after `Interrupt` is killed and the turn ends interrupted.

Plan: coder, one pass. Injectable clock; a fake mode that ignores the interrupt; test the kill
comes at the timeout and not before.

## Audit follow and on-demand reconcile while the bot runs

Anchor: `internal/cli/audit.go` (`--follow`); `internal/cli/reconcile.go`. Both fall back to the
store file and report `not-running`. Done: with the bot up both use the control socket,
`audit --follow` streams, and reconcile skips the bot's live sessions.

Plan: coder, one pass. Socket stream for audit; a reconcile call excluding the live set; a repro
in `tests/repro/` against a serving bot.

2026-10-22 (coordinator, the first live run): `audit --follow` returns `not-running` before it
dials anything (`internal/cli/audit.go`, the `follow` branch), so with the bot up and `status`
answering over the pipe it still says `no bot at <pipe>`, and the owner chased a pipe that was
fine. The first pass makes `--follow` real; until it lands the message should be `not-built`.
`session tail <id> --follow` is the live view meanwhile.

## Open

Whether `session invite` checks the invitee's roles before Discord (P12) can.
