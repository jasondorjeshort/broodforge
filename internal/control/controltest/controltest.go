// SPDX-License-Identifier: GPL-3.0-or-later

// Package controltest is the in-process harness: a control server on a temporary root with the
// services built so far, for tests that drive the binary (or control.Do) against it before
// `serve` exists. It retires into `serve` (08-implementation-plan.md, P10).
package controltest

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/jasondorjeshort/broodforge/internal/control"
)

// Harness is one running server and the root it serves.
type Harness struct {
	Root         string
	InstanceName string
	Addr         string
	Server       *control.Server
}

// Root makes a temporary instance root holding config/instance.json with a fresh instanceName
// ("test-<12 hex>", unique per run because pipe names are machine-wide) and returns the root,
// the name and the control address. Nothing listens there.
func Root(t testing.TB) (root, instanceName, addr string) {
	t.Helper()
	root = t.TempDir()
	var b [6]byte
	if _, err := rand.Read(b[:]); err != nil {
		t.Fatal(err)
	}
	instanceName = "test-" + hex.EncodeToString(b[:])
	if err := os.MkdirAll(filepath.Join(root, "config"), 0o700); err != nil {
		t.Fatal(err)
	}
	cfg := `{ "schemaVersion": 1, "instanceName": "` + instanceName + `" }` + "\n"
	if err := os.WriteFile(filepath.Join(root, "config", "instance.json"), []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	addr, err := control.Address(root, instanceName)
	if err != nil {
		t.Fatal(err)
	}
	return root, instanceName, addr
}

// Start makes a root as Root does and serves it: the built-in commands plus whatever each
// register function adds. The server closes when the test ends, and a Serve error fails it.
func Start(t testing.TB, register ...func(*control.Registry)) *Harness {
	t.Helper()
	root, name, addr := Root(t)
	reg := control.NewRegistry()
	for _, r := range register {
		r(reg)
	}
	srv, err := control.Listen(addr, reg)
	if err != nil {
		t.Fatalf("control.Listen(%s): %v", addr, err)
	}
	served := make(chan error, 1)
	go func() { served <- srv.Serve() }()
	t.Cleanup(func() {
		if err := srv.Close(); err != nil {
			t.Errorf("close control server: %v", err)
		}
		if err := <-served; err != nil {
			t.Errorf("control server: %v", err)
		}
	})
	return &Harness{Root: root, InstanceName: name, Addr: addr, Server: srv}
}
