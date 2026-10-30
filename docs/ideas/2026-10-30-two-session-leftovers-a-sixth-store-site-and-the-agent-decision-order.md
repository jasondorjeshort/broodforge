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
