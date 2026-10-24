// SPDX-License-Identifier: GPL-3.0-or-later

package session

import (
	"context"
	"strings"
	"testing"

	"github.com/jasondorjeshort/broodforge/internal/store"
)

// TestModelAndEffort: P12a's service side. A start's --model is resolved and copied to the
// row; the driver's switch updates the row, is audited with from and to, and reaches the
// engine at the next turn; a bad value is bad-args before anything is written.
func TestModelAndEffort(t *testing.T) {
	h := newHarness(t, "", true)
	ctx := context.Background()
	if _, err := h.svc.Start(ctx, StartRequest{Channel: "100", As: "1001", Model: "gpt"}); code(err) != CodeBadArgs {
		t.Fatalf("start with model gpt: %v", err)
	}
	if n := count(t, h.st, "SELECT count(*) FROM session"); n != 0 {
		t.Fatalf("a refused --model wrote %d session rows", n)
	}
	if _, err := h.svc.Start(ctx, StartRequest{Channel: "100", As: "1001", Model: "sonnet"}); err != nil {
		t.Fatal(err)
	}
	d, err := h.svc.Show("s0001")
	if err != nil || d.Model == nil || *d.Model != "claude-sonnet-5-5" || d.Effort != nil {
		t.Fatalf("show after start: %+v %v", d, err)
	}
	evs, _ := h.svc.Events("s0001", 0)
	if st := evs[0].Start; st == nil || st.Model != "claude-sonnet-5-5" || st.ChannelModel != "" || st.Effort != "" {
		t.Errorf("start info %+v", evs[0].Start)
	}
	rows, err := List(h.st, "")
	if err != nil || len(rows) != 1 || rows[0].Model == nil || *rows[0].Model != "claude-sonnet-5-5" {
		t.Errorf("list %+v %v", rows, err)
	}

	act := ActRequest{Session: "s0001", As: "1001"}
	if _, err := h.svc.SetModel(ModelRequest{ActRequest: ActRequest{Session: "s0001", As: "1002"}, Model: "fable"}); code(err) != CodeRefused {
		t.Errorf("model by a non-driver: %v", err)
	}
	for _, bad := range []string{"gpt", "claude-opus-5-55", "claude-opus-5-5 --x", ""} {
		if _, err := h.svc.SetModel(ModelRequest{ActRequest: act, Model: bad}); code(err) != CodeBadArgs {
			t.Errorf("model %q: %v", bad, err)
		}
	}
	if _, err := h.svc.SetEffort(EffortRequest{ActRequest: act, Effort: "turbo"}); code(err) != CodeBadArgs ||
		!strings.Contains(err.Error(), "want low, medium, high, xhigh, max") {
		t.Errorf("effort turbo: %v", err)
	}
	e, err := h.svc.SetModel(ModelRequest{ActRequest: act, Model: "fable"})
	if err != nil || e.Type != EventModelChanged || e.Turn != 1 || e.Result != "claude-fable-5-1" || e.Summary != "From the next turn: Fable." {
		t.Fatalf("model: %+v %v", e, err)
	}
	e, err = h.svc.SetEffort(EffortRequest{ActRequest: act, Effort: "high"})
	if err != nil || e.Type != EventEffortChanged || e.Turn != 1 || e.Result != "high" || e.Summary != "From the next turn: effort high." {
		t.Fatalf("effort: %+v %v", e, err)
	}
	if n := count(t, h.st, `SELECT count(*) FROM audit_event WHERE kind = 'model-changed' AND details = '{"from":"claude-sonnet-5-5","to":"claude-fable-5-1"}'`); n != 1 {
		t.Error("no model-changed audit event with from and to")
	}
	if n := count(t, h.st, `SELECT count(*) FROM audit_event WHERE kind = 'effort-changed' AND details = '{"to":"high"}'`); n != 1 {
		t.Error("no effort-changed audit event")
	}

	// A typed full id the map knows passes as typed.
	if _, err := h.svc.SetModel(ModelRequest{ActRequest: act, Model: "claude-haiku-5-5-20261001"}); err != nil {
		t.Fatal(err)
	}
	if _, err := h.svc.Submit(ctx, SendRequest{Session: "s0001", As: "1001", Text: "go"}); err != nil {
		t.Fatal(err)
	}
	waitFor(t, h.svc, "s0001", 0, isType(EventTurnEnded, 1))
	if req := received(t, h.root, "s0001", 1); req.Model != "claude-haiku-5-5-20261001" || req.Effort != "high" {
		t.Errorf("turn 1 ran with model %q effort %q", req.Model, req.Effort)
	}
	// The spawn reads the row: a change written there is what the next turn gets.
	exec(t, h.st, "UPDATE session SET model = 'claude-opus-5-5', effort = NULL WHERE id = 's0001'")
	if _, err := h.svc.Submit(ctx, SendRequest{Session: "s0001", As: "1001", Text: "again"}); err != nil {
		t.Fatal(err)
	}
	waitFor(t, h.svc, "s0001", 0, isType(EventTurnEnded, 2))
	if req := received(t, h.root, "s0001", 2); req.Model != "claude-opus-5-5" || req.Effort != "" {
		t.Errorf("turn 2 ran with model %q effort %q", req.Model, req.Effort)
	}
}

