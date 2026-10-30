// SPDX-License-Identifier: GPL-3.0-or-later

package setup_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jasondorjeshort/broodforge/internal/config"
	"github.com/jasondorjeshort/broodforge/internal/control"
	"github.com/jasondorjeshort/broodforge/internal/discord"
	"github.com/jasondorjeshort/broodforge/internal/git/gittest"
	"github.com/jasondorjeshort/broodforge/internal/setup"
	"github.com/jasondorjeshort/broodforge/internal/spawn/spawntest"
)

// TestMain makes this test binary the stand-in engine when the engine step runs it with
// spawntest.Env set: its --version prints 2.1.296 (Claude Code).
func TestMain(m *testing.M) {
	spawntest.Main()
	os.Exit(m.Run())
}

// syncBuffer is a bytes.Buffer safe for a run writing while the test reads.
type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

// fakeClock advances by each Sleep, so a poll runs out at once.
type fakeClock struct{ now time.Time }

func (c *fakeClock) Now() time.Time        { return c.now }
func (c *fakeClock) Sleep(d time.Duration) { c.now = c.now.Add(d) }

// fixture is one test's world: a bare repository holding main, the stand-in engine, a root
// under the test's directory and an instance name no other run uses (pipe names are machine-wide).
type fixture struct {
	t        *testing.T
	dir      string
	root     string
	instance string
	repo     string
	engine   string // "" leaves the engine off PATH
	// fake is the scripted Discord the five Discord steps call; nil leaves them out.
	fake *discord.FakeSetup
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	dir := t.TempDir()
	_, remote := gittest.Project(t, filepath.Join(dir, "src"), "demo")
	var b [6]byte
	if _, err := rand.Read(b[:]); err != nil {
		t.Fatal(err)
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(spawntest.Env, "1")
	t.Setenv("BROODFORGE_KEY_FILE", "")
	return &fixture{t: t, dir: dir, root: filepath.ToSlash(filepath.Join(dir, "r", "root")), instance: "t-" + hex.EncodeToString(b[:]),
		repo: remote, engine: filepath.ToSlash(exe)}
}

func (f *fixture) env(stdin io.Reader, stdout io.Writer) setup.Env {
	env := setup.Env{
		Stdin: stdin, Stdout: stdout, Stderr: io.Discard,
		LookupEnv: func(k string) (string, bool) {
			if k == "ProgramData" {
				return filepath.Join(f.dir, "programdata"), true
			}
			if k == "BROODFORGE_KEY_FILE" {
				return "", false
			}
			return os.LookupEnv(k)
		},
		LookPath: func(string) (string, error) {
			if f.engine == "" {
				return "", errors.New("not found")
			}
			return f.engine, nil
		},
		Clock:   &fakeClock{now: time.Unix(0, 0)},
		Install: f.dir,
		Checks: func(context.Context, string) []setup.Echo {
			return []setup.Echo{{OK: true, Text: "config check  ok  1  0"}, {OK: true, Text: "spawn preflight  ok  backend  same-account"}}
		},
	}
	if f.fake != nil {
		env.Discord = f.fake.API
	}
	return env
}

func (f *fixture) opts(mod ...func(*setup.Options)) setup.Options {
	o := setup.Options{Root: f.root, Instance: f.instance, Repo: f.repo, Yes: true, Wait: time.Second}
	for _, m := range mod {
		m(&o)
	}
	return o
}

// run runs setup and returns its stdout and refusal code ("" for none).
func (f *fixture) run(stdin string, mod ...func(*setup.Options)) (string, string) {
	f.t.Helper()
	var out syncBuffer
	err := setup.Run(context.Background(), f.opts(mod...), f.env(strings.NewReader(stdin), &out))
	return out.String(), code(err)
}

// runTTY is run with stdin a terminal: the questions are asked, not refused.
func (f *fixture) runTTY(stdin io.Reader, mod ...func(*setup.Options)) (string, string) {
	f.t.Helper()
	var out syncBuffer
	env := f.env(stdin, &out)
	env.TTY = true
	err := setup.Run(context.Background(), f.opts(mod...), env)
	return out.String(), code(err)
}

// lineThenFail gives one empty line (the key's Enter), then fails every read.
type lineThenFail struct{ done bool }

func (r *lineThenFail) Read(p []byte) (int, error) {
	if r.done {
		return 0, errors.New("the terminal went away")
	}
	r.done = true
	return copy(p, "\n"), nil
}

func code(err error) string {
	if err == nil {
		return ""
	}
	var ce *control.Error
	if errors.As(err, &ce) {
		return ce.Code
	}
	return "error: " + err.Error()
}

// results is each line's first two columns: `ok root`.
func results(out string) []string {
	var r []string
	for _, l := range strings.Split(strings.TrimSpace(out), "\n") {
		f := strings.SplitN(l, "  ", 3)
		if len(f) >= 2 {
			r = append(r, f[0]+" "+f[1])
		}
	}
	return r
}

var discordSteps = []string{"token", "application", "guild", "channels", "role"}

func TestSteps(t *testing.T) {
	f := newFixture(t)
	out, c := f.run("")
	if c != "" {
		t.Fatalf("setup refused %s:\n%s", c, out)
	}
	want := []string{"ok root", "ok config", "ok key", "ok engine"}
	for _, s := range discordSteps {
		want = append(want, "skip "+s)
	}
	want = append(want, "skip git-credential", "ok project", "ok defaults", "ok check", "ok check", "ok next")
	if got := results(out); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("lines:\n%s\nwant %v", out, want)
	}
	for _, s := range discordSteps {
		if !strings.Contains(out, "skip  "+s+"  no Discord\n") {
			t.Errorf("no `skip  %s  no Discord`:\n%s", s, out)
		}
	}
	c2, err := config.Load(f.root, config.AnyCredential)
	if err != nil {
		t.Fatalf("config does not load after setup: %v", err)
	}
	in := c2.Instance
	if in.InstanceName != f.instance || in.Engine.Path != f.engine || in.Engine.Version != "2.1.296" || in.Defaults.Mode != config.ModePrompt ||
		strings.Join(in.Privacy.Allow, ",") != "broodforge,broodforge-engine" {
		t.Errorf("instance: %+v", in)
	}
	if len(c2.Projects) != 1 || c2.Projects[0].ID != "demo" || c2.Projects[0].Repo.DefaultBranch != "main" {
		t.Errorf("projects: %+v", c2.Projects)
	}
	if _, err := os.Stat(filepath.Join(f.root, "projects", "demo", "README")); err != nil {
		t.Errorf("no clone: %v", err)
	}
	if b, err := os.ReadFile(filepath.Join(f.root, "config", "instance.json")); err != nil || !bytes.Contains(b, []byte(`"//"`)) {
		t.Errorf("instance.json has no comments: %v", err)
	}
	if strings.Contains(out, "the bot is running.") != true {
		t.Errorf("no closing line:\n%s", out)
	}
}

