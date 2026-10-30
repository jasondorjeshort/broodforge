---
date: 2026-10-30
kind: bug
related: [2026-10-30-a-transcript-decision-line-names-neither-its-call-nor-its-depth.md]
---

# A store failure inside a turn is logged as a permission decision

`internal/session/turn.go` (the decision write near line 591, `subagentResult` near 787 and the new
`saveMeter` near 803) logs a store failure as `EventPermissionDecision` with only
`Summary: "store: <err>"`. `internal/discord/transcript.go` (lines 76 to 86) then renders it as
`  -  ()` and drops the error text; `internal/discord/render.go` (`decision`, near 504) posts
nothing; only `session tail` shows it. P28 followed its plan ("as `subagentResult` writes one"). No
session event type carries internal errors, and `docs/design/03-architecture.md` does not say where
they go. Found by a reviewer, pre-existing.

Acceptance test (the reviewer's failing test), `internal/discord/transcript_test.go`:

```go
func TestTranscriptStoreFailureIsNoDecision(t *testing.T) {
	evs := []session.Event{
		{Type: session.EventTurnStarted, Turn: 1},
		{Type: session.EventPermissionDecision, Turn: 1, Summary: "store: disk I/O error"},
	}
	tr := Transcript("s0001", evs, func(u string) string { return u })
	if strings.Contains(tr, "  -  ()") {
		t.Fatalf("a store failure reads as an empty decision:\n%s", tr)
	}
}
```

Question: A new session event is a promised surface (ADR 0012). Do internal errors get their own
event type, or ride `turn-ended`'s fields? Recommendation: yes, a type `EventInternalError`, which
the transcript renders as one line carrying the error text and Discord posts nothing for; the three
call sites switch to it and `03-architecture.md` names it. Riding `turn-ended` would lose errors
that happen mid-turn and tie a reading (the meter) to the turn's end.
Answer:
