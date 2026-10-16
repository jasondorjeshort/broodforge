// SPDX-License-Identifier: GPL-3.0-or-later

package session

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jasondorjeshort/broodforge/internal/engine"
	"github.com/jasondorjeshort/broodforge/internal/git"
	"github.com/jasondorjeshort/broodforge/internal/store"
)

// P10 half (b): prompt mode against `defer` and `parallel`, the timeouts on the manual clock,
// interrupt, the usage records against the fixtures, mode, invite, end and reach-report.

// waitFor follows the session from seq until an event matches, and returns it.
func waitFor(t *testing.T, svc *Service, id string, seq int64, match func(Event) bool) Event {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	var got Event
	err := svc.Follow(ctx, id, seq, func(ev Event) (bool, error) {
		if match(ev) {
			got = ev
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		t.Fatalf("follow %s: %v", id, err)
	}
	return got
}

func isType(typ EventType, turn int) func(Event) bool {
	return func(ev Event) bool { return ev.Type == typ && (turn == 0 || ev.Turn == turn) }
}

func received(t *testing.T, root, id string, n int) engine.TurnRequest {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, "sessions", id, "fake", "received-"+itoa(n)+".json"))
	if err != nil {
		t.Fatal(err)
	}
	var req engine.TurnRequest
	if err := json.Unmarshal(b, &req); err != nil {
		t.Fatal(err)
	}
	return req
}

func hookVar(req engine.TurnRequest, name string) string {
	for _, v := range req.HookEnv {
		if v.Name == name {
			return v.Value
		}
	}
	return ""
}

// TestDeferAnswerAndPromptTimeout drives `defer`: turn 1 defers a Write and is answered
// allow by the driver; turn 2 defers again and the prompt timeout denies it.
func TestDeferAnswerAndPromptTimeout(t *testing.T) {
	h := newHarness(t, "", true)
	ctx := context.Background()
	if _, err := h.svc.Start(ctx, StartRequest{Channel: "100", As: "1001", Mode: "prompt", Fixture: "defer", PromptTimeoutSeconds: 300}); err != nil {
		t.Fatal(err)
	}
	sub, err := h.svc.Submit(ctx, SendRequest{Session: "s0001", As: "1001", Text: "write hi.txt"})
	if err != nil {
		t.Fatal(err)
	}
	w := waitFor(t, h.svc, "s0001", sub.Seq, isType(EventWaitingForAnswer, 1))
	if w.ToolUseID != "toolu_01NmkmkiJgpriuU7yQrtDWYi" || w.Tool != "Write" {
		t.Fatalf("waiting-for-answer %+v", w)
	}
	d, err := h.svc.Show("s0001")
	if err != nil || d.State != StateWaitingForAnswer || d.Pending == nil || d.Pending.ToolUseID != w.ToolUseID ||
		d.Pending.Until != "2026-10-16T12:05:00Z" {
		t.Fatalf("show while waiting: %+v %+v, %v", d, d.Pending, err)
	}
	if n := count(t, h.st, `SELECT count(*) FROM permission_decision WHERE request_id = '`+w.ToolUseID+`' AND decision = 'deferred'`); n != 1 {
		t.Error("no deferred row")
	}
	// Someone else's answer, checked against a driver-only answer list, is refused in 04's words.
	if _, err := h.svc.Answer(ctx, AnswerRequest{ActRequest: ActRequest{Session: "s0001", As: "1002", Roles: "10"}, Decision: "allow"}); code(err) != CodeRefused ||
		!strings.Contains(err.Error(), "Only Ada Example (the driver) can answer this.") {
		t.Errorf("answer by 1002: %v", err)
	}
	if _, err := h.svc.Answer(ctx, AnswerRequest{ActRequest: ActRequest{Session: "s0001", As: "1001"}, ToolUseID: "toolu_x", Decision: "allow"}); code(err) != CodeNotPending {
		t.Errorf("answer for another call: %v", err)
	}
	ans, err := h.svc.Answer(ctx, AnswerRequest{ActRequest: ActRequest{Session: "s0001", As: "1001", Roles: "10"}, Decision: "allow"})
	if err != nil || ans.State != SubmitResumed || ans.Turn != 1 {
		t.Fatalf("answer: %+v, %v", ans, err)
	}
	if e := waitFor(t, h.svc, "s0001", ans.Seq, isType(EventTurnEnded, 1)); e.Result != ResultCompleted {
		t.Errorf("turn 1 ended %+v", e)
	}
	if req := received(t, h.root, "s0001", 2); req.Answer == nil || req.Answer.Decision != "allow" || req.Input != "" ||
		req.EngineSessionID != "93ab3f0d-b9f2-40b7-8921-a95858da0251" {
		t.Errorf("resume request: %+v", req)
	}
	if n := count(t, h.st, `SELECT count(*) FROM permission_decision WHERE request_id = '`+w.ToolUseID+`'
    AND decision = 'allow' AND decided_by = 'participant' AND user_id = '1001'`); n != 1 {
		t.Error("the answered row is not allow by participant 1001")
	}

	sub, err = h.svc.Submit(ctx, SendRequest{Session: "s0001", As: "1001", Text: "write it again"})
	if err != nil {
		t.Fatal(err)
	}
	w = waitFor(t, h.svc, "s0001", sub.Seq, isType(EventWaitingForAnswer, 2))
	seq := waitSeq(h.svc, "s0001")
	h.clock.Advance(299 * time.Second)
	if d, _ := h.svc.Show("s0001"); d.Pending == nil {
		t.Fatal("the prompt timed out early")
	}
	h.clock.Advance(time.Second)
	if e := waitFor(t, h.svc, "s0001", seq, isType(EventTurnEnded, 2)); e.Result != ResultCompleted {
		t.Errorf("turn 2 ended %+v", e)
	}
	if req := received(t, h.root, "s0001", 4); req.Answer == nil || req.Answer.Decision != "deny" ||
		hookVar(req, envAnswerReason) != "the prompt timed out after 5 minutes" {
		t.Errorf("timeout resume request: %+v", req)
	}
	if n := count(t, h.st, `SELECT count(*) FROM audit_event WHERE kind = 'permission-decision' AND actor = 'instance'
    AND details LIKE '%"decidedBy":"prompt-timeout","decision":"deny"%'`); n != 1 {
		t.Error("the prompt timeout's deny is not in audit")
	}
	tail, _ := h.svc.Events("s0001", 0)
	found := false
	for _, ev := range tail {
		if ev.Type == EventToolResult && ev.Summary == "[Write] /home/user/bf-spike/tree/bye.txt -> denied: the prompt timed out after 5 minutes." {
			found = true
		}
	}
	if !found {
		t.Errorf("no timeout trace line in %v", summaries(tail))
	}
}

