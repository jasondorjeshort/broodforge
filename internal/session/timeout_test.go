// SPDX-License-Identifier: GPL-3.0-or-later

package session

import (
	"context"
	"testing"
	"time"
)

func TestParseIdleTimeout(t *testing.T) {
	for in, want := range map[string]time.Duration{
		"never": 0, "Off": 0, "0": 0, "30": 30 * time.Minute, "90m": 90 * time.Minute, "2h": 2 * time.Hour, " 1h30m ": 90 * time.Minute,
	} {
		if got, ok := ParseIdleTimeout(in); !ok || got != want {
			t.Errorf("%q: %v %v, want %v", in, got, ok, want)
		}
	}
	for _, bad := range []string{"", "-5", "30s", "soon", "-1h"} {
		if _, ok := ParseIdleTimeout(bad); ok {
			t.Errorf("%q parsed", bad)
		}
	}
}

// TestSetIdleTimeout: the driver's switch takes effect now: never stops the clock, a new value
// restarts it; it is audited with from and to; a non-driver and a bad value are refused.
func TestSetIdleTimeout(t *testing.T) {
	h := newHarness(t, "", true)
	ctx := context.Background()
	if _, err := h.svc.Start(ctx, StartRequest{Channel: "100", As: "1001", IdleTimeoutSeconds: 3600}); err != nil {
		t.Fatal(err)
	}
	act := ActRequest{Session: "s0001", As: "1001"}
	if _, err := h.svc.SetIdleTimeout(TimeoutRequest{ActRequest: ActRequest{Session: "s0001", As: "1002"}, Timeout: "never"}); code(err) != CodeRefused {
		t.Errorf("timeout by a non-driver: %v", err)
	}
	if _, err := h.svc.SetIdleTimeout(TimeoutRequest{ActRequest: act, Timeout: "soon"}); code(err) != CodeBadArgs {
		t.Errorf("timeout soon: %v", err)
	}
	e, err := h.svc.SetIdleTimeout(TimeoutRequest{ActRequest: act, Timeout: "never"})
	if err != nil || e.Type != EventTimeoutChanged || e.Result != "never" || e.Summary != "Never ends for inactivity." {
		t.Fatalf("never: %+v %v", e, err)
	}
	h.clock.Advance(3 * time.Hour)
	if d, _ := h.svc.Show("s0001"); d.State != StateWaitingForInput {
		t.Fatalf("ended with no timeout: %s", d.State)
	}
	e, err = h.svc.SetIdleTimeout(TimeoutRequest{ActRequest: act, Timeout: "30"})
	if err != nil || e.Result != "30 minutes" || e.Summary != "Ends after 30 minutes without activity." {
		t.Fatalf("30: %+v %v", e, err)
	}
	h.clock.Advance(29 * time.Minute)
	if d, _ := h.svc.Show("s0001"); d.State != StateWaitingForInput {
		t.Fatalf("ended early: %s", d.State)
	}
	h.clock.Advance(time.Minute)
	if e := waitFor(t, h.svc, "s0001", 0, isType(EventSessionEnded, 0)); e.Summary != "Session s0001 ended: no activity for 30 minutes." {
		t.Errorf("ended %+v", e)
	}
	if n := count(t, h.st, `SELECT count(*) FROM audit_event WHERE kind = 'timeout-changed' AND details = '{"from":"60 minutes","to":"never"}'`); n != 1 {
		t.Error("no timeout-changed audit event with from and to")
	}
}
