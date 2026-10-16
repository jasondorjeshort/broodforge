// SPDX-License-Identifier: GPL-3.0-or-later

package control

import (
	"context"
	"errors"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sys/windows"
)

// The named pipe is built on golang.org/x/sys/windows alone. Every handle is opened with
// FILE_FLAG_OVERLAPPED and every operation (connect, read, write) waits on its own event and on
// a close event, so Close cancels whatever is pending.
//
// Three rules keep still the memory the kernel touches after a Go call has returned:
//   - every OVERLAPPED lives in an op allocated on the heap and kept by its owner, never in a
//     local: a stack-allocated one moves when the goroutine's stack grows between the call and
//     its completion, and the kernel then completes into the old copy (seen as
//     ERROR_IO_INCOMPLETE after the event fired);
//   - ReadFile and WriteFile use the connection's own heap buffers and copy, never the caller's
//     slice, which may live on a stack;
//   - the handle array of WaitForMultipleObjects is the op's hs field, never a slice literal: a
//     literal does not escape, and x/sys v0.49.0 turns &handles[0] into a uintptr and then calls
//     another Go function before the syscall, so a stack move in between hands the kernel freed
//     stack (seen as ERROR_INVALID_HANDLE on a fresh connection; TestStressUnderGC catches it).
//
// os.NewFile is not used: on Go 1.27.2 its poll.FD updates one offset field from both Read and
// Write of a pipe under separate locks, a data race `go test -race` reports when a connection
// reads and writes at once, as this one does.

func hostAddress(_, instanceName string) string {
	return `\\.\pipe\broodforge-` + instanceName
}

const (
	pipeBuffer = 64 << 10
	// PIPE_REJECT_REMOTE_CLIENTS: the pipe answers this host only (no network access).
	pipeMode = windows.PIPE_TYPE_BYTE | windows.PIPE_READMODE_BYTE | windows.PIPE_WAIT | windows.PIPE_REJECT_REMOTE_CLIENTS
	// How long a client retries a pipe whose every instance is busy.
	pipeBusyWait = 2 * time.Second
)

// op is one overlapped operation slot: an OVERLAPPED with its manual-reset event, on the heap.
type op struct {
	ov windows.Overlapped
	n  uint32 // bytes moved, from GetOverlappedResult
	// done receives ReadFile and WriteFile's own count, which x/sys reads under -race;
	// for an overlapped call it is not the result.
	done uint32
	hs   [2]windows.Handle // WaitForMultipleObjects' array: the event, then the close event
}

func newOp() (*op, error) {
	ev, err := windows.CreateEvent(nil, 1, 0, nil)
	if err != nil {
		return nil, err
	}
	o := &op{}
	o.ov.HEvent = ev
	return o, nil
}

func (o *op) close() { _ = windows.CloseHandle(o.ov.HEvent) }

// cancel cancels the pending operation on h and waits until the kernel is done with it.
func (o *op) cancel(h windows.Handle) {
	_ = windows.CancelIoEx(h, &o.ov)
	_ = windows.GetOverlappedResult(h, &o.ov, &o.n, true)
}

// wait finishes an operation on h that its call reported as started (nil or IO_PENDING; a
// synchronous completion also signals the event). If closeEv fires first the operation is
// cancelled and waited for, and cancelled is true. If the wait itself fails the operation is
// also cancelled and waited for before the error returns, so the op and its buffer are free
// for the next call.
func (o *op) wait(h, closeEv windows.Handle) (n int, cancelled bool, err error) {
	o.hs = [2]windows.Handle{o.ov.HEvent, closeEv}
	which, err := windows.WaitForMultipleObjects(o.hs[:], false, windows.INFINITE)
	if err != nil {
		o.cancel(h)
		return 0, false, err
	}
	if which != windows.WAIT_OBJECT_0 {
		o.cancel(h)
		return int(o.n), true, nil
	}
	err = windows.GetOverlappedResult(h, &o.ov, &o.n, false)
	return int(o.n), false, err
}

type pipeListener struct {
	addr   string
	name16 *uint16
	cancel windows.Handle // manual-reset event, set by Close

	acceptMu sync.Mutex // one Accept at a time; Close takes it before freeing cancel and conn
	conn     *op        // the connect operation, used under acceptMu

	mu     sync.Mutex
	closed bool
	next   windows.Handle // the instance the next client connects to, or InvalidHandle
}

