// SPDX-License-Identifier: GPL-3.0-or-later

package session

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jasondorjeshort/broodforge/internal/engine"
	"github.com/jasondorjeshort/broodforge/internal/git"
)

// waitingWithQueue starts a prompt-mode session on `defer`, leaves it waiting for an answer
// with one more message queued, and returns the harness.
func waitingWithQueue(t *testing.T, req StartRequest) *harness {
	t.Helper()
	h := newHarness(t, "", true)
	ctx := context.Background()
	req.Channel, req.As, req.Mode, req.Fixture = "100", "1001", "prompt", "defer"
	if _, err := h.svc.Start(ctx, req); err != nil {
		t.Fatal(err)
	}
	sub, err := h.svc.Submit(ctx, SendRequest{Session: "s0001", As: "1001", Text: "write hi.txt"})
	if err != nil {
		t.Fatal(err)
	}
	waitFor(t, h.svc, "s0001", sub.Seq, isType(EventWaitingForAnswer, 1))
	if q, err := h.svc.Submit(ctx, SendRequest{Session: "s0001", As: "1001", Text: "and then this"}); err != nil || q.State != SubmitQueued {
		t.Fatalf("queue: %+v, %v", q, err)
	}
	return h
}

// assertClosedPending: the waiting turn's row is ended, the call is dropped with the cause,
// turn-ended and message-dropped were published, the queued message is audited.
func assertClosedPending(t *testing.T, h *harness, cause string) {
	t.Helper()
	if n := count(t, h.st, `SELECT count(*) FROM turn WHERE session_id = 's0001' AND seq = 1 AND ended_at IS NOT NULL AND result = 'interrupted'`); n != 1 {
		t.Error("the waiting turn's row is not ended")
	}
	if n := count(t, h.st, `SELECT count(*) FROM permission_decision WHERE request_id = 'toolu_01NmkmkiJgpriuU7yQrtDWYi'
    AND decision = 'dropped' AND decided_by = 'instance' AND reason = '`+cause+` before an answer'`); n != 1 {
		t.Errorf("the pending call is not dropped with %q", cause)
	}
	if n := count(t, h.st, `SELECT count(*) FROM audit_event WHERE kind = 'action-refused' AND details LIKE '%"reason":"message-dropped"%'`); n != 1 {
		t.Error("the queued message's drop is not audited")
	}
	evs, _ := h.svc.Events("s0001", 0)
	ended, dropped := false, false
	for _, ev := range evs {
		ended = ended || (ev.Type == EventTurnEnded && ev.Turn == 1 && ev.Summary == cause)
		dropped = dropped || (ev.Type == EventMessageDropped && ev.Summary == cause+": and then this")
	}
	if !ended || !dropped {
		t.Errorf("turn-ended %v, message-dropped %v in %v", ended, dropped, summaries(evs))
	}
}

func TestEndWhileACallWaits(t *testing.T) {
	h := waitingWithQueue(t, StartRequest{})
	if _, err := h.svc.End(context.Background(), EndRequest{ActRequest: ActRequest{Session: "s0001", As: "1001"}}); err != nil {
		t.Fatal(err)
	}
	assertClosedPending(t, h, "the session ended")
}

func TestIdleWhileACallWaits(t *testing.T) {
	h := waitingWithQueue(t, StartRequest{IdleTimeoutSeconds: 60})
	h.clock.Advance(60 * time.Second)
	waitFor(t, h.svc, "s0001", 0, isType(EventSessionEnded, 0))
	assertClosedPending(t, h, "the session ended: no activity")
}

func TestStopWhileACallWaits(t *testing.T) {
	h := waitingWithQueue(t, StartRequest{})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := h.svc.Shutdown(ctx); err != nil {
		t.Fatal(err)
	}
	assertClosedPending(t, h, "the bot stopped")
}

// TestIdleTickRacingASend: a tick whose generation a send has since replaced does nothing,
// though it was due: the send was activity.
func TestIdleTickRacingASend(t *testing.T) {
	h := newHarness(t, delayed(t, 200*time.Millisecond), true)
	ctx := context.Background()
	if _, err := h.svc.Start(ctx, StartRequest{Channel: "100", As: "1001", IdleTimeoutSeconds: 60}); err != nil {
		t.Fatal(err)
	}
	l := h.svc.lookup("s0001")
	l.mu.Lock()
	gen := l.idleGen
	l.mu.Unlock()
	if _, err := h.svc.Submit(ctx, SendRequest{Session: "s0001", As: "1001", Text: "go"}); err != nil {
		t.Fatal(err)
	}
	h.svc.idleFired(l, gen) // the tick that was due when the send came in
	if _, err := h.svc.end(ctx, l, ActorInstance, "idle", "x", gen); code(err) != CodeSessionClosed {
		t.Errorf("an idle end with a stale generation: %v", err)
	}
	if d, _ := h.svc.Show("s0001"); d.State != StateRunningTurn {
		t.Errorf("the turn the send started is %s", d.State)
	}
	waitFor(t, h.svc, "s0001", 0, isType(EventTurnEnded, 1))
}

