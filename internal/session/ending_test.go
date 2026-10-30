// SPDX-License-Identifier: GPL-3.0-or-later

package session

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jasondorjeshort/broodforge/internal/engine"
	"github.com/jasondorjeshort/broodforge/internal/git"
	"github.com/jasondorjeshort/broodforge/internal/git/gittest"
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
	if _, err := h.svc.end(ctx, l, ActorInstance, "", "idle", "x", gen); code(err) != CodeSessionClosed {
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

// heldHarness is a session s0001 on channel 100 that ran one turn, interrupted by a stop, then
// loaded by a new service on the same store as a restart does (ADR 0026): the canonical holds
// refs/sessions/s0001 and Config.Commits counts one commit. prepare runs between the stop and
// the load, on the store and the session's directory as the restart finds them. It returns the
// harness and the ids the load held.
func heldHarness(t *testing.T, prepare func(h *harness)) (*harness, []string) {
	t.Helper()
	h := newHarness(t, "", true)
	ctx := context.Background()
	reviewCanonical(t, h)
	if _, err := h.svc.Start(ctx, StartRequest{Channel: "100", As: "1001"}); err != nil {
		t.Fatal(err)
	}
	if _, err := h.svc.Submit(ctx, SendRequest{Session: "s0001", As: "1001", Text: "go"}); err != nil {
		t.Fatal(err)
	}
	waitTurn(t, h.svc, "s0001", 1)
	sctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if marked, err := h.svc.Shutdown(sctx); err != nil || len(marked) != 1 {
		t.Fatalf("stop: %v %v", marked, err)
	}
	if prepare != nil {
		prepare(h)
	}
	one := 1
	h.cfg.Commits = func(context.Context, string) (*int, error) { return &one, nil }
	var started []string
	h.cfg.Started = func(id string) { started = append(started, id) }
	h.svc = New(h.cfg)
	rows := count(t, h.st, `SELECT count(*) FROM audit_event`)
	held, err := h.svc.LoadInterrupted(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n := count(t, h.st, `SELECT count(*) FROM audit_event`); n != rows {
		t.Errorf("loading wrote %d audit rows", n-rows)
	}
	if strings.Join(started, " ") != strings.Join(held, " ") {
		t.Errorf("Started ran for %v, held %v", started, held)
	}
	return h, held
}

// endHeld ends s0001 as its driver and returns the outcome, the session-ended event and how
// many handoffs ran.
func endHeld(t *testing.T, h *harness) (git.Outcome, Event, int) {
	t.Helper()
	calls := 0
	h.mu.Lock()
	prev := h.handoff
	h.handoff = func(id string) (git.Outcome, error) {
		calls++
		if prev != nil {
			return prev(id)
		}
		return git.Outcome{Outcome: git.OutcomeNone}, nil
	}
	h.mu.Unlock()
	out, err := h.svc.End(context.Background(), EndRequest{ActRequest: ActRequest{Session: "s0001", As: "1001", Roles: "10"}})
	if err != nil {
		t.Fatalf("end: %v", err)
	}
	evs, _ := h.svc.Events("s0001", 0)
	last := evs[len(evs)-1]
	if last.Type != EventSessionEnded || last.End == nil {
		t.Fatalf("the last event: %+v", last)
	}
	if n := count(t, h.st, `SELECT count(*) FROM audit_event WHERE kind = 'session-ended' AND session_id = 's0001'
    AND details LIKE '%"reason":"end"%' AND details LIKE '%"state":"interrupted"%'`); n != 1 {
		t.Errorf("%d session-ended rows with state interrupted", n)
	}
	if d, _ := h.svc.Show("s0001"); d.State != StateEnded {
		t.Errorf("after end: %s", d.State)
	}
	return out, last, calls
}

func removeTree(t *testing.T, h *harness) {
	t.Helper()
	if err := os.RemoveAll(filepath.Join(h.root, "sessions", "s0001", "tree")); err != nil {
		t.Fatal(err)
	}
}

// TestHeld: after a restart an interrupted session is held (ADR 0026): its log opens with the
// restart and the header's facts, every action but end is refused session-closed, end hands it
// off and closes it, a refused handoff returns it to interrupted, and the two escapes end it
// without a handoff when its tree or its ref is gone.
func TestHeld(t *testing.T) {
	t.Run("held then end", func(t *testing.T) {
		h, held := heldHarness(t, nil)
		if strings.Join(held, " ") != "s0001" {
			t.Fatalf("held %v", held)
		}
		evs, err := h.svc.Events("s0001", 0)
		if err != nil || len(evs) != 1 {
			t.Fatalf("events %v %v", summaries(evs), err)
		}
		first := evs[0]
		if first.Type != EventSessionInterrupted || first.Summary != InterruptedRestart || first.Start == nil ||
			first.Start.Commits == nil || *first.Start.Commits != 1 || first.Start.TreeMissing ||
			first.Start.Branch != "broodforge/test/s0001" || first.Start.DriverName != "Ada Example" ||
			strings.Join(first.Start.Answer, ",") != "driver" || first.Start.CredentialKind != "api-key" {
			t.Fatalf("the opening event: %+v %+v", first, first.Start)
		}
		out, ended, calls := endHeld(t, h)
		if calls != 1 || out.Outcome != git.OutcomeNone || ended.Summary != "Session s0001 ended by Ada Example." ||
			ended.End.Turns != 1 || ended.End.Commits != 0 || ended.End.StartedAt == "" {
			t.Errorf("end: %d handoffs, %+v, %+v", calls, out, ended.End)
		}
	})

	t.Run("refused then interrupted", func(t *testing.T) {
		h, _ := heldHarness(t, nil)
		h.mu.Lock()
		h.handoff = func(string) (git.Outcome, error) {
			return git.Outcome{}, &git.Error{Code: git.CodeRefusedAttribution, Message: "1 commit lacks the required attribution."}
		}
		h.mu.Unlock()
		_, err := h.svc.End(context.Background(), EndRequest{ActRequest: ActRequest{Session: "s0001", As: "1001"}})
		var ge *git.Error
		if !errors.As(err, &ge) || ge.Code != git.CodeRefusedAttribution {
			t.Fatalf("end: %v", err)
		}
		if d, _ := h.svc.Show("s0001"); d.State != StateInterrupted {
			t.Errorf("after the refused handoff: %s", d.State)
		}
		if h.clock.Pending() != 0 {
			t.Errorf("%d timers armed on a held session", h.clock.Pending())
		}
		evs, _ := h.svc.Events("s0001", 0)
		last := evs[len(evs)-1]
		want := "1 commit lacks the required attribution. The session was interrupted, so no agent can amend them; " +
			"the owner can fix the tree and retry with broodforge worktree handoff s0001."
		if last.Type != EventHandoffRefused || last.Text != want {
			t.Errorf("the refusal's event: %+v", last)
		}
		if _, err := h.svc.Submit(context.Background(), SendRequest{Session: "s0001", As: "1001", Text: "amend"}); code(err) != CodeSessionClosed {
			t.Errorf("a message after the refusal: %v", err)
		}
		h.mu.Lock()
		h.handoff = nil
		h.mu.Unlock()
		if out, _, calls := endHeld(t, h); calls != 1 || out.Outcome != git.OutcomeNone {
			t.Errorf("the second end: %+v, %d handoffs", out, calls)
		}
	})

	t.Run("actions refused", func(t *testing.T) {
		h, _ := heldHarness(t, nil)
		ctx := context.Background()
		act := ActRequest{Session: "s0001", As: "1002", Roles: "10"} // not the driver: still session-closed
		closed := "session-closed: s0001 is interrupted"
		checks := map[string]error{}
		_, checks["send"] = h.svc.Submit(ctx, SendRequest{Session: "s0001", As: "1001", Text: "hi"})
		_, checks["mode"] = h.svc.SetMode(ModeRequest{ActRequest: act, Mode: "prompt"})
		_, checks["model"] = h.svc.SetModel(ModelRequest{ActRequest: act, Model: "sonnet"})
		_, checks["effort"] = h.svc.SetEffort(EffortRequest{ActRequest: act, Effort: "high"})
		_, checks["timeout"] = h.svc.SetIdleTimeout(TimeoutRequest{ActRequest: act, Timeout: "5"})
		_, checks["answer"] = h.svc.Answer(ctx, AnswerRequest{ActRequest: act, Decision: "allow"})
		_, checks["interrupt"] = h.svc.Interrupt(ctx, act)
		_, checks["invite"] = h.svc.Invite(InviteRequest{ActRequest: act, User: "1002"})
		for action, err := range checks {
			if err == nil || !strings.HasPrefix(err.Error(), closed) {
				t.Errorf("%s: %v, want %s", action, err, closed)
			}
		}
		if d, _ := h.svc.Show("s0001"); d.State != StateInterrupted {
			t.Errorf("after the refusals: %s", d.State)
		}
	})

	t.Run("tree gone, ref present", func(t *testing.T) {
		h, _ := heldHarness(t, func(h *harness) { removeTree(t, h) })
		evs, _ := h.svc.Events("s0001", 0)
		if st := evs[0].Start; st == nil || !st.TreeMissing || st.Commits != nil {
			t.Fatalf("the opening event's tree: %+v", st)
		}
		out, ended, calls := endHeld(t, h)
		if calls != 0 || out.Outcome != OutcomeTreeMissing || ended.End.Outcome != OutcomeTreeMissing || ended.End.Commits != CommitsUnknown {
			t.Errorf("end: %d handoffs, %+v, %+v", calls, out, ended.End)
		}
		if n := count(t, h.st, `SELECT count(*) FROM session WHERE id = 's0001' AND end_outcome = 'tree-missing'`); n != 1 {
			t.Error("the row's outcome is not tree-missing")
		}
		if gittest.Git(t, filepath.Join(h.root, "canonical"), "for-each-ref", git.SessionRef("s0001")) == "" {
			t.Error("the ref is gone: it is the owner's to recover")
		}
	})

	// A `worktree handoff` finished earlier removes the tree and the ref and records the branch
	// (or the pull request's URL) in end_outcome; a none handoff records nothing.
	handedOff := func(outcome string) func(h *harness) {
		return func(h *harness) {
			removeTree(t, h)
			gittest.Git(t, filepath.Join(h.root, "canonical"), "update-ref", "-d", git.SessionRef("s0001"))
			exec(t, h.st, `UPDATE worktree SET state = 'removed' WHERE session_id = 's0001'`)
			if outcome != "" {
				exec(t, h.st, `UPDATE session SET end_outcome = '`+outcome+`' WHERE id = 's0001'`)
			}
		}
	}
	t.Run("handed off by worktree handoff, then end", func(t *testing.T) {
		h, _ := heldHarness(t, handedOff("broodforge/test/s0001"))
		evs, _ := h.svc.Events("s0001", 0)
		if st := evs[0].Start; st == nil || st.TreeMissing || st.Commits != nil {
			t.Fatalf("the opening event's tree: %+v", st)
		}
		out, ended, calls := endHeld(t, h)
		if calls != 0 || out.Outcome != git.OutcomePushed || ended.End.Branch != "broodforge/test/s0001" ||
			ended.End.Commits != CommitsUnknown || ended.End.URL != "" {
			t.Errorf("end: %d handoffs, %+v, %+v", calls, out, ended.End)
		}
		if n := count(t, h.st, `SELECT count(*) FROM session WHERE id = 's0001' AND end_outcome = 'broodforge/test/s0001'`); n != 1 {
			t.Error("the row's recorded branch was replaced")
		}
	})
	t.Run("pull request opened earlier", func(t *testing.T) {
		url := "https://github.com/example/p/pull/7"
		h, _ := heldHarness(t, handedOff(url))
		out, ended, calls := endHeld(t, h)
		if calls != 0 || out.Outcome != git.OutcomePullRequest || ended.End.URL != url {
			t.Errorf("end: %d handoffs, %+v, %+v", calls, out, ended.End)
		}
	})
	t.Run("none handed off earlier", func(t *testing.T) {
		h, _ := heldHarness(t, handedOff(""))
		out, ended, calls := endHeld(t, h)
		if calls != 0 || out.Outcome != git.OutcomeNone || ended.End.Commits != CommitsUnknown {
			t.Errorf("end: %d handoffs, %+v, %+v", calls, out, ended.End)
		}
	})

	t.Run("unconfigured channel", func(t *testing.T) {
		h, _ := heldHarness(t, func(h *harness) {
			exec(t, h.st, `UPDATE session SET channel_id = '999' WHERE id = 's0001'`)
		})
		evs, _ := h.svc.Events("s0001", 0)
		if st := evs[0].Start; st == nil || st.Channel != "999" || strings.Join(st.Answer, ",") != "driver" {
			t.Fatalf("the opening event: %+v", st)
		}
		_, err := h.svc.End(context.Background(), EndRequest{ActRequest: ActRequest{Session: "s0001", As: "1002", Roles: "10"}})
		if code(err) != CodeRefused || !strings.Contains(err.Error(), "Only Ada Example (the driver) can end this.") {
			t.Errorf("end by another: %v", err)
		}
		if _, _, calls := endHeld(t, h); calls != 1 {
			t.Errorf("%d handoffs", calls)
		}
	})

	// A channel no longer configured, its tree gone and its ref present: the handoff, which reads
	// the project's canonical, would refuse the bundle of the missing tree on every attempt
	// (handoff-refused-bundle, as WorktreeService's does) and send the session back to
	// interrupted; the escape must end it tree-missing first, and the notice say so.
	refusingBundle := func(h *harness) {
		h.mu.Lock()
		h.handoff = func(string) (git.Outcome, error) {
			return git.Outcome{}, &git.Error{Code: git.CodeRefusedBundle, Message: "git rev-list: cannot change to the tree"}
		}
		h.mu.Unlock()
	}
	for _, c := range []struct{ name, project string }{
		{"unconfigured channel, tree gone", "p"}, // the project's repository: the ref is found
		{"unconfigured project, tree gone", "q"}, // no repository at all: the canonical cannot be read
	} {
		t.Run(c.name, func(t *testing.T) {
			h, _ := heldHarness(t, func(h *harness) {
				exec(t, h.st, `UPDATE session SET channel_id = '999', project_id = '`+c.project+`' WHERE id = 's0001'`)
				removeTree(t, h)
			})
			refusingBundle(h)
			evs, _ := h.svc.Events("s0001", 0)
			if st := evs[0].Start; st == nil || !st.TreeMissing || st.Commits != nil {
				t.Fatalf("the opening event's tree: %+v", st)
			}
			out, ended, calls := endHeld(t, h)
			if calls != 0 || out.Outcome != OutcomeTreeMissing || out.Remote != "origin" || ended.End.Commits != CommitsUnknown {
				t.Errorf("end: %d handoffs, %+v, %+v", calls, out, ended.End)
			}
		})
	}

	// The same channel gone, but handed off earlier: the project's canonical shows the ref gone,
	// so the recorded branch ends it pushed, not tree-missing.
	t.Run("unconfigured channel, handed off earlier", func(t *testing.T) {
		h, _ := heldHarness(t, func(h *harness) {
			handedOff("broodforge/test/s0001")(h)
			exec(t, h.st, `UPDATE session SET channel_id = '999' WHERE id = 's0001'`)
		})
		refusingBundle(h)
		out, ended, calls := endHeld(t, h)
		if calls != 0 || out.Outcome != git.OutcomePushed || out.Mode != git.ModePush || out.Remote != "origin" ||
			ended.End.Branch != "broodforge/test/s0001" {
			t.Errorf("end: %d handoffs, %+v, %+v", calls, out, ended.End)
		}
	})
}

// TestEndingEvent: before the handoff runs, end publishes session-ending with the tree's commit
// count (04: `Ending: checking and handing off 2 commits...`), unknown when it cannot be read;
// the summary's facts carry each commit's hash and subject, the pull request's title and its web
// URL (P11 remainder, items 1 and 2).
func TestEndingEvent(t *testing.T) {
	for _, c := range []struct {
		name    string
		commits func(context.Context, string) (*int, error)
		summary string
		want    *int
	}{
		{"counted", func(context.Context, string) (*int, error) { n := 2; return &n, nil }, "2 commits", ptr(2)},
		{"one", func(context.Context, string) (*int, error) { n := 1; return &n, nil }, "1 commit", ptr(1)},
		{"unreadable", func(context.Context, string) (*int, error) { return nil, errors.New("no tree") }, "commits unknown", nil},
		{"no counter", nil, "commits unknown", nil},
	} {
		t.Run(c.name, func(t *testing.T) {
			h := newHarness(t, "", true)
			h.cfg.Commits = c.commits
			h.svc = New(h.cfg)
			ctx := context.Background()
			if _, err := h.svc.Start(ctx, StartRequest{Channel: "100", As: "1001"}); err != nil {
				t.Fatal(err)
			}
			web := "https://github.com/o/r/pull/42"
			h.handoff = func(string) (git.Outcome, error) {
				return git.Outcome{Outcome: git.OutcomePullRequest, Mode: git.ModePullRequest, Commits: 2, URL: &web,
					Request: &git.PullRequest{URL: "https://api.github.com/repos/o/r/pulls", Body: git.PullRequestBody{Title: "Session s0001: fix it"}},
					Log:     []git.CommitLine{{Hash: "a1b2c3d4", Subject: "Add config reload"}, {Hash: "9f8e7d6c", Subject: "Docs"}}}, nil
			}
			if _, err := h.svc.End(ctx, EndRequest{ActRequest: ActRequest{Session: "s0001", As: "1001"}}); err != nil {
				t.Fatal(err)
			}
			evs, _ := h.svc.Events("s0001", 0)
			n := len(evs)
			if n < 2 || evs[n-2].Type != EventSessionEnding || evs[n-1].Type != EventSessionEnded {
				t.Fatalf("the last two events: %v", summaries(evs))
			}
			ending, end := evs[n-2], evs[n-1].End
			if ending.Summary != c.summary || (ending.Commits == nil) != (c.want == nil) || (c.want != nil && *ending.Commits != *c.want) {
				t.Errorf("session-ending %q %v, want %q %v", ending.Summary, ending.Commits, c.summary, c.want)
			}
			if end == nil || end.URL != web || end.Title != "Session s0001: fix it" || len(end.Log) != 2 ||
				end.Log[0] != (CommitLine{Hash: "a1b2c3d4", Subject: "Add config reload"}) {
				t.Errorf("end info %+v", end)
			}
		})
	}
}

func ptr(n int) *int { return &n }
