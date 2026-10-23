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
		"1": time.Minute, "1h30m15s": 90*time.Minute + 15*time.Second, "525600": MaxTimeout, "8760h": MaxTimeout,
	} {
		if got, capped, ok := ParseIdleTimeout(in); !ok || capped || got != want {
			t.Errorf("%q: %v %v %v, want %v uncapped", in, got, capped, ok, want)
		}
	}
}

func TestParseIdleTimeoutCapsAboveAYear(t *testing.T) {
	for _, in := range []string{"525601", "307445735", "153722868", "9223372036854775807", "99999999999999999999", "8761h"} {
		if got, capped, ok := ParseIdleTimeout(in); !ok || !capped || got != MaxTimeout {
			t.Errorf("%q: %v %v %v, want %v capped", in, got, capped, ok, MaxTimeout)
		}
	}
}

func TestParseIdleTimeoutRefusesWhatIsNoTimeout(t *testing.T) {
	// 9999999h overflows what Go reads as a duration.
	for _, bad := range []string{"", "-5", "30s", "soon", "-1h", "9999999h"} {
		if _, _, ok := ParseIdleTimeout(bad); ok {
			t.Errorf("%q parsed", bad)
		}
	}
}

func TestSecondsTimeoutEdge(t *testing.T) {
	for in, want := range map[int]time.Duration{31536000: MaxTimeout, 31536001: MaxTimeout, 90: 90 * time.Second, 9000000000000: MaxTimeout} {
		if got := secondsTimeout(in); got != want {
			t.Errorf("%d seconds: %v, want %v", in, got, want)
		}
	}
}

// TestNoPromptTimeout: with the channel's prompt timeout 0 ("never"), a deferred call still
// waits past a day for its answer, and shows a real deadline a year out.
func TestNoPromptTimeout(t *testing.T) {
	h := newHarness(t, "", true)
	ctx := context.Background()
	if _, err := h.svc.Start(ctx, StartRequest{Channel: "100", As: "1001", Mode: "prompt", Fixture: "defer"}); err != nil {
		t.Fatal(err)
	}
	sub, err := h.svc.Submit(ctx, SendRequest{Session: "s0001", As: "1001", Text: "write hi.txt"})
	if err != nil {
		t.Fatal(err)
	}
	waitFor(t, h.svc, "s0001", sub.Seq, isType(EventWaitingForAnswer, 1))
	d, err := h.svc.Show("s0001")
	if err != nil || d.Pending == nil {
		t.Fatalf("show: %+v, %v", d, err)
	}
	start, perr := time.Parse(time.RFC3339, d.Pending.Since)
	until, uerr := time.Parse(time.RFC3339, d.Pending.Until)
	if perr != nil || uerr != nil || until.Sub(start) != MaxTimeout {
		t.Fatalf("deadline %q since %q, want a year apart", d.Pending.Until, d.Pending.Since)
	}
	h.clock.Advance(24 * time.Hour)
	if d, err := h.svc.Show("s0001"); err != nil || d.State != StateWaitingForAnswer {
		t.Fatalf("show a day later: %+v, %v", d, err)
	}
}

// TestStartCapsATimeoutOverrideAtAYear: 31536000 seconds is a year as given; one more is capped.
func TestStartCapsATimeoutOverrideAtAYear(t *testing.T) {
	for _, sec := range []int{31536000, 31536001} {
		h := newHarness(t, "", true)
		if _, err := h.svc.Start(context.Background(), StartRequest{Channel: "100", As: "1001", Mode: "prompt", Fixture: "defer", IdleTimeoutSeconds: sec, PromptTimeoutSeconds: sec}); err != nil {
			t.Fatalf("%d seconds: %v", sec, err)
		}
		e := waitFor(t, h.svc, "s0001", 0, isType(EventSessionStarted, 0))
		if e.Start == nil || e.Start.IdleTimeout != "525600 minutes" {
			t.Errorf("%d seconds: idle timeout %+v", sec, e.Start)
		}
		sub, err := h.svc.Submit(context.Background(), SendRequest{Session: "s0001", As: "1001", Text: "write hi.txt"})
		if err != nil {
			t.Fatal(err)
		}
		waitFor(t, h.svc, "s0001", sub.Seq, isType(EventWaitingForAnswer, 1))
		d, err := h.svc.Show("s0001")
		if err != nil || d.Pending == nil {
			t.Fatalf("%d seconds: show %+v %v", sec, d, err)
		}
		since, _ := time.Parse(time.RFC3339, d.Pending.Since)
		until, _ := time.Parse(time.RFC3339, d.Pending.Until)
		if until.Sub(since) != MaxTimeout {
			t.Errorf("%d seconds: prompt deadline %v after the call, want a year", sec, until.Sub(since))
		}
	}
}

func TestStartRefusesANegativeTimeoutOverride(t *testing.T) {
	h := newHarness(t, "", true)
	if _, err := h.svc.Start(context.Background(), StartRequest{Channel: "100", As: "1001", IdleTimeoutSeconds: -1}); code(err) != CodeBadArgs {
		t.Errorf("negative override: %v", err)
	}
}

func TestSetIdleTimeoutAboveAYearIsCappedAndSaid(t *testing.T) {
	h := newHarness(t, "", true)
	if _, err := h.svc.Start(context.Background(), StartRequest{Channel: "100", As: "1001"}); err != nil {
		t.Fatal(err)
	}
	act := ActRequest{Session: "s0001", As: "1001"}
	for in, capped := range map[string]bool{"525600": false, "525601": true} {
		e, err := h.svc.SetIdleTimeout(TimeoutRequest{ActRequest: act, Timeout: in})
		want := "Ends after 525600 minutes without activity."
		if capped {
			want += " Capped at a year."
		}
		if err != nil || e.Summary != want {
			t.Errorf("%s: %+v %v, want %q", in, e, err, want)
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