func waitSeq(svc *Service, id string) int64 {
	evs, _ := svc.Events(id, 0)
	return int64(len(evs))
}

func summaries(evs []Event) []string {
	var out []string
	for _, ev := range evs {
		out = append(out, string(ev.Type)+": "+ev.Summary)
	}
	return out
}

// TestParallelDropsOne drives `parallel` (C28): two Writes in one message, the result names
// the last; the first gets one dropped row. After the answer the re-issued call defers again.
func TestParallelDropsOne(t *testing.T) {
	h := newHarness(t, "", true)
	ctx := context.Background()
	if _, err := h.svc.Start(ctx, StartRequest{Channel: "100", As: "1001", Mode: "prompt", Fixture: "parallel"}); err != nil {
		t.Fatal(err)
	}
	sub, err := h.svc.Submit(ctx, SendRequest{Session: "s0001", As: "1001", Text: "write both"})
	if err != nil {
		t.Fatal(err)
	}
	w := waitFor(t, h.svc, "s0001", sub.Seq, isType(EventWaitingForAnswer, 1))
	if w.ToolUseID != "toolu_01FfytaZavRivXiQSPniauVu" {
		t.Fatalf("waiting on %s", w.ToolUseID)
	}
	if n := count(t, h.st, `SELECT count(*) FROM permission_decision WHERE decision = 'dropped' AND request_id = 'toolu_01B5b1rkmDCfPP4aQPdM3VCT'`); n != 1 {
		t.Error("no dropped row for the first call")
	}
	if n := count(t, h.st, `SELECT count(*) FROM audit_event WHERE kind = 'permission-decision' AND details LIKE '%"decision":"dropped"%'`); n != 1 {
		t.Errorf("%d dropped events in audit, want 1", n)
	}
	if d, _ := h.svc.Show("s0001"); d.Pending == nil || d.Pending.Dropped != 1 {
		t.Errorf("pending %+v", d.Pending)
	}
	ans, err := h.svc.Answer(ctx, AnswerRequest{ActRequest: ActRequest{Session: "s0001", As: "1001"}, Decision: "allow-session"})
	if err != nil {
		t.Fatal(err)
	}
	w = waitFor(t, h.svc, "s0001", ans.Seq, isType(EventWaitingForAnswer, 1))
	if w.ToolUseID != "toolu_01DSwcaGCZMPUQgCp2cERJcp" {
		t.Errorf("re-issued call %s", w.ToolUseID)
	}
	// allow-session holds from the resume on: the dropped call comes back in that process.
	if req := received(t, h.root, "s0001", 2); hookVar(req, envSessionAllow) != "Write" {
		t.Errorf("the resume lacks BROODFORGE_SESSION_ALLOW=Write: %v", req.HookEnv)
	}
}

