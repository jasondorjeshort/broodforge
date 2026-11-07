// SPDX-License-Identifier: GPL-3.0-or-later

package engine

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/jasondorjeshort/broodforge/internal/spawn"
)

// Delivery kinds of TurnRequest.Delivery (03-architecture.md, The engine adapter).
const (
	DeliveryConfigDir = "config-dir"
	DeliveryAPIKey    = "api-key"
)

// hookLogVar is the hook variable whose directory is the session home (03-hook-contract.md,
// Environment: sessions/<id>/home/hook.log).
const hookLogVar = "BROODFORGE_HOOK_LOG"

// answerVar carries a person's answer on a resume (03-hook-contract.md).
const answerVar = "BROODFORGE_ANSWER"

// eventBuffer is the events channel's capacity.
const eventBuffer = 64

// ClaudeCode is the claude-code engine: one engine process per turn through the spawn
// backend (ADR 0020, Mechanism), its stdout parsed line by line into EngineEvents.
type ClaudeCode struct {
	Backend spawn.Backend
	// Secret returns an API key credential's value, for the engine process only.
	Secret func(credentialID string) ([]byte, error)
	// Raw, when set, receives every stdout line as the engine wrote it, newline included:
	// the recorder's sink.
	Raw func(line []byte)
	// Stderr receives the engine's stderr; nil discards it.
	Stderr io.Writer
	// ExitTimeout is how long a prompt-mode process may run after its stdin is closed at a
	// result with no request open before the backend's interrupt ends it (ADR 0029,
	// Amendments 2026-11-07 decision 1); 0 is spawn.DefaultInterrupt. Set from
	// spawn.interruptTimeoutSeconds; no key of its own.
	ExitTimeout time.Duration

	// wrapStdin, when set, wraps a prompt-mode process's stdin writer: the lifetime test's tap.
	wrapStdin func(io.WriteCloser) io.WriteCloser
}

// turnEndedMessage is the deny the adapter writes at result for every request still open
// (03-hook-contract-prompt-tool.md, The process; a placeholder until the owner words it).
const turnEndedMessage = "Denied by Broodforge: the turn ended before anyone answered"

// errStdinClosed is Answer's refusal once the turn's stdin is closed, or on a process that
// never held it open.
var errStdinClosed = errors.New("the engine's stdin is closed")

// promptStdin is a prompt-mode process's stdin and the requests open on it (ADR 0029 rule 2;
// the companion, The process): closed exactly once, at the first result or at exit, never
// while a request is open, the open ones answered deny with interrupt true first.
type promptStdin struct {
	mu    sync.Mutex
	w     io.WriteCloser // nil once closed
	ended bool           // the first result was handled
	open  map[string]json.RawMessage
	order []string                     // open request ids as they came
	warn  func(step string, err error) // a failed step, one stderr line; never nil
}

// lockedWriter serialises the adapter's own stderr lines with the engine's copied stderr.
type lockedWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func (l *lockedWriter) Write(b []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.w.Write(b)
}

// warnTo is the adapter's warn: one line "engine: <step>: <error>" to w.
func warnTo(w io.Writer) func(string, error) {
	return func(step string, err error) {
		if err != nil {
			_, _ = fmt.Fprintf(w, "engine: %s: %v\n", step, err)
		}
	}
}

// request records a prompt event as open while stdin is.
func (s *promptStdin) request(p *PromptEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.w == nil || s.ended {
		return
	}
	if _, dup := s.open[p.RequestID]; !dup {
		s.order = append(s.order, p.RequestID)
	}
	s.open[p.RequestID] = p.Input
}

// answer writes one control_response for an open request.
func (s *promptStdin) answer(id string, allow bool, message string, interrupt bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.w == nil {
		return errStdinClosed
	}
	input, ok := s.open[id]
	if !ok {
		return fmt.Errorf("%w: %s", errNoRequest, id)
	}
	delete(s.open, id)
	s.order = slices.DeleteFunc(s.order, func(o string) bool { return o == id })
	_, err := s.w.Write(controlResponse(id, input, allow, message, interrupt))
	return err
}

