// SPDX-License-Identifier: GPL-3.0-or-later

package cli_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jasondorjeshort/broodforge/internal/audit"
	"github.com/jasondorjeshort/broodforge/internal/cli"
	"github.com/jasondorjeshort/broodforge/internal/cli/clitest"
	"github.com/jasondorjeshort/broodforge/internal/control"
	"github.com/jasondorjeshort/broodforge/internal/credentials"
	"github.com/jasondorjeshort/broodforge/internal/spawn"
	"github.com/jasondorjeshort/broodforge/internal/store"
)

// The owner's half of the P7 Proof, before serve exists (P10): the harness on a root that
// stays. With BROODFORGE_OWNER_ROOT (a directory that does not exist yet) and
// BROODFORGE_OWNER_ENGINE (the engine's .exe) set,
//
//	go test ./internal/cli -run TestOwnerSubscription -count=1 -v
//
// makes the root (config naming the engine and the version its --version prints, the key
// under the root, a store with user 1001 registered), serves the credential commands on it,
// runs `spawn preflight`, `credential add --kind subscription --user 1001 --label owner`
// and `credential login <id>` on the built binary and logs their output; the login command is
// the owner's to run. No engine turn runs; skipped when the variables are unset.
func TestOwnerSubscription(t *testing.T) {
	root, engine := os.Getenv("BROODFORGE_OWNER_ROOT"), os.Getenv("BROODFORGE_OWNER_ENGINE")
	if root == "" {
		t.Skip("BROODFORGE_OWNER_ROOT is not set")
	}
	if engine == "" {
		t.Fatal("BROODFORGE_OWNER_ENGINE is not set: name the engine's .exe")
	}
	if _, err := os.Stat(root); err == nil {
		t.Fatalf("%s exists; the owner root is made once, name a new one", root)
	}
	raw, version, err := spawn.EngineVersion(engine)
	if err != nil {
		t.Fatalf("%s --version: %v", engine, err)
	}
	t.Logf("engine %s: %s", engine, raw)
	writeInstance(t, root, engine, version, spawn.NameSameAccount)
	keyFile := filepath.Join(root, "test.key")
	if _, err := credentials.Init(keyFile); err != nil {
		t.Fatal(err)
	}
	st, err := store.Create(root)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.Write(func(tx *store.Tx) error {
		_, err := tx.Exec(`INSERT INTO registration (user_id, git_name, git_email, display_name, registered_at)
VALUES ('1001', 'Broodforge Owner Test', 'owner@broodforge.invalid', 'owner', '2026-10-16T00:00:00Z')`)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	backend, err := spawn.NewSameAccount(spawn.Config{Backend: spawn.NameSameAccount, EnginePath: engine})
	if err != nil {
		t.Fatal(err)
	}
	svc, err := credentials.NewService(st, audit.NewWriter(nil), credentials.Config{Root: root, KeyFile: keyFile,
		EnginePath: engine, Spawn: backend})
	if err != nil {
		t.Fatal(err)
	}
	addr, err := control.Address(root, "p7-test")
	if err != nil {
		t.Fatal(err)
	}
	reg := control.NewRegistry()
	cli.CredentialHandlers(svc)(reg)
	srv, err := control.Listen(addr, reg)
	if err != nil {
		t.Fatal(err)
	}
	served := make(chan error, 1)
	go func() { served <- srv.Serve() }()
	defer func() {
		if err := srv.Close(); err != nil {
			t.Error(err)
		}
		if err := <-served; err != nil {
			t.Error(err)
		}
	}()

	pf := clitest.Run(t, nil, "--root", root, "spawn", "preflight")
	if pf.Code != 0 {
		t.Fatalf("spawn preflight: exit %d\n%s%s", pf.Code, pf.Stdout, pf.Stderr)
	}
	add := runStdin(t, nil, "", "--root", root, "credential", "add", "--kind", "subscription", "--user", "1001", "--label", "owner")
	id := strings.TrimSpace(add.Stdout)
	if add.Code != 0 || !credID.MatchString(id) {
		t.Fatalf("credential add: exit %d stdout %q stderr %q", add.Code, add.Stdout, add.Stderr)
	}
	login := clitest.Run(t, nil, "--root", root, "credential", "login", id)
	if login.Code != 0 {
		t.Fatalf("credential login: exit %d stderr %q", login.Code, login.Stderr)
	}
	t.Logf("root        %s", filepath.ToSlash(root))
	t.Logf("credential  %s", id)
	t.Logf("login       %s", strings.TrimSpace(login.Stdout))
}
