// SPDX-License-Identifier: GPL-3.0-or-later

package session

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jasondorjeshort/broodforge/internal/git"
)

// subagentFixture is the hand-written stream of a driver launching the reviewer through Agent.
const subagentFixture = fixtures + "/subagent"

// TestSubagentCallsIndented: a subagent's text, call, decision and result are indented one
// level under its Agent line in `session tail`, from parent_tool_use_id; the Agent call's audit
// row names the subagent type (the review fact's input).
func TestSubagentCallsIndented(t *testing.T) {
	h := newHarness(t, "", true)
	ctx := context.Background()
	if _, err := h.svc.Start(ctx, StartRequest{Channel: "100", As: "1001", Fixture: subagentFixture}); err != nil {
		t.Fatal(err)
	}
	if _, err := h.svc.Submit(ctx, SendRequest{Session: "s0001", As: "1001", Text: "review it"}); err != nil {
		t.Fatal(err)
	}
	evs := waitTurn(t, h.svc, "s0001", 1)
	var got []string
	traced := map[EventType]bool{EventText: true, EventToolCall: true, EventPermissionDecision: true, EventToolResult: true}
	for _, ev := range evs {
		if traced[ev.Type] {
			got = append(got, string(ev.Type)+"  "+ev.Summary)
		}
	}
	want := []string{
		"text  Asking the reviewer to read the change.",
		"tool-call  [Agent]",
		"tool-call    [Read] hello.txt",
		"permission-decision    allow [Read] hello.txt (policy)",
		"tool-result    [Read] hello.txt",
		"text    No findings.",
		"permission-decision  allow [Agent] (policy)",
		"tool-result  [Agent]",
		"text  Reviewed: no findings.",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("session tail\n got:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	for _, ev := range evs {
		if ev.ToolUseID == "toolu_sub_read" && (ev.Parent != "toolu_sub_agent" || ev.Depth != 1) {
			t.Errorf("%s of the subagent's Read: parent %q depth %d", ev.Type, ev.Parent, ev.Depth)
		}
	}
	if n := count(t, h.st, `SELECT count(*) FROM audit_event WHERE kind = 'permission-decision' AND session_id = 's0001'
    AND details LIKE '%"agent":"reviewer"%' AND details LIKE '%"toolUseId":"toolu_sub_agent"%'`); n != 1 {
		t.Errorf("%d Agent rows naming the reviewer, want 1", n)
	}
}

// TestWorkflowCopyAtStart: Config.Workflow's copy reaches every process: the header at the top
// of instructions.md, the project's text after a blank line, --agents when the copy has agents,
// BROODFORGE_WORKFLOW when something was copied; `session show`'s workflow and hooks from the
// session-started event; --header prints the header alone.
func TestWorkflowCopyAtStart(t *testing.T) {
	h := newHarness(t, "", true)
	ctx := context.Background()
	var asked WorkflowRequest
	h.svc.cfg.Workflow = func(_ context.Context, req WorkflowRequest) (WorkflowCopy, error) {
		asked = req
		if err := os.MkdirAll(req.Dir, 0o755); err != nil {
			return WorkflowCopy{}, err
		}
		for name, text := range map[string]string{"instructions.md": "Contents of CLAUDE.md:\n\nBe brief.\n", "agents.json": "{}\n"} {
			if err := os.WriteFile(filepath.Join(req.Dir, name), []byte(text), 0o644); err != nil {
				return WorkflowCopy{}, err
			}
		}
		return WorkflowCopy{Line: "abc123 2026-10-23T09:30:00Z", Copied: true, Branch: "main", Commit: "abc123",
			Agents: req.Dir + "/agents.json", Hooks: []string{"git-guard.js", "post-edit.js"}}, nil
	}
	if _, err := h.svc.Start(ctx, StartRequest{Channel: "100", As: "1001"}); err != nil {
		t.Fatal(err)
	}
	dir := h.root + "/workflow/s0001"
	if asked.Dir != dir || asked.Branch != "main" || asked.Canonical != h.root+"/canonical" {
		t.Errorf("asked %+v", asked)
	}
	if _, err := h.svc.Submit(ctx, SendRequest{Session: "s0001", As: "1001", Text: "hi"}); err != nil {
		t.Fatal(err)
	}
	waitTurn(t, h.svc, "s0001", 1)
	req := received(t, h.root, "s0001", 1)
	if req.SystemPromptFile != dir+"/instructions.md" || req.AgentsFile != dir+"/agents.json" || hookVar(req, envWorkflow) != dir {
		t.Errorf("request: file %q agents %q workflow %q", req.SystemPromptFile, req.AgentsFile, hookVar(req, envWorkflow))
	}
	b, err := os.ReadFile(req.SystemPromptFile)
	if err != nil || !strings.HasPrefix(string(b), headerPrecedence+"\n") ||
		!strings.Contains(string(b), "taken from main at abc123 on ") || !strings.HasSuffix(string(b), ".\n\nContents of CLAUDE.md:\n\nBe brief.\n") {
		t.Errorf("instructions.md:\n%s %v", b, err)
	}
	d, err := h.svc.Inspect(ctx, "s0001", true)
	if err != nil || d.Workflow == nil || *d.Workflow != "abc123 2026-10-23T09:30:00Z" ||
		strings.Join(d.Hooks, ", ") != "git-guard.js, post-edit.js" ||
		// The harness's canonical is no repository: the review fact is an error, never `-`.
		d.Review == nil || d.Review.Error == nil || d.Review.Error.Code != git.CodeSessionRef {
		t.Errorf("show: %+v %v", d, err)
	}
	if !strings.HasPrefix(d.Header, headerPrecedence+"\n") || strings.Contains(d.Header, "Be brief") || !strings.HasSuffix(d.Header, ".\n") {
		t.Errorf("header %q", d.Header)
	}
}

// TestWorkflowProbeRefusesStart: a probe failure refuses the start with its code before the
// clone, keeps the row ended `refused: workflow-probe-failed`, audits action-refused, removes the
// copy, and opens no session (no Started call: nothing is posted anywhere; the starter's own
// reply is the only notice, the owner's decision of 2026-10-23).
func TestWorkflowProbeRefusesStart(t *testing.T) {
	h := newHarness(t, "", true)
	ctx := context.Background()
	h.svc.cfg.Workflow = func(_ context.Context, req WorkflowRequest) (WorkflowCopy, error) {
		return WorkflowCopy{}, &WorkflowRefusal{Code: "workflow-probe-failed", Message: "nosuchruntime x.js needs nosuchruntime"}
	}
	var started []string
	h.svc.cfg.Started = func(id string) { started = append(started, id) }
	created := false
	create := h.svc.cfg.CreateTree
	h.svc.cfg.CreateTree = func(ctx context.Context, s git.Session) (string, error) { created = true; return create(ctx, s) }

	_, err := h.svc.Start(ctx, StartRequest{Channel: "100", As: "1001"})
	var se *Error
	if !errors.As(err, &se) || se.Code != "workflow-probe-failed" || se.Message != "nosuchruntime x.js needs nosuchruntime" {
		t.Fatalf("start: %v", err)
	}
	if created {
		t.Error("the tree was cloned for a refused probe")
	}
	if len(started) != 0 {
		t.Errorf("a refused start was announced: %q", started)
	}
	if n := count(t, h.st, `SELECT count(*) FROM session WHERE id = 's0001' AND state = 'ended' AND end_outcome = 'refused: workflow-probe-failed'`); n != 1 {
		t.Error("the session row is not ended as refused")
	}
	if n := count(t, h.st, `SELECT count(*) FROM audit_event WHERE kind = 'action-refused' AND session_id = 's0001'
    AND details LIKE '%"reason":"workflow"%' AND details LIKE '%needs nosuchruntime%'`); n != 1 {
		t.Error("no action-refused row for the probe")
	}
	if _, err := os.Stat(filepath.Join(h.root, "workflow", "s0001")); !os.IsNotExist(err) {
		t.Errorf("the copy survived: %v", err)
	}
}