// atResult is the first result's sequence: each open request answered deny with interrupt
// true, then stdin closed. It reports whether any request was open; false on a later result.
func (s *promptStdin) atResult() (handled, wasOpen bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ended || s.w == nil {
		return false, false
	}
	s.ended = true
	for _, id := range s.order {
		if _, err := s.w.Write(controlResponse(id, nil, false, turnEndedMessage, true)); err != nil {
			s.warn("deny request "+id+" at the first result", err)
		}
	}
	wasOpen = len(s.order) > 0
	s.order, s.open = nil, map[string]json.RawMessage{}
	s.warn("close stdin at the first result", s.w.Close())
	s.w = nil
	return true, wasOpen
}

// closeAtExit closes stdin when the process exits without a result.
func (s *promptStdin) closeAtExit() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.w != nil {
		s.warn("close stdin at exit", s.w.Close())
		s.w = nil
	}
}

var _ Engine = (*ClaudeCode)(nil)

// Name is the harness name configuration uses.
func (c *ClaudeCode) Name() string { return "claude-code" }

// Args is ADR 0020's command line after the binary: the fixed flags, the empty MCP map, then
// RequestArgs. --effort is the engine's own flag (`claude --help` on 2.1.310: low, medium, high,
// xhigh, max); the session sends only a level policy.Efforts lists.
func Args(req TurnRequest, mcpConfig string) []string {
	args := []string{"-p", "--output-format", "stream-json", "--verbose", "--input-format", "stream-json",
		"--permission-mode", permissionMode(req), "--setting-sources", "user", "--strict-mcp-config", "--mcp-config", mcpConfig}
	return append(args, requestFlags(req)...)
}

// permissionMode is --permission-mode's value: the request's, else default (ADR 0020 as amended
// 2026-11-01: set per process from the session's mode, so a switch takes effect at the next
// process). --permission-prompts is never set, in any mode (the amendment's rule; ADR 0029
// rule 2).
func permissionMode(req TurnRequest) string {
	if req.PermissionMode == "" {
		return PermissionDefault
	}
	return req.PermissionMode
}

// RequestArgs are the flags the request sets: --permission-mode always, then
// --permission-prompt-tool stdio on a prompt-mode process (ADR 0029 rule 2), then --resume,
// --model, --effort, --append-system-prompt-file and --agents, each when the request names it.
// The text form --append-system-prompt is retired (ADR 0025 rule 5): the session's prompt is a
// file in its workflow copy, passed on every process. The fake records these as
// received-<n>.json's `args`; Args puts --permission-mode among the fixed flags.
func RequestArgs(req TurnRequest) []string {
	return append([]string{"--permission-mode", permissionMode(req)}, requestFlags(req)...)
}

// requestFlags are RequestArgs after --permission-mode.
func requestFlags(req TurnRequest) []string {
	var args []string
	if req.Mode == modePrompt {
		args = append(args, "--permission-prompt-tool", "stdio")
	}
	if req.EngineSessionID != "" {
		args = append(args, "--resume", req.EngineSessionID)
	}
	if req.Model != "" {
		args = append(args, "--model", req.Model)
	}
	if req.Effort != "" {
		args = append(args, "--effort", req.Effort)
	}
	if req.SystemPromptFile != "" {
		args = append(args, "--append-system-prompt-file", req.SystemPromptFile)
	}
	if req.AgentsFile != "" {
		args = append(args, "--agents", req.AgentsFile)
	}
	return args
}

// UserLine is the one stream-json user message written to stdin: one text block.
func UserLine(text string) []byte {
	type block struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	v := struct {
		Type    string `json:"type"`
		Message struct {
			Role    string  `json:"role"`
			Content []block `json:"content"`
		} `json:"message"`
	}{Type: "user"}
	v.Message.Role = "user"
	v.Message.Content = []block{{"text", text}}
	b, err := encodeJSON(v)
	if err != nil {
		panic(err) // strings only
	}
	return append(b, '\n')
}

