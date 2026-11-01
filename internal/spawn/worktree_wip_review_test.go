// SPDX-License-Identifier: GPL-3.0-or-later

package spawn

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jasondorjeshort/broodforge/internal/git/gittest"
)

// TestWorktreeWIPLeavesSubagentWorktrees: a subagent's worktree inside the tree is neither dirt nor a gitlink in the WIP commit.
func TestWorktreeWIPLeavesSubagentWorktrees(t *testing.T) {
	s, w := worktreeFixture(t)
	if err := s.WorktreeCreate(w); err != nil {
		t.Fatal(err)
	}
	tree := w.Tree()
	gittest.Git(t, tree, "worktree", "add", "--quiet", "-b", "agent1", ".claude/worktrees/agent1", "HEAD")
	if dirty, err := s.WorktreeDirty(w); err != nil || dirty {
		t.Errorf("a tree holding only a subagent worktree: dirty %v, %v", dirty, err)
	}
	if err := os.WriteFile(filepath.Join(tree, "w.txt"), []byte("wip\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	sha, err := s.WorktreeWIP(w, WIP{Name: gittest.Driver.Name, Email: gittest.Driver.Email, Message: "WIP\n"})
	if err != nil || sha == "" {
		t.Fatalf("WorktreeWIP: %q %v", sha, err)
	}
	if files := strings.TrimSpace(gittest.Git(t, tree, "show", "--name-only", "--format=", sha)); files != "w.txt" {
		t.Errorf("WIP files %q, want w.txt alone", files)
	}
}