// TestIdleTimeoutEndsTheSession: no activity for the idle timeout runs the end of session.
func TestIdleTimeoutEndsTheSession(t *testing.T) {
	h := newHarness(t, "", true)
	ctx := context.Background()
	if _, err := h.svc.Start(ctx, StartRequest{Channel: "100", As: "1001", IdleTimeoutSeconds: 3600}); err != nil {
		t.Fatal(err)
	}
	called := ""
	h.mu.Lock()
	h.handoff = func(id string) (git.Outcome, error) { called = id; return git.Outcome{Outcome: git.OutcomeNone}, nil }
	h.mu.Unlock()
	h.clock.Advance(3599 * time.Second)
	if d, _ := h.svc.Show("s0001"); d.State != StateWaitingForInput {
		t.Fatalf("ended early: %s", d.State)
	}
	h.clock.Advance(time.Second)
	e := waitFor(t, h.svc, "s0001", 0, isType(EventSessionEnded, 0))
	if e.Summary != "Session s0001 ended: no activity for 60 minutes." || called != "s0001" {
		t.Errorf("ended %+v, handoff %q", e, called)
	}
	if n := count(t, h.st, `SELECT count(*) FROM session WHERE id = 's0001' AND state = 'ended' AND end_outcome = 'none'`); n != 1 {
		t.Error("session not ended with outcome none")
	}
	if n := count(t, h.st, `SELECT count(*) FROM audit_event WHERE kind = 'session-ended' AND actor = 'instance' AND details LIKE '%"reason":"idle"%'`); n != 1 {
		t.Error("no session-ended event for idle")
	}
	if _, err := h.svc.Submit(ctx, SendRequest{Session: "s0001", As: "1001", Text: "late"}); code(err) != CodeSessionClosed {
		t.Errorf("send to an ended session: %v", err)
	}
}

// TestInterruptRunningTurn stops a slow turn; it ends interrupted and the session waits again.
func TestInterruptRunningTurn(t *testing.T) {
	h := newHarness(t, delayed(t, 2*time.Second), true)
	ctx := context.Background()
	if _, err := h.svc.Start(ctx, StartRequest{Channel: "100", As: "1001"}); err != nil {
		t.Fatal(err)
	}
	if _, err := h.svc.Interrupt(ctx, ActRequest{Session: "s0001", As: "1001"}); code(err) != CodeNoRunning {
		t.Errorf("interrupt with nothing running: %v", err)
	}
	if _, err := h.svc.Submit(ctx, SendRequest{Session: "s0001", As: "1001", Text: "slow"}); err != nil {
		t.Fatal(err)
	}
	begun := time.Now()
	e, err := h.svc.Interrupt(ctx, ActRequest{Session: "s0001", As: "1001"})
	if err != nil || e.Result != ResultInterrupted || e.User != "1001" {
		t.Fatalf("interrupt: %+v, %v", e, err)
	}
	if time.Since(begun) > 1500*time.Millisecond {
		t.Error("interrupt waited out the fixture instead of stopping it")
	}
	if d, _ := h.svc.Show("s0001"); d.State != StateWaitingForInput {
		t.Errorf("after interrupt: %s", d.State)
	}
}

