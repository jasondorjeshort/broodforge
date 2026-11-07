// SPDX-License-Identifier: GPL-3.0-or-later

package hook

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jasondorjeshort/broodforge/internal/policy"
)

// A subagent's call (ADR 0029 rule 1, P40a): the hook reads agent_id and agent_type; in prompt
// mode an ask-cell call is ask, and a reviewer launch is deny in every mode.

// spikeSubagentLine is a subagent's hook stdin as the 2026-11-07 spike recorded it on 2.1.319
// (c42-allow), its paths replaced by placeholders: every key and the field order kept.
const spikeSubagentLine = `{"session_id":"4745ab0a-3231-46cc-8c33-2512019f4c7a","transcript_path":"/tmp/cfg/projects/x/s.jsonl",` +
	`"cwd":"/tmp/x","prompt_id":"9697dbbd-9d29-45b2-8ee2-d79a5166d983","permission_mode":"default",` +
	`"agent_id":"a26c6f2b725f33c8e","agent_type":"probe","effort":{"level":"medium"},"hook_event_name":"PreToolUse",` +
	`"tool_name":"Bash","tool_input":{"command":"echo hi","description":"Print hi via echo"},"tool_use_id":"toolu_01GgR6UWwxWCU6FVfmQSDfqe"}`

// spikeStubAsk is the line the spike's stub hook printed for a subagent's call and the engine
// accepted (the finding, What was tried), newline included.
const spikeStubAsk = `{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"ask","permissionDecisionReason":` +
	`"Broodforge: asked a person in this session; if this reads as a refusal, no prompt reached them: say so and stop"}}` + "\n"

const reviewerDeny = "Denied by Broodforge: in this session the reviewer is launched by the driver process; report and let it launch one."

func TestParseInputAgent(t *testing.T) {
	in, err := ParseInput([]byte(spikeSubagentLine))
	if err != nil || in.AgentID != "a26c6f2b725f33c8e" || in.AgentType != "probe" {
		t.Fatalf("spike line: %+v %v", in, err)
	}
	in, err = ParseInput([]byte(writeInput))
	if err != nil || in.AgentID != "" || in.AgentType != "" {
		t.Fatalf("the driver's own call: %+v %v", in, err)
	}
	for _, c := range []struct{ field, value, want string }{
		{"agent_id", `7`, "hook: agent_id is not a string"},
		{"agent_id", `null`, "hook: agent_id is not a string"},
		{"agent_id", `{"id":"a1"}`, "hook: agent_id is not a string"},
		{"agent_id", `["a1"]`, "hook: agent_id is not a string"},
		{"agent_type", `true`, "hook: agent_type is not a string"},
	} {
		line := strings.TrimSuffix(writeInput, "}") + `,"` + c.field + `":` + c.value + "}"
		if _, err := ParseInput([]byte(line)); err == nil || err.Error() != c.want {
			t.Errorf("%s %s: %v, want %q", c.field, c.value, err, c.want)
		}
	}
}

// TestEncodeAskIsTheSpikeStubs: the ask output is byte for byte the line the spike's stub
// printed and the engine took as an ask.
func TestEncodeAskIsTheSpikeStubs(t *testing.T) {
	r := policy.Decide(policy.Request{Tool: "Bash", ToolUseID: "toolu_01", Mode: policy.ModePrompt, Reach: policy.ReachCode, Subagent: true})
	b, err := Encode(r)
	if err != nil || string(b) != spikeStubAsk {
		t.Fatalf("got %q %v\nwant %q", b, err, spikeStubAsk)
	}
}

