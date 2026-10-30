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
