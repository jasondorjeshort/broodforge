// SPDX-License-Identifier: GPL-3.0-or-later

package session

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jasondorjeshort/broodforge/internal/git"
	"github.com/jasondorjeshort/broodforge/internal/policy"
	"github.com/jasondorjeshort/broodforge/internal/store"
)

// The P38 Proof's session half (08-implementation-plan-landing.md): the bot records when it
// first saw each model (model_seen), writes the subagent-result row's model, and rewrites the
// trailers file at each first sighting.

// subagentPinnedFixture is subagent with the Agent input pinning haiku and the subagent's lines
// naming claude-haiku-5-5.
const subagentPinnedFixture = fixtures + "/subagent-pinned"

// tickClock is a clock that moves one second on every Now, so the stamps of the turn's start,
// a launch and later lines differ; its timers are a manual clock's, which never fire here.
type tickClock struct {
	mu  sync.Mutex
	now time.Time
	*ManualClock
}

func (c *tickClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(time.Second)
	return c.now
}

func withTickClock(h *harness) {
	start := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	h.cfg.Clock = &tickClock{now: start, ManualClock: NewManualClock(start)}
	h.svc = New(h.cfg)
}

type seenRow struct {
	model, via, toolUseID, agent string
	at                           time.Time
}

func seenRows(t *testing.T, st *store.Store, id string) []seenRow {
	t.Helper()
	var out []seenRow
	err := st.Query(`SELECT model, first_seen_at, via, coalesce(tool_use_id, ''), coalesce(agent, '') FROM model_seen
WHERE session_id = ? ORDER BY first_seen_at, model`, []store.Value{store.Text(id)}, func(r *store.Row) error {
		at, err := time.Parse(store.TimeLayout, r.Text(1))
		out = append(out, seenRow{model: r.Text(0), at: at, via: r.Text(2), toolUseID: r.Text(3), agent: r.Text(4)})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func trailerLine(model string) string {
	return git.Identity{Name: policy.ModelDisplayName(model), Email: git.ModelEmail}.Trailer()
}

func TestAttributionFirstSeenPin(t *testing.T) {
	h := newHarness(t, "", true)
	withTickClock(h)
	ctx := context.Background()
	if _, err := h.svc.Start(ctx, StartRequest{Channel: "100", As: "1001", Fixture: subagentPinnedFixture, Model: "opus"}); err != nil {
		t.Fatal(err)
	}
	if _, err := h.svc.Submit(ctx, SendRequest{Session: "s0001", As: "1001", Text: "review it"}); err != nil {
		t.Fatal(err)
	}
	waitTurn(t, h.svc, "s0001", 1)

	rows := seenRows(t, h.st, "s0001")
	var started string
	if err := h.st.Query("SELECT started_at FROM turn WHERE session_id = 's0001' AND seq = 1", nil,
		func(r *store.Row) error { started = r.Text(0); return nil }); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0].model != "claude-opus-5-5" || rows[0].via != seenTurn || store.FormatTime(rows[0].at) != started ||
		rows[1].model != "claude-haiku-5-5" || rows[1].via != seenPin || rows[1].toolUseID != "toolu_sub_agent" ||
		rows[1].agent != "reviewer" || !rows[1].at.After(rows[0].at) {
		t.Fatalf("model_seen %+v (turn started %s)", rows, started)
	}
	if n := count(t, h.st, `SELECT count(*) FROM audit_event WHERE kind = 'subagent-result' AND session_id = 's0001'
    AND details LIKE '%"model":"Claude Haiku 5.5"%' AND details LIKE '%"toolUseId":"toolu_sub_agent"%'`); n != 1 {
		t.Errorf("%d subagent-result rows naming Claude Haiku 5.5, want 1", n)
	}
	b, err := os.ReadFile(filepath.FromSlash(git.TrailersPath(h.root + "/sessions/s0001")))
	if want := trailerLine("claude-opus-5-5") + "\n" + trailerLine("claude-haiku-5-5") + "\n"; err != nil || string(b) != want {
		t.Errorf("trailers %q %v, want %q", b, err, want)
	}

	// A commit stamped before the pin is not owed Haiku; one at or after it is.
	a := git.Attribution{Driver: git.Identity{Name: "Ada Example", Email: "ada@example.com"}}
	for _, r := range rows {
		a.Models = append(a.Models, git.Contributor{Identity: git.Identity{Name: policy.ModelDisplayName(r.model), Email: git.ModelEmail}, Since: r.at})
	}
	if got := strings.Join(a.Required(rows[1].at.Add(-time.Second)), "\n"); got != trailerLine("claude-opus-5-5") {
		t.Errorf("before the pin: %q", got)
	}
	if got := strings.Join(a.Required(rows[1].at), "\n"); got != trailerLine("claude-opus-5-5")+"\n"+trailerLine("claude-haiku-5-5") {
		t.Errorf("after the pin: %q", got)
	}
}

// A denied Agent launch credits nothing: its pin row, written at the tool_use block, is
// deleted when the result classifies as a deny, and the trailers file loses the Haiku line.
func TestAttributionDeniedPinCreditsNothing(t *testing.T) {
	h := newHarness(t, "", true)
	withTickClock(h)
	ctx := context.Background()
	if _, err := h.svc.Start(ctx, StartRequest{Channel: "100", As: "1001", Fixture: fixtures + "/subagent-pinned-denied", Model: "opus"}); err != nil {
		t.Fatal(err)
	}
	if _, err := h.svc.Submit(ctx, SendRequest{Session: "s0001", As: "1001", Text: "review it"}); err != nil {
		t.Fatal(err)
	}
	waitTurn(t, h.svc, "s0001", 1)
	for _, r := range seenRows(t, h.st, "s0001") {
		if r.model == "claude-haiku-5-5" {
			t.Errorf("a denied launch left %+v", r)
		}
	}
	b, err := os.ReadFile(filepath.FromSlash(git.TrailersPath(h.root + "/sessions/s0001")))
	if err != nil || strings.Contains(string(b), "Haiku") || string(b) != trailerLine("claude-opus-5-5")+"\n" {
		t.Errorf("trailers %q %v", b, err)
	}
	if n := count(t, h.st, `SELECT count(*) FROM audit_event WHERE kind = 'subagent-result' AND session_id = 's0001'
    AND details LIKE '%"result":"error"%'`); n != 1 {
		t.Errorf("%d errored subagent-result rows, want 1", n)
	}
}

// Under a workflow copy whose agents.json pins the reviewer, an Agent call with no model pins
// from the copy; without a copy, the call pins nothing and its row says none.
func TestAttributionPinFromTheCopy(t *testing.T) {
	h := newHarness(t, "", true)
	withTickClock(h)
	ctx := context.Background()
	h.svc.cfg.Workflow = func(_ context.Context, req WorkflowRequest) (WorkflowCopy, error) {
		if err := os.MkdirAll(req.Dir, 0o755); err != nil {
			return WorkflowCopy{}, err
		}
		agents := `{"reviewer":{"description":"Reviews","prompt":"Review.","model":"sonnet"}}` + "\n"
		if err := os.WriteFile(filepath.Join(req.Dir, "agents.json"), []byte(agents), 0o644); err != nil {
			return WorkflowCopy{}, err
		}
		return WorkflowCopy{Line: "abc123 2026-10-23T09:30:00Z", Copied: true, Branch: "main", Commit: "abc123",
			Agents: req.Dir + "/agents.json"}, nil
	}
	if _, err := h.svc.Start(ctx, StartRequest{Channel: "100", As: "1001", Fixture: subagentFixture}); err != nil {
		t.Fatal(err)
	}
	if _, err := h.svc.Submit(ctx, SendRequest{Session: "s0001", As: "1001", Text: "review it"}); err != nil {
		t.Fatal(err)
	}
	waitTurn(t, h.svc, "s0001", 1)
	rows := seenRows(t, h.st, "s0001")
	// No session model: the driver's model is first seen on its first assistant line.
	if len(rows) != 2 || rows[0].model != "claude-opus-5-5" || rows[0].via != seenStream ||
		rows[1].model != "claude-sonnet-5-5" || rows[1].via != seenPin || rows[1].agent != "reviewer" {
		t.Errorf("model_seen %+v", rows)
	}
	if n := count(t, h.st, `SELECT count(*) FROM audit_event WHERE kind = 'subagent-result' AND session_id = 's0001'
    AND details LIKE '%"model":"Claude Sonnet 5.5"%'`); n != 1 {
		t.Errorf("%d subagent-result rows naming Claude Sonnet 5.5, want 1", n)
	}

	h.svc.cfg.Workflow = nil
	if _, err := h.svc.Start(ctx, StartRequest{Channel: "100", As: "1001", Fixture: subagentFixture}); err != nil {
		t.Fatal(err)
	}
	if _, err := h.svc.Submit(ctx, SendRequest{Session: "s0002", As: "1001", Text: "review it"}); err != nil {
		t.Fatal(err)
	}
	waitTurn(t, h.svc, "s0002", 1)
	if rows := seenRows(t, h.st, "s0002"); len(rows) != 1 || rows[0].via != seenStream {
		t.Errorf("model_seen without a copy %+v", rows)
	}
	if n := count(t, h.st, `SELECT count(*) FROM audit_event WHERE kind = 'subagent-result' AND session_id = 's0002'
    AND details LIKE '%"model":"none"%'`); n != 1 {
		t.Errorf("%d subagent-result rows naming none, want 1", n)
	}
}
