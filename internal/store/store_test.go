// SPDX-License-Identifier: GPL-3.0-or-later

package store

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

const triggerText = "audit_event is append-only (ADR 0021); change this trigger only through a numbered migration"

const (
	replaceInto     = "REPLACE INTO audit_event (seq, time, kind, actor) VALUES (1, 't', 'rewritten', 'attacker')"
	insertOrReplace = "INSERT OR REPLACE INTO audit_event (seq, time, kind, actor) VALUES (1, 't', 'rewritten', 'attacker')"
)

func TestClosedStore(t *testing.T) {
	s, _ := newStore(t)
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if err := s.Write(func(*Tx) error { return nil }); !errors.Is(err, ErrClosed) {
		t.Errorf("Write after Close: %v", err)
	}
	if err := s.Query("SELECT 1", nil, nil); !errors.Is(err, ErrClosed) {
		t.Errorf("Query after Close: %v", err)
	}
}

func newStore(t *testing.T) (*Store, string) {
	t.Helper()
	root := t.TempDir()
	s, err := Create(root)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s, root
}

func pragma(t *testing.T, q Querier, name string) string {
	t.Helper()
	var v string
	if err := q.Query("PRAGMA "+name, nil, func(r *Row) error { v = r.Text(0); return nil }); err != nil {
		t.Fatalf("PRAGMA %s: %v", name, err)
	}
	return v
}

func wantCode(t *testing.T, err error, code string) *Error {
	t.Helper()
	var se *Error
	if !errors.As(err, &se) || se.Code != code {
		t.Fatalf("got %v, want a %s error", err, code)
	}
	return se
}

func TestCreateMigratesAndSetsPragmas(t *testing.T) {
	s, root := newStore(t)
	if got, want := s.Path(), filepath.ToSlash(filepath.Join(root, "data", "store.sqlite")); got != want {
		t.Errorf("Path %q, want %q", got, want)
	}
	if id, err := SchemaID(s); err != nil || id != Known() || id < 1 {
		t.Errorf("SchemaID %d %v, want %d", id, err, Known())
	}
	for name, want := range map[string]string{"journal_mode": "wal", "synchronous": "2", "foreign_keys": "1", "busy_timeout": "5000"} {
		if got := pragma(t, s, name); got != want {
			t.Errorf("writer %s = %q, want %q", name, got, want)
		}
	}
	for _, table := range []string{"registration", "credential", "session", "participant", "worktree", "turn",
		"permission_decision", "git_commit", "usage_record", "audit_event", "migrations"} {
		n := 0
		if err := s.Query("SELECT 1 FROM sqlite_schema WHERE type = 'table' AND name = ?", []Value{Text(table)},
			func(*Row) error { n++; return nil }); err != nil || n != 1 {
			t.Errorf("table %s: %d %v", table, n, err)
		}
	}
	s.Close()
	again, err := Create(root) // a second start migrates nothing and succeeds
	if err != nil {
		t.Fatalf("second Create: %v", err)
	}
	defer again.Close()
	n := 0
	if err := again.Query("SELECT count(*) FROM migrations", nil, func(r *Row) error { n = int(r.Int(0)); return nil }); err != nil || int64(n) != Known() {
		t.Errorf("migrations rows %d %v, want %d", n, err, Known())
	}
}

func insertAudit(t *testing.T, s *Store) {
	t.Helper()
	err := s.Write(func(tx *Tx) error {
		_, err := tx.Insert("INSERT INTO audit_event (time, kind, actor) VALUES (?, ?, ?)",
			Text(FormatTime(time.Now())), Text("instance-started"), Text("instance"))
		return err
	})
	if err != nil {
		t.Fatalf("insert audit event: %v", err)
	}
}

