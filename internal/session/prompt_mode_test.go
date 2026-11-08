// SPDX-License-Identifier: GPL-3.0-or-later

package session

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
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

// P40d: a subagent's live request in a prompt turn (ADR 0029 rules 4 to 8 as amended
// 2026-11-07), each prompt-* fixture driven through the service.

// promptSession starts a prompt-mode session on channel 100 replaying fixture and sends one
// message.
func promptSession(t *testing.T, h *harness, fixture string) (string, Submitted) {
	t.Helper()
	ctx := context.Background()
	st, err := h.svc.Start(ctx, StartRequest{Channel: "100", As: "1001", Mode: "prompt", Fixture: fixture})
	if err != nil {
		t.Fatal(err)
	}
	sub, err := h.svc.Submit(ctx, SendRequest{Session: st.ID, As: "1001", Text: "delegate"})
	if err != nil {
		t.Fatal(err)
	}
	return st.ID, sub
}

// receivedAnswers is received-<n>.json's answers: the lines the fake got through Answer.
func receivedAnswers(t *testing.T, root, id string, n int) []string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, "sessions", id, "fake", "received-"+itoa(n)+".json"))
	if err != nil {
		t.Fatal(err)
	}
	var rec struct {
		Answers []json.RawMessage `json:"answers"`
	}
	if err := json.Unmarshal(b, &rec); err != nil {
		t.Fatal(err)
	}
	out := make([]string, len(rec.Answers))
	for i, a := range rec.Answers {
		var b bytes.Buffer
		if err := json.Compact(&b, a); err != nil {
			t.Fatal(err)
		}
		out[i] = b.String()
	}
	return out
}

func answer(t *testing.T, h *harness, id, decision, toolUseID string) Submitted {
	t.Helper()
	sub, err := h.svc.Answer(context.Background(), AnswerRequest{ActRequest: ActRequest{Session: id, As: "1001"}, ToolUseID: toolUseID, Decision: decision})
	if err != nil {
		t.Fatalf("answer %s %s: %v", decision, toolUseID, err)
	}
	return sub
}

func rowCount(t *testing.T, h *harness, id, toolUseID, where string) int64 {
	t.Helper()
	return count(t, h.st, `SELECT count(*) FROM permission_decision WHERE session_id = '`+id+`' AND request_id = '`+toolUseID+`' AND `+where)
}

// TestSubagentPromptAllow is prompt-ask-allow: the subagent's Bash waits in place, its prompt
// indented under the Agent call with the agent type; the driver's allow answers the live
// request, the turn completes, and the result's user_temporary source takes the record's
// decider (no hook-bypassed row).
func TestSubagentPromptAllow(t *testing.T) {
	h := newHarness(t, "", true)
	id, sub := promptSession(t, h, "prompt-ask-allow")
	const bash, agent = "toolu_01GgR6UWwxWCU6FVfmQSDfqe", "toolu_01Gpstfx3apMMJNRZ5MhKHq5"
	w := waitFor(t, h.svc, id, sub.Seq, isType(EventWaitingForAnswer, 1))
	if w.ToolUseID != bash || w.Agent != "probe" || w.Parent != agent || w.Depth != 1 || !strings.HasPrefix(w.Summary, "  "+bash) {
		t.Fatalf("waiting-for-answer %+v", w)
	}
	d, err := h.svc.Show(id)
	if err != nil || d.State != StateWaitingForAnswer || d.Pending != nil || len(d.Requests) != 1 ||
		d.Requests[0].ToolUseID != bash || d.Requests[0].Agent != "probe" || d.Requests[0].Until != "2026-10-16T13:00:00Z" {
		t.Fatalf("show while waiting: %+v %+v, %v", d, d.Requests, err)
	}
	if req := received(t, h.root, id, 1); req.Mode != "prompt" {
		t.Errorf("received-1.json mode %q, want prompt", req.Mode)
	}
	if n := rowCount(t, h, id, bash, "decision = 'deferred' AND decided_by = 'policy'"); n != 1 {
		t.Error("no deferred row")
	}
	ans := answer(t, h, id, AnswerAllow, "")
	if e := waitFor(t, h.svc, id, ans.Seq, isType(EventTurnEnded, 1)); e.Result != ResultCompleted {
		t.Errorf("turn 1 ended %+v", e)
	}
	if n := rowCount(t, h, id, bash, "decision = 'allow' AND decided_by = 'participant' AND user_id = '1001'"); n != 1 {
		t.Error("the answered row is not allow by participant 1001")
	}
	if a := receivedAnswers(t, h.root, id, 1); len(a) != 1 || !strings.Contains(a[0], `"behavior":"allow"`) ||
		!strings.Contains(a[0], `"request_id":"ac51e519-431c-4ad3-a5ff-6e3a99987959"`) {
		t.Errorf("answers %q", a)
	}
	if n := count(t, h.st, `SELECT count(*) FROM audit_event WHERE kind IN ('hook-bypassed', 'engine-prompt-tool-missing')`); n != 0 {
		t.Errorf("%d hook-bypassed or engine-prompt-tool-missing rows, want none", n)
	}
	if d, _ := h.svc.Show(id); d.State != StateWaitingForInput || len(d.Requests) != 0 {
		t.Errorf("after the turn: %s, %d requests", d.State, len(d.Requests))
	}
}

