// SPDX-License-Identifier: GPL-3.0-or-later

package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestMeterUpsert: no row before an event; two events for one credential leave one row, the
// later one's; a credential the store does not hold is refused by the reference (P28).
func TestMeterUpsert(t *testing.T) {
	s, err := Create(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.Write(func(tx *Tx) error {
		_, err := tx.Exec(`INSERT INTO credential (id, kind, provider, user_id, label, status, added_at, store_ref)
VALUES ('cred-a', 'api-key', 'anthropic', NULL, 'l', 'usable', '2026-10-30T00:00:00Z', 'cred-a')`)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if m, err := Meter(s, "cred-a"); err != nil || m != nil {
		t.Fatalf("before any event: %+v, %v; want nil", m, err)
	}
	first := MeterRow{Credential: "cred-a", SeenAt: "2026-10-30T12:00:00Z", Session: "s0001", Status: "allowed",
		RateLimitType: "five_hour", HasFiveHour: true, FiveHourUtilization: 0.16, FiveHourResetsAt: 1792141200,
		HasSevenDay: true, SevenDayUtilization: 0.04, SevenDayResetsAt: 1792728000, OverageStatus: "rejected"}
	later := MeterRow{Credential: "cred-a", SeenAt: "2026-10-30T12:05:00Z", Session: "s0002", Status: "allowed_warning",
		RateLimitType: "seven_day", HasFiveHour: true, FiveHourUtilization: 0.09, FiveHourResetsAt: 1792248000,
		HasSevenDay: true, SevenDayUtilization: 0.68, SevenDayResetsAt: 1792728000, OverageStatus: "rejected", IsUsingOverage: true}
	for _, m := range []MeterRow{first, later} {
		if err := SaveMeter(s, m); err != nil {
			t.Fatal(err)
		}
	}
	rows := 0
	if err := s.Query("SELECT count(*) FROM credential_meter", nil, func(r *Row) error { rows = int(r.Int(0)); return nil }); err != nil || rows != 1 {
		t.Errorf("%d rows, %v; want 1", rows, err)
	}
	if m, err := Meter(s, "cred-a"); err != nil || m == nil || *m != later {
		t.Errorf("Meter = %+v, %v; want %+v", m, err, later)
	}
	if err := SaveMeter(s, MeterRow{Credential: "cred-none", SeenAt: "x"}); err == nil {
		t.Error("a meter for a credential the store does not hold was written")
	}
}

// TestMeterAbsentWindow: a window the event did not carry is stored as NULL and read back
// absent, never as zeros, and either window may be the absent one (ADR 0021, Amendments
// 2026-11-07); a present window at utilization 0 stays present.
func TestMeterAbsentWindow(t *testing.T) {
	s, err := Create(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.Write(func(tx *Tx) error {
		_, err := tx.Exec(`INSERT INTO credential (id, kind, provider, user_id, label, status, added_at, store_ref)
VALUES ('cred-a', 'api-key', 'anthropic', NULL, 'l', 'usable', '2026-10-30T00:00:00Z', 'cred-a')`)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	for _, m := range []MeterRow{
		{Credential: "cred-a", SeenAt: "2026-11-07T17:23:33Z", Session: "s0001", Status: "allowed", RateLimitType: "seven_day",
			HasSevenDay: true, SevenDayUtilization: 0.33, SevenDayResetsAt: 1794542400, OverageStatus: "rejected"},
		{Credential: "cred-a", SeenAt: "2026-11-07T17:24:00Z", Session: "s0001", Status: "allowed", RateLimitType: "five_hour",
			HasFiveHour: true, FiveHourResetsAt: 1792141200, OverageStatus: "rejected"},
		{Credential: "cred-a", SeenAt: "2026-11-07T17:25:00Z", Session: "s0001", Status: "allowed"},
	} {
		if err := SaveMeter(s, m); err != nil {
			t.Fatal(err)
		}
		got, err := Meter(s, "cred-a")
		if err != nil || got == nil || *got != m {
			t.Errorf("Meter = %+v, %v; want %+v", got, err, m)
		}
		var nulls int
		if err := s.Query(`SELECT (five_hour_utilization IS NULL) + (five_hour_resets_at IS NULL) +
    (seven_day_utilization IS NULL) + (seven_day_resets_at IS NULL) FROM credential_meter`, nil,
			func(r *Row) error { nulls = int(r.Int(0)); return nil }); err != nil {
			t.Fatal(err)
		}
		want := 0
		if !m.HasFiveHour {
			want += 2
		}
		if !m.HasSevenDay {
			want += 2
		}
		if nulls != want {
			t.Errorf("%+v: %d NULL window columns, want %d", m, nulls, want)
		}
	}
}

// TestMeterMigration0006: on a schema-5 store, a row whose window 0004 stored as zeros (the
// event did not carry it) reads back absent after migration 0006, and a real window is kept.
func TestMeterMigration0006(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Dir(Path(root)), 0o700); err != nil {
		t.Fatal(err)
	}
	conn, err := openConn(Path(root))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 7, 0, 0, 0, 0, time.UTC)
	if err := query1(conn, createMigrations, nil, nil); err != nil {
		t.Fatal(err)
	}
	for _, m := range migrations()[:5] {
		if err := applyOne(conn, m, now); err != nil {
			t.Fatalf("migration %s: %v", m.name, err)
		}
	}
	for _, stmt := range []string{
		`INSERT INTO credential (id, kind, provider, user_id, label, status, added_at, store_ref) VALUES ('cred-a', 'api-key', 'anthropic', NULL, 'a', 'usable', '2026-01-01T00:00:00Z', 'cred-a')`,
		`INSERT INTO credential (id, kind, provider, user_id, label, status, added_at, store_ref) VALUES ('cred-b', 'api-key', 'anthropic', NULL, 'b', 'usable', '2026-01-01T00:00:00Z', 'cred-b')`,
		`INSERT INTO credential_meter VALUES ('cred-a', '2026-11-07T17:23:33Z', 's0001', 'allowed', 'seven_day', 0, 0, 0.33, 1794542400, 'rejected', 0)`,
		`INSERT INTO credential_meter VALUES ('cred-b', '2026-10-30T12:00:00Z', 's0002', 'allowed', 'five_hour', 0.16, 1792141200, 0.04, 1792728000, 'rejected', 1)`,
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
	for _, want := range []MeterRow{
		{Credential: "cred-a", SeenAt: "2026-11-07T17:23:33Z", Session: "s0001", Status: "allowed", RateLimitType: "seven_day",
			HasSevenDay: true, SevenDayUtilization: 0.33, SevenDayResetsAt: 1794542400, OverageStatus: "rejected"},
		{Credential: "cred-b", SeenAt: "2026-10-30T12:00:00Z", Session: "s0002", Status: "allowed", RateLimitType: "five_hour",
			HasFiveHour: true, FiveHourUtilization: 0.16, FiveHourResetsAt: 1792141200,
			HasSevenDay: true, SevenDayUtilization: 0.04, SevenDayResetsAt: 1792728000, OverageStatus: "rejected", IsUsingOverage: true},
	} {
		if got, err := Meter(s, want.Credential); err != nil || got == nil || *got != want {
			t.Errorf("Meter(%s) = %+v, %v; want %+v", want.Credential, got, err, want)
		}
	}
}
