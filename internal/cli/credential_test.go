// SPDX-License-Identifier: GPL-3.0-or-later

package cli_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/jasondorjeshort/broodforge/internal/audit"
	"github.com/jasondorjeshort/broodforge/internal/cli"
	"github.com/jasondorjeshort/broodforge/internal/cli/clitest"
	"github.com/jasondorjeshort/broodforge/internal/control"
	"github.com/jasondorjeshort/broodforge/internal/control/controltest"
	"github.com/jasondorjeshort/broodforge/internal/credentials"
	"github.com/jasondorjeshort/broodforge/internal/store"
)

// The credential half of the P6 Proof, through the harness: a control server on a temporary
// root serving the credential handlers over a store the test made, the rows P10's bot owns.

func writeFile(path, content string) error { return os.WriteFile(path, []byte(content), 0o600) }

// credentialHarness serves the credential commands on a fresh root with user 1001
// registered; stop ends the server so the read fallback can be tried on the same root.
func credentialHarness(t *testing.T) (root string, stop func()) {
	t.Helper()
	root, _, addr := controltest.Root(t)
	st, err := store.Create(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	if err := st.Write(func(tx *store.Tx) error {
		_, err := tx.Exec(`INSERT INTO registration (user_id, git_name, git_email, display_name, registered_at)
VALUES ('1001', 'A', 'a@example.com', 'A', '2026-10-16T00:00:00Z')`)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	keyFile := filepath.Join(root, "test.key")
	if _, err := credentials.Init(keyFile); err != nil {
		t.Fatal(err)
	}
	svc, err := credentials.NewService(st, audit.NewWriter(nil), credentials.Config{Root: root, KeyFile: keyFile, EnginePath: "/opt/claude"})
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
	stopped := false
	stop = func() {
		if stopped {
			return
		}
		stopped = true
		if err := srv.Close(); err != nil {
			t.Errorf("close: %v", err)
		}
		if err := <-served; err != nil {
			t.Errorf("serve: %v", err)
		}
	}
	t.Cleanup(stop)
	return root, stop
}

var credID = regexp.MustCompile(`^cred-[0-9a-z]+$`)

func TestProofCredentials(t *testing.T) {
	root, stop := credentialHarness(t)

	r := runStdin(t, nil, "scratch", "--root", root, "credential", "add", "--kind", "api-key", "--user", "1001", "--label", "t")
	id := strings.TrimSuffix(r.Stdout, "\n")
	if r.Code != 0 || r.Stderr != "" || !credID.MatchString(id) {
		t.Fatalf("add: exit %d stdout %q stderr %q", r.Code, r.Stdout, r.Stderr)
	}

	r = clitest.Run(t, nil, "--root", root, "credential", "show", id, "--json")
	if r.Code != 0 || strings.Contains(r.Stdout, "scratch") {
		t.Fatalf("show --json: exit %d stdout %q", r.Code, r.Stdout)
	}
	var rec map[string]json.RawMessage
	if err := json.Unmarshal([]byte(r.Stdout), &rec); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"id", "kind", "provider", "user", "label", "status", "addedAt", "lastUsedAt", "storeRef", "meter"} {
		if _, ok := rec[k]; !ok {
			t.Errorf("show --json lacks %s", k)
		}
	}
	if string(rec["meter"]) != "null" {
		t.Errorf("show --json: meter %s before any rate-limit event, want null", rec["meter"])
	}
	if len(rec) != 10 {
		t.Errorf("show --json has %d keys: %s", len(rec), r.Stdout)
	}
	want(t, clitest.Run(t, nil, "--root", root, "credential", "show", id), 0, id+"  api-key  1001  t  usable  -\n", "")
	want(t, clitest.Run(t, nil, "--root", root, "credential", "show", "cred-nope"), 1, "", "error: unknown-credential: no credential cred-nope\n")

	// No value on stdin is a usage error, nothing is sent.
	want(t, runStdin(t, nil, "", "--root", root, "credential", "add", "--kind", "api-key", "--user", "1001", "--label", "t"), 2, "",
		"error: usage: credential add: --kind api-key reads its value from stdin, 1 to 65536 bytes\n")
	want(t, runStdin(t, nil, "x", "--root", root, "credential", "add", "--kind", "api-key", "--user", "1002", "--label", "t"), 1, "",
		"error: unknown-user: 1002 is not registered\n")

	// A subscription: needs-login, the login command, then --done.
	r = runStdin(t, nil, "", "--root", root, "credential", "add", "--kind", "subscription", "--user", "1001", "--label", "owner")
	sub := strings.TrimSuffix(r.Stdout, "\n")
	if r.Code != 0 || !credID.MatchString(sub) {
		t.Fatalf("add subscription: exit %d stdout %q stderr %q", r.Code, r.Stdout, r.Stderr)
	}
	dir := filepath.ToSlash(filepath.Join(root, "harness", sub))
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		t.Errorf("harness-create made no %s: %v", dir, err)
	}
	want(t, clitest.Run(t, nil, "--root", root, "credential", "login", sub), 0,
		"CLAUDE_CONFIG_DIR="+dir+" /opt/claude auth login\n", "")
	want(t, clitest.Run(t, nil, "--root", root, "credential", "login", id), 1, "",
		"error: not-subscription: "+id+" is an api-key credential\n")
	want(t, clitest.Run(t, nil, "--root", root, "credential", "login", "--done", sub), 0, sub+"  subscription  1001  owner  usable  -\n", "")

	want(t, clitest.Run(t, nil, "--root", root, "credential", "remove", id), 0, id+"  api-key  1001  t  removed  -\n", "")
	want(t, clitest.Run(t, nil, "--root", root, "credential", "list"), 0,
		id+"  api-key  1001  t  removed  -\n"+sub+"  subscription  1001  owner  usable  -\n", "")

	// Without the bot: reads fall back to the store file, writes are not-running.
	stop()
	want(t, clitest.Run(t, nil, "--root", root, "credential", "list", "--user", "1001"), 0,
		id+"  api-key  1001  t  removed  -\n"+sub+"  subscription  1001  owner  usable  -\n", "")
	want(t, clitest.Run(t, nil, "--root", root, "credential", "show", sub), 0, sub+"  subscription  1001  owner  usable  -\n", "")
	r = runStdin(t, nil, "x", "--root", root, "credential", "add", "--kind", "api-key", "--user", "1001", "--label", "t")
	if r.Code != 1 || !strings.HasPrefix(r.Stderr, "error: not-running: no bot at ") {
		t.Errorf("add without the bot: exit %d stderr %q", r.Code, r.Stderr)
	}

	// The value is in the store file, sealed, and nowhere in the clear.
	b, err := os.ReadFile(filepath.Join(root, "credentials", "store.json"))
	if err != nil || strings.Contains(string(b), "scratch") {
		t.Errorf("store file: %v %q", err, b)
	}
}

func TestProofCredentialInit(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "config", "projects"), 0o700); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"instance.json", "projects/scratch.json"} {
		b, err := os.ReadFile(filepath.Join("../../tests/config/valid/config", f))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "config", f), b, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	key := filepath.ToSlash(filepath.Join(root, "keys", "test.key"))
	env := []string{"BROODFORGE_KEY_FILE=" + key}
	r := clitest.Run(t, env, "--root", root, "credential", "init")
	if r.Code != 0 || !regexp.MustCompile(`^[0-9a-f]{64}  `+regexp.QuoteMeta(key)+"\n$").MatchString(r.Stdout) {
		t.Fatalf("init: exit %d stdout %q stderr %q", r.Code, r.Stdout, r.Stderr)
	}
	b, err := os.ReadFile(key)
	if err != nil || strings.TrimSpace(string(b)) != r.Stdout[:64] {
		t.Errorf("key file %q %v", b, err)
	}
	want(t, clitest.Run(t, env, "--root", root, "credential", "init"), 1, "", "error: key-exists: "+key+"\n")
}