// TestSubagentPromptDenyThenAllow is prompt-ask-deny: the first request denied (interrupt
// false, 04's text), the second allowed, each by its tool use id.
func TestSubagentPromptDenyThenAllow(t *testing.T) {
	h := newHarness(t, "", true)
	id, sub := promptSession(t, h, "prompt-ask-deny")
	const first, second = "toolu_01MfdrJthAZWsma6Syx73Pqc", "toolu_01FVXoecWd9U1dadJXjTgAWu"
	if w := waitFor(t, h.svc, id, sub.Seq, isType(EventWaitingForAnswer, 1)); w.ToolUseID != first {
		t.Fatalf("first waiting-for-answer %+v", w)
	}
	ans := answer(t, h, id, AnswerDeny, first)
	if w := waitFor(t, h.svc, id, ans.Seq, isType(EventWaitingForAnswer, 1)); w.ToolUseID != second {
		t.Fatalf("second waiting-for-answer %+v", w)
	}
	ans = answer(t, h, id, AnswerAllow, second)
	if e := waitFor(t, h.svc, id, ans.Seq, isType(EventTurnEnded, 1)); e.Result != ResultCompleted {
		t.Errorf("turn 1 ended %+v", e)
	}
	if rowCount(t, h, id, first, "decision = 'deny' AND decided_by = 'participant'") != 1 ||
		rowCount(t, h, id, second, "decision = 'allow' AND decided_by = 'participant'") != 1 {
		t.Error("the rows are not deny then allow by participant")
	}
	a := receivedAnswers(t, h.root, id, 1)
	if len(a) != 2 || !strings.Contains(a[0], `"behavior":"deny","message":"Denied by Broodforge: a person in this session denied this call","interrupt":false`) ||
		!strings.Contains(a[1], `"behavior":"allow"`) {
		t.Errorf("answers %q", a)
	}
	evs, _ := h.svc.Events(id, 0)
	found := false
	for _, ev := range evs {
		found = found || ev.Type == EventToolResult && ev.ToolUseID == first && strings.HasSuffix(ev.Summary, "-> denied by Ada Example")
	}
	if !found {
		t.Errorf("no denied trace line for the first call:\n%s", strings.Join(summaries(evs), "\n"))
	}
}

