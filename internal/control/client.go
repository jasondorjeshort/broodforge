// SPDX-License-Identifier: GPL-3.0-or-later

package control

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
)

// errNoServer is what dial returns when nothing listens at the address.
var errNoServer = errors.New("control: no server")

// NotRunning is the error for an address nothing answers on:
// `not-running: no bot at <addr>` (05-cli-conventions.md, Surface).
func NotRunning(addr string) *Error {
	return &Error{Code: CodeNotRunning, Message: "no bot at " + addr}
}

// Dial opens a raw connection to addr for a caller that writes and reads the lines itself
// (the wire tests). Nothing listening is NotRunning(addr); any other failure is dialError's.
func Dial(ctx context.Context, addr string) (io.ReadWriteCloser, error) {
	c, err := dial(ctx, addr)
	if err != nil {
		return nil, dialError(ctx, addr, err)
	}
	return c, nil
}

// dialError names why a connection failed: not-running only when nothing listens at addr;
// connect-failed when something may be there but the connection was refused (access denied,
// every pipe instance still busy after the retry, a bad address); the context's own error when
// it ended first.
func dialError(ctx context.Context, addr string, err error) error {
	switch {
	case errors.Is(err, errNoServer):
		return NotRunning(addr)
	case ctx.Err() != nil:
		return ctx.Err()
	}
	return NewError(CodeConnectFailed, "cannot connect to "+addr+": "+err.Error())
}

// Do sends one request on a fresh connection to addr and reads lines until its final
// response, handing each event to onEvent (which may be nil) in order. It returns the result,
// or an *Error: the server's refusal, not-running when nothing listens, connect-failed when the
// connection is refused, connection-lost when it ends early. An error from onEvent ends the call and is returned as is.
func Do(ctx context.Context, addr, cmd string, args json.RawMessage, onEvent func(json.RawMessage) error) (json.RawMessage, error) {
	c, err := dial(ctx, addr)
	if err != nil {
		return nil, dialError(ctx, addr, err)
	}
	stop := context.AfterFunc(ctx, func() { _ = c.Close() })
	defer func() {
		stop()
		_ = c.Close()
	}()

	const id = 1
	b, err := json.Marshal(Request{ID: id, Cmd: cmd, Args: args})
	if err != nil {
		return nil, err
	}
	if _, err := c.Write(append(b, '\n')); err != nil {
		return nil, lost(ctx, err)
	}
	sc := bufio.NewScanner(c)
	sc.Buffer(make([]byte, 0, 64<<10), maxLine)
	for sc.Scan() {
		var l Line
		if err := json.Unmarshal(sc.Bytes(), &l); err != nil {
			return nil, NewError(CodeConnectionLost, "bad line from the bot: "+err.Error())
		}
		if l.ID != id {
			continue
		}
		switch l.Kind() {
		case LineEvent:
			if onEvent != nil {
				if err := onEvent(l.Event); err != nil {
					return nil, err
				}
			}
		case LineResult:
			return l.Result, nil
		case LineError:
			return nil, l.Error
		case LineInvalid:
			return nil, NewError(CodeConnectionLost, "bad line from the bot: "+OneLine(sc.Text()))
		}
	}
	err = sc.Err()
	if err == nil {
		err = io.ErrUnexpectedEOF
	}
	return nil, lost(ctx, err)
}

func lost(ctx context.Context, err error) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return NewError(CodeConnectionLost, err.Error())
}
