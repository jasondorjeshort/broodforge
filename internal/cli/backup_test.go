// SPDX-License-Identifier: GPL-3.0-or-later

package cli_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jasondorjeshort/broodforge/internal/audit"
	"github.com/jasondorjeshort/broodforge/internal/cli/clitest"
	"github.com/jasondorjeshort/broodforge/internal/control/controltest"
	"github.com/jasondorjeshort/broodforge/internal/store"
)

// The P5 Proof (08-implementation-plan.md) on the built binary, against a root that
// store.Create made and no bot: run with -v to see each command, its exit code and output.

// storeRoot is an instance root with config/instance.json (so the socket address resolves and
// read commands fall back to the file) and a fresh store; it returns the root and the
// address nothing listens on.
func storeRoot(t *testing.T) (root, addr string) {
	t.Helper()
	root, _, addr = controltest.Root(t)
	s, err := store.Create(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	return root, addr
}

func TestProofBackupSnapshotVerify(t *testing.T) {
	root, _ := storeRoot(t)
	r := clitest.Run(t, nil, "--root", root, "backup", "verify", "--expect", "ok")
	want(t, r, 1, "", "error: expect: wanted ok, got no-snapshot\n")

	r = clitest.Run(t, nil, "--root", root, "backup", "snapshot")
	prefix := "ok  " + filepath.ToSlash(root) + "/backup/store-"
	if r.Code != 0 || !strings.HasPrefix(r.Stdout, prefix) || !strings.HasSuffix(r.Stdout, ".sqlite  0\n") || r.Stderr != "" {
		t.Fatalf("snapshot: exit %d stdout %q stderr %q", r.Code, r.Stdout, r.Stderr)
	}
	snap := strings.Fields(r.Stdout)[1]

	r = clitest.Run(t, nil, "--root", root, "backup", "verify", "--expect", "ok")
	if r.Code != 0 || !strings.HasPrefix(r.Stdout, "ok\nmigration  4\naudit_event  0\n") || r.Stderr != "" {
		t.Errorf("verify: exit %d stdout %q stderr %q", r.Code, r.Stdout, r.Stderr)
	}
	r = clitest.Run(t, nil, "--root", root, "--json", "backup", "verify")
	if r.Code != 0 || !strings.HasPrefix(r.Stdout, `{"result":"ok","path":"`+snap+`","migration":4,"tables":[{"table":"audit_event","rows":0},`) {
		t.Errorf("verify --json: exit %d stdout %q", r.Code, r.Stdout)
	}

	b, err := os.ReadFile(snap)
	if err != nil {
		t.Fatal(err)
	}
	truncated := filepath.ToSlash(filepath.Join(t.TempDir(), "truncated.sqlite"))
	if err := os.WriteFile(truncated, b[:len(b)/2], 0o600); err != nil {
		t.Fatal(err)
	}
	r = clitest.Run(t, nil, "--root", root, "backup", "verify", "--path", truncated)
	if r.Code != 1 || r.Stdout != "" || !strings.HasPrefix(r.Stderr, "error: integrity-check: ") {
		t.Errorf("truncated: exit %d stdout %q stderr %q", r.Code, r.Stdout, r.Stderr)
	}
	want(t, clitest.Run(t, nil, "--root", root, "backup", "verify", "--path", truncated, "--expect", "integrity-check"), 0, "", "")
}

func TestProofAuditUsageReconcile(t *testing.T) {
	root, addr := storeRoot(t)
	want(t, clitest.Run(t, nil, "--root", root, "audit", "--limit", "1"), 0, "", "")
	want(t, clitest.Run(t, nil, "--root", root, "usage"), 0, "total  0  0  0  0  0  0\n", "")
	want(t, clitest.Run(t, nil, "--root", root, "--json", "usage"), 0,
		`{"model":"total","turns":0,"input":0,"output":0,"cacheRead":0,"cacheWrite":0,"interruptedTurns":0}`+"\n", "")
	want(t, clitest.Run(t, nil, "--root", root, "usage", "--by", "turn"), 0, "", "")
	want(t, clitest.Run(t, nil, "--root", root, "reconcile", "--report-only"), 0, "", "")
	want(t, clitest.Run(t, nil, "--root", root, "reconcile"), 1, "", "error: not-running: no bot at "+addr+"\n")
	want(t, clitest.Run(t, nil, "--root", root, "audit", "--follow"), 1, "", "error: not-running: no bot at "+addr+"\n")
	want(t, clitest.Run(t, nil, "--root", root, "usage", "--user", "1", "--channel", "2"), 2, "",
		"error: usage: usage: give at most one of --session, --user, --credential, --channel, --project\n")

	s, err := store.Create(root)
	if err != nil {
		t.Fatal(err)
	}
	w := audit.NewWriter(nil)
	err = s.Write(func(tx *store.Tx) error {
		if _, err := w.Append(tx, audit.Event{Kind: audit.KindInstanceStarted, Actor: "instance"}); err != nil {
			return err
		}
		_, err := w.Append(tx, audit.Event{Kind: audit.KindPush, Phase: audit.PhaseFailed, Actor: "instance", Session: "s0001",
			Details: []byte(`{"remote":"origin","error":"` + strings.Repeat("x", 100) + `"}`)})
		return err
	})
	s.Close()
	if err != nil {
		t.Fatal(err)
	}
	r := clitest.Run(t, nil, "--root", root, "audit", "--limit", "1")
	f := strings.Split(strings.TrimSuffix(r.Stdout, "\n"), "  ")
	if r.Code != 0 || len(f) != 7 || f[0] != "2" || f[2] != "push" || f[3] != "failed" || f[5] != "s0001" || len(f[6]) != 80 {
		t.Errorf("audit --limit 1: exit %d fields %q", r.Code, f)
	}
	r = clitest.Run(t, nil, "--root", root, "audit", "--kind", "instance-started")
	if f := strings.Split(strings.TrimSuffix(r.Stdout, "\n"), "  "); r.Code != 0 || len(f) != 7 || f[0] != "1" || f[3] != "-" || f[5] != "-" || f[6] != "{}" {
		t.Errorf("audit --kind: exit %d stdout %q", r.Code, r.Stdout)
	}
}

func TestProofNoStore(t *testing.T) {
	root, _, _ := controltest.Root(t)
	path := store.Path(root)
	want(t, clitest.Run(t, nil, "--root", root, "usage"), 1, "", "error: no-store: no store at "+path+"\n")
	want(t, clitest.Run(t, nil, "--root", root, "audit"), 1, "", "error: no-store: no store at "+path+"\n")
	want(t, clitest.Run(t, nil, "--root", root, "backup", "snapshot"), 1, "", "error: no-store: no store at "+path+"\n")
}
