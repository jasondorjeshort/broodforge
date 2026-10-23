// SPDX-License-Identifier: GPL-3.0-or-later

package cli_test

import (
	"os"
	"strings"
	"testing"

	"github.com/jasondorjeshort/broodforge/internal/cli/clitest"
	"github.com/jasondorjeshort/broodforge/internal/config"
	"github.com/jasondorjeshort/broodforge/internal/control/controltest"
	"github.com/jasondorjeshort/broodforge/internal/spawn/spawntest"
)

// The P2 Proof (08-implementation-plan.md), on the built binary against the in-process
// harness. Run with -v to see each command, its exit code and its output.

// TestMain also makes this test binary the stand-in engine when the spawn runs it with
// spawntest.Env set (engine_real_test.go).
// It also empties config.MachineLiterals, so no test of this package reads the machine's user
// or host name (the harness's binary is built with the same switch, clitest).
func TestMain(m *testing.M) {
	config.MachineLiterals = func() []config.MachineLiteral { return nil }
	spawntest.Main()
	os.Exit(clitest.Main(m))
}

func want(t *testing.T, r clitest.Result, code int, stdout, stderr string) {
	t.Helper()
	if r.Code != code || r.Stdout != stdout || r.Stderr != stderr {
		t.Errorf("exit %d, want %d\nstdout %q, want %q\nstderr %q, want %q", r.Code, code, r.Stdout, stdout, r.Stderr, stderr)
	}
}

func TestProofEcho(t *testing.T) {
	h := controltest.Start(t)
	want(t, clitest.Run(t, nil, "--root", h.Root, "control", "echo", "--expect", "ok"), 0, "ok\n", "")
	want(t, clitest.Run(t, nil, "--root", h.Root, "--json", "control", "echo", "--expect", "ok"), 0, `{"text":"ok"}`+"\n", "")
	want(t, clitest.Run(t, []string{"BROODFORGE_ROOT=" + h.Root}, "control", "echo"), 0, "ok\n", "")
}

func TestProofStreaming(t *testing.T) {
	h := controltest.Start(t)
	want(t, clitest.Run(t, nil, "--root", h.Root, "control", "echo", "--events", "3", "--expect", "ok"), 0,
		"ok  1\nok  2\nok  3\nok\n", "")
	want(t, clitest.Run(t, nil, "--root", h.Root, "--json", "control", "echo", "--events", "2"), 0,
		`{"seq":1,"text":"ok"}`+"\n"+`{"seq":2,"text":"ok"}`+"\n"+`{"text":"ok"}`+"\n", "")
}

func TestProofNoRootExits2(t *testing.T) {
	r := clitest.Run(t, nil, "control", "echo")
	want(t, r, 2, "", "error: usage: no instance root: pass --root <path> or set BROODFORGE_ROOT\n")
	r = clitest.Run(t, nil)
	if r.Code != 2 || !strings.HasPrefix(r.Stderr, "usage: broodforge ") {
		t.Errorf("no args: exit %d stderr %q", r.Code, r.Stderr)
	}
}

func TestProofExpectMismatch(t *testing.T) {
	h := controltest.Start(t)
	want(t, clitest.Run(t, nil, "--root", h.Root, "control", "echo", "--expect", "nope"), 1, "ok\n",
		"error: expect: wanted nope, got ok\n")
	want(t, clitest.Run(t, nil, "--root", h.Root, "control", "echo", "--fail", "busy", "--expect", "busy"), 0, "", "")
	want(t, clitest.Run(t, nil, "--root", h.Root, "control", "echo", "--fail", "busy"), 1, "",
		"error: busy: echo was asked to fail\n")
}

func TestProofNoServer(t *testing.T) {
	root, _, addr := controltest.Root(t)
	want(t, clitest.Run(t, nil, "--root", root, "control", "echo"), 1, "", "error: not-running: no bot at "+addr+"\n")
}
