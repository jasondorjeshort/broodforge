// SPDX-License-Identifier: GPL-3.0-or-later

package cli

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/jasondorjeshort/broodforge/internal/control"
)

type nopConn struct{ closed *bool }

func (c nopConn) Read([]byte) (int, error)    { return 0, io.EOF }
func (c nopConn) Write(b []byte) (int, error) { return len(b), nil }
func (c nopConn) Close() error                { *c.closed = true; return nil }

// TestProbeControl: serve goes on only when nothing listens; a bot answering, or a failure that
// may hide one (connect-failed), refuses start-control before anything is written or reconciled.
func TestProbeControl(t *testing.T) {
	closed := false
	for _, c := range []struct {
		name string
		dial dialFunc
		want string // "" goes on; else the start-control message's start
	}{
		{"nothing listens", func(context.Context, string) (io.ReadWriteCloser, error) {
			return nil, control.NotRunning("addr")
		}, ""},
		{"a bot answers", func(context.Context, string) (io.ReadWriteCloser, error) {
			return nopConn{&closed}, nil
		}, "another bot owns addr"},
		{"connect-failed", func(context.Context, string) (io.ReadWriteCloser, error) {
			return nil, control.NewError(control.CodeConnectFailed, "access denied")
		}, "cannot tell whether a bot owns addr: connect-failed: access denied"},
		{"any other error", func(context.Context, string) (io.ReadWriteCloser, error) {
			return nil, errors.New("boom")
		}, "cannot tell whether a bot owns addr: boom"},
	} {
		err := probeControl(context.Background(), "addr", c.dial)
		if c.want == "" {
			if err != nil {
				t.Errorf("%s: %v", c.name, err)
			}
			continue
		}
		var ce *control.Error
		if !errors.As(err, &ce) || ce.Code != control.CodeStartControl || !strings.HasPrefix(ce.Message, c.want) {
			t.Errorf("%s: got %v, want start-control: %s", c.name, err, c.want)
		}
	}
	if !closed {
		t.Error("the probe left the bot's connection open")
	}
}