// ErrDelivery is returned by StartTurn for a request whose credential cannot be delivered.
var ErrDelivery = errors.New("delivery")

// spec turns the request into the spawn's process description.
func (c *ClaudeCode) spec(req TurnRequest) (spawn.Spec, error) {
	s := spawn.Spec{Args: Args(req, c.Backend.MCPConfig()), Cwd: req.Cwd, StdinOpen: req.Mode == modePrompt}
	switch req.Delivery.Kind {
	case DeliveryConfigDir:
		s.Delivery = spawn.Delivery{Kind: spawn.DeliverConfigDir, ConfigDir: req.Delivery.Path}
	case DeliveryAPIKey:
		if c.Secret == nil {
			return s, errors.Join(ErrDelivery, errors.New("no secret source for "+req.Delivery.CredentialID))
		}
		key, err := c.Secret(req.Delivery.CredentialID)
		if err != nil {
			return s, err
		}
		s.Delivery = spawn.Delivery{Kind: spawn.DeliverAPIKey, APIKey: key}
	default:
		return s, errors.Join(ErrDelivery, errors.New("delivery kind "+string(mustJSON(req.Delivery.Kind))))
	}
	for _, v := range req.HookEnv {
		s.Env = append(s.Env, spawn.Var{Name: v.Name, Value: v.Value})
		if v.Name == hookLogVar && v.Value != "" {
			s.Home = filepath.Dir(v.Value)
		}
	}
	if id := req.GitIdentity; id.Name != "" || id.Email != "" {
		s.Env = append(s.Env, spawn.Var{Name: "GIT_AUTHOR_NAME", Value: id.Name}, spawn.Var{Name: "GIT_AUTHOR_EMAIL", Value: id.Email},
			spawn.Var{Name: "GIT_COMMITTER_NAME", Value: id.Name}, spawn.Var{Name: "GIT_COMMITTER_EMAIL", Value: id.Email})
	}
	if req.Answer != nil {
		// A deferred call's resume takes no stdin line (C27); the answer is in the environment.
		s.Env = append(s.Env, spawn.Var{Name: answerVar, Value: req.Answer.ToolUseID + ":" + req.Answer.Decision})
	} else if req.Input != "" {
		s.Stdin = UserLine(req.Input)
	}
	return s, nil
}

func mustJSON(s string) []byte {
	b, _ := json.Marshal(s)
	return b
}

