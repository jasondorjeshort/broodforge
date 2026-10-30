// SPDX-License-Identifier: GPL-3.0-or-later

package store

import "testing"

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
		RateLimitType: "five_hour", FiveHourUtilization: 0.16, FiveHourResetsAt: 1792141200,
		SevenDayUtilization: 0.04, SevenDayResetsAt: 1792728000, OverageStatus: "rejected"}
	later := MeterRow{Credential: "cred-a", SeenAt: "2026-10-30T12:05:00Z", Session: "s0002", Status: "allowed_warning",
		RateLimitType: "seven_day", FiveHourUtilization: 0.09, FiveHourResetsAt: 1792248000,
		SevenDayUtilization: 0.68, SevenDayResetsAt: 1792728000, OverageStatus: "rejected", IsUsingOverage: true}
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