// TestSubagentPromptSessionAllow is prompt-ask-session-allow: Allow Bash this session on the
// first request; the second is answered allow at once, no prompt, decider participant with
// reason session-allow.
func TestSubagentPromptSessionAllow(t *testing.T) {
	h := newHarness(t, "", true)
	id, sub := promptSession(t, h, "prompt-ask-session-allow")
	const first, second = "toolu_01MfdrJthAZWsma6Syx73Pqc", "toolu_01FVXoecWd9U1dadJXjTgAWu"
	waitFor(t, h.svc, id, sub.Seq, isType(EventWaitingForAnswer, 1))
	ans := answer(t, h, id, AnswerAllowSession, first)
	if e := waitFor(t, h.svc, id, ans.Seq, isType(EventTurnEnded, 1)); e.Result != ResultCompleted {
		t.Errorf("turn 1 ended %+v", e)
	}
	if rowCount(t, h, id, second, "decision = 'allow' AND decided_by = 'participant' AND reason = 'session-allow' AND user_id IS NULL") != 1 {
		t.Error("the second call is not allowed at once by session-allow")
	}
	evs, _ := h.svc.Events(id, 0)
	waits := 0
	for _, ev := range evs {
		if ev.Type == EventWaitingForAnswer {
			waits++
		}
	}
	if waits != 1 {
		t.Errorf("%d prompts, want 1", waits)
	}
	if a := receivedAnswers(t, h.root, id, 1); len(a) != 2 || !strings.Contains(a[1], `"behavior":"allow"`) {
		t.Errorf("answers %q", a)
	}
}

// twoAtOnce replays prompt-ask-two's turn without the fake's wait at each request: both
// requests are open together until both are answered (C41b), then the rest streams.
type twoAtOnce struct {
	mu      sync.Mutex
	answers []string
	both    chan struct{}
}

func (e *twoAtOnce) Name() string { return "fake" }

func (e *twoAtOnce) StartTurn(engine.TurnRequest) (engine.TurnHandle, error) {
	data, err := os.ReadFile(engine.TurnFile(filepath.Join(fixtures, "prompt-ask-two"), 1))
	if err != nil {
		return engine.TurnHandle{}, err
	}
	events := make(chan engine.EngineEvent)
	go func() {
		defer close(events)
		waited := false
		for _, line := range strings.Split(string(data), "\n") {
			for _, ev := range engine.ParseLine([]byte(line)) {
				if ev.Kind == engine.KindToolResult && !waited {
					<-e.both
					waited = true
				}
				events <- ev
			}
		}
		code := 0
		events <- engine.EngineEvent{Kind: engine.KindExit, Exit: &engine.ExitEvent{Code: &code}}
	}()
	return engine.TurnHandle{Events: events, Interrupt: func() error { return nil },
		Answer: func(requestID string, allow bool, message string, interrupt bool) error {
			e.mu.Lock()
			defer e.mu.Unlock()
			e.answers = append(e.answers, requestID)
			if len(e.answers) == 2 {
				close(e.both)
			}
			return nil
		}}, nil
}