// expectedUsage replays a fixture's turns through engine.UsageOf with S2's baselines (per
// engine session, never an interrupted turn) and sums them per session turn and model:
// groups[i] is how many fixture turns session turn i+1 took (a deferral and its resume are one).
func expectedUsage(t *testing.T, fixture string, groups []int) map[string][4]int64 {
	t.Helper()
	dir := filepath.Join(fixtures, fixture)
	fx, err := engine.ReadFixture(dir)
	if err != nil {
		t.Fatal(err)
	}
	baselines := map[string]map[string]engine.ModelUsage{}
	want := map[string][4]int64{}
	n := 0
	for turn, size := range groups {
		for i := 0; i < size; i++ {
			ft := fx.Turns[n]
			n++
			data, err := os.ReadFile(engine.TurnFile(dir, ft.N))
			if err != nil {
				t.Fatal(err)
			}
			var evs []engine.EngineEvent
			for _, line := range strings.Split(string(data), "\n") {
				evs = append(evs, engine.ParseLine([]byte(line))...)
			}
			u := engine.UsageOf(evs, baselines[ft.EngineSessionID])
			if u.Cumulative != nil {
				baselines[ft.EngineSessionID] = u.Cumulative
			}
			for model, d := range u.Delta {
				k := itoa(turn+1) + " " + model
				w := want[k]
				w[0] += int64(d["inputTokens"])
				w[1] += int64(d["outputTokens"])
				w[2] += int64(d["cacheReadInputTokens"])
				w[3] += int64(d["cacheCreationInputTokens"])
				want[k] = w
			}
		}
	}
	return want
}

func storedUsage(t *testing.T, st *store.Store, id string) map[string][4]int64 {
	t.Helper()
	rows, err := store.UsageByTurn(st, store.UsageFilter{Session: id})
	if err != nil {
		t.Fatal(err)
	}
	got := map[string][4]int64{}
	for _, r := range rows {
		got[itoa(int(r.Turn))+" "+r.Model] = [4]int64{r.Input, r.Output, r.CacheRead, r.CacheWrite}
	}
	return got
}

// TestUsageEqualsFixtureDeltas: the records `usage --by turn` reads equal the fixtures' deltas,
// for a resumed session (basic), a session with an interrupted turn (interrupted, S2) and
// prompt-mode turns spanning two processes each (defer).
func TestUsageEqualsFixtureDeltas(t *testing.T) {
	for _, c := range []struct {
		fixture, mode string
		groups        []int
	}{
		{"basic", "policy", []int{1, 1, 1}},
		{"interrupted", "policy", []int{1, 1, 1}},
		{"defer", "prompt", []int{2, 2}},
	} {
		h := newHarness(t, "", true)
		ctx := context.Background()
		if _, err := h.svc.Start(ctx, StartRequest{Channel: "100", As: "1001", Mode: c.mode, Fixture: c.fixture}); err != nil {
			t.Fatal(err)
		}
		for turn := 1; turn <= len(c.groups); turn++ {
			sub, err := h.svc.Submit(ctx, SendRequest{Session: "s0001", As: "1001", Text: "turn " + itoa(turn)})
			if err != nil {
				t.Fatal(err)
			}
			for p := 1; p < c.groups[turn-1]; p++ {
				waitFor(t, h.svc, "s0001", sub.Seq, isType(EventWaitingForAnswer, turn))
				ans, err := h.svc.Answer(ctx, AnswerRequest{ActRequest: ActRequest{Session: "s0001", As: "1001"}, Decision: "allow"})
				if err != nil {
					t.Fatal(err)
				}
				sub.Seq = ans.Seq
			}
			waitFor(t, h.svc, "s0001", sub.Seq, isType(EventTurnEnded, turn))
		}
		want, got := expectedUsage(t, c.fixture, c.groups), storedUsage(t, h.st, "s0001")
		if len(want) == 0 || len(got) != len(want) {
			t.Errorf("%s: stored %v, want %v", c.fixture, got, want)
		}
		for k, w := range want {
			if got[k] != w {
				t.Errorf("%s turn %s: stored %v, want %v", c.fixture, k, got[k], w)
			}
		}
		if c.fixture == "interrupted" {
			if n := count(t, h.st, `SELECT count(*) FROM usage_record WHERE turn_seq = 2 AND interrupted = 1 AND output_tokens = 0`); n != 1 {
				t.Error("turn 2 is not recorded interrupted without output")
			}
		}
	}
}

