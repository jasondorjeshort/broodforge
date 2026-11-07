// SPDX-License-Identifier: GPL-3.0-or-later

package policy

import "testing"

func TestSubagentWebIgnoresSessionAllow(t *testing.T) {
	req := Request{Tool: "WebFetch", ToolUseID: "toolu_01", Mode: ModePrompt, Reach: ReachCode, SessionAllow: []string{"WebFetch"}, Subagent: true}
	if got := Decide(req); got.Decision != Ask {
		t.Errorf("subagent WebFetch session-allowed: %v, want ask (a session allow does not cover web)", got.Decision)
	}
}

// TestSubagentAskAndWeb:a subagent's ask- or web-cell call in prompt mode is ask with the
// owner's reason, a session-allowed ask-cell tool allow (web stays ask); in policy and auto mode it decides as the
// driver's own call; the driver's own call in prompt mode still defers.
func TestSubagentAskAndWeb(t *testing.T) {
	edit := []Path{{Target: "/t/a.go", UnderTree: true}}
	for _, c := range []struct {
		tool  string
		paths []Path
		reach Reach
	}{
		{"Bash", nil, ReachCode},
		{"Write", edit, ReachCode},
		{"Edit", edit, ReachCode},
		{"WebFetch", nil, ReachCode},
		{"WebSearch", nil, ReachReadOnly},
	} {
		web := ClassOf(c.tool) == ClassWeb
		req := Request{Tool: c.tool, ToolUseID: "toolu_01", Mode: ModePrompt, Reach: c.reach, Paths: c.paths, Subagent: true}
		if got := Decide(req); got.Decision != Ask || got.Reason != reasonAsk {
			t.Errorf("%s prompt subagent: %v %q, want ask", c.tool, got.Decision, got.Reason)
		}
		req.SessionAllow = []string{"Read", c.tool}
		if got := Decide(req); web && (got.Decision != Ask || got.Reason != reasonAsk) {
			t.Errorf("%s prompt subagent session-allowed: %v %q, want ask (a session allow does not cover web)", c.tool, got.Decision, got.Reason)
		} else if !web && (got.Decision != Allow || got.Reason != reasonSessionAllow) {
			t.Errorf("%s prompt subagent session-allowed: %v %q, want allow", c.tool, got.Decision, got.Reason)
		}
		req.SessionAllow = nil
		req.Answer = &Answer{ToolUseID: "toolu_01", Decision: Deny}
		if got := Decide(req); got.Decision != Deny || got.Reason != AnswerDenyText {
			t.Errorf("%s prompt subagent answered: %v %q", c.tool, got.Decision, got.Reason)
		}
		req.Answer = nil
		for _, mode := range []Mode{ModePolicy, ModeAuto} {
			req.Mode, req.Subagent = mode, true
			sub := Decide(req)
			req.Subagent = false
			own := Decide(req)
			if sub != own || sub.Decision == Ask {
				t.Errorf("%s %s: subagent %v, driver %v, want the same", c.tool, mode, sub, own)
			}
		}
		req.Mode, req.Subagent = ModePrompt, false
		if got := Decide(req); got.Decision != Defer {
			t.Errorf("%s prompt, the driver's own call: %v, want defer", c.tool, got.Decision)
		}
		if req.SessionAllow = []string{c.tool}; web {
			if got := Decide(req); got.Decision != Defer {
				t.Errorf("%s prompt, the driver's own call session-allowed: %v, want defer as before", c.tool, got.Decision)
			}
		}
	}
	if Ask.String() != "ask" {
		t.Errorf("Ask is %q", Ask)
	}
}

// TestSubagentDeniesFirst: a reach or credential deny stands for a subagent's call, and a
// subagent's reviewer launch is deny with the owner's text in every mode and reach, an answer
// for it included; the allow cell and other subagent types are untouched.
func TestSubagentDeniesFirst(t *testing.T) {
	for _, mode := range []Mode{ModePolicy, ModePrompt, ModeAuto} {
		if got := Decide(Request{Tool: "Bash", Mode: mode, Reach: ReachReadOnly, Subagent: true}); got.Decision != Deny || got.Cell != CellDeny {
			t.Errorf("Bash read-only %s subagent: %v %v", mode, got.Decision, got.Cell)
		}
		if got := Decide(Request{Tool: "Read", Mode: mode, Reach: ReachCode, Paths: []Path{cred}, Subagent: true}); got.Decision != Deny ||
			got.Reason != CredentialDenyText(cred.Target, false) {
			t.Errorf("credential %s subagent: %v %q", mode, got.Decision, got.Reason)
		}
		for _, reach := range []Reach{ReachReadOnly, ReachCode} {
			for _, tool := range []string{"Agent", "Task"} {
				req := Request{Tool: tool, ToolUseID: "toolu_rev", Mode: mode, Reach: reach, SubagentType: ReviewerAgent, Subagent: true}
				want := "Denied by Broodforge: in this session the reviewer is launched by the driver process; report and let it launch one."
				if got := Decide(req); got.Decision != Deny || got.Reason != want || got.Cell != CellReviewer {
					t.Errorf("%s reviewer %s %s subagent: %v %q %v", tool, mode, reach, got.Decision, got.Reason, got.Cell)
				}
				req.Answer = &Answer{ToolUseID: "toolu_rev", Decision: Allow}
				if got := Decide(req); got.Decision != Deny {
					t.Errorf("%s reviewer %s %s subagent answered allow: %v", tool, mode, reach, got.Decision)
				}
				req = Request{Tool: tool, ToolUseID: "toolu_rev", Mode: mode, Reach: reach, SubagentType: "builder", Subagent: true}
				if got := Decide(req); got.Decision != Allow || got.Cell != CellAllow {
					t.Errorf("%s builder %s %s subagent: %v %v", tool, mode, reach, got.Decision, got.Cell)
				}
			}
		}
	}
}
