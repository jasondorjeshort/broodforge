// SPDX-License-Identifier: GPL-3.0-or-later

package git_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jasondorjeshort/broodforge/internal/git"
	"github.com/jasondorjeshort/broodforge/internal/git/gittest"
)

// TestReviewPin: PinReview leaves refs/sessions/<id> where Refresh would and returns its sha
// with the tree's tracked changes: none for a clean tree or one with only an untracked file, the
// porcelain line of a modified tracked file, and of a deleted one (ADR 0028 Decision 3).
func TestReviewPin(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	ref := func() string {
		return strings.TrimSpace(gittest.Git(t, f.sess.Canonical, "rev-parse", git.SessionRef(f.sess.ID)))
	}
	base := ref()
	p, err := git.PinReview(ctx, f.backend, f.sess)
	if err != nil || p.Commit != base || p.Dirty != "" {
		t.Fatalf("no commits: %+v %v", p, err)
	}
	one := gittest.Commit(t, gittest.Driver, nil, f.tree, "a.txt", "a\n", msg("one", opus))
	if err := os.WriteFile(filepath.Join(f.tree, "untracked.txt"), []byte("u\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if p, err = git.PinReview(ctx, f.backend, f.sess); err != nil || p.Commit != one || ref() != one || p.Dirty != "" {
		t.Fatalf("one commit, an untracked file: %+v %v", p, err)
	}
	if err := os.WriteFile(filepath.Join(f.tree, "a.txt"), []byte("changed\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if p, err = git.PinReview(ctx, f.backend, f.sess); err != nil || p.Commit != one || p.Dirty != " M a.txt" {
		t.Fatalf("a modified file: %+v %v", p, err)
	}
	if err := os.Remove(filepath.Join(f.tree, "a.txt")); err != nil {
		t.Fatal(err)
	}
	if p, err = git.PinReview(ctx, f.backend, f.sess); err != nil || p.Dirty != " D a.txt" {
		t.Fatalf("a deleted file: %+v %v", p, err)
	}
}

// TestReviewPinWaitsOnTheBundleLock: a pin while the session's bundle lock is held waits, and
// gives up with the context; another session's lock does not hold it.
func TestReviewPinWaitsOnTheBundleLock(t *testing.T) {
	f := newFixture(t)
	unlock, err := git.LockBundle(context.Background(), f.sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	var ge *git.Error
	if _, err := git.PinReview(ctx, f.backend, f.sess); !errors.As(err, &ge) || !strings.Contains(ge.Message, "bundle lock") {
		t.Errorf("a held lock: %v", err)
	}
	other, err := git.LockBundle(context.Background(), "s0002")
	if err != nil {
		t.Fatal(err)
	}
	other()
	unlock()
	if _, err := git.PinReview(context.Background(), f.backend, f.sess); err != nil {
		t.Errorf("after the lock went: %v", err)
	}
}

// TestTrackedChanges: untracked and ignored lines go, every other porcelain line stays.
func TestTrackedChanges(t *testing.T) {
	in := " M a.go\n?? new.txt\nD  gone.go\n!! build/\nR  old -> new\nUU both.go\n\n"
	if got, want := git.TrackedChanges(in), " M a.go\nD  gone.go\nR  old -> new\nUU both.go"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got := git.TrackedChanges("?? only.txt\n"); got != "" {
		t.Errorf("untracked only: %q", got)
	}
}
