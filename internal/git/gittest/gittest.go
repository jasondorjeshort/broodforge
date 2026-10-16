// SPDX-License-Identifier: GPL-3.0-or-later

// Package gittest makes fixture repositories for tests: a canonical checkout with a local bare
// remote, and commits under an explicit identity. Every git command runs through os/exec with
// the four GIT_AUTHOR_* and GIT_COMMITTER_* variables set, never from a shell
// (07-spike-plan.md, Fixture commits), and only in the test's own temporary directories.
package gittest

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Who is a commit's author and committer.
type Who struct {
	Name  string
	Email string
}

// Driver is the identity fixtures commit as unless told otherwise.
var Driver = Who{"Test Driver", "driver@example.com"}

// Env is the environment of a fixture git command: the test's own minus every inherited GIT_*
// name (a surrounding session's GIT_DIR or GIT_CONFIG_PARAMETERS would aim it elsewhere), then
// the four identity variables and the extras.
func Env(who Who, extra ...string) []string {
	var env []string
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(strings.ToUpper(kv), "GIT_") {
			env = append(env, kv)
		}
	}
	env = append(env, "GIT_AUTHOR_NAME="+who.Name, "GIT_AUTHOR_EMAIL="+who.Email,
		"GIT_COMMITTER_NAME="+who.Name, "GIT_COMMITTER_EMAIL="+who.Email, "GIT_TERMINAL_PROMPT=0")
	return append(env, extra...)
}

// Git runs `git -C dir args...` as Driver and returns stdout, failing the test on an error.
func Git(t testing.TB, dir string, args ...string) string {
	t.Helper()
	return GitAs(t, Driver, nil, dir, args...)
}

// GitAs runs `git -C dir args...` as who with extra environment entries.
func GitAs(t testing.TB, who Who, extra []string, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = Env(who, extra...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("git -C %s %s: %v\n%s", dir, strings.Join(args, " "), err, stderr.String())
	}
	return stdout.String()
}

// Commit writes file in the repository at dir and commits it as who with the message; extra
// holds more environment entries (GIT_COMMITTER_DATE) and args more commit arguments
// (--author). It returns the new commit's hash.
func Commit(t testing.TB, who Who, extra []string, dir, file, content, msg string, args ...string) string {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, file), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	GitAs(t, who, extra, dir, "add", "--", file)
	GitAs(t, who, extra, dir, append([]string{"commit", "--quiet", "-m", msg}, args...)...)
	return strings.TrimSpace(GitAs(t, who, extra, dir, "rev-parse", "HEAD"))
}

// Project makes <root>/projects/<name>, a canonical checkout on main with one commit, and
// <root>/remote/<name>.git, a bare repository that is the canonical's origin and already holds
// main. It returns both paths with forward slashes.
func Project(t testing.TB, root, name string) (canonical, remote string) {
	t.Helper()
	canonical = filepath.ToSlash(filepath.Join(root, "projects", name))
	remote = filepath.ToSlash(filepath.Join(root, "remote", name+".git"))
	for _, d := range []string{canonical, remote} {
		if err := os.MkdirAll(d, 0o750); err != nil {
			t.Fatal(err)
		}
	}
	Git(t, remote, "init", "--quiet", "--bare", "-b", "main")
	Git(t, canonical, "init", "--quiet", "-b", "main")
	Commit(t, Driver, nil, canonical, "README", "base\n", "base")
	Git(t, canonical, "remote", "add", "origin", remote)
	Git(t, canonical, "push", "--quiet", "origin", "main")
	return canonical, remote
}
