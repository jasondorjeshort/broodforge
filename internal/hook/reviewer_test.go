// SPDX-License-Identifier: GPL-3.0-or-later

package hook

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jasondorjeshort/broodforge/internal/policy"
)

func TestReviewerCellBadType(t *testing.T) {
	for _, in := range []string{`{"subagent_type":7}`, `{"subagent_type":["reviewer"]}`, `{"subagent_type":{"name":"reviewer"}}`} {
		_, err := Decide(Input{ToolName: "Agent", ToolInput: json.RawMessage(in), ToolUseID: "toolu_01",
			TranscriptPath: "/tmp/cfg/projects/x/s.jsonl"}, Settings{Mode: policy.ModePolicy, Reach: policy.ReachCode, Tree: "/tmp/t", Canonical: "/tmp/c"})
		if err == nil {
			t.Errorf("%s: no error, want tool_input.subagent_type is not a string", in)
		}
	}
}

// The `reviewer` cell (ADR 0028, P33): an Agent or Task call whose subagent_type is reviewer
// defers in every mode, the chain still judging it; the bot's resume answers allow.

func reviewerInput(agent, model string) string {
	in := map[string]string{"description": "d", "prompt": "p", "subagent_type": agent}
	if model != "" {
		in["model"] = model
	}
	b, _ := json.Marshal(in)
	return string(b)
}

// TestReviewerCellDefers: in both modes and both reaches the first call defers from the
// `reviewer` cell, logged with that cell, and writes no pin; the resume with
// BROODFORGE_ANSWER `<id>:allow` returns allow and writes the call's pin as any allowed launch
// does; a builder stays allow.
func TestReviewerCellDefers(t *testing.T) {
	for _, tool := range []string{"Agent", "Task"} {
		for _, mode := range []string{"policy", "prompt"} {
			for _, reach := range []string{"read-only", "code"} {
				w := newPinWorld(t)
				extra := map[string]string{EnvMode: mode, EnvReach: reach}
				d, lines := w.runTool(t, tool, reviewerInput("reviewer", "haiku"), extra)
				if d != "defer" || !slices.Equal(lines, []string{sonnetLine}) {
					t.Errorf("%s %s %s: %s %q, want defer and no pin", tool, mode, reach, d, lines)
				}
				log := readLog(t, w.log)
				if last := log[len(log)-1]; field(last, "cell") != "reviewer" || field(last, "decision") != "defer" {
					t.Errorf("%s %s %s: log %v", tool, mode, reach, last)
				}
				extra[EnvAnswer] = "toolu_01:allow"
				d, lines = w.runTool(t, tool, reviewerInput("reviewer", "haiku"), extra)
				if d != "allow" || !slices.Equal(lines, []string{sonnetLine, haikuLine}) {
					t.Errorf("%s %s %s resumed: %s %q, want allow and the Haiku pin", tool, mode, reach, d, lines)
				}
				delete(extra, EnvAnswer)
				if d, _ := w.runTool(t, tool, reviewerInput("builder", ""), extra); d != "allow" {
					t.Errorf("%s %s %s builder: %s, want allow", tool, mode, reach, d)
				}
			}
		}
	}
}

// runTool is pinWorld.run for any subagent tool, without the copy.
func (w pinWorld) runTool(t *testing.T, tool, input string, extra map[string]string) (string, []string) {
	t.Helper()
	if tool == "Agent" {
		return w.run(t, input, false, extra)
	}
	vars := map[string]string{EnvSession: "s0001", EnvMode: "policy", EnvReach: "code", EnvTree: filepath.ToSlash(w.tree),
		EnvCanonical: filepath.ToSlash(w.tree), EnvHookLog: w.log}
	for k, v := range extra {
		vars[k] = v
	}
	stdin, err := json.Marshal(engineInput{ToolName: tool, ToolInput: json.RawMessage(input), ToolUseID: "toolu_01",
		Cwd: filepath.ToSlash(w.tree), TranscriptPath: "/tmp/cfg/projects/x/s.jsonl"})
	if err != nil {
		t.Fatal(err)
	}
	code, out, errs := run(string(stdin), vars)
	if code != 0 || errs != "" {
		t.Fatalf("exit %d stderr %q", code, errs)
	}
	var o Output
	if err := json.Unmarshal([]byte(out), &o); err != nil {
		t.Fatalf("stdout %q: %v", out, err)
	}
	return o.HookSpecificOutput.PermissionDecision, w.trailers(t)
}

// TestReviewerCellKeepsTheChain: the project's PreToolUse children still judge a reviewer call,
// so a child deny turns the cell's defer into a deny, logged with the reviewer cell.
func TestReviewerCellKeepsTheChain(t *testing.T) {
	environ := childEnviron(t)
	wf := writeCopy(t, `[{"event":"PreToolUse","matcher":"Agent","command":"echo no reviewers here >&2; exit 2","name":"no-review","timeout":null}]`)
	tree := absDir(t, t.TempDir())
	log := filepath.Join(t.TempDir(), "hook.log")
	stdin, _ := json.Marshal(engineInput{HookEventName: EventPreToolUse, ToolName: "Agent",
		ToolInput: json.RawMessage(reviewerInput("reviewer", "")), ToolUseID: "toolu_01", Cwd: tree,
		TranscriptPath: "/tmp/cfg/projects/x/s.jsonl"})
	var out, errb bytes.Buffer
	code := PreToolUse(bytes.NewReader(stdin), &out, &errb, env(chainVars(t, tree, wf, log)), environ, time.Now)
	var o Output
	if code != 0 || json.Unmarshal(out.Bytes(), &o) != nil || o.HookSpecificOutput.PermissionDecision != "deny" ||
		!strings.Contains(o.HookSpecificOutput.PermissionDecisionReason, "no-review") {
		t.Fatalf("exit %d stdout %q stderr %q", code, out.String(), errb.String())
	}
	lines := readLog(t, log)
	if last := lines[len(lines)-1]; field(last, "cell") != "reviewer" || field(last, "decision") != "deny" {
		t.Errorf("log %v", lines)
	}
}