// TestSubagentPromptTwoOpen: two requests open at once; an answer naming neither is bad-args
// naming both, an unknown id not-pending; answered out of order by --tool-use-id.
func TestSubagentPromptTwoOpen(t *testing.T) {
	h := newHarness(t, "", true)
	eng := &twoAtOnce{both: make(chan struct{})}
	h.cfg.Engine = func(string, string, string) (engine.Engine, error) { return eng, nil }
	h.svc = New(h.cfg)
	id, sub := promptSession(t, h, "basic")
	const alpha, beta = "toolu_018uaKNYzw3pU4MMDYJq1Mhh", "toolu_0142ACPeh2HeMXsQwZ2wcynU"
	waitFor(t, h.svc, id, sub.Seq, func(ev Event) bool { return ev.Type == EventWaitingForAnswer && ev.ToolUseID == beta })
	d, _ := h.svc.Show(id)
	if len(d.Requests) != 2 || d.Requests[0].ToolUseID != alpha || d.Requests[1].ToolUseID != beta {
		t.Fatalf("requests %+v", d.Requests)
	}
	ctx := context.Background()
	_, err := h.svc.Answer(ctx, AnswerRequest{ActRequest: ActRequest{Session: id, As: "1001"}, Decision: AnswerAllow})
	if code(err) != CodeBadArgs || !strings.Contains(err.Error(), alpha) || !strings.Contains(err.Error(), beta) {
		t.Errorf("answer without --tool-use-id: %v", err)
	}
	if _, err := h.svc.Answer(ctx, AnswerRequest{ActRequest: ActRequest{Session: id, As: "1001"}, ToolUseID: "toolu_x", Decision: AnswerAllow}); code(err) != CodeNotPending {
		t.Errorf("answer for another call: %v", err)
	}
	answer(t, h, id, AnswerAllow, beta)
	if d, _ := h.svc.Show(id); d.State != StateWaitingForAnswer || len(d.Requests) != 1 {
		t.Errorf("after one answer: %s, %d requests", d.State, len(d.Requests))
	}
	ans := answer(t, h, id, AnswerDeny, alpha)
	if e := waitFor(t, h.svc, id, ans.Seq, isType(EventTurnEnded, 1)); e.Result != ResultCompleted {
		t.Errorf("turn 1 ended %+v", e)
	}
	if strings.Join(eng.answers, ",") != "83db9cdd-ea79-41e0-abdf-4189d5bd06ad,b892d423-55ac-46b8-b467-e692f2681dbe" {
		t.Errorf("answered %q, want beta then alpha", eng.answers)
	}
	if rowCount(t, h, id, alpha, "decision = 'deny' AND decided_by = 'participant'") != 1 ||
		rowCount(t, h, id, beta, "decision = 'allow' AND decided_by = 'participant'") != 1 {
		t.Error("the rows are not alpha deny, beta allow")
	}
}

// TestSubagentPromptDropAtFirstResult is prompt-ask-background: the request open at the
// coordinator's first result is dropped (decider instance, the turn ended); the recording's
// allow after it, its second init and second result are logged and never classified: one
// turn-ended with the first result's usage, no row for the second turn's call.
func TestSubagentPromptDropAtFirstResult(t *testing.T) {
	h := newHarness(t, "", true)
	id, _ := promptSession(t, h, "prompt-ask-background")
	const bash = "toolu_01SJPk6DxJLkMgsps4sriE7d"
	evs := waitTurn(t, h.svc, id, 1)
	ended := 0
	var drop, trace bool
	for _, ev := range evs {
		switch {
		case ev.Type == EventTurnEnded:
			ended++
			if ev.Result != ResultCompleted {
				t.Errorf("turn-ended %+v", ev)
			}
		case ev.Type == EventPermissionDecision && ev.ToolUseID == bash && ev.Decision == DecisionDropped:
			drop = ev.DecidedBy == DecidedByInstance && ev.Reason == causeTurnEnded && strings.HasSuffix(ev.Summary, "(instance)")
		case ev.Type == EventToolResult && ev.ToolUseID == bash:
			trace = strings.HasSuffix(ev.Summary, "-> "+traceDroppedTurnEnded)
		}
	}
	if ended != 1 || !drop || !trace {
		t.Errorf("ended %d, drop %v, trace %v:\n%s", ended, drop, trace, strings.Join(summaries(evs), "\n"))
	}
	if n := rowCount(t, h, id, bash, "1"); n != 1 || rowCount(t, h, id, bash, "decision = 'dropped' AND decided_by = 'instance'") != 1 {
		t.Errorf("%d rows for the dropped call, want its one dropped row", n)
	}
	// The first result's usage, not the second's.
	data, err := os.ReadFile(engine.TurnFile(filepath.Join(fixtures, "prompt-ask-background"), 1))
	if err != nil {
		t.Fatal(err)
	}
	var first *engine.ResultEvent
	for _, line := range strings.Split(string(data), "\n") {
		for _, ev := range engine.ParseLine([]byte(line)) {
			if ev.Kind == engine.KindResult && first == nil {
				first = ev.Result
			}
		}
	}
	var want int64
	for _, d := range engine.UsageDelta(first.ModelUsage, nil) {
		want += int64(d["outputTokens"])
	}
	if got := count(t, h.st, `SELECT coalesce(sum(output_tokens), 0) FROM usage_record WHERE session_id = '`+id+`'`); got != want || want == 0 {
		t.Errorf("output tokens %d, want the first result's %d", got, want)
	}
}

