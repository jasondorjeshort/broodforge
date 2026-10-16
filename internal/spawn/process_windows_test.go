// SPDX-License-Identifier: GPL-3.0-or-later

package spawn

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

// The P7 Proof's job-object half (08-implementation-plan.md): a grandchild dies when the
// engine's job is closed. The engine is cmd.exe running `ping -n 31 127.0.0.1` (the findings:
// the engine refuses a standalone sleep, so the spike killed a ping); the grandchild is the
// PING.EXE cmd.exe starts.

func cmdBackend(t *testing.T) *SameAccount {
	t.Helper()
	cmd := filepath.Join(os.Getenv("SystemRoot"), "System32", "cmd.exe")
	s, err := NewSameAccount(Config{Backend: NameSameAccount, EnginePath: cmd, Binary: "C:/bf/bin/broodforge.exe", Install: "C:/bf"})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// startPing starts the cmd.exe engine and waits for its PING.EXE child.
func startPing(t *testing.T) (*Process, procInfo) {
	t.Helper()
	tmp := t.TempDir()
	p, err := cmdBackend(t).Start(Spec{Args: []string{"/c", "ping -n 31 127.0.0.1"}, Cwd: tmp,
		Delivery: Delivery{Kind: DeliverConfigDir, ConfigDir: filepath.Join(tmp, "cfg")}})
	if err != nil {
		t.Fatal(err)
	}
	go func() { _, _ = io.Copy(io.Discard, p.Stdout) }()
	go func() { _, _ = io.Copy(io.Discard, p.Stderr) }()
	t.Cleanup(func() {
		_ = p.Interrupt()
		p.Close()
	})
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		procs, err := tree(p.Pid())
		if err != nil {
			t.Fatal(err)
		}
		for _, q := range procs {
			if strings.EqualFold(q.name, "PING.EXE") {
				return p, q
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("no PING.EXE under the engine")
	return nil, procInfo{}
}

// interruptAndCheck interrupts p and proves both the engine and its grandchild are gone.
func interruptAndCheck(t *testing.T, p *Process, ping procInfo) {
	t.Helper()
	if !alive(ping) {
		t.Fatal("ping is not running before the interrupt")
	}
	start := time.Now()
	if err := p.Interrupt(); err != nil {
		t.Fatalf("Interrupt: %v", err)
	}
	select {
	case <-p.Done():
	case <-time.After(10 * time.Second):
		t.Fatal("the engine did not exit after the interrupt")
	}
	exit := p.Wait()
	if !exit.Interrupted {
		t.Errorf("exit %+v: not interrupted", exit)
	}
	for alive(ping) {
		if time.Since(start) > 10*time.Second {
			t.Fatalf("PING.EXE %d still running %s after the interrupt", ping.pid, time.Since(start))
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Logf("engine %d and PING.EXE %d gone %s after the interrupt", p.Pid(), ping.pid, time.Since(start).Round(time.Millisecond))
}

func TestJobKillsGrandchild(t *testing.T) {
	p, ping := startPing(t)
	p.host.mu.Lock()
	job, assignErr := p.host.job, p.host.assignErr
	p.host.mu.Unlock()
	if job == 0 || assignErr != nil {
		t.Fatalf("the engine is in no job: %v", assignErr)
	}
	interruptAndCheck(t, p, ping)
}

func TestFallbackKillsGrandchild(t *testing.T) {
	assignToJob = func(windows.Handle, windows.Handle) error { return errors.New("assignment refused by the test") }
	t.Cleanup(func() { assignToJob = windows.AssignProcessToJobObject })
	p, ping := startPing(t)
	if p.host.job != 0 || p.host.assignErr == nil {
		t.Fatal("the engine is in a job; the fallback is not exercised")
	}
	interruptAndCheck(t, p, ping)
}

// TestJobEndsLeftovers: a process the engine leaves running dies when the engine exits.
func TestJobEndsLeftovers(t *testing.T) {
	tmp := t.TempDir()
	p, err := cmdBackend(t).Start(Spec{Args: []string{"/c", "start /b ping -n 31 127.0.0.1 >nul & ping -n 3 127.0.0.1 >nul"}, Cwd: tmp,
		Delivery: Delivery{Kind: DeliverConfigDir, ConfigDir: filepath.Join(tmp, "cfg")}})
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	// Both pings are children of cmd.exe: the leftover (31 echoes) and cmd.exe's own wait (3).
	var pings []procInfo
	for deadline := time.Now().Add(10 * time.Second); len(pings) < 2 && time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		all, err := snapshot()
		if err != nil {
			t.Fatal(err)
		}
		pings = pings[:0]
		for _, q := range all {
			if q.ppid == p.Pid() && strings.EqualFold(q.name, "PING.EXE") {
				pings = append(pings, q)
			}
		}
	}
	if len(pings) < 2 {
		t.Fatalf("PING.EXE children: %d, want 2", len(pings))
	}
	go func() { _, _ = io.Copy(io.Discard, p.Stdout) }()
	go func() { _, _ = io.Copy(io.Discard, p.Stderr) }()
	exit := p.Wait()
	if exit.Interrupted || exit.Code == nil {
		t.Errorf("exit %+v", exit)
	}
	for _, ping := range pings {
		for start := time.Now(); alive(ping); time.Sleep(20 * time.Millisecond) {
			if time.Since(start) > 10*time.Second {
				t.Fatalf("the leftover PING.EXE %d outlived the engine", ping.pid)
			}
		}
	}
}

func TestJoinKillOnCloseJob(t *testing.T) {
	if err := JoinKillOnCloseJob(); err != nil {
		t.Fatal(err)
	}
	if err := JoinKillOnCloseJob(); err != nil {
		t.Fatal(err)
	}
	if selfJob == 0 {
		t.Error("no job held")
	}
}