// TestRerun is prediction 1 of ADR 0027: a rerun prints only ok and skip and changes no file.
func TestRerun(t *testing.T) {
	f := newFixture(t)
	if out, c := f.run(""); c != "" {
		t.Fatalf("first run refused %s:\n%s", c, out)
	}
	before := snapshot(t, f.root)
	out, c := f.run("")
	if c != "" {
		t.Fatalf("rerun refused %s:\n%s", c, out)
	}
	for _, r := range results(out) {
		if !strings.HasPrefix(r, "ok ") && !strings.HasPrefix(r, "skip ") {
			t.Errorf("rerun printed %q:\n%s", r, out)
		}
	}
	for _, s := range []string{"root", "config", "key", "engine", "project", "defaults"} {
		if !strings.Contains(out, "skip  "+s+"  ") {
			t.Errorf("rerun: %s is not skip:\n%s", s, out)
		}
	}
	if after := snapshot(t, f.root); after != before {
		t.Errorf("the rerun changed config/ or the key")
	}
	// --check on the configured root: every line passes.
	out, c = f.run("", func(o *setup.Options) { o.Check = true })
	if c != "" {
		t.Errorf("--check on a configured root refused %s:\n%s", c, out)
	}
}

// snapshot is config/ and the key file, as one string.
func snapshot(t *testing.T, root string) string {
	t.Helper()
	var b strings.Builder
	for _, dir := range []string{filepath.Join(root, "config"), filepath.Dir(root)} {
		_ = filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || (dir != filepath.Join(root, "config") && filepath.Ext(p) != ".key") {
				return nil
			}
			c, err := os.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}
			b.WriteString(p + "\n" + string(c))
			return nil
		})
	}
	return b.String()
}

