// SPDX-License-Identifier: GPL-3.0-or-later

package cli_test

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jasondorjeshort/broodforge/internal/cli"
	"github.com/jasondorjeshort/broodforge/internal/cli/clitest"
	"github.com/jasondorjeshort/broodforge/internal/git/gittest"
)

// The P15 Proof (08-implementation-plan-session-workflow.md): a session tree refuses every push
// with the one line and exit 1, through the binary and, with bin/ away, through the script
// alone; worktree check reports the hook. Every push runs here through os/exec against the
// test's own canonical under its temporary directory, never from a shell.

// sessionGitEnv is a session's git environment: no inherited GIT_* or BROODFORGE_* name, the
// driver's identity, then extra.
func sessionGitEnv(extra ...string) []string {
	var env []string
	for _, kv := range gittest.Env(gittest.Driver) {
		if !strings.HasPrefix(strings.ToUpper(kv), "BROODFORGE_") {
			env = append(env, kv)
		}
	}
	return append(env, extra...)
}

type gitRun struct {
	code           int
	stdout, stderr string
}

// pushFrom runs `git -C tree args...` as the session would and returns its exit and output.
func pushFrom(t *testing.T, tree string, env []string, args ...string) gitRun {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", tree}, args...)...)
	cmd.Env = env
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	r := gitRun{stdout: stdout.String(), stderr: stderr.String()}
	var ee *exec.ExitError
	switch {
	case err == nil:
	case errors.As(err, &ee):
		r.code = ee.ExitCode()
	default:
		t.Fatalf("git %q: %v", args, err)
	}
	t.Logf("$ git -C <tree> %s\nexit %d\nstdout: %q\nstderr: %q", strings.Join(args, " "), r.code, r.stdout, r.stderr)
	return r
}

// wantRefused checks a refused push: exit 1, stderr's first line the refusal, git's own
// failed-to-push line after it.
func wantRefused(t *testing.T, r gitRun, line string) {
	t.Helper()
	first, rest, _ := strings.Cut(strings.ReplaceAll(r.stderr, "\r\n", "\n"), "\n")
	if r.code != 1 || first != line || !strings.Contains(rest, "error: failed to push some refs") {
		t.Errorf("push: exit %d, stderr %q; want exit 1, first line %q, then git's failed-to-push line", r.code, r.stderr, line)
	}
}

func TestProofPrePushRefusesEveryPush(t *testing.T) {
	p := newWorktreeProof(t)
	s := p.start(t, "s0001")
	tree := filepath.ToSlash(filepath.Join(s.Dir(), "tree"))
	head := gittest.Commit(t, gittest.Driver, nil, tree, "README", "base\nmore\n", "Add a line")
	env := sessionGitEnv("BROODFORGE_SESSION=s0001")
	line := cli.PrePushLine("s0001")
	refs := func() string { return gittest.Git(t, p.canonical, "for-each-ref") }
	before := refs()

	// Through the binary: to origin and to the canonical's path, named directly.
	wantRefused(t, pushFrom(t, tree, env, "push", "origin", "HEAD"), line)
	wantRefused(t, pushFrom(t, tree, env, "push", p.canonical, "HEAD"), line)
	// And to any other path: the canonical's own bare remote.
	wantRefused(t, pushFrom(t, tree, env, "push", p.remote, "HEAD"), line)
	// An install without bin/ (its own layout, so nothing in use is renamed): the script prints
	// the same line itself. The command-line hooks path stands in for the tree's.
	noBin := filepath.ToSlash(filepath.Join(p.root, "nobin", "share", "hooks"))
	copyHook(t, noBin)
	wantRefused(t, pushFrom(t, tree, env, "-c", "core.hooksPath="+noBin, "push", "origin", "HEAD"), line)
	wantRefused(t, pushFrom(t, tree, sessionGitEnv(), "-c", "core.hooksPath="+noBin, "push", "origin", "HEAD"), cli.PrePushLine(""))
	if after := refs(); after != before {
		t.Errorf("the canonical changed under refused pushes:\n%s\nwas\n%s", after, before)
	}
	if b := gittest.Git(t, p.remote, "branch", "--list", "broodforge/*"); b != "" {
		t.Errorf("the bare remote took a refused push: %q", b)
	}

	root := []string{"--root", p.root}
	want(t, clitest.Run(t, nil, append(root, "worktree", "check", "s0001")...), 0, "ok  pre-push\n"+head+"  ok\nok  1\n", "")
	// --json: the hook row has its own shape, never read as a commit line.
	want(t, clitest.Run(t, nil, append(root, "--json", "worktree", "check", "s0001")...), 0,
		`{"hook":"pre-push","status":"ok"}`+"\n"+`{"commit":"`+head+`","status":"ok"}`+"\n"+`{"result":"ok","commits":1}`+"\n", "")

	// The lift (the 0025 companion's lift column): --no-verify skips the hook and the push
	// lands. A refusal with a message, not the boundary; the boundary is the droplet's
	// read-only canonical and the absent credential (ADR 0025, rule 4).
	r := pushFrom(t, tree, env, "push", "--no-verify", "origin", "HEAD")
	branch := s.Branch()
	if r.code != 0 || strings.TrimSpace(gittest.Git(t, p.canonical, "rev-parse", "refs/heads/"+branch)) != head {
		t.Errorf("push --no-verify: exit %d, %q; the canonical lacks %s at %s", r.code, r.stderr, branch, head)
	}
	// The other lift: an empty hooks path on the command line, here to the bare remote.
	r = pushFrom(t, tree, env, "-c", "core.hooksPath=", "push", p.remote, "HEAD")
	if r.code != 0 || strings.TrimSpace(gittest.Git(t, p.remote, "rev-parse", "refs/heads/"+branch)) != head {
		t.Errorf("push -c core.hooksPath=: exit %d, %q; the remote lacks %s at %s", r.code, r.stderr, branch, head)
	}

	// The session owns its tree's configuration: with core.hooksPath unset the hook is missing.
	gittest.Git(t, tree, "config", "--unset", "core.hooksPath")
	want(t, clitest.Run(t, nil, append(root, "worktree", "check", "s0001")...), 1, "missing  pre-push\n"+head+"  ok\n",
		"error: pre-push-missing: the tree's core.hooksPath is not "+p.backend.HooksDir()+" with its pre-push hook; handoff still runs\n")
}

