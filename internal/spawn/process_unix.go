// SPDX-License-Identifier: GPL-3.0-or-later

//go:build !windows

package spawn

import (
	"os"
	"strconv"
	"syscall"
	"time"
)

// Linux process control (ADR 0020, Interrupt): the engine starts in its own process group, so
// a terminal's Ctrl-C reaches the bot alone (03, Signals and leftovers); interrupt is SIGINT to
// the engine, which ends the turn with a result, then SIGKILL to the group after the
// interrupt timeout.

func sysProcAttr() *syscall.SysProcAttr { return &syscall.SysProcAttr{Setpgid: true} }

// foldName keeps environment names as they are: Linux names are case-sensitive.
func foldName(name string) string { return name }

type hostProc struct {
	pid     int
	timeout time.Duration
}

func attach(p *os.Process, timeout time.Duration) *hostProc {
	return &hostProc{pid: p.Pid, timeout: timeout}
}

func (h *hostProc) exited() {}

func (h *hostProc) interrupt(p *Process) error {
	// Through os.Process (a pidfd where the kernel has one), so a reaped and reused pid is
	// never signalled.
	if err := p.proc.Signal(syscall.SIGINT); err != nil {
		return &Error{CodeSpawnFailed, "SIGINT to " + strconv.Itoa(h.pid) + ": " + err.Error()}
	}
	go func() {
		t := time.NewTimer(h.timeout)
		defer t.Stop()
		select {
		case <-p.exited:
		case <-t.C:
			_ = syscall.Kill(-h.pid, syscall.SIGKILL)
		}
	}()
	return nil
}

func exitOf(state *os.ProcessState, e *Exit) {
	if ws, ok := state.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
		e.Signal = ws.Signal().String()
		return
	}
	code := state.ExitCode()
	e.Code = &code
}

// JoinKillOnCloseJob is Windows' (ADR 0023); on Linux systemd's KillMode=mixed ends the
// leftovers with the unit, so it does nothing.
func JoinKillOnCloseJob() error { return nil }
