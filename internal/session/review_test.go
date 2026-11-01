// SPDX-License-Identifier: GPL-3.0-or-later

package session

import (
	"context"
	"errors"
	"os"
	osexec "os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/jasondorjeshort/broodforge/internal/engine"
	"github.com/jasondorjeshort/broodforge/internal/git"
	"github.com/jasondorjeshort/broodforge/internal/git/gittest"
)

// reviewCanonical makes the harness's canonical a repository whose refs/sessions/s0001 is a
// commit dated 2020, and returns its path and that commit.
func reviewCanonical(t *testing.T, h *harness) (string, string) {
	t.Helper()
	canonical := filepath.Join(h.root, "canonical")
	if err := os.MkdirAll(canonical, 0o750); err != nil {
		t.Fatal(err)
	}
	gittest.Git(t, canonical, "init", "--quiet", "-b", "main")
	old := gittest.Commit(t, gittest.Driver, []string{"GIT_COMMITTER_DATE=2020-01-01T00:00:00Z"}, canonical, "a.txt", "a", "base")
	gittest.Git(t, canonical, "update-ref", git.SessionRef("s0001"), old)
	return canonical, old
}

// The hand-written streams of the `reviewer` cell (ADR 0028 Decision 3): turn 1 ends deferred on
// the reviewer's Agent call, turn 2 is the resume; reviewer-empty's result has no text.
const (
	reviewerDeferFixture = fixtures + "/reviewer-defer"
	reviewerEmptyFixture = fixtures + "/reviewer-empty"
	reviewerCall         = "toolu_rev_agent"
)

// standInRefresh is git.PinReview for a canonical whose own main stands in for the session's
// tree: the "bundle" sets refs/sessions/s0001 to main's commit and returns it, as the bot's
// fetch would, with dirty as the tree's tracked changes. It runs on the turn's goroutine too, so
// it fails by its error, never t.
func standInRefresh(canonical, dirty string) func(context.Context, git.Session) (git.Pin, error) {
	return func(ctx context.Context, s git.Session) (git.Pin, error) {
		run := func(args ...string) (string, error) {
			cmd := osexec.CommandContext(ctx, "git", append([]string{"-C", canonical}, args...)...)
			cmd.Env = gittest.Env(gittest.Driver)
			out, err := cmd.Output()
			return strings.TrimSpace(string(out)), err
		}
		sha, err := run("rev-parse", "--verify", "main^{commit}")
		if err != nil {
			return git.Pin{}, err
		}
		if _, err := run("update-ref", git.SessionRef(s.ID), sha); err != nil {
			return git.Pin{}, err
		}
		return git.Pin{Commit: sha, Dirty: dirty}, nil
	}
}

// withRefresh restarts the harness's service with refresh as its Config.Refresh.
func withRefresh(h *harness, refresh func(context.Context, git.Session) (git.Pin, error)) {
	h.cfg.Refresh = refresh
	h.svc = New(h.cfg)
}

// reviewTurn starts s0001 on fixture and runs its first turn to the end.
func reviewTurn(t *testing.T, h *harness, fixture string) []Event {
	t.Helper()
	ctx := context.Background()
	if _, err := h.svc.Start(ctx, StartRequest{Channel: "100", As: "1001", Fixture: fixture}); err != nil {
		t.Fatal(err)
	}
	if _, err := h.svc.Submit(ctx, SendRequest{Session: "s0001", As: "1001", Text: "review it"}); err != nil {
		t.Fatal(err)
	}
	return waitTurn(t, h.svc, "s0001", 1)
}

