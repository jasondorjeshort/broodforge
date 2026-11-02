// SPDX-License-Identifier: GPL-3.0-or-later

package session

import (
	"context"
	"errors"
	osexec "os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jasondorjeshort/broodforge/internal/git"
	"github.com/jasondorjeshort/broodforge/internal/git/gittest"
)

// landCall is one Config.Land call the stand-in saw.
type landCall struct {
	actor, claimed string
	dryRun         bool
}

// landRig is the review harness (reviewCanonical, standInRefresh) with channel 100's land list
// read at each call from land, which a test edits as `config reload` would, a Config.Land
// standing in for git.Handoff.Land, and a Config.Handoff standing in for git.Handoff.Run on a
// main project (runMain): each runs the gate on refs/sessions/<id>'s sha, as the land's step 3
// does after its refresh, and records the call. endErr, when set, is the end's refusal before
// the gate (land-canonical-diverged, a handoff refusal).
type landRig struct {
	h         *harness
	canonical string

	mu     sync.Mutex
	land   []string
	calls  []landCall
	ends   []git.EndLand
	endErr error
}

func newLandRig(t *testing.T, land []string) *landRig {
	t.Helper()
	r := &landRig{h: newHarness(t, "", true), land: land}
	r.canonical, _ = reviewCanonical(t, r.h)
	base := r.h.cfg.Channel
	r.h.cfg.Channel = func(id string) (Channel, bool) {
		ch, ok := base(id)
		r.mu.Lock()
		ch.Roles.Land = r.land
		r.mu.Unlock()
		return ch, ok
	}
	r.h.cfg.Land = func(ctx context.Context, id, actor string, opts git.LandOptions) (git.LandResult, error) {
		tip, err := r.tip(ctx, id)
		if err != nil {
			return git.LandResult{}, err
		}
		r.mu.Lock()
		r.calls = append(r.calls, landCall{actor, opts.Claimed, opts.DryRun})
		r.mu.Unlock()
		if err := opts.Gate(ctx, tip); err != nil {
			return git.LandResult{}, err
		}
		return git.LandResult{Outcome: git.Outcome{Outcome: git.OutcomeLanded, Commits: 1}, Sha: tip}, nil
	}
	r.h.cfg.Handoff = func(ctx context.Context, id string, land git.EndLand) (git.Outcome, error) {
		r.mu.Lock()
		r.ends = append(r.ends, land)
		endErr := r.endErr
		r.mu.Unlock()
		if endErr != nil {
			return git.Outcome{}, endErr
		}
		tip, err := r.tip(ctx, id)
		if err != nil {
			return git.Outcome{}, err
		}
		err = land.Gate(ctx, tip)
		var ge *git.Error
		switch {
		case err == nil:
			return git.Outcome{Outcome: git.OutcomeLanded, Mode: git.ModeMain, Commits: 1}, nil
		case errors.As(err, &ge) && git.FallsBack(ge.Code):
			branch := git.Branch("test", id)
			return git.Outcome{Outcome: git.OutcomePushed, Mode: git.ModePush, Commits: 1, Fallback: &git.LandRefusal{
				Code: ge.Code, Message: ge.Message, Line: git.FallbackLine("main", ge.Message, branch)}}, nil
		}
		return git.Outcome{}, err
	}
	withRefresh(r.h, standInRefresh(r.canonical, ""))
	return r
}

