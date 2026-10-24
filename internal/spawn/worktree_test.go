// SPDX-License-Identifier: GPL-3.0-or-later

package spawn

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jasondorjeshort/broodforge/internal/git/gittest"
)

func worktreeFixture(t *testing.T) (*SameAccount, Worktree) {
	t.Helper()
	root := t.TempDir()
	canon, _ := gittest.Project(t, root, "p")
	install := filepath.Join(root, "install")
	s, err := NewSameAccount(Config{Backend: NameSameAccount, Install: install, Binary: filepath.Join(install, "bin", "broodforge")})
	if err != nil {
		t.Fatal(err)
	}
	base := strings.TrimSpace(gittest.Git(t, canon, "rev-parse", "main"))
	return s, Worktree{Canonical: canon, Session: filepath.ToSlash(filepath.Join(root, "sessions", "s0001")),
		BaseBranch: "main", Base: base, Branch: "broodforge/test/s0001"}
}

func TestWorktreeCreateBundleRemove(t *testing.T) {
	s, w := worktreeFixture(t)
	if err := s.WorktreeCreate(w); err != nil {
		t.Fatal(err)
	}
	tree := w.Tree()
	if got := strings.TrimSpace(gittest.Git(t, tree, "symbolic-ref", "--short", "HEAD")); got != w.Branch {
		t.Errorf("HEAD %q, want %q", got, w.Branch)
	}
	if got := strings.TrimSpace(gittest.Git(t, tree, "config", "core.hooksPath")); got != s.HooksDir() {
		t.Errorf("core.hooksPath %q, want %q", got, s.HooksDir())
	}
	if _, err := os.Stat(filepath.Join(tree, ".git", "objects", "info", "alternates")); err != nil {
		t.Errorf("not a shared clone: %v", err)
	}
	if err := s.WorktreeCreate(w); err == nil || !strings.Contains(err.Error(), "exists") {
		t.Errorf("second create: %v, want exists", err)
	}

	b, err := s.WorktreeBundle(w)
	if err != nil || b.Commits != 0 || b.Path != "" {
		t.Fatalf("bundle with no commits: %+v %v", b, err)
	}
	gittest.Commit(t, gittest.Driver, nil, tree, "a.txt", "a\n", "one")
	gittest.Commit(t, gittest.Driver, nil, tree, "b.txt", "b\n", "two")
	// The count alone writes no bundle (ADR 0026: a held session's notice).
	if n, err := s.WorktreeCommits(w); err != nil || n != 2 {
		t.Fatalf("commits: %d %v", n, err)
	}
	if _, err := os.Stat(w.BundlePath()); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("WorktreeCommits wrote a bundle: %v", err)
	}
	b, err = s.WorktreeBundle(w)
	if err != nil || b.Commits != 2 || b.Path != w.BundlePath() {
		t.Fatalf("bundle: %+v %v", b, err)
	}
	if heads := gittest.Git(t, w.Canonical, "bundle", "list-heads", b.Path); !strings.Contains(heads, "refs/heads/"+w.Branch) {
		t.Errorf("bundle heads %q lack the session branch", heads)
	}

	if err := s.WorktreeRemove(w.Session); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{tree, w.BundlePath()} {
		if _, err := os.Stat(p); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("%s survives removal: %v", p, err)
		}
	}
	if err := s.WorktreeRemove(w.Session); err != nil {
		t.Errorf("second remove: %v", err)
	}
	if _, err := s.WorktreeCommits(w); err == nil {
		t.Error("commits of a removed tree: no error")
	}
}

func TestWorktreeCreateFailureLeavesNoTree(t *testing.T) {
	s, w := worktreeFixture(t)
	w.BaseBranch = "no-such-branch"
	err := s.WorktreeCreate(w)
	var se *Error
	if !errors.As(err, &se) || se.Code != CodeWorktree || !strings.Contains(se.Message, "git clone") {
		t.Fatalf("create from a missing branch: %v", err)
	}
	if _, err := os.Stat(w.Tree()); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("tree left behind: %v", err)
	}
}

func TestWorktreeRefusesOtherDirectories(t *testing.T) {
	s, w := worktreeFixture(t)
	root := filepath.Dir(filepath.Dir(w.Session))
	for _, dir := range []string{"sessions/s0001", filepath.Join(root, "harness", "s0001"), filepath.Join(root, "sessions", "x"),
		filepath.Join(root, "sessions", "s01"), root} {
		if err := s.WorktreeRemove(dir); err == nil {
			t.Errorf("remove %q: no refusal", dir)
		}
		w2 := w
		w2.Session = dir
		if err := s.WorktreeCreate(w2); err == nil {
			t.Errorf("create %q: no refusal", dir)
		}
		if _, err := s.WorktreeBundle(w2); err == nil {
			t.Errorf("bundle %q: no refusal", dir)
		}
		if _, err := s.WorktreeCommits(w2); err == nil {
			t.Errorf("commits %q: no refusal", dir)
		}
	}
}