// TestReviewFact: `review:` is the reviewer Agent call whose subagent-result row names the
// commit the bot bundled while answering its defer, while refs/sessions/<id> still holds it. A
// commit after the review, refreshed by `worktree check`, voids it, also one whose committer
// date is a year before the review (the case the committer-time rule missed); a reset back to
// the reviewed commit holds again; a moved canonical is an error, never hidden; a session whose
// ref is gone has no fact.
func TestReviewFact(t *testing.T) {
	h := newHarness(t, "", true)
	ctx := context.Background()
	canonical, old := reviewCanonical(t, h)
	refresh := standInRefresh(canonical, "")
	withRefresh(h, refresh)
	worktreeCheck := func() string {
		t.Helper()
		p, err := refresh(ctx, git.Session{ID: "s0001"})
		if err != nil {
			t.Fatal(err)
		}
		return p.Commit
	}
	if _, err := h.svc.Start(ctx, StartRequest{Channel: "100", As: "1001", Fixture: reviewerDeferFixture}); err != nil {
		t.Fatal(err)
	}
	d, err := h.svc.Inspect(ctx, "s0001", false)
	if err != nil || d.Review == nil || d.Review.Commit != old || d.Review.ToolUseID != nil || d.Review.Error != nil {
		t.Fatalf("before the review: %+v %v", d.Review, err)
	}
	if _, err := h.svc.Submit(ctx, SendRequest{Session: "s0001", As: "1001", Text: "review it"}); err != nil {
		t.Fatal(err)
	}
	waitTurn(t, h.svc, "s0001", 1)
	d, err = h.svc.Inspect(ctx, "s0001", false)
	if err != nil || d.Review == nil || d.Review.Commit != old || d.Review.ToolUseID == nil || *d.Review.ToolUseID != reviewerCall {
		t.Fatalf("after the review: %+v %v", d.Review, err)
	}
	if n := count(t, h.st, `SELECT count(*) FROM audit_event WHERE kind = 'subagent-result' AND session_id = 's0001'
    AND details = '{"agent":"reviewer","commit":"`+old+`","model":"none","result":"ok","tool":"Agent","toolUseId":"`+reviewerCall+`","turn":"1"}'`); n != 1 {
		t.Errorf("%d subagent-result rows for the reviewer naming %s, want 1", n, old)
	}

	// A commit after the review dated a year before it (the harness clock is 2026-10-16), then
	// worktree check: the ref moved, so no review since it.
	backdated := gittest.Commit(t, gittest.Driver, []string{"GIT_COMMITTER_DATE=2025-10-16T12:00:00Z"}, canonical, "b.txt", "b", "backdated")
	if got := worktreeCheck(); got != backdated {
		t.Fatalf("worktree check fetched %s, want %s", got, backdated)
	}
	if d, err = h.svc.Inspect(ctx, "s0001", false); err != nil || d.Review == nil || d.Review.Commit != backdated || d.Review.ToolUseID != nil {
		t.Errorf("after a backdated commit: %+v %v", d.Review, err)
	}

	// Reset back to the reviewed commit: the newest review names it again.
	gittest.Git(t, canonical, "reset", "--hard", "--quiet", old)
	worktreeCheck()
	if d, err = h.svc.Inspect(ctx, "s0001", false); err != nil || d.Review == nil || d.Review.Commit != old || d.Review.ToolUseID == nil {
		t.Errorf("reset to the reviewed commit: %+v %v", d.Review, err)
	}

	// A commit after the review, fetched by the next bundle: no review since it.
	later := gittest.Commit(t, gittest.Driver, nil, canonical, "c.txt", "c", "after")
	worktreeCheck()
	if d, err = h.svc.Inspect(ctx, "s0001", false); err != nil || d.Review == nil || d.Review.Commit != later || d.Review.ToolUseID != nil {
		t.Errorf("after a later commit: %+v %v", d.Review, err)
	}

	// A canonical moved away: the error with its code, not a silent `-`.
	moved := canonical + "-moved"
	if err := os.Rename(canonical, moved); err != nil {
		t.Fatal(err)
	}
	if d, err = h.svc.Inspect(ctx, "s0001", false); err != nil || d.Review == nil || d.Review.Error == nil ||
		d.Review.Error.Code != git.CodeSessionRef || d.Review.ToolUseID != nil {
		t.Errorf("moved canonical: %+v %v", d.Review, err)
	}
	if err := os.Rename(moved, canonical); err != nil {
		t.Fatal(err)
	}

	// Handed off: the ref is gone, the fact unknown.
	gittest.Git(t, canonical, "update-ref", "-d", git.SessionRef("s0001"))
	if d, err = h.svc.Inspect(ctx, "s0001", false); err != nil || d.Review != nil {
		t.Errorf("handed off: %+v %v", d.Review, err)
	}
}

