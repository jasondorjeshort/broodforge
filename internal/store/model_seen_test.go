// SPDX-License-Identifier: GPL-3.0-or-later

package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestModelSeenBackfill: migration 0005 on a schema-4 store gives each model with usage one
// `turn` row at the start of its first turn with usage (the usage row's time without a turn), so a
// session from before P38 keeps owing the models that worked on its earlier commits.
func TestModelSeenBackfill(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Dir(Path(root)), 0o700); err != nil {
		t.Fatal(err)
	}
	conn, err := openConn(Path(root))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC)
	if err := query1(conn, createMigrations, nil, nil); err != nil {
		t.Fatal(err)
	}
	for _, m := range migrations()[:4] {
		if err := applyOne(conn, m, now); err != nil {
			t.Fatalf("migration %s: %v", m.name, err)
		}
	}
	for _, stmt := range []string{
		`INSERT INTO registration (user_id, git_name, git_email, display_name, registered_at) VALUES ('1001', 'Ada', 'ada@example.com', '', '2026-01-01T00:00:00Z')`,
		`INSERT INTO credential (id, kind, provider, user_id, label, status, added_at, store_ref) VALUES ('cred-a', 'api-key', 'anthropic', '1001', 'a', 'usable', '2026-01-01T00:00:00Z', 'cred-a')`,
		`INSERT INTO session (id, channel_id, project_id, driver_id, state, harness, credential_id, mode, reach, started_at)
VALUES ('s0001', '100', 'p', '1001', 'waiting-for-input', 'claude-code', 'cred-a', 'policy', 'code', '2026-01-01T00:00:00Z')`,
		`INSERT INTO turn (session_id, seq, requester_id, input, started_at) VALUES ('s0001', 1, '1001', 'hi', '2026-01-01T00:05:00Z')`,
		`INSERT INTO turn (session_id, seq, requester_id, input, started_at) VALUES ('s0001', 2, '1001', 'again', '2026-01-01T00:09:00Z')`,
		`INSERT INTO usage_record (session_id, turn_seq, requester_id, credential_id, channel_id, project_id, harness, model,
    input_tokens, output_tokens, recorded_at)
VALUES ('s0001', 2, '1001', 'cred-a', '100', 'p', 'claude-code', 'claude-opus-5-5', 10, 20, '2026-01-01T00:10:00Z')`,
	} {
		if err := query1(conn, stmt, nil, nil); err != nil {
			t.Fatalf("%s: %v", stmt, err)
		}
	}
	if err := conn.Close(); err != nil {
		t.Fatal(err)
	}

	s, err := Create(root)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if id, err := SchemaID(s); err != nil || id != 5 {
		t.Fatalf("SchemaID %d %v, want 5", id, err)
	}
	var rows []string
	err = s.Query(`SELECT session_id || ' ' || model || ' ' || first_seen_at || ' ' || via || ' ' ||
    coalesce(tool_use_id, '-') || ' ' || coalesce(agent, '-') FROM model_seen`, nil, func(r *Row) error {
		rows = append(rows, r.Text(0))
		return nil
	})
	if want := "s0001 claude-opus-5-5 2026-01-01T00:09:00Z turn - -"; err != nil || len(rows) != 1 || rows[0] != want {
		t.Errorf("model_seen %q %v, want [%s]", rows, err, want)
	}
}
