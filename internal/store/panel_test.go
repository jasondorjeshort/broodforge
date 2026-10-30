// SPDX-License-Identifier: GPL-3.0-or-later

package store

import (
	"reflect"
	"testing"
)

// TestPanels: one row per channel, replaced by the next save; Discord's id is null until known.
func TestPanels(t *testing.T) {
	s, err := Create(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	a := PanelRow{Channel: "100", MessageID: "m2", PostedAt: "2026-10-30T12:00:00Z"}
	b := PanelRow{Channel: "200", MessageID: "m5", PostedAt: "2026-10-30T12:01:00Z", DiscordID: "8001"}
	for _, p := range []PanelRow{a, b} {
		if err := SavePanel(s, p); err != nil {
			t.Fatal(err)
		}
	}
	a.DiscordID, a.PostedAt = "8002", "2026-10-30T13:00:00Z"
	if err := SavePanel(s, a); err != nil {
		t.Fatal(err)
	}
	got, err := Panels(s)
	if err != nil || !reflect.DeepEqual(got, []PanelRow{a, b}) {
		t.Errorf("Panels = %+v, %v", got, err)
	}
	nulls := 0
	if err := SavePanel(s, PanelRow{Channel: "300", MessageID: "m9", PostedAt: "x"}); err != nil {
		t.Fatal(err)
	}
	if err := s.Query("SELECT count(*) FROM panel WHERE discord_message_id IS NULL", nil, func(r *Row) error { nulls = int(r.Int(0)); return nil }); err != nil || nulls != 1 {
		t.Errorf("%d rows with a null Discord id, %v", nulls, err)
	}
}
