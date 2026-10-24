// SPDX-License-Identifier: GPL-3.0-or-later

package policy

import (
	"strings"
	"testing"
)

var (
	inTree  = Path{Target: "/t/a", UnderTree: true}
	inCanon = Path{Target: "/c/a", UnderCanonical: true}
	outside = Path{Target: "/etc/x"}
	cred    = Path{Target: "/t/.credentials.json", UnderTree: true, Credential: true}
)

// TestTable checks every cell of 03-hook-contract.md's table in both reaches and both modes.
func TestTable(t *testing.T) {
	type row struct {
		tool  string
		reach Reach
		paths []Path
		mode  Mode
		want  Decision
	}
	var rows []row
	add := func(tool string, r Reach, paths []Path, policy, prompt Decision) {
		rows = append(rows, row{tool, r, paths, ModePolicy, policy}, row{tool, r, paths, ModePrompt, prompt})
	}
	for _, tool := range []string{"Read", "Glob", "Grep", "LS", "NotebookRead"} {
		add(tool, ReachReadOnly, []Path{inTree}, Allow, Allow)
		add(tool, ReachReadOnly, []Path{inCanon}, Allow, Allow)
		add(tool, ReachReadOnly, []Path{outside}, Deny, Deny)
		add(tool, ReachCode, []Path{inTree}, Allow, Allow)
		add(tool, ReachCode, []Path{outside}, Allow, Allow)
		add(tool, ReachCode, []Path{cred}, Deny, Deny)
		add(tool, ReachReadOnly, []Path{cred}, Deny, Deny)
	}
	for _, tool := range []string{"TodoWrite", "Agent", "Task"} {
		add(tool, ReachReadOnly, nil, Allow, Allow)
		add(tool, ReachCode, nil, Allow, Allow)
	}
	for _, tool := range []string{"WebFetch", "WebSearch"} {
		add(tool, ReachReadOnly, nil, Deny, Defer)
		add(tool, ReachCode, nil, Deny, Defer)
	}
	for _, tool := range []string{"Edit", "MultiEdit", "Write", "NotebookEdit"} {
		add(tool, ReachReadOnly, []Path{inTree}, Deny, Deny)
		add(tool, ReachCode, []Path{inTree}, Allow, Defer)
		add(tool, ReachCode, []Path{inCanon}, Deny, Deny)
		add(tool, ReachCode, []Path{outside}, Deny, Deny)
		add(tool, ReachCode, []Path{cred}, Deny, Deny)
	}
	add("Bash", ReachReadOnly, nil, Deny, Deny)
	add("Bash", ReachCode, nil, Allow, Defer)
	add("mcp__x__y", ReachCode, nil, Deny, Deny)
	add("AskUserQuestion", ReachReadOnly, nil, Deny, Deny)
	for _, r := range rows {
		got := Decide(Request{Tool: r.tool, ToolUseID: "toolu_01", Mode: r.mode, Reach: r.reach, Paths: r.paths})
		if got.Decision != r.want {
			t.Errorf("%s %s %s %v: %s, want %s", r.tool, r.reach, r.mode, r.paths, got.Decision, r.want)
		}
		if got.Decision == Deny && !strings.HasPrefix(got.Reason, DenyPrefix) {
			t.Errorf("%s: deny reason %q lacks the prefix", r.tool, got.Reason)
		}
	}
}

func TestDecisionOrder(t *testing.T) {
	allow := &Answer{ToolUseID: "toolu_01", Decision: Allow}
	deny := &Answer{ToolUseID: "toolu_01", Decision: Deny}
	cases := []struct {
		name string
		req  Request
		want Decision
		why  string
	}{
		{"answer cannot lift a reach deny", Request{Tool: "Write", ToolUseID: "toolu_01", Mode: ModePrompt, Reach: ReachReadOnly, Paths: []Path{inTree}, Answer: allow}, Deny, DenyPrefix},
		{"answer allow", Request{Tool: "Write", ToolUseID: "toolu_01", Mode: ModePrompt, Reach: ReachCode, Paths: []Path{inTree}, Answer: allow}, Allow, reasonAnswerAllow},
		{"answer deny, default text", Request{Tool: "Bash", ToolUseID: "toolu_01", Mode: ModePrompt, Reach: ReachCode, Answer: deny}, Deny, AnswerDenyText},
		{"answer deny, session reason", Request{Tool: "Bash", ToolUseID: "toolu_01", Mode: ModePrompt, Reach: ReachCode,
			Answer: &Answer{ToolUseID: "toolu_01", Decision: Deny, Reason: "the prompt timed out after 5 minutes"}}, Deny, "the prompt timed out after 5 minutes"},
		{"answer for another id", Request{Tool: "Bash", ToolUseID: "toolu_02", Mode: ModePrompt, Reach: ReachCode, Answer: allow}, Defer, reasonDefer},
		{"answer allow on a web tool in prompt mode", Request{Tool: "WebFetch", ToolUseID: "toolu_01", Mode: ModePrompt, Reach: ReachCode, Answer: allow}, Allow, reasonAnswerAllow},
		{"session allow", Request{Tool: "Bash", ToolUseID: "toolu_01", Mode: ModePrompt, Reach: ReachCode, SessionAllow: []string{"Write", "Bash"}}, Allow, reasonSessionAllow},
		{"session allow does not cover web", Request{Tool: "WebFetch", ToolUseID: "toolu_01", Mode: ModePrompt, Reach: ReachCode, SessionAllow: []string{"WebFetch"}}, Defer, reasonDefer},
		{"session allow does not lift deny", Request{Tool: "Bash", ToolUseID: "toolu_01", Mode: ModePrompt, Reach: ReachReadOnly, SessionAllow: []string{"Bash"}}, Deny, DenyPrefix},
	}
	for _, c := range cases {
		got := Decide(c.req)
		if got.Decision != c.want || !strings.HasPrefix(got.Reason, c.why) {
			t.Errorf("%s: %s %q, want %s %q", c.name, got.Decision, got.Reason, c.want, c.why)
		}
	}
}