// TestPreToolUseSubagent runs the spike's line through the hook as the engine does: ask in
// prompt mode, exit 0, the stub's bytes on stdout, the log line naming the agent; allow in
// policy mode; the driver's own call defers and its log line has no agent keys.
func TestPreToolUseSubagent(t *testing.T) {
	log := filepath.Join(t.TempDir(), "hook.log")
	code, out, errs := run(spikeSubagentLine, baseEnv(log))
	if code != 0 || out != spikeStubAsk || errs != "" {
		t.Fatalf("prompt: exit %d stdout %q stderr %q", code, out, errs)
	}
	vars := baseEnv(log)
	vars[EnvMode] = "policy"
	if code, out, _ := run(spikeSubagentLine, vars); code != 0 || !strings.Contains(out, `"permissionDecision":"allow"`) {
		t.Fatalf("policy: exit %d stdout %q", code, out)
	}
	if code, out, _ := run(writeInput, baseEnv(log)); code != 0 || !strings.Contains(out, `"permissionDecision":"defer"`) {
		t.Fatalf("the driver's own call: exit %d stdout %q", code, out)
	}
	lines := readLog(t, log)
	if len(lines) != 3 {
		t.Fatalf("log %v", lines)
	}
	for i, want := range []string{"ask", "allow"} {
		if l := lines[i]; field(l, "agentId") != "a26c6f2b725f33c8e" || field(l, "agentType") != "probe" ||
			field(l, "decision") != want || field(l, "cell") != "ask" {
			t.Errorf("line %d: %v", i, l)
		}
	}
	if l := lines[2]; l["agentId"] != nil || l["agentType"] != nil || field(l, "decision") != "defer" {
		t.Errorf("the driver's own line: %v", l)
	}
	bad := strings.Replace(spikeSubagentLine, `"agent_id":"a26c6f2b725f33c8e"`, `"agent_id":17`, 1)
	if code, out, errs := run(bad, baseEnv(log)); code != 2 || out != "" || errs != "hook: agent_id is not a string\n" {
		t.Errorf("a non-string agent_id: exit %d stdout %q stderr %q", code, out, errs)
	}
}

// TestPreToolUseSubagentReviewer: a subagent's reviewer launch is deny with the owner's text in
// every mode and reach, no pin written, while the driver's own still defers.
func TestPreToolUseSubagentReviewer(t *testing.T) {
	for _, mode := range []string{"policy", "prompt", "auto"} {
		for _, reach := range []string{"read-only", "code"} {
			w := newPinWorld(t)
			vars := map[string]string{EnvSession: "s0001", EnvMode: mode, EnvReach: reach, EnvTree: filepath.ToSlash(w.tree),
				EnvCanonical: filepath.ToSlash(w.tree), EnvHookLog: w.log}
			stdin, _ := json.Marshal(engineInput{ToolName: "Agent", ToolInput: json.RawMessage(reviewerInput("reviewer", "haiku")),
				ToolUseID: "toolu_01", Cwd: filepath.ToSlash(w.tree), TranscriptPath: "/tmp/cfg/projects/x/s.jsonl",
				AgentID: "a1", AgentType: "builder"})
			code, out, errs := run(string(stdin), vars)
			var o Output
			if code != 0 || errs != "" || json.Unmarshal([]byte(out), &o) != nil ||
				o.HookSpecificOutput.PermissionDecision != "deny" || o.HookSpecificOutput.PermissionDecisionReason != reviewerDeny {
				t.Errorf("%s %s: exit %d stdout %q stderr %q", mode, reach, code, out, errs)
			}
			if lines := w.trailers(t); len(lines) != 1 {
				t.Errorf("%s %s: a pin was written: %q", mode, reach, lines)
			}
			log := readLog(t, w.log)
			if last := log[len(log)-1]; field(last, "cell") != "reviewer" || field(last, "decision") != "deny" || field(last, "agentId") != "a1" {
				t.Errorf("%s %s: log %v", mode, reach, last)
			}
		}
	}
}

