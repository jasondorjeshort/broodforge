// SPDX-License-Identifier: GPL-3.0-or-later

package discord

import (
	"strings"
	"testing"

	"github.com/jasondorjeshort/broodforge/internal/session"
)

// TestTranscriptDecisionNamesItsCallAndDepth: a decision prints as `- <decision> (<source>) for
// <call>` at the call's depth, and a subagent's text is indented at its depth.
func TestTranscriptDecisionNamesItsCallAndDepth(t *testing.T) {
	events := []session.Event{
		{Type: session.EventPermissionDecision, Tool: "Bash", Target: "ls", Decision: "allow", DecidedBy: "policy"},
		{Type: session.EventText, Text: "No findings.", Parent: "toolu_a", Depth: 1},
		{Type: session.EventPermissionDecision, Tool: "Read", Target: "go.mod", Decision: "allow", DecidedBy: "policy",
			Parent: "toolu_a", Depth: 1},
	}
	got := Transcript("s0001", events, func(u string) string { return u })
	for _, want := range []string{
		"\n  - allow (policy) for [Bash] ls\n",
		"\n  No findings.\n",
		"\n    - allow (policy) for [Read] go.mod\n",
	} {
		if !strings.Contains("\n"+got, want) {
			t.Errorf("transcript lacks %q:\n%s", want, got)
		}
	}
}

// TestTranscriptStoreFailureIsNoDecision: a store failure mid-turn is an internal-error event,
// printed as `- internal: <error>` at its call's depth, never as an empty decision.
func TestTranscriptStoreFailureIsNoDecision(t *testing.T) {
	evs := []session.Event{
		{Type: session.EventTurnStarted, Turn: 1},
		{Type: session.EventInternalError, Turn: 1, Summary: "store: disk I/O error"},
		{Type: session.EventInternalError, Turn: 1, Tool: "Read", Target: "go.mod", Parent: "toolu_a", Depth: 1,
			Summary: "store: database is locked"},
	}
	tr := Transcript("s0001", evs, func(u string) string { return u })
	if strings.Contains(tr, "  -  ()") {
		t.Fatalf("a store failure reads as an empty decision:\n%s", tr)
	}
	for _, want := range []string{
		"\n  - internal: store: disk I/O error\n",
		"\n    - internal: store: database is locked\n",
	} {
		if !strings.Contains("\n"+tr, want) {
			t.Errorf("transcript lacks %q:\n%s", want, tr)
		}
	}
}

func TestTranscriptSubagentTextHasNoTrailingSpace(t *testing.T) {
	got := Transcript("s0001", []session.Event{
		{Type: session.EventText, Text: "One.\n\nTwo.\n", Parent: "toolu_a", Depth: 1},
	}, func(u string) string { return u })
	if !strings.Contains(got, "  One.\n\n  Two.\n") || strings.Contains(got, "\n\n\n") {
		t.Errorf("subagent text: %q", got)
	}
	for i, l := range strings.Split(got, "\n") {
		if strings.TrimRight(l, " ") != l {
			t.Errorf("line %d has trailing whitespace: %q", i, l)
		}
	}
}
