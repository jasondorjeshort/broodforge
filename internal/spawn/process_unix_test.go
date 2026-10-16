// SPDX-License-Identifier: GPL-3.0-or-later

//go:build !windows

package spawn

import (
	"io"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// Linux interrupt: SIGINT first; an engine that ignores it, and the child it started, are
// killed with the process group after the interrupt timeout.
func TestInterruptEscalatesToGroupKill(t *testing.T) {
	tmp := t.TempDir()
	s, err := NewSameAccount(Config{Backend: NameSameAccount, EnginePath: "/bin/sh", Binary: "/bf/bin/broodforge",
		InterruptTimeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	p, err := s.Start(Spec{Args: []string{"-c", `trap "" INT; sleep 30 & echo $!; wait`}, Cwd: tmp,
		Delivery: Delivery{Kind: DeliverConfigDir, ConfigDir: filepath.Join(tmp, "cfg")}})
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	var line [32]byte
	n, _ := p.Stdout.Read(line[:])
	var child int
	for _, c := range line[:n] {
		if c >= '0' && c <= '9' {
			child = child*10 + int(c-'0')
		}
	}
	go func() { _, _ = io.Copy(io.Discard, p.Stdout) }()
	go func() { _, _ = io.Copy(io.Discard, p.Stderr) }()
	if err := p.Interrupt(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-p.Done():
	case <-time.After(10 * time.Second):
		t.Fatal("no exit after the group kill")
	}
	if exit := p.Wait(); !exit.Interrupted || exit.Signal == "" {
		t.Errorf("exit %+v", exit)
	}
	for start := time.Now(); child > 0 && syscall.Kill(child, 0) == nil; time.Sleep(20 * time.Millisecond) {
		if time.Since(start) > 5*time.Second {
			t.Fatalf("child %d outlived the group kill", child)
		}
	}
}