// TestReviewFactExclusions: each row below, alone, leaves `review: none since`, though most name
// the tip: a non-reviewer subagent, an errored result, a denied call, a reviewer allowed and then
// interrupted (a decision row, no result), a result with no agent field, a reviewer result of
// another tool, a reviewer row with no commit (written before P31), one whose bundle failed
// (commitError), one for another commit. Then a reviewer row of `Task` naming the tip counts,
// and a newer ok row for another commit voids that older row's match.
func TestReviewFactExclusions(t *testing.T) {
	h := newHarness(t, "", true)
	ctx := context.Background()
	_, old := reviewCanonical(t, h)
	if _, err := h.svc.Start(ctx, StartRequest{Channel: "100", As: "1001"}); err != nil {
		t.Fatal(err)
	}
	row := func(kind, details string) string {
		return `INSERT INTO audit_event (time, kind, actor, session_id, details) VALUES ('2026-10-16T12:00:00Z', '` + kind +
			`', 'instance', 's0001', '` + details + `')`
	}
	tip := `"commit":"` + old + `",`
	other := `"commit":"0123456789abcdef0123456789abcdef01234567",`
	for _, c := range []struct{ name, stmt string }{
		{"not the reviewer", row("subagent-result", `{"agent":"builder",`+tip+`"result":"ok","tool":"Agent","toolUseId":"t3"}`)},
		{"an errored result", row("subagent-result", `{"agent":"reviewer",`+tip+`"result":"error","tool":"Agent","toolUseId":"t4"}`)},
		{"a denied call", row("permission-decision", `{"agent":"reviewer","decision":"deny","tool":"Agent","toolUseId":"t5"}`)},
		{"allowed, then interrupted", row("permission-decision", `{"agent":"reviewer","decision":"allow","tool":"Agent","toolUseId":"t6"}`)},
		{"no agent field", row("subagent-result", `{`+tip+`"result":"ok","tool":"Agent","toolUseId":"t7"}`)},
		{"not the subagent tool", row("subagent-result", `{"agent":"reviewer",`+tip+`"result":"ok","tool":"Read","toolUseId":"t8"}`)},
		{"no commit", row("subagent-result", `{"agent":"reviewer","result":"ok","tool":"Agent","toolUseId":"t1"}`)},
		{"a failed bundle", row("subagent-result", `{"agent":"reviewer",`+tip+`"commitError":"refused-bundle: no tree","result":"ok","tool":"Agent","toolUseId":"t2"}`)},
		{"another commit", row("subagent-result", `{"agent":"reviewer",`+other+`"result":"ok","tool":"Agent","toolUseId":"t10"}`)},
	} {
		exec(t, h.st, c.stmt)
		if d, err := h.svc.Inspect(ctx, "s0001", false); err != nil || d.Review == nil || d.Review.Commit != old || d.Review.ToolUseID != nil {
			t.Errorf("%s counted: %+v %v", c.name, d.Review, err)
		}
	}
	exec(t, h.st, row("subagent-result", `{"agent":"reviewer",`+tip+`"result":"ok","tool":"Task","toolUseId":"t9"}`))
	if d, err := h.svc.Inspect(ctx, "s0001", false); err != nil || d.Review == nil || d.Review.ToolUseID == nil || *d.Review.ToolUseID != "t9" {
		t.Errorf("the reviewer's result: %+v %v", d.Review, err)
	}
	exec(t, h.st, row("subagent-result", `{"agent":"reviewer",`+other+`"result":"ok","tool":"Agent","toolUseId":"t11"}`))
	if d, err := h.svc.Inspect(ctx, "s0001", false); err != nil || d.Review == nil || d.Review.ToolUseID != nil {
		t.Errorf("an older row's match survived a newer review of another commit: %+v %v", d.Review, err)
	}
}

