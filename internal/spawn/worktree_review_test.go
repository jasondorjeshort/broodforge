// SPDX-License-Identifier: GPL-3.0-or-later

package spawn

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jasondorjeshort/broodforge/internal/git/gittest"
)

// TestWorktreeBundleStatus: the bundle's Status is the tree's `status --porcelain`, a modified
// tracked file and an untracked one both listed, "" for a clean tree; a context already ended
// kills the git and writes no bundle (the reviewer pin, ADR 0028 Decision 3).
func TestWorktreeBundleStatus(t *testing.T) {
	s, w := worktreeFixture(t)
	if err := s.WorktreeCreate(w); err != nil {
		t.Fatal(err)
	}
	tree := w.Tree()
	gittest.Commit(t, gittest.Driver, nil, tree, "a.txt", "a\n", "one")
	b, err := s.WorktreeBundle(context.Background(), w)
	if err != nil || b.Commits != 1 || b.Status != "" {
		t.Fatalf("clean tree: %+v %v", b, err)
	}
	if err := os.WriteFile(filepath.Join(tree, "a.txt"), []byte("changed\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tree, "new.txt"), []byte("new\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	b, err = s.WorktreeBundle(context.Background(), w)
	if err != nil || b.Commits != 1 || !strings.Contains(b.Status, " M a.txt") || !strings.Contains(b.Status, "?? new.txt") {
		t.Fatalf("dirty tree: %+v %v", b, err)
	}
	if err := os.Remove(w.BundlePath()); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	if _, err := s.WorktreeBundle(ctx, w); err == nil || !strings.Contains(err.Error(), "deadline") {
		t.Errorf("ended context: %v, want the deadline", err)
	}
	if _, err := os.Stat(w.BundlePath()); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("an ended context wrote a bundle: %v", err)
	}
}
