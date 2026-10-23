// SPDX-License-Identifier: GPL-3.0-or-later

package spawn

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jasondorjeshort/broodforge/internal/spawn/spawntest"
)

func TestMain(m *testing.M) {
	spawntest.Main()
	os.Exit(m.Run())
}

func code(err error) string {
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	if err != nil {
		return err.Error()
	}
	return ""
}

func TestEnvironment(t *testing.T) {
	parent := []string{"=C:=C:\\x", "PATH=/usr/bin", "CLAUDE_CONFIG_DIR=/owner", "ANTHROPIC_API_KEY=leak",
		"claude_code_entrypoint=x", "BROODFORGE_KEY_FILE=/k", "GIT_AUTHOR_NAME=Owner", "AI_AGENT=1",
		"CLAUDE_CODE_GIT_BASH_PATH=/bash", "HOME=/home/user", "GIT_DIR=/bot/.git", "GIT_WORK_TREE=/bot",
		"GIT_INDEX_FILE=/bot/.git/index", "GIT_CONFIG_COUNT=1", "GIT_CONFIG_KEY_0=core.hooksPath", "GIT_CONFIG_VALUE_0=/x",
		"GIT_SSH_COMMAND=ssh"}
	got := Environment(parent, []Var{{"CLAUDE_CONFIG_DIR", "/cfg"}, {"PATH", "/p"}})
	want := []string{"=C:=C:\\x", "CLAUDE_CODE_GIT_BASH_PATH=/bash", "HOME=/home/user", "GIT_SSH_COMMAND=ssh", "CLAUDE_CONFIG_DIR=/cfg", "PATH=/p"}
	if !slices.Equal(got, want) {
		t.Errorf("Environment\n got %q\nwant %q", got, want)
	}
}

func TestSettings(t *testing.T) {
	var v struct {
		Hooks map[string][]struct {
			Matcher string `json:"matcher"`
			Hooks   []struct {
				Type, Command string
				Timeout       int
			} `json:"hooks"`
		} `json:"hooks"`
	}
	bin := "C:/Program Files/bf/bin/broodforge.exe"
	b := Settings(HookCommand(bin), PostHookCommand(bin))
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatal(err)
	}
	if len(v.Hooks) != 2 {
		t.Fatalf("settings shape: %s", b)
	}
	for event, sub := range map[string]string{"PreToolUse": "pre-tool-use", "PostToolUse": "post-tool-use"} {
		m := v.Hooks[event]
		if len(m) != 1 || m[0].Matcher != "" || len(m[0].Hooks) != 1 {
			t.Fatalf("settings shape: %s", b)
		}
		h := m[0].Hooks[0]
		if h.Type != "command" || h.Timeout != 30 || h.Command != `"C:/Program Files/bf/bin/broodforge.exe" hook `+sub {
			t.Errorf("%s hook %+v", event, h)
		}
	}
	if c := HookCommand(`C:\bf\bin\broodforge.exe`); c != "C:/bf/bin/broodforge.exe hook pre-tool-use" && runtime.GOOS == "windows" {
		t.Errorf("HookCommand %q", c)
	}
}

func TestSettingsPath(t *testing.T) {
	if p, err := SettingsPath(Delivery{Kind: DeliverConfigDir, ConfigDir: "/r/harness/cred-a"}, ""); err != nil || p != "/r/harness/cred-a/settings.json" {
		t.Errorf("config-dir: %q %v", p, err)
	}
	if p, err := SettingsPath(Delivery{Kind: DeliverAPIKey}, "/r/sessions/s1/home"); err != nil || p != "/r/sessions/s1/home/.claude/settings.json" {
		t.Errorf("api-key: %q %v", p, err)
	}
	if _, err := SettingsPath(Delivery{Kind: DeliverAPIKey}, ""); code(err) != CodeBadDelivery {
		t.Errorf("api-key without home: %v", err)
	}
	if _, err := SettingsPath(Delivery{Kind: "token"}, "/h"); code(err) != CodeBadDelivery {
		t.Errorf("unknown kind: %v", err)
	}
}

