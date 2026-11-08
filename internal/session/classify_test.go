// SPDX-License-Identifier: GPL-3.0-or-later

package session

import (
	"bytes"
	"encoding/json"
	"os"
	osexec "os/exec"
	"path/filepath"
	"strings"
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
			case engine.KindInit, engine.KindResult, engine.KindExit, engine.KindRateLimit, engine.KindPrompt, engine.KindUnknown:
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

// TestClassifyStdioSourcesWithoutRecord is ADR 0029 Amendments 2026-11-07 (3) with no request
// record: a user_* source is hook bypassed, cause engine answered; config with reason_type other
// is hook bypassed, cause engine aborted; plain config stays today's (P40d).
func TestClassifyStdioSourcesWithoutRecord(t *testing.T) {
	ran := Classify(engine.ToolResultEvent{Text: "hi", DecidedBy: "user_temporary", PermissionDecision: engine.PermissionAccept}, "Bash")
	if ran.Kind != ClassBypassed || ran.Decision != DecisionAllow || !ran.Ran || ran.Cause != CauseEngineAnswered || ran.Reason != CauseEngineAnswered {
		t.Errorf("user_temporary: %+v", ran)
	}
	rej := Classify(engine.ToolResultEvent{IsError: true, Text: "no", DecidedBy: "user_reject", PermissionDecision: engine.PermissionReject}, "Bash")
	if rej.Kind != ClassBypassed || rej.Decision != DecisionDeny || rej.Ran || rej.Cause != CauseEngineAnswered {
		t.Errorf("user_reject: %+v", rej)
	}
	abort := Classify(engine.ToolResultEvent{IsError: true, Text: "Tool permission request failed: AbortError", DecidedBy: "config",
		ReasonType: "other"}, "Bash")
	if abort.Kind != ClassBypassed || abort.Decision != DecisionDeny || abort.DecidedBy != DecidedByConfig || abort.Cause != CauseEngineAborted ||
		abort.Reason != CauseEngineAborted {
		t.Errorf("config/other: %+v", abort)
	}
	plain := Classify(engine.ToolResultEvent{IsError: true, Text: "timed out", DecidedBy: "config"}, "Bash")
	if plain.Kind != ClassBypassed || plain.Cause != "" || plain.Reason != "timed out" {
		t.Errorf("config: %+v", plain)
	}
}

// TestClassifyAutoReadOnlyTimeoutIsBypassed (P39): in an auto session a Write result whose
// decider is config is the engine's own decision only on a call whose cell for the session's
// reach is ask. In read-only reach the Write's cell is deny, so a config decider there is the
// hook bypassed (a timeout on an older engine), never engine; its code-reach twin under the
// tree is ask, so the same result is engine.
func TestClassifyAutoReadOnlyTimeoutIsBypassed(t *testing.T) {
	tree := filepath.ToSlash(t.TempDir())
	target, _ := json.Marshal(tree + "/a.go")
	in := json.RawMessage(`{"file_path":` + string(target) + `,"content":"x"}`)
	refused := engine.ToolResultEvent{ToolUseID: "t", IsError: true, DecidedBy: "config", PermissionDecision: "reject",
		Text: "Claude requested permissions to write to " + tree + "/a.go, but you haven't granted it yet."}
	ro := ClassifyIn(refused, "Write", AskCell("Write", in, "read-only", tree))
	if ro.Kind != ClassBypassed || ro.DecidedBy != DecidedByConfig || ro.Decision != DecisionDeny {
		t.Errorf("read-only reach: %+v, want hook bypassed by config", ro)
	}
	code := ClassifyIn(refused, "Write", AskCell("Write", in, "code", tree))
	if code.Kind != ClassEngine || code.DecidedBy != DecidedByEngine || code.Decision != DecisionDeny || code.Reason != "config" {
		t.Errorf("code reach under the tree: %+v, want engine deny with the raw source", code)
	}
	if line := ResultLine("Write", "a.go", "code", true, refused.Text, code); !strings.HasPrefix(line,
		"[Write] a.go -> refused by the engine's check: Claude requested permissions to write to ") {
		t.Errorf("engine refusal line %q", line)
	}
	ran := ClassifyIn(engine.ToolResultEvent{ToolUseID: "t", Text: "ok", DecidedBy: "config"}, "Write", AskCell("Write", in, "code", tree))
	if ran.Kind != ClassEngine || ran.Decision != DecisionAllow {
		t.Errorf("code reach, ran: %+v", ran)
	}
	// The hook's own answers stay the hook's under auto: a hook-sourced result, a policy deny
	// and a hook failure, the onFailure block's text included.
	if c := ClassifyIn(engine.ToolResultEvent{ToolUseID: "t", Text: "ok", DecidedBy: "hook"}, "Write", true); c.Kind != ClassAllowed || c.DecidedBy != DecidedByPolicy {
		t.Errorf("hook-sourced allow under auto: %+v", c)
	}
	block := engine.ToolResultEvent{ToolUseID: "t", IsError: true, DecidedBy: "config",
		Text: `PreToolUse:Write hook error: [C:/bf/bin/broodforge.exe hook pre-tool-use]: failed; blocking because onFailure is "block"`}
	for _, ask := range []bool{false, true} {
		if c := ClassifyIn(block, "Write", ask); c.Kind != ClassHookFailed || c.DecidedBy != DecidedByHook {
			t.Errorf("onFailure block (ask %v): %+v, want a hook failure", ask, c)
		}
	}
}

// TestClassifyAutoRanAndFailedIsAllow: a call the engine's check let run and that then failed
// is an engine allow, read from permission_decision.decision, not from the error flag.
func TestClassifyAutoRanAndFailedIsAllow(t *testing.T) {
	const tree = "/r/sessions/s0001/tree"
	ask := AskCell("Bash", json.RawMessage(`{"command":"go test ./..."}`), "code", tree)
	failed := engine.ToolResultEvent{ToolUseID: "t", IsError: true, DecidedBy: "config", PermissionDecision: "accept",
		Text: "Exit code 1\n--- FAIL: TestX"}
	c := ClassifyIn(failed, "Bash", ask)
	if c.Kind != ClassEngine || c.Decision != DecisionAllow {
		t.Errorf("ran and failed: %+v, want an engine allow", c)
	}
	if line := ResultLine("Bash", "go test ./...", "code", true, failed.Text, c); strings.Contains(line, "refused by the engine's check") {
		t.Errorf("ran and failed reads as refused: %q", line)
	}
	refused := failed
	refused.PermissionDecision = "reject"
	if c := ClassifyIn(refused, "Bash", ask); c.Kind != ClassEngine || c.Decision != DecisionDeny {
		t.Errorf("rejected: %+v, want an engine deny", c)
	}
}

// askLink makes a directory link at at pointing to target, as internal/hook's tests do: a
// symlink, else a Windows junction; false when neither can be made.
func askLink(target, at string) bool {
	if err := os.Symlink(target, at); err == nil {
		return true
	}
	return osexec.Command("cmd", "/c", "mklink", "/J", filepath.FromSlash(at), filepath.FromSlash(target)).Run() == nil
}

// TestAskCell: the cell session computes for a call's decider, by tool, reach and target, with
// the target resolved as the hook resolves it: `~` and `/<letter>/` expanded, links followed.
func TestAskCell(t *testing.T) {
	base := filepath.ToSlash(t.TempDir())
	tree, outside := base+"/tree", base+"/outside"
	for _, d := range []string{tree, outside} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	q := func(p string) string { b, _ := json.Marshal(p); return string(b) }
	rows := []struct {
		tool, input, reach string
		want               bool
	}{
		{"Write", `{"file_path":` + q(tree+"/a.go") + `}`, "code", true},
		{"Write", `{"file_path":"a.go"}`, "code", true},
		{"Write", `{"file_path":"../outside.go"}`, "code", false},
		{"Write", `{"file_path":` + q(outside+"/a.go") + `}`, "code", false},
		{"Write", `{"file_path":` + q(tree+"/.credentials.json") + `}`, "code", false},
		{"Write", `{"file_path":` + q(tree+"/a.go") + `}`, "read-only", false},
		{"Write", `{"file_path":"~/a.go"}`, "code", false},
		{"Write", `{"file_path":"~someone/a.go"}`, "code", false},
		{"Write", `{}`, "code", false},
		{"Bash", `{"command":"go test ./..."}`, "code", true},
		{"Bash", `{"command":"go test ./..."}`, "read-only", false},
		{"Read", `{"file_path":` + q(tree+"/a.go") + `}`, "code", false},
		{"WebFetch", `{"url":"https://example.com"}`, "code", false},
		{"Agent", `{"subagent_type":"reviewer"}`, "code", false},
	}
	// On a host with drive letters `/c/...` is `C:\...`, as the engine and the hook expand it.
	if vol := filepath.VolumeName(tree); len(vol) == 2 && vol[1] == ':' {
		posix := "/" + strings.ToLower(vol[:1]) + tree[2:]
		rows = append(rows, struct {
			tool, input, reach string
			want               bool
		}{"Write", `{"file_path":` + q(posix+"/a.go") + `}`, "code", true})
	}
	// A link inside the tree that leads out of it: the hook denies, so it is never ask.
	if askLink(outside, tree+"/link") {
		rows = append(rows, struct {
			tool, input, reach string
			want               bool
		}{"Write", `{"file_path":` + q(tree+"/link/a.go") + `}`, "code", false},
			struct {
				tool, input, reach string
				want               bool
			}{"Write", `{"file_path":"link/a.go"}`, "code", false})
	} else {
		t.Log("no symlink or junction on this host: the link rows are skipped")
	}
	for _, c := range rows {
		if got := AskCell(c.tool, json.RawMessage(c.input), c.reach, tree); got != c.want {
			t.Errorf("AskCell(%s %s, %s) = %v, want %v", c.tool, c.input, c.reach, got, c.want)
		}
	}
}