func TestRefusals(t *testing.T) {
	t.Run("check on an empty root", func(t *testing.T) {
		f := newFixture(t)
		out, c := f.run("", func(o *setup.Options) { o.Check = true })
		if c != setup.CodeCheckFailed {
			t.Fatalf("code %q:\n%s", c, out)
		}
		for _, s := range []string{"root", "config", "key", "engine", "git-credential", "project", "defaults", "next"} {
			if !strings.Contains(out, "fix  "+s+"  ") {
				t.Errorf("no fix for %s:\n%s", s, out)
			}
		}
		if _, err := os.Stat(filepath.Dir(f.root)); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("--check created %s: %v", filepath.Dir(f.root), err)
		}
	})
	t.Run("needs-input-repo", func(t *testing.T) {
		f := newFixture(t)
		_, c := f.run("", func(o *setup.Options) { o.Repo = "" })
		if c != "needs-input-repo" {
			t.Errorf("code %q", c)
		}
	})
	t.Run("needs-input-project", func(t *testing.T) {
		f := newFixture(t)
		_, c := f.run("", func(o *setup.Options) { o.Repo = filepath.ToSlash(filepath.Join(f.dir, "9lives.git")) })
		if c != "needs-input-project" {
			t.Errorf("code %q", c)
		}
	})
	t.Run("needs-input-root off a terminal", func(t *testing.T) {
		f := newFixture(t)
		_, c := f.run("", func(o *setup.Options) { o.Yes, o.Instance = false, "" })
		if c != "needs-input-root" {
			t.Errorf("code %q", c)
		}
	})
	t.Run("timeout-engine", func(t *testing.T) {
		f := newFixture(t)
		f.engine = ""
		out, c := f.run("")
		if c != "timeout-engine" || !strings.Contains(out, "fix  engine  install Claude Code\n") {
			t.Errorf("code %q:\n%s", c, out)
		}
	})
	t.Run("mismatch-key and setup key", func(t *testing.T) {
		f := newFixture(t)
		if out, c := f.run(""); c != "" {
			t.Fatalf("first run refused %s:\n%s", c, out)
		}
		key := filepath.Join(filepath.Dir(f.root), f.instance+".key")
		if err := os.Chmod(key, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(key, []byte("placeholder\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if out, c := f.run(""); c != "mismatch-key" {
			t.Errorf("code %q:\n%s", c, out)
		}
		out, c := f.run("", func(o *setup.Options) { o.Step = "key" })
		if c != "" || !strings.Contains(out, "old  ") || !strings.Contains(out, "\nok  key  ") {
			t.Errorf("setup key: code %q:\n%s", c, out)
		}
		// With an entry in the store, a new key would make it unreadable: key-in-use.
		if out, c := f.run("ghp_example\n", func(o *setup.Options) { o.Step = "git-credential" }); c != "" {
			t.Fatalf("setup git-credential: %s:\n%s", c, out)
		}
		if out, c := f.run("", func(o *setup.Options) { o.Step = "key" }); c != setup.CodeKeyInUse {
			t.Errorf("code %q:\n%s", c, out)
		}
	})
	t.Run("mismatch-engine", func(t *testing.T) {
		f := newFixture(t)
		if out, c := f.run(""); c != "" {
			t.Fatalf("first run refused %s:\n%s", c, out)
		}
		t.Setenv(spawntest.EnvVersion, "2.1.300")
		if out, c := f.run(""); c != "mismatch-engine" {
			t.Errorf("code %q:\n%s", c, out)
		}
		out, c := f.run("", func(o *setup.Options) { o.Step = "engine" })
		if c != "" || !strings.Contains(out, "ok  engine  "+f.engine+"  2.1.300") {
			t.Errorf("setup engine: code %q:\n%s", c, out)
		}
	})
	t.Run("refused-project", func(t *testing.T) {
		f := newFixture(t)
		f.repo = filepath.ToSlash(filepath.Join(f.dir, "missing", "demo.git"))
		out, c := f.run("")
		if c != "refused-project" || !strings.Contains(out, "; add the token with setup git-credential and rerun") {
			t.Errorf("code %q:\n%s", c, out)
		}
	})
}

// TestConfigKeys is the rerun rule on a present, hand-written instance file: absent keys are
// written, present ones kept; privacy.allow without the service accounts is mismatch-config,
// and `setup config` adds them to the owner's list.
func TestConfigKeys(t *testing.T) {
	f := newFixture(t)
	hand := `{ "//": "by hand", "schemaVersion": 1, "instanceName": "` + f.instance + `",
  "discord": { "mode": "dry-run" }, "spawn": { "backend": "same-account" },
  "defaults": { "idleTimeoutMinutes": 60, "mode": "policy", "harness": "claude-code" },
  "privacy": { "allow": ["owner-word"] } }
`
	path := filepath.Join(f.root, "config", "instance.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(hand), 0o600); err != nil {
		t.Fatal(err)
	}
	if out, c := f.run(""); c != "mismatch-config" || !strings.Contains(out, "found the service accounts broodforge, broodforge-engine missing from it") {
		t.Fatalf("code %q:\n%s", c, out)
	}
	out, c := f.run("", func(o *setup.Options) { o.Step = "config" })
	if c != "" || !strings.Contains(out, `new  privacy.allow ["owner-word","broodforge","broodforge-engine"]`) {
		t.Fatalf("setup config: code %q:\n%s", c, out)
	}
	if out, c := f.run(""); c != "" {
		t.Fatalf("setup after setup config: code %q:\n%s", c, out)
	}
	c2, err := config.Load(f.root, config.AnyCredential)
	if err != nil {
		t.Fatalf("config does not load: %v", err)
	}
	d := c2.Instance.Defaults
	if d.IdleTimeoutMinutes != 60 || d.PromptTimeoutMinutes != 0 || d.Mode != config.ModePolicy || c2.Instance.Credentials.KeyFile == "" ||
		c2.Instance.Engine.Path == "" {
		t.Errorf("present keys changed or absent ones not written: %+v", c2.Instance)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"//": "by hand"`) {
		t.Errorf("the owner's comment is gone:\n%s", b)
	}
}

// TestReview is the review of P21: each case one fix.
func TestReview(t *testing.T) {
	t.Run("a missing key with entries in the store is key-in-use", func(t *testing.T) {
		f := newFixture(t)
		if out, c := f.run(""); c != "" {
			t.Fatalf("first run refused %s:\n%s", c, out)
		}
		if out, c := f.run("ghp_example\n", func(o *setup.Options) { o.Step = "git-credential" }); c != "" {
			t.Fatalf("setup git-credential: %s:\n%s", c, out)
		}
		key := filepath.Join(filepath.Dir(f.root), f.instance+".key")
		if err := os.Chmod(key, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(key); err != nil {
			t.Fatal(err)
		}
		if out, c := f.run(""); c != setup.CodeKeyInUse {
			t.Errorf("code %q:\n%s", c, out)
		}
		if _, err := os.Stat(key); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("a new key was written over a populated store: %v", err)
		}
	})
	t.Run("a CRLF key is valid", func(t *testing.T) {
		f := newFixture(t)
		if out, c := f.run(""); c != "" {
			t.Fatalf("first run refused %s:\n%s", c, out)
		}
		key := filepath.Join(filepath.Dir(f.root), f.instance+".key")
		b, err := os.ReadFile(key)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(key, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(key, []byte(strings.TrimSpace(string(b))+"\r\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if out, c := f.run(""); c != "" || !strings.Contains(out, "skip  key  exists\n") {
			t.Errorf("code %q:\n%s", c, out)
		}
	})
	t.Run("the rerun after refused-project finds the URL, or asks on a terminal", func(t *testing.T) {
		f := newFixture(t)
		late := filepath.Join(f.dir, "late")
		f.repo = filepath.ToSlash(filepath.Join(late, "remote", "demo.git"))
		if out, c := f.run(""); c != "refused-project" {
			t.Fatalf("code %q:\n%s", c, out)
		}
		gittest.Project(t, late, "demo") // the repository now exists at the URL
		out, c := f.run("", func(o *setup.Options) { o.Repo = "" })
		if c != "" || !strings.Contains(out, "\nok  project  ") {
			t.Fatalf("rerun without --repo: code %q:\n%s", c, out)
		}
		// With no URL kept and no clone, a terminal is asked for it, and the answer is kept.
		g := newFixture(t)
		if out, c := g.run(""); c != "" {
			t.Fatalf("first run refused %s:\n%s", c, out)
		}
		pfile := filepath.Join(g.root, "config", "projects", "demo.json")
		pdoc, _, err := config.ReadDoc(pfile)
		if err != nil {
			t.Fatal(err)
		}
		if err := pdoc.Set("", config.RawValue(""), "repo", config.SetupURLKey); err != nil {
			t.Fatal(err)
		}
		if err := config.WriteDoc(pfile, pdoc); err != nil {
			t.Fatal(err)
		}
		if err := os.RemoveAll(filepath.Join(g.root, "projects", "demo")); err != nil {
			t.Fatal(err)
		}
		if out, c := g.run("", func(o *setup.Options) { o.Repo = "" }); c != "needs-input-repo" {
			t.Errorf("off a terminal: code %q:\n%s", c, out)
		}
		out, c = g.runTTY(strings.NewReader(g.repo+"\n"), func(o *setup.Options) { o.Repo, o.Yes, o.Step = "", false, "project" })
		if c != "" || !strings.Contains(out, "ok  project  ") {
			t.Fatalf("on a terminal: code %q:\n%s", c, out)
		}
		if pdoc, _, _ = config.ReadDoc(pfile); pdoc.String("repo", config.SetupURLKey) != g.repo {
			t.Errorf("the asked URL is not kept")
		}
	})
	t.Run("a placeholder instanceName is mismatch-config", func(t *testing.T) {
		f := newFixture(t)
		path := filepath.Join(f.root, "config", "instance.json")
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(`{ "schemaVersion": 1, "instanceName": "<name>" }`+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if out, c := f.run("", func(o *setup.Options) { o.Instance = "" }); c != "mismatch-config" {
			t.Errorf("code %q:\n%s", c, out)
		}
	})
	t.Run("a bot under the configured name is running whatever --instance says", func(t *testing.T) {
		f := newFixture(t)
		if out, c := f.run(""); c != "" {
			t.Fatalf("first run refused %s:\n%s", c, out)
		}
		addr, err := control.Address(f.root, f.instance)
		if err != nil {
			t.Fatal(err)
		}
		reg := control.NewRegistry()
		reg.Handle("status", func(context.Context, *control.Call) (json.RawMessage, error) {
			return json.RawMessage(`{"config":"ok"}`), nil
		})
		srv, err := control.Listen(addr, reg)
		if err != nil {
			t.Fatal(err)
		}
		go func() { _ = srv.Serve() }()
		defer srv.Close()
		other := f.instance + "-y"
		if out, c := f.run("", func(o *setup.Options) { o.Instance, o.Step = other, "config" }); c != setup.CodeRunning {
			t.Errorf("code %q:\n%s", c, out)
		}
		if c2, err := config.Load(f.root, config.AnyCredential); err != nil || c2.Instance.InstanceName != f.instance {
			t.Errorf("the live file changed: %v", err)
		}
	})
	t.Run("a failed hidden read prints only the error", func(t *testing.T) {
		f := newFixture(t)
		out, c := f.runTTY(&lineThenFail{}, func(o *setup.Options) { o.Yes = false })
		if c == "" || strings.Contains(out, "skip  git-credential") {
			t.Errorf("code %q:\n%s", c, out)
		}
	})
}

func TestGitCredential(t *testing.T) {
	f := newFixture(t)
	if out, c := f.run(""); c != "" {
		t.Fatalf("first run refused %s:\n%s", c, out)
	}
	out, c := f.run("ghp_first\n", func(o *setup.Options) { o.Step = "git-credential" })
	if c != "" || !strings.HasPrefix(out, "ok  git-credential  cred-") {
		t.Fatalf("code %q:\n%s", c, out)
	}
	first := strings.TrimSpace(strings.TrimPrefix(out, "ok  git-credential  "))
	out, c = f.run("ghp_second\n", func(o *setup.Options) { o.Step = "git-credential" })
	if c != "" || !strings.Contains(out, "old  "+first+"\n") || !strings.Contains(out, first+" removed") {
		t.Fatalf("code %q:\n%s", c, out)
	}
	for _, p := range []string{filepath.Join(f.root, "credentials", "store.json"), filepath.Join(f.root, "config", "projects", "demo.json")} {
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(b, []byte("ghp_")) {
			t.Errorf("%s holds the token in the clear", p)
		}
	}
	if out, c := f.run("", func(o *setup.Options) { o.Step = "git-credential" }); c != "needs-input-git-credential" {
		t.Errorf("empty stdin: code %q:\n%s", c, out)
	}
}

func TestSingleWriter(t *testing.T) {
	f := newFixture(t)
	// A setup paused at the key, reading its Enter from a pipe.
	pr, pw := io.Pipe()
	var out syncBuffer
	done := make(chan error, 1)
	go func() {
		done <- setup.Run(context.Background(), f.opts(func(o *setup.Options) { o.Yes = false }), f.env(pr, &out))
	}()
	deadline := time.Now().Add(30 * time.Second)
	for !strings.Contains(out.String(), "ok  key  ") {
		if time.Now().After(deadline) {
			t.Fatalf("setup did not reach the key:\n%s", out.String())
		}
		time.Sleep(20 * time.Millisecond)
	}
	if _, c := f.run(""); c != setup.CodeSetupRunning {
		t.Errorf("second setup: code %q", c)
	}
	addr, err := control.Address(f.root, f.instance)
	if err != nil {
		t.Fatal(err)
	}
	res, err := control.Do(context.Background(), addr, "status", json.RawMessage(`{}`), nil)
	if err != nil || !strings.Contains(string(res), `"config":"setup"`) {
		t.Errorf("status: %s %v", res, err)
	}
	// --check binds nothing: it runs beside the held address.
	if _, c := f.run("", func(o *setup.Options) { o.Check = true }); c == setup.CodeSetupRunning {
		t.Errorf("--check was refused setup-running")
	}
	_, _ = pw.Write([]byte("\n"))
	if err := <-done; err != nil {
		t.Fatalf("paused setup: %v\n%s", err, out.String())
	}

	// A bot on the address: running, with the owner's text.
	reg := control.NewRegistry()
	reg.Handle("status", func(context.Context, *control.Call) (json.RawMessage, error) {
		return json.RawMessage(`{"config":"ok"}`), nil
	})
	srv, err := control.Listen(addr, reg)
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = srv.Serve() }()
	defer srv.Close()
	err = setup.Run(context.Background(), f.opts(), f.env(strings.NewReader(""), io.Discard))
	var ce *control.Error
	if !errors.As(err, &ce) || ce.Code != setup.CodeRunning ||
		ce.Message != "a bot answers at "+addr+"; stop it with broodforge stop in this shell (every open session is interrupted) or use setup --check" {
		t.Errorf("with a bot: %v", err)
	}
	if _, c := f.run("", func(o *setup.Options) { o.Check = true }); c != "" {
		t.Errorf("--check with a bot: code %q", c)
	}
}

func TestJSON(t *testing.T) {
	f := newFixture(t)
	out, c := f.run("", func(o *setup.Options) { o.JSON = true })
	if c != "" {
		t.Fatalf("code %q:\n%s", c, out)
	}
	var key string
	for _, l := range strings.Split(strings.TrimSpace(out), "\n") {
		var obj map[string]string
		if err := json.Unmarshal([]byte(l), &obj); err != nil {
			t.Fatalf("not one object per line: %q: %v", l, err)
		}
		if obj["result"] == "" || obj["step"] == "" {
			t.Errorf("line without result or step: %q", l)
		}
		if obj["step"] == "key" {
			key = obj["key"]
			if len(key) != 64 || obj["path"] == "" {
				t.Errorf("key object: %q", l)
			}
		}
	}
	if key == "" || strings.Count(out, key) != 1 {
		t.Errorf("the key is printed %d times, want once in the key object:\n%s", strings.Count(out, key), out)
	}
	// The key never prints again: the rerun's key object carries none.
	out, _ = f.run("", func(o *setup.Options) { o.JSON = true })
	if strings.Contains(out, key) || strings.Contains(out, `"key":"`) {
		t.Errorf("the rerun printed a key:\n%s", out)
	}
}

// dryRun is --discord dry-run.
func dryRun(o *setup.Options) { o.DryRun = true }

// withFixtures gives the fixture's install directory (Env.Install, paths.install) a
// fixtures/engine, as a checkout has beside bin/.
func withFixtures(f *fixture) *fixture {
	f.t.Helper()
	if err := os.MkdirAll(filepath.Join(f.dir, "fixtures", "engine"), 0o700); err != nil {
		f.t.Fatal(err)
	}
	return f
}

// TestDryRun: the dry-run form (decisions/0027-setup-commands.md, The dry-run form; P27): no
// Discord call, the three skip lines, the placeholder channels and role, engine.fakeFixtures,
// PATH before the pinned engine, no discord check, the dry-run closing line; the rerun only ok
// and skip; live-instance on a live root and for a live export; refused-channels without
// fixtures/engine.
func TestDryRun(t *testing.T) {
	// The scripted Discord is set too: dry-run wins and calls nothing.
	f := withFake(withFixtures(newFixture(t)), fresh())
	// A pinned engine that is no engine: the live order would take it and fail its --version.
	pinned := filepath.Join(f.dir, "programdata", "Broodforge", "engine", "claude.exe")
	if err := os.MkdirAll(filepath.Dir(pinned), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pinned, []byte("not an engine"), 0o600); err != nil {
		t.Fatal(err)
	}
	out, c := f.run("", dryRun)
	if c != "" {
		t.Fatalf("setup --discord dry-run refused %s:\n%s", c, out)
	}
	want := []string{"ok root", "ok config", "ok key", "ok engine", "skip token", "skip application", "skip guild",
		"ok channels", "ok role", "skip git-credential", "ok project", "ok defaults", "ok check", "ok check", "ok next"}
	if got := results(out); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("lines:\n%s\nwant %v", out, want)
	}
	fixtures := filepath.ToSlash(filepath.Join(f.dir, "fixtures", "engine"))
	for _, w := range []string{"skip  token  dry-run", "skip  application  dry-run", "skip  guild  dry-run",
		"ok  engine  " + f.engine + "  2.1.296",
		"ok  channels  dry-run 100 fake, 200 claude-code; fixtures " + fixtures, "ok  role  dry-run 10",
		"ok  next  the dry-run instance is ready: broodforge serve --discord dry-run starts it; a rerun with --discord live adds Discord"} {
		if !strings.Contains(out, w+"\n") {
			t.Errorf("no %q:\n%s", w, out)
		}
	}
	if strings.Contains(out, "discord check") {
		t.Errorf("a discord check line under dry-run:\n%s", out)
	}
	if n := len(f.fake.Script().Calls); n != 0 {
		t.Errorf("dry-run called the scripted Discord: %v", f.fake.Script().Calls)
	}
	cfg, err := config.Load(f.root, config.AnyCredential)
	if err != nil {
		t.Fatalf("config after setup: %v", err)
	}
	in := cfg.Instance
	if in.Discord.Mode != config.DiscordDryRun || strings.Join(in.RoleGroups["trusted"], ",") != config.PlaceholderRole ||
		in.Engine.FakeFixtures == nil || *in.Engine.FakeFixtures != fixtures {
		t.Errorf("instance: mode %s, trusted %v, fakeFixtures %v", in.Discord.Mode, in.RoleGroups["trusted"], in.Engine.FakeFixtures)
	}
	p := cfg.Projects[0]
	if len(p.Channels) != len(config.DryRunChannels) || p.Venue.Category != "" {
		t.Errorf("project channels %v, category %q", p.Channels, p.Venue.Category)
	}
	for _, dc := range config.DryRunChannels {
		ch, ok := p.Channels[dc.ID]
		if !ok || ch.Reach != config.ReachCode || ch.Mode == nil || *ch.Mode != config.ModePrompt || ch.Harness == nil ||
			*ch.Harness != dc.Harness || strings.Join(ch.Roles.Start, ",") != "trusted" ||
			strings.Join(ch.Roles.Contribute, ",") != "trusted" || strings.Join(ch.Roles.Answer, ",") != "trusted" {
			t.Errorf("channel %s: %+v", dc.ID, ch)
		}
	}

	// The rerun: only ok and skip, the Discord steps skip, config/ unchanged.
	before := snapshot(t, f.root)
	out, c = f.run("", dryRun)
	if c != "" {
		t.Fatalf("rerun refused %s:\n%s", c, out)
	}
	for _, r := range results(out) {
		if !strings.HasPrefix(r, "ok ") && !strings.HasPrefix(r, "skip ") {
			t.Errorf("rerun printed %q:\n%s", r, out)
		}
	}
	for _, w := range []string{"skip  channels  dry-run 100 fake, 200 claude-code; fixtures " + fixtures, "skip  role  dry-run 10"} {
		if !strings.Contains(out, w+"\n") {
			t.Errorf("rerun: no %q:\n%s", w, out)
		}
	}
	if after := snapshot(t, f.root); after != before {
		t.Error("the rerun changed config/ or the key")
	}
	// --check under the flag: every line passes.
	if out, c := f.run("", dryRun, func(o *setup.Options) { o.Check = true }); c != "" {
		t.Errorf("--check --discord dry-run refused %s:\n%s", c, out)
	}

	t.Run("live root", func(t *testing.T) {
		f.setKey("instance.json", `"live"`, "discord", "mode")
		before := snapshot(t, f.root)
		for _, check := range []bool{false, true} {
			var out syncBuffer
			err := setup.Run(context.Background(), f.opts(dryRun, func(o *setup.Options) { o.Check = check }), f.env(strings.NewReader(""), &out))
			var ce *control.Error
			if !errors.As(err, &ce) || ce.Code != setup.CodeLiveInstance || ce.Message != "--discord dry-run is refused on a live instance" {
				t.Errorf("check %v: got %v, want live-instance", check, err)
			}
			if out.String() != "" {
				t.Errorf("check %v: lines before the refusal:\n%s", check, out.String())
			}
		}
		if after := snapshot(t, f.root); after != before {
			t.Error("the refusal changed config/ or the key")
		}
	})

	t.Run("live export", func(t *testing.T) {
		live := withFake(newFixture(t), fresh())
		if out, c := live.run(tokA + "\n"); c != "" {
			t.Fatalf("the live setup refused %s:\n%s", c, out)
		}
		export := exportRoot(t, live.root, live.dir)
		g := newFixture(t)
		_, c := g.run("", dryRun, func(o *setup.Options) { o.From = export })
		if c != setup.CodeLiveInstance {
			t.Errorf("code %q, want live-instance", c)
		}
		if _, err := os.Stat(filepath.Dir(g.root)); err == nil {
			t.Error("the refused run created the root")
		}
	})

	t.Run("no fixtures", func(t *testing.T) {
		g := newFixture(t)
		out, c := g.run("", dryRun)
		if c != "refused-channels" {
			t.Fatalf("code %q:\n%s", c, out)
		}
		if l := line(out, "fix  channels  "); l != "fix  channels  build the binary in a checkout: no fixtures/engine beside bin/" {
			t.Errorf("channels line %q", l)
		}
		d, _, err := config.ReadDoc(g.root + "/config/projects/demo.json")
		if err != nil || d == nil {
			t.Fatalf("project file: %v", err)
		}
		if raw, _ := d.Get("channels"); strings.TrimSpace(string(raw)) != "{}" {
			t.Errorf("channels written without fixtures: %s", raw)
		}
	})
}