func TestHarness(t *testing.T) {
	root := t.TempDir()
	var s SameAccount
	dir := HarnessDir(root, "cred-abc")
	if err := s.HarnessCreate(dir); err != nil {
		t.Fatal(err)
	}
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		t.Fatalf("no directory: %v", err)
	}
	if err := s.HarnessCreate(dir); code(err) != CodeHarness {
		t.Errorf("second create: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".credentials.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := s.HarnessRemove(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("still there: %v", err)
	}
	if err := s.HarnessRemove(dir); err != nil {
		t.Errorf("remove of a missing directory: %v", err)
	}
	for _, bad := range []string{root, filepath.Join(root, "sessions", "cred-a"), filepath.Join(root, "harness", "x"), "harness/cred-a"} {
		if err := s.HarnessRemove(bad); code(err) != CodeHarness {
			t.Errorf("remove %q: %v", bad, err)
		}
	}
}

func TestBwrapHeld(t *testing.T) {
	b, err := New(Config{Backend: NameLinuxSudoBwrap})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := b.Start(Spec{}); err == nil || err.Error() != "preflight-backend: held" {
		t.Errorf("Start: %v", err)
	}
	if err := b.HarnessCreate("/r/harness/cred-a"); code(err) != CodeBackendHeld {
		t.Errorf("HarnessCreate: %v", err)
	}
	if c := b.Preflight(); len(c) != 1 || c[0].OK || c[0].Code() != "preflight-backend" || c[0].Detail != "held" {
		t.Errorf("Preflight: %+v", c)
	}
	if _, err := New(Config{Backend: "docker"}); code(err) != CodeBackendHeld {
		t.Errorf("unknown backend: %v", err)
	}
}

// repoRoot is the module root, which holds share/.
func repoRoot(t *testing.T) string {
	t.Helper()
	r, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// standin is a same-account backend whose engine is this test binary as the stand-in.
func standin(t *testing.T, version string) (*SameAccount, string) {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	log := filepath.Join(t.TempDir(), "standin.json")
	t.Setenv(spawntest.Env, "1")
	t.Setenv(spawntest.EnvLog, log)
	s, err := NewSameAccount(Config{Backend: NameSameAccount, EnginePath: exe, EngineVersion: version, Install: repoRoot(t),
		Binary: "/opt/bf/bin/broodforge"})
	if err != nil {
		t.Fatal(err)
	}
	return s, log
}

// run starts spec and reads it to the end.
func run(t *testing.T, s Backend, spec Spec) (string, Exit) {
	t.Helper()
	p, err := s.Start(spec)
	if err != nil {
		t.Fatal(err)
	}
	out, _ := io.ReadAll(p.Stdout)
	_, _ = io.ReadAll(p.Stderr)
	exit := p.Wait()
	p.Close()
	return string(out), exit
}

func TestStartSubscription(t *testing.T) {
	s, log := standin(t, "2.1.296")
	t.Setenv("ANTHROPIC_API_KEY", "leak")
	t.Setenv("BROODFORGE_KEY_FILE", "/k")
	t.Setenv("GIT_DIR", "/bot/.git")
	t.Setenv("GIT_WORK_TREE", "/bot")
	tmp := t.TempDir()
	cfg := filepath.ToSlash(filepath.Join(tmp, "harness", "cred-a"))
	home := filepath.Join(tmp, "sessions", "s1", "home")
	_, exit := run(t, s, Spec{Args: []string{"-p", "x"}, Cwd: tmp, Home: home,
		Delivery: Delivery{Kind: DeliverConfigDir, ConfigDir: cfg},
		Env:      []Var{{"BROODFORGE_SESSION", "s1"}, {"CLAUDE_CONFIG_DIR", "/spec-cannot-override"}}, Stdin: []byte("hello\n")})
	if exit.Code == nil || *exit.Code != 0 || exit.Interrupted {
		t.Fatalf("exit %+v", exit)
	}
	r, err := spawntest.ReadRecord(log)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(r.Args, []string{"-p", "x"}) || r.Stdin != "hello\n" || r.Values["CLAUDE_CONFIG_DIR"] != cfg || r.Values["BROODFORGE_SESSION"] != "s1" {
		t.Errorf("record %+v", r)
	}
	for _, name := range []string{"ANTHROPIC_API_KEY", "BROODFORGE_KEY_FILE", "GIT_DIR", "GIT_WORK_TREE"} {
		if r.Has(name) {
			t.Errorf("%s reached the engine", name)
		}
	}
	for _, name := range []string{"CLAUDE_CODE_SUBPROCESS_ENV_SCRUB", "DISABLE_AUTOUPDATER"} {
		if !r.Has(name) {
			t.Errorf("%s missing", name)
		}
	}
	if r.Settings != string(Settings("/opt/bf/bin/broodforge hook pre-tool-use", "/opt/bf/bin/broodforge hook post-tool-use")) {
		t.Errorf("settings %q", r.Settings)
	}
	if st, err := os.Stat(home); err != nil || !st.IsDir() {
		t.Errorf("home not made: %v", err)
	}
}

func TestStartAPIKey(t *testing.T) {
	s, log := standin(t, "2.1.296")
	tmp := t.TempDir()
	home := filepath.ToSlash(filepath.Join(tmp, "home"))
	_, exit := run(t, s, Spec{Cwd: tmp, Home: home, Delivery: Delivery{Kind: DeliverAPIKey, APIKey: []byte("k")}})
	if exit.Code == nil || *exit.Code != 0 {
		t.Fatalf("exit %+v", exit)
	}
	r, err := spawntest.ReadRecord(log)
	if err != nil {
		t.Fatal(err)
	}
	if !r.Has("ANTHROPIC_API_KEY") || r.Values["CLAUDE_CONFIG_DIR"] != home+"/.claude" || !strings.Contains(r.Settings, "hook pre-tool-use") {
		t.Errorf("record %+v", r)
	}
	if _, err := s.Start(Spec{Cwd: tmp, Home: home, Delivery: Delivery{Kind: DeliverAPIKey}}); code(err) != CodeBadDelivery {
		t.Errorf("no key: %v", err)
	}
}

func TestStartExitCodeAndRefusals(t *testing.T) {
	s, _ := standin(t, "2.1.296")
	t.Setenv(spawntest.EnvExit, "3")
	tmp := t.TempDir()
	d := Delivery{Kind: DeliverConfigDir, ConfigDir: filepath.Join(tmp, "cfg")}
	if _, exit := run(t, s, Spec{Cwd: tmp, Delivery: d}); exit.Code == nil || *exit.Code != 3 {
		t.Errorf("exit %+v", exit)
	}
	rel, err := NewSameAccount(Config{EnginePath: "claude.exe", Binary: "/b"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := rel.Start(Spec{Cwd: tmp, Delivery: d}); code(err) != CodeSpawnFailed {
		t.Errorf("relative engine path: %v", err)
	}
}

func TestInterruptAfterExit(t *testing.T) {
	s, _ := standin(t, "2.1.296")
	tmp := t.TempDir()
	p, err := s.Start(Spec{Cwd: tmp, Delivery: Delivery{Kind: DeliverConfigDir, ConfigDir: filepath.Join(tmp, "cfg")}})
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.ReadAll(p.Stdout)
	select {
	case <-p.Done():
	case <-time.After(30 * time.Second):
		t.Fatal("no exit")
	}
	if err := p.Interrupt(); err != nil {
		t.Errorf("Interrupt after the exit: %v", err)
	}
	p.Close()
}

func checks(cs []Check) map[string]Check {
	m := map[string]Check{}
	for _, c := range cs {
		m[c.Name] = c
	}
	return m
}

func TestPreflight(t *testing.T) {
	s, _ := standin(t, "2.1.296")
	cs := checks(s.Preflight())
	for _, name := range []string{"backend", "engine-path", "engine-version", "share", "account"} {
		if !cs[name].OK {
			t.Errorf("%s: %+v", name, cs[name])
		}
	}
	if cs["engine-version"].Detail != "2.1.296" {
		t.Errorf("version %+v", cs["engine-version"])
	}
	if _, ok := cs["git-bash"]; ok != (runtime.GOOS == "windows") {
		t.Errorf("git-bash check on %s: %v", runtime.GOOS, ok)
	}

	s.c.EngineVersion = "2.1.300"
	if c := checks(s.Preflight())["engine-version"]; c.OK || !strings.Contains(c.Detail, "engine.version is 2.1.300, the binary says 2.1.296") {
		t.Errorf("mismatch: %+v", c)
	}
	s.c.EnginePath = filepath.Join(t.TempDir(), "claude.exe")
	cs = checks(s.Preflight())
	if c := cs["engine-path"]; c.OK || c.Code() != "preflight-engine-path" {
		t.Errorf("missing engine: %+v", c)
	}
	if _, ok := cs["engine-version"]; ok {
		t.Error("the version check ran without an engine")
	}
	if runtime.GOOS == "windows" {
		s.c.EnginePath = filepath.Join(t.TempDir(), "claude.cmd")
		if c := checks(s.Preflight())["engine-path"]; c.OK || !strings.Contains(c.Detail, "not the engine's .exe") {
			t.Errorf(".cmd shim: %+v", c)
		}
		s.c.GitBashPath = filepath.Join(t.TempDir(), "bash.exe")
		if c := checks(s.Preflight())["git-bash"]; c.OK {
			t.Errorf("missing git bash: %+v", c)
		}
	}
	s.install = t.TempDir()
	if c := checks(s.Preflight())["share"]; c.OK {
		t.Errorf("empty install: %+v", c)
	}
}
