// SPDX-License-Identifier: GPL-3.0-or-later

package engine

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/jasondorjeshort/broodforge/internal/spawn"
	"github.com/jasondorjeshort/broodforge/internal/spawn/spawntest"
)

// The claude-code engine against the stand-in (spawntest): this test binary, run by the spawn
// as the engine, replays a recorded turn on stdout, so the adapter is proven on every fixture
// without a real turn.

func TestMain(m *testing.M) {
	spawntest.Main()
	os.Exit(m.Run())
}

// standinEngine is a ClaudeCode whose backend runs this test binary as the stand-in, replaying
// fixture (a turn file; "" prints nothing). It returns the stand-in's record path.
func standinEngine(t *testing.T, fixture string) (*ClaudeCode, string) {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	install, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	log := filepath.Join(t.TempDir(), "standin.json")
	t.Setenv(spawntest.Env, "1")
	t.Setenv(spawntest.EnvLog, log)
	if fixture != "" {
		abs, err := filepath.Abs(fixture)
		if err != nil {
			t.Fatal(err)
		}
		t.Setenv(spawntest.EnvFixture, abs)
	}
	b, err := spawn.NewSameAccount(spawn.Config{Backend: spawn.NameSameAccount, EnginePath: exe, Install: install, Binary: "/bf/bin/broodforge"})
	if err != nil {
		t.Fatal(err)
	}
	return &ClaudeCode{Backend: b}, log
}

// subscriptionRequest is a turn on a temporary tree and config directory.
func subscriptionRequest(t *testing.T) TurnRequest {
	t.Helper()
	tmp := t.TempDir()
	return TurnRequest{Cwd: tmp, Input: "x", Delivery: Delivery{Kind: DeliveryConfigDir, Path: filepath.Join(tmp, "harness", "cred-a")}}
}

func lines(t *testing.T, evs []EngineEvent) []string {
	t.Helper()
	var out []string
	for _, ev := range evs {
		b, err := ev.MarshalJSON()
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, string(b))
	}
	return out
}