// TestSwitchClosed: P12b. On an ended session the three switches are session-closed with end's
// text, each audited action-refused with the reason closed, and nothing is written: the row, the
// *-changed audit rows and the events stay as they were.
func TestSwitchClosed(t *testing.T) {
	h := newHarness(t, "", true)
	ctx := context.Background()
	if _, err := h.svc.Start(ctx, StartRequest{Channel: "100", As: "1001", Model: "sonnet"}); err != nil {
		t.Fatal(err)
	}
	act := ActRequest{Session: "s0001", As: "1001"}
	if _, err := h.svc.End(ctx, EndRequest{ActRequest: act}); err != nil {
		t.Fatal(err)
	}
	row := func() string {
		var s string
		if err := h.st.Query("SELECT coalesce(model, '-') || ' ' || coalesce(effort, '-') || ' ' || mode FROM session WHERE id = 's0001'", nil,
			func(r *store.Row) error { s = r.Text(0); return nil }); err != nil {
			t.Fatal(err)
		}
		return s
	}
	before, _ := h.svc.Events("s0001", 0)
	rowBefore := row()
	for name, call := range map[string]func() (Event, error){
		"mode":   func() (Event, error) { return h.svc.SetMode(ModeRequest{ActRequest: act, Mode: "prompt"}) },
		"model":  func() (Event, error) { return h.svc.SetModel(ModelRequest{ActRequest: act, Model: "opus"}) },
		"effort": func() (Event, error) { return h.svc.SetEffort(EffortRequest{ActRequest: act, Effort: "high"}) },
	} {
		if _, err := call(); code(err) != CodeSessionClosed || err.Error() != "session-closed: s0001 is ended" {
			t.Errorf("%s on an ended session: %v", name, err)
		}
	}
	if got := row(); got != rowBefore {
		t.Errorf("the row changed: %q, was %q", got, rowBefore)
	}
	if n := count(t, h.st, `SELECT count(*) FROM audit_event WHERE kind IN ('mode-changed', 'model-changed', 'effort-changed')`); n != 0 {
		t.Errorf("%d *-changed audit rows", n)
	}
	for _, action := range []string{"mode", "model", "effort"} {
		if n := count(t, h.st, `SELECT count(*) FROM audit_event WHERE kind = 'action-refused' AND session_id = 's0001' AND details = '{"action":"`+
			action+`","message":"s0001 is ended","reason":"closed"}'`); n != 1 {
			t.Errorf("%d action-refused rows for %s, want 1", n, action)
		}
	}
	if after, _ := h.svc.Events("s0001", 0); len(after) != len(before) {
		t.Errorf("%d events after the refusals, %d before", len(after), len(before))
	}
}

