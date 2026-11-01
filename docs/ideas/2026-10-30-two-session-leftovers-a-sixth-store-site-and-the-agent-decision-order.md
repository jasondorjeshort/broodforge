---
date: 2026-10-30
kind: bug
---

# Two session leftovers: a sixth swallowed store write, and the Agent decision order

Reviewers found two residuals after the internal-error work. First, five swallowed store writes were
moved to the `internal-error` event, but a sixth, `setEngineSession`, still drops its error. Second,
an `Agent` call's decision prints after its subagent's block, because `turn.go` emits the decision just
before the result rather than when the call starts.

Anchor: `internal/session/turn.go:527` (`setEngineSession`) and the decision emission in
`internal/session/turn.go`.

Plan: tester. Part one, confirmed 2026-10-31: `setEngineSession` (`internal/session/turn.go:527`) still
discards the `Store.Write` error (`_ =`), while the five other sites add `storeFailed(...)` (the
`internal-error` event, `turn.go:840`) to `l.log`. Route the sixth the same way; the call site at
`turn.go:343` has the turn number `n`, so pass it in. Part two: the decision event is built in
`toolResult` (`turn.go` around 563 to 602) from `Classify(tr, ...)`, the tool's result, so it cannot be known
when the call starts; a subagent's events arrive before the `Agent` call's own result. So place the
decision line directly after its `tool-call` line when the turn is rendered (or insert it at the call's
position in the log); do not guess a decision early. If that proves a session-service change beyond a test
pass, stop and report for the coder. Update `tests/discord/golden/subagent-end/` and any other golden that
moves. Verify: `go test ./internal/session/... ./internal/discord/...` and the golden run
(`discord render --golden`). Out of scope: other event ordering, the closing-message idea.
