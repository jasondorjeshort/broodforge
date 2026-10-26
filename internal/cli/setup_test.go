// SPDX-License-Identifier: GPL-3.0-or-later

package cli_test

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jasondorjeshort/broodforge/internal/cli/clitest"
	"github.com/jasondorjeshort/broodforge/internal/git/gittest"
	"github.com/jasondorjeshort/broodforge/internal/spawn/spawntest"
)

// setupWorld is the P21 Proof's fixture: a bare repository holding main, this test binary
// copied as the stand-in engine on PATH (its --version prints 2.1.296), and an instance name
// no other run uses, since pipe names are machine-wide.
type setupWorld struct {
	dir, repo, instance string
	env                 []string
}

func newSetupWorld(t *testing.T) *setupWorld {
	t.Helper()
	dir := t.TempDir()
	_, remote := gittest.Project(t, filepath.Join(dir, "src"), "demo")
	stub := filepath.Join(dir, "stub")
	if err := os.MkdirAll(stub, 0o755); err != nil {
		t.Fatal(err)
	}
	name := "claude"
	if runtime.GOOS == "windows" {
		name = "claude.exe"
	}
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(self)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stub, name), b, 0o755); err != nil {
		t.Fatal(err)
	}
	var r [6]byte
	if _, err := rand.Read(r[:]); err != nil {
		t.Fatal(err)
	}
	return &setupWorld{dir: dir, repo: remote, instance: "t-" + hex.EncodeToString(r[:]),
		env: []string{"PATH=" + stub + string(os.PathListSeparator) + os.Getenv("PATH"), spawntest.Env + "=1",
			"ProgramData=" + filepath.Join(dir, "programdata")}}
}

func (w *setupWorld) root(n string) string { return filepath.ToSlash(filepath.Join(w.dir, n, "root")) }

// seeded is a root whose instance file holds only its name and paths.install, the checkout:
// setup writes paths.install empty (the binary's parent directory), and the test's binary sits
// in a temporary directory without share/, which spawn preflight in the check step reads. Setup
// writes every other key, the absent ones.
func (w *setupWorld) seeded(t *testing.T, n string) string {
	t.Helper()
	root := w.root(n)
	repo, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	doc := `{ "schemaVersion": 1, "instanceName": "` + w.instance + `", "paths": { "install": "` + filepath.ToSlash(repo) + `" } }` + "\n"
	if err := os.MkdirAll(filepath.Join(root, "config"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "config", "instance.json"), []byte(doc), 0o600); err != nil {
		t.Fatal(err)
	}
	return root
}

func (w *setupWorld) run(t *testing.T, args ...string) clitest.Result {
	t.Helper()
	return clitest.Run(t, w.env, args...)
}

func firstColumns(out string) []string {
	var r []string
	for _, l := range strings.Split(strings.TrimSpace(out), "\n") {
		f := strings.SplitN(l, "  ", 3)
		if len(f) >= 2 {
			r = append(r, f[0]+" "+f[1])
		}
	}
	return r
}

