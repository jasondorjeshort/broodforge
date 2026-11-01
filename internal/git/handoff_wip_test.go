// SPDX-License-Identifier: GPL-3.0-or-later

package git_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jasondorjeshort/broodforge/internal/audit"
	"github.com/jasondorjeshort/broodforge/internal/git"
	"github.com/jasondorjeshort/broodforge/internal/git/gittest"
)

func write(t *testing.T, path, text string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestHandoffCommitsADirtyTree: the end's WIP commit of an edited file and an untracked one, by
// the driver with the required trailer, handed off as the session's one commit; the check passes it.
func TestHandoffCommitsADirtyTree(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	write(t, filepath.Join(f.tree, "new.txt"), "new\n")
	if n, err := git.Pending(ctx, f.backend, f.sess); err != nil || n != 1 {
		t.Errorf("Pending = %d, %v; want 1", n, err)
	}
	h := f.handoff(git.ModePush)
	h.CommitDirty = true
	out, err := h.Run(ctx, false)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out.Outcome != git.OutcomePushed || out.Commits != 1 || out.WIP == "" || !out.Removed {
		t.Fatalf("outcome %+v", out)
	}
	if len(out.Log) != 1 || out.Log[0].Subject != git.WIPSubject || out.Log[0].Hash != out.WIP {
		t.Errorf("log %+v", out.Log)
	}
	pushed := strings.TrimSpace(gittest.Git(t, f.remote, "rev-parse", "refs/heads/"+f.sess.Branch()))
	if pushed != out.WIP {
		t.Errorf("remote branch %s, WIP %s", pushed, out.WIP)
	}
	body := gittest.Git(t, f.remote, "log", "-1", "--format=%an <%ae>|%cn <%ce>|%B", out.WIP)
	if !strings.HasPrefix(body, driver.String()+"|"+driver.String()+"|"+git.WIPSubject+"\n\n"+opus.Trailer()) {
		t.Errorf("WIP commit %q", body)
	}
	if files := gittest.Git(t, f.remote, "show", "--name-only", "--format=", out.WIP); strings.TrimSpace(files) != "new.txt" {
		t.Errorf("WIP files %q", files)
	}
}

// TestHandoffCleanTreeMakesNoWIP: CommitDirty on a clean tree is the plain handoff.
func TestHandoffCleanTreeMakesNoWIP(t *testing.T) {
	f := newFixture(t)
	h := f.handoff(git.ModePush)
	h.CommitDirty = true
	out, err := h.Run(context.Background(), false)
	if err != nil || out.Outcome != git.OutcomeNone || out.WIP != "" || out.Commits != 0 {
		t.Fatalf("Run: %+v, %v", out, err)
	}
	if got := f.kinds(); len(got) != 0 {
		t.Errorf("audit %q", got)
	}
}

// TestHandoffWIPRefusedOffBranch: a HEAD off the session branch is handoff-refused-wip, the tree
// and its changes kept.
func TestHandoffWIPRefusedOffBranch(t *testing.T) {
	f := newFixture(t)
	gittest.Git(t, f.tree, "checkout", "--quiet", "-b", "elsewhere")
	write(t, filepath.Join(f.tree, "new.txt"), "new\n")
	h := f.handoff(git.ModePush)
	h.CommitDirty = true
	_, err := h.Run(context.Background(), false)
	var ge *git.Error
	if !errors.As(err, &ge) || ge.Code != git.CodeRefusedWIP {
		t.Fatalf("Run: %v", err)
	}
	if !exists(filepath.Join(f.tree, "new.txt")) {
		t.Error("the tree's file is gone")
	}
	if got := f.kinds(); len(got) != 1 || got[0] != "handoff-refused:" {
		t.Errorf("audit %q", got)
	}
}

// TestDiscard: one commit and a dirty tree end as two commits on refs/aborted/s0001, nothing
// pushed, the tree and refs/sessions/s0001 gone, one session-discarded row.
func TestDiscard(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	gittest.Commit(t, gittest.Driver, nil, f.tree, "a.txt", "a\n", msg("one", opus))
	write(t, filepath.Join(f.tree, "b.txt"), "b\n")
	out, err := f.handoff(git.ModePullRequest).Discard(ctx)
	if err != nil {
		t.Fatalf("Discard: %v", err)
	}
	ref := git.AbortedRef("s0001")
	if out.Outcome != git.OutcomeDiscarded || out.Commits != 2 || out.Ref != ref || out.WIP == "" || !out.Removed ||
		out.Refspec != git.SessionRef("s0001")+":"+ref || out.URL != nil || out.Request != nil {
		t.Fatalf("outcome %+v", out)
	}
	if tip := strings.TrimSpace(gittest.Git(t, f.sess.Canonical, "rev-parse", ref)); tip != out.WIP {
		t.Errorf("%s %s, WIP %s", ref, tip, out.WIP)
	}
	if refs := gittest.Git(t, f.sess.Canonical, "for-each-ref", git.SessionRef("s0001")); refs != "" {
		t.Errorf("refs/sessions/s0001 kept: %q", refs)
	}
	if heads := gittest.Git(t, f.remote, "for-each-ref", "refs/heads/broodforge/"); heads != "" {
		t.Errorf("pushed: %q", heads)
	}
	if exists(f.tree) {
		t.Error("the tree is kept")
	}
	if got := f.kinds(); len(got) != 1 || got[0] != string(audit.KindSessionDiscarded)+":" {
		t.Fatalf("audit %q", got)
	}
	var d map[string]string
	if err := json.Unmarshal(f.events[0].Details, &d); err != nil || d["commits"] != "2" || d["ref"] != ref || d["wip"] != out.WIP {
		t.Errorf("details %s, %v", f.events[0].Details, err)
	}
}

// TestDiscardNothing: a clean session with no commits writes no ref.
func TestDiscardNothing(t *testing.T) {
	f := newFixture(t)
	out, err := f.handoff(git.ModePush).Discard(context.Background())
	if err != nil || out.Outcome != git.OutcomeDiscarded || out.Commits != 0 || out.Ref != "" || out.Refspec != "" || !out.Removed {
		t.Fatalf("Discard: %+v, %v", out, err)
	}
	if refs := gittest.Git(t, f.sess.Canonical, "for-each-ref", "refs/aborted/"); refs != "" {
		t.Errorf("aborted refs: %q", refs)
	}
	var d map[string]string
	if len(f.events) != 1 || json.Unmarshal(f.events[0].Details, &d) != nil || d["commits"] != "0" || d["ref"] != "" {
		t.Errorf("audit %+v", f.events)
	}
}
