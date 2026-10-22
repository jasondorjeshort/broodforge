// SPDX-License-Identifier: GPL-3.0-or-later

package credentials

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jasondorjeshort/broodforge/internal/audit"
	"github.com/jasondorjeshort/broodforge/internal/store"
)

func TestKeyInitAndRead(t *testing.T) {
	path := filepath.Join(t.TempDir(), "k", "test.key")
	key, err := Init(path)
	if err != nil || len(key) != 64 {
		t.Fatalf("Init: %q %v", key, err)
	}
	if fi, err := os.Stat(path); err != nil || fi.Mode().Perm()&0o222 != 0 {
		t.Errorf("key file writable: %v %v", fi.Mode(), err)
	}
	if _, err := Init(path); err == nil || err.(*Error).Code != CodeKeyExists {
		t.Errorf("second Init: %v, want key-exists", err)
	}
	if _, err := ReadKey(path); err != nil {
		t.Errorf("ReadKey: %v", err)
	}
	if _, err := ReadKey(path + ".missing"); err == nil || err.(*Error).Code != CodeKeyMissing {
		t.Errorf("missing key: %v", err)
	}
	if err := os.WriteFile(path+".bad", []byte("zz\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadKey(path + ".bad"); err == nil || err.(*Error).Code != CodeBadKey {
		t.Errorf("bad key: %v", err)
	}
}

func TestKeyPath(t *testing.T) {
	env := func(v string) func(string) (string, bool) {
		return func(string) (string, bool) { return v, v != "" }
	}
	cases := []struct {
		instance, configured, envValue, want string
	}{
		{"pc", "/etc/bf/pc.key", "", "/etc/bf/pc.key"},
		{"pc", "/etc/bf/pc.key", "/override.key", "/override.key"},
		{"scratch-0123456789ab", "data/scratch.key", "/override.key", "/root/data/scratch.key"},
	}
	for _, c := range cases {
		got, err := KeyPath("/root", c.instance, c.configured, env(c.envValue))
		if err != nil || got != c.want {
			t.Errorf("KeyPath(%s, %s, env %s) = %q %v, want %q", c.instance, c.configured, c.envValue, got, err, c.want)
		}
	}
	if _, err := KeyPath("/root", "pc", "", env("")); err == nil || err.(*Error).Code != CodeNoKeyFile {
		t.Errorf("nothing set: %v", err)
	}
}

func newVault(t *testing.T) (*Vault, string) {
	t.Helper()
	dir := t.TempDir()
	keyFile := filepath.Join(dir, "k.key")
	if _, err := Init(keyFile); err != nil {
		t.Fatal(err)
	}
	v, err := OpenVault(filepath.Join(dir, "credentials", "store.json"), keyFile)
	if err != nil {
		t.Fatal(err)
	}
	return v, dir
}

func TestVaultRemovesStaleTemp(t *testing.T) {
	v, dir := newVault(t)
	stale := filepath.Join(filepath.Dir(v.path), ".store-123.json")
	if err := os.MkdirAll(filepath.Dir(stale), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stale, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenVault(v.path, filepath.Join(dir, "k.key")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stale); err == nil {
		t.Error("a stale temp file survived OpenVault")
	}
}

func TestVault(t *testing.T) {
	v, dir := newVault(t)
	if err := v.Put("cred-a", []byte("secret-a")); err != nil {
		t.Fatal(err)
	}
	if err := v.Put("cred-b", []byte("secret-b")); err != nil {
		t.Fatal(err)
	}
	got, err := v.Get("cred-a")
	if err != nil || string(got) != "secret-a" {
		t.Fatalf("Get: %q %v", got, err)
	}
	raw, err := os.ReadFile(v.path)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(raw, []byte("secret")) {
		t.Error("the store file holds a value in the clear")
	}

	// The id is the additional data: an entry moved under another id does not open.
	var f vaultFile
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatal(err)
	}
	f.Entries["cred-a"] = f.Entries["cred-b"]
	if err := v.write(f); err != nil {
		t.Fatal(err)
	}
	if _, err := v.Get("cred-a"); err == nil || err.(*Error).Code != CodeStoreUnreadable {
		t.Errorf("swapped entry: %v, want store-unreadable", err)
	}

	// Another key does not open it either.
	other := filepath.Join(dir, "other.key")
	if _, err := Init(other); err != nil {
		t.Fatal(err)
	}
	v2, err := OpenVault(v.path, other)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := v2.Get("cred-b"); err == nil || err.(*Error).Code != CodeStoreUnreadable {
		t.Errorf("other key: %v, want store-unreadable", err)
	}

	if err := v.Delete("cred-b"); err != nil {
		t.Fatal(err)
	}
	if _, err := v.Get("cred-b"); err == nil || err.(*Error).Code != CodeUnknownCredential {
		t.Errorf("deleted: %v", err)
	}
}

func newService(t *testing.T) (*Service, *store.Store, string) {
	t.Helper()
	root := t.TempDir()
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
	if _, err := Init(keyFile); err != nil {
		t.Fatal(err)
	}
	svc, err := NewService(st, audit.NewWriter(nil), Config{Root: root, KeyFile: keyFile, EnginePath: "C:/Program Files/claude.exe"})
	if err != nil {
		t.Fatal(err)
	}
	return svc, st, root
}

func code(err error) string {
	if e, ok := IsRefusal(err); ok {
		return e.Code
	}
	if err != nil {
		return err.Error()
	}
	return ""
}

func TestServiceAPIKey(t *testing.T) {
	svc, st, _ := newService(t)
	r, err := svc.Add(KindAPIKey, "1001", "t", []byte("scratch"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(r.ID, "cred-") || r.Status != StatusUsable || r.StoreRef != r.ID {
		t.Errorf("record %+v", r)
	}
	if v, err := svc.Secret(r.ID); err != nil || string(v) != "scratch" {
		t.Errorf("Secret: %q %v", v, err)
	}
	list, err := List(st, "1001")
	if err != nil || len(list) != 1 || list[0] != r {
		t.Errorf("List: %+v %v", list, err)
	}
	if !Exists(st)(r.ID) || Exists(st)("cred-nope") {
		t.Error("Exists")
	}
	if _, err := svc.LoginCommand(r.ID); code(err) != CodeNotSubscription {
		t.Errorf("login of an api key: %v", err)
	}
	if _, err := svc.Remove(r.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Remove(r.ID); code(err) != CodeRemoved {
		t.Errorf("second remove: %v", err)
	}
	if _, err := svc.Secret(r.ID); code(err) != CodeRemoved {
		t.Errorf("secret after remove: %v", err)
	}
	if Exists(st)(r.ID) {
		t.Error("a removed credential exists")
	}
	events, err := audit.Query(st, audit.Filter{Kinds: []audit.Kind{audit.KindCredentialAdded, audit.KindCredentialRemoved}})
	if err != nil || len(events) != 2 {
		t.Fatalf("audit: %v %v", events, err)
	}
	for _, e := range events {
		if bytes.Contains(e.Details, []byte("scratch")) {
			t.Errorf("audit details hold the value: %s", e.Details)
		}
	}
}

func TestServiceSubscription(t *testing.T) {
	svc, _, root := newService(t)
	r, err := svc.Add(KindSubscription, "1001", "owner", nil)
	if err != nil {
		t.Fatal(err)
	}
	if r.Status != StatusNeedsLogin || r.StoreRef != filepath.ToSlash(filepath.Join(root, "harness", r.ID)) {
		t.Errorf("record %+v", r)
	}
	if st, err := os.Stat(r.StoreRef); err != nil || !st.IsDir() {
		t.Errorf("harness-create made no %s: %v", r.StoreRef, err)
	}
	cmd, err := svc.LoginCommand(r.ID)
	if err != nil {
		t.Fatal(err)
	}
	if want := "CLAUDE_CONFIG_DIR=" + shellQuote(r.StoreRef) + " 'C:/Program Files/claude.exe' auth login"; cmd != want {
		t.Errorf("login command\n got %q\nwant %q", cmd, want)
	}
	done, err := svc.LoginDone(r.ID)
	if err != nil || done.Status != StatusUsable {
		t.Errorf("LoginDone: %+v %v", done, err)
	}
	if _, err := svc.Secret(r.ID); code(err) != CodeBadKind {
		t.Errorf("secret of a subscription: %v", err)
	}
}

func TestServiceRefusals(t *testing.T) {
	svc, _, _ := newService(t)
	cases := []struct {
		kind, user, label, value, want string
	}{
		{"api-key", "1002", "t", "x", CodeUnknownUser},
		{"api-key", "1001", "t", "", CodeBadValue},
		{"subscription", "1001", "t", "x", CodeBadValue},
		{"oauth", "1001", "t", "x", CodeBadKind},
		{"api-key", "1001", "", "x", CodeBadValue},
	}
	for _, c := range cases {
		if _, err := svc.Add(c.kind, c.user, c.label, []byte(c.value)); code(err) != c.want {
			t.Errorf("Add(%s, %s, %q): %v, want %s", c.kind, c.user, c.label, err, c.want)
		}
	}
	if _, err := svc.Show("cred-nope"); code(err) != CodeUnknownCredential {
		t.Errorf("Show: %v", err)
	}
}

// TestServiceBotToken: the bot token is the instance's own credential, user_id null, read back
// by BotToken only; it has no login, and Add with a user refuses it.
func TestServiceBotToken(t *testing.T) {
	svc, st, root := newService(t)
	if _, err := svc.Add(KindBotToken, "1001", "bot", []byte("x")); code(err) != CodeBadKind {
		t.Errorf("a user's bot token: %v", err)
	}
	if _, err := svc.AddInstance(KindAPIKey, "k", []byte("x")); code(err) != CodeBadKind {
		t.Errorf("an instance api key: %v", err)
	}
	if _, err := svc.AddInstance(KindBotToken, "bot", nil); code(err) != CodeBadValue {
		t.Errorf("no value: %v", err)
	}
	r, err := svc.AddInstance(KindBotToken, "bot", []byte("placeholder"))
	if err != nil {
		t.Fatal(err)
	}
	if r.User != "" || r.Provider != ProviderDiscord || r.Status != StatusUsable || r.StoreRef != r.ID {
		t.Errorf("record %+v", r)
	}
	nullUser := false
	if err := st.Query("SELECT user_id IS NULL FROM credential WHERE id = ?", []store.Value{store.Text(r.ID)},
		func(row *store.Row) error { nullUser = row.Int(0) == 1; return nil }); err != nil || !nullUser {
		t.Errorf("user_id is not null: %v", err)
	}
	events, err := audit.Query(st, audit.Filter{Kinds: []audit.Kind{audit.KindCredentialAdded}})
	if err != nil || len(events) != 1 || !bytes.Contains(events[0].Details, []byte(`"user":null`)) || bytes.Contains(events[0].Details, []byte("placeholder")) {
		t.Errorf("credential-added: %v %v", events, err)
	}
	if v, err := BotToken(st, root, filepath.Join(root, "test.key"), r.ID); err != nil || string(v) != "placeholder" {
		t.Errorf("BotToken: %q %v", v, err)
	}
	if _, err := svc.LoginCommand(r.ID); code(err) != CodeBadKind {
		t.Errorf("login of a bot token: %v", err)
	}
	if _, err := svc.Secret(r.ID); code(err) != CodeBadKind {
		t.Errorf("the engine's secret: %v", err)
	}
	api, err := svc.Add(KindAPIKey, "1001", "k", []byte("scratch"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := BotToken(st, root, filepath.Join(root, "test.key"), api.ID); code(err) != CodeBadKind {
		t.Errorf("BotToken of an api key: %v", err)
	}
	if _, err := svc.Remove(r.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := BotToken(st, root, filepath.Join(root, "test.key"), r.ID); code(err) != CodeRemoved {
		t.Errorf("BotToken after remove: %v", err)
	}
}
