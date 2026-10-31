// SPDX-License-Identifier: GPL-3.0-or-later

package session

import (
	"context"
	"errors"
	"os"
	osexec "os/exec"
	"path/filepath"
	"strings"
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

// standInRefresh is git.Refresh for a canonical whose own main stands in for the session's
// tree: the "bundle" sets refs/sessions/s0001 to main's commit and returns it, as the bot's
// fetch would. It runs on the turn's goroutine too, so it fails by its error, never t.
func standInRefresh(canonical string) func(context.Context, git.Session) (string, error) {
	return func(ctx context.Context, s git.Session) (string, error) {
		run := func(args ...string) (string, error) {
			cmd := osexec.CommandContext(ctx, "git", append([]string{"-C", canonical}, args...)...)
			cmd.Env = gittest.Env(gittest.Driver)
			out, err := cmd.Output()
			return strings.TrimSpace(string(out)), err
		}
		sha, err := run("rev-parse", "--verify", "main^{commit}")
		if err != nil {
			return "", err
		}
		if _, err := run("update-ref", git.SessionRef(s.ID), sha); err != nil {
			return "", err
		}
		return sha, nil
	}
}

// withRefresh restarts the harness's service with refresh as its Config.Refresh.
func withRefresh(h *harness, refresh func(context.Context, git.Session) (string, error)) {
	h.cfg.Refresh = refresh
	h.svc = New(h.cfg)
}

// TestReviewFact: `review:` is the reviewer Agent call whose subagent-result row names the
// commit the bot bundled at its launch, while refs/sessions/<id> still holds it. A commit after
// the review, refreshed by `worktree check`, voids it, also one whose committer date is a year
// before the review (the case the committer-time rule missed); a reset back to the reviewed
// commit holds again; a moved canonical is an error, never hidden; a session whose ref is gone
// has no fact.
func TestReviewFact(t *testing.T) {
	h := newHarness(t, "", true)
	ctx := context.Background()
	canonical, old := reviewCanonical(t, h)
	refresh := standInRefresh(canonical)
	withRefresh(h, refresh)
	worktreeCheck := func() string {
		t.Helper()
		sha, err := refresh(ctx, git.Session{ID: "s0001"})
		if err != nil {
			t.Fatal(err)
		}
		return sha
	}
	if _, err := h.svc.Start(ctx, StartRequest{Channel: "100", As: "1001", Fixture: subagentFixture}); err != nil {
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
	if err != nil || d.Review == nil || d.Review.Commit != old || d.Review.ToolUseID == nil || *d.Review.ToolUseID != "toolu_sub_agent" {
		t.Fatalf("after the review: %+v %v", d.Review, err)
	}
	if n := count(t, h.st, `SELECT count(*) FROM audit_event WHERE kind = 'subagent-result' AND session_id = 's0001'
    AND details = '{"agent":"reviewer","commit":"`+old+`","result":"ok","tool":"Agent","toolUseId":"toolu_sub_agent","turn":"1"}'`); n != 1 {
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

// TestReviewFactRefreshFailed: a bundle that fails at the reviewer's launch leaves commitError
// in the subagent-result row and no commit, and the fact stays none.
func TestReviewFactRefreshFailed(t *testing.T) {
	h := newHarness(t, "", true)
	ctx := context.Background()
	_, old := reviewCanonical(t, h)
	withRefresh(h, func(context.Context, git.Session) (string, error) { return "", errors.New("refused-bundle: no tree") })
	if _, err := h.svc.Start(ctx, StartRequest{Channel: "100", As: "1001", Fixture: subagentFixture}); err != nil {
		t.Fatal(err)
	}
	if _, err := h.svc.Submit(ctx, SendRequest{Session: "s0001", As: "1001", Text: "review it"}); err != nil {
		t.Fatal(err)
	}
	waitTurn(t, h.svc, "s0001", 1)
	if n := count(t, h.st, `SELECT count(*) FROM audit_event WHERE kind = 'subagent-result' AND session_id = 's0001'
    AND details = '{"agent":"reviewer","commitError":"refused-bundle: no tree","result":"ok","tool":"Agent","toolUseId":"toolu_sub_agent","turn":"1"}'`); n != 1 {
		t.Errorf("%d subagent-result rows carrying the bundle's error, want 1", n)
	}
	if d, err := h.svc.Inspect(ctx, "s0001", false); err != nil || d.Review == nil || d.Review.Commit != old || d.Review.ToolUseID != nil {
		t.Errorf("a review whose bundle failed counted: %+v %v", d.Review, err)
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