// TestClaudeCodeMatchesFake: every recorded turn read through the real adapter yields the
// fake's events, and the recorder writes the stdout back byte for byte.
func TestClaudeCodeMatchesFake(t *testing.T) {
	for _, dir := range fixtureDirs(t) {
		fx, err := ReadFixture(dir)
		if err != nil {
			t.Fatal(err)
		}
		for _, turn := range fx.Turns {
			name := filepath.Base(dir) + "/" + filepath.Base(TurnFile(dir, turn.N))
			fake, err := NewFake(dir, t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			fake.SetNextTurn(turn.N)
			h, err := fake.StartTurn(TurnRequest{})
			if err != nil {
				t.Fatal(err)
			}
			want := lines(t, drain(h))

			cc, _ := standinEngine(t, TurnFile(dir, turn.N))
			t.Setenv(spawntest.EnvExit, "0")
			rec, err := NewRecording(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			cc.Raw = rec.Write
			h, err = cc.StartTurn(subscriptionRequest(t))
			if err != nil {
				t.Fatal(err)
			}
			got := drain(h)
			// The fake's exit carries the recorded code; the stand-in exits 0.
			if !slices.Equal(lines(t, got[:len(got)-1]), want[:len(want)-1]) {
				t.Errorf("%s: events differ from the fake's", name)
			}
			if last := got[len(got)-1]; last.Kind != KindExit || last.Exit.Code == nil || *last.Exit.Code != 0 || last.Exit.Interrupted {
				t.Errorf("%s: last event %s", name, lines(t, got[len(got)-1:]))
			}
			if err := rec.Finish(got[len(got)-1].Exit, "s", "2.1.296"); err != nil {
				t.Fatal(err)
			}
			recorded, _ := os.ReadFile(rec.Path())
			original, _ := os.ReadFile(TurnFile(dir, turn.N))
			if !bytes.Equal(recorded, original) {
				t.Errorf("%s: the recording differs from the stdout (%d bytes, want %d)", name, len(recorded), len(original))
			}
		}
	}
}

func TestArgsAndUserLine(t *testing.T) {
	base := []string{"-p", "--output-format", "stream-json", "--verbose", "--input-format", "stream-json",
		"--permission-mode", "default", "--setting-sources", "user", "--strict-mcp-config", "--mcp-config", "/bf/share/empty/mcp.json"}
	if got := Args(TurnRequest{}, "/bf/share/empty/mcp.json"); !slices.Equal(got, base) {
		t.Errorf("Args\n got %q\nwant %q", got, base)
	}
	got := Args(TurnRequest{EngineSessionID: "s1", Model: "haiku", SystemPromptAppend: "one call at a time"}, "/m")
	if want := []string{"--resume", "s1", "--model", "haiku", "--append-system-prompt", "one call at a time"}; !slices.Equal(got[len(base):], want) {
		t.Errorf("Args tail %q", got[len(base):])
	}
	if l := string(UserLine("a <b> & c")); l != `{"type":"user","message":{"role":"user","content":[{"type":"text","text":"a <b> & c"}]}}`+"\n" {
		t.Errorf("UserLine %s", l)
	}
}

func TestClaudeCodeSpec(t *testing.T) {
	cc, log := standinEngine(t, "")
	req := subscriptionRequest(t)
	home := filepath.ToSlash(filepath.Join(req.Cwd, "home"))
	req.HookEnv = []HookVar{{Name: "BROODFORGE_SESSION", Value: "s1"}, {Name: hookLogVar, Value: home + "/hook.log"}}
	req.GitIdentity = GitIdentity{Name: "A Person", Email: "a@example.com"}
	req.Answer = &TurnAnswer{ToolUseID: "toolu_1", Decision: "allow"}
	req.EngineSessionID = "s-1"
	h, err := cc.StartTurn(req)
	if err != nil {
		t.Fatal(err)
	}
	drain(h)
	r, err := spawntest.ReadRecord(log)
	if err != nil {
		t.Fatal(err)
	}
	if r.Stdin != "" {
		t.Errorf("a resume with an answer wrote stdin %q", r.Stdin)
	}
	for k, v := range map[string]string{"BROODFORGE_ANSWER": "toolu_1:allow", "BROODFORGE_SESSION": "s1", "GIT_AUTHOR_NAME": "A Person",
		"GIT_COMMITTER_EMAIL": "a@example.com", "CLAUDE_CONFIG_DIR": filepath.ToSlash(req.Delivery.Path)} {
		if r.Values[k] != v {
			t.Errorf("%s = %q, want %q", k, r.Values[k], v)
		}
	}
	if !slices.Contains(r.Args, "--resume") {
		t.Errorf("args %q", r.Args)
	}
	if st, err := os.Stat(home); err != nil || !st.IsDir() {
		t.Errorf("the session home was not made: %v", err)
	}

	// A first turn writes the one user line.
	req.Answer, req.EngineSessionID = nil, ""
	h, err = cc.StartTurn(req)
	if err != nil {
		t.Fatal(err)
	}
	drain(h)
	if r, _ = spawntest.ReadRecord(log); r.Stdin != string(UserLine("x")) {
		t.Errorf("stdin %q", r.Stdin)
	}
}

func TestClaudeCodeDeliveryRefusals(t *testing.T) {
	cc, _ := standinEngine(t, "")
	req := subscriptionRequest(t)
	req.Delivery = Delivery{Kind: DeliveryAPIKey, CredentialID: "cred-k"}
	if _, err := cc.StartTurn(req); !errors.Is(err, ErrDelivery) {
		t.Errorf("api-key without a secret source: %v", err)
	}
	req.Delivery = Delivery{Kind: "token"}
	if _, err := cc.StartTurn(req); !errors.Is(err, ErrDelivery) {
		t.Errorf("unknown kind: %v", err)
	}
}

// TestClaudeCodeInterrupt: an engine still running after its first lines ends on Interrupt,
// the exit marked interrupted, and a reader that walks away after Interrupt leaves no block.
func TestClaudeCodeInterrupt(t *testing.T) {
	cc, _ := standinEngine(t, filepath.Join(fixturesRoot, "basic", "turn-1.jsonl"))
	t.Setenv(spawntest.EnvSleep, "60s")
	h, err := cc.StartTurn(subscriptionRequest(t))
	if err != nil {
		t.Fatal(err)
	}
	first := <-h.Events
	if first.Kind == KindExit {
		t.Fatal("the stand-in exited at once")
	}
	start := time.Now()
	if err := h.Interrupt(); err != nil {
		t.Fatal(err)
	}
	var exit *ExitEvent
	timeout := time.After(30 * time.Second)
	for exit == nil {
		select {
		case ev, ok := <-h.Events:
			if !ok {
				t.Fatal("closed without an exit")
			}
			if ev.Kind == KindExit {
				exit = ev.Exit
			}
		case <-timeout:
			t.Fatal("no exit after the interrupt")
		}
	}
	if !exit.Interrupted {
		t.Errorf("exit %+v", exit)
	}
	t.Logf("exit %s after the interrupt", time.Since(start).Round(time.Millisecond))

	// Walk away after Interrupt: the turn still ends (the events channel closes).
	h, err = cc.StartTurn(subscriptionRequest(t))
	if err != nil {
		t.Fatal(err)
	}
	<-h.Events
	if err := h.Interrupt(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(2 * time.Second)
	closed := time.After(30 * time.Second)
	for {
		select {
		case _, ok := <-h.Events:
			if !ok {
				return
			}
		case <-closed:
			t.Fatal("the events channel never closed")
		}
	}
}

func TestRecordingNumbersAndResumes(t *testing.T) {
	src := filepath.Join(fixturesRoot, "basic")
	dir := t.TempDir()
	b, err := os.ReadFile(TurnFile(src, 1))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(TurnFile(dir, 1), b, 0o644); err != nil {
		t.Fatal(err)
	}
	// No fixture.json: the resume comes from turn 1's init line.
	rec, err := NewRecording(dir)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Turn() != 2 || rec.Resume() != "f0451bd7-4428-40b5-a52e-ad6b604c210a" {
		t.Errorf("turn %d resume %q", rec.Turn(), rec.Resume())
	}
	rec.Write([]byte("{\"type\":\"system\"}\n"))
	if err := rec.Finish(&ExitEvent{Interrupted: true}, "s2", "2.1.296"); err != nil {
		t.Fatal(err)
	}
	fx, err := ReadFixture(dir)
	if err != nil {
		t.Fatal(err)
	}
	if fx.Claim != RecordedClaim || fx.EngineVersion != "2.1.296" || fx.DelayMs != 0 || len(fx.Turns) != 1 || fx.Turns[0] != (FixtureTurn{N: 2, Exit: 1, EngineSessionID: "s2"}) {
		raw, _ := json.Marshal(fx)
		t.Errorf("fixture.json %s", raw)
	}
	// fixture.json now names turn 2's session for turn 3.
	rec, err = NewRecording(dir)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Turn() != 3 || rec.Resume() != "s2" {
		t.Errorf("turn %d resume %q", rec.Turn(), rec.Resume())
	}
	rec.Abort()
	if _, err := os.Stat(TurnFile(dir, 3)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Abort left %s", TurnFile(dir, 3))
	}
}
