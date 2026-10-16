// SPDX-License-Identifier: GPL-3.0-or-later

package control

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sync"
)

// listener is the host's socket or pipe listener (address_unix.go, pipe_windows.go).
type listener interface {
	// Accept waits for the next connection; after Close it returns errListenerClosed.
	Accept() (io.ReadWriteCloser, error)
	Close() error
}

var errListenerClosed = errors.New("control: listener closed")

var instanceNameRE = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)

// ValidInstanceName reports whether name can name a pipe: 1 to 64 characters of letters,
// digits, dot, underscore and hyphen, starting with a letter or digit.
func ValidInstanceName(name string) bool { return instanceNameRE.MatchString(name) }

// Address returns the control socket of the instance at root: <root>/data/control.sock on
// Linux, \\.\pipe\broodforge-<instanceName> on Windows. instanceName is checked on every host.
func Address(root, instanceName string) (string, error) {
	if !ValidInstanceName(instanceName) {
		return "", fmt.Errorf("instanceName %q: want 1 to 64 of [A-Za-z0-9._-], starting with a letter or digit", instanceName)
	}
	return hostAddress(root, instanceName), nil
}

// Server serves the commands of one registry on one control address.
type Server struct {
	reg  *Registry
	ln   listener
	addr string

	ctx    context.Context
	cancel context.CancelFunc

	mu     sync.Mutex
	closed bool
	conns  map[io.ReadWriteCloser]struct{}
	wg     sync.WaitGroup
}

// Listen claims addr. If another server already answers there it returns an *Error with code
// start-control ("another bot owns <addr>"); a stale unix socket file is replaced.
func Listen(addr string, reg *Registry) (*Server, error) {
	ln, err := listen(addr)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Server{reg: reg, ln: ln, addr: addr, ctx: ctx, cancel: cancel, conns: map[io.ReadWriteCloser]struct{}{}}, nil
}

// Addr is the address the server listens on.
func (s *Server) Addr() string { return s.addr }

// Serve accepts connections until Close; it returns nil after Close.
func (s *Server) Serve() error {
	for {
		c, err := s.ln.Accept()
		if err != nil {
			if errors.Is(err, errListenerClosed) {
				return nil
			}
			return err
		}
		s.mu.Lock()
		if s.closed {
			s.mu.Unlock()
			_ = c.Close()
			return nil
		}
		s.conns[c] = struct{}{}
		s.wg.Add(1)
		s.mu.Unlock()
		go s.serveConn(c)
	}
}

// Close stops accepting, cancels every call in flight, closes every connection and waits for
// the handlers to return.
func (s *Server) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	conns := make([]io.ReadWriteCloser, 0, len(s.conns))
	for c := range s.conns {
		conns = append(conns, c)
	}
	s.mu.Unlock()
	s.cancel()
	err := s.ln.Close()
	for _, c := range conns {
		_ = c.Close()
	}
	s.wg.Wait()
	return err
}

// serveConn reads request lines until EOF; each request runs in its own goroutine, so one
// connection may carry several ids at once. The connection's end cancels its calls.
func (s *Server) serveConn(c io.ReadWriteCloser) {
	defer s.wg.Done()
	ctx, cancel := context.WithCancel(s.ctx)
	var wmu sync.Mutex
	write := func(l Line) error {
		b, err := encodeLine(l)
		if err != nil {
			return err
		}
		wmu.Lock()
		defer wmu.Unlock()
		_, err = c.Write(b)
		return err
	}
	var calls sync.WaitGroup
	sc := bufio.NewScanner(c)
	sc.Buffer(make([]byte, 0, 64<<10), maxLine)
	for sc.Scan() {
		var req Request
		if err := json.Unmarshal(sc.Bytes(), &req); err != nil {
			_ = write(errorLine(req.ID, NewError(CodeBadRequest, err.Error())))
			continue
		}
		if req.Cmd == "" {
			_ = write(errorLine(req.ID, NewError(CodeBadRequest, "no cmd")))
			continue
		}
		h := s.reg.lookup(req.Cmd)
		if h == nil {
			_ = write(errorLine(req.ID, NewError(CodeUnknownCommand, req.Cmd)))
			continue
		}
		if len(req.Args) == 0 || string(req.Args) == "null" {
			req.Args = json.RawMessage("{}")
		}
		call := &Call{ID: req.ID, Cmd: req.Cmd, Args: req.Args}
		call.emit = func(ev json.RawMessage) error { return write(eventLine(call.ID, ev)) }
		calls.Add(1)
		go func() {
			defer calls.Done()
			_ = write(runCall(ctx, h, call))
		}()
	}
	if err := sc.Err(); err != nil {
		_ = write(errorLine(0, NewError(CodeBadRequest, err.Error())))
	}
	cancel()
	calls.Wait()
	_ = c.Close()
	s.mu.Lock()
	delete(s.conns, c)
	s.mu.Unlock()
}

// runCall runs one handler and turns its outcome, or its panic, into the final line.
func runCall(ctx context.Context, h Handler, call *Call) (final Line) {
	defer func() {
		call.finish()
		if p := recover(); p != nil {
			final = errorLine(call.ID, NewError(CodeInternal, fmt.Sprint("panic: ", p)))
		}
	}()
	result, err := h(ctx, call)
	if err != nil {
		var ce *Error
		if errors.As(err, &ce) && ValidCode(ce.Code) {
			return errorLine(call.ID, NewError(ce.Code, ce.Message))
		}
		return errorLine(call.ID, NewError(CodeInternal, err.Error()))
	}
	if result != nil && !json.Valid(result) {
		return errorLine(call.ID, NewError(CodeInternal, "handler returned invalid JSON"))
	}
	return resultLine(call.ID, result)
}