// The P5 Proof's audit test: on a plain connection (no authorizer) UPDATE and DELETE of
// audit_event raise the trigger's text; on the writer the authorizer refuses them, and DROP
// TRIGGER, before any trigger runs.
func TestAuditAppendOnly(t *testing.T) {
	s, _ := newStore(t)
	insertAudit(t, s)

	plain, err := openConn(s.Path())
	if err != nil {
		t.Fatal(err)
	}
	defer plain.Close()
	for _, q := range []string{"UPDATE audit_event SET actor = 'x'", "DELETE FROM audit_event",
		replaceInto, insertOrReplace} {
		err := query1(plain, q, nil, nil)
		if err == nil || !strings.Contains(err.Error(), triggerText) {
			t.Errorf("plain %q: %v, want the trigger text", q, err)
		} else {
			t.Logf("plain %q: %v", q, err)
		}
	}

	for _, q := range []string{"UPDATE audit_event SET actor = 'x'", "DELETE FROM audit_event",
		"DROP TRIGGER audit_event_no_update", "DROP TRIGGER audit_event_no_delete", "DROP TABLE audit_event"} {
		err := s.Write(func(tx *Tx) error { _, err := tx.Exec(q); return err })
		if err == nil || !strings.Contains(err.Error(), "not authorized") || strings.Contains(err.Error(), triggerText) {
			t.Errorf("writer %q: %v, want the authorizer's refusal", q, err)
		} else {
			t.Logf("writer %q: %v", q, err)
		}
	}
	// REPLACE is an INSERT to the authorizer, so on the writer too the trigger refuses it.
	for _, q := range []string{replaceInto, insertOrReplace} {
		err := s.Write(func(tx *Tx) error { _, err := tx.Exec(q); return err })
		if err == nil || !strings.Contains(err.Error(), triggerText) {
			t.Errorf("writer %q: %v, want the trigger text", q, err)
		} else {
			t.Logf("writer %q: %v", q, err)
		}
	}
	// The ways around the guard: rename or alter the table, or write the schema directly.
	for _, q := range []string{"ALTER TABLE audit_event RENAME TO ae2", "ALTER TABLE audit_event DROP COLUMN details",
		"ALTER TABLE audit_event ADD COLUMN x TEXT", "PRAGMA writable_schema=ON", "PRAGMA writable_schema",
		"DELETE FROM sqlite_schema WHERE type = 'trigger'", "UPDATE sqlite_master SET sql = '' WHERE type = 'trigger'",
		"INSERT INTO sqlite_schema (type, name, tbl_name, rootpage, sql) VALUES ('table', 'x', 'x', 0, '')"} {
		err := s.Write(func(tx *Tx) error { _, err := tx.Exec(q); return err })
		if err == nil || !(strings.Contains(err.Error(), "not authorized") || strings.Contains(err.Error(), "may not be modified")) {
			t.Errorf("writer %q: %v, want the authorizer's or SQLite's refusal", q, err)
		} else {
			t.Logf("writer %q: %v", q, err)
		}
	}
	var seq1 string
	if err := s.Query("SELECT kind || '/' || actor FROM audit_event WHERE seq = 1", nil, func(r *Row) error { seq1 = r.Text(0); return nil }); err != nil || seq1 != "instance-started/instance" {
		t.Errorf("seq 1 after the attempts: %q %v", seq1, err)
	}
	n := 0
	if err := s.Query("SELECT count(*) FROM audit_event", nil, func(r *Row) error { n = int(r.Int(0)); return nil }); err != nil || n != 1 {
		t.Errorf("audit rows after the attempts: %d %v, want 1", n, err)
	}
	triggers := 0
	if err := s.Query("SELECT count(*) FROM sqlite_schema WHERE type = 'trigger' AND tbl_name = 'audit_event'", nil, func(r *Row) error { triggers = int(r.Int(0)); return nil }); err != nil || triggers != 3 {
		t.Errorf("audit triggers: %d %v, want 3", triggers, err)
	}
	// An ordinary insert still passes, and schema changes elsewhere are not guarded.
	insertAudit(t, s)
	err = s.Write(func(tx *Tx) error {
		if _, err := tx.Exec("CREATE TABLE scratch_t (a)"); err != nil {
			return err
		}
		if _, err := tx.Exec("ALTER TABLE scratch_t ADD COLUMN b"); err != nil {
			return err
		}
		_, err := tx.Exec("DROP TABLE scratch_t")
		return err
	})
	if err != nil {
		t.Errorf("schema change of another table: %v", err)
	}
	// Other tables are not guarded.
	err = s.Write(func(tx *Tx) error {
		if _, err := tx.Exec("INSERT INTO registration (user_id, git_name, git_email, display_name, registered_at) VALUES ('1', 'n', 'e', 'd', 't')"); err != nil {
			return err
		}
		_, err := tx.Exec("UPDATE registration SET git_name = 'm'")
		return err
	})
	if err != nil {
		t.Errorf("update of another table: %v", err)
	}
}