func listen(addr string) (listener, error) {
	name16, err := windows.UTF16PtrFromString(addr)
	if err != nil {
		return nil, NewError(CodeStartControl, err.Error())
	}
	l := &pipeListener{addr: addr, name16: name16, next: windows.InvalidHandle}
	// FILE_FLAG_FIRST_PIPE_INSTANCE: the first instance fails if the name already exists, so a
	// second bot on the same instanceName cannot join the first one's pipe.
	h, err := l.create(true)
	if err != nil {
		if errors.Is(err, windows.ERROR_ACCESS_DENIED) {
			return nil, NewError(CodeStartControl, "another bot owns "+addr)
		}
		return nil, NewError(CodeStartControl, addr+": "+err.Error())
	}
	cancel, err := windows.CreateEvent(nil, 1, 0, nil)
	if err != nil {
		_ = windows.CloseHandle(h)
		return nil, NewError(CodeStartControl, err.Error())
	}
	conn, err := newOp()
	if err != nil {
		_ = windows.CloseHandle(h)
		_ = windows.CloseHandle(cancel)
		return nil, NewError(CodeStartControl, err.Error())
	}
	l.next, l.cancel, l.conn = h, cancel, conn
	return l, nil
}

func (l *pipeListener) create(first bool) (windows.Handle, error) {
	flags := uint32(windows.PIPE_ACCESS_DUPLEX | windows.FILE_FLAG_OVERLAPPED)
	if first {
		flags |= windows.FILE_FLAG_FIRST_PIPE_INSTANCE
	}
	return windows.CreateNamedPipe(l.name16, flags, pipeMode, windows.PIPE_UNLIMITED_INSTANCES,
		pipeBuffer, pipeBuffer, 0, nil)
}

func (l *pipeListener) Accept() (io.ReadWriteCloser, error) {
	l.acceptMu.Lock()
	defer l.acceptMu.Unlock()
	for {
		l.mu.Lock()
		if l.closed {
			l.mu.Unlock()
			return nil, errListenerClosed
		}
		h := l.next
		l.next = windows.InvalidHandle
		l.mu.Unlock()
		if h == windows.InvalidHandle {
			var err error
			if h, err = l.create(false); err != nil {
				return nil, err
			}
		}
		cancelled, err := l.connect(h)
		if cancelled {
			_ = windows.CloseHandle(h)
			return nil, errListenerClosed
		}
		if err != nil {
			// The client went away before the connection completed: drop this instance.
			_ = windows.CloseHandle(h)
			continue
		}
		// Make the next instance before handing this one out, so a client never finds the
		// name missing between two accepts.
		l.mu.Lock()
		if !l.closed {
			if nh, err := l.create(false); err == nil {
				l.next = nh
			}
		}
		l.mu.Unlock()
		c, err := newPipeConn(h)
		if err != nil {
			_ = windows.CloseHandle(h)
			continue
		}
		return c, nil
	}
}

// connect waits for a client on h, or for Close. The caller holds acceptMu.
func (l *pipeListener) connect(h windows.Handle) (cancelled bool, err error) {
	switch err := windows.ConnectNamedPipe(h, &l.conn.ov); {
	case err == nil, errors.Is(err, windows.ERROR_PIPE_CONNECTED):
		return false, nil
	case errors.Is(err, windows.ERROR_IO_PENDING):
	default:
		return false, err
	}
	_, cancelled, err = l.conn.wait(h, l.cancel)
	return cancelled, err
}

func (l *pipeListener) Close() error {
	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return nil
	}
	l.closed = true
	if l.next != windows.InvalidHandle {
		_ = windows.CloseHandle(l.next)
		l.next = windows.InvalidHandle
	}
	err := windows.SetEvent(l.cancel)
	l.mu.Unlock()
	l.acceptMu.Lock() // a pending Accept has returned
	_ = windows.CloseHandle(l.cancel)
	l.conn.close()
	l.acceptMu.Unlock()
	return err
}

