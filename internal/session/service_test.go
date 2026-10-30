// SPDX-License-Identifier: GPL-3.0-or-later

package session

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jasondorjeshort/broodforge/internal/audit"
	"github.com/jasondorjeshort/broodforge/internal/engine"
	"github.com/jasondorjeshort/broodforge/internal/git"
	"github.com/jasondorjeshort/broodforge/internal/store"
)

type harness struct {
	svc   *Service
	cfg   Config // the service's, for a restart on the same store (TestHeld)
	st    *store.Store
	root  string
	clock *ManualClock

	mu      sync.Mutex
	removed []string // the sessions RemoveTree was called for
	// handoff, when set, is what the end of session gets from the handoff; nil hands nothing off.
	handoff func(id string) (git.Outcome, error)
}

func exec(t *testing.T, st *store.Store, stmts ...string) {
	t.Helper()
	err := st.Write(func(tx *store.Tx) error {
		for _, s := range stmts {
			if _, err := tx.Exec(s); err != nil {
				return errors.New(s + ": " + err.Error())
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// newHarness makes a store with users 1001 (credential cred-a, usable) and 1002 (no default),
// a subscription cred-b that needs login, and channels on the fake, every list group
// trusted (role 10): 100 (code reach, policy mode), 200 (read-only, policy) and 300 (100 with
// a model policy does not know). The tree is a
// directory, not a clone; RemoveTree removes it and records the session.
func newHarness(t *testing.T, fixtureDir string, dryRun bool) *harness {
	t.Helper()
	root := filepath.ToSlash(t.TempDir())
	st, err := store.Create(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	exec(t, st,
		`INSERT INTO registration (user_id, git_name, git_email, display_name, registered_at) VALUES ('1001', 'Ada Example', 'ada@example.com', '', '2026-10-16T00:00:00Z')`,
		`INSERT INTO registration (user_id, git_name, git_email, display_name, registered_at) VALUES ('1002', 'Sam Example', 'sam@example.com', '', '2026-10-16T00:00:00Z')`,
		`INSERT INTO credential (id, kind, provider, user_id, label, status, added_at, store_ref) VALUES ('cred-a', 'api-key', 'anthropic', '1001', 'a', 'usable', '2026-10-16T00:00:00Z', 'cred-a')`,
		`INSERT INTO credential (id, kind, provider, user_id, label, status, added_at, store_ref) VALUES ('cred-b', 'subscription', 'anthropic', '1001', 'b', 'needs-login', '2026-10-16T00:00:00Z', '/x')`,
		`UPDATE registration SET default_credential_id = 'cred-a' WHERE user_id = '1001'`)
	fx, err := filepath.Abs(fixtures)
	if err != nil {
		t.Fatal(err)
	}
	roles := Roles{Start: []string{"trusted"}, Contribute: []string{"trusted"}, Answer: []string{"driver"}}
	repo := Repo{Canonical: root + "/canonical", Remote: "origin", DefaultBranch: "main"}
	channels := map[string]Channel{
		"100": {ID: "100", Project: "p", ProjectName: "Project P", Reach: "code", Mode: "policy", Harness: "fake", Roles: roles, Repo: repo},
		"200": {ID: "200", Project: "p", ProjectName: "Project P", Reach: "read-only", Mode: "policy", Harness: "fake", Roles: roles, Repo: repo},
		// 300 holds a model policy does not know, as a last-good file from before P12a may (model_test.go).
		"300": {ID: "300", Project: "p", ProjectName: "Project P", Reach: "code", Mode: "policy", Harness: "fake", Roles: roles, Repo: repo, Model: "gpt-5"},
	}
	clock := NewManualClock(time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC))
	h := &harness{st: st, root: root, clock: clock}
	h.cfg = Config{
		Root: root, Instance: "test", DryRun: dryRun, Groups: func() map[string][]string { return map[string][]string{"trusted": {"10"}} },
		Channel: func(id string) (Channel, bool) { ch, ok := channels[id]; return ch, ok },
		// Project p's repository, as serve reads it from the configuration (a held session's).
		ProjectRepo: func(id string) (Repo, bool) { return repo, id == "p" },
		Fixtures:    fx,
		Engine: func(harness, fixture, received string) (engine.Engine, error) {
			if fixtureDir != "" {
				fixture = fixtureDir
			}
			return engine.NewFake(fixture, received)
		},
		CreateTree: func(_ context.Context, s git.Session) (string, error) {
			return "base", os.MkdirAll(filepath.FromSlash(s.Dir()+"/tree"), 0o750)
		},
		Handoff: func(_ context.Context, id string) (git.Outcome, error) {
			h.mu.Lock()
			f := h.handoff
			h.mu.Unlock()
			if f == nil {
				return git.Outcome{Outcome: git.OutcomeNone}, nil
			}
			return f(id)
		},
		RemoveTree: func(_ context.Context, s git.Session) error {
			h.mu.Lock()
			h.removed = append(h.removed, s.ID)
			h.mu.Unlock()
			return os.RemoveAll(filepath.FromSlash(s.Dir() + "/tree"))
		},
		Store: st, Audit: audit.NewWriter(nil), Clock: clock,
	}
	h.svc = New(h.cfg)
	return h
}

// delayed copies basic's first two turns into a fixture whose lines are delay apart.
func delayed(t *testing.T, delay time.Duration) string {
	t.Helper()
	dir := t.TempDir()
	for n := 1; n <= 2; n++ {
		b, err := os.ReadFile(engine.TurnFile(filepath.Join(fixtures, "basic"), n))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(engine.TurnFile(dir, n), b, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	fx := `{"claim":"test","delayMs":` + itoa(int(delay/time.Millisecond)) + `,"turns":[{"n":1,"exit":0},{"n":2,"exit":0}]}`
	if err := os.WriteFile(filepath.Join(dir, engine.FixtureFile), []byte(fx), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

func itoa(n int) string { b, _ := json.Marshal(n); return string(b) }

// waitTurn follows the session until turn n ends and returns every event seen.
func waitTurn(t *testing.T, svc *Service, id string, n int) []Event {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	var seen []Event
	err := svc.Follow(ctx, id, 0, func(ev Event) (bool, error) {
		seen = append(seen, ev)
		return ev.Type == EventTurnEnded && ev.Turn == n, nil
	})
	if err != nil {
		t.Fatalf("follow %s to turn %d: %v (seen %v)", id, n, err, seen)
	}
	return seen
}

func code(err error) string {
	var se *Error
	if errors.As(err, &se) {
		return se.Code
	}
	if err == nil {
		return ""
	}
	return "other: " + err.Error()
}

func count(t *testing.T, st *store.Store, query string) int64 {
	t.Helper()
	var n int64
	if err := st.Query(query, nil, func(r *store.Row) error { n = r.Int(0); return nil }); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestPolicyTurnAgainstBasic(t *testing.T) {
	h := newHarness(t, "", true)
	ctx := context.Background()
	started, err := h.svc.Start(ctx, StartRequest{Channel: "100", As: "1001", Roles: "10", Fixture: "basic"})
	if err != nil {
		t.Fatal(err)
	}
	if started.ID != "s0001" {
		t.Fatalf("id %s, want s0001", started.ID)
	}
	sub, err := h.svc.Submit(ctx, SendRequest{Session: "s0001", As: "1001", Text: "Read hello.txt and reply with its one word."})
	if err != nil {
		t.Fatal(err)
	}
	if sub.Turn != 1 || sub.State != SubmitStarted {
		t.Fatalf("submit: %+v", sub)
	}
	seen := waitTurn(t, h.svc, "s0001", 1)
	var types []string
	for _, ev := range seen {
		types = append(types, string(ev.Type))
	}
	want := "session-started turn-started tool-call permission-decision tool-result text turn-ended"
	if got := strings.Join(types, " "); got != want {
		t.Errorf("events:\n got %s\nwant %s", got, want)
	}
	for _, ev := range seen {
		// Only the events with fields to check; the order is checked above.
		//exhaustive:ignore
		switch ev.Type {
		case EventPermissionDecision:
			if ev.Decision != DecisionAllow || ev.DecidedBy != DecidedByPolicy || ev.Tool != "Read" {
				t.Errorf("decision event: %+v", ev)
			}
		case EventText:
			if ev.Summary != "hello" {
				t.Errorf("text %q", ev.Summary)
			}
		case EventTurnEnded:
			if ev.Result != ResultCompleted || ev.Summary != "end_turn" {
				t.Errorf("turn-ended: %+v", ev)
			}
		}
	}
	if n := count(t, h.st, `SELECT count(*) FROM permission_decision WHERE session_id = 's0001' AND turn_seq = 1
    AND decided_by = 'policy' AND decision = 'allow' AND tool = 'Read'`); n != 1 {
		t.Errorf("%d decision rows, want 1", n)
	}
	if n := count(t, h.st, `SELECT count(*) FROM turn WHERE session_id = 's0001' AND seq = 1 AND result = 'completed'
    AND requester_id = '1001' AND input LIKE 'From: Ada Example (driver)%'`); n != 1 {
		t.Errorf("turn row missing or wrong")
	}
	if n := count(t, h.st, `SELECT count(*) FROM audit_event WHERE kind = 'permission-decision' AND session_id = 's0001'
    AND actor = 'policy'`); n != 1 {
		t.Errorf("%d permission-decision events, want 1", n)
	}
	// The turn's rate_limit_event is the payer's meter (P28): basic turn 1's values, no audit event.
	if m, err := store.Meter(h.st, "cred-a"); err != nil || m == nil || m.Session != "s0001" || m.Status != "allowed" ||
		m.FiveHourUtilization != 0.16 || m.SevenDayUtilization != 0.04 || m.SevenDayResetsAt != 1792728000 || m.SeenAt == "" {
		t.Errorf("meter of cred-a: %+v, %v", m, err)
	}
	d, err := h.svc.Show("s0001")
	if err != nil {
		t.Fatal(err)
	}
	if d.State != StateWaitingForInput || d.EngineSessionID == nil || d.Worktree == nil || d.Turns != 1 || len(d.Participants) != 1 {
		t.Errorf("show: %+v", d)
	}
	// The request the fake received (05, A turn against the fake engine: Observe).
	b, err := os.ReadFile(filepath.Join(h.root, "sessions", "s0001", "fake", "received-1.json"))
	if err != nil {
		t.Fatal(err)
	}
	var req engine.TurnRequest
	if err := json.Unmarshal(b, &req); err != nil {
		t.Fatal(err)
	}
	if req.Delivery.Kind != engine.DeliveryAPIKey || req.Delivery.CredentialID != "cred-a" || req.GitIdentity.Name != "Ada Example" ||
		!strings.HasPrefix(req.Input, "From: Ada Example (driver)\n") || req.SystemPromptFile != h.root+"/workflow/s0001/instructions.md" ||
		req.AgentsFile != "" {
		t.Errorf("received request: %+v", req)
	}
	// Without Config.Workflow the file is the header alone.
	if b, err := os.ReadFile(req.SystemPromptFile); err != nil || !strings.Contains(string(b), "one tool call at a time") ||
		!strings.HasPrefix(string(b), "This session is run by Broodforge.") || strings.Contains(string(b), "taken from") {
		t.Errorf("instructions.md: %q %v", b, err)
	}
	names := map[string]string{}
	for _, v := range req.HookEnv {
		names[v.Name] = v.Value
	}
	if names[envMode] != "policy" || names[envReach] != "code" || names[envSession] != "s0001" ||
		names[envTree] != h.root+"/sessions/s0001/tree" || !strings.HasSuffix(names[envHookLog], "/sessions/s0001/home/hook.log") {
		t.Errorf("hook env: %v", names)
	}
}

func TestQueuedMessageJoinsAsContributor(t *testing.T) {
	h := newHarness(t, delayed(t, 20*time.Millisecond), true)
	ctx := context.Background()
	if _, err := h.svc.Start(ctx, StartRequest{Channel: "100", As: "1001"}); err != nil {
		t.Fatal(err)
	}
	first, err := h.svc.Submit(ctx, SendRequest{Session: "s0001", As: "1001", Text: "one"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := h.svc.Submit(ctx, SendRequest{Session: "s0001", As: "1002", Roles: "10", Text: "two"})
	if err != nil {
		t.Fatal(err)
	}
	if first.State != SubmitStarted || second.State != SubmitQueued || second.Turn != 2 {
		t.Fatalf("first %+v, second %+v", first, second)
	}
	seen := waitTurn(t, h.svc, "s0001", 2)
	joined := false
	for _, ev := range seen {
		if ev.Type == EventParticipantJoined && ev.Summary == "Sam Example joined as a contributor." {
			joined = true
		}
	}
	if !joined {
		t.Errorf("no participant-joined event: %v", seen)
	}
	if n := count(t, h.st, `SELECT count(*) FROM turn WHERE session_id = 's0001' AND seq = 2 AND requester_id = '1002'
    AND input = 'From: Sam Example (contributor)' || char(10) || 'two'`); n != 1 {
		t.Errorf("turn 2 row missing or wrong")
	}
	if n := count(t, h.st, `SELECT count(*) FROM participant WHERE session_id = 's0001' AND user_id = '1002' AND role = 'contributor'`); n != 1 {
		t.Errorf("no contributor row")
	}
	// Turn 2 resumes the engine session turn 1 opened.
	b, err := os.ReadFile(filepath.Join(h.root, "sessions", "s0001", "fake", "received-2.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"engineSessionId": "f0451bd7-4428-40b5-a52e-ad6b604c210a"`) {
		t.Errorf("turn 2 is not a resume:\n%s", b)
	}
	// The contributor is now a co-author line, then the model that has usage (03, Attribution).
	tr, err := os.ReadFile(filepath.FromSlash(git.TrailersPath(h.root + "/sessions/s0001")))
	if err != nil || string(tr) != "Co-Authored-By: Sam Example <sam@example.com>\nCo-Authored-By: Claude Haiku 5.5 <noreply@anthropic.com>\n" {
		t.Errorf("trailers %q, %v", tr, err)
	}
}

func TestRefusals(t *testing.T) {
	h := newHarness(t, "", true)
	ctx := context.Background()
	for _, c := range []struct {
		name string
		req  StartRequest
		want string
	}{
		{"unknown channel", StartRequest{Channel: "999", As: "1001"}, CodeUnknownChannel},
		{"no driver", StartRequest{Channel: "100"}, CodeBadArgs},
		{"unregistered", StartRequest{Channel: "100", As: "4242"}, CodeUnregistered},
		{"roles", StartRequest{Channel: "100", As: "1001", Roles: "11"}, CodeRefused},
		{"no credential", StartRequest{Channel: "100", As: "1002"}, CodeNoCredential},
		{"needs login", StartRequest{Channel: "100", As: "1001", Credential: "cred-b"}, CodeCredentialStatus},
		{"unknown credential", StartRequest{Channel: "100", As: "1001", Credential: "cred-zz"}, "unknown-credential"},
		{"bad mode", StartRequest{Channel: "100", As: "1001", Mode: "auto"}, CodeBadArgs},
		{"unknown fixture", StartRequest{Channel: "100", As: "1001", Fixture: "nope"}, CodeUnknownFixture},
	} {
		if _, err := h.svc.Start(ctx, c.req); code(err) != c.want {
			t.Errorf("%s: got %s, want %s", c.name, code(err), c.want)
		}
	}
	if n := count(t, h.st, "SELECT count(*) FROM session"); n != 0 {
		t.Errorf("a refused start left %d session rows", n)
	}
	if n := count(t, h.st, "SELECT count(*) FROM audit_event WHERE kind = 'action-refused'"); n < 6 {
		t.Errorf("%d action-refused events", n)
	}
	if _, err := h.svc.Start(ctx, StartRequest{Channel: "100", As: "1001"}); err != nil {
		t.Fatal(err)
	}
	if _, err := h.svc.Submit(ctx, SendRequest{Session: "s0001", As: "1002", Roles: "11", Text: "x"}); code(err) != CodeRefused {
		t.Errorf("contribute with role 11: %s", code(err))
	}
	if _, err := h.svc.Submit(ctx, SendRequest{Session: "s0009", As: "1001", Text: "x"}); code(err) != CodeUnknownSession {
		t.Errorf("unknown session: %s", code(err))
	}

	live := newHarness(t, "", false)
	if _, err := live.svc.Start(ctx, StartRequest{Channel: "100", As: "1001", Reach: "read-only"}); code(err) != CodeLiveInstance {
		t.Errorf("--reach on a live instance: %s", code(err))
	}
}

func TestShutdownInterruptsTheRunningTurn(t *testing.T) {
	h := newHarness(t, delayed(t, 5*time.Second), true)
	ctx := context.Background()
	if _, err := h.svc.Start(ctx, StartRequest{Channel: "100", As: "1001"}); err != nil {
		t.Fatal(err)
	}
	if _, err := h.svc.Submit(ctx, SendRequest{Session: "s0001", As: "1001", Text: "slow"}); err != nil {
		t.Fatal(err)
	}
	sctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	begun := time.Now()
	marked, err := h.svc.Shutdown(sctx)
	if err != nil {
		t.Fatal(err)
	}
	if time.Since(begun) > 4*time.Second {
		t.Errorf("shutdown waited out the fixture's delay instead of interrupting")
	}
	if len(marked) != 1 || marked[0] != "s0001" {
		t.Errorf("marked %v", marked)
	}
	if n := count(t, h.st, `SELECT count(*) FROM turn WHERE session_id = 's0001' AND seq = 1 AND result = 'interrupted'`); n != 1 {
		t.Errorf("turn 1 not interrupted")
	}
	if n := count(t, h.st, `SELECT count(*) FROM session WHERE id = 's0001' AND state = 'interrupted' AND ended_at IS NOT NULL`); n != 1 {
		t.Errorf("session not interrupted")
	}
	if n := count(t, h.st, `SELECT count(*) FROM audit_event WHERE kind = 'session-interrupted' AND session_id = 's0001'`); n != 1 {
		t.Errorf("no session-interrupted event")
	}
	if _, err := h.svc.Submit(ctx, SendRequest{Session: "s0001", As: "1001", Text: "late"}); code(err) != CodeShuttingDown {
		t.Errorf("submit after shutdown: %s", code(err))
	}
}

func TestReconcileAtStart(t *testing.T) {
	h := newHarness(t, "", true)
	exec(t, h.st,
		`INSERT INTO session (id, channel_id, project_id, driver_id, state, harness, credential_id, mode, reach, started_at)
VALUES ('s0001', '100', 'p', '1001', 'running-turn', 'fake', 'cred-a', 'policy', 'code', '2026-10-16T00:00:00Z')`,
		`INSERT INTO worktree (path, project_id, session_id, branch, state, created_at)
VALUES ('`+h.root+`/sessions/s0001/tree', 'p', 's0001', 'b', 'active', '2026-10-16T00:00:00Z')`)
	findings, err := Reconcile(h.st, audit.NewWriter(nil), h.root, h.clock.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 2 {
		t.Fatalf("findings %+v", findings)
	}
	if n := count(t, h.st, `SELECT count(*) FROM session WHERE id = 's0001' AND state = 'interrupted'`); n != 1 {
		t.Errorf("session not interrupted")
	}
	if n := count(t, h.st, `SELECT count(*) FROM worktree WHERE state = 'active' AND missing_since = '2026-10-16T12:00:00Z'`); n != 1 {
		t.Errorf("worktree state changed or missing_since unset")
	}
	if n := count(t, h.st, `SELECT count(*) FROM audit_event WHERE kind = 'session-interrupted' AND details LIKE '%reconciliation%'`); n != 1 {
		t.Errorf("no reconciliation event")
	}
}

func TestRegister(t *testing.T) {
	h := newHarness(t, "", true)
	r, err := h.svc.Register("1003", "New Person", "new@example.com", "")
	if err != nil || r.Name != "New Person" || r.DefaultCredential != nil {
		t.Fatalf("register: %+v, %v", r, err)
	}
	if _, err := h.svc.Register("1003", "Renamed", "renamed@example.com", ""); err != nil {
		t.Fatal(err)
	}
	if n := count(t, h.st, `SELECT count(*) FROM audit_event WHERE kind = 'registration-changed' AND registration_id = '1003'
    AND details LIKE '%"old":"New Person <new@example.com>"%'`); n != 1 {
		t.Errorf("re-registration did not record the old pair")
	}
	for _, c := range []struct{ user, name, email, want string }{
		{"1003", "", "a@example.com", CodeBadRegistration},
		{"1003", strings.Repeat("n", 101), "a@example.com", CodeBadRegistration},
		{"1003", "N", "no-at", CodeBadRegistration},
		{"1003", "N", "a b@example.com", CodeBadRegistration},
		{"x1", "N", "a@example.com", CodeBadArgs},
	} {
		if _, err := h.svc.Register(c.user, c.name, c.email, ""); code(err) != c.want {
			t.Errorf("register %q %q %q: %s, want %s", c.user, c.name, c.email, code(err), c.want)
		}
	}
	if r, err := h.svc.SetDefault("1002", "cred-a", ""); err != nil || r.DefaultCredential == nil || *r.DefaultCredential != "cred-a" {
		t.Errorf("set-default: %+v, %v", r, err)
	}
}

// TestFailedStartLeavesNothingAndNoIDIsReused: a start that fails after the clone (here a
// fixture.json that is not JSON) removes the tree and the ref and keeps its row ended, so the
// next starts get new ids and succeed.
func TestFailedStartLeavesNothingAndNoIDIsReused(t *testing.T) {
	h := newHarness(t, "", true)
	ctx := context.Background()
	broken := t.TempDir()
	if err := os.WriteFile(filepath.Join(broken, engine.FixtureFile), []byte("{\n bad"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := h.svc.Start(ctx, StartRequest{Channel: "100", As: "1001", Fixture: broken})
	var se *Error
	if !errors.As(err, &se) || se.Code != CodeBadFixture || !strings.HasSuffix(se.Message, "fixture.json is not valid JSON at line 2") ||
		strings.Contains(se.Message, `\`) {
		t.Fatalf("broken fixture: %v", err)
	}
	if len(h.removed) != 1 || h.removed[0] != "s0001" {
		t.Errorf("RemoveTree calls %v, want [s0001]", h.removed)
	}
	if _, err := os.Stat(filepath.Join(h.root, "sessions", "s0001")); !os.IsNotExist(err) {
		t.Errorf("sessions/s0001 left behind: %v", err)
	}
	if n := count(t, h.st, `SELECT count(*) FROM session WHERE id = 's0001' AND state = 'ended' AND end_outcome = 'refused: bad-fixture'`); n != 1 {
		t.Errorf("the failed start's row is not kept as ended")
	}
	if n := count(t, h.st, `SELECT count(*) FROM audit_event WHERE kind = 'action-refused' AND session_id = 's0001'
    AND details LIKE '%"reason":"engine"%'`); n != 1 {
		t.Errorf("the refusal does not name the engine stage")
	}
	for _, want := range []string{"s0002", "s0003"} {
		st, err := h.svc.Start(ctx, StartRequest{Channel: "100", As: "1001"})
		if err != nil || st.ID != want {
			t.Fatalf("start after the failure: %+v, %v; want %s", st, err, want)
		}
	}
}

// TestConcurrentSessions: two sessions on channels 100 and 200 run their turns at once against
// different fixtures; each sees only its own events and rows (N-26). Meant for -race.
func TestConcurrentSessions(t *testing.T) {
	h := newHarness(t, "", true)
	ctx := context.Background()
	slow := delayed(t, 10*time.Millisecond)
	a, err := h.svc.Start(ctx, StartRequest{Channel: "100", As: "1001", Fixture: slow})
	if err != nil {
		t.Fatal(err)
	}
	b, err := h.svc.Start(ctx, StartRequest{Channel: "200", As: "1001", Fixture: "hook-exit2"})
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	seen := map[string][]Event{}
	var mu sync.Mutex
	for _, id := range []string{a.ID, b.ID} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := h.svc.Submit(ctx, SendRequest{Session: id, As: "1001", Text: "go"}); err != nil {
				t.Error(err)
				return
			}
			evs := waitTurn(t, h.svc, id, 1)
			mu.Lock()
			seen[id] = evs
			mu.Unlock()
		}()
	}
	wg.Wait()
	if t.Failed() {
		return
	}
	decisions := func(id string) string {
		var out []string
		for _, ev := range seen[id] {
			if ev.Type == EventPermissionDecision {
				out = append(out, ev.Decision+"/"+ev.DecidedBy)
			}
		}
		return strings.Join(out, " ")
	}
	if got := decisions(a.ID); got != "allow/policy" {
		t.Errorf("%s decisions %q", a.ID, got)
	}
	if got := decisions(b.ID); got != "deny/hook deny/hook" {
		t.Errorf("%s decisions %q", b.ID, got)
	}
	if n := count(t, h.st, `SELECT count(*) FROM permission_decision WHERE session_id = '`+b.ID+`' AND decided_by = 'hook'`); n != 2 {
		t.Errorf("%s: %d hook rows, want 2", b.ID, n)
	}
	if n := count(t, h.st, `SELECT count(*) FROM audit_event WHERE kind = 'action-refused' AND actor = 'hook' AND session_id = '`+b.ID+`'`); n != 2 {
		t.Errorf("%s: %d hook-failed refusals, want 2", b.ID, n)
	}
}

// TestQueuedTurnThatCannotStart: the fixture has one turn, so the queued second message's turn
// fails to start; it still ends (a `send --wait` returns) and is audited.
func TestQueuedTurnThatCannotStart(t *testing.T) {
	dir := delayed(t, 20*time.Millisecond)
	if err := os.Remove(engine.TurnFile(dir, 2)); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, dir, true)
	ctx := context.Background()
	if _, err := h.svc.Start(ctx, StartRequest{Channel: "100", As: "1001"}); err != nil {
		t.Fatal(err)
	}
	if _, err := h.svc.Submit(ctx, SendRequest{Session: "s0001", As: "1001", Text: "one"}); err != nil {
		t.Fatal(err)
	}
	if sub, err := h.svc.Submit(ctx, SendRequest{Session: "s0001", As: "1001", Text: "two"}); err != nil || sub.State != SubmitQueued {
		t.Fatalf("second submit: %+v, %v", sub, err)
	}
	seen := waitTurn(t, h.svc, "s0001", 2)
	if last := seen[len(seen)-1]; last.Result != ResultError || !strings.HasPrefix(last.Summary, "engine: ") {
		t.Errorf("turn 2 ended %+v", last)
	}
	if n := count(t, h.st, `SELECT count(*) FROM audit_event WHERE kind = 'action-refused' AND details LIKE '%"reason":"turn-start"%'`); n != 1 {
		t.Errorf("%d turn-start refusals, want 1", n)
	}
	d, err := h.svc.Show("s0001")
	if err != nil || d.State != StateWaitingForInput {
		t.Errorf("after the failed turn: %+v, %v", d, err)
	}
}