func TestWriteRollsBackAndRunsOnCommit(t *testing.T) {
	s, _ := newStore(t)
	ran := 0
	boom := errors.New("boom")
	err := s.Write(func(tx *Tx) error {
		tx.OnCommit(func() { ran++ })
		if _, err := tx.Insert("INSERT INTO audit_event (time, kind, actor) VALUES ('t', 'k', 'a')"); err != nil {
			return err
		}
		return boom
	})
	if !errors.Is(err, boom) || ran != 0 {
		t.Fatalf("rolled back write: err %v, OnCommit ran %d times", err, ran)
	}
	if err := s.Write(func(tx *Tx) error { tx.OnCommit(func() { ran++ }); return nil }); err != nil || ran != 1 {
		t.Fatalf("committed write: err %v, OnCommit ran %d times", err, ran)
	}
	n := 0
	if err := s.Query("SELECT count(*) FROM audit_event", nil, func(r *Row) error { n = int(r.Int(0)); return nil }); err != nil || n != 0 {
		t.Errorf("rows after rollback: %d %v", n, err)
	}
}

func TestSchemaNewerRefused(t *testing.T) {
	s, root := newStore(t)
	plain, err := openConn(s.Path())
	if err != nil {
		t.Fatal(err)
	}
	if err := query1(plain, "INSERT INTO migrations (id, applied_at) VALUES (9999, 't')", nil, nil); err != nil {
		t.Fatal(err)
	}
	plain.Close()
	s.Close()
	want := "store schema 9999, binary knows " + strconv.FormatInt(Known(), 10) + "; upgrade the binary or restore an older snapshot"
	_, err = Create(root)
	if se := wantCode(t, err, CodeStoreSchema); se.Message != want {
		t.Errorf("Create: %q, want %q", se.Message, want)
	}
	_, err = OpenReadOnly(root)
	if se := wantCode(t, err, CodeStoreSchema); se.Message != want {
		t.Errorf("OpenReadOnly: %q, want %q", se.Message, want)
	}
}

func TestReadOnlyRefusesAnUnmigratedStore(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "data"), 0o700); err != nil {
		t.Fatal(err)
	}
	plain, err := openConn(Path(root))
	if err != nil {
		t.Fatal(err)
	}
	if err := query1(plain, createMigrations, nil, nil); err != nil {
		t.Fatal(err)
	}
	plain.Close()
	_, err = OpenReadOnly(root)
	if se := wantCode(t, err, CodeStoreSchema); se.Message != "store schema 0, binary knows "+strconv.FormatInt(Known(), 10)+"; start the bot once to migrate it" {
		t.Errorf("message %q", se.Message)
	}
}

func TestOpenReadOnly(t *testing.T) {
	root := t.TempDir()
	_, err := OpenReadOnly(root)
	if se := wantCode(t, err, CodeNoStore); se.Message != "no store at "+Path(root) {
		t.Errorf("message %q", se.Message)
	}
	s, err := Create(root)
	if err != nil {
		t.Fatal(err)
	}
	insertAudit(t, s)
	s.Close()
	r, err := OpenReadOnly(root)
	if err != nil {
		t.Fatalf("OpenReadOnly: %v", err)
	}
	defer r.Close()
	for name, want := range map[string]string{"journal_mode": "wal", "synchronous": "2", "foreign_keys": "1", "busy_timeout": "5000"} {
		if got := pragma(t, r, name); got != want {
			t.Errorf("reader %s = %q, want %q", name, got, want)
		}
	}
	if err := r.Query("INSERT INTO audit_event (time, kind, actor) VALUES ('t', 'k', 'a')", nil, nil); err == nil {
		t.Error("a write through the reader succeeded")
	}
	n := 0
	if err := r.Query("SELECT count(*) FROM audit_event", nil, func(row *Row) error { n = int(row.Int(0)); return nil }); err != nil || n != 1 {
		t.Errorf("read %d %v", n, err)
	}
}

