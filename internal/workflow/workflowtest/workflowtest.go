// SPDX-License-Identifier: GPL-3.0-or-later

// Package workflowtest makes canonical repositories from fixtures/workflow/ for tests: the tree
// copied into a temporary directory, edited by the test, committed on main through os/exec
// (gittest), never from a shell.
package workflowtest

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/jasondorjeshort/broodforge/internal/git/gittest"
)

// Fixture is the absolute path of fixtures/workflow/.
func Fixture(t testing.TB) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("workflowtest: no caller")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "..", "fixtures", "workflow")
}

// Canonical copies the fixture into a new temporary directory, lets edit change it (nil for
// none), and commits it on main as gittest.Driver. It returns the repository's path.
func Canonical(t testing.TB, edit func(dir string)) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "canonical")
	src := Fixture(t)
	err := filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		dst := filepath.Join(dir, rel)
		if d.IsDir() {
			return os.MkdirAll(dst, 0o755)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(dst, b, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	if edit != nil {
		edit(dir)
	}
	gittest.Git(t, dir, "init", "--quiet", "-b", "main")
	gittest.Git(t, dir, "add", "--all")
	gittest.Git(t, dir, "-c", "commit.gpgsign=false", "-c", "core.hooksPath="+os.DevNull, "commit", "--quiet", "-m", "fixture")
	return filepath.ToSlash(dir)
}

// Write writes text to rel under dir, making its directory.
func Write(t testing.TB, dir, rel, text string) {
	t.Helper()
	p := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}
