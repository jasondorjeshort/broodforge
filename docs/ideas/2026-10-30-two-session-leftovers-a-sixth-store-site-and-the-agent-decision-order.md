---
date: 2026-10-30
kind: bug
---

# Two session leftovers: a sixth swallowed store write, and the Agent decision order

Reviewers found two residuals after the internal-error work. First, five swallowed store writes were
moved to the `internal-error` event, but a sixth, `setEngineSession`, still drops its error. Second,
an `Agent` call's decision prints after its subagent's block, because `turn.go` emits the decision just
before the result rather than when the call starts.

Anchor: `internal/discord/transcript.go:81` (the `EventPermissionDecision` case), for the part that remains.

Plan: coder. Part one done 2026-11-01: `setEngineSession` routes its store error through `internal-error`
(`TestEngineSessionWriteFailureIsInternalError`). Part two is a `coder` pass in the Discord transcript
(`internal/discord/transcript.go`): the `tool-call` line arrives first and the decision after the subagent's block,
so move the decision up to right after its `tool-call` line (`Transcript` receives the whole event slice, so a
pre-pass from ToolUseID to decision works). `render.go` only if the coder finds a decision line there
(`View.decision` posts nothing for a policy allow). The session log stays untouched
(append-only, seq-numbered, followers read from a seq position); do not guess a decision early. The
`subagent-end` transcript golden moves, `outbox.jsonl` probably does not; update them and any other golden that
moves. Verify: `go test ./internal/discord/...` and `bash tests/repro/golden-sessions.sh`. Out of scope: other
event ordering, the closing-message idea.