// TestPlaceholders: a dry-run root turned live against the scripted Discord keeps no 100, 200 or
// 10 and prints no mismatch, no old or new record (decisions/0027-setup-commands.md, the rerun
// rule's placeholder entry); --check without the flag on the dry-run root prints no mismatch.
func TestPlaceholders(t *testing.T) {
	f := withFixtures(newFixture(t))
	if out, c := f.run("", dryRun); c != "" {
		t.Fatalf("setup --discord dry-run refused %s:\n%s", c, out)
	}
	withFake(f, fresh())
	out, c := f.run("", func(o *setup.Options) { o.Check = true })
	if c != setup.CodeCheckFailed || strings.Contains(out, "mismatch") {
		t.Errorf("--check on the dry-run root: code %q:\n%s", c, out)
	}
	for _, s := range discordSteps {
		if !strings.Contains(out, "fix  "+s+"  ") {
			t.Errorf("--check: %s is not fix:\n%s", s, out)
		}
	}
	out, c = f.run(tokA + "\n")
	if c != "" {
		t.Fatalf("the live run refused %s:\n%s", c, out)
	}
	if strings.Contains(out, "mismatch") || line(out, "old  ") != "" || line(out, "new  ") != "" {
		t.Errorf("the live run asked or printed a mismatch:\n%s", out)
	}
	for _, w := range []string{"ok  channels  Sessions 9001, #demo 9002, #demo-notices 9003", "ok  role  Trusted  9004  held by 6000; discord.mode live"} {
		if !strings.Contains(out, w+"\n") {
			t.Errorf("no %q:\n%s", w, out)
		}
	}
	cfg, err := config.Load(f.root, config.AnyCredential)
	if err != nil {
		t.Fatalf("config after the live run: %v", err)
	}
	if got := strings.Join(cfg.Instance.RoleGroups["trusted"], ","); got != "9004" {
		t.Errorf("trusted %s, want 9004", got)
	}
	p := cfg.Projects[0]
	if _, ok := p.Channels["9002"]; !ok || len(p.Channels) != 1 {
		t.Errorf("channels %v, want the lobby 9002 alone", p.Channels)
	}
	for _, dc := range config.DryRunChannels {
		if _, ok := p.Channels[dc.ID]; ok {
			t.Errorf("placeholder channel %s kept", dc.ID)
		}
	}
	if cfg.Instance.Discord.Mode != config.DiscordLive {
		t.Errorf("discord.mode %s, want live", cfg.Instance.Discord.Mode)
	}
	// The live rerun: only ok and skip.
	out, c = f.run("")
	if c != "" {
		t.Fatalf("the live rerun refused %s:\n%s", c, out)
	}
	for _, r := range results(out) {
		if !strings.HasPrefix(r, "ok ") && !strings.HasPrefix(r, "skip ") {
			t.Errorf("live rerun printed %q:\n%s", r, out)
		}
	}
}
