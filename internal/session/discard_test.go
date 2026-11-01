// SPDX-License-Identifier: GPL-3.0-or-later

package session

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jasondorjeshort/broodforge/internal/git"
)

// TestEndDiscard: 04-ux-session-end.md. DiscardCount is the Pending count for the driver and a
// roles refusal for anyone else; End with Discard runs Config.Discard for the actor, not the
// handoff, with no Ending line, and ends the session `discarded` with the ref in its summary.
// A refused discard keeps the session open, as a refused handoff does.
func TestEndDiscard(t *testing.T) {
	h := newHarness(t, "", true)
	var discarded []string
	refuse := true
	h.cfg.Discard = func(_ context.Context, id, actor string) (git.Outcome, error) {
		if refuse {
			refuse = false
			return git.Outcome{}, &git.Error{Code: git.CodeRefusedWIP, Message: "the tree's HEAD is not on b"}
		}
		discarded = append(discarded, id+" "+actor)
		return git.Outcome{Outcome: git.OutcomeDiscarded, Commits: 2, Ref: git.AbortedRef(id), WIP: "abc"}, nil
	}
	h.cfg.Pending = func(context.Context, string) (*int, error) { n := 2; return &n, nil }
	h.handoff = func(string) (git.Outcome, error) { t.Error("the handoff ran on a discard"); return git.Outcome{}, nil }
	h.svc = New(h.cfg)
	ctx := context.Background()
	if _, err := h.svc.Start(ctx, StartRequest{Channel: "100", As: "1001"}); err != nil {
		t.Fatal(err)
	}
	act := ActRequest{Session: "s0001", As: "1001", Roles: "10"}
	if n, err := h.svc.DiscardCount(ctx, act); err != nil || n != 2 {
		t.Errorf("DiscardCount = %d, %v", n, err)
	}
	if _, err := h.svc.DiscardCount(ctx, ActRequest{Session: "s0001", As: "1002", Roles: "10"}); code(err) != CodeRefused {
		t.Errorf("a non-driver's discard: %v", err)
	}
	var ge *git.Error
	if _, err := h.svc.End(ctx, EndRequest{ActRequest: act, Discard: true}); !errors.As(err, &ge) || ge.Code != git.CodeRefusedWIP {
		t.Fatalf("refused discard: %v", err)
	}
	if d, _ := h.svc.Show("s0001"); d.State != StateWaitingForInput {
		t.Errorf("after a refused discard: %s", d.State)
	}
	out, err := h.svc.End(ctx, EndRequest{ActRequest: act, Discard: true})
	if err != nil || out.Outcome != git.OutcomeDiscarded || len(discarded) != 1 || discarded[0] != "s0001 1001" {
		t.Fatalf("discard: %+v, %v, %q", out, err, discarded)
	}
	evs, _ := h.svc.Events("s0001", 0)
	var end *EndInfo
	for _, ev := range evs {
		if ev.Type == EventSessionEnding {
			t.Error("a discard published session-ending")
		}
		if ev.Type == EventSessionEnded {
			end = ev.End
		}
	}
	if end == nil || end.Outcome != git.OutcomeDiscarded || end.Commits != 2 || end.Ref != "refs/aborted/s0001" || end.WIP != "abc" {
		t.Errorf("ended %+v", end)
	}
	if n := count(t, h.st, `SELECT count(*) FROM audit_event WHERE kind = 'session-ended'
    AND details LIKE '%"outcome":"discarded"%' AND details LIKE '%"wip":"abc"%'`); n != 1 {
		t.Error("no session-ended row with outcome discarded and the wip")
	}
	if _, err := h.svc.DiscardCount(ctx, act); code(err) != CodeSessionClosed {
		t.Errorf("DiscardCount after the end: %v", err)
	}
}

// TestHeldRetry: a refusal on a held session names a retry that keeps the tree's work: the WIP
// commit's and the discard's are session end, never worktree handoff.
func TestHeldRetry(t *testing.T) {
	for _, c := range []struct {
		code    string
		discard bool
		want    string
	}{
		{git.CodeRefusedAttribution, false, "retry with broodforge worktree handoff s0001."},
		{git.CodeRefusedWIP, false, "retry with broodforge session end s0001."},
		{git.CodeRefusedWIP, true, "retry with broodforge session end s0001 --discard."},
		{git.CodeRefusedBundle, true, "retry with broodforge session end s0001 --discard."},
	} {
		if got := heldRetry("s0001", c.code, c.discard); !strings.HasSuffix(got, c.want) {
			t.Errorf("heldRetry(%s, %v) = %q", c.code, c.discard, got)
		}
	}
}