func TestDenyText(t *testing.T) {
	got := Decide(Request{Tool: "Write", Mode: ModePolicy, Reach: ReachCode, Paths: []Path{inTree, outside}}).Reason
	want := "Denied by Broodforge: Write on /etc/x is outside this channel's reach (code); nobody in this session can allow it; the channel's configuration sets reach"
	if got != want {
		t.Errorf("deny text\n got %q\nwant %q", got, want)
	}
	if got := DenyText("Bash", "", ReachReadOnly); !strings.HasPrefix(got, "Denied by Broodforge: Bash is outside this channel's reach (read-only);") {
		t.Errorf("no-target text %q", got)
	}
}

func TestPermitted(t *testing.T) {
	groups := map[string][]string{"trusted": {"10"}, "members": {"11", "12"}}
	cases := []struct {
		held   []string
		list   []string
		driver bool
		want   bool
	}{
		{[]string{"10"}, []string{"trusted"}, false, true},
		{[]string{"11"}, []string{"trusted"}, false, false},
		{[]string{"12"}, []string{"trusted", "members"}, false, true},
		{[]string{"99"}, []string{"99"}, false, true},
		{nil, []string{"driver"}, true, true},
		{[]string{"10"}, []string{"driver"}, false, false},
		{nil, nil, true, false},
	}
	for _, c := range cases {
		if got := Permitted(c.held, c.list, groups, c.driver); got != c.want {
			t.Errorf("Permitted(%v, %v, driver %v) = %v", c.held, c.list, c.driver, got)
		}
	}
}

func TestModelDisplayName(t *testing.T) {
	for id, want := range map[string]string{
		"claude-haiku-5-5":          "Claude Haiku 5.5",
		"claude-opus-5-5-20261001":  "Claude Opus 5.5",
		"claude-opus-5-55":          "claude-opus-5-55",
		"some-other-model":          "some-other-model",
		"claude-sonnet-5-5[1m]-x":   "claude-sonnet-5-5[1m]-x",
		"claude-sonnet-5-5-preview": "Claude Sonnet 5.5",
	} {
		if got := ModelDisplayName(id); got != want {
			t.Errorf("ModelDisplayName(%q) = %q, want %q", id, got, want)
		}
	}
}

// TestClassRowsMatchClassOf: the rows reach-report prints name the tools ClassOf classifies.
func TestClassRowsMatchClassOf(t *testing.T) {
	for _, row := range ClassRows() {
		for _, tool := range row.Tools {
			if got := ClassOf(tool); got != row.Class {
				t.Errorf("ClassOf(%s) = %d, row %d", tool, got, row.Class)
			}
		}
	}
	if ClassOf("mcp__x__y") != ClassOther {
		t.Error("an mcp tool is not ClassOther")
	}
}

func TestResolveModelAndEfforts(t *testing.T) {
	if got := strings.Join(ModelAliases(), ","); got != "opus,sonnet,haiku,fable" {
		t.Errorf("aliases %s", got)
	}
	for v, want := range map[string]string{
		"opus": "claude-opus-5-5", "Sonnet": "claude-sonnet-5-5", "haiku": "claude-haiku-5-5", "fable": "claude-fable-5-1",
		"claude-fable-5-1": "claude-fable-5-1", "claude-opus-5-5-20261001": "claude-opus-5-5-20261001",
	} {
		if got, ok := ResolveModel(v); !ok || got != want {
			t.Errorf("ResolveModel(%q) = %q, %v; want %q", v, got, ok, want)
		}
	}
	for _, v := range []string{"", "gpt", "claude-opus-5-55", "claude-opus", "claude-sonnet-5-5[1m]", "claude-opus-5-5 --x", "CLAUDE-OPUS-5-5",
		"claude-opus-5-5-x", "claude-opus-5-5-2026100", "claude-opus-5-5-202610011", "claude-opus-5-5-20261001-x"} {
		if got, ok := ResolveModel(v); ok {
			t.Errorf("ResolveModel(%q) = %q, want refused", v, got)
		}
	}
	if ModelDisplayName("claude-fable-5-1") != "Claude Fable 5.1" {
		t.Error("fable has no display name")
	}
	// The engine's order (`claude --help`), xhigh between high and max.
	if got := strings.Join(Efforts, ","); got != "low,medium,high,xhigh,max" {
		t.Errorf("efforts %s", got)
	}
	if !ParseEffort("max") || !ParseEffort("xhigh") || ParseEffort("turbo") || ParseEffort("XHIGH") || ParseEffort("") {
		t.Error("ParseEffort")
	}
}

func TestAgentAndTaskBothClassifyAsAgent(t *testing.T) {
	for _, tool := range []string{"Agent", "Task"} {
		if got := ClassOf(tool); got != ClassAgent {
			t.Errorf("ClassOf(%q) = %v, want ClassAgent", tool, got)
		}
	}
}
