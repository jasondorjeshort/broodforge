// SPDX-License-Identifier: GPL-3.0-or-later

package cli_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/jasondorjeshort/broodforge/internal/audit"
	"github.com/jasondorjeshort/broodforge/internal/cli/clitest"
	"github.com/jasondorjeshort/broodforge/internal/credentials"
	"github.com/jasondorjeshort/broodforge/internal/spawn/spawntest"
	"github.com/jasondorjeshort/broodforge/internal/store"
)

// The P7 Proof's agent half (08-implementation-plan.md) on the built binary: engine run --real,
// engine record and spawn preflight against the stand-in engine, this test binary run by the
// spawn (spawntest), which replays a recorded turn. No real engine turn runs here.

func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func repoRoot(t *testing.T) string {
	t.Helper()
	r, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	return filepath.ToSlash(r)
}

// writeInstance writes config/instance.json naming the engine at enginePath.
func writeInstance(t *testing.T, root, enginePath, version, backend string) {
	t.Helper()
	cfg := `{ "schemaVersion": 1, "instanceName": "p7-test",
  "credentials": { "keyFile": ` + jsonString(filepath.ToSlash(filepath.Join(root, "test.key"))) + ` },
  "discord": { "mode": "dry-run", "applicationId": "", "guildId": "", "botCredentialId": "", "noticeChannelId": "" },
  "engine": { "path": ` + jsonString(filepath.ToSlash(enginePath)) + `, "version": ` + jsonString(version) + `, "gitBashPath": "", "fakeFixtures": null },
  "spawn": { "backend": ` + jsonString(backend) + `, "interruptTimeoutSeconds": 20 },
  "paths": { "install": ` + jsonString(repoRoot(t)) + ` }, "audit": { "fileSink": null },
  "relay": { "loginByDirectMessage": false }, "roleGroups": {},
  "defaults": { "idleTimeoutMinutes": 60, "promptTimeoutMinutes": 5, "mode": "prompt", "harness": "claude-code" } }
`
	if err := os.MkdirAll(filepath.Join(root, "config"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "config", "instance.json"), []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
}

// realRoot is an instance root whose engine is the stand-in, with user 1001 registered and
// one subscription and one API-key credential; the store is closed again.
func realRoot(t *testing.T) (root, sub, key string) {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	root = t.TempDir()
	writeInstance(t, root, exe, "2.1.296", "same-account")
	keyFile := filepath.Join(root, "test.key")
	if _, err := credentials.Init(keyFile); err != nil {
		t.Fatal(err)
	}
	st, err := store.Create(root)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.Write(func(tx *store.Tx) error {
		_, err := tx.Exec(`INSERT INTO registration (user_id, git_name, git_email, display_name, registered_at)
VALUES ('1001', 'A Person', 'a@example.com', 'A', '2026-10-16T00:00:00Z')`)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	svc, err := credentials.NewService(st, audit.NewWriter(nil), credentials.Config{Root: root, KeyFile: keyFile, EnginePath: exe})
	if err != nil {
		t.Fatal(err)
	}
	s, err := svc.Add(credentials.KindSubscription, "1001", "sub", nil)
	if err != nil {
		t.Fatal(err)
	}
	k, err := svc.Add(credentials.KindAPIKey, "1001", "key", []byte("k-value"))
	if err != nil {
		t.Fatal(err)
	}
	return root, s.ID, k.ID
}

// standinEnv makes the engine the stand-in replaying fixture's turn n, recording into log.
func standinEnv(t *testing.T, fixture string, n int, log string) []string {
	t.Helper()
	turn, err := filepath.Abs(filepath.Join(fixtures, fixture, "turn-"+strconv.Itoa(n)+".jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	return []string{spawntest.Env + "=1", spawntest.EnvFixture + "=" + turn, spawntest.EnvLog + "=" + log}
}

func fakeRun(t *testing.T, dir string, n string) clitest.Result {
	t.Helper()
	return clitest.Run(t, nil, "engine", "run", "--fake", dir, "--turn", n, "--prompt", "x", "--received-dir", t.TempDir())
}

func TestProofEngineRealMatchesFake(t *testing.T) {
	root, sub, _ := realRoot(t)
	tree := t.TempDir()
	log := filepath.Join(t.TempDir(), "standin.json")
	r := clitest.Run(t, standinEnv(t, "basic", 1, log), "--root", root, "engine", "run", "--real", "--credential", sub,
		"--cwd", tree, "--prompt", "Read hello.txt and reply with its one word.", "--expect", "result")
	if r.Code != 0 || r.Stderr != "" {
		t.Fatalf("engine run --real: exit %d stderr %q", r.Code, r.Stderr)
	}
	if f := fakeRun(t, fixtures+"basic", "1"); r.Stdout != f.Stdout {
		t.Errorf("--real and --fake differ on basic turn 1")
	}
	rec, err := spawntest.ReadRecord(log)
	if err != nil {
		t.Fatal(err)
	}
	mcp := repoRoot(t) + "/share/empty/mcp.json"
	// defaults.mode is prompt: the stdio prompt tool (ADR 0029 rule 2).
	if want := []string{"-p", "--output-format", "stream-json", "--verbose", "--input-format", "stream-json", "--permission-mode", "default",
		"--setting-sources", "user", "--strict-mcp-config", "--mcp-config", mcp, "--permission-prompt-tool", "stdio"}; !slices.Equal(rec.Args, want) {
		t.Errorf("args %q", rec.Args)
	}
	home := filepath.ToSlash(filepath.Join(root, "sessions", "engine-run", "home"))
	for k, v := range map[string]string{
		"CLAUDE_CONFIG_DIR": filepath.ToSlash(filepath.Join(root, "harness", sub)), "BROODFORGE_SESSION": "engine-run",
		"BROODFORGE_MODE": "prompt", "BROODFORGE_REACH": "code", "BROODFORGE_TREE": filepath.ToSlash(tree),
		"BROODFORGE_CANONICAL": filepath.ToSlash(tree), "BROODFORGE_HOOK_LOG": home + "/hook.log",
		"GIT_AUTHOR_NAME": "A Person", "GIT_AUTHOR_EMAIL": "a@example.com", "GIT_COMMITTER_NAME": "A Person",
	} {
		if rec.Values[k] != v {
			t.Errorf("%s = %q, want %q", k, rec.Values[k], v)
		}
	}
	if rec.Has("ANTHROPIC_API_KEY") || !rec.Has("CLAUDE_CODE_SUBPROCESS_ENV_SCRUB") {
		t.Errorf("env names %q", rec.EnvNames)
	}
	if !strings.Contains(rec.Stdin, `"text":"Read hello.txt and reply with its one word."`) {
		t.Errorf("stdin %q", rec.Stdin)
	}
	if !strings.Contains(rec.Settings, `hook pre-tool-use"`) || !strings.Contains(rec.Settings, `"timeout": 30`) ||
		!strings.Contains(rec.Settings, `"onFailure": "block"`) {
		t.Errorf("settings %q", rec.Settings)
	}
	if !strings.EqualFold(rec.Cwd, filepath.ToSlash(tree)) {
		t.Errorf("cwd %q", rec.Cwd)
	}
}

func TestProofEngineRealAPIKey(t *testing.T) {
	root, _, key := realRoot(t)
	log := filepath.Join(t.TempDir(), "standin.json")
	r := clitest.Run(t, standinEnv(t, "basic", 2, log), "--root", root, "engine", "run", "--real", "--credential", key,
		"--cwd", t.TempDir(), "--prompt", "x")
	if r.Code != 0 {
		t.Fatalf("exit %d stderr %q", r.Code, r.Stderr)
	}
	rec, err := spawntest.ReadRecord(log)
	if err != nil {
		t.Fatal(err)
	}
	cfg := filepath.ToSlash(filepath.Join(root, "sessions", "engine-run", "home", ".claude"))
	if !rec.Has("ANTHROPIC_API_KEY") || rec.Values["CLAUDE_CONFIG_DIR"] != cfg || !strings.Contains(rec.Settings, "hook pre-tool-use") {
		t.Errorf("record %+v", rec)
	}
	if strings.Contains(r.Stdout, "k-value") || strings.Contains(r.Stderr, "k-value") {
		t.Error("the key was printed")
	}
}

// TestProofEngineRunWorkflow: engine run --real --workflow <dir> hands the hook
// BROODFORGE_WORKFLOW, the copy's absolute path, and the settings file registers both hooks;
// --workflow without --real, or naming no directory, is refused before any spawn.
func TestProofEngineRunWorkflow(t *testing.T) {
	root, sub, _ := realRoot(t)
	log := filepath.Join(t.TempDir(), "standin.json")
	r := clitest.Run(t, standinEnv(t, "basic", 1, log), "--root", root, "engine", "run", "--real", "--credential", sub,
		"--cwd", t.TempDir(), "--prompt", "x", "--workflow", "../../tests/hook/chain")
	if r.Code != 0 {
		t.Fatalf("exit %d stderr %q", r.Code, r.Stderr)
	}
	rec, err := spawntest.ReadRecord(log)
	if err != nil {
		t.Fatal(err)
	}
	if got := rec.Values["BROODFORGE_WORKFLOW"]; got != repoRoot(t)+"/tests/hook/chain" {
		t.Errorf("BROODFORGE_WORKFLOW = %q", got)
	}
	if !strings.Contains(rec.Settings, `hook pre-tool-use"`) || !strings.Contains(rec.Settings, `hook post-tool-use"`) {
		t.Errorf("settings %q", rec.Settings)
	}
	none := filepath.Join(t.TempDir(), "standin.json")
	env := standinEnv(t, "basic", 1, none)
	want(t, clitest.Run(t, env, "--root", root, "engine", "run", "--real", "--credential", sub, "--cwd", t.TempDir(), "--prompt", "x",
		"--workflow", "../../tests/hook/nosuchcopy"), 1, "", "error: bad-workflow: ../../tests/hook/nosuchcopy is not a directory\n")
	want(t, clitest.Run(t, env, "engine", "run", "--fake", fixtures+"basic", "--prompt", "x", "--workflow", "../../tests/hook/chain"), 2, "",
		"error: usage: engine run: --workflow needs --real\n")
	if _, err := os.Stat(none); !os.IsNotExist(err) {
		t.Errorf("an engine was spawned: %v", err)
	}
}

// TestProofEngineRealModeAuto: --mode auto runs the engine with --permission-mode auto and the
// hook with BROODFORGE_MODE=auto (the C40 recording); a mode that is not one of the three is
// refused before any spawn.
func TestProofEngineRealModeAuto(t *testing.T) {
	root, sub, _ := realRoot(t)
	log := filepath.Join(t.TempDir(), "standin.json")
	r := clitest.Run(t, standinEnv(t, "basic", 1, log), "--root", root, "engine", "record", "--credential", sub,
		"--cwd", t.TempDir(), "--out", filepath.Join(t.TempDir(), "rec"), "--prompt", "x", "--mode", "auto")
	if r.Code != 0 {
		t.Fatalf("exit %d stderr %q", r.Code, r.Stderr)
	}
	rec, err := spawntest.ReadRecord(log)
	if err != nil {
		t.Fatal(err)
	}
	if i := slices.Index(rec.Args, "--permission-mode"); i < 0 || rec.Args[i+1] != "auto" {
		t.Errorf("args %q: no --permission-mode auto", rec.Args)
	}
	if got := rec.Values["BROODFORGE_MODE"]; got != "auto" {
		t.Errorf("BROODFORGE_MODE = %q", got)
	}
	none := filepath.Join(t.TempDir(), "standin.json")
	want(t, clitest.Run(t, standinEnv(t, "basic", 1, none), "--root", root, "engine", "run", "--real", "--credential", sub,
		"--cwd", t.TempDir(), "--prompt", "x", "--mode", "yolo"), 2, "", "error: usage: engine run: --mode yolo: want prompt, policy or auto\n")
	want(t, clitest.Run(t, standinEnv(t, "basic", 1, none), "engine", "run", "--fake", fixtures+"basic", "--prompt", "x", "--mode", "auto"), 2, "",
		"error: usage: engine run: --mode needs --real\n")
	if _, err := os.Stat(none); !os.IsNotExist(err) {
		t.Errorf("an engine was spawned: %v", err)
	}
}

// TestProofEngineRealDefaultsModeAuto: with no --mode a standalone real turn follows
// defaults.mode, so an instance whose default is auto runs the engine with --permission-mode
// auto, as a session in that mode would (session.PermissionModeOf).
func TestProofEngineRealDefaultsModeAuto(t *testing.T) {
	root, sub, _ := realRoot(t)
	cfgPath := filepath.Join(root, "config", "instance.json")
	b, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	cfg := strings.Replace(string(b), `"mode": "prompt", "harness"`, `"mode": "auto", "harness"`, 1)
	if cfg == string(b) {
		t.Fatal("the instance config has no prompt default to replace")
	}
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	log := filepath.Join(t.TempDir(), "standin.json")
	r := clitest.Run(t, standinEnv(t, "basic", 1, log), "--root", root, "engine", "run", "--real", "--credential", sub,
		"--cwd", t.TempDir(), "--prompt", "x")
	if r.Code != 0 {
		t.Fatalf("exit %d stderr %q", r.Code, r.Stderr)
	}
	rec, err := spawntest.ReadRecord(log)
	if err != nil {
		t.Fatal(err)
	}
	if i := slices.Index(rec.Args, "--permission-mode"); i < 0 || rec.Args[i+1] != "auto" {
		t.Errorf("args %q: no --permission-mode auto", rec.Args)
	}
	if got := rec.Values["BROODFORGE_MODE"]; got != "auto" {
		t.Errorf("BROODFORGE_MODE = %q", got)
	}
}

// TestProofEngineRealRequestsDefaultsMode (P40c): with defaults.mode prompt, engine run --real
// --requests allow answers the request; with defaults.mode policy it is bad-args, nothing spawned.
func TestProofEngineRealRequestsDefaultsMode(t *testing.T) {
	root, sub, _ := realRoot(t)
	log := filepath.Join(t.TempDir(), "standin.json")
	env := append(standinEnv(t, "prompt-ask-allow", 1, log), spawntest.EnvWait+"=1")
	r := clitest.Run(t, env, "--root", root, "engine", "run", "--real", "--credential", sub,
		"--cwd", t.TempDir(), "--prompt", "x", "--requests", "allow")
	if r.Code != 0 || r.Stderr != "request  "+allowRequest+"  allow\n" {
		t.Errorf("exit %d stderr %q", r.Code, r.Stderr)
	}

	cfgPath := filepath.Join(root, "config", "instance.json")
	b, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	cfg := strings.Replace(string(b), `"mode": "prompt", "harness"`, `"mode": "policy", "harness"`, 1)
	if cfg == string(b) {
		t.Fatal("the instance config has no prompt default to replace")
	}
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	none := filepath.Join(t.TempDir(), "standin.json")
	want(t, clitest.Run(t, standinEnv(t, "basic", 1, none), "--root", root, "engine", "run", "--real", "--credential", sub,
		"--cwd", t.TempDir(), "--prompt", "x", "--requests", "allow"), 1, "",
		"error: bad-args: engine run: --requests allow: the mode is policy, and only a prompt-mode process raises requests\n")
	if _, err := os.Stat(none); !os.IsNotExist(err) {
		t.Errorf("an engine was spawned: %v", err)
	}
}

// TestProofEngineRealRefusals: no credential, or one the store does not hold, exits with its
// error line and no engine is spawned.
func TestProofEngineRealRefusals(t *testing.T) {
	root, _, _ := realRoot(t)
	log := filepath.Join(t.TempDir(), "standin.json")
	env := standinEnv(t, "basic", 1, log)
	tree := t.TempDir()
	want(t, clitest.Run(t, env, "--root", root, "engine", "run", "--real", "--cwd", tree, "--prompt", "x"), 2, "",
		"error: usage: engine run: --credential <id> is required\n")
	want(t, clitest.Run(t, env, "--root", root, "engine", "run", "--real", "--credential", "cred-nope", "--cwd", tree, "--prompt", "x"), 1, "",
		"error: unknown-credential: no credential cred-nope\n")
	want(t, clitest.Run(t, env, "engine", "run", "--real", "--credential", "cred-nope", "--cwd", tree, "--prompt", "x"), 2, "",
		"error: usage: engine run: no instance root: pass --root <path> or set BROODFORGE_ROOT\n")
	want(t, clitest.Run(t, env, "--root", root, "engine", "run", "--real", "--fake", fixtures+"basic", "--prompt", "x"), 2, "",
		"error: usage: engine run: --fake and --real exclude each other\n")
	if _, err := os.Stat(log); !os.IsNotExist(err) {
		t.Errorf("an engine was spawned: %v", err)
	}
}

func TestProofEngineRecordReplays(t *testing.T) {
	root, sub, _ := realRoot(t)
	tree := t.TempDir()
	out := filepath.Join(t.TempDir(), "rec")
	log := filepath.Join(t.TempDir(), "standin.json")
	r := clitest.Run(t, standinEnv(t, "basic", 1, log), "--root", root, "engine", "record", "--credential", sub,
		"--cwd", tree, "--out", out, "--prompt", "x")
	if r.Code != 0 {
		t.Fatalf("record: exit %d stderr %q", r.Code, r.Stderr)
	}
	got, _ := os.ReadFile(filepath.Join(out, "turn-1.jsonl"))
	orig, _ := os.ReadFile(filepath.Join(fixtures, "basic", "turn-1.jsonl"))
	if string(got) != string(orig) {
		t.Error("turn-1.jsonl differs from the engine's stdout")
	}
	if f := fakeRun(t, out, "1"); f.Code != 0 || f.Stdout != r.Stdout {
		t.Errorf("engine run --fake on the recording: exit %d, output differs: %v", f.Code, f.Stdout != r.Stdout)
	}
	// The second turn resumes the first's session.
	r = clitest.Run(t, standinEnv(t, "basic", 2, log), "--root", root, "engine", "record", "--credential", sub,
		"--cwd", tree, "--out", out, "--prompt", "Say the word again.", "--model", "haiku")
	if r.Code != 0 {
		t.Fatalf("record turn 2: exit %d stderr %q", r.Code, r.Stderr)
	}
	rec, err := spawntest.ReadRecord(log)
	if err != nil {
		t.Fatal(err)
	}
	if i := slices.Index(rec.Args, "--model"); i < 0 || rec.Args[i+1] != "haiku" {
		t.Errorf("turn 2 args %q: no --model haiku", rec.Args)
	}
	if i := slices.Index(rec.Args, "--resume"); i < 0 || rec.Args[i+1] != "f0451bd7-4428-40b5-a52e-ad6b604c210a" {
		t.Errorf("turn 2 args %q", rec.Args)
	}
	var fx struct {
		Claim, EngineVersion string
		Turns                []struct {
			N, Exit         int
			EngineSessionID string
		}
	}
	b, err := os.ReadFile(filepath.Join(out, "fixture.json"))
	if err != nil || json.Unmarshal(b, &fx) != nil || fx.EngineVersion != "2.1.296" || len(fx.Turns) != 2 || fx.Turns[1].N != 2 {
		t.Errorf("fixture.json %s %v", b, err)
	}
	if f := fakeRun(t, out, "2"); f.Code != 0 || f.Stdout != r.Stdout {
		t.Error("the replay of turn 2 differs")
	}

	for _, in := range []string{filepath.Join(root, "sessions", "s0001", "rec"), filepath.Join(root, "harness", sub), filepath.Join(tree, "rec")} {
		r = clitest.Run(t, standinEnv(t, "basic", 1, log), "--root", root, "engine", "record", "--credential", sub,
			"--cwd", tree, "--out", in, "--prompt", "x")
		if r.Code != 1 || !strings.HasPrefix(r.Stderr, "error: out-in-session: ") {
			t.Errorf("--out %s: exit %d stderr %q", in, r.Code, r.Stderr)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "harness", sub, "turn-1.jsonl")); !os.IsNotExist(err) {
		t.Errorf("a turn was written into the harness directory: %v", err)
	}
}

func TestProofSpawnPreflight(t *testing.T) {
	root, _, _ := realRoot(t)
	env := []string{spawntest.Env + "=1"}
	r := clitest.Run(t, env, "--root", root, "spawn", "preflight", "--expect", "ok")
	if r.Code != 0 || r.Stderr != "" || !strings.Contains(r.Stdout, "ok  engine-version  2.1.296\n") || !strings.Contains(r.Stdout, "ok  share  ") {
		t.Errorf("preflight: exit %d stdout %q stderr %q", r.Code, r.Stdout, r.Stderr)
	}
	exe, _ := os.Executable()
	writeInstance(t, root, exe, "2.1.300", "same-account")
	r = clitest.Run(t, env, "--root", root, "spawn", "preflight")
	if r.Code != 1 || !strings.HasPrefix(r.Stderr, "error: preflight-engine-version: engine.version is 2.1.300, the binary says 2.1.296") {
		t.Errorf("version mismatch: exit %d stderr %q", r.Code, r.Stderr)
	}
	writeInstance(t, root, filepath.Join(root, "missing", "claude.exe"), "2.1.296", "same-account")
	r = clitest.Run(t, env, "--root", root, "spawn", "preflight")
	if r.Code != 1 || !strings.HasPrefix(r.Stderr, "error: preflight-engine-path: ") {
		t.Errorf("wrong path: exit %d stderr %q", r.Code, r.Stderr)
	}
	writeInstance(t, root, exe, "2.1.296", "linux-sudo-bwrap")
	want(t, clitest.Run(t, env, "--root", root, "spawn", "preflight"), 1, "", "error: preflight-backend: held\n")
}

// TestProofEngineRealAnswer (P40e): --answer resumes a deferred call as a session does: the
// engine resumes the session, the hook's BROODFORGE_ANSWER (and a deny's
// BROODFORGE_ANSWER_REASON) is set and nothing is written to stdin; engine record resumes its
// previous turn the same way.
func TestProofEngineRealAnswer(t *testing.T) {
	root, sub, _ := realRoot(t)
	log := filepath.Join(t.TempDir(), "standin.json")
	r := clitest.Run(t, standinEnv(t, "basic", 2, log), "--root", root, "engine", "run", "--real", "--credential", sub,
		"--cwd", t.TempDir(), "--mode", "prompt", "--resume", "e1", "--answer", "toolu_1:deny", "--reason", "Denied by Broodforge: not now",
		"--expect", "result")
	if r.Code != 0 || r.Stderr != "" {
		t.Fatalf("engine run --real --answer: exit %d stderr %q", r.Code, r.Stderr)
	}
	rec, err := spawntest.ReadRecord(log)
	if err != nil {
		t.Fatal(err)
	}
	if i := slices.Index(rec.Args, "--resume"); i < 0 || rec.Args[i+1] != "e1" {
		t.Errorf("args %q: no --resume e1", rec.Args)
	}
	if rec.Values["BROODFORGE_ANSWER"] != "toolu_1:deny" || rec.Values["BROODFORGE_ANSWER_REASON"] != "Denied by Broodforge: not now" {
		t.Errorf("answer variables %v", rec.Values)
	}
	if rec.Stdin != "" || !rec.StdinClosed {
		t.Errorf("the stand-in read %q (closed %v), want nothing", rec.Stdin, rec.StdinClosed)
	}

	// engine record: turn 2 answers a call of turn 1, resuming turn 1's session (basic's, which
	// is never re-recorded; the stand-in replays a stream, whatever the answer).
	out := filepath.Join(t.TempDir(), "rec")
	tree := t.TempDir()
	if r := clitest.Run(t, standinEnv(t, "basic", 1, log), "--root", root, "engine", "record", "--credential", sub,
		"--cwd", tree, "--out", out, "--prompt", "x", "--mode", "prompt"); r.Code != 0 {
		t.Fatalf("record turn 1: exit %d stderr %q", r.Code, r.Stderr)
	}
	r = clitest.Run(t, standinEnv(t, "basic", 2, log), "--root", root, "engine", "record", "--credential", sub,
		"--cwd", tree, "--out", out, "--answer", "toolu_1:allow", "--mode", "prompt")
	if r.Code != 0 {
		t.Fatalf("record turn 2: exit %d stderr %q", r.Code, r.Stderr)
	}
	if rec, err = spawntest.ReadRecord(log); err != nil {
		t.Fatal(err)
	}
	if i := slices.Index(rec.Args, "--resume"); i < 0 || rec.Args[i+1] != "f0451bd7-4428-40b5-a52e-ad6b604c210a" {
		t.Errorf("turn 2 args %q", rec.Args)
	}
	if rec.Values["BROODFORGE_ANSWER"] != "toolu_1:allow" || rec.Has("BROODFORGE_ANSWER_REASON") || rec.Stdin != "" {
		t.Errorf("turn 2: answer variables %v, stdin %q", rec.Values, rec.Stdin)
	}
	if got, _ := os.ReadFile(filepath.Join(out, "turn-2.jsonl")); string(got) == "" {
		t.Error("turn-2.jsonl is empty")
	}
}

// TestProofEngineRealAnswerRefusals: --answer with --prompt, a malformed answer, an answer with
// no session to resume, and --reason without a deny answer are bad-args; nothing is spawned and
// engine record leaves no turn file. Without --real, --answer is a usage error.
func TestProofEngineRealAnswerRefusals(t *testing.T) {
	root, sub, _ := realRoot(t)
	none := filepath.Join(t.TempDir(), "standin.json")
	env := standinEnv(t, "basic", 1, none)
	tree := t.TempDir()
	run := func(extra ...string) clitest.Result {
		return clitest.Run(t, env, append([]string{"--root", root, "engine", "run", "--real", "--credential", sub, "--cwd", tree}, extra...)...)
	}
	want(t, run("--resume", "e1", "--answer", "toolu_1:allow", "--prompt", "x"), 1, "",
		"error: bad-args: engine run: --prompt and --answer exclude each other: a resume writes no input\n")
	for _, bad := range []string{"toolu_1", "toolu_1:maybe", ":allow", "toolu_1:"} {
		want(t, run("--resume", "e1", "--answer", bad), 1, "", "error: bad-args: engine run: --answer "+bad+": want <tool_use_id>:allow|deny\n")
	}
	want(t, run("--answer", "toolu_1:allow"), 1, "",
		"error: bad-args: engine run: --answer needs --resume <engine session id>: the answer resumes a deferred call\n")
	want(t, run("--resume", "e1", "--reason", "no"), 1, "", "error: bad-args: engine run: --reason needs --answer <tool_use_id>:deny\n")
	want(t, run("--resume", "e1", "--answer", "toolu_1:allow", "--reason", "no"), 1, "",
		"error: bad-args: engine run: --reason needs --answer <tool_use_id>:deny\n")
	want(t, run("--resume", "e1"), 2, "", "error: usage: engine run: --prompt <text> is required\n")
	want(t, clitest.Run(t, env, "engine", "run", "--fake", fixtures+"basic", "--answer", "toolu_1:allow"), 2, "",
		"error: usage: engine run: --answer and --reason need --real\n")

	out := filepath.Join(t.TempDir(), "rec")
	want(t, clitest.Run(t, env, "--root", root, "engine", "record", "--credential", sub, "--cwd", tree, "--out", out,
		"--answer", "toolu_1:allow"), 1, "", "error: bad-args: engine record: --answer needs a previous turn to resume: "+filepath.ToSlash(out)+" has none\n")
	if _, err := os.Stat(filepath.Join(out, "turn-1.jsonl")); !os.IsNotExist(err) {
		t.Errorf("engine record left a turn file: %v", err)
	}
	if _, err := os.Stat(none); !os.IsNotExist(err) {
		t.Errorf("an engine was spawned: %v", err)
	}
}

// allowRequest is prompt-ask-allow's one request: id, tool, agent id.
const allowRequest = "ac51e519-431c-4ad3-a5ff-6e3a99987959  Bash  a26c6f2b725f33c8e"

// TestProofEngineRunFakeRequests (P40c): engine run --fake answers the request the fake waits
// on, deny by default, one stderr line each, so the run never hangs; received-1.json holds the
// line under answers; --requests allow answers allow; another value is bad-args.
func TestProofEngineRunFakeRequests(t *testing.T) {
	for _, policy := range []string{"", "deny", "allow"} {
		recv := t.TempDir()
		args := []string{"engine", "run", "--fake", fixtures + "prompt-ask-allow", "--turn", "1", "--prompt", "x", "--received-dir", recv, "--expect", "result"}
		if policy != "" {
			args = append(args, "--requests", policy)
		}
		r := clitest.Run(t, nil, args...)
		answer := policy
		if answer == "" {
			answer = "deny"
		}
		if r.Code != 0 || r.Stderr != "request  "+allowRequest+"  "+answer+"\n" {
			t.Errorf("--requests %q: exit %d stderr %q", policy, r.Code, r.Stderr)
		}
		b, err := os.ReadFile(filepath.Join(recv, "received-1.json"))
		if err != nil {
			t.Fatal(err)
		}
		var rec struct {
			Answers []struct {
				Response struct {
					Response struct {
						Behavior  string `json:"behavior"`
						Message   string `json:"message"`
						Interrupt *bool  `json:"interrupt"`
					} `json:"response"`
				} `json:"response"`
			} `json:"answers"`
		}
		if err := json.Unmarshal(b, &rec); err != nil {
			t.Fatal(err)
		}
		if len(rec.Answers) != 1 || rec.Answers[0].Response.Response.Behavior != answer {
			t.Errorf("--requests %q: received-1.json %s", policy, b)
		} else if answer == "deny" {
			if d := rec.Answers[0].Response.Response; d.Message != "Denied by Broodforge: answered by the recording's policy" || d.Interrupt == nil || *d.Interrupt {
				t.Errorf("deny %+v", d)
			}
		}
	}
	want(t, clitest.Run(t, nil, "engine", "run", "--fake", fixtures+"prompt-ask-allow", "--prompt", "x", "--received-dir", t.TempDir(),
		"--requests", "maybe"), 1, "", "error: bad-args: engine run: --requests maybe: want allow or deny\n")
}

// TestProofEngineRealRequests (P40c): under --mode prompt the real engine runs with stdin held
// open and --requests answers its request on stdin (the stand-in waits for it, as the engine
// does); engine record writes the policy to fixture.json; --requests under another mode is
// bad-args, nothing spawned.
func TestProofEngineRealRequests(t *testing.T) {
	root, sub, _ := realRoot(t)
	log := filepath.Join(t.TempDir(), "standin.json")
	out := filepath.Join(t.TempDir(), "rec")
	env := append(standinEnv(t, "prompt-ask-allow", 1, log), spawntest.EnvWait+"=1")
	r := clitest.Run(t, env, "--root", root, "engine", "record", "--credential", sub, "--cwd", t.TempDir(), "--out", out,
		"--prompt", "x", "--mode", "prompt", "--requests", "allow")
	if r.Code != 0 || r.Stderr != "request  "+allowRequest+"  allow\n" {
		t.Fatalf("exit %d stderr %q", r.Code, r.Stderr)
	}
	rec, err := spawntest.ReadRecord(log)
	if err != nil {
		t.Fatal(err)
	}
	wantLine := `{"type":"control_response","response":{"subtype":"success","request_id":"ac51e519-431c-4ad3-a5ff-6e3a99987959",` +
		`"response":{"behavior":"allow","updatedInput":{"command":"echo hi","description":"Print hi via echo"}}}}` + "\n"
	if !strings.HasSuffix(rec.Stdin, "\n"+wantLine) || !rec.StdinClosed || !slices.Contains(rec.Args, "--permission-prompt-tool") {
		t.Errorf("the stand-in read %q (closed %v), args %q", rec.Stdin, rec.StdinClosed, rec.Args)
	}
	b, err := os.ReadFile(filepath.Join(out, "fixture.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fx struct {
		Requests string `json:"requests"`
	}
	if err := json.Unmarshal(b, &fx); err != nil || fx.Requests != "allow" {
		t.Errorf("fixture.json %s", b)
	}

	none := filepath.Join(t.TempDir(), "standin.json")
	want(t, clitest.Run(t, standinEnv(t, "basic", 1, none), "--root", root, "engine", "run", "--real", "--credential", sub,
		"--cwd", t.TempDir(), "--prompt", "x", "--mode", "policy", "--requests", "deny"), 1, "",
		"error: bad-args: engine run: --requests deny: the mode is policy, and only a prompt-mode process raises requests\n")
	if _, err := os.Stat(none); !os.IsNotExist(err) {
		t.Errorf("an engine was spawned: %v", err)
	}
}
