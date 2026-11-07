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
		Status: "allowed", RateLimitType: "five_hour", HasFiveHour: true, FiveHourUtilization: 0.16, FiveHourResetsAt: 1792141200,
		HasSevenDay: true, SevenDayUtilization: 0.04, SevenDayResetsAt: 1792728000, OverageStatus: "rejected"}); err != nil {
		t.Fatal(err)
	}
	const meter = `{"status":"allowed","rateLimitType":"five_hour","fiveHour":{"utilization":0.16,"resetsAt":1792141200},"sevenDay":{"utilization":0.04,"resetsAt":1792728000},"overageStatus":"rejected","isUsingOverage":false,"seenAt":"2026-10-30T12:00:00Z","session":"s0001"}`
	want(t, clitest.Run(t, nil, "--root", root, "usage", "--credential", "cred-a"), 0, total+"meter  0.16  0.04  allowed  2026-10-30T12:00:00Z\n", "")
	want(t, clitest.Run(t, nil, "--root", root, "--json", "usage", "--credential", "cred-a"), 0, totalJSON+`{"meter":`+meter+"}\n", "")
	want(t, clitest.Run(t, nil, "--root", root, "credential", "show", "cred-a", "--json"), 0, record+`"meter":`+meter+"}\n", "")
	want(t, clitest.Run(t, nil, "--root", root, "credential", "show", "cred-a"), 0, "cred-a  api-key  -  l  usable  -\n", "")
	// Another filter prints no meter line.
	want(t, clitest.Run(t, nil, "--root", root, "usage", "--session", "s0001"), 0, total, "")

	// A window the event did not carry is `-` in its own column and null in JSON, either one
	// (ADR 0021, Amendments 2026-11-07).
	for _, c := range []struct {
		row        store.MeterRow
		text, json string
	}{
		{store.MeterRow{Credential: "cred-a", SeenAt: "2026-11-07T17:23:33Z", Session: "s0002", Status: "allowed", RateLimitType: "seven_day",
			HasSevenDay: true, SevenDayUtilization: 0.33, SevenDayResetsAt: 1794542400, OverageStatus: "rejected"},
			"meter  -  0.33  allowed  2026-11-07T17:23:33Z\n",
			`{"status":"allowed","rateLimitType":"seven_day","fiveHour":null,"sevenDay":{"utilization":0.33,"resetsAt":1794542400},"overageStatus":"rejected","isUsingOverage":false,"seenAt":"2026-11-07T17:23:33Z","session":"s0002"}`},
		{store.MeterRow{Credential: "cred-a", SeenAt: "2026-11-07T17:24:00Z", Session: "s0002", Status: "allowed", RateLimitType: "five_hour",
			HasFiveHour: true, FiveHourResetsAt: 1792141200, OverageStatus: "rejected"},
			"meter  0  -  allowed  2026-11-07T17:24:00Z\n",
			`{"status":"allowed","rateLimitType":"five_hour","fiveHour":{"utilization":0,"resetsAt":1792141200},"sevenDay":null,"overageStatus":"rejected","isUsingOverage":false,"seenAt":"2026-11-07T17:24:00Z","session":"s0002"}`},
	} {
		if err := store.SaveMeter(s, c.row); err != nil {
			t.Fatal(err)
		}
		want(t, clitest.Run(t, nil, "--root", root, "usage", "--credential", "cred-a"), 0, total+c.text, "")
		want(t, clitest.Run(t, nil, "--root", root, "--json", "usage", "--credential", "cred-a"), 0, totalJSON+`{"meter":`+c.json+"}\n", "")
		want(t, clitest.Run(t, nil, "--root", root, "credential", "show", "cred-a", "--json"), 0, record+`"meter":`+c.json+"}\n", "")
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
}