// git-pre-push itself: the line on stderr, nothing on stdout, exit 1, in and out of a session.
func TestGitPrePushRefuses(t *testing.T) {
	want(t, clitest.Run(t, []string{"BROODFORGE_SESSION=s0007"}, "git-pre-push"), 1, "",
		"Broodforge: session s0007 does not push. Commit in this tree; Broodforge pushes at the handoff when the session ends.\n")
	want(t, clitest.Run(t, nil, "git-pre-push"), 1, "", cli.PrePushLine("")+"\n")
}

// copyHook writes the repository's share/hooks/pre-push into dir, which it creates.
func copyHook(t *testing.T, dir string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	src, err := os.ReadFile(filepath.Join("..", "..", "share", "hooks", "pre-push"))
	if err != nil {
		t.Fatal(err)
	}
	hook := filepath.Join(dir, "pre-push")
	if err := os.WriteFile(hook, src, 0o755); err != nil {
		t.Fatal(err)
	}
	return hook
}

// runHook runs the hook script as git would, with BROODFORGE_SESSION set to id unless it is "",
// and checks exit 1, nothing on stdout and stderr's first line the refusal.
func runHook(t *testing.T, hook, id, why string) string {
	t.Helper()
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("no sh on PATH")
	}
	env := sessionGitEnv()
	if id != "" {
		env = append(env, "BROODFORGE_SESSION="+id)
	}
	cmd := exec.Command(sh, filepath.ToSlash(hook), "origin", "file:///nowhere")
	cmd.Env = env
	cmd.Stdin = strings.NewReader("")
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err = cmd.Run()
	t.Logf("$ sh %s origin file:///nowhere (%s, session %q)\nerr: %v\nstdout: %q\nstderr: %q", hook, why, id, err, stdout.String(), stderr.String())
	first, _, _ := strings.Cut(stderr.String(), "\n")
	var ee *exec.ExitError
	if !errors.As(err, &ee) || ee.ExitCode() != 1 || stdout.Len() != 0 || first != cli.PrePushLine(id) {
		t.Errorf("hook %s (session %q): %v %q %q", why, id, err, stdout.String(), stderr.String())
	}
	return stderr.String()
}

// The pre-push hook with no binary beside it: the script's own copy of the line, which must be
// cli.PrePushRefusal's, and exit 1 on that path too, unlike commit-msg.
func TestPrePushHookWithoutTheBinary(t *testing.T) {
	hook := copyHook(t, filepath.Join(t.TempDir(), "share", "hooks"))
	for _, id := range []string{"s0001", ""} {
		if got := runHook(t, hook, id, "without the binary"); got != cli.PrePushLine(id)+"\n" {
			t.Errorf("without the binary: stderr %q, want the line alone", got)
		}
	}
}

// A binary that runs but prints nothing, or that cannot start: the script prints the line
// first itself, whatever the binary did, and still exits 1.
func TestPrePushHookWhenTheBinaryFails(t *testing.T) {
	for _, c := range []struct{ why, name, content string }{
		{"silent binary", "broodforge", "#!/bin/sh\n: > \"$(dirname \"$0\")/ran\"\nexit 0\n"},
		{"binary that cannot start", "broodforge.exe", "\x00\x01 not a program\n"},
	} {
		bf := t.TempDir()
		hook := copyHook(t, filepath.Join(bf, "share", "hooks"))
		bin := filepath.Join(bf, "bin")
		if err := os.MkdirAll(bin, 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(bin, c.name), []byte(c.content), 0o755); err != nil {
			t.Fatal(err)
		}
		runHook(t, hook, "s0001", c.why)
		if _, err := os.Stat(filepath.Join(bin, "ran")); c.name == "broodforge" && err != nil {
			t.Errorf("%s: the hook never ran it: %v", c.why, err)
		}
	}
}
