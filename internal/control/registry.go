// SPDX-License-Identifier: GPL-3.0-or-later

package control

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"sync"
)

// Handler serves one command. Args is the request's args object ({} when the request sent
// none). It may call call.Emit any number of times before returning; the returned value is the
// final result (nil means null). An *Error chooses the wire error code; any other error is sent
// as "internal". ctx is cancelled when the client disconnects or the server closes.
type Handler func(ctx context.Context, call *Call) (json.RawMessage, error)

// Call is one request in flight on the server.
type Call struct {
	ID   int64
	Cmd  string
	Args json.RawMessage

	mu    sync.Mutex
	done  bool
	emit  func(json.RawMessage) error
	after []func()
}

// AfterResponse runs f once the call's final line has been written to the client (or failed
// to be): how `stop` answers before the bot closes its own server. f runs on the call's own
// goroutine, which Server.Close waits for, so it must signal, never wait for Close itself.
func (c *Call) AfterResponse(f func()) {
	c.mu.Lock()
	c.after = append(c.after, f)
	c.mu.Unlock()
}

// responded runs the AfterResponse functions, in order.
func (c *Call) responded() {
	c.mu.Lock()
	after := c.after
	c.after = nil
	c.mu.Unlock()
	for _, f := range after {
		f()
	}
}

// ErrCallDone is returned by Emit after the handler has returned.
var ErrCallDone = errors.New("control: emit after the final response")

// Emit sends one {"id":n,"event":{...}} line. event must be one JSON value.
func (c *Call) Emit(event json.RawMessage) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.done {
		return ErrCallDone
	}
	return c.emit(event)
}

func (c *Call) finish() {
	c.mu.Lock()
	c.done = true
	c.mu.Unlock()
}

// Registry maps "<group>.<command>" to its handler. Every registry holds the built-in
// control.echo; a service adds its commands with Handle before the server starts.
type Registry struct {
	handlers map[string]Handler
}

// NewRegistry returns a registry holding the built-in commands.
func NewRegistry() *Registry {
	r := &Registry{handlers: map[string]Handler{}}
	r.Handle(CmdEcho, echo)
	return r
}

// Handle registers h for cmd. It panics on a duplicate or an empty name: both are programming
// errors found by the first test that builds the registry.
func (r *Registry) Handle(cmd string, h Handler) {
	if cmd == "" || h == nil {
		panic("control: Handle with an empty command or a nil handler")
	}
	if _, dup := r.handlers[cmd]; dup {
		panic("control: command registered twice: " + cmd)
	}
	r.handlers[cmd] = h
}

// Commands lists the registered command names, sorted.
func (r *Registry) Commands() []string {
	names := make([]string, 0, len(r.handlers))
	for n := range r.handlers {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

func (r *Registry) lookup(cmd string) Handler { return r.handlers[cmd] }