// StartTurn starts the engine. Its stdout is read as fast as it comes, whoever reads Events,
// so the engine never blocks on its pipe. Events carries the parsed lines, then an exit, then
// closes. Until Interrupt every event waits for the reader; after it, an event that finds the
// channel full drops the oldest one waiting (the reader may have walked away, as the fake's
// contract allows), so the turn always ends.
func (c *ClaudeCode) StartTurn(req TurnRequest) (TurnHandle, error) {
	spec, err := c.spec(req)
	if err != nil {
		return TurnHandle{}, err
	}
	p, err := c.Backend.Start(spec)
	if err != nil {
		return TurnHandle{}, err
	}
	stderr := c.Stderr
	if stderr == nil {
		stderr = io.Discard
	}
	stderr = &lockedWriter{w: stderr}
	warn := warnTo(stderr)
	var errDone sync.WaitGroup
	errDone.Go(func() { _, _ = io.Copy(stderr, p.Stderr) })

	var in *promptStdin
	if spec.StdinOpen && p.Stdin != nil {
		w := p.Stdin
		if c.wrapStdin != nil {
			w = c.wrapStdin(w)
		}
		in = &promptStdin{w: w, open: map[string]json.RawMessage{}, warn: warn}
	}
	exitTimeout := c.ExitTimeout
	if exitTimeout <= 0 {
		exitTimeout = spawn.DefaultInterrupt
	}
	var exitTimer *time.Timer

	q := newQueue()
	go func() {
		r := bufio.NewReaderSize(p.Stdout, 64<<10)
		for {
			line, err := r.ReadBytes('\n')
			if len(line) > 0 {
				if c.Raw != nil {
					c.Raw(line)
				}
				if t := bytes.TrimRight(line, "\r\n"); len(bytes.TrimSpace(t)) > 0 {
					evs := ParseLine(t)
					if in != nil {
						for _, ev := range evs {
							exitTimer = in.observe(ev, p, exitTimeout, exitTimer)
						}
					}
					q.push(evs...)
				}
			}
			if err != nil {
				break
			}
		}
		exit := p.Wait()
		if exitTimer != nil {
			exitTimer.Stop()
		}
		if in != nil {
			in.closeAtExit()
		}
		errDone.Wait()
		p.Close()
		q.push(EngineEvent{Kind: KindExit, Exit: &ExitEvent{Code: exit.Code, Signal: exit.Signal, Interrupted: exit.Interrupted}})
		q.close()
	}()

	events := make(chan EngineEvent, eventBuffer)
	stop := make(chan struct{})
	var once sync.Once
	go func() {
		defer close(events)
		for {
			ev, ok := q.pop()
			if !ok {
				return
			}
			select {
			case events <- ev:
				continue
			case <-stop:
			}
			select {
			case events <- ev:
			default:
				select {
				case <-events:
				default:
				}
				events <- ev // the pump is the only sender, so the slot just freed is free
			}
		}
	}()
	return TurnHandle{
		Events: events,
		Interrupt: func() error {
			var err error
			once.Do(func() {
				err = p.Interrupt()
				close(stop)
			})
			return err
		},
		Answer: func(requestID string, allow bool, message string, interrupt bool) error {
			if in == nil {
				return errStdinClosed
			}
			return in.answer(requestID, allow, message, interrupt)
		},
	}, nil
}

// observe is the reader's step for one event of a prompt-mode process, before the event is
// pushed: a prompt event opens its request; the first result runs atResult, then calls the
// backend's interrupt at once when a request was open, else arms the exit timer, which it
// returns. Events after the result change nothing here.
func (s *promptStdin) observe(ev EngineEvent, p *spawn.Process, exitTimeout time.Duration, timer *time.Timer) *time.Timer {
	switch ev.Kind {
	case KindPrompt:
		s.request(ev.Prompt)
	case KindResult:
		handled, wasOpen := s.atResult()
		switch {
		case !handled:
		case wasOpen:
			s.warn("interrupt after the first result", p.Interrupt())
		default:
			timer = time.AfterFunc(exitTimeout, func() { s.warn("interrupt at the exit timer", p.Interrupt()) })
		}
	case KindInit, KindAssistant, KindToolResult, KindExit, KindRateLimit, KindUnknown:
	}
	return timer
}

// queue is an unbounded FIFO of events between the stdout reader and the pump.
type queue struct {
	mu     sync.Mutex
	cond   *sync.Cond
	items  []EngineEvent
	closed bool
}

func newQueue() *queue {
	q := &queue{}
	q.cond = sync.NewCond(&q.mu)
	return q
}

func (q *queue) push(evs ...EngineEvent) {
	q.mu.Lock()
	q.items = append(q.items, evs...)
	q.mu.Unlock()
	q.cond.Signal()
}

func (q *queue) close() {
	q.mu.Lock()
	q.closed = true
	q.mu.Unlock()
	q.cond.Signal()
}

// pop waits for the next event; false once the queue is closed and empty.
func (q *queue) pop() (EngineEvent, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	for len(q.items) == 0 && !q.closed {
		q.cond.Wait()
	}
	if len(q.items) == 0 {
		return EngineEvent{}, false
	}
	ev := q.items[0]
	q.items = q.items[1:]
	return ev, true
}
