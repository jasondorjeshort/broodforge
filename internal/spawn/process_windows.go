// SPDX-License-Identifier: GPL-3.0-or-later

package spawn

import (
	"errors"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Windows process control (ADR 0023): every engine is assigned to its own kill-on-close job
// right after Start (the window before the assignment is accepted); interrupt terminates and
// closes that job, which ends the engine and every process it started; a process left by the
// engine after its own exit dies when the job is closed at the exit. When the assignment
// failed, interrupt walks the tree by parent process id (the spike's Get-CimInstance walk,
// done here through a Toolhelp32 snapshot) and runs taskkill /T /F, then ends any walked
// process still alive.

// sysProcAttr starts the engine in its own process group with its own console and no window,
// so a Ctrl-C in the bot's console reaches the bot alone (03, Signals and leftovers).
func sysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{CreationFlags: windows.CREATE_NEW_PROCESS_GROUP | windows.CREATE_NO_WINDOW}
}

// foldName compares environment names as Windows does, case-insensitively.
func foldName(name string) string { return strings.ToUpper(name) }

// assignToJob is AssignProcessToJobObject; a test replaces it to exercise the fallback.
var assignToJob = windows.AssignProcessToJobObject

type hostProc struct {
	mu        sync.Mutex
	pid       int
	job       windows.Handle // 0 once closed, or when the assignment failed
	assignErr error
	reaped    bool // Wait has returned: the pid may be reused, so nothing is killed by it
}

// newKillOnCloseJob creates an unnamed job whose processes die when its last handle closes.
func newKillOnCloseJob() (windows.Handle, error) {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return 0, err
	}
	var info windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION
	info.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	if _, err := windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info))); err != nil {
		_ = windows.CloseHandle(job)
		return 0, err
	}
	return job, nil
}

// attach assigns the new process to its own job. The pid cannot be reused here: os.Process
// holds the process handle until Wait returns.
func attach(p *os.Process, _ time.Duration) *hostProc {
	h := &hostProc{pid: p.Pid}
	job, err := newKillOnCloseJob()
	if err != nil {
		h.assignErr = err
		return h
	}
	ph, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(p.Pid))
	if err == nil {
		err = assignToJob(job, ph)
		_ = windows.CloseHandle(ph)
	}
	if err != nil {
		_ = windows.CloseHandle(job)
		h.assignErr = err
		return h
	}
	h.job = job
	return h
}

// exited closes the job, which ends whatever the engine left running.
func (h *hostProc) exited() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.reaped = true
	if h.job != 0 {
		_ = windows.CloseHandle(h.job)
		h.job = 0
	}
}

func (h *hostProc) interrupt(_ *Process) error {
	h.mu.Lock()
	job, reaped := h.job, h.reaped
	h.job = 0
	h.mu.Unlock()
	if reaped {
		return nil
	}
	if job != 0 {
		err := windows.TerminateJobObject(job, 1)
		_ = windows.CloseHandle(job) // kill-on-close ends the tree even if Terminate failed
		if err == nil {
			return nil
		}
	}
	return killTree(h.pid)
}

// exitOf reads the exit code; Windows has no signals.
func exitOf(state *os.ProcessState, e *Exit) {
	code := state.ExitCode()
	e.Code = &code
}

// procInfo is one process of a snapshot.
type procInfo struct {
	pid, ppid int
	name      string
	created   uint64 // FILETIME; 0 when the process could not be opened
}

func snapshot() ([]procInfo, error) {
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, err
	}
	defer windows.CloseHandle(snap)
	var out []procInfo
	var e windows.ProcessEntry32
	e.Size = uint32(unsafe.Sizeof(e))
	for err = windows.Process32First(snap, &e); err == nil; err = windows.Process32Next(snap, &e) {
		out = append(out, procInfo{pid: int(e.ProcessID), ppid: int(e.ParentProcessID),
			name: windows.UTF16ToString(e.ExeFile[:]), created: created(int(e.ProcessID))})
	}
	if !errors.Is(err, windows.ERROR_NO_MORE_FILES) {
		return nil, err
	}
	return out, nil
}

// created is a process's creation time, 0 when it cannot be read.
func created(pid int) uint64 {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return 0
	}
	defer windows.CloseHandle(h)
	var c, x, k, u windows.Filetime
	if windows.GetProcessTimes(h, &c, &x, &k, &u) != nil {
		return 0
	}
	return uint64(c.HighDateTime)<<32 | uint64(c.LowDateTime)
}