// TestReviewFactSkipsEmptyAndFailed: a later reviewer row with no result text, a dirty pin or a
// failed bundle is skipped when choosing the newest row (ADR 0028 Decision 3), so an earlier good
// review of the tip still holds, and no last launch is shown beside a fact.
func TestReviewFactSkipsEmptyAndFailed(t *testing.T) {
	h := newHarness(t, "", true)
	ctx := context.Background()
	_, old := reviewCanonical(t, h)
	if _, err := h.svc.Start(ctx, StartRequest{Channel: "100", As: "1001"}); err != nil {
		t.Fatal(err)
	}
	row := func(details string) string {
		return `INSERT INTO audit_event (time, kind, actor, session_id, details) VALUES ('2026-10-16T12:00:00Z', 'subagent-result', 'instance', 's0001', '` +
			details + `')`
	}
	tip := `"commit":"` + old + `",`
	exec(t, h.st, row(`{"agent":"reviewer",`+tip+`"result":"ok","tool":"Agent","toolUseId":"good"}`))
	for _, later := range []string{
		`{"agent":"reviewer",` + tip + `"empty":"true","result":"ok","tool":"Agent","toolUseId":"empty"}`,
		`{"agent":"reviewer",` + tip + `"commitError":"refused-bundle: no tree","result":"ok","tool":"Agent","toolUseId":"failed"}`,
		`{"agent":"reviewer",` + tip + `"commitError":"dirty tree at the reviewer''s launch","dirty":" M a.txt","result":"ok","tool":"Agent","toolUseId":"dirty"}`,
	} {
		exec(t, h.st, row(later))
		if d, err := h.svc.Inspect(ctx, "s0001", false); err != nil || d.Review == nil || d.Review.ToolUseID == nil ||
			*d.Review.ToolUseID != "good" || d.Review.LastLaunch != "" {
			t.Errorf("after %s: %+v %v", later, d.Review, err)
		}
	}
}

// TestReviewFactRefreshFailed: a bundle that fails while answering the reviewer's defer leaves
// commitError in the subagent-result row and no commit; the call still resumes, and the fact
// stays none with the error as the last launch.
func TestReviewFactRefreshFailed(t *testing.T) {
	h := newHarness(t, "", true)
	ctx := context.Background()
	_, old := reviewCanonical(t, h)
	withRefresh(h, func(context.Context, git.Session) (git.Pin, error) {
		return git.Pin{}, errors.New("refused-bundle: no tree")
	})
	reviewTurn(t, h, reviewerDeferFixture)
	if n := count(t, h.st, `SELECT count(*) FROM audit_event WHERE kind = 'subagent-result' AND session_id = 's0001'
    AND details = '{"agent":"reviewer","commitError":"refused-bundle: no tree","model":"none","result":"ok","tool":"Agent","toolUseId":"`+reviewerCall+`","turn":"1"}'`); n != 1 {
		t.Errorf("%d subagent-result rows carrying the bundle's error, want 1", n)
	}
	if d, err := h.svc.Inspect(ctx, "s0001", false); err != nil || d.Review == nil || d.Review.Commit != old || d.Review.ToolUseID != nil ||
		d.Review.LastLaunch != "refused-bundle: no tree" {
		t.Errorf("a review whose bundle failed counted: %+v %v", d.Review, err)
	}
}