// TestProofSetup is the P21 Proof (08-implementation-plan-setup.md) on the built binary.
func TestProofSetup(t *testing.T) {
	w := newSetupWorld(t)
	r1 := w.seeded(t, "1")
	full := []string{"setup", "--yes", "--root", r1, "--instance", w.instance, "--repo", w.repo, "--wait", "1"}

	// A fresh run: ok for every step but the Discord ones and the deferred git credential.
	r := w.run(t, full...)
	if r.Code != 0 {
		t.Fatalf("setup exit %d\n%s%s", r.Code, r.Stdout, r.Stderr)
	}
	for _, c := range firstColumns(r.Stdout) {
		step := strings.SplitN(c, " ", 2)[1]
		switch {
		case strings.Contains(" token application guild channels role ", " "+step+" "):
			if !strings.Contains(r.Stdout, "skip  "+step+"  not built\n") {
				t.Errorf("%s is not `skip  not built`", step)
			}
		case step == "git-credential":
			if c != "skip git-credential" {
				t.Errorf("git-credential: %s", c)
			}
		case !strings.HasPrefix(c, "ok "):
			t.Errorf("%s, want ok", c)
		}
	}
	if r := w.run(t, "config", "check", "--root", r1); r.Code != 0 {
		t.Errorf("config check after setup: exit %d %s", r.Code, r.Stderr)
	}

	// The rerun: only ok and skip, and config/ unchanged (ADR 0027, prediction 1).
	before := readTree(t, filepath.Join(r1, "config"))
	r = w.run(t, full...)
	if r.Code != 0 {
		t.Fatalf("rerun exit %d\n%s%s", r.Code, r.Stdout, r.Stderr)
	}
	for _, c := range firstColumns(r.Stdout) {
		if !strings.HasPrefix(c, "ok ") && !strings.HasPrefix(c, "skip ") {
			t.Errorf("rerun printed %s", c)
		}
	}
	if after := readTree(t, filepath.Join(r1, "config")); after != before {
		t.Errorf("the rerun changed config/")
	}

	// --check on an empty root: fix for every step but the Discord ones, nothing created.
	r2 := w.root("2")
	r = w.run(t, "setup", "--check", "--root", r2, "--instance", w.instance)
	if r.Code != 1 || !strings.HasPrefix(r.Stderr, "error: check-failed: root config key engine git-credential project defaults check next") {
		t.Errorf("--check on an empty root: exit %d %s\n%s", r.Code, r.Stderr, r.Stdout)
	}
	if _, err := os.Stat(filepath.Dir(r2)); !os.IsNotExist(err) {
		t.Errorf("--check created %s", filepath.Dir(r2))
	}

	// --yes and no repository.
	r = w.run(t, "setup", "--yes", "--root", w.root("3"), "--instance", w.instance, "--wait", "1")
	want(t, clitest.Result{Code: r.Code, Stderr: r.Stderr}, 1, "", "error: needs-input-repo: pass --repo\n")

	// A bot on the root: setup is refused running with the owner's text; --check still runs.
	serve := exec.Command(clitest.Binary(t), "serve", "--root", r1)
	serve.Env = clitest.Environ(w.env)
	if err := serve.Start(); err != nil {
		t.Fatal(err)
	}
	if r := w.run(t, "status", "--root", r1, "--wait", "10", "--expect", "ok"); r.Code != 0 {
		_ = serve.Process.Kill()
		t.Fatalf("serve did not answer: %s", r.Stderr)
	}
	r = w.run(t, "setup", "--root", r1, "--yes")
	if r.Code != 1 || !strings.HasPrefix(r.Stderr, "error: running: a bot answers at ") ||
		!strings.HasSuffix(r.Stderr, "; stop it with broodforge stop in this shell (every open session is interrupted) or use setup --check\n") {
		t.Errorf("setup with a bot: exit %d %q", r.Code, r.Stderr)
	}
	if r := w.run(t, "setup", "--check", "--root", r1); r.Code != 0 {
		t.Errorf("setup --check with a bot: exit %d %s", r.Code, r.Stderr)
	}
	w.run(t, "stop", "--root", r1)
	_ = serve.Wait()

	// A setup paused at the key holds the address: serve fails start-control, a second setup is
	// setup-running, status answers `config: setup`.
	r5 := w.seeded(t, "5")
	paused := exec.Command(clitest.Binary(t), "setup", "--root", r5, "--instance", w.instance, "--repo", w.repo, "--wait", "1")
	paused.Env = clitest.Environ(w.env)
	stdin, err := paused.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := paused.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := paused.Start(); err != nil {
		t.Fatal(err)
	}
	lines := make(chan string, 64)
	go func() {
		sc := bufio.NewScanner(stdout)
		for sc.Scan() {
			lines <- sc.Text()
		}
		close(lines)
	}()
	waitFor(t, lines, "ok  key  ")
	r = w.run(t, "serve", "--root", r5)
	if r.Code != 1 || !strings.HasPrefix(r.Stderr, "error: start-control: ") {
		t.Errorf("serve beside setup: exit %d %q", r.Code, r.Stderr)
	}
	r = w.run(t, "setup", "--root", r5, "--yes")
	if r.Code != 1 || !strings.HasPrefix(r.Stderr, "error: setup-running: another setup holds ") {
		t.Errorf("second setup: exit %d %q", r.Code, r.Stderr)
	}
	want(t, w.run(t, "status", "--root", r5, "--expect", "setup"), 0,
		"config: setup\ndiscord: -\nsessions: -\nworktrees: -\nbackup: -\nengine: -\n", "")
	_, _ = io.WriteString(stdin, "\n")
	_ = stdin.Close()
	for range lines {
	}
	if err := paused.Wait(); err != nil {
		t.Errorf("the paused setup: %v", err)
	}

	// --json: one object per line, the key in the key object only.
	r = w.run(t, "setup", "--json", "--yes", "--root", w.seeded(t, "6"), "--instance", w.instance, "--repo", w.repo, "--wait", "1")
	if r.Code != 0 {
		t.Fatalf("--json exit %d %s", r.Code, r.Stderr)
	}
	key := ""
	for _, l := range strings.Split(strings.TrimSpace(r.Stdout), "\n") {
		var obj map[string]string
		if err := json.Unmarshal([]byte(l), &obj); err != nil {
			t.Fatalf("not an object: %q", l)
		}
		if obj["step"] == "key" {
			key = obj["key"]
		}
	}
	if len(key) != 64 || strings.Count(r.Stdout, key) != 1 {
		t.Errorf("the key in --json: %q, %d times", key, strings.Count(r.Stdout, key))
	}
}

func waitFor(t *testing.T, lines <-chan string, prefix string) {
	t.Helper()
	timeout := time.After(60 * time.Second)
	for {
		select {
		case l, ok := <-lines:
			if !ok {
				t.Fatalf("setup ended before %q", prefix)
			}
			if strings.HasPrefix(l, prefix) {
				return
			}
		case <-timeout:
			t.Fatalf("no %q line", prefix)
		}
	}
}

// readTree is every file under dir with its content, as one string.
func readTree(t *testing.T, dir string) string {
	t.Helper()
	var b strings.Builder
	err := filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		c, err := os.ReadFile(p)
		b.WriteString(p + "\n" + string(c))
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return b.String()
}