// TestSubagentPromptStop is prompt-ask-interrupt: Stop with a request open answers it deny with
// interrupt true, records deny by the participant with reason interrupted, interrupts the
// process at once and ends the turn interrupted on its exit. End on another session drops its
// request the same way, decider instance.
func TestSubagentPromptStop(t *testing.T) {
	h := newHarness(t, "", true)
	ctx := context.Background()
	id, sub := promptSession(t, h, "prompt-ask-interrupt")
	const bash = "toolu_01TtW1H5QRaiYYDcAhv3RS5i"
	waitFor(t, h.svc, id, sub.Seq, isType(EventWaitingForAnswer, 1))
	ended, err := h.svc.Interrupt(ctx, ActRequest{Session: id, As: "1001"})
	if err != nil || ended.Type != EventTurnEnded || ended.Turn != 1 || ended.Result != ResultInterrupted {
		t.Fatalf("interrupt: %+v, %v", ended, err)
	}
	if rowCount(t, h, id, bash, "decision = 'deny' AND decided_by = 'participant' AND reason = 'interrupted' AND user_id = '1001'") != 1 {
		t.Error("the request is not deny (participant) interrupted")
	}
	if a := receivedAnswers(t, h.root, id, 1); len(a) != 1 ||
		!strings.Contains(a[0], `"behavior":"deny","message":"`+stopDenyMessage+`","interrupt":true`) {
		t.Errorf("answers %q", a)
	}
	// The turn ended on the exit: nothing runs, and the next message starts turn 2.
	if d, _ := h.svc.Show(id); d.State != StateWaitingForInput || len(d.Requests) != 0 {
		t.Errorf("after Stop: %s, %d requests", d.State, len(d.Requests))
	}

	id2, sub2 := promptSession(t, h, "prompt-ask-allow")
	waitFor(t, h.svc, id2, sub2.Seq, isType(EventWaitingForAnswer, 1))
	if _, err := h.svc.End(ctx, EndRequest{ActRequest: ActRequest{Session: id2, As: "1001"}}); err != nil {
		t.Fatal(err)
	}
	if rowCount(t, h, id2, "toolu_01GgR6UWwxWCU6FVfmQSDfqe", "decision = 'dropped' AND decided_by = 'instance' AND reason = 'the session ended before an answer'") != 1 {
		t.Error("end did not drop the request")
	}
	if a := receivedAnswers(t, h.root, id2, 1); len(a) != 1 || !strings.Contains(a[0], `"interrupt":true`) {
		t.Errorf("end's answers %q", a)
	}
}

// TestSubagentPromptKill is prompt-ask-kill: the process exits 1 with the request open and no
// result; the request is dropped as at a result and the turn ends interrupted, its exit kept.
func TestSubagentPromptKill(t *testing.T) {
	h := newHarness(t, "", true)
	id, _ := promptSession(t, h, "prompt-ask-kill")
	evs := waitTurn(t, h.svc, id, 1)
	e := evs[len(evs)-1]
	if e.Result != ResultInterrupted || e.Usage == nil || e.Usage.Exit != "1" {
		t.Errorf("turn-ended %+v %+v", e, e.Usage)
	}
	if rowCount(t, h, id, "toolu_01BGWT2bcFyymZNcmJtg42DG", "decision = 'dropped' AND decided_by = 'instance' AND reason = 'the turn ended'") != 1 {
		t.Error("the killed process's request is not dropped")
	}
}