// TestSubagentAskKeepsTheChain: the project's PreToolUse children run on an ask as on an allow,
// with the engine's stdin, agent_id included; a child deny is a deny, logged with the agent.
func TestSubagentAskKeepsTheChain(t *testing.T) {
	environ := childEnviron(t)
	wf := writeCopy(t, `[{"event":"PreToolUse","matcher":"Bash","command":"if grep -q '\"agent_id\":\"a1\"'; then echo no subagent bash >&2; exit 2; fi","name":"no-sub","timeout":null}]`)
	tree := absDir(t, t.TempDir())
	for _, c := range []struct {
		agent, want string
	}{
		{"a1", "deny"}, // the child reads agent_id and refuses
		{"a2", "ask"},  // the child passes: the ask stands
		{"", "defer"},  // the driver's own call
	} {
		log := filepath.Join(t.TempDir(), "hook.log")
		vars := chainVars(t, tree, wf, log)
		vars[EnvMode] = "prompt"
		in, _ := json.Marshal(map[string]string{"command": "ls"})
		stdin, _ := json.Marshal(engineInput{HookEventName: EventPreToolUse, ToolName: "Bash", ToolInput: in, ToolUseID: "toolu_01",
			Cwd: tree, TranscriptPath: "/tmp/cfg/projects/x/s.jsonl", AgentID: c.agent})
		var out, errb bytes.Buffer
		code := PreToolUse(bytes.NewReader(stdin), &out, &errb, env(vars), environ, time.Now)
		var o Output
		if code != 0 || json.Unmarshal(out.Bytes(), &o) != nil || o.HookSpecificOutput.PermissionDecision != c.want {
			t.Errorf("agent %q: exit %d stdout %q stderr %q", c.agent, code, out.String(), errb.String())
			continue
		}
		if c.want == "deny" && !strings.Contains(o.HookSpecificOutput.PermissionDecisionReason, "no-sub: no subagent bash") {
			t.Errorf("agent %q: reason %q", c.agent, o.HookSpecificOutput.PermissionDecisionReason)
		}
		lines := readLog(t, log)
		if len(lines) != 2 || field(lines[0], "child") != "no-sub" || field(lines[0], "agentId") != c.agent {
			t.Errorf("agent %q: log %v", c.agent, lines)
		}
		if last := lines[len(lines)-1]; field(last, "decision") != c.want || field(last, "cell") != "ask" {
			t.Errorf("agent %q: decision line %v", c.agent, last)
		}
	}
}

// TestTrySubagent: hook try and the cases carry the agent: a subagent's call decides as the
// hook's does, and a case's agentId that is not a string is the error outcome.
func TestTrySubagent(t *testing.T) {
	try := Try{Tool: "Bash", Input: json.RawMessage(`{"command":"echo hi"}`), Mode: "prompt", Tree: "/tmp/t",
		Transcript: "/tmp/cfg/projects/x/s.jsonl", AgentID: "a1", AgentType: "probe"}
	if d, reason, _, err := try.Run(); err != nil || d != "ask" ||
		reason != "Broodforge: asked a person in this session; if this reads as a refusal, no prompt reached them: say so and stop" {
		t.Errorf("try: %s %q %v", d, reason, err)
	}
	cases := `{"tool":"Bash","input":{"command":"ls"},"tree":"/tmp/t","transcript":"/tmp/cfg/projects/x/s.jsonl","agentId":"a1","expect":"ask"}
{"tool":"Bash","input":{"command":"ls"},"tree":"/tmp/t","transcript":"/tmp/cfg/projects/x/s.jsonl","agentId":7,"expect":"error"}
{"tool":"Bash","input":{"command":"ls"},"tree":"/tmp/t","transcript":"/tmp/cfg/projects/x/s.jsonl","agentId":"a1","agentType":["x"],"expect":"error"}
{"tool":"Bash","input":{"command":"ls"},"tree":"/tmp/t","transcript":"/tmp/cfg/projects/x/s.jsonl","agentId":"","expect":"defer"}
`
	res, err := Cases(strings.NewReader(cases), t.TempDir())
	if err != nil || len(res) != 4 {
		t.Fatalf("%v %v", res, err)
	}
	for _, r := range res {
		if r.Result != ResultOK {
			t.Errorf("line %d: got %s want %s", r.Line, r.Got, r.Want)
		}
	}
}