func dial(ctx context.Context, addr string) (io.ReadWriteCloser, error) {
	name16, err := windows.UTF16PtrFromString(addr)
	if err != nil {
		return nil, err
	}
	deadline := time.Now().Add(pipeBusyWait)
	for {
		// SECURITY_IDENTIFICATION: a server squatting the name can identify the caller but not
		// impersonate it.
		h, err := windows.CreateFile(name16, windows.GENERIC_READ|windows.GENERIC_WRITE, 0, nil,
			windows.OPEN_EXISTING,
			windows.FILE_FLAG_OVERLAPPED|windows.SECURITY_SQOS_PRESENT|windows.SECURITY_IDENTIFICATION, 0)
		switch {
		case err == nil:
			c, err := newPipeConn(h)
			if err != nil {
				_ = windows.CloseHandle(h)
				return nil, err
			}
			return c, nil
		case errors.Is(err, windows.ERROR_FILE_NOT_FOUND):
			return nil, errNoServer
		case errors.Is(err, windows.ERROR_PIPE_BUSY) && time.Now().Before(deadline):
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(10 * time.Millisecond):
			}
		default:
			return nil, err
		}
	}
}

// pipeConn is one connected pipe handle. One Read and one Write may run at once; Close cancels
// both, waits for them to return, then closes the handles. A closed far end reads as EOF.
type pipeConn struct {
	h       windows.Handle
	closeEv windows.Handle // manual-reset event, set by Close

	closed atomic.Bool
	once   sync.Once

	rmu  sync.Mutex // guards rop and rbuf
	rop  *op
	rbuf []byte

	wmu  sync.Mutex // guards wop and wbuf
	wop  *op
	wbuf []byte
}

func newPipeConn(h windows.Handle) (*pipeConn, error) {
	closeEv, err := windows.CreateEvent(nil, 1, 0, nil)
	if err != nil {
		return nil, err
	}
	rop, err := newOp()
	if err != nil {
		_ = windows.CloseHandle(closeEv)
		return nil, err
	}
	wop, err := newOp()
	if err != nil {
		_ = windows.CloseHandle(closeEv)
		rop.close()
		return nil, err
	}
	return &pipeConn{h: h, closeEv: closeEv, rop: rop, wop: wop,
		rbuf: make([]byte, pipeBuffer), wbuf: make([]byte, pipeBuffer)}, nil
}

func (c *pipeConn) Read(p []byte) (int, error) {
	c.rmu.Lock()
	defer c.rmu.Unlock()
	if len(p) == 0 {
		return 0, nil
	}
	if c.closed.Load() {
		return 0, os.ErrClosed
	}
	buf := c.rbuf[:min(len(p), len(c.rbuf))]
	n, cancelled, err := c.do(c.rop, windows.ReadFile(c.h, buf, &c.rop.done, &c.rop.ov))
	copy(p, buf[:n])
	switch {
	case cancelled:
		return n, os.ErrClosed
	case errors.Is(err, windows.ERROR_BROKEN_PIPE), errors.Is(err, windows.ERROR_PIPE_NOT_CONNECTED),
		errors.Is(err, windows.ERROR_NO_DATA), err == nil && n == 0:
		return n, io.EOF
	}
	return n, err
}

func (c *pipeConn) Write(p []byte) (int, error) {
	c.wmu.Lock()
	defer c.wmu.Unlock()
	total := 0
	for total < len(p) {
		if c.closed.Load() {
			return total, os.ErrClosed
		}
		k := copy(c.wbuf, p[total:])
		n, cancelled, err := c.do(c.wop, windows.WriteFile(c.h, c.wbuf[:k], &c.wop.done, &c.wop.ov))
		total += n
		switch {
		case cancelled:
			return total, os.ErrClosed
		case err != nil:
			return total, err
		case n == 0:
			return total, io.ErrShortWrite
		}
	}
	return total, nil
}

// do finishes the operation whose call returned started.
func (c *pipeConn) do(o *op, started error) (int, bool, error) {
	if started != nil && !errors.Is(started, windows.ERROR_IO_PENDING) {
		return 0, false, started
	}
	return o.wait(c.h, c.closeEv)
}

func (c *pipeConn) Close() error {
	var err error
	c.once.Do(func() {
		c.closed.Store(true)
		err = windows.SetEvent(c.closeEv)
		c.rmu.Lock() // pending operations have been cancelled and have returned
		c.wmu.Lock()
		if cerr := windows.CloseHandle(c.h); err == nil {
			err = cerr
		}
		_ = windows.CloseHandle(c.closeEv)
		c.rop.close()
		c.wop.close()
		c.wmu.Unlock()
		c.rmu.Unlock()
	})
	return err
}
