// SPDX-License-Identifier: GPL-3.0-or-later

//go:build !windows

package control

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// maxSocketPath is the portable bound on a unix socket path (sun_path is 104 bytes on the BSDs,
// 108 on Linux, both counting the terminating NUL).
const maxSocketPath = 103

func hostAddress(root, _ string) string {
	return filepath.Join(root, "data", "control.sock")
}

type unixListener struct{ ln *net.UnixListener }

func (l unixListener) Accept() (io.ReadWriteCloser, error) {
	c, err := l.ln.Accept()
	if errors.Is(err, net.ErrClosed) {
		return nil, errListenerClosed
	}
	return c, err
}

func (l unixListener) Close() error { return l.ln.Close() }

func listen(addr string) (listener, error) {
	if len(addr) > maxSocketPath {
		return nil, NewError(CodeStartControl, "socket path longer than 103 bytes: "+addr)
	}
	if err := os.MkdirAll(filepath.Dir(addr), 0o700); err != nil {
		return nil, NewError(CodeStartControl, err.Error())
	}
	if _, err := os.Lstat(addr); err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		c, derr := dial(ctx, addr)
		cancel()
		if derr == nil {
			_ = c.Close()
			return nil, NewError(CodeStartControl, "another bot owns "+addr)
		}
		// A socket file nothing answers on is left by a crashed bot: replace it.
		if err := os.Remove(addr); err != nil {
			return nil, NewError(CodeStartControl, err.Error())
		}
	}
	ln, err := net.ListenUnix("unix", &net.UnixAddr{Name: addr, Net: "unix"})
	if err != nil {
		return nil, NewError(CodeStartControl, err.Error())
	}
	if err := os.Chmod(addr, 0o600); err != nil {
		_ = ln.Close()
		return nil, NewError(CodeStartControl, err.Error())
	}
	return unixListener{ln: ln}, nil
}

func dial(ctx context.Context, addr string) (io.ReadWriteCloser, error) {
	var d net.Dialer
	c, err := d.DialContext(ctx, "unix", addr)
	if err != nil {
		if errors.Is(err, syscall.ENOENT) || errors.Is(err, syscall.ECONNREFUSED) {
			return nil, errNoServer
		}
		return nil, err
	}
	return c, nil
}