// tree is root and its descendants by parent process id; a child that predates its parent
// is a reused pid, not a child (the spike's rule).
func tree(root int) ([]procInfo, error) {
	all, err := snapshot()
	if err != nil {
		return nil, err
	}
	var out []procInfo
	for _, p := range all {
		if p.pid == root {
			out = append(out, p)
		}
	}
	for i := 0; i < len(out); i++ {
		for _, p := range all {
			if p.ppid != out[i].pid || p.pid == out[i].pid {
				continue
			}
			if p.created != 0 && out[i].created != 0 && p.created < out[i].created {
				continue
			}
			seen := false
			for _, q := range out {
				seen = seen || q.pid == p.pid
			}
			if !seen {
				out = append(out, p)
			}
		}
	}
	return out, nil
}

// alive reports whether pid still runs as the process created at the given time.
func alive(p procInfo) bool {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION|windows.SYNCHRONIZE, false, uint32(p.pid))
	if err != nil {
		return false
	}
	defer windows.CloseHandle(h)
	if ev, err := windows.WaitForSingleObject(h, 0); err != nil || ev != uint32(windows.WAIT_TIMEOUT) {
		return false
	}
	return p.created == 0 || created(p.pid) == p.created
}

// killWait bounds the wait for a terminated process to be gone: TerminateProcess and taskkill
// /F return before the process has ended.
const killWait = 2 * time.Second

// endProcess waits until deadline for p, the process created at p.created, to end after
// taskkill, terminates it if it has not, and waits up to killWait more; it reports whether p
// is gone.
func endProcess(p procInfo, deadline time.Time) bool {
	h, err := windows.OpenProcess(windows.SYNCHRONIZE|windows.PROCESS_TERMINATE|windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(p.pid))
	if err != nil {
		return true // no such process any more
	}
	defer windows.CloseHandle(h)
	if p.created != 0 {
		var c, x, k, u windows.Filetime
		if windows.GetProcessTimes(h, &c, &x, &k, &u) == nil && uint64(c.HighDateTime)<<32|uint64(c.LowDateTime) != p.created {
			return true // the pid was reused: not the process walked
		}
	}
	if waitGone(h, time.Until(deadline)) {
		return true
	}
	_ = windows.TerminateProcess(h, 1)
	return waitGone(h, killWait)
}

// waitGone waits up to d for the process handle to be signalled.
func waitGone(h windows.Handle, d time.Duration) bool {
	ms := uint32(0)
	if d > 0 {
		ms = uint32(d / time.Millisecond)
	}
	ev, err := windows.WaitForSingleObject(h, ms)
	return err == nil && ev == windows.WAIT_OBJECT_0
}

// killTree is the fallback: the walk, taskkill /T /F on the root, then a direct terminate of
// any walked process still alive (one whose parent had already gone is not in taskkill's tree).
func killTree(root int) error {
	walked, walkErr := tree(root)
	out, tkErr := exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(root)).CombinedOutput()
	var left []string
	deadline := time.Now().Add(killWait)
	for _, p := range walked {
		if !endProcess(p, deadline) {
			left = append(left, strconv.Itoa(p.pid))
		}
	}
	if len(left) > 0 {
		msg := "still running after taskkill: " + strings.Join(left, ", ")
		if tkErr != nil {
			msg += "; taskkill: " + strings.TrimSpace(string(out))
		}
		return &Error{CodeSpawnFailed, msg}
	}
	if walkErr != nil && tkErr != nil {
		return &Error{CodeSpawnFailed, "tree walk: " + walkErr.Error() + "; taskkill: " + strings.TrimSpace(string(out))}
	}
	return nil
}

var (
	selfJobMu sync.Mutex
	selfJob   windows.Handle
)

// JoinKillOnCloseJob puts the calling process in a kill-on-close job held open for its life
// (ADR 0023: the bot at start), so every engine it starts, and the engine's own job nested
// inside, ends when the bot ends, a crash included. A second call does nothing.
func JoinKillOnCloseJob() error {
	selfJobMu.Lock()
	defer selfJobMu.Unlock()
	if selfJob != 0 {
		return nil
	}
	job, err := newKillOnCloseJob()
	if err != nil {
		return err
	}
	if err := windows.AssignProcessToJobObject(job, windows.CurrentProcess()); err != nil {
		_ = windows.CloseHandle(job)
		return err
	}
	selfJob = job
	return nil
}