// tip is refs/sessions/<id> in the rig's canonical; it may run on the idle timer's goroutine,
// so it fails by its error, never t.
func (r *landRig) tip(ctx context.Context, id string) (string, error) {
	cmd := osexec.CommandContext(ctx, "git", "-C", r.canonical, "rev-parse", git.SessionRef(id))
	cmd.Env = gittest.Env(gittest.Driver)
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

func (r *landRig) setLand(land []string) {
	r.mu.Lock()
	r.land = land
	r.mu.Unlock()
}

// start opens id on reviewer-defer; reviewed runs its reviewer turn, the pin at the tip.
func (r *landRig) start(t *testing.T, req StartRequest) {
	t.Helper()
	req.Channel, req.As, req.Fixture = "100", "1001", reviewerDeferFixture
	if _, err := r.h.svc.Start(context.Background(), req); err != nil {
		t.Fatal(err)
	}
}

func (r *landRig) reviewed(t *testing.T, id string) {
	t.Helper()
	if _, err := r.h.svc.Submit(context.Background(), SendRequest{Session: id, As: "1001", Text: "review it"}); err != nil {
		t.Fatal(err)
	}
	waitTurn(t, r.h.svc, id, 1)
}

func (r *landRig) land1(as, roles string) error {
	_, err := r.h.svc.Land(context.Background(), LandRequest{ActRequest: ActRequest{Session: "s0001", As: as, Roles: roles}, Host: true})
	return err
}

// ended is id's session-ended facts.
func (r *landRig) ended(t *testing.T, id string) *EndInfo {
	t.Helper()
	return waitFor(t, r.h.svc, id, 0, isType(EventSessionEnded, 0)).End
}

func landCode(err error) string {
	var ge *git.Error
	if errors.As(err, &ge) {
		return ge.Code
	}
	return code(err)
}

// TestLandGate: before the reviewer ran the land is land-unreviewed; after the reviewer's
// deferred pin a role outside roles.land is land-not-allowed, a role in it lands, and no
// --roles leaves the role unchecked; a host land's actor is host and --as is claimed; every
// refusal is an action-refused row with its code; a commit after the review voids the fact.
func TestLandGate(t *testing.T) {
	r := newLandRig(t, []string{"trusted"})
	r.start(t, StartRequest{})
	err := r.land1("1001", "")
	if landCode(err) != git.CodeLandUnreviewed || !strings.Contains(err.Error(), "Not landed: review: none since ") {
		t.Fatalf("before the review: %v", err)
	}
	if n := count(t, r.h.st, `SELECT count(*) FROM audit_event WHERE kind = 'action-refused' AND actor = 'host'
    AND details LIKE '%"action":"land"%' AND details LIKE '%"claimed":"1001"%' AND details LIKE '%"reason":"land-unreviewed"%'`); n != 1 {
		t.Errorf("%d action-refused rows for the unreviewed land, want 1", n)
	}
	r.reviewed(t, "s0001")
	if err := r.land1("1001", "11"); landCode(err) != git.CodeLandNotAllowed {
		t.Errorf("role 11: %v", err)
	}
	if err := r.land1("1001", "10"); err != nil {
		t.Errorf("role 10: %v", err)
	}
	if err := r.land1("", ""); err != nil {
		t.Errorf("no --roles: %v", err)
	}
	want := []landCall{{"host", "1001", false}, {"host", "1001", false}, {"host", "1001", false}, {"host", "", false}}
	if len(r.calls) != len(want) || r.calls[2] != want[2] || r.calls[3] != want[3] {
		t.Errorf("calls %+v, want %+v", r.calls, want)
	}
	// A commit after the review, bundled: the tip moved and the fact is void.
	gittest.Commit(t, gittest.Driver, nil, r.canonical, "b.txt", "b", "after the review")
	if _, err := r.h.cfg.Refresh(context.Background(), git.Session{ID: "s0001"}); err != nil {
		t.Fatal(err)
	}
	if err := r.land1("", ""); landCode(err) != git.CodeLandUnreviewed {
		t.Errorf("after a later commit: %v", err)
	}
}

// TestLandGateWithoutALandList: a channel with no roles.land is land-not-allowed whoever asks,
// --roles or none, before the fact is read.
func TestLandGateWithoutALandList(t *testing.T) {
	r := newLandRig(t, nil)
	r.start(t, StartRequest{})
	for _, roles := range []string{"", "10"} {
		err := r.land1("", roles)
		if landCode(err) != git.CodeLandNotAllowed || !strings.HasSuffix(err.Error(), "Not landed: landing on main needs a land role in this channel.") {
			t.Errorf("roles %q: %v", roles, err)
		}
	}
}

// TestLandReloadLiftsNotAllowed: the gate reads the channel's roles.land as configured at each
// land, so the lift (`roles.land`, `config reload`) reaches an open session.
func TestLandReloadLiftsNotAllowed(t *testing.T) {
	r := newLandRig(t, nil)
	r.start(t, StartRequest{})
	r.reviewed(t, "s0001")
	if err := r.land1("1001", "10"); landCode(err) != git.CodeLandNotAllowed {
		t.Fatalf("before the reload: %v", err)
	}
	r.setLand([]string{"trusted"})
	if err := r.land1("1001", "10"); err != nil {
		t.Errorf("after the reload: %v", err)
	}
}

// TestLandHostAsIsNeverTheActor: a host land with an unregistered --as is judged and refused
// with host as the actor, the --as claimed; so is one on an ended session. No action-refused
// row names the --as value as its actor.
func TestLandHostAsIsNeverTheActor(t *testing.T) {
	r := newLandRig(t, []string{"trusted"})
	r.start(t, StartRequest{})
	if err := r.land1("9999", "10"); landCode(err) != git.CodeLandUnreviewed {
		t.Errorf("unreviewed: %v", err)
	}
	r.mu.Lock()
	calls := r.calls
	r.mu.Unlock()
	if len(calls) != 1 || calls[0] != (landCall{"host", "9999", false}) {
		t.Errorf("calls %+v", calls)
	}
	if _, err := r.h.svc.End(context.Background(), EndRequest{ActRequest: ActRequest{Session: "s0001"}, Host: true}); err != nil {
		t.Fatal(err)
	}
	if err := r.land1("9999", "10"); code(err) != CodeSessionClosed {
		t.Errorf("after the end: %v", err)
	}
	if n := count(t, r.h.st, `SELECT count(*) FROM audit_event WHERE kind = 'action-refused' AND actor != 'host'`); n != 0 {
		t.Errorf("%d action-refused rows name an actor other than host", n)
	}
}

// TestLandHandoffAtTheEnd: land-canonical-diverged at a main end keeps the session open; a land
// refused for the fact by a member with the role falls back, the summary carrying the
// companion's line and the end's land naming the member.
func TestLandHandoffAtTheEnd(t *testing.T) {
	r := newLandRig(t, []string{"trusted"})
	r.start(t, StartRequest{})
	ctx := context.Background()
	act := ActRequest{Session: "s0001", As: "1001", Roles: "10"}
	r.endErr = &git.Error{Code: git.CodeLandCanonicalDiverged, Message: "Not landed: the host's copy of main and origin have diverged; the owner reconciles it."}
	if _, err := r.h.svc.End(ctx, EndRequest{ActRequest: act}); landCode(err) != git.CodeLandCanonicalDiverged {
		t.Fatalf("diverged end: %v", err)
	}
	if d, _ := r.h.svc.Show("s0001"); d.State != StateWaitingForInput {
		t.Errorf("after a diverged main end: %s", d.State)
	}
	r.endErr = nil
	out, err := r.h.svc.End(ctx, EndRequest{ActRequest: act})
	if err != nil || out.Outcome != git.OutcomePushed || out.Fallback == nil || out.Fallback.Code != git.CodeLandUnreviewed {
		t.Fatalf("unreviewed end: %+v %v", out, err)
	}
	line := "Not landed on main: review: none since "
	if end := r.ended(t, "s0001"); end.Outcome != git.OutcomePushed || !strings.HasPrefix(end.NotLanded, line) ||
		!strings.HasSuffix(end.NotLanded, "then push again. Handed off as broodforge/test/s0001 instead.") {
		t.Errorf("ended %+v", end)
	}
	if last := r.ends[len(r.ends)-1]; last.Actor != "1001" || last.Claimed != "" {
		t.Errorf("the end's land %+v, want the member 1001", last)
	}
}

// TestLandIdleEndJudgesNoRoles: the idle end has no roles behind it, so a reviewed tip on a
// channel whose list is the session's role is still land-not-allowed and falls back to push,
// its land rows the instance's.
func TestLandIdleEndJudgesNoRoles(t *testing.T) {
	r := newLandRig(t, []string{"trusted"})
	r.start(t, StartRequest{IdleTimeoutSeconds: 60})
	r.reviewed(t, "s0001")
	r.h.clock.Advance(60 * time.Second)
	end := r.ended(t, "s0001")
	if end.Reason != "idle" || end.Outcome != git.OutcomePushed ||
		end.NotLanded != "Not landed on main: landing on main needs a land role in this channel. Handed off as broodforge/test/s0001 instead." {
		t.Errorf("idle end %+v", end)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.ends) != 1 || r.ends[0].Actor != ActorInstance {
		t.Errorf("the end's land %+v, want the instance's", r.ends)
	}
}

// TestLandHostEndJudgesNoRoles: `session end <id>` on the host without --roles is
// land-not-allowed and falls back; with --as and --roles the reviewed tip lands, the land rows
// host's with --as claimed.
func TestLandHostEndJudgesNoRoles(t *testing.T) {
	r := newLandRig(t, []string{"trusted"})
	ctx := context.Background()
	r.start(t, StartRequest{})
	r.reviewed(t, "s0001")
	out, err := r.h.svc.End(ctx, EndRequest{ActRequest: ActRequest{Session: "s0001"}, Host: true})
	if err != nil || out.Fallback == nil || out.Fallback.Code != git.CodeLandNotAllowed || out.Outcome != git.OutcomePushed {
		t.Fatalf("host end, no roles: %+v %v", out, err)
	}
	r.start(t, StartRequest{})
	r.reviewed(t, "s0002")
	out, err = r.h.svc.End(ctx, EndRequest{ActRequest: ActRequest{Session: "s0002", As: "1001", Roles: "10"}, Host: true})
	if err != nil || out.Outcome != git.OutcomeLanded || out.Fallback != nil {
		t.Fatalf("host end with the role: %+v %v", out, err)
	}
	if last := r.ends[len(r.ends)-1]; last.Actor != ActorHost || last.Claimed != "1001" {
		t.Errorf("the end's land %+v, want host claiming 1001", last)
	}
}
