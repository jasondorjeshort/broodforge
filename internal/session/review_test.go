// SPDX-License-Identifier: GPL-3.0-or-later

package session

import (
	"context"
	"os"
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

// TestReviewFact: `review:` is the reviewer Agent call whose result's audit row
// (subagent-result) is after the committer time of refs/sessions/<id>; a later commit (as of the
// next bundle) makes it none since that commit; a moved canonical is an error, never hidden; a
// session whose ref is gone has no fact.
func TestReviewFact(t *testing.T) {
	h := newHarness(t, "", true)
	ctx := context.Background()
	canonical, old := reviewCanonical(t, h)
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
    AND details = '{"agent":"reviewer","result":"ok","tool":"Agent","toolUseId":"toolu_sub_agent","turn":"1"}'`); n != 1 {
		t.Errorf("%d subagent-result rows for the reviewer, want 1", n)
	}

	// A commit after the review, fetched by the next bundle: no review since it.
	later := gittest.Commit(t, gittest.Driver, []string{"GIT_COMMITTER_DATE=2099-01-01T00:00:00Z"}, canonical, "b.txt", "b", "after")
	gittest.Git(t, canonical, "update-ref", git.SessionRef("s0001"), later)
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

// TestReviewFactExclusions: each row below, alone, leaves `review: none since`: a result before
// the commit or in its own second, a non-reviewer subagent, an errored result, a denied call, a
// reviewer allowed and then interrupted (a decision row, no result), a result with no agent
// field, a reviewer result of another tool. Then one reviewer result after the commit counts.
func TestReviewFactExclusions(t *testing.T) {
	h := newHarness(t, "", true)
	ctx := context.Background()
	_, old := reviewCanonical(t, h)
	if _, err := h.svc.Start(ctx, StartRequest{Channel: "100", As: "1001"}); err != nil {
		t.Fatal(err)
	}
	row := func(when, kind, details string) string {
		return `INSERT INTO audit_event (time, kind, actor, session_id, details) VALUES ('` + when + `', '` + kind +
			`', 'instance', 's0001', '` + details + `')`
	}
	after := "2025-01-01T00:00:00Z"
	for _, c := range []struct{ name, stmt string }{
		{"before the commit", row("2019-12-31T23:59:59Z", "subagent-result", `{"agent":"reviewer","result":"ok","tool":"Agent","toolUseId":"t1"}`)},
		{"the commit's own second", row("2020-01-01T00:00:00Z", "subagent-result", `{"agent":"reviewer","result":"ok","tool":"Agent","toolUseId":"t2"}`)},
		{"not the reviewer", row(after, "subagent-result", `{"agent":"builder","result":"ok","tool":"Agent","toolUseId":"t3"}`)},
		{"an errored result", row(after, "subagent-result", `{"agent":"reviewer","result":"error","tool":"Agent","toolUseId":"t4"}`)},
		{"a denied call", row(after, "permission-decision", `{"agent":"reviewer","decision":"deny","tool":"Agent","toolUseId":"t5"}`)},
		{"allowed, then interrupted", row(after, "permission-decision", `{"agent":"reviewer","decision":"allow","tool":"Agent","toolUseId":"t6"}`)},
		{"no agent field", row(after, "subagent-result", `{"result":"ok","tool":"Agent","toolUseId":"t7"}`)},
		{"not the subagent tool", row(after, "subagent-result", `{"agent":"reviewer","result":"ok","tool":"Read","toolUseId":"t8"}`)},
	} {
		exec(t, h.st, c.stmt)
		if d, err := h.svc.Inspect(ctx, "s0001", false); err != nil || d.Review == nil || d.Review.Commit != old || d.Review.ToolUseID != nil {
			t.Errorf("%s counted: %+v %v", c.name, d.Review, err)
		}
	}
	exec(t, h.st, row(after, "subagent-result", `{"agent":"reviewer","result":"ok","tool":"Task","toolUseId":"t9"}`))
	if d, err := h.svc.Inspect(ctx, "s0001", false); err != nil || d.Review == nil || d.Review.ToolUseID == nil || *d.Review.ToolUseID != "t9" {
		t.Errorf("the reviewer's result: %+v %v", d.Review, err)
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