// TestSwitchClosedByNonDriver: P12b. The closed check comes before the driver check, so a
// non-driver's late switch on an ended session is session-closed, audited with the reason
// closed and never with the reason driver.
func TestSwitchClosedByNonDriver(t *testing.T) {
	h := newHarness(t, "", true)
	ctx := context.Background()
	if _, err := h.svc.Start(ctx, StartRequest{Channel: "100", As: "1001", Model: "sonnet"}); err != nil {
		t.Fatal(err)
	}
	if _, err := h.svc.End(ctx, EndRequest{ActRequest: ActRequest{Session: "s0001", As: "1001"}}); err != nil {
		t.Fatal(err)
	}
	other := ActRequest{Session: "s0001", As: "1002"}
	for name, call := range map[string]func() (Event, error){
		"mode":   func() (Event, error) { return h.svc.SetMode(ModeRequest{ActRequest: other, Mode: "prompt"}) },
		"model":  func() (Event, error) { return h.svc.SetModel(ModelRequest{ActRequest: other, Model: "opus"}) },
		"effort": func() (Event, error) { return h.svc.SetEffort(EffortRequest{ActRequest: other, Effort: "high"}) },
	} {
		if _, err := call(); code(err) != CodeSessionClosed || err.Error() != "session-closed: s0001 is ended" {
			t.Errorf("%s by a non-driver on an ended session: %v", name, err)
		}
		if n := count(t, h.st, `SELECT count(*) FROM audit_event WHERE kind = 'action-refused' AND session_id = 's0001' AND details = '{"action":"`+
			name+`","message":"s0001 is ended","reason":"closed"}'`); n != 1 {
			t.Errorf("%d closed action-refused rows for %s, want 1", n, name)
		}
	}
	if n := count(t, h.st, `SELECT count(*) FROM audit_event WHERE kind = 'action-refused' AND details LIKE '%"reason":"driver"%'`); n != 0 {
		t.Errorf("%d driver refusals on an ended session", n)
	}
}

// TestStartModelFromTheChannel: absent a choice, the channel's model (an alias resolved); an
// explicit choice wins over a channel value policy does not know; without one, that value
// refuses the start, but only after the registration and role checks.
func TestStartModelFromTheChannel(t *testing.T) {
	h := newHarness(t, "", true)
	ctx := context.Background()
	model, effort, channel, err := h.svc.startModel(ActorHost, Channel{ID: "9", Model: "opus", Effort: "low"}, "")
	if err != nil || model != "claude-opus-5-5" || channel != "claude-opus-5-5" || effort != "low" {
		t.Errorf("channel opus: %q %q %q %v", model, effort, channel, err)
	}
	model, _, channel, err = h.svc.startModel(ActorHost, Channel{ID: "9", Model: "opus"}, "HAIKU")
	if err != nil || model != "claude-haiku-5-5" || channel != "claude-opus-5-5" {
		t.Errorf("--model HAIKU on an opus channel: %q %q %v", model, channel, err)
	}
	if _, _, _, err := h.svc.startModel(ActorHost, Channel{ID: "9", Effort: "turbo"}, ""); code(err) != CodeBadArgs {
		t.Errorf("channel effort turbo: %v", err)
	}

	// 1002 is registered but holds no start role; 1003 is not registered: neither sees the
	// configuration's text, and only their own refusal is audited.
	if _, err := h.svc.Start(ctx, StartRequest{Channel: "300", As: "1003", Roles: "10"}); code(err) != CodeUnregistered {
		t.Errorf("unregistered on 300: %v", err)
	}
	if _, err := h.svc.Start(ctx, StartRequest{Channel: "300", As: "1002", Roles: "11"}); code(err) != CodeRefused {
		t.Errorf("no role on 300: %v", err)
	}
	if n := count(t, h.st, `SELECT count(*) FROM audit_event WHERE kind = 'action-refused' AND details LIKE '%channel-model%'`); n != 0 {
		t.Errorf("%d channel-model refusals for people the checks refused first", n)
	}
	if _, err := h.svc.Start(ctx, StartRequest{Channel: "300", As: "1001", Roles: "10"}); code(err) != CodeBadArgs ||
		!strings.Contains(err.Error(), `channel 300 has model "gpt-5" in its configuration`) {
		t.Errorf("no choice on 300: %v", err)
	}
	if n := count(t, h.st, `SELECT count(*) FROM audit_event WHERE kind = 'action-refused' AND details LIKE '%channel-model%'`); n != 1 {
		t.Errorf("%d channel-model refusals, want 1", n)
	}
	st, err := h.svc.Start(ctx, StartRequest{Channel: "300", As: "1001", Roles: "10", Model: "sonnet"})
	if err != nil {
		t.Fatalf("an explicit choice on 300: %v", err)
	}
	evs, _ := h.svc.Events(st.ID, 0)
	if s := evs[0].Start; s == nil || s.Model != "claude-sonnet-5-5" || s.ChannelModel != "" {
		t.Errorf("start info %+v", evs[0].Start)
	}
}