func TestHeldBackendRefusesWorktrees(t *testing.T) {
	var b Backend = LinuxSudoBwrap{}
	if err := b.WorktreeCreate(Worktree{}); err == nil || !strings.Contains(err.Error(), "held") {
		t.Errorf("create: %v", err)
	}
	if _, err := b.WorktreeBundle(Worktree{}); err == nil {
		t.Error("bundle: no refusal")
	}
	if err := b.WorktreeRemove(""); err == nil {
		t.Error("remove: no refusal")
	}
	if _, err := b.WorktreeCommits(Worktree{}); err == nil || !strings.Contains(err.Error(), "held") {
		t.Errorf("commits: %v", err)
	}
}

// WorktreePrePush is true only while the tree's core.hooksPath is HooksDir and the hook is there.
func TestWorktreePrePush(t *testing.T) {
	s, w := worktreeFixture(t)
	if err := s.WorktreeCreate(w); err != nil {
		t.Fatal(err)
	}
	check := func(want bool, why string) {
		t.Helper()
		got, err := s.WorktreePrePush(w.Session)
		if err != nil || got != want {
			t.Errorf("%s: WorktreePrePush %v %v, want %v", why, got, err, want)
		}
	}
	check(false, "no pre-push under HooksDir")
	if err := os.MkdirAll(s.HooksDir(), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(s.HooksDir(), "pre-push"), []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	check(true, "hooks path set, hook present")
	gittest.Git(t, w.Tree(), "config", "core.hooksPath", filepath.ToSlash(t.TempDir()))
	check(false, "hooks path elsewhere")
	// Whitespace around the bot's directory: git runs no hook from there, so it is missing.
	for _, v := range []string{s.HooksDir() + " ", " " + s.HooksDir(), s.HooksDir() + "\n", s.HooksDir() + "\t"} {
		gittest.Git(t, w.Tree(), "config", "core.hooksPath", v)
		check(false, fmt.Sprintf("hooks path %q", v))
	}
	gittest.Git(t, w.Tree(), "config", "core.hooksPath", s.HooksDir())
	check(true, "hooks path set again")
	gittest.Git(t, w.Tree(), "config", "--unset", "core.hooksPath")
	check(false, "hooks path unset")
	if _, err := s.WorktreePrePush("sessions/s0001"); err == nil {
		t.Error("a relative session directory: no refusal")
	}
	if _, err := (LinuxSudoBwrap{}).WorktreePrePush(w.Session); err == nil || !strings.Contains(err.Error(), "held") {
		t.Errorf("held backend: %v", err)
	}
}

func TestAskpassAndHooksUnderShare(t *testing.T) {
	s, _ := worktreeFixture(t)
	share := filepath.ToSlash(filepath.Join(s.install, "share"))
	if s.HooksDir() != share+"/hooks" {
		t.Errorf("HooksDir %q", s.HooksDir())
	}
	if a := s.Askpass(); a != share+"/askpass.sh" && a != share+"/askpass.cmd" {
		t.Errorf("Askpass %q", a)
	}
}

// link makes at a symbolic link to target, or on Windows without the privilege a junction.
func link(t *testing.T, target, at string) {
	t.Helper()
	if err := os.Symlink(target, at); err == nil {
		return
	}
	if out, err := exec.Command("cmd", "/c", "mklink", "/J", filepath.FromSlash(at), filepath.FromSlash(target)).CombinedOutput(); err != nil {
		t.Fatalf("mklink /J: %v %s", err, out)
	}
}

// victim makes <dir>/tree/keep.txt, which no removal may reach.
func victim(t *testing.T, dir string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, "tree"), 0o750); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "tree", "keep.txt")
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestWorktreeRemoveRefusesDotDot(t *testing.T) {
	s, w := worktreeFixture(t)
	other := filepath.Join(t.TempDir(), "sessions", "s0009")
	keep := victim(t, other)
	rel, err := filepath.Rel(filepath.FromSlash(w.Session), other)
	if err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{w.Session + "/" + filepath.ToSlash(rel), w.Session + "/", w.Session + "/./"} {
		if err := s.WorktreeRemove(dir); err == nil {
			t.Errorf("remove %q: no refusal", dir)
		}
	}
	if _, err := os.Stat(keep); err != nil {
		t.Errorf("the victim is gone: %v", err)
	}
}

func TestWorktreeRemoveRefusesALinkedSession(t *testing.T) {
	s, w := worktreeFixture(t)
	keep := victim(t, t.TempDir())
	if err := os.MkdirAll(filepath.Dir(w.Session), 0o750); err != nil {
		t.Fatal(err)
	}
	link(t, filepath.Dir(filepath.Dir(keep)), w.Session)
	if err := s.WorktreeRemove(w.Session); err == nil || !strings.Contains(err.Error(), "is a link") {
		t.Errorf("remove through a linked session directory: %v", err)
	}
	if _, err := os.Stat(keep); err != nil {
		t.Errorf("the victim is gone: %v", err)
	}
}

func TestWorktreeRemoveTakesALinkedTreeAsTheLink(t *testing.T) {
	s, w := worktreeFixture(t)
	keep := victim(t, t.TempDir())
	if err := os.MkdirAll(w.Session, 0o750); err != nil {
		t.Fatal(err)
	}
	link(t, filepath.Dir(keep), filepath.Join(w.Session, "tree"))
	if err := s.WorktreeRemove(w.Session); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(keep); err != nil {
		t.Errorf("the victim is gone: %v", err)
	}
	if _, err := os.Lstat(w.Tree()); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the link survives: %v", err)
	}
}
