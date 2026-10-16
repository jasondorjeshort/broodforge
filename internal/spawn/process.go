// SPDX-License-Identifier: GPL-3.0-or-later

package spawn

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

// Exit is how an engine process ended. Code is nil when it did not exit by itself (a signal);
// Interrupted is set when Interrupt was called before it ended.
type Exit struct {
	Code        *int
	Signal      string
	Interrupted bool
}

// pipeGrace is how long after the process exits its stdout and stderr may stay open, held by
// a leftover child, before they are closed so a reader is not stuck.
const pipeGrace = 5 * time.Second

// Process is one running engine. Read Stdout and Stderr to their end, then Wait.
type Process struct {
	Stdout io.Reader
	Stderr io.Reader

	proc   *os.Process
	outR   *os.File
	errR   *os.File
	host   *hostProc
	exited chan struct{}
	exit   Exit

	mu          sync.Mutex
	interrupted bool
	closeOnce   sync.Once
}

// Pid is the engine's process id.
func (p *Process) Pid() int { return p.proc.Pid }

// Wait blocks until the process has exited and returns how.
func (p *Process) Wait() Exit {
	<-p.exited
	return p.exit
}

// Done is closed when the process has exited.
func (p *Process) Done() <-chan struct{} { return p.exited }

// Interrupt ends the turn early (03-architecture.md, interrupt). Windows: the engine's job is
// closed, which ends its whole tree; when the job was never assigned, the tree is walked and
// killed. Linux: SIGINT to the engine, SIGKILL to its process group after the interrupt
// timeout. Calling it again, or after the exit, does nothing.
func (p *Process) Interrupt() error {
	p.mu.Lock()
	if p.interrupted {
		p.mu.Unlock()
		return nil
	}
	p.interrupted = true
	p.mu.Unlock()
	select {
	case <-p.exited:
		return nil
	default:
	}
	return p.host.interrupt(p)
}

// Close releases the read ends of stdout and stderr; Wait's caller calls it when done reading.
func (p *Process) Close() {
	p.closeOnce.Do(func() {
		_ = p.outR.Close()
		_ = p.errR.Close()
	})
}

// start runs path with args and env in cwd, stdin written then closed, stdout and stderr on
// pipes the caller reads.
func start(path string, args []string, cwd string, env []string, stdin []byte, timeout time.Duration) (*Process, error) {
	if !filepath.IsAbs(path) {
		return nil, &Error{CodeSpawnFailed, "the engine path " + quote(path) + " is not absolute"}
	}
	inR, inW, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	outR, outW, err := os.Pipe()
	if err != nil {
		closeAll(inR, inW)
		return nil, err
	}
	errR, errW, err := os.Pipe()
	if err != nil {
		closeAll(inR, inW, outR, outW)
		return nil, err
	}
	cmd := exec.Command(path, args...)
	cmd.Dir = cwd
	cmd.Env = env
	cmd.Stdin, cmd.Stdout, cmd.Stderr = inR, outW, errW
	cmd.SysProcAttr = sysProcAttr()
	err = cmd.Start()
	closeAll(inR, outW, errW) // the child holds its own copies
	if err != nil {
		closeAll(inW, outR, errR)
		return nil, &Error{CodeSpawnFailed, err.Error()}
	}
	p := &Process{Stdout: outR, Stderr: errR, proc: cmd.Process, outR: outR, errR: errR, exited: make(chan struct{})}
	p.host = attach(cmd.Process, timeout)
	go func() {
		_, _ = inW.Write(stdin) // an engine that exits without reading closes the pipe first
		_ = inW.Close()
	}()
	go p.wait()
	return p, nil
}

// wait reaps the process; cmd.Wait is not used, because it waits for the pipes too, and a
// leftover child may hold them.
func (p *Process) wait() {
	state, err := p.proc.Wait()
	p.host.exited()
	p.mu.Lock()
	p.exit.Interrupted = p.interrupted
	p.mu.Unlock()
	if err == nil {
		exitOf(state, &p.exit)
	}
	close(p.exited)
	time.AfterFunc(pipeGrace, p.Close)
}

func closeAll(fs ...*os.File) {
	for _, f := range fs {
		_ = f.Close()
	}
}
