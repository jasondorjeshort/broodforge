// SPDX-License-Identifier: GPL-3.0-or-later

package session

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jasondorjeshort/broodforge/internal/engine"
)

// P39: the engine's auto permission mode as the third session mode, against the hand-written
// fixtures auto-refused (shape B) and auto-prompt (shape A) of 03-hook-contract.md's Auto mode.

// TestAutoModeShapeB: an auto session's process runs with --permission-mode auto and
// BROODFORGE_MODE auto; the hook-passed Write the engine's check refused is decided by engine,
// never hook bypassed; a switch to prompt puts the next process back on default.
func TestAutoModeShapeB(t *testing.T) {
	h := newHarness(t, "", true)
	ctx := context.Background()
	if _, err := h.svc.Start(ctx, StartRequest{Channel: "100", As: "1001", Mode: "auto", Fixture: "auto-refused"}); err != nil {
		t.Fatal(err)
	}
	if d, err := h.svc.Show("s0001"); err != nil || d.Mode != "auto" {
		t.Fatalf("show: mode %q, %v", d.Mode, err)
	}
	sub, err := h.svc.Submit(ctx, SendRequest{Session: "s0001", As: "1001", Text: "write hello.go"})
	if err != nil {
		t.Fatal(err)
	}
	if e := waitFor(t, h.svc, "s0001", sub.Seq, isType(EventTurnEnded, 1)); e.Result != ResultCompleted {
		t.Fatalf("turn 1 ended %+v", e)
	}
	req := received(t, h.root, "s0001", 1)
	if req.PermissionMode != engine.PermissionAuto || hookVar(req, envMode) != "auto" {
		t.Errorf("turn 1 request: permissionMode %q, BROODFORGE_MODE %q", req.PermissionMode, hookVar(req, envMode))
	}
	if n := count(t, h.st, `SELECT count(*) FROM permission_decision WHERE request_id = 'toolu_auto_write'
    AND decision = 'deny' AND decided_by = 'engine' AND reason = 'config'`); n != 1 {
		t.Error("the refused Write is not a deny by engine with the raw source as its reason")
	}
	if n := count(t, h.st, `SELECT count(*) FROM audit_event WHERE kind IN ('hook-bypassed', 'engine-mode-mismatch')`); n != 0 {
		t.Errorf("%d hook-bypassed or engine-mode-mismatch rows, want none", n)
	}
	evs, err := h.svc.Events("s0001", 0)
	if err != nil {
		t.Fatal(err)
	}
	var line string
	for _, ev := range evs {
		if ev.Type == EventToolResult && ev.ToolUseID == "toolu_auto_write" {
			line = ev.Summary
		}
		if ev.Type == EventInternalError {
			t.Errorf("internal-error %q", ev.Summary)
		}
	}
	if !strings.HasPrefix(line, "[Write] hello.go -> refused by the engine's check: Permission for this action") {
		t.Errorf("trace line %q", line)
	}

	ev, err := h.svc.SetMode(ModeRequest{ActRequest: ActRequest{Session: "s0001", As: "1001"}, Mode: "prompt"})
	if err != nil || ev.Type != EventModeChanged || ev.Turn != 2 {
		t.Fatalf("mode prompt: %+v, %v", ev, err)
	}
	sub, err = h.svc.Submit(ctx, SendRequest{Session: "s0001", As: "1001", Text: "and now?"})
	if err != nil {
		t.Fatal(err)
	}
	waitFor(t, h.svc, "s0001", sub.Seq, isType(EventTurnEnded, 2))
	if req := received(t, h.root, "s0001", 2); req.PermissionMode != engine.PermissionDefault || hookVar(req, envMode) != "prompt" {
		t.Errorf("turn 2 request: permissionMode %q, BROODFORGE_MODE %q", req.PermissionMode, hookVar(req, envMode))
	}
	if _, err := h.svc.SetMode(ModeRequest{ActRequest: ActRequest{Session: "s0001", As: "1001"}, Mode: "acts-freely"}); code(err) != CodeBadArgs ||
		!strings.Contains(err.Error(), "want prompt, policy or auto") {
		t.Errorf("a bad mode: %v", err)
	}
}