func TestOpenFileNotADatabase(t *testing.T) {
	p := filepath.Join(t.TempDir(), "x.sqlite")
	if err := os.WriteFile(p, []byte(strings.Repeat("not a database ", 100)), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := OpenFile(p)
	wantCode(t, err, CodeStoreOpen)
}

func seedUsage(t *testing.T, s *Store) {
	t.Helper()
	err := s.Write(func(tx *Tx) error {
		stmts := []string{
			"INSERT INTO registration (user_id, git_name, git_email, display_name, registered_at) VALUES ('1001', 'A', 'a@example.com', 'a', '2026-10-16T00:00:00Z')",
			"INSERT INTO registration (user_id, git_name, git_email, display_name, registered_at) VALUES ('1002', 'B', 'b@example.com', 'b', '2026-10-16T00:00:00Z')",
			"INSERT INTO credential (id, kind, provider, user_id, label, status, added_at, store_ref) VALUES ('cred-a', 'api-key', 'anthropic', '1001', 'l', 'usable', '2026-10-16T00:00:00Z', 'cred-a')",
			"INSERT INTO session (id, channel_id, project_id, driver_id, state, harness, credential_id, mode, reach, started_at) VALUES ('s0001', '100', 'p', '1001', 'ended', 'fake', 'cred-a', 'prompt', 'code', '2026-10-16T00:00:00Z')",
			"INSERT INTO session (id, channel_id, project_id, driver_id, state, harness, credential_id, mode, reach, started_at) VALUES ('s0002', '200', 'p', '1002', 'running-turn', 'fake', 'cred-a', 'policy', 'read-only', '2026-10-16T01:00:00Z')",
			"INSERT INTO turn (session_id, seq, requester_id, input, started_at) VALUES ('s0001', 1, '1001', 'x', '2026-10-16T00:00:00Z')",
			"INSERT INTO turn (session_id, seq, requester_id, input, started_at) VALUES ('s0001', 2, '1002', 'y', '2026-10-16T00:10:00Z')",
			"INSERT INTO turn (session_id, seq, requester_id, input, started_at) VALUES ('s0002', 1, '1002', 'z', '2026-10-16T01:00:00Z')",
		}
		for _, q := range stmts {
			if _, err := tx.Exec(q); err != nil {
				return err
			}
		}
		rows := []struct {
			session, requester, channel, model, at string
			turn, in, out, interrupted             int64
		}{
			{"s0001", "1001", "100", "opus", "2026-10-16T00:01:00Z", 1, 10, 20, 0},
			{"s0001", "1001", "100", "haiku", "2026-10-16T00:01:00Z", 1, 1, 2, 0},
			{"s0001", "1002", "100", "opus", "2026-10-16T00:11:00Z", 2, 100, 200, 1},
			{"s0002", "1002", "200", "opus", "2026-10-16T01:01:00Z", 1, 1000, 2000, 0},
		}
		for _, r := range rows {
			_, err := tx.Insert(`INSERT INTO usage_record (session_id, turn_seq, requester_id, credential_id, channel_id, project_id,
harness, model, input_tokens, output_tokens, cache_read_tokens, cache_write_tokens, interrupted, raw, recorded_at)
VALUES (?, ?, ?, 'cred-a', ?, 'p', 'fake', ?, ?, ?, 3, 4, ?, '{"usage":{"input_tokens":1}}', ?)`,
				Text(r.session), Int(r.turn), Text(r.requester), Text(r.channel), Text(r.model), Int(r.in), Int(r.out),
				Int(r.interrupted), Text(r.at))
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
}

func TestUsage(t *testing.T) {
	s, _ := newStore(t)
	rows, total, err := UsageByModel(s, UsageFilter{})
	if err != nil || len(rows) != 0 || total != (UsageTotal{Model: TotalModel}) {
		t.Fatalf("empty: %v %+v %v", rows, total, err)
	}
	seedUsage(t, s)
	rows, total, err = UsageByModel(s, UsageFilter{})
	if err != nil {
		t.Fatal(err)
	}
	want := []UsageTotal{{"haiku", 1, 1, 2, 3, 4, 0}, {"opus", 3, 1110, 2220, 9, 12, 1}}
	if len(rows) != 2 || rows[0] != want[0] || rows[1] != want[1] {
		t.Errorf("by model %+v, want %+v", rows, want)
	}
	if wt := (UsageTotal{TotalModel, 3, 1111, 2222, 12, 16, 1}); total != wt {
		t.Errorf("total %+v, want %+v", total, wt)
	}
	for _, c := range []struct {
		f    UsageFilter
		want UsageTotal
	}{
		{UsageFilter{Session: "s0001"}, UsageTotal{TotalModel, 2, 111, 222, 9, 12, 1}},
		{UsageFilter{User: "1002"}, UsageTotal{TotalModel, 2, 1100, 2200, 6, 8, 1}},
		{UsageFilter{Credential: "cred-a"}, UsageTotal{TotalModel, 3, 1111, 2222, 12, 16, 1}},
		{UsageFilter{Channel: "200"}, UsageTotal{TotalModel, 1, 1000, 2000, 3, 4, 0}},
		{UsageFilter{Project: "nope"}, UsageTotal{Model: TotalModel}},
		{UsageFilter{Since: "2026-10-16T00:11:00Z"}, UsageTotal{TotalModel, 2, 1100, 2200, 6, 8, 1}},
	} {
		if _, got, err := UsageByModel(s, c.f); err != nil || got != c.want {
			t.Errorf("%+v: %+v %v, want %+v", c.f, got, err, c.want)
		}
	}
	turns, err := UsageByTurn(s, UsageFilter{Session: "s0001"})
	if err != nil || len(turns) != 3 {
		t.Fatalf("by turn: %+v %v", turns, err)
	}
	if tu := turns[2]; tu.Turn != 2 || tu.Model != "opus" || !tu.Interrupted || string(tu.Raw) != `[{"usage":{"input_tokens":1}}]` {
		t.Errorf("turn 2 %+v raw %s", tu, tu.Raw)
	}
}

func TestReconcile(t *testing.T) {
	s, root := newStore(t)
	seedUsage(t, s)
	present := filepath.ToSlash(filepath.Join(root, "sessions", "s0001", "tree"))
	gone := filepath.ToSlash(filepath.Join(root, "sessions", "s0002", "tree"))
	for _, d := range []string{present, filepath.Join(root, "sessions", "s0009")} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	err := s.Write(func(tx *Tx) error {
		if _, err := tx.Exec("INSERT INTO worktree (path, project_id, session_id, branch, state, created_at, missing_since) VALUES (?, 'p', 's0001', 'b1', 'kept', 't', '2026-10-15T00:00:00Z')", Text(present)); err != nil {
			return err
		}
		_, err := tx.Exec("INSERT INTO worktree (path, project_id, session_id, branch, state, created_at) VALUES (?, 'p', 's0002', 'b2', 'active', 't')", Text(gone))
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := Reconcile(s, root)
	if err != nil {
		t.Fatal(err)
	}
	want := []Finding{
		{FindingInterrupt, "s0002", "s0002", "running-turn"},
		{FindingPresent, present, "s0001", "2026-10-15T00:00:00Z"},
		{FindingMissing, gone, "s0002", ""},
		{FindingUnrecorded, filepath.ToSlash(filepath.Join(root, "sessions", "s0009")), "", ""},
	}
	if len(got) != len(want) {
		t.Fatalf("findings %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("finding %d: %+v, want %+v", i, got[i], want[i])
		}
	}
	n := 0
	if err := s.Query("SELECT count(*) FROM worktree WHERE missing_since IS NULL", nil, func(r *Row) error { n = int(r.Int(0)); return nil }); err != nil || n != 1 {
		t.Errorf("reconcile wrote: %d rows without missing_since, %v", n, err)
	}
}