func TestModeInviteEndReach(t *testing.T) {
	h := newHarness(t, "", true)
	ctx := context.Background()
	if _, err := h.svc.Start(ctx, StartRequest{Channel: "100", As: "1001"}); err != nil {
		t.Fatal(err)
	}
	if _, err := h.svc.SetMode(ModeRequest{ActRequest: ActRequest{Session: "s0001", As: "1002"}, Mode: "prompt"}); code(err) != CodeRefused {
		t.Errorf("mode by a non-driver: %v", err)
	}
	e, err := h.svc.SetMode(ModeRequest{ActRequest: ActRequest{Session: "s0001", As: "1001"}, Mode: "prompt"})
	if err != nil || e.Summary != "From the next turn: asks first." {
		t.Fatalf("mode: %+v, %v", e, err)
	}
	if _, err := h.svc.Submit(ctx, SendRequest{Session: "s0001", As: "1001", Text: "go"}); err != nil {
		t.Fatal(err)
	}
	waitFor(t, h.svc, "s0001", 0, isType(EventTurnEnded, 1))
	if req := received(t, h.root, "s0001", 1); hookVar(req, envMode) != "prompt" {
		t.Errorf("the next turn's mode is %q", hookVar(req, envMode))
	}
	if _, err := h.svc.Invite(InviteRequest{ActRequest: ActRequest{Session: "s0001", As: "1001"}, User: "1002"}); err != nil {
		t.Fatal(err)
	}
	if n := count(t, h.st, `SELECT count(*) FROM participant WHERE user_id = '1002'`); n != 0 {
		t.Error("an invitation made a participant: a co-author line would be owed for no message")
	}
	if n := count(t, h.st, `SELECT count(*) FROM audit_event WHERE kind = 'participant-invited' AND registration_id = '1002'`); n != 1 {
		t.Error("no participant-invited event")
	}
	rep, err := Reach(h.st, "s0001")
	if err != nil || rep.Reach != "code" || len(rep.Binds) != 0 || rep.Policy[0].InTree != "allow" || rep.Policy[3].InTree != "ask" ||
		rep.Policy[3].Elsewhere != "deny" || rep.Policy[5].Tools[0] != "*" {
		t.Errorf("reach report %+v, %v", rep, err)
	}

	// A refused handoff keeps the session open; the next end hands off.
	h.mu.Lock()
	h.handoff = func(string) (git.Outcome, error) {
		return git.Outcome{}, &git.Error{Code: "handoff-refused-attribution", Message: "1 of 1 commits fail the attribution check"}
	}
	h.mu.Unlock()
	if _, err := h.svc.End(ctx, EndRequest{ActRequest: ActRequest{Session: "s0001", As: "1001"}}); err == nil {
		t.Fatal("a refused handoff ended the session")
	}
	if d, _ := h.svc.Show("s0001"); d.State != StateWaitingForInput {
		t.Errorf("after a refused handoff: %s", d.State)
	}
	h.mu.Lock()
	h.handoff = func(string) (git.Outcome, error) { return git.Outcome{Outcome: git.OutcomePushed}, nil }
	h.mu.Unlock()
	out, err := h.svc.End(ctx, EndRequest{ActRequest: ActRequest{Session: "s0001", As: "1001"}})
	if err != nil || out.Outcome != git.OutcomePushed {
		t.Fatalf("end: %+v, %v", out, err)
	}
	e = waitFor(t, h.svc, "s0001", 0, isType(EventSessionEnded, 0))
	if e.Summary != "Session s0001 ended by Ada Example." {
		t.Errorf("ended %+v", e)
	}
	if _, err := h.svc.End(ctx, EndRequest{ActRequest: ActRequest{Session: "s0001"}}); code(err) != CodeSessionClosed {
		t.Errorf("a second end: %v", err)
	}
}

func TestHumanDuration(t *testing.T) {
	for d, want := range map[time.Duration]string{5 * time.Minute: "5 minutes", time.Minute: "1 minute",
		2 * time.Second: "2 seconds", 90 * time.Second: "90 seconds", time.Second: "1 second"} {
		if got := humanDuration(d); got != want {
			t.Errorf("%v: %q, want %q", d, got, want)
		}
	}
}

// TestTwoSessionsOnOneFixture: two sessions replaying one fixture share its engine session
// ids; their usage records must not collide (harness_ref carries the session id).
func TestTwoSessionsOnOneFixture(t *testing.T) {
	h := newHarness(t, "", true)
	ctx := context.Background()
	for _, id := range []string{"s0001", "s0002"} {
		if _, err := h.svc.Start(ctx, StartRequest{Channel: "100", As: "1001", Fixture: "basic"}); err != nil {
			t.Fatal(err)
		}
		if _, err := h.svc.Submit(ctx, SendRequest{Session: id, As: "1001", Text: "go"}); err != nil {
			t.Fatal(err)
		}
		if e := waitFor(t, h.svc, id, 0, isType(EventTurnEnded, 1)); e.Result != ResultCompleted || e.Summary != "end_turn" {
			t.Errorf("%s turn 1 ended %+v", id, e)
		}
	}
	if n := count(t, h.st, "SELECT count(DISTINCT session_id) FROM usage_record"); n != 2 {
		t.Errorf("usage records for %d sessions, want 2", n)
	}
}