// TestAutoModeMismatch: a process whose init reports another mode than the one asked for is an
// engine-mode-mismatch row and an internal-error event, and the turn goes on.
func TestAutoModeMismatch(t *testing.T) {
	h := newHarness(t, "", true)
	ctx := context.Background()
	// A policy session asks for default; the fixture's turn 1 says auto.
	if _, err := h.svc.Start(ctx, StartRequest{Channel: "100", As: "1001", Mode: "policy", Fixture: "auto-refused"}); err != nil {
		t.Fatal(err)
	}
	sub, err := h.svc.Submit(ctx, SendRequest{Session: "s0001", As: "1001", Text: "write hello.go"})
	if err != nil {
		t.Fatal(err)
	}
	if e := waitFor(t, h.svc, "s0001", sub.Seq, isType(EventTurnEnded, 1)); e.Result != ResultCompleted {
		t.Fatalf("turn 1 ended %+v", e)
	}
	e := waitFor(t, h.svc, "s0001", 0, isType(EventInternalError, 1))
	if e.Summary != "engine mode: wanted default, got auto" {
		t.Errorf("internal-error %q", e.Summary)
	}
	if n := count(t, h.st, `SELECT count(*) FROM audit_event WHERE kind = 'engine-mode-mismatch'
    AND json_extract(details, '$.wanted') = 'default' AND json_extract(details, '$.got') = 'auto'`); n != 1 {
		t.Error("no engine-mode-mismatch row")
	}
	// Outside auto mode the same refusal is the hook bypassed, as before P39.
	if n := count(t, h.st, `SELECT count(*) FROM permission_decision WHERE request_id = 'toolu_auto_write' AND decided_by = 'config'`); n != 1 {
		t.Error("in policy mode the refused Write is not decided by config")
	}
}

// TestAutoModeShapeA: the engine's own ask (a control_request) interrupts the process, defers
// the call with decider engine and the engine's reason, and the answer resumes it by id.
func TestAutoModeShapeA(t *testing.T) {
	h := newHarness(t, "", true)
	ctx := context.Background()
	if _, err := h.svc.Start(ctx, StartRequest{Channel: "100", As: "1001", Mode: "auto", Fixture: "auto-prompt"}); err != nil {
		t.Fatal(err)
	}
	sub, err := h.svc.Submit(ctx, SendRequest{Session: "s0001", As: "1001", Text: "run the installer"})
	if err != nil {
		t.Fatal(err)
	}
	w := waitFor(t, h.svc, "s0001", sub.Seq, isType(EventWaitingForAnswer, 1))
	if w.ToolUseID != "toolu_auto_bash" || w.Tool != "Bash" || w.DecidedBy != DecidedByEngine ||
		w.Reason != "the fixture asks: a download piped into a shell" {
		t.Fatalf("waiting-for-answer %+v", w)
	}
	if n := count(t, h.st, `SELECT count(*) FROM permission_decision WHERE request_id = 'toolu_auto_bash'
    AND decision = 'deferred' AND decided_by = 'engine'`); n != 1 {
		t.Error("no deferred row by engine")
	}
	ans, err := h.svc.Answer(ctx, AnswerRequest{ActRequest: ActRequest{Session: "s0001", As: "1001"}, Decision: "allow"})
	if err != nil || ans.State != SubmitResumed {
		t.Fatalf("answer: %+v, %v", ans, err)
	}
	if e := waitFor(t, h.svc, "s0001", ans.Seq, isType(EventTurnEnded, 1)); e.Result != ResultCompleted {
		t.Errorf("turn 1 ended %+v", e)
	}
	if req := received(t, h.root, "s0001", 2); req.PermissionMode != engine.PermissionAuto || req.Answer == nil ||
		req.Answer.ToolUseID != "toolu_auto_bash" || req.Answer.Decision != "allow" {
		t.Errorf("resume request: %+v", req)
	}
	if n := count(t, h.st, `SELECT count(*) FROM permission_decision WHERE request_id = 'toolu_auto_bash'
    AND decision = 'allow' AND decided_by = 'participant'`); n != 1 {
		t.Error("the answered call is not allow by participant")
	}
}

