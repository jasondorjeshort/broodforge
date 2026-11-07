// SPDX-License-Identifier: GPL-3.0-or-later

package engine

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
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
			want := lines(t, drainDenying(t, h))

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

// TestRecordingKeepsUnknownLine (P29): the recorder writes the stdout line before the parser
// shrinks it, so a recorded turn holds the whole commands_changed line while its event carries
// only the length and head.
func TestRecordingKeepsUnknownLine(t *testing.T) {
	lines := commandsChanged(t)
	eng, _ := standinEngine(t, TurnFile(fixturesRoot+"/basic", 1))
	t.Setenv(spawntest.EnvExit, "0")
	rec, err := NewRecording(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	eng.Raw = rec.Write
	h, err := eng.StartTurn(subscriptionRequest(t))
	if err != nil {
		t.Fatal(err)
	}
	evs := drain(h)
	if err := rec.Finish(evs[len(evs)-1].Exit, "s", "2.1.296"); err != nil {
		t.Fatal(err)
	}
	recorded, err := os.ReadFile(rec.Path())
	if err != nil {
		t.Fatal(err)
	}
	for i, line := range lines {
		if !bytes.Contains(recorded, []byte("\n"+line+"\n")) {
			t.Errorf("%s lacks commands_changed line %d (%d bytes)", rec.Path(), i+1, len(line))
		}
	}
	var shrunk []*UnknownEvent
	for _, ev := range evs {
		if ev.Kind == KindUnknown && ev.Unknown.Subtype == "commands_changed" {
			shrunk = append(shrunk, ev.Unknown)
		}
	}
	if len(shrunk) != len(lines) {
		t.Fatalf("%d commands_changed events, want %d", len(shrunk), len(lines))
	}
	for i, u := range shrunk {
		if u.Bytes != len(lines[i]) || len(u.Head) > UnknownHeadBytes || !strings.HasPrefix(lines[i], u.Head) {
			t.Errorf("commands_changed event %d: %d bytes, head %d bytes", i+1, u.Bytes, len(u.Head))
		}
	}
}

func TestArgsAndUserLine(t *testing.T) {
	base := []string{"-p", "--output-format", "stream-json", "--verbose", "--input-format", "stream-json",
		"--permission-mode", "default", "--setting-sources", "user", "--strict-mcp-config", "--mcp-config", "/bf/share/empty/mcp.json"}
	if got := Args(TurnRequest{}, "/bf/share/empty/mcp.json"); !slices.Equal(got, base) {
		t.Errorf("Args\n got %q\nwant %q", got, base)
	}
	got := Args(TurnRequest{EngineSessionID: "s1", Model: "haiku", Effort: "high", SystemPromptFile: "/r/workflow/s1/instructions.md",
		AgentsFile: "/r/workflow/s1/agents.json"}, "/m")
	if want := []string{"--resume", "s1", "--model", "haiku", "--effort", "high", "--append-system-prompt-file",
		"/r/workflow/s1/instructions.md", "--agents", "/r/workflow/s1/agents.json"}; !slices.Equal(got[len(base):], want) {
		t.Errorf("Args tail %q", got[len(base):])
	}
	// The file flag without agents: no --agents; the retired text flag never.
	got = Args(TurnRequest{SystemPromptFile: "/i.md"}, "/m")
	if want := []string{"--append-system-prompt-file", "/i.md"}; !slices.Equal(got[len(base):], want) || slices.Contains(got, "--append-system-prompt") {
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

// stdinTap records what the adapter does with a prompt-mode process's stdin writer: each line
// written and each close, in order (the user line is spawn's, written before the tap).
type stdinTap struct {
	w io.WriteCloser

	mu     sync.Mutex
	ops    []string
	closes int
}

func (s *stdinTap) Write(b []byte) (int, error) {
	s.mu.Lock()
	s.ops = append(s.ops, string(b))
	s.mu.Unlock()
	return s.w.Write(b)
}

func (s *stdinTap) Close() error {
	s.mu.Lock()
	s.ops = append(s.ops, "close")
	s.closes++
	s.mu.Unlock()
	return s.w.Close()
}

func (s *stdinTap) snapshot() ([]string, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.ops), s.closes
}

// tapped is a standin engine on fixture whose prompt-mode stdin goes through a tap.
func tapped(t *testing.T, fixture string, exitTimeout time.Duration) (*ClaudeCode, string, *stdinTap) {
	t.Helper()
	cc, log := standinEngine(t, fixture)
	cc.ExitTimeout = exitTimeout
	tap := &stdinTap{}
	cc.wrapStdin = func(w io.WriteCloser) io.WriteCloser { tap.w = w; return tap }
	return cc, log, tap
}

func promptRequest(t *testing.T) TurnRequest {
	t.Helper()
	req := subscriptionRequest(t)
	req.Mode = "prompt"
	return req
}

// exitOf is the turn's last event's exit, failing when the last event is not one.
func exitOf(t *testing.T, evs []EngineEvent) *ExitEvent {
	t.Helper()
	if len(evs) == 0 || evs[len(evs)-1].Kind != KindExit {
		t.Fatalf("the turn did not end with an exit: %d events", len(evs))
	}
	return evs[len(evs)-1].Exit
}

// TestPromptStdinLifetime (P40c; ADR 0029 rule 2 and Amendments 2026-11-07 decision 1): a
// prompt-mode process's stdin stays open through an open request, closes exactly once (after
// the deny-interrupt lines at the first result, or at exit without one), the backend's
// interrupt comes at once when a request was open at the result, at ExitTimeout when no exit
// follows a clean close, and not at all when the engine exits; a policy process keeps the
// write-then-close.
func TestPromptStdinLifetime(t *testing.T) {
	user := string(UserLine("x"))
	t.Run("open-request-answered", func(t *testing.T) {
		cc, log, tap := tapped(t, filepath.Join(fixturesRoot, "prompt-ask-allow", "turn-1.jsonl"), time.Minute)
		t.Setenv(spawntest.EnvWait, "1")
		h, err := cc.StartTurn(promptRequest(t))
		if err != nil {
			t.Fatal(err)
		}
		got := nextUntil(t, h, KindPrompt)
		p := got[len(got)-1].Prompt
		time.Sleep(300 * time.Millisecond) // the request stays open; the stand-in waits on it
		if ops, closes := tap.snapshot(); len(ops) != 0 || closes != 0 {
			t.Fatalf("stdin touched while the request was open: %q", ops)
		}
		if err := h.Answer(p.RequestID, true, "", false); err != nil {
			t.Fatal(err)
		}
		evs := append(got, drain(h)...)
		allow := string(controlResponse(p.RequestID, p.Input, true, "", false))
		if ops, closes := tap.snapshot(); !slices.Equal(ops, []string{allow, "close"}) || closes != 1 {
			t.Errorf("stdin ops %q", ops)
		}
		if exit := exitOf(t, evs); exit.Interrupted || exit.Code == nil || *exit.Code != 0 {
			t.Errorf("exit %+v: the engine exited by itself after the close", exit)
		}
		if r, err := spawntest.ReadRecord(log); err != nil || r.Stdin != user+allow || !r.StdinClosed {
			t.Errorf("the stand-in read %q (closed %v), err %v", r.Stdin, r.StdinClosed, err)
		}
		if err := h.Answer(p.RequestID, true, "", false); err == nil {
			t.Error("an answer after the close was taken")
		}
	})
	t.Run("open-at-result", func(t *testing.T) {
		cc, _, tap := tapped(t, filepath.Join(fixturesRoot, "prompt-ask-background", "turn-1.jsonl"), time.Minute)
		t.Setenv(spawntest.EnvHold, "sleep")
		t.Setenv(spawntest.EnvSleep, "60s")
		start := time.Now()
		h, err := cc.StartTurn(promptRequest(t))
		if err != nil {
			t.Fatal(err)
		}
		evs := drain(h)
		took := time.Since(start)
		var id string
		for _, ev := range evs {
			if ev.Kind == KindPrompt {
				id = ev.Prompt.RequestID
				break
			}
		}
		deny := string(controlResponse(id, nil, false, turnEndedMessage, true))
		if ops, closes := tap.snapshot(); !slices.Equal(ops, []string{deny, "close"}) || closes != 1 {
			t.Errorf("stdin ops %q, want the deny-interrupt line then one close", ops)
		}
		if exit := exitOf(t, evs); !exit.Interrupted || took > 30*time.Second {
			t.Errorf("exit %+v after %v: want the interrupt at once (ExitTimeout is a minute)", exit, took)
		}
		if !slices.ContainsFunc(evs, func(ev EngineEvent) bool { return ev.Kind == KindResult }) {
			t.Error("no result event")
		}
	})
	t.Run("none-open-clean-close", func(t *testing.T) {
		cc, _, tap := tapped(t, filepath.Join(fixturesRoot, "basic", "turn-1.jsonl"), time.Minute)
		h, err := cc.StartTurn(promptRequest(t))
		if err != nil {
			t.Fatal(err)
		}
		evs := drain(h)
		if ops, closes := tap.snapshot(); !slices.Equal(ops, []string{"close"}) || closes != 1 {
			t.Errorf("stdin ops %q", ops)
		}
		if exit := exitOf(t, evs); exit.Interrupted {
			t.Errorf("exit %+v: interrupted with no request open", exit)
		}
	})
	t.Run("exit-timeout", func(t *testing.T) {
		cc, _, tap := tapped(t, filepath.Join(fixturesRoot, "basic", "turn-1.jsonl"), 500*time.Millisecond)
		t.Setenv(spawntest.EnvHold, "sleep")
		t.Setenv(spawntest.EnvSleep, "60s")
		start := time.Now()
		h, err := cc.StartTurn(promptRequest(t))
		if err != nil {
			t.Fatal(err)
		}
		evs := drain(h)
		took := time.Since(start)
		if ops, closes := tap.snapshot(); !slices.Equal(ops, []string{"close"}) || closes != 1 {
			t.Errorf("stdin ops %q", ops)
		}
		if exit := exitOf(t, evs); !exit.Interrupted || took < 500*time.Millisecond || took > 30*time.Second {
			t.Errorf("exit %+v after %v: want the interrupt at ExitTimeout", exit, took)
		}
	})
	t.Run("exit-without-result", func(t *testing.T) {
		cc, _, tap := tapped(t, filepath.Join(fixturesRoot, "prompt-ask-kill", "turn-1.jsonl"), time.Minute)
		t.Setenv(spawntest.EnvHold, "sleep")
		h, err := cc.StartTurn(promptRequest(t))
		if err != nil {
			t.Fatal(err)
		}
		evs := drain(h)
		if ops, closes := tap.snapshot(); !slices.Equal(ops, []string{"close"}) || closes != 1 {
			t.Errorf("stdin ops %q, want one close at exit", ops)
		}
		if exit := exitOf(t, evs); exit.Interrupted {
			t.Errorf("exit %+v", exit)
		}
	})
	t.Run("policy-write-then-close", func(t *testing.T) {
		cc, log, tap := tapped(t, filepath.Join(fixturesRoot, "basic", "turn-1.jsonl"), time.Minute)
		req := subscriptionRequest(t)
		req.Mode = "policy"
		h, err := cc.StartTurn(req)
		if err != nil {
			t.Fatal(err)
		}
		evs := drain(h)
		if tap.w != nil {
			t.Error("a policy process handed its stdin to the adapter")
		}
		if r, err := spawntest.ReadRecord(log); err != nil || r.Stdin != user || !r.StdinClosed {
			t.Errorf("the stand-in read %q (closed %v), err %v", r.Stdin, r.StdinClosed, err)
		}
		if exit := exitOf(t, evs); exit.Interrupted {
			t.Errorf("exit %+v", exit)
		}
		if err := h.Answer("r", true, "", false); err == nil {
			t.Error("a policy process took an answer")
		}
	})
}

// TestArgsPromptTool (P40c; ADR 0029 rule 2): --permission-prompt-tool stdio under prompt
// only, on the command line and in RequestArgs; --permission-prompts under no mode.
func TestArgsPromptTool(t *testing.T) {
	for _, mode := range []string{"prompt", "policy", "auto", ""} {
		req := TurnRequest{Mode: mode, EngineSessionID: "s1"}
		if mode == "auto" {
			req.PermissionMode = PermissionAuto
		}
		for name, args := range map[string][]string{"Args": Args(req, "/m"), "RequestArgs": RequestArgs(req)} {
			i := slices.Index(args, "--permission-prompt-tool")
			if has := i >= 0 && i+1 < len(args) && args[i+1] == "stdio"; has != (mode == "prompt") {
				t.Errorf("mode %q %s %q: --permission-prompt-tool stdio %v", mode, name, args, has)
			}
			if slices.Contains(args, "--permission-prompts") {
				t.Errorf("mode %q %s %q carries --permission-prompts", mode, name, args)
			}
		}
	}
}

// TestAnswerLines (P40c): Answer's three lines are byte for byte the ones the engine accepted
// in the 2026-11-07 spike (c42-allow, c42-deny, c49): the allow's updatedInput the request's
// input unchanged, no updatedPermissions.
func TestAnswerLines(t *testing.T) {
	in := json.RawMessage(`{"command":"echo hi","description":"Print hi via echo"}`)
	for _, c := range []struct {
		allow, interrupt bool
		id, msg, want    string
	}{
		{true, false, "ac51e519-431c-4ad3-a5ff-6e3a99987959", "",
			`{"type":"control_response","response":{"subtype":"success","request_id":"ac51e519-431c-4ad3-a5ff-6e3a99987959","response":{"behavior":"allow","updatedInput":{"command":"echo hi","description":"Print hi via echo"}}}}`},
		{false, false, "af2e31c2-29f0-48cf-b992-4814ad04e9c2", "Denied by Broodforge: a person in this session denied this call",
			`{"type":"control_response","response":{"subtype":"success","request_id":"af2e31c2-29f0-48cf-b992-4814ad04e9c2","response":{"behavior":"deny","message":"Denied by Broodforge: a person in this session denied this call","interrupt":false}}}`},
		{false, true, "821bdfe0-faa6-4f28-9551-3a4386050c48", "Denied by Broodforge: the session was interrupted",
			`{"type":"control_response","response":{"subtype":"success","request_id":"821bdfe0-faa6-4f28-9551-3a4386050c48","response":{"behavior":"deny","message":"Denied by Broodforge: the session was interrupted","interrupt":true}}}`},
	} {
		if got := string(controlResponse(c.id, in, c.allow, c.msg, c.interrupt)); got != c.want+"\n" {
			t.Errorf("got  %s\nwant %s", got, c.want)
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
	rec.Requests = "deny" // engine record --mode prompt's policy (P40c)
	if err := rec.Finish(&ExitEvent{Interrupted: true}, "s2", "2.1.296"); err != nil {
		t.Fatal(err)
	}
	fx, err := ReadFixture(dir)
	if err != nil {
		t.Fatal(err)
	}
	if fx.Claim != RecordedClaim || fx.EngineVersion != "2.1.296" || fx.DelayMs != 0 || fx.Requests != "deny" || len(fx.Turns) != 1 || fx.Turns[0] != (FixtureTurn{N: 2, Exit: 1, EngineSessionID: "s2"}) {
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
