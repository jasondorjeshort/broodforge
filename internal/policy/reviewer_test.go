// SPDX-License-Identifier: GPL-3.0-or-later

package policy

import "testing"

// TestReviewerCell: an Agent, or Task on an older pin, whose subagent_type is reviewer is defer
// in every mode and both reaches, from the `reviewer` cell; the bot's allow on the resume
// returns allow and its deny deny; a credential path still denies first; any other subagent
// type, or another tool naming reviewer, keeps its row's cell.
func TestReviewerCell(t *testing.T) {
	for _, tool := range []string{"Agent", "Task"} {
		for _, mode := range []Mode{ModePolicy, ModePrompt} {
			for _, reach := range []Reach{ReachReadOnly, ReachCode} {
				req := Request{Tool: tool, ToolUseID: "toolu_rev", Mode: mode, Reach: reach, SubagentType: ReviewerAgent}
				if got := Decide(req); got.Decision != Defer || got.Cell != CellReviewer {
					t.Errorf("%s reviewer %s %s: %v %v, want defer from the reviewer cell", tool, mode, reach, got.Decision, got.Cell)
				}
				req.Answer = &Answer{ToolUseID: "toolu_rev", Decision: Allow}
				if got := Decide(req); got.Decision != Allow || got.Cell != CellReviewer {
					t.Errorf("%s reviewer %s %s resumed allow: %v %v", tool, mode, reach, got.Decision, got.Cell)
				}
				req.Answer = &Answer{ToolUseID: "toolu_other", Decision: Allow}
				if got := Decide(req); got.Decision != Defer {
					t.Errorf("%s reviewer %s %s, an answer for another call: %v, want defer", tool, mode, reach, got.Decision)
				}
				req.Answer = &Answer{ToolUseID: "toolu_rev", Decision: Deny}
				if got := Decide(req); got.Decision != Deny || got.Reason != AnswerDenyText {
					t.Errorf("%s reviewer %s %s resumed deny: %v %q", tool, mode, reach, got.Decision, got.Reason)
				}
				for _, other := range []string{"", "builder", "general-purpose", "Reviewer"} {
					req := Request{Tool: tool, ToolUseID: "toolu_rev", Mode: mode, Reach: reach, SubagentType: other}
					if got := Decide(req); got.Decision != Allow || got.Cell != CellAllow {
						t.Errorf("%s %q %s %s: %v %v, want allow", tool, other, mode, reach, got.Decision, got.Cell)
					}
				}
			}
		}
	}
	for _, tool := range []string{"TodoWrite", "Bash", "Read"} {
		if ReviewerCall(tool, ReviewerAgent) {
			t.Errorf("%s with subagent_type reviewer falls in the reviewer cell", tool)
		}
	}
	req := Request{Tool: "Agent", ToolUseID: "toolu_rev", Mode: ModePolicy, Reach: ReachCode, SubagentType: ReviewerAgent,
		Paths: []Path{cred}, Answer: &Answer{ToolUseID: "toolu_rev", Decision: Allow}}
	if got := Decide(req); got.Decision != Deny || got.Cell != CellDeny {
		t.Errorf("a reviewer call naming a credential path: %v %v, want deny", got.Decision, got.Cell)
	}
}
