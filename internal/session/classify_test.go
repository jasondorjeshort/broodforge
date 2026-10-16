// SPDX-License-Identifier: GPL-3.0-or-later

package session

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/jasondorjeshort/broodforge/internal/engine"
)

const fixtures = "../../fixtures/engine"

// fixtureResults replays a recorded turn through the parser and returns its tool results with
// the tool name of the call each answers.
func fixtureResults(t *testing.T, name string, turn int) []struct {
	tool string
	tr   engine.ToolResultEvent
} {
	t.Helper()
	data, err := os.ReadFile(engine.TurnFile(filepath.Join(fixtures, name), turn))
	if err != nil {
		t.Fatal(err)
	}
	tools := map[string]string{}
	var out []struct {
		tool string
		tr   engine.ToolResultEvent
	}
	for _, line := range bytes.Split(data, []byte("\n")) {
		for _, ev := range engine.ParseLine(line) {
			switch ev.Kind {
			case engine.KindAssistant:
				for _, b := range ev.Assistant.Blocks {
					if b.Kind == engine.BlockToolUse {
						tools[b.ID] = b.Name
					}
				}
			case engine.KindToolResult:
				out = append(out, struct {
					tool string
					tr   engine.ToolResultEvent
				}{tools[ev.ToolResult.ToolUseID], *ev.ToolResult})
			case engine.KindInit, engine.KindResult, engine.KindExit, engine.KindUnknown:
			}
		}
	}
	return out
}

func TestStripWrapper(t *testing.T) {
	for _, c := range []struct{ text, tool, want string }{
		{"PreToolUse:Read hook error: [node /x/hook.js]: hook: spike exit 2\n", "Read", "hook: spike exit 2"},
		{"PreToolUse:Write hook error: Denied by Broodforge: spike deny", "Write", "Denied by Broodforge: spike deny"},
		{"PreToolUse:Bash hook error: [C:/bf/bin/broodforge.exe hook pre-tool-use]:  hook: no transcript_path \n", "Bash", "hook: no transcript_path"},
		// Another tool's wrapper, or none, is left alone but trimmed.
		{"PreToolUse:Read hook error: hook: x", "Bash", "PreToolUse:Read hook error: hook: x"},
		{"  File does not exist.  ", "Read", "File does not exist."},
	} {
		if got := StripWrapper(c.text, c.tool); got != c.want {
			t.Errorf("StripWrapper(%q, %s) = %q, want %q", c.text, c.tool, got, c.want)
		}
	}
}

// TestClassifyFixtures classifies every recorded tool result the hook fixtures hold, as the
// contract reads them (03-hook-contract.md, What the bot sees).
func TestClassifyFixtures(t *testing.T) {
	type want struct {
		kind      ClassKind
		decision  string
		decidedBy string
		ran       bool
	}
	for _, c := range []struct {
		fixture string
		turn    int
		want    []want
	}{
		{"basic", 1, []want{{ClassAllowed, DecisionAllow, DecidedByPolicy, false}}},
		{"hook-exit2", 1, []want{
			{ClassHookFailed, DecisionDeny, DecidedByHook, false},
			{ClassHookFailed, DecisionDeny, DecidedByHook, false}}},
		{"hook-exit1", 1, []want{
			{ClassBypassed, DecisionAllow, DecidedByConfig, true},
			{ClassBypassed, DecisionDeny, DecidedByConfig, false},
			{ClassBypassed, DecisionAllow, DecidedByConfig, true}}},
		{"hook-sleep", 1, []want{
			{ClassBypassed, DecisionAllow, DecidedByConfig, true},
			{ClassBypassed, DecisionDeny, DecidedByConfig, false},
			{ClassBypassed, DecisionAllow, DecidedByConfig, true}}},
		{"defer", 2, []want{{ClassAllowed, DecisionAllow, DecidedByPolicy, false}}},
		{"defer", 4, []want{{ClassPolicyDenied, DecisionDeny, DecidedByPolicy, false}}},
		// The engine refuses the standalone sleep before any hook; the background run is allowed.
		{"interrupted", 1, []want{{ClassNoDecision, "", "", false}, {ClassAllowed, DecisionAllow, DecidedByPolicy, false}}},
	} {
		results := fixtureResults(t, c.fixture, c.turn)
		if len(results) != len(c.want) {
			t.Fatalf("%s turn %d: %d tool results, want %d", c.fixture, c.turn, len(results), len(c.want))
		}
		for i, r := range results {
			got := Classify(r.tr, r.tool)
			w := c.want[i]
			if got.Kind != w.kind || got.Decision != w.decision || got.DecidedBy != w.decidedBy || got.Ran != w.ran {
				t.Errorf("%s turn %d result %d (%s): got %+v, want %+v", c.fixture, c.turn, i, r.tool, got, w)
			}
		}
	}
}

// TestClassifyUnknownSource covers a result without tool_result_meta's source.
func TestClassifyUnknownSource(t *testing.T) {
	ok := Classify(engine.ToolResultEvent{ToolUseID: "t", Text: "fine"}, "Read")
	if ok.Kind != ClassAllowed || ok.DecidedBy != DecidedByUnknown {
		t.Errorf("success without source: %+v", ok)
	}
	bad := Classify(engine.ToolResultEvent{ToolUseID: "t", IsError: true, Text: "refused"}, "Write")
	if bad.Kind != ClassBypassed || bad.Decision != DecisionDeny || bad.DecidedBy != DecidedByUnknown {
		t.Errorf("error without source or prefix: %+v", bad)
	}
	// A contains-match is never used: the prefix must lead the stripped text.
	mid := Classify(engine.ToolResultEvent{IsError: true, Text: "x Denied by Broodforge: y", DecidedBy: "hook"}, "Read")
	if mid.Kind != ClassAllowed {
		t.Errorf("a deny prefix inside the text classified as %+v", mid)
	}
}
