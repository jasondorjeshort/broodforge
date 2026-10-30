// SPDX-License-Identifier: GPL-3.0-or-later

package cli_test

import (
	"testing"

	"github.com/jasondorjeshort/broodforge/internal/cli/clitest"
	"github.com/jasondorjeshort/broodforge/internal/store"
)

// TestMeterReads: usage --credential ends in `meter  -` and credential show --json carries
// "meter":null before any rate-limit event; after one, both carry the row (P28; 05-debug-interface.md,
// Credentials and Usage and audit queries). The text form of credential show is unchanged.
func TestMeterReads(t *testing.T) {
	root, _ := storeRoot(t)
	s, err := store.Create(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Write(func(tx *store.Tx) error {
		_, err := tx.Exec(`INSERT INTO credential (id, kind, provider, user_id, label, status, added_at, store_ref)
VALUES ('cred-a', 'api-key', 'anthropic', NULL, 'l', 'usable', '2026-10-30T00:00:00Z', 'cred-a')`)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	const total = "total  0  0  0  0  0  0\n"
	const totalJSON = `{"model":"total","turns":0,"input":0,"output":0,"cacheRead":0,"cacheWrite":0,"interruptedTurns":0}` + "\n"
	const record = `{"id":"cred-a","kind":"api-key","provider":"anthropic","user":null,"label":"l","status":"usable","addedAt":"2026-10-30T00:00:00Z","lastUsedAt":null,"storeRef":"cred-a",`
	want(t, clitest.Run(t, nil, "--root", root, "usage", "--credential", "cred-a"), 0, total+"meter  -\n", "")
	want(t, clitest.Run(t, nil, "--root", root, "--json", "usage", "--credential", "cred-a"), 0, totalJSON+`{"meter":null}`+"\n", "")
	want(t, clitest.Run(t, nil, "--root", root, "credential", "show", "cred-a", "--json"), 0, record+`"meter":null}`+"\n", "")

	if err := store.SaveMeter(s, store.MeterRow{Credential: "cred-a", SeenAt: "2026-10-30T12:00:00Z", Session: "s0001",
		Status: "allowed", RateLimitType: "five_hour", FiveHourUtilization: 0.16, FiveHourResetsAt: 1792141200,
		SevenDayUtilization: 0.04, SevenDayResetsAt: 1792728000, OverageStatus: "rejected"}); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	const meter = `{"status":"allowed","rateLimitType":"five_hour","fiveHour":{"utilization":0.16,"resetsAt":1792141200},"sevenDay":{"utilization":0.04,"resetsAt":1792728000},"overageStatus":"rejected","isUsingOverage":false,"seenAt":"2026-10-30T12:00:00Z","session":"s0001"}`
	want(t, clitest.Run(t, nil, "--root", root, "usage", "--credential", "cred-a"), 0, total+"meter  0.16  0.04  allowed  2026-10-30T12:00:00Z\n", "")
	want(t, clitest.Run(t, nil, "--root", root, "--json", "usage", "--credential", "cred-a"), 0, totalJSON+`{"meter":`+meter+"}\n", "")
	want(t, clitest.Run(t, nil, "--root", root, "credential", "show", "cred-a", "--json"), 0, record+`"meter":`+meter+"}\n", "")
	want(t, clitest.Run(t, nil, "--root", root, "credential", "show", "cred-a"), 0, "cred-a  api-key  -  l  usable  -\n", "")
	// Another filter prints no meter line.
	want(t, clitest.Run(t, nil, "--root", root, "usage", "--session", "s0001"), 0, total, "")
}
