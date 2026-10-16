// SPDX-License-Identifier: GPL-3.0-or-later

package backup

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jasondorjeshort/broodforge/internal/store"
)

// privateTemp points the OS temp directory at a fresh directory, so the rehearsal's scratch
// roots are the test's own and a leftover is visible.
func privateTemp(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("TMPDIR", dir)
	t.Setenv("TMP", dir)
	t.Setenv("TEMP", dir)
	return dir
}

// copyConfig stands in for config.CopyForScratch: an instance file naming the scratch root.
func copyConfig(_, dst, name string) error {
	if err := os.MkdirAll(filepath.Join(dst, "config"), 0o700); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dst, "config", "instance.json"), []byte(`{"instanceName":"`+name+`"}`), 0o600)
}

func noTree(string) error { return nil }

// sourceRoot is a root whose store records one session and its worktree, whose tree exists
// in the source (as on a live instance), then holds one snapshot. The restored copy has no
// tree, so the rehearsal must report it missing.
func sourceRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	s, err := store.Create(root)
	if err != nil {
		t.Fatal(err)
	}
	tree := filepath.ToSlash(filepath.Join(root, "sessions", "s0001", "tree"))
	if err := os.MkdirAll(tree, 0o700); err != nil {
		t.Fatal(err)
	}
	err = s.Write(func(tx *store.Tx) error {
		for _, q := range []string{
			"INSERT INTO registration (user_id, git_name, git_email, display_name, registered_at) VALUES ('1001', 'A', 'a@example.com', 'a', '2026-10-16T00:00:00Z')",
			"INSERT INTO credential (id, kind, provider, user_id, label, status, added_at, store_ref) VALUES ('cred-a', 'api-key', 'anthropic', '1001', 'l', 'usable', '2026-10-16T00:00:00Z', 'cred-a')",
			"INSERT INTO session (id, channel_id, project_id, driver_id, state, harness, credential_id, mode, reach, started_at) VALUES ('s0001', '100', 'p', '1001', 'ended', 'fake', 'cred-a', 'prompt', 'code', '2026-10-16T00:00:00Z')",
		} {
			if _, err := tx.Exec(q); err != nil {
				return err
			}
		}
		_, err := tx.Exec("INSERT INTO worktree (path, project_id, session_id, branch, state, created_at) VALUES (?, 'p', 's0001', 'b', 'kept', '2026-10-16T00:00:00Z')", store.Text(tree))
		return err
	})
	if cerr := s.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Snapshot(root, lineExport); err != nil {
		t.Fatal(err)
	}
	return root
}

func scratchLeft(t *testing.T, tmp string) []string {
	t.Helper()
	entries, err := os.ReadDir(tmp)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range entries {
		if store.IsScratchName(e.Name()) {
			out = append(out, e.Name())
		}
	}
	return out
}

func TestRehearse(t *testing.T) {
	src := sourceRoot(t)
	tmp := privateTemp(t)
	res, kept, err := Rehearse(Rehearsal{Root: src, CopyConfig: copyConfig, RemoveTree: noTree})
	if err != nil {
		t.Fatal(err)
	}
	latest, _ := Latest(src)
	if res.Result != "ok" || res.Migration != store.Known() || res.Path != latest || res.ReconcileMissing != 1 || kept != "" || res.Root != "" {
		t.Errorf("rehearse: %+v kept %q", res, kept)
	}
	rows := map[string]int64{}
	for _, tr := range res.Tables {
		rows[tr.Table] = tr.Rows
	}
	if rows["session"] != 1 || rows["worktree"] != 1 || rows["credential"] != 1 {
		t.Errorf("rows %v", rows)
	}
	if left := scratchLeft(t, tmp); len(left) != 0 {
		t.Errorf("scratch roots left: %v", left)
	}

	res, kept, err = Rehearse(Rehearsal{Root: src, Keep: true, CopyConfig: copyConfig, RemoveTree: noTree})
	if err != nil {
		t.Fatal(err)
	}
	if kept == "" || res.Root != kept || !store.IsScratchName(filepath.Base(kept)) {
		t.Fatalf("keep: %+v kept %q", res, kept)
	}
	if _, err := os.Stat(store.Path(kept)); err != nil {
		t.Errorf("kept root has no store: %v", err)
	}
	if err := store.DeleteScratch(kept, store.FixedName(filepath.Base(kept)), noTree); err != nil {
		t.Fatal(err)
	}
}

func TestRehearseFailuresKeepTheRoot(t *testing.T) {
	src := sourceRoot(t)
	tmp := privateTemp(t)
	latest, err := Latest(src)
	if err != nil {
		t.Fatal(err)
	}

	// No snapshot file: refused before a root is made.
	_, kept, err := Rehearse(Rehearsal{Root: src, Path: latest + ".nope", CopyConfig: copyConfig, RemoveTree: noTree})
	wantCode(t, err, CodeNoSnapshot)
	if kept != "" || len(scratchLeft(t, tmp)) != 0 {
		t.Errorf("no snapshot left a root: %q", kept)
	}

	// A truncated snapshot does not open: the root is kept for inspection.
	b, err := os.ReadFile(latest)
	if err != nil {
		t.Fatal(err)
	}
	bad := filepath.Join(t.TempDir(), "truncated.sqlite")
	if err := os.WriteFile(bad, b[:len(b)/3], 0o600); err != nil {
		t.Fatal(err)
	}
	_, kept, err = Rehearse(Rehearsal{Root: src, Path: bad, CopyConfig: copyConfig, RemoveTree: noTree})
	if err == nil || kept == "" {
		t.Fatalf("truncated: err %v kept %q", err, kept)
	}
	if _, serr := os.Stat(kept); serr != nil {
		t.Errorf("kept root missing: %v", serr)
	}
	t.Logf("truncated snapshot: %v", err)

	// A snapshot newer than the binary is refused by the migration step, root kept.
	nr := t.TempDir()
	if err := os.MkdirAll(filepath.Dir(store.Path(nr)), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(store.Path(nr), b, 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := store.Create(nr)
	if err != nil {
		t.Fatal(err)
	}
	err = s.Write(func(tx *store.Tx) error {
		_, err := tx.Exec("INSERT INTO migrations (id, applied_at) VALUES (?, '2026-10-16T00:00:00Z')", store.Int(store.Known()+1))
		return err
	})
	if cerr := s.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		t.Fatal(err)
	}
	newer := filepath.Join(t.TempDir(), "newer.sqlite") // a snapshot refuses a newer store, so copy it
	raw, err := os.ReadFile(store.Path(nr))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newer, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	_, kept2, err := Rehearse(Rehearsal{Root: src, Path: newer, CopyConfig: copyConfig, RemoveTree: noTree})
	wantCode(t, err, store.CodeStoreSchema)
	if kept2 == "" || kept2 == kept {
		t.Errorf("newer: kept %q", kept2)
	}
	if left := scratchLeft(t, tmp); len(left) != 2 {
		t.Errorf("want the two failed rehearsals kept, got %v", left)
	}
	for _, k := range []string{kept, kept2} {
		if err := store.DeleteScratch(k, store.FixedName(filepath.Base(k)), noTree); err != nil {
			t.Error(err)
		}
	}
	if left := scratchLeft(t, tmp); len(left) != 0 {
		t.Errorf("left: %v", left)
	}
}