// TestSubagentPromptToolMissing is prompt-ask-no-tool: the hook's ask on a subagent's Bash
// (decision_reason_type hook on system/permission_denied) reaches its result with no request:
// rule 8's internal-error, the audit row and the trace line naming the lift.
func TestSubagentPromptToolMissing(t *testing.T) {
	h := newHarness(t, "", true)
	id, _ := promptSession(t, h, "prompt-ask-no-tool")
	evs := waitTurn(t, h.svc, id, 1)
	const bash = "toolu_01PZ3MExc6x98Sb1m8PrCkAa"
	var internal, trace bool
	for _, ev := range evs {
		internal = internal || ev.Type == EventInternalError && ev.ToolUseID == bash && ev.Summary == promptToolMissingSummary
		trace = trace || ev.Type == EventToolResult && ev.ToolUseID == bash && strings.HasSuffix(ev.Summary, "-> "+tracePromptToolMissing)
	}
	if !internal || !trace {
		t.Errorf("internal %v, trace %v:\n%s", internal, trace, strings.Join(summaries(evs), "\n"))
	}
	if n := count(t, h.st, `SELECT count(*) FROM audit_event WHERE kind = 'engine-prompt-tool-missing' AND details LIKE '%`+bash+`%'`); n != 1 {
		t.Errorf("%d engine-prompt-tool-missing rows, want 1", n)
	}
}

func TestSubagentPromptToolMissingNoTimedOutBypass(t *testing.T) {
	h := newHarness(t, "", true)
	id, _ := promptSession(t, h, "prompt-ask-no-tool")
	waitTurn(t, h.svc, id, 1)
	if n := count(t, h.st, `SELECT count(*) FROM audit_event WHERE kind = 'hook-bypassed' AND details LIKE '%"cause":"timed-out"%'`); n != 0 {
		t.Errorf("%d hook-bypassed rows with cause timed-out beside engine-prompt-tool-missing", n)
	}
}

// TestSubagentPromptStoreFailureDenies: a request whose deferred row the store refuses is
// denied at once (interrupt false), not left open to hang the turn.
func TestSubagentPromptStoreFailureDenies(t *testing.T) {
	h := newHarness(t, "", true)
	exec(t, h.st, `CREATE TRIGGER refuse_deferred BEFORE INSERT ON permission_decision WHEN NEW.decision = 'deferred'
BEGIN SELECT RAISE(ABORT, 'refused'); END`)
	id, _ := promptSession(t, h, "prompt-ask-allow")
	waitTurn(t, h.svc, id, 1)
	a := receivedAnswers(t, h.root, id, 1)
	if len(a) != 1 || !strings.Contains(a[0], `"behavior":"deny"`) || !strings.Contains(a[0], storeFailedDenyMessage) {
		t.Errorf("answers %q, want one deny naming the store failure", a)
	}
}

// TestSubagentPromptRecordFirst is prompt-ask-abort: the engine's own deny at a stdin close
// (source config, reason_type other) for a call whose request the bot holds closes it on the
// record, decider instance, cause engine aborted, never hook bypassed.
func TestSubagentPromptRecordFirst(t *testing.T) {
	h := newHarness(t, "", true)
	id, _ := promptSession(t, h, "prompt-ask-abort")
	waitTurn(t, h.svc, id, 1)
	const bash = "toolu_017aPsf6jZXGiTNgGECxMbG4"
	if rowCount(t, h, id, bash, "decision = 'deny' AND decided_by = 'instance' AND reason = '"+CauseEngineAborted+"'") != 1 {
		t.Error("the aborted call is not deny (instance) engine aborted")
	}
	if n := count(t, h.st, `SELECT count(*) FROM audit_event WHERE kind = 'hook-bypassed'`); n != 0 {
		t.Errorf("%d hook-bypassed rows, want none", n)
	}
	if d, _ := h.svc.Show(id); len(d.Requests) != 0 {
		t.Errorf("%d requests left open", len(d.Requests))
	}
}
