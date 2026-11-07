// SPDX-License-Identifier: GPL-3.0-or-later

package engine

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func drain(h TurnHandle) []EngineEvent {
	var evs []EngineEvent
	for ev := range h.Events {
		evs = append(evs, ev)
	}
	return evs
}

// drainDenying is drain answering every prompt event deny, as engine run's default
// --requests does, so a fake waiting on a request goes on.
func drainDenying(t *testing.T, h TurnHandle) []EngineEvent {
	t.Helper()
	var evs []EngineEvent
	for ev := range h.Events {
		evs = append(evs, ev)
		if ev.Kind == KindPrompt {
			if err := h.Answer(ev.Prompt.RequestID, false, "denied by the test", false); err != nil {
				t.Errorf("answer %s: %v", ev.Prompt.RequestID, err)
			}
		}
	}
	return evs
}

// TestFakeReplaysEveryTurn: every turn of every fixture plays through the fake and ends with
// one exit carrying fixture.json's code; turns play in order.
func TestFakeReplaysEveryTurn(t *testing.T) {
	for _, dir := range fixtureDirs(t) {
		f, err := NewFake(dir, t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		for _, turn := range f.Fixture().Turns {
			h, err := f.StartTurn(TurnRequest{Input: "x"})
			if err != nil {
				t.Fatal(err)
			}
			evs := drainDenying(t, h)
			want := parseTurn(t, dir, turn.N)
			if len(evs) != len(want)+1 {
				t.Fatalf("%s turn %d: %d events, want %d and an exit", dir, turn.N, len(evs), len(want))
			}
			last := evs[len(evs)-1]
			if last.Kind != KindExit || last.Exit.Interrupted || last.Exit.Code == nil || *last.Exit.Code != turn.Exit {
				t.Errorf("%s turn %d: last event %+v, want exit %d", dir, turn.N, last.Exit, turn.Exit)
			}
		}
		if _, err := f.StartTurn(TurnRequest{}); !errors.Is(err, ErrNoTurn) {
			t.Errorf("%s: a turn past the last: %v, want ErrNoTurn", dir, err)
		}
	}
}

func TestFakeWritesReceived(t *testing.T) {
	recv := t.TempDir()
	f, err := NewFake(fixturesRoot+"/defer", recv)
	if err != nil {
		t.Fatal(err)
	}
	f.SetNextTurn(2)
	req := TurnRequest{
		SessionID:        "s0001",
		EngineSessionID:  "e1",
		Cwd:              "/tmp/tree",
		Delivery:         Delivery{Kind: "api-key", CredentialID: "cred-scratch"},
		HookEnv:          []HookVar{{Name: "BROODFORGE_MODE", Value: "prompt"}},
		Answer:           &TurnAnswer{ToolUseID: "toolu_1", Decision: "allow"},
		SystemPromptFile: "/r/workflow/s0001/instructions.md",
	}
	h, err := f.StartTurn(req)
	if err != nil {
		t.Fatal(err)
	}
	drain(h)
	b, err := os.ReadFile(filepath.Join(recv, "received-2.json"))
	if err != nil {
		t.Fatal(err)
	}
	var got TurnRequest
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got.Answer == nil || *got.Answer != *req.Answer || got.Delivery != req.Delivery || got.SystemPromptFile != req.SystemPromptFile || len(got.HookEnv) != 1 {
		t.Errorf("received-2.json: %s", b)
	}
	var keys map[string]json.RawMessage
	_ = json.Unmarshal(b, &keys)
	for _, k := range []string{"delivery", "hookEnv", "systemPromptFile", "answer", "args"} {
		if _, ok := keys[k]; !ok {
			t.Errorf("received-2.json has no %q key", k)
		}
	}
	// args: the flags the request sets on the real engine (RequestArgs).
	if want := `["--permission-mode","default","--resume","e1","--append-system-prompt-file","/r/workflow/s0001/instructions.md"]`; compactJSON(keys["args"]) != want {
		t.Errorf("args %s, want %s", keys["args"], want)
	}
}

// TestFakeInterrupt: with a per-line delay, Interrupt ends the file early and the exit carries
// interrupted: true and no code.
func TestFakeInterrupt(t *testing.T) {
	dir := t.TempDir()
	data, err := os.ReadFile(TurnFile(fixturesRoot+"/basic", 1))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(TurnFile(dir, 1), data, 0o644); err != nil {
		t.Fatal(err)
	}
	fx := `{"claim":"test","delayMs":20,"turns":[{"n":1,"exit":0,"engineSessionId":"e"}]}`
	if err := os.WriteFile(filepath.Join(dir, FixtureFile), []byte(fx), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := NewFake(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	h, err := f.StartTurn(TurnRequest{Input: "x"})
	if err != nil {
		t.Fatal(err)
	}
	first := <-h.Events
	if first.Kind != KindInit {
		t.Fatalf("first event %s, want init", first.Kind)
	}
	if err := h.Interrupt(); err != nil {
		t.Fatal(err)
	}
	if err := h.Interrupt(); err != nil {
		t.Fatalf("a second interrupt: %v", err)
	}
	rest := drain(h)
	if len(rest) == 0 || len(rest)+1 >= len(parseTurn(t, dir, 1))+1 {
		t.Fatalf("%d events after the interrupt, want fewer than the file's", len(rest))
	}
	last := rest[len(rest)-1]
	if last.Kind != KindExit || !last.Exit.Interrupted || last.Exit.Code != nil {
		t.Errorf("last event %+v, want an interrupted exit without a code", last.Exit)
	}
	if _, err := os.Stat(filepath.Join(dir, "received-1.json")); err != nil {
		t.Errorf("received-1.json in the fixture directory: %v", err)
	}
}

// delayedFixture is basic turn 1 in a temp directory with a per-line delay.
func delayedFixture(t *testing.T, delayMs string) string {
	t.Helper()
	dir := t.TempDir()
	data, err := os.ReadFile(TurnFile(fixturesRoot+"/basic", 1))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(TurnFile(dir, 1), data, 0o644); err != nil {
		t.Fatal(err)
	}
	fx := `{"claim":"test","delayMs":` + delayMs + `,"turns":[{"n":1,"exit":0,"engineSessionId":"e"}]}`
	if err := os.WriteFile(filepath.Join(dir, FixtureFile), []byte(fx), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// TestFakeInterruptThenWalkAway: a reader that interrupts and stops reading leaves no
// goroutine behind, whether the replay was waiting on a delay or on a send. The real adapter
// (P7) keeps the same shape.
func TestFakeInterruptThenWalkAway(t *testing.T) {
	before := runtime.NumGoroutine()
	for _, delay := range []string{"0", "20"} {
		dir := delayedFixture(t, delay)
		for i := 0; i < 25; i++ {
			f, err := NewFake(dir, t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			h, err := f.StartTurn(TurnRequest{Input: "x"})
			if err != nil {
				t.Fatal(err)
			}
			<-h.Events
			time.Sleep(25 * time.Millisecond) // let the replay fill the slot and block on a send
			if err := h.Interrupt(); err != nil {
				t.Fatal(err)
			}
		}
	}
	deadline := time.Now().Add(5 * time.Second)
	for runtime.NumGoroutine() > before && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if n := runtime.NumGoroutine(); n > before {
		t.Errorf("%d goroutines after 50 abandoned turns, %d before", n, before)
	}
}

// TestFakeNoDelayAfterLastLine: the delay runs between lines only, so a turn of k lines takes
// k-1 delays and its exit carries the code, not interrupted.
func TestFakeNoDelayAfterLastLine(t *testing.T) {
	dir := delayedFixture(t, "100")
	lines := len(turnLines(t, dir, 1))
	f, err := NewFake(dir, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	h, err := f.StartTurn(TurnRequest{Input: "x"})
	if err != nil {
		t.Fatal(err)
	}
	var sawResult bool
	for ev := range h.Events {
		if ev.Kind == KindResult {
			sawResult = true
		}
		if ev.Kind == KindExit && (ev.Exit.Interrupted || ev.Exit.Code == nil || !sawResult) {
			t.Errorf("exit %+v after result %v", ev.Exit, sawResult)
		}
	}
	if took, max := time.Since(start), time.Duration(lines)*100*time.Millisecond; took >= max {
		t.Errorf("%d lines took %v, want under %v (k-1 delays)", lines, took, max)
	}
}

func TestFakeReceivedDir(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")
	if _, err := NewFake(fixturesRoot+"/basic", missing); !errors.Is(err, ErrReceivedDir) {
		t.Errorf("missing received dir: %v, want ErrReceivedDir", err)
	}
}

// nextUntil reads events up to and including the first of kind, failing after a timeout.
func nextUntil(t *testing.T, h TurnHandle, kind EventKind) []EngineEvent {
	t.Helper()
	var evs []EngineEvent
	timeout := time.After(30 * time.Second)
	for {
		select {
		case ev, ok := <-h.Events:
			if !ok {
				t.Fatalf("the turn ended before a %s event", kind)
			}
			evs = append(evs, ev)
			if ev.Kind == kind {
				return evs
			}
		case <-timeout:
			t.Fatalf("no %s event within 30s", kind)
		}
	}
}

// receivedAnswers is received-<n>.json's answers.
func receivedAnswers(t *testing.T, path string) []json.RawMessage {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var rec struct {
		Answers *[]json.RawMessage `json:"answers"`
	}
	if err := json.Unmarshal(b, &rec); err != nil {
		t.Fatal(err)
	}
	if rec.Answers == nil {
		t.Fatalf("%s has no answers key", path)
	}
	return *rec.Answers
}

// TestFakeWaitsForAnswer (P40c): at prompt-ask-allow's request, which the recording answered
// before its result, the fake waits; an answer given after a delay lets it stream on to the
// exit, and received-1.json holds the line the real adapter would have written. An interrupt
// during the wait ends the replay.
func TestFakeWaitsForAnswer(t *testing.T) {
	dir := fixturesRoot + "/prompt-ask-allow"
	recv := t.TempDir()
	f, err := NewFake(dir, recv)
	if err != nil {
		t.Fatal(err)
	}
	h, err := f.StartTurn(TurnRequest{Input: "x"})
	if err != nil {
		t.Fatal(err)
	}
	got := nextUntil(t, h, KindPrompt)
	p := got[len(got)-1].Prompt
	select {
	case ev := <-h.Events:
		t.Fatalf("the fake streamed %s before an answer", ev.Kind)
	case <-time.After(300 * time.Millisecond):
	}
	if err := h.Answer("no-such-request", true, "", false); err == nil {
		t.Error("an answer to an unknown request was taken")
	}
	if err := h.Answer(p.RequestID, true, "", false); err != nil {
		t.Fatal(err)
	}
	if err := h.Answer(p.RequestID, true, "", false); err == nil {
		t.Error("a second answer to the same request was taken")
	}
	got = append(got, drain(h)...)
	if want := parseTurn(t, dir, 1); len(got) != len(want)+1 {
		t.Fatalf("%d events, want %d and an exit", len(got), len(want))
	}
	if last := got[len(got)-1]; last.Kind != KindExit || last.Exit.Interrupted || last.Exit.Code == nil || *last.Exit.Code != 0 {
		t.Errorf("last event %+v", last.Exit)
	}
	answers := receivedAnswers(t, filepath.Join(recv, "received-1.json"))
	want := `{"type":"control_response","response":{"subtype":"success","request_id":"` + p.RequestID +
		`","response":{"behavior":"allow","updatedInput":{"command":"echo hi","description":"Print hi via echo"}}}}`
	if len(answers) != 1 || compactJSON(answers[0]) != want {
		t.Errorf("answers %s, want [%s]", answers, want)
	}

	// Interrupted while waiting: the replay ends, the exit interrupted.
	f, err = NewFake(dir, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	h, err = f.StartTurn(TurnRequest{Input: "x"})
	if err != nil {
		t.Fatal(err)
	}
	nextUntil(t, h, KindPrompt)
	if err := h.Interrupt(); err != nil {
		t.Fatal(err)
	}
	rest := drain(h)
	if last := rest[len(rest)-1]; len(rest) != 1 || last.Kind != KindExit || !last.Exit.Interrupted {
		t.Errorf("after an interrupt in the wait: %d events, last %+v", len(rest), last.Exit)
	}
}

// TestFakeAbortStreamsOn (P40c): prompt-ask-abort's request was aborted by the engine at a
// stdin close (source config, reason_type other), never answered, so the fake streams on.
func TestFakeAbortStreamsOn(t *testing.T) {
	f, err := NewFake(fixturesRoot+"/prompt-ask-abort", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	h, err := f.StartTurn(TurnRequest{Input: "x"})
	if err != nil {
		t.Fatal(err)
	}
	nextUntil(t, h, KindPrompt)
	select {
	case <-h.Events:
	case <-time.After(2 * time.Second):
		t.Fatal("the fake waited on a request the engine aborted")
	}
	_ = h.Interrupt()
	drain(h)
}

// TestFakeAnswerUntilFirstResult (P40c): the fake takes an answer to an emitted, unanswered
// request until it emits the turn's first result, and refuses it after with errNoRequest.
func TestFakeAnswerUntilFirstResult(t *testing.T) {
	f, err := NewFake(fixturesRoot+"/prompt-ask-abort", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	h, err := f.StartTurn(TurnRequest{Input: "x"})
	if err != nil {
		t.Fatal(err)
	}
	p := nextUntil(t, h, KindPrompt)
	id := p[len(p)-1].Prompt.RequestID
	if err := h.Answer(id, true, "", false); err != nil {
		t.Errorf("an answer before the first result: %v", err)
	}
	drain(h)
	if err := h.Answer(id, true, "", false); !errors.Is(err, errNoRequest) {
		t.Errorf("an answer after the turn: %v, want errNoRequest", err)
	}
}

// TestFakeBackgroundStreamsOn (P40c): prompt-ask-background's request was answered only after
// the turn's first result, so the fake streams on without an answer, the whole recording
// replayed: both results, then the exit; received-1.json's answers is empty.
func TestFakeBackgroundStreamsOn(t *testing.T) {
	dir := fixturesRoot + "/prompt-ask-background"
	recv := t.TempDir()
	f, err := NewFake(dir, recv)
	if err != nil {
		t.Fatal(err)
	}
	h, err := f.StartTurn(TurnRequest{Input: "x"})
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan []EngineEvent)
	go func() { done <- drain(h) }()
	var evs []EngineEvent
	select {
	case evs = <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("the fake waited on a request answered after the result")
	}
	if want := parseTurn(t, dir, 1); len(evs) != len(want)+1 {
		t.Fatalf("%d events, want %d and an exit", len(evs), len(want))
	}
	prompts, results := 0, 0
	for _, ev := range evs {
		switch ev.Kind {
		case KindPrompt:
			prompts++
		case KindResult:
			results++
		case KindInit, KindAssistant, KindToolResult, KindExit, KindRateLimit, KindUnknown:
		}
	}
	if prompts != 1 || results != 2 || evs[len(evs)-1].Kind != KindExit {
		t.Errorf("%d prompts, %d results, last %s", prompts, results, evs[len(evs)-1].Kind)
	}
	if answers := receivedAnswers(t, filepath.Join(recv, "received-1.json")); len(answers) != 0 {
		t.Errorf("answers %s", answers)
	}
}

// compactJSON is raw without insignificant white space, "" when it does not parse.
func compactJSON(raw json.RawMessage) string {
	var b bytes.Buffer
	if json.Compact(&b, raw) != nil {
		return ""
	}
	return b.String()
}