// TestIdleRefusedHandoffTriesOnce: an idle end whose handoff is refused stays open and does not
// try again every period; the next activity re-arms the timer.
func TestIdleRefusedHandoffTriesOnce(t *testing.T) {
	h := newHarness(t, "", true)
	ctx := context.Background()
	if _, err := h.svc.Start(ctx, StartRequest{Channel: "100", As: "1001", IdleTimeoutSeconds: 60}); err != nil {
		t.Fatal(err)
	}
	calls := 0
	h.mu.Lock()
	h.handoff = func(string) (git.Outcome, error) {
		calls++
		return git.Outcome{}, &git.Error{Code: "handoff-refused-attribution", Message: "refused"}
	}
	h.mu.Unlock()
	h.clock.Advance(60 * time.Second)
	h.clock.Advance(600 * time.Second)
	if calls != 1 || h.clock.Pending() != 0 {
		t.Errorf("%d handoffs, %d timers pending; want 1 and 0", calls, h.clock.Pending())
	}
	if d, _ := h.svc.Show("s0001"); d.State != StateWaitingForInput {
		t.Errorf("after the refused idle handoff: %s", d.State)
	}
	if _, err := h.svc.Submit(ctx, SendRequest{Session: "s0001", As: "1001", Text: "amend"}); err != nil {
		t.Fatal(err)
	}
	waitFor(t, h.svc, "s0001", 0, isType(EventTurnEnded, 1))
	if h.clock.Pending() != 1 {
		t.Errorf("the activity did not re-arm the idle timer: %d pending", h.clock.Pending())
	}
}

// TestInterruptWhileACallWaits: the default until the owner decides: the call is denied with
// the reason `interrupted`, the turn ends without a resume, the queued message runs next.
func TestInterruptWhileACallWaits(t *testing.T) {
	h := waitingWithQueue(t, StartRequest{})
	e, err := h.svc.Interrupt(context.Background(), ActRequest{Session: "s0001", As: "1001"})
	if err != nil || e.Type != EventTurnEnded || e.Turn != 1 || e.Result != ResultInterrupted {
		t.Fatalf("interrupt: %+v, %v", e, err)
	}
	if n := count(t, h.st, `SELECT count(*) FROM permission_decision WHERE request_id = 'toolu_01NmkmkiJgpriuU7yQrtDWYi'
    AND decision = 'deny' AND decided_by = 'participant' AND user_id = '1001' AND reason = 'interrupted'`); n != 1 {
		t.Error("the pending call is not denied as interrupted")
	}
	// The queued message became turn 2, which replays the fixture's next process (turn 2's file).
	waitFor(t, h.svc, "s0001", 0, isType(EventTurnStarted, 2))
	if req := received(t, h.root, "s0001", 2); req.Answer != nil {
		t.Errorf("the interrupted call was resumed: %+v", req.Answer)
	}
}

// TestAnswerRefusals: an allow's reason reaches nobody; a resume that cannot start is reported.
func TestAnswerRefusals(t *testing.T) {
	h := newHarness(t, "", true)
	ctx := context.Background()
	dir := t.TempDir()
	b, err := os.ReadFile(engine.TurnFile(filepath.Join(fixtures, "defer"), 1))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(engine.TurnFile(dir, 1), b, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := h.svc.Start(ctx, StartRequest{Channel: "100", As: "1001", Mode: "prompt", Fixture: dir}); err != nil {
		t.Fatal(err)
	}
	sub, err := h.svc.Submit(ctx, SendRequest{Session: "s0001", As: "1001", Text: "write"})
	if err != nil {
		t.Fatal(err)
	}
	waitFor(t, h.svc, "s0001", sub.Seq, isType(EventWaitingForAnswer, 1))
	if _, err := h.svc.Answer(ctx, AnswerRequest{ActRequest: ActRequest{Session: "s0001", As: "1001"}, Decision: "allow", Reason: "why"}); code(err) != CodeBadArgs {
		t.Errorf("allow with a reason: %v", err)
	}
	if _, err := h.svc.Answer(ctx, AnswerRequest{ActRequest: ActRequest{Session: "s0001", As: "1001"}, Decision: "allow"}); code(err) != CodeResumeFailed {
		t.Errorf("a resume with no fixture turn: %v", err)
	}
}
