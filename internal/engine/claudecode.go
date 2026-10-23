// SPDX-License-Identifier: GPL-3.0-or-later

package engine

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"path/filepath"
	"sync"

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
}

var _ Engine = (*ClaudeCode)(nil)

// Name is the harness name configuration uses.
func (c *ClaudeCode) Name() string { return "claude-code" }

// Args is ADR 0020's command line after the binary: the fixed flags, the empty MCP map, then
// RequestArgs. --effort is the engine's own flag (`claude --help` on 2.1.309: low, medium, high,
// xhigh, max); the session sends only a level policy.Efforts lists.
func Args(req TurnRequest, mcpConfig string) []string {
	args := []string{"-p", "--output-format", "stream-json", "--verbose", "--input-format", "stream-json",
		"--permission-mode", "default", "--setting-sources", "user", "--strict-mcp-config", "--mcp-config", mcpConfig}
	return append(args, RequestArgs(req)...)
}

// RequestArgs are the flags the request sets: --resume, --model, --effort,
// --append-system-prompt-file and --agents, each when the request names it. The text form
// --append-system-prompt is retired (ADR 0025 rule 5): the session's prompt is a file in its
// workflow copy, passed on every process. The fake records these as received-<n>.json's `args`.
func RequestArgs(req TurnRequest) []string {
	var args []string
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
	s := spawn.Spec{Args: Args(req, c.Backend.MCPConfig()), Cwd: req.Cwd}
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
	var errDone sync.WaitGroup
	errDone.Go(func() { _, _ = io.Copy(stderr, p.Stderr) })

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
					q.push(ParseLine(t)...)
				}
			}
			if err != nil {
				break
			}
		}
		exit := p.Wait()
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
	}, nil
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
