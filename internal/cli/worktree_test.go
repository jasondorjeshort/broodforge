// SPDX-License-Identifier: GPL-3.0-or-later

package cli_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/jasondorjeshort/broodforge/internal/audit"
	"github.com/jasondorjeshort/broodforge/internal/cli"
	"github.com/jasondorjeshort/broodforge/internal/cli/clitest"
	"github.com/jasondorjeshort/broodforge/internal/config"
	"github.com/jasondorjeshort/broodforge/internal/control"
	"github.com/jasondorjeshort/broodforge/internal/control/controltest"
	"github.com/jasondorjeshort/broodforge/internal/credentials"
	"github.com/jasondorjeshort/broodforge/internal/git"
	"github.com/jasondorjeshort/broodforge/internal/git/gittest"
	"github.com/jasondorjeshort/broodforge/internal/spawn"
	"github.com/jasondorjeshort/broodforge/internal/store"
)

// The P8 Proof (08-implementation-plan.md), on the built binary against the in-process
// harness: the rows P10 will write seeded through the store API, a canonical and a bare
// remote, worktree-create, commits in the tree through os/exec under the four identity
// variables (the commit-msg hook appends the trailer), then worktree check and handoff.

// install lays out <bf> as an install does: bin/ holds the built binary, share/ the
// repository's scripts (modes set here, so the checkout's do not matter).
func install(t *testing.T, dir string) string {
	t.Helper()
	bin := clitest.Binary(t)
	b, err := os.ReadFile(bin)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "bin"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "bin", filepath.Base(bin)), b, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"askpass.sh", "askpass.cmd", "hooks/commit-msg", "hooks/pre-push"} {
		src, err := os.ReadFile(filepath.Join("..", "..", "share", filepath.FromSlash(f)))
		if err != nil {
			t.Fatal(err)
		}
		dst := filepath.Join(dir, "share", filepath.FromSlash(f))
		if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dst, src, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func seed(t *testing.T, st *store.Store, stmts ...string) {
	t.Helper()
	if err := st.Write(func(tx *store.Tx) error {
		for _, s := range stmts {
			if _, err := tx.Exec(s); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

// seedSession writes a session, its driver's participant row, one turn started before any
// commit, and that turn's usage row recorded at usageAt.
func seedSession(t *testing.T, st *store.Store, id, usageAt string) {
	t.Helper()
	seed(t, st,
		`INSERT INTO session (id, channel_id, project_id, driver_id, state, harness, credential_id, mode, reach, started_at)
VALUES ('`+id+`', '100', 'p', '1001', 'waiting-for-input', 'claude-code', 'cred-a', 'prompt', 'code', '2026-01-01T00:00:00Z')`,
		`INSERT INTO participant (session_id, user_id, role, joined_at) VALUES ('`+id+`', '1001', 'driver', '2026-01-01T00:00:00Z')`,
		`INSERT INTO turn (session_id, seq, requester_id, input, started_at) VALUES ('`+id+`', 1, '1001', 'Add a line to README.', '2026-01-01T00:00:00Z')`,
		`INSERT INTO usage_record (session_id, turn_seq, requester_id, credential_id, channel_id, project_id, harness, model,
    input_tokens, output_tokens, recorded_at)
VALUES ('`+id+`', 1, '1001', 'cred-a', '100', 'p', 'claude-code', 'claude-opus-5-5', 10, 20, '`+usageAt+`')`)
}

func serve(t *testing.T, addr string, register func(*control.Registry)) {
	t.Helper()
	reg := control.NewRegistry()
	register(reg)
	srv, err := control.Listen(addr, reg)
	if err != nil {
		t.Fatal(err)
	}
	served := make(chan error, 1)
	go func() { served <- srv.Serve() }()
	t.Cleanup(func() {
		if err := srv.Close(); err != nil {
			t.Errorf("close: %v", err)
		}
		if err := <-served; err != nil {
			t.Errorf("serve: %v", err)
		}
	})
}

type worktreeProof struct {
	root, instance, canonical, remote string
	st                                *store.Store
	backend                           spawn.Backend
	// cfg is the configuration the service reads per call, so a test can change it between calls.
	cfg atomic.Pointer[config.Config]
}

func newWorktreeProof(t *testing.T) *worktreeProof {
	t.Helper()
	root, name, addr := controltest.Root(t)
	p := &worktreeProof{root: root, instance: name}
	p.canonical, p.remote = gittest.Project(t, root, "p")
	b, err := spawn.NewSameAccount(spawn.Config{Backend: spawn.NameSameAccount, Install: install(t, filepath.Join(root, "install"))})
	if err != nil {
		t.Fatal(err)
	}
	p.backend = b
	if p.st, err = store.Create(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { p.st.Close() })
	seed(t, p.st,
		`INSERT INTO registration (user_id, git_name, git_email, display_name, registered_at)
VALUES ('1001', '`+gittest.Driver.Name+`', '`+gittest.Driver.Email+`', 'driver', '2026-01-01T00:00:00Z')`,
		`INSERT INTO credential (id, kind, provider, user_id, label, status, added_at, store_ref)
VALUES ('cred-a', 'api-key', 'anthropic', '1001', 'a', 'usable', '2026-01-01T00:00:00Z', 'cred-a')`)
	cfg := &config.Config{Instance: config.Instance{SchemaVersion: 1, InstanceName: name},
		Projects: []config.Project{{ID: "p", Name: "P", Handoff: config.HandoffPush,
			Repo: config.Repo{Canonical: p.canonical, Remote: "origin", DefaultBranch: "main"}}}}
	p.cfg.Store(cfg)
	serve(t, addr, cli.WorktreeHandlers(&cli.WorktreeService{Root: root, Config: cfg, Current: p.cfg.Load, Store: p.st,
		Audit: audit.NewWriter(nil), Backend: b}))
	return p
}

// start is what P10's session start will do: worktree-create, the worktree row, the trailers
// file the commit-msg hook reads.
func (p *worktreeProof) start(t *testing.T, id string) git.Session {
	t.Helper()
	return p.startAt(t, id, "2026-01-01T00:01:00Z", true)
}

func (p *worktreeProof) startAt(t *testing.T, id, usageAt string, trailers bool) git.Session {
	t.Helper()
	seedSession(t, p.st, id, usageAt)
	s := git.Session{Root: p.root, ID: id, Instance: p.instance, Canonical: p.canonical, Remote: "origin", DefaultBranch: "main"}
	if _, err := git.Create(t.Context(), p.backend, s); err != nil {
		t.Fatal(err)
	}
	tree := filepath.ToSlash(filepath.Join(s.Dir(), "tree"))
	seed(t, p.st, `INSERT INTO worktree (path, project_id, session_id, branch, state, created_at)
VALUES ('`+tree+`', 'p', '`+id+`', '`+s.Branch()+`', 'active', '2026-01-01T00:00:00Z')`)
	if !trailers {
		return s
	}
	if err := git.WriteTrailers(s.Dir(), []string{"Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"}); err != nil {
		t.Fatal(err)
	}
	return s
}

var hexHash = regexp.MustCompile(`^[0-9a-f]{40}$`)

func TestProofWorktree(t *testing.T) {
	p := newWorktreeProof(t)
	s := p.start(t, "s0001")
	tree := filepath.Join(s.Dir(), "tree")
	head := gittest.Commit(t, gittest.Driver, nil, tree, "README", "base\nmore\n", "Add a line")
	if m := gittest.Git(t, tree, "log", "-1", "--format=%B"); !strings.Contains(m, "\n\nCo-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>\n") {
		t.Errorf("the commit-msg hook appended no trailer: %q", m)
	}
	root := []string{"--root", p.root}

	r := clitest.Run(t, nil, append(root, "worktree", "check", "s0001")...)
	want(t, r, 0, "ok  pre-push\n"+head+"  ok\nok  1\n", "")
	branch := "broodforge/" + p.instance + "/s0001"
	refspec := "refs/sessions/s0001:refs/heads/" + branch
	r = clitest.Run(t, nil, append(root, "worktree", "handoff", "s0001", "--dry-run", "--expect", "dry-run")...)
	want(t, r, 0, "dry-run  origin  "+refspec+"  1  -  -\n", "")
	r = clitest.Run(t, nil, append(root, "worktree", "handoff", "s0001", "--expect", "pushed")...)
	want(t, r, 0, "pushed  origin  "+refspec+"  1  -  -\n", "")
	branches := gittest.Git(t, p.remote, "branch")
	t.Logf("$ git -C <remote> branch\n%s", branches)
	if !strings.Contains(branches, branch) || strings.TrimSpace(gittest.Git(t, p.remote, "rev-parse", branch)) != head {
		t.Errorf("remote branches %q lack %s at %s", branches, branch, head)
	}
	r = clitest.Run(t, nil, append(root, "worktree", "handoff", "s0001")...)
	if r.Code != 1 || r.Stdout != "" || !strings.HasPrefix(r.Stderr, "error: already-handed-off: ") {
		t.Errorf("second handoff: %+v", r)
	}
	var state, outcome string
	if err := p.st.Query(`SELECT w.state, coalesce(s.end_outcome, '') FROM worktree w JOIN session s ON s.id = w.session_id WHERE s.id = 's0001'`,
		nil, func(r *store.Row) error { state, outcome = r.Text(0), r.Text(1); return nil }); err != nil {
		t.Fatal(err)
	}
	if state != store.WorktreeRemoved || outcome != branch {
		t.Errorf("worktree %q, end outcome %q", state, outcome)
	}

	// The wrong author: refused, the failing line streamed, the tree kept.
	s2 := p.start(t, "s0002")
	tree2 := filepath.Join(s2.Dir(), "tree")
	bad := gittest.Commit(t, gittest.Driver, nil, tree2, "t.txt", "t\n", "t", "--author", "X <x@example.com>")
	r = clitest.Run(t, nil, append(root, "worktree", "handoff", "s0002")...)
	want(t, r, 1, bad+"  author:X <x@example.com>\n",
		"error: handoff-refused-attribution: 1 of 1 commits fail the attribution check\n")
	r = clitest.Run(t, nil, append(root, "worktree", "handoff", "s0002", "--expect", "handoff-refused-attribution")...)
	want(t, r, 0, bad+"  author:X <x@example.com>\n", "")
	r = clitest.Run(t, nil, append(root, "worktree", "check", "s0002")...)
	want(t, r, 1, "ok  pre-push\n"+bad+"  author:X <x@example.com>\n", "error: handoff-refused-attribution: 1 of 1 commits fail the attribution check\n")
	if _, err := os.Stat(tree2); err != nil {
		t.Errorf("a refused handoff removed the tree: %v", err)
	}
	r = clitest.Run(t, nil, append(root, "worktree", "bundle", "s0002")...)
	want(t, r, 0, filepath.ToSlash(filepath.Join(s2.Dir(), "out.bundle"))+"  1\n", "")

	var kinds []string
	if err := p.st.Query("SELECT kind || ':' || coalesce(phase, '') || ':' || coalesce(session_id, '') FROM audit_event ORDER BY seq", nil,
		func(r *store.Row) error { kinds = append(kinds, r.Text(0)); return nil }); err != nil {
		t.Fatal(err)
	}
	wantKinds := "push:attempted:s0001 push:done:s0001 handoff-refused::s0002 handoff-refused::s0002"
	if strings.Join(kinds, " ") != wantKinds {
		t.Errorf("audit %q, want %q", kinds, wantKinds)
	}
	if !hexHash.MatchString(head) || !hexHash.MatchString(bad) {
		t.Errorf("hashes %q %q", head, bad)
	}
}

func TestWorktreeRefusals(t *testing.T) {
	p := newWorktreeProof(t)
	root := []string{"--root", p.root}
	want(t, clitest.Run(t, nil, append(root, "worktree", "check", "s0009")...), 1, "", "error: unknown-session: no session s0009\n")
	want(t, clitest.Run(t, nil, append(root, "worktree", "handoff", "../x")...), 1, "", "error: bad-args: \"../x\" is not a session id\n")
	want(t, clitest.Run(t, nil, append(root, "worktree", "handoff")...), 2, "", "error: usage: worktree handoff: missing <id>\n")
	// No commits: nothing to push, the tree removed.
	s := p.start(t, "s0003")
	want(t, clitest.Run(t, nil, append(root, "worktree", "bundle", "s0003")...), 0, "none  0\n", "")
	want(t, clitest.Run(t, nil, append(root, "worktree", "handoff", "s0003")...), 0, "none  origin  -  0  -  -\n", "")
	if _, err := os.Stat(filepath.Join(s.Dir(), "tree")); !os.IsNotExist(err) {
		t.Errorf("tree after a no-commit handoff: %v", err)
	}
}

// The askpass path: the helper the push names, run by git itself, answers with the credential
// the bot read from the store and put on that one process; without one it refuses and git
// reads no password.
func TestAskpassAnswersFromTheStore(t *testing.T) {
	root, _, _ := controltest.Root(t)
	st, err := store.Create(root)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	keyFile := filepath.Join(root, "test.key")
	if _, err := credentials.Init(keyFile); err != nil {
		t.Fatal(err)
	}
	v, err := credentials.OpenVault(credentials.StorePath(root), keyFile)
	if err != nil {
		t.Fatal(err)
	}
	if err := v.Put("cred-git", []byte("ghp_test0123456789")); err != nil {
		t.Fatal(err)
	}
	seed(t, st, `INSERT INTO credential (id, kind, provider, user_id, label, status, added_at, store_ref)
VALUES ('cred-git', 'api-key', 'github', NULL, 'git', 'usable', '2026-01-01T00:00:00Z', 'cred-git')`)
	svc, err := credentials.NewService(st, audit.NewWriter(nil), credentials.Config{Root: root, KeyFile: keyFile})
	if err != nil {
		t.Fatal(err)
	}
	token, err := svc.Secret("cred-git")
	if err != nil {
		t.Fatal(err)
	}
	b, err := spawn.NewSameAccount(spawn.Config{Backend: spawn.NameSameAccount, Install: install(t, filepath.Join(root, "install"))})
	if err != nil {
		t.Fatal(err)
	}
	fill := func(tok []byte) (string, string, error) {
		var env []string
		for _, v := range git.PushEnv(b.Askpass(), tok) {
			env = append(env, v.Name+"="+v.Value)
		}
		cmd := exec.Command("git", "-c", "credential.helper=", "credential", "fill")
		cmd.Env = gittest.Env(gittest.Driver, env...)
		cmd.Stdin = strings.NewReader("protocol=https\nhost=example.com\n\n")
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		err := cmd.Run()
		t.Logf("$ git credential fill (askpass %s)\nstdout: %q\nstderr: %q\nerr: %v", b.Askpass(), stdout.String(), stderr.String(), err)
		return stdout.String(), stderr.String(), err
	}
	out, _, err := fill(token)
	if err != nil || !strings.Contains(out, "username=x-access-token\npassword=ghp_test0123456789\n") {
		t.Errorf("with the credential: %q %v", out, err)
	}
	out, stderr, err := fill(nil)
	if err == nil || strings.Contains(out, "password=") || !strings.Contains(stderr, "error: no-git-credential: ") {
		t.Errorf("without one: %q %q %v", out, stderr, err)
	}
	want(t, clitest.Run(t, []string{git.CredentialEnv + "=v1"}, "git-askpass", "Password for 'https://github.com': "), 0, "v1\n", "")
	want(t, clitest.Run(t, nil, "git-askpass", "Username for 'https://github.com': "), 0, "x-access-token\n", "")
	want(t, clitest.Run(t, []string{git.CredentialEnv + "=v1"}, "git-askpass", "Enter passphrase for key 'k': "), 1, "",
		"error: no-git-credential: the helper answers Username and Password prompts only\n")
}

// A model is a co-author from the start of its first turn, not from its usage row, which lands
// at the turn's end: a commit made during turn 1 without the trailer is missing it.
func TestWorktreeModelCountsFromItsFirstTurn(t *testing.T) {
	p := newWorktreeProof(t)
	s := p.startAt(t, "s0004", "2099-01-01T00:00:00Z", false)
	head := gittest.Commit(t, gittest.Driver, nil, filepath.Join(s.Dir(), "tree"), "a.txt", "a\n", "during turn 1")
	want(t, clitest.Run(t, nil, "--root", p.root, "worktree", "check", "s0004"), 1,
		"ok  pre-push\n"+head+"  missing:Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>\n",
		"error: handoff-refused-attribution: 1 of 1 commits fail the attribution check\n")
}

// The commit-msg hook with no binary beside it: one stderr line, exit 0, so the commit goes on.
func TestCommitMsgHookWithoutTheBinary(t *testing.T) {
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("no sh on PATH")
	}
	dir := filepath.Join(t.TempDir(), "share", "hooks")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	src, err := os.ReadFile(filepath.Join("..", "..", "share", "hooks", "commit-msg"))
	if err != nil {
		t.Fatal(err)
	}
	hook := filepath.Join(dir, "commit-msg")
	if err := os.WriteFile(hook, src, 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(sh, filepath.ToSlash(hook), "msg")
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err = cmd.Run()
	t.Logf("$ sh %s msg\nerr: %v\nstdout: %q\nstderr: %q", hook, err, stdout.String(), stderr.String())
	if err != nil || stdout.Len() != 0 || strings.Count(stderr.String(), "\n") != 1 || !strings.HasPrefix(stderr.String(), "commit-msg: no ") {
		t.Errorf("hook without the binary: %v %q %q", err, stdout.String(), stderr.String())
	}
}