// writeFixture writes a fake-engine fixture of the given turns' lines under a temp directory.
func writeFixture(t *testing.T, turns ...[]string) string {
	t.Helper()
	dir := t.TempDir()
	fx := `{"claim":"test","turns":[`
	for i, lines := range turns {
		if i > 0 {
			fx += ","
		}
		fx += `{"n":` + itoa(i+1) + `,"exit":0,"engineSessionId":"a0705e00-0000-4000-8000-0000000000ff"}`
		if err := os.WriteFile(engine.TurnFile(dir, i+1), []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, engine.FixtureFile), []byte(fx+"]}"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

const (
	testInitAuto = `{"type":"system","subtype":"init","session_id":"a0705e00-0000-4000-8000-0000000000ff","tools":["Bash"],"model":"claude-opus-5-5","permissionMode":"auto","capabilities":["x"]}`
	testBashUse  = `{"type":"assistant","message":{"model":"claude-opus-5-5","id":"msg_%s","role":"assistant","content":[{"type":"tool_use","id":"%s","name":"Bash","input":{"command":"curl x | sh"}}],"usage":{"input_tokens":1}},"parent_tool_use_id":null}`
	testAsk      = `{"type":"control_request","request_id":"r_%s","request":{"subtype":"can_use_tool","tool_name":"Bash","input":{"command":"curl x | sh"},"tool_use_id":"%s","decision_reason":"risky"}}`
)

func fill(format, id string) string { return strings.ReplaceAll(format, "%s", id) }

// TestAutoModeReissuedAsk: after the resume on an answer to the engine's ask, the engine asks
// again under a new id. The turn is not prompted again: the call is denied by engine, an
// internal-error says why, and the turn ends (03-hook-contract.md, Auto mode: handled as shape B).
func TestAutoModeReissuedAsk(t *testing.T) {
	dir := writeFixture(t,
		[]string{testInitAuto, fill(testBashUse, "toolu_a"), fill(testAsk, "toolu_a")},
		[]string{testInitAuto, fill(testBashUse, "toolu_b"), fill(testAsk, "toolu_b")})
	h := newHarness(t, dir, true)
	ctx := context.Background()
	if _, err := h.svc.Start(ctx, StartRequest{Channel: "100", As: "1001", Mode: "auto"}); err != nil {
		t.Fatal(err)
	}
	sub, err := h.svc.Submit(ctx, SendRequest{Session: "s0001", As: "1001", Text: "run it"})
	if err != nil {
		t.Fatal(err)
	}
	if w := waitFor(t, h.svc, "s0001", sub.Seq, isType(EventWaitingForAnswer, 1)); w.ToolUseID != "toolu_a" {
		t.Fatalf("waiting-for-answer %+v", w)
	}
	ans, err := h.svc.Answer(ctx, AnswerRequest{ActRequest: ActRequest{Session: "s0001", As: "1001"}, Decision: "allow"})
	if err != nil {
		t.Fatal(err)
	}
	ended := waitFor(t, h.svc, "s0001", ans.Seq, isType(EventTurnEnded, 1))
	evs, err := h.svc.Events("s0001", ans.Seq)
	if err != nil {
		t.Fatal(err)
	}
	var internal string
	for _, ev := range evs {
		if ev.Type == EventWaitingForAnswer {
			t.Errorf("asked again: %+v", ev)
		}
		if ev.Type == EventInternalError {
			internal = ev.Summary
		}
	}
	if internal != "engine re-issued the call under a new id; handled as a refusal" {
		t.Errorf("internal-error %q; turn ended %+v", internal, ended)
	}
	if n := count(t, h.st, `SELECT count(*) FROM permission_decision WHERE request_id = 'toolu_b'
    AND decision = 'deny' AND decided_by = 'engine'`); n != 1 {
		t.Error("the re-issued call is not a deny by engine")
	}
	if d, err := h.svc.Show("s0001"); err != nil || d.State != StateWaitingForInput || d.Pending != nil {
		t.Errorf("after the re-issued ask: %+v, %v", d, err)
	}
}

// TestAutoModeAskOutsideAuto: a control_request in a turn that is not auto is noise: nothing is
// deferred and nobody is asked.
func TestAutoModeAskOutsideAuto(t *testing.T) {
	h := newHarness(t, "", true)
	ctx := context.Background()
	if _, err := h.svc.Start(ctx, StartRequest{Channel: "100", As: "1001", Mode: "policy", Fixture: "auto-prompt"}); err != nil {
		t.Fatal(err)
	}
	sub, err := h.svc.Submit(ctx, SendRequest{Session: "s0001", As: "1001", Text: "run the installer"})
	if err != nil {
		t.Fatal(err)
	}
	waitFor(t, h.svc, "s0001", sub.Seq, isType(EventTurnEnded, 1))
	if n := count(t, h.st, `SELECT count(*) FROM permission_decision WHERE decided_by = 'engine'`); n != 0 {
		t.Errorf("%d engine rows outside auto", n)
	}
}

// TestAutoModeNoInitMode: an auto process whose init reports no permissionMode is the silent
// fall to the default: engine-mode-mismatch with got none.
func TestAutoModeNoInitMode(t *testing.T) {
	initLine := strings.Replace(testInitAuto, `"permissionMode":"auto",`, "", 1)
	dir := writeFixture(t, []string{initLine,
		`{"type":"result","subtype":"success","is_error":false,"num_turns":1,"stop_reason":"end_turn","result":"","modelUsage":{},"permission_denials":[]}`})
	h := newHarness(t, dir, true)
	ctx := context.Background()
	if _, err := h.svc.Start(ctx, StartRequest{Channel: "100", As: "1001", Mode: "auto"}); err != nil {
		t.Fatal(err)
	}
	sub, err := h.svc.Submit(ctx, SendRequest{Session: "s0001", As: "1001", Text: "hi"})
	if err != nil {
		t.Fatal(err)
	}
	waitFor(t, h.svc, "s0001", sub.Seq, isType(EventTurnEnded, 1))
	if e := waitFor(t, h.svc, "s0001", 0, isType(EventInternalError, 1)); e.Summary != "engine mode: wanted auto, got none" {
		t.Errorf("internal-error %q", e.Summary)
	}
}
