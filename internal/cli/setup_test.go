// SPDX-License-Identifier: GPL-3.0-or-later

package cli_test

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jasondorjeshort/broodforge/internal/cli"
	"github.com/jasondorjeshort/broodforge/internal/cli/clitest"
	"github.com/jasondorjeshort/broodforge/internal/config"
	"github.com/jasondorjeshort/broodforge/internal/discord"
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
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(self)
	if err != nil {
		t.Fatal(err)
	}
	// Both names: setup looks up its host's engine name, and this test does not branch on the host.
	for _, name := range []string{"claude", "claude.exe"} {
		if err := os.WriteFile(filepath.Join(stub, name), b, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	var r [6]byte
	if _, err := rand.Read(r[:]); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(dir, "discord.json")
	if err := os.WriteFile(script, fakeDiscordScript(t), 0o600); err != nil {
		t.Fatal(err)
	}
	return &setupWorld{dir: dir, repo: remote, instance: "t-" + hex.EncodeToString(r[:]),
		env: []string{"PATH=" + stub + string(os.PathListSeparator) + os.Getenv("PATH"), spawntest.Env + "=1",
			"ProgramData=" + filepath.Join(dir, "programdata"), cli.FakeDiscordEnv + "=" + script}}
}

// fakeToken is the bot token the scripted Discord accepts.
const fakeToken = "NTAwMA.abcd.abcdefghijklmnopqrstuvwxyz"

// fakeDiscordScript is the P22 Proof's scripted Discord: an application whose bot is in no
// server until the invite's poll, joining with Manage Roles, which the owner removes on the poll
// after the removal fix. Every root of the test shares it, so a later root finds by name what
// the first created.
func fakeDiscordScript(t *testing.T) []byte {
	t.Helper()
	b, err := json.Marshal(discord.FakeScript{Token: fakeToken, Application: discord.FakeApp{ID: "4000", BotID: "5000"},
		Invite: &discord.FakeGuild{ID: "3000", Name: "Test", OwnerID: "6000",
			Roles: []discord.FakeRole{{ID: "3000", Name: "@everyone"},
				{ID: "5001", Name: "Broodforge", Position: 2, Permissions: discord.PermissionBits | discord.PermManageRoles}},
			BotRoles: []string{"5001"}, DropManageRolesAfter: 1},
		JoinAfter: 1})
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// runIn is run with stdin.
func (w *setupWorld) runIn(t *testing.T, stdin string, args ...string) clitest.Result {
	t.Helper()
	cmd := exec.Command(clitest.Binary(t), args...)
	cmd.Env = clitest.Environ(w.env)
	cmd.Stdin = strings.NewReader(stdin)
	var stdout, stderr strings.Builder
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	r := clitest.Result{Stdout: stdout.String(), Stderr: stderr.String()}
	var ee *exec.ExitError
	switch {
	case err == nil:
	case errors.As(err, &ee):
		r.Code = ee.ExitCode()
	default:
		t.Fatalf("run broodforge %q: %v", args, err)
	}
	t.Logf("$ broodforge %s <stdin\nexit %d\nstdout: %q\nstderr: %q", strings.Join(args, " "), r.Code, r.Stdout, r.Stderr)
	return r
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

	// A fresh run, the token on stdin: ok for every step but the deferred git credential; the
	// invite and the Manage Roles removal print their fix, then ok once the poll passes.
	r := w.runIn(t, fakeToken+"\n", full...)
	if r.Code != 0 {
		t.Fatalf("setup exit %d\n%s%s", r.Code, r.Stdout, r.Stderr)
	}
	if !strings.Contains(r.Stderr, "waiting for the bot to join\n") || strings.Contains(r.Stdout+r.Stderr, fakeToken) {
		t.Errorf("stderr %q", r.Stderr)
	}
	for _, c := range firstColumns(r.Stdout) {
		step := strings.SplitN(c, " ", 2)[1]
		switch {
		case c == "fix guild" || c == "fix role":
		case strings.Contains(" token application guild channels role ", " "+step+" "):
			if c != "ok "+step {
				t.Errorf("%s, want ok", c)
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

	// --check on an empty root: fix for every step, nothing created.
	r2 := w.root("2")
	r = w.run(t, "setup", "--check", "--root", r2, "--instance", w.instance)
	if r.Code != 1 || !strings.HasPrefix(r.Stderr, "error: check-failed: root config key engine token application guild channels role git-credential project defaults check next") {
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
	var pausedErr strings.Builder
	paused.Stderr = &pausedErr
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
	// Off a terminal without --yes stdin is read for the key alone: the token step refuses.
	if err := paused.Wait(); err == nil || !strings.Contains(pausedErr.String(), "error: needs-input-token: pass the bot token on stdin with --yes\n") {
		t.Errorf("the paused setup: %v %q", err, pausedErr.String())
	}

	// --json: one object per line, the key in the key object only.
	r = w.runIn(t, fakeToken, "setup", "--json", "--yes", "--root", w.seeded(t, "6"), "--instance", w.instance, "--repo", w.repo, "--wait", "1")
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

// setKeyCLI sets one key of a config file of root, as the owner's hand edit.
func setKeyCLI(t *testing.T, root, file, value string, path ...string) {
	t.Helper()
	p := filepath.Join(root, "config", file)
	d, _, err := config.ReadDoc(p)
	if err != nil || d == nil {
		t.Fatalf("read %s: %v", p, err)
	}
	if err := d.Set("", []byte(value), path...); err != nil {
		t.Fatal(err)
	}
	if err := config.WriteDoc(p, d); err != nil {
		t.Fatal(err)
	}
}

// TestProofSetupDryRun is P27's flag on the built binary: a value outside live|dry-run is a
// usage error; under dry-run BROODFORGE_SETUP_FAKE_DISCORD is not read (a script that does not
// parse is no refusal); --check --discord dry-run prints the form's fix lines; setup <step>
// takes the flag.
func TestProofSetupDryRun(t *testing.T) {
	w := newSetupWorld(t)
	root := w.seeded(t, "d")

	r := w.run(t, "setup", "--discord", "bogus", "--yes", "--root", root)
	if r.Code != 2 || !strings.HasPrefix(r.Stderr, `error: usage: setup: --discord "bogus": want live or dry-run`) {
		t.Errorf("--discord bogus: exit %d %q", r.Code, r.Stderr)
	}

	// A script that does not parse: refused bad-args when read, so a pass proves it unread.
	bad := filepath.Join(w.dir, "not-a-script.json")
	if err := os.WriteFile(bad, []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	w.env = append(w.env, cli.FakeDiscordEnv+"="+bad)
	r = w.run(t, "setup", "--check", "--root", root)
	if r.Code != 1 || !strings.HasPrefix(r.Stderr, "error: bad-args: "+cli.FakeDiscordEnv+": ") {
		t.Fatalf("the bad script without dry-run: exit %d %q", r.Code, r.Stderr)
	}

	// --check --discord dry-run before setup: fix lines, the dry-run next.
	r = w.run(t, "setup", "--check", "--discord", "dry-run", "--root", root, "--repo", w.repo)
	if r.Code != 1 || !strings.HasPrefix(r.Stderr, "error: check-failed: ") ||
		!strings.Contains(r.Stdout, "skip  token  dry-run\n") || !strings.Contains(r.Stdout, "fix  next  run broodforge setup --discord dry-run\n") {
		t.Errorf("--check --discord dry-run: exit %d %q\n%s", r.Code, r.Stderr, r.Stdout)
	}

	full := []string{"setup", "--discord", "dry-run", "--yes", "--root", root, "--instance", w.instance, "--repo", w.repo, "--wait", "1"}
	r = w.run(t, full...)
	if r.Code != 0 {
		t.Fatalf("setup --discord dry-run: exit %d %s\n%s", r.Code, r.Stderr, r.Stdout)
	}
	for _, l := range []string{"skip  token  dry-run", "skip  application  dry-run", "skip  guild  dry-run", "ok  role  dry-run 10"} {
		if !strings.Contains(r.Stdout, l+"\n") {
			t.Errorf("no %q:\n%s", l, r.Stdout)
		}
	}
	if !strings.Contains(r.Stdout, "ok  channels  dry-run 100 fake, 200 claude-code; fixtures ") {
		t.Errorf("no dry-run channels line:\n%s", r.Stdout)
	}

	// The placeholders removed by hand: --check names the form's two fix texts.
	setKeyCLI(t, root, "projects/demo.json", "{}", "channels")
	setKeyCLI(t, root, "instance.json", "[]", "roleGroups", "trusted")
	r = w.run(t, "setup", "--check", "--discord", "dry-run", "--root", root)
	for _, l := range []string{"fix  channels  the dry-run channels are not configured; run setup --discord dry-run",
		"fix  role  roleGroups.trusted is not set; run setup --discord dry-run", "fix  next  run broodforge setup --discord dry-run"} {
		if !strings.Contains(r.Stdout, l+"\n") {
			t.Errorf("--check: no %q:\n%s", l, r.Stdout)
		}
	}
	if r.Code != 1 || !strings.HasPrefix(r.Stderr, "error: check-failed: channels role next") {
		t.Errorf("--check: exit %d %q", r.Code, r.Stderr)
	}

	// setup <step> --discord dry-run writes that step's placeholders alone.
	for _, step := range []string{"channels", "role"} {
		r = w.run(t, "setup", step, "--discord", "dry-run", "--yes", "--root", root)
		if r.Code != 0 || len(firstColumns(r.Stdout)) != 1 || firstColumns(r.Stdout)[0] != "ok "+step {
			t.Errorf("setup %s --discord dry-run: exit %d %s\n%s", step, r.Code, r.Stderr, r.Stdout)
		}
	}
	if r := w.run(t, "setup", "--check", "--discord", "dry-run", "--root", root); r.Code != 0 {
		t.Errorf("--check after setup <step>: exit %d %s\n%s", r.Code, r.Stderr, r.Stdout)
	}
}