// TestReviewerDeferPins: the `reviewer` cell's defer is answered by the bot: the deferred row by
// policy, then the allow by instance, the resume at once with no waiting-for-answer and no
// pending call, the subagent's lines under the Agent call, the session waiting for input after.
// The pin is taken once, at the defer, never at the tool-use read.
func TestReviewerDeferPins(t *testing.T) {
	h := newHarness(t, "", true)
	ctx := context.Background()
	canonical, old := reviewCanonical(t, h)
	var pins atomic.Int32
	refresh := standInRefresh(canonical, "")
	withRefresh(h, func(ctx context.Context, s git.Session) (git.Pin, error) { pins.Add(1); return refresh(ctx, s) })
	evs := reviewTurn(t, h, reviewerDeferFixture)
	var got []string
	traced := map[EventType]bool{EventText: true, EventToolCall: true, EventPermissionDecision: true, EventToolResult: true}
	for _, ev := range evs {
		if ev.Type == EventWaitingForAnswer {
			t.Errorf("waiting-for-answer published for the reviewer: %+v", ev)
		}
		if ev.Type == EventTurnEnded && ev.Result != ResultCompleted {
			t.Errorf("turn ended %s: %s", ev.Result, ev.Summary)
		}
		if traced[ev.Type] {
			got = append(got, string(ev.Type)+"  "+ev.Summary)
		}
	}
	want := []string{
		"text  Launching the reviewer on its own.",
		"tool-call  [Agent]",
		"permission-decision  deferred [Agent] (policy)",
		"permission-decision  allow [Agent] (instance)",
		"tool-call    [Read] hello.txt",
		"permission-decision    allow [Read] hello.txt (policy)",
		"tool-result    [Read] hello.txt",
		"text    No findings.",
		"tool-result  [Agent]",
		"text  The reviewer found nothing to fix.",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("session tail\n got:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	if pins.Load() != 1 {
		t.Errorf("%d pins, want 1", pins.Load())
	}
	for _, c := range []struct{ decision, decidedBy, actor string }{{"deferred", "policy", "policy"}, {"allow", "instance", "instance"}} {
		if n := count(t, h.st, `SELECT count(*) FROM audit_event WHERE kind = 'permission-decision' AND session_id = 's0001' AND actor = '`+c.actor+`'
    AND details LIKE '%"decision":"`+c.decision+`"%' AND details LIKE '%"decidedBy":"`+c.decidedBy+`"%' AND details LIKE '%"toolUseId":"`+reviewerCall+`"%'`); n != 1 {
			t.Errorf("%d %s rows by %s for the reviewer call, want 1", n, c.decision, c.decidedBy)
		}
	}
	d, err := h.svc.Show("s0001")
	if err != nil || d.State != StateWaitingForInput || d.Pending != nil {
		t.Errorf("after the resume: %s %+v %v", d.State, d.Pending, err)
	}
	if d, err := h.svc.Inspect(ctx, "s0001", false); err != nil || d.Review == nil || d.Review.ToolUseID == nil || *d.Review.ToolUseID != reviewerCall ||
		d.Review.Commit != old {
		t.Errorf("review: %+v %v", d.Review, err)
	}
}

// TestReviewerEmptyResult: a reviewer whose result has no text writes its row with `empty`, and
// the fact stays none though the pin names the tip.
func TestReviewerEmptyResult(t *testing.T) {
	h := newHarness(t, "", true)
	ctx := context.Background()
	canonical, old := reviewCanonical(t, h)
	withRefresh(h, standInRefresh(canonical, ""))
	reviewTurn(t, h, reviewerEmptyFixture)
	if n := count(t, h.st, `SELECT count(*) FROM audit_event WHERE kind = 'subagent-result' AND session_id = 's0001'
    AND details = '{"agent":"reviewer","commit":"`+old+`","empty":"true","model":"none","result":"ok","tool":"Agent","toolUseId":"`+reviewerCall+`","turn":"1"}'`); n != 1 {
		t.Errorf("%d empty reviewer rows naming %s, want 1", n, old)
	}
	if d, err := h.svc.Inspect(ctx, "s0001", false); err != nil || d.Review == nil || d.Review.Commit != old || d.Review.ToolUseID != nil ||
		d.Review.LastLaunch != "" {
		t.Errorf("an empty review counted: %+v %v", d.Review, err)
	}
}

// TestReviewerDirtyPin: a pin over a modified or deleted tracked file keeps the sha and the
// porcelain lines, and its commitError voids the fact, shown as the last launch.
func TestReviewerDirtyPin(t *testing.T) {
	h := newHarness(t, "", true)
	ctx := context.Background()
	canonical, old := reviewCanonical(t, h)
	withRefresh(h, standInRefresh(canonical, " M a.txt"))
	reviewTurn(t, h, reviewerDeferFixture)
	if n := count(t, h.st, `SELECT count(*) FROM audit_event WHERE kind = 'subagent-result' AND session_id = 's0001'
    AND details = '{"agent":"reviewer","commit":"`+old+`","commitError":"dirty tree at the reviewer''s launch","dirty":" M a.txt","model":"none","result":"ok","tool":"Agent","toolUseId":"`+reviewerCall+`","turn":"1"}'`); n != 1 {
		t.Errorf("%d dirty reviewer rows, want 1", n)
	}
	if d, err := h.svc.Inspect(ctx, "s0001", false); err != nil || d.Review == nil || d.Review.ToolUseID != nil ||
		d.Review.LastLaunch != "dirty tree at the reviewer's launch" {
		t.Errorf("a dirty pin counted: %+v %v", d.Review, err)
	}
}

// TestReviewerNotDeferred: a reviewer call that arrives with its result and no defer (the
// engine ran it, as several calls in one message would) gets no pin, Refresh is not run, and
// the fact stays none.
func TestReviewerNotDeferred(t *testing.T) {
	h := newHarness(t, "", true)
	ctx := context.Background()
	canonical, old := reviewCanonical(t, h)
	var pins atomic.Int32
	refresh := standInRefresh(canonical, "")
	withRefresh(h, func(ctx context.Context, s git.Session) (git.Pin, error) { pins.Add(1); return refresh(ctx, s) })
	reviewTurn(t, h, subagentFixture)
	if pins.Load() != 0 {
		t.Errorf("%d pins for a call the engine did not defer, want 0", pins.Load())
	}
	if n := count(t, h.st, `SELECT count(*) FROM audit_event WHERE kind = 'subagent-result' AND session_id = 's0001'
    AND details = '{"agent":"reviewer","model":"none","result":"ok","tool":"Agent","toolUseId":"toolu_sub_agent","turn":"1"}'`); n != 1 {
		t.Errorf("%d reviewer rows without a commit, want 1", n)
	}
	if d, err := h.svc.Inspect(ctx, "s0001", false); err != nil || d.Review == nil || d.Review.Commit != old || d.Review.ToolUseID != nil {
		t.Errorf("a review the engine did not defer counted: %+v %v", d.Review, err)
	}
}

// TestReviewFactErroredResult: a reviewer whose Agent result is an error writes its
// subagent-result row as an error, and the fact stays none.
func TestReviewFactErroredResult(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{engine.FixtureFile, "turn-1.jsonl"} {
		b, err := os.ReadFile(filepath.Join(subagentFixture, name))
		if err != nil {
			t.Fatal(err)
		}
		if name != engine.FixtureFile {
			from := `"tool_use_id":"toolu_sub_agent","content":[{"type":"text","text":"No findings."}]`
			if !strings.Contains(string(b), from) {
				t.Fatal("the fixture's Agent result changed shape")
			}
			b = []byte(strings.Replace(string(b), from, `"tool_use_id":"toolu_sub_agent","is_error":true,"content":"the subagent failed"`, 1))
		}
		if err := os.WriteFile(filepath.Join(dir, name), b, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	h := newHarness(t, "", true)
	ctx := context.Background()
	_, old := reviewCanonical(t, h)
	if _, err := h.svc.Start(ctx, StartRequest{Channel: "100", As: "1001", Fixture: dir}); err != nil {
		t.Fatal(err)
	}
	if _, err := h.svc.Submit(ctx, SendRequest{Session: "s0001", As: "1001", Text: "review it"}); err != nil {
		t.Fatal(err)
	}
	waitTurn(t, h.svc, "s0001", 1)
	if n := count(t, h.st, `SELECT count(*) FROM audit_event WHERE kind = 'subagent-result' AND details LIKE '%"result":"error"%'`); n != 1 {
		t.Errorf("%d errored subagent-result rows, want 1", n)
	}
	if d, err := h.svc.Inspect(ctx, "s0001", false); err != nil || d.Review == nil || d.Review.Commit != old || d.Review.ToolUseID != nil {
		t.Errorf("an errored review counted: %+v %v", d.Review, err)
	}
}
