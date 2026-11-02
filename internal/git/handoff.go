// SPDX-License-Identifier: GPL-3.0-or-later

package git

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jasondorjeshort/broodforge/internal/audit"
	"github.com/jasondorjeshort/broodforge/internal/spawn"
)

// Handoff modes (the project's handoff) and outcomes.
const (
	ModePush        = "push"
	ModePullRequest = "pull-request"
	ModeMain        = "main" // a land on the default branch, falling back to Handoff.Fallback (ADR 0028)

	OutcomeLanded      = "landed"       // main: the session's commits are on the default branch
	OutcomeNone        = "none"         // no commits: the tree was removed
	OutcomePushed      = "pushed"       // push: the branch is on the remote
	OutcomePullRequest = "pull-request" // pull-request: pushed and the request opened
	OutcomeDryRun      = "dry-run"      // --dry-run: checked, nothing sent
	OutcomeDiscarded   = "discarded"    // Discard: nothing sent, the commits kept on refs/aborted/<id>
)

// WIPSubject is the subject of the commit the end makes of a dirty tree (04-ux-session-end.md).
const WIPSubject = "WIP: uncommitted changes committed at end"

// CredentialEnv carries the git credential to the askpass helper, set only on the push and
// request processes and never kept (ADR 0022 companion, Handoff step 4).
const CredentialEnv = "BROODFORGE_GIT_CREDENTIAL"

// Collected is steps 1 to 3 of the handoff: the base after the fetch, the session's own
// commits since it (the fetched commits the default branch does not hold), and their check.
type Collected struct {
	Base    string
	Commits int
	Checks  []CommitCheck
}

// Failing is the commits that did not pass.
func (c Collected) Failing() []CommitCheck {
	var out []CommitCheck
	for _, ch := range c.Checks {
		if !ch.OK() {
			out = append(out, ch)
		}
	}
	return out
}

// BundleStep is step 1 alone: the backend's worktree-bundle of the session's commits since
// the base (`worktree bundle <id>`).
func BundleStep(ctx context.Context, b spawn.Backend, s Session) (spawn.Bundle, string, error) {
	if err := s.check(); err != nil {
		return spawn.Bundle{}, "", err
	}
	if _, err := run(ctx, CodeAlreadyHandedOff, s.Canonical, nil, "rev-parse", "--verify", "--quiet", SessionRef(s.ID)); err != nil {
		return spawn.Bundle{}, "", &Error{Code: CodeAlreadyHandedOff,
			Message: s.Canonical + " holds no " + SessionRef(s.ID) + ": the session was handed off or never started"}
	}
	base, err := Base(ctx, s)
	if err != nil {
		return spawn.Bundle{}, "", err
	}
	bundle, err := b.WorktreeBundle(ctx, s.Worktree(base))
	if err != nil {
		var se *spawn.Error
		if errors.As(err, &se) {
			return spawn.Bundle{}, base, &Error{Code: CodeRefusedBundle, Message: se.Message}
		}
		return spawn.Bundle{}, base, err
	}
	return bundle, base, nil
}

// Commits is the session's commit count over its base, read through the backend from its tree
// (spawn's WorktreeCommits) without writing a bundle: a held session's notice after a restart
// (ADR 0026). A canonical without refs/sessions/<id> is already-handed-off, as BundleStep's; a
// canonical that cannot be read is session-ref (SessionTip's).
func Commits(ctx context.Context, b spawn.Backend, s Session) (int, error) {
	_, found, err := SessionTip(ctx, s)
	if err != nil {
		return 0, err
	}
	if !found {
		return 0, &Error{Code: CodeAlreadyHandedOff,
			Message: s.Canonical + " holds no " + SessionRef(s.ID) + ": the session was handed off or never started"}
	}
	base, err := Base(ctx, s)
	if err != nil {
		return 0, err
	}
	n, err := b.WorktreeCommits(s.Worktree(base))
	if err != nil {
		var se *spawn.Error
		if errors.As(err, &se) {
			return 0, &Error{Code: CodeRefusedBundle, Message: se.Message}
		}
		return 0, err
	}
	return n, nil
}

// Pending is the commits an end would hand off or a discard keep, read before it runs: Commits,
// and one more when the tree has uncommitted changes, which the end commits first
// (Handoff.CommitDirty, Discard). The tree is read through the backend (spawn's WorktreeDirty).
func Pending(ctx context.Context, b spawn.Backend, s Session) (int, error) {
	n, err := Commits(ctx, b, s)
	if err != nil {
		return 0, err
	}
	base, err := Base(ctx, s)
	if err != nil {
		return 0, err
	}
	dirty, err := b.WorktreeDirty(s.Worktree(base))
	if err != nil {
		var se *spawn.Error
		if errors.As(err, &se) {
			return 0, &Error{Code: CodeRefusedWIP, Message: se.Message}
		}
		return 0, err
	}
	if dirty {
		n++
	}
	return n, nil
}

// Refresh is steps 1 and 2 of the handoff (ADR 0022 companion, Handoff at session end):
// worktree-bundle, then in the canonical `git bundle verify` and `git fetch <bundle>
// +<session branch>:refs/sessions/<id>`, returning the commit the ref then holds. The fetch is
// forced because the driver may have had commits amended after a refused check; the ref is the
// bot's own. No commits since the base sets the ref back to the base, which is returned. The
// bot never opens the session's tree. `worktree check` and the handoff run it; the reviewer pin
// runs the same body (PinReview). It holds the session's bundle lock throughout (ADR 0028
// Decision 1), waiting no longer than ctx.
func Refresh(ctx context.Context, b spawn.Backend, s Session) (sha string, err error) {
	unlock, err := LockBundle(ctx, s.ID)
	if err != nil {
		return "", err
	}
	defer unlock()
	bundle, base, err := BundleStep(ctx, b, s)
	if err != nil {
		return "", err
	}
	return fetchBundle(ctx, s, bundle, base)
}

// Collect runs steps 1 to 3 (ADR 0022 companion, Handoff at session end): Refresh, then Check
// of the session's commits over the base.
func Collect(ctx context.Context, b spawn.Backend, s Session, a Attribution) (Collected, error) {
	unlock, err := LockBundle(ctx, s.ID)
	if err != nil {
		return Collected{}, err
	}
	defer unlock()
	c, _, err := collectLocked(ctx, b, s, a)
	return c, err
}

// collectLocked is Collect with the session's bundle lock held by the caller (the land holds it
// throughout, ADR 0028 Decision 1); it also returns the sha refs/sessions/<id> holds after the
// fetch, the session's tip.
func collectLocked(ctx context.Context, b spawn.Backend, s Session, a Attribution) (Collected, string, error) {
	bundle, base, err := BundleStep(ctx, b, s)
	if err != nil {
		return Collected{}, "", err
	}
	sha, err := fetchBundle(ctx, s, bundle, base)
	if err != nil {
		return Collected{}, "", err
	}
	// The check's base is taken after the fetch: a session that merged the default branch's
	// newer commits forks from it later, and those commits are not the session's.
	var c Collected
	if c.Base, err = Base(ctx, s); err != nil {
		return c, sha, err
	}
	if sha == c.Base {
		return c, sha, nil // no commits since the base
	}
	if c.Checks, err = Check(ctx, s, c.Base, a); err != nil {
		return c, sha, err
	}
	c.Commits = len(c.Checks)
	return c, sha, nil
}

// vet is what a handoff and a land refuse on after Collect: a commit failing the attribution
// check, then, with every commit passing it, a hit of the push scan.
func (h *Handoff) vet(ctx context.Context, c Collected) error {
	if failing := c.Failing(); len(failing) > 0 {
		return &Error{Code: CodeRefusedAttribution, Commits: failing,
			Message: strconv.Itoa(len(failing)) + " of " + strconv.Itoa(len(c.Checks)) + " commits fail the attribution check"}
	}
	if c.Commits == 0 {
		return nil
	}
	hits, err := ScanPrivacy(ctx, h.Session, c.Base, h.Privacy.WithSession(h.Attribution))
	if err == nil && len(hits) > 0 {
		err = PrivacyRefusal(hits, c.Commits)
	}
	return err
}

// Outcome is what a handoff did, or with --dry-run would do.
type Outcome struct {
	Outcome string       `json:"outcome"`
	Mode    string       `json:"mode"`
	Remote  string       `json:"remote"`
	Refspec string       `json:"refspec,omitempty"`
	Commits int          `json:"commits"`
	URL     *string      `json:"url"`
	Request *PullRequest `json:"request"`
	// Removed is false when the tree outlived a done handoff; RemoveError says why.
	Removed     bool   `json:"removed"`
	RemoveError string `json:"removeError,omitempty"`
	// WIP is the commit the end made of the tree's uncommitted changes (Handoff.CommitDirty,
	// Discard), "" when the tree was clean; Ref is the ref a discard kept the commits on.
	WIP string `json:"wip,omitempty"`
	Ref string `json:"ref,omitempty"`
	// Log is the commits handed off, oldest first, for the end-of-session summary (04, Ending a
	// session); `worktree handoff` does not print it.
	Log []CommitLine `json:"-"`
	// Fallback is a main project's land refusal when Run handed off as Handoff.Fallback instead
	// (land-not-fast-forward, land-rejected; ADR 0028 Decision 1): the result carries both.
	Fallback *LandRefusal `json:"fallback,omitempty"`
}

// LandRefusal is a land's refusal as an outcome carries it: its code and text.
type LandRefusal struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	// Line is the companion's fallback line (FallbackLine).
	Line string `json:"line"`
}

// CommitLine is one commit as the summary lists it: its hash and its subject.
type CommitLine struct {
	Hash    string `json:"hash"`
	Subject string `json:"subject"`
}

// Handoff is one session's handoff: the five steps of the ADR 0022 companion.
type Handoff struct {
	Backend spawn.Backend
	Session Session
	Mode    string // ModePush, ModePullRequest or ModeMain
	// Fallback is a main project's handoffFallback, ModePush ("" reads as it) or
	// ModePullRequest: what Run hands off as when the end's land refuses anything but
	// land-canonical-diverged (ADR 0028 Decision 1, Amendments 2026-11-02).
	Fallback string
	// Gate is Run's gate for a main land (LandOptions.Gate); nil refuses land-not-allowed.
	Gate LandGate
	// LandActor is the actor of the end's `land` audit rows (the ender: a Discord user id, host,
	// or instance for the idle end), "" for Actor; Claimed is a host end's --as (LandOptions).
	LandActor, Claimed string
	// Opened is the web URL of the pull request a land already opened for this session, "" for
	// none: a later land, or the end, pushes the branch and opens no second request.
	Opened      string
	Attribution Attribution
	// Privacy is the push scan's lists (ADR 0025 rule 7); the scan adds the Attribution's
	// identities (WithSession).
	Privacy Privacy
	// CredentialID is the project's git credential; "" sends none, and a remote that asks for
	// one gets no answer from the helper.
	CredentialID string
	Secret       func(id string) ([]byte, error)
	Forge        PullRequester // nil: GitHub{}
	API          string        // "": GitHubAPI
	Title, Body  string        // the pull request's
	// Record writes one audit event (push, pull-request-opened, handoff-refused, session-discarded); nil records nothing.
	Record func(audit.Event) error
	Actor  string
	// CommitDirty is a session's end (04-ux-session-end.md): before the bundle, the tree's
	// uncommitted changes become one WIP commit through the backend (commitDirty), so the end
	// loses nothing. `worktree handoff` leaves it false.
	CommitDirty bool
}

// Run hands the session off. Refusals keep the tree; a done push or request removes it through
// worktree-remove and deletes refs/sessions/<id> (R-35). dryRun stops after the check: the
// attribution check, then, with every commit passing it, the push scan. On a main project it is
// the land (Land, with Gate) followed by the removal; a land refused land-not-fast-forward or
// land-rejected hands off as Fallback instead, the outcome carrying the refusal (ADR 0028).
func (h *Handoff) Run(ctx context.Context, dryRun bool) (Outcome, error) {
	if h.Mode != ModePush && h.Mode != ModePullRequest && h.Mode != ModeMain {
		return Outcome{}, &Error{Code: CodeBadSession, Message: "handoff is " + strconv.Quote(h.Mode) + ", want push, pull-request or main"}
	}
	var wip string
	if h.CommitDirty && !dryRun {
		var err error
		if wip, err = h.commitDirty(ctx); err != nil {
			h.refused(err)
			return Outcome{}, err
		}
	}
	var out Outcome
	var err error
	if h.Mode == ModeMain {
		out, err = h.runMain(ctx, dryRun)
	} else {
		out, err = h.run(ctx, dryRun)
	}
	out.WIP = wip
	return out, err
}

// runMain is Run on a main project: the land, then step 5's removal; on a land refusal the
// fallback (FallsBack), the outcome carrying the first refusal met in the land's order (role,
// fact, fast-forward, push). land-canonical-diverged and the handoff's own refusals are final.
func (h *Handoff) runMain(ctx context.Context, dryRun bool) (Outcome, error) {
	res, err := h.Land(ctx, LandOptions{DryRun: dryRun, Gate: h.Gate, Claimed: h.Claimed})
	if err == nil {
		out := res.Outcome
		if !dryRun {
			h.remove(ctx, &out)
		}
		return out, nil
	}
	var ge *Error
	if !errors.As(err, &ge) || !FallsBack(ge.Code) {
		return Outcome{}, err
	}
	fb := *h
	fb.Mode = h.Fallback
	if fb.Mode == "" {
		fb.Mode = ModePush
	}
	if fb.Mode != ModePush && fb.Mode != ModePullRequest {
		return Outcome{}, &Error{Code: CodeBadSession, Message: "handoffFallback is " + strconv.Quote(fb.Mode) + ", want push or pull-request"}
	}
	out, ferr := fb.run(ctx, dryRun)
	out.Fallback = &LandRefusal{Code: ge.Code, Message: ge.Message,
		Line: FallbackLine(h.Session.DefaultBranch, ge.Message, h.Session.Branch())}
	return out, ferr
}

// FallsBack reports whether a main end's land refused with code hands off as handoffFallback
// instead (ADR 0028 Amendments 2026-11-02): every land refusal but land-canonical-diverged,
// which names a fault of the host's canonical a fallback push would publish.
func FallsBack(code string) bool {
	switch code {
	case CodeLandNotAllowed, CodeLandUnreviewed, CodeLandNotFastForward, CodeLandRejected:
		return true
	}
	return false
}

// FallbackLine is the companion's line for an end that fell back (wording: owner): `Not landed
// on <default>: <refusal text>. Handed off as <branch> instead.`, the refusal's own `Not landed: `
// and closing period dropped so the line reads once.
func FallbackLine(defaultBranch, refusal, branch string) string {
	text := strings.TrimSuffix(strings.TrimPrefix(refusal, "Not landed: "), ".")
	return "Not landed on " + defaultBranch + ": " + text + ". Handed off as " + branch + " instead."
}

// run is Run after the WIP commit: the five steps of the ADR 0022 companion.
func (h *Handoff) run(ctx context.Context, dryRun bool) (Outcome, error) {
	s := h.Session
	c, err := Collect(ctx, h.Backend, s, h.Attribution)
	if err == nil {
		err = h.vet(ctx, c)
	}
	if err != nil {
		h.refused(err)
		return Outcome{}, err
	}
	refspec := SessionRef(s.ID) + ":refs/heads/" + s.Branch()
	out := Outcome{Mode: h.Mode, Remote: s.Remote, Refspec: refspec, Commits: c.Commits}
	for _, ch := range c.Checks {
		out.Log = append(out.Log, CommitLine{Hash: ch.Hash, Subject: ch.Subject})
	}
	if c.Commits == 0 {
		out.Outcome, out.Refspec = OutcomeNone, ""
		if dryRun {
			out.Outcome = OutcomeDryRun
			return out, nil
		}
		h.remove(ctx, &out)
		return out, nil
	}
	if h.Mode == ModePullRequest && h.Opened == "" {
		pr, err := h.request(ctx)
		if err != nil {
			return out, err
		}
		out.Request = &pr
	}
	if dryRun {
		out.Outcome = OutcomeDryRun
		return out, nil
	}
	token, err := h.token()
	if err != nil {
		return out, err
	}
	if err := h.push(ctx, out, token); err != nil {
		return out, err
	}
	out.Outcome = OutcomePushed
	switch {
	case out.Request != nil:
		url, err := h.open(ctx, *out.Request, token)
		if err != nil {
			return out, err
		}
		out.Outcome, out.URL = OutcomePullRequest, &url
	case h.Mode == ModePullRequest:
		url := h.Opened // a land opened the request: the push moved its branch
		out.Outcome, out.URL = OutcomePullRequest, &url
	}
	h.remove(ctx, &out)
	return out, nil
}

// commitDirty makes the end's WIP commit of the tree's uncommitted changes through the backend
// (spawn's WorktreeWIP; the bot's git never opens the tree, ADR 0022) and returns its hash, ""
// for a clean tree. Author and committer are the driver and the trailers those the check
// requires of a commit made now (Attribution.Required), so the commit passes the check as the
// session's own commits do (03-architecture-git.md, Attribution). A canonical without
// refs/sessions/<id> is already-handed-off, as BundleStep's.
func (h *Handoff) commitDirty(ctx context.Context) (string, error) {
	s := h.Session
	if _, found, err := SessionTip(ctx, s); err != nil {
		return "", err
	} else if !found {
		return "", &Error{Code: CodeAlreadyHandedOff,
			Message: s.Canonical + " holds no " + SessionRef(s.ID) + ": the session was handed off or never started"}
	}
	base, err := Base(ctx, s)
	if err != nil {
		return "", err
	}
	msg := WIPSubject + "\n"
	if lines := h.Attribution.Required(time.Now()); len(lines) > 0 {
		msg += "\n" + strings.Join(lines, "\n") + "\n"
	}
	sha, err := h.Backend.WorktreeWIP(s.Worktree(base), spawn.WIP{Name: h.Attribution.Driver.Name,
		Email: h.Attribution.Driver.Email, Message: msg})
	if err != nil {
		var se *spawn.Error
		if errors.As(err, &se) {
			return "", &Error{Code: CodeRefusedWIP, Message: se.Message}
		}
		return "", err
	}
	return sha, nil
}

// Discard is the end without a handoff (`session end <id> --discard`, 04-ux-session-end.md):
// the WIP commit of a dirty tree, then Collect (bundle, verify, fetch into refs/sessions/<id>,
// the check, whose result decides nothing here), then refs/aborted/<id> set to the session's
// tip when it has commits, the `session-discarded` audit row, and step 5's removal of the tree
// and refs/sessions/<id>. Nothing is pushed; the commits stay recoverable in the canonical. A
// refusal keeps the tree, as Run's do.
func (h *Handoff) Discard(ctx context.Context) (Outcome, error) {
	s := h.Session
	wip, err := h.commitDirty(ctx)
	if err != nil {
		h.refused(err)
		return Outcome{}, err
	}
	c, err := Collect(ctx, h.Backend, s, h.Attribution)
	if err != nil {
		h.refused(err)
		return Outcome{WIP: wip}, err
	}
	out := Outcome{Outcome: OutcomeDiscarded, Mode: h.Mode, Remote: s.Remote, Commits: c.Commits, WIP: wip}
	for _, ch := range c.Checks {
		out.Log = append(out.Log, CommitLine{Hash: ch.Hash, Subject: ch.Subject})
	}
	details := map[string]string{"commits": strconv.Itoa(c.Commits)}
	if c.Commits > 0 {
		ref := AbortedRef(s.ID)
		if _, err := run(ctx, CodeRefusedAbortedRef, s.Canonical, nil, "update-ref", ref, SessionRef(s.ID)); err != nil {
			// A refusal: the tree and refs/sessions/<id> are kept, the session stays open.
			h.refused(err)
			return Outcome{WIP: wip}, err
		}
		out.Ref, out.Refspec = ref, SessionRef(s.ID)+":"+ref
		details["ref"] = ref
	}
	if wip != "" {
		details["wip"] = wip
	}
	h.record(audit.KindSessionDiscarded, audit.PhaseNone, details)
	h.remove(ctx, &out)
	return out, nil
}

// PushEnv is the push's own variables: the askpass helper, the credential for it when there
// is one, and git's terminal prompt off.
func PushEnv(askpass string, token []byte) []spawn.Var {
	vars := []spawn.Var{{Name: "GIT_ASKPASS", Value: askpass}}
	if len(token) > 0 {
		vars = append(vars, spawn.Var{Name: CredentialEnv, Value: string(token)})
	}
	return vars
}

// push is step 4's `git push <remote> refs/sessions/<id>:refs/heads/<branch>` from the
// canonical, with the account's credential helpers cleared so only the askpass helper answers.
func (h *Handoff) push(ctx context.Context, out Outcome, token []byte) error {
	details := map[string]string{"remote": out.Remote, "refspec": out.Refspec, "commits": strconv.Itoa(out.Commits)}
	h.record(audit.KindPush, audit.PhaseAttempted, details)
	_, err := run(ctx, CodeFailedPush, h.Session.Canonical, PushEnv(h.Backend.Askpass(), token),
		"-c", "credential.helper=", "push", "--quiet", out.Remote, out.Refspec)
	if err != nil {
		details["error"] = messageOf(err)
		h.record(audit.KindPush, audit.PhaseFailed, details)
		return err
	}
	h.record(audit.KindPush, audit.PhaseDone, details)
	return nil
}

// request builds the pull request from the remote's URL; only github remotes have one.
func (h *Handoff) request(ctx context.Context) (PullRequest, error) {
	s := h.Session
	url, err := run(ctx, CodeFailedPullRequest, s.Canonical, nil, "remote", "get-url", s.Remote)
	if err != nil {
		return PullRequest{}, err
	}
	owner, repo, ok := GitHubRepo(url)
	if !ok {
		return PullRequest{}, &Error{Code: CodeFailedPullRequest, Message: "remote " + s.Remote + " (" + strings.TrimSpace(url) + ") is not a GitHub repository"}
	}
	api := h.API
	if api == "" {
		api = GitHubAPI
	}
	return PullRequest{URL: PullsURL(api, owner, repo), Credential: h.CredentialID,
		Body: PullRequestBody{Title: h.Title, Head: s.Branch(), Base: s.DefaultBranch, Body: h.Body}}, nil
}

func (h *Handoff) open(ctx context.Context, pr PullRequest, token []byte) (string, error) {
	forge := h.Forge
	if forge == nil {
		forge = GitHub{}
	}
	details := map[string]string{"request": pr.URL, "head": pr.Body.Head, "base": pr.Body.Base}
	h.record(audit.KindPullRequestOpened, audit.PhaseAttempted, details)
	url, err := forge.Open(ctx, pr, token)
	if err != nil {
		details["error"] = messageOf(err)
		h.record(audit.KindPullRequestOpened, audit.PhaseFailed, details)
		var ge *Error
		if !errors.As(err, &ge) {
			err = &Error{Code: CodeFailedPullRequest, Message: err.Error()}
		}
		return "", err
	}
	details["url"] = url
	h.record(audit.KindPullRequestOpened, audit.PhaseDone, details)
	return url, nil
}

// remove is step 5: worktree-remove, then `git update-ref -d refs/sessions/<id>`.
func (h *Handoff) remove(ctx context.Context, out *Outcome) {
	s := h.Session
	if err := h.Backend.WorktreeRemove(s.Dir()); err != nil {
		out.RemoveError = err.Error()
		return
	}
	if _, err := run(ctx, CodeBadSession, s.Canonical, nil, "update-ref", "-d", SessionRef(s.ID)); err != nil {
		out.RemoveError = err.Error()
		return
	}
	out.Removed = true
}

func (h *Handoff) refused(err error) {
	var ge *Error
	if !errors.As(err, &ge) || ge.Code == CodeAlreadyHandedOff {
		return
	}
	type commit struct {
		Commit string `json:"commit"`
		Status string `json:"status"`
	}
	type hit struct {
		Commit   string `json:"commit"`
		Class    string `json:"class"`
		Location string `json:"location"`
	}
	d := struct {
		Code    string   `json:"code"`
		Message string   `json:"message"`
		Commits []commit `json:"commits,omitempty"`
		Privacy []hit    `json:"privacy,omitempty"`
	}{Code: ge.Code, Message: ge.Message}
	for _, c := range ge.Commits {
		d.Commits = append(d.Commits, commit{c.Hash, c.Status()})
	}
	for _, x := range ge.Hits {
		d.Privacy = append(d.Privacy, hit(x))
	}
	if b, err := json.Marshal(d); err == nil {
		h.emit(audit.KindHandoffRefused, audit.PhaseNone, b)
	}
}

func (h *Handoff) record(kind audit.Kind, phase audit.Phase, details map[string]string) {
	h.recordAs(h.Actor, kind, phase, details)
}

// recordAs is record for another actor than h.Actor ("" reads as it): the end's land rows name
// the ender (LandActor).
func (h *Handoff) recordAs(actor string, kind audit.Kind, phase audit.Phase, details map[string]string) {
	if actor == "" {
		actor = h.Actor
	}
	if b, err := json.Marshal(details); err == nil {
		h.emitAs(actor, kind, phase, b)
	}
}

// emit writes the event; an audit failure does not change the handoff's outcome, which has
// already happened outside the store (ADR 0021: attempted before, done or failed after).
func (h *Handoff) emit(kind audit.Kind, phase audit.Phase, b json.RawMessage) {
	h.emitAs(h.Actor, kind, phase, b)
}

func (h *Handoff) emitAs(actor string, kind audit.Kind, phase audit.Phase, b json.RawMessage) {
	if h.Record == nil {
		return
	}
	if actor == "" {
		actor = "instance"
	}
	_ = h.Record(audit.Event{Kind: kind, Phase: phase, Actor: actor, Session: h.Session.ID, Details: b})
}

func messageOf(err error) string {
	var ge *Error
	if errors.As(err, &ge) {
		return ge.Message
	}
	return err.Error()
}

// token is the project's git credential for a fetch or a push, nil when it has none; one that
// cannot be read is handoff-failed-push.
func (h *Handoff) token() ([]byte, error) {
	if h.CredentialID == "" || h.Secret == nil {
		return nil, nil
	}
	t, err := h.Secret(h.CredentialID)
	if err != nil {
		return nil, &Error{Code: CodeFailedPush, Message: "git credential " + h.CredentialID + ": " + err.Error()}
	}
	return t, nil
}

// LandGate is a main land's gate (ADR 0028 Decision 1, step 3): the driver's land role, then
// the review fact naming tip, the sha refs/sessions/<id> holds after the land's own refresh. The
// session service supplies it, so git reads no store; a refusal is its error (LandNotAllowed,
// LandUnreviewed).
type LandGate func(ctx context.Context, tip string) error

// LandOptions are one land's: DryRun runs everything but the push; Gate is a main land's gate
// (nil refuses land-not-allowed); Claimed is a host land's --as value, recorded on the audit
// row as claimed, never as the actor (ADR 0028 Decision 5).
type LandOptions struct {
	DryRun  bool
	Gate    LandGate
	Claimed string
}

// EndLand is what a main end's land takes from the session service (ADR 0028 Amendments
// 2026-11-02): the gate over the ender's roles, and the ender, the `land` rows' actor (a Discord
// user id, host, or instance for the idle end) with a host end's --as as Claimed.
type EndLand struct {
	Gate           LandGate
	Actor, Claimed string
}

// LandResult is what a land did, as `session land` prints it: the handoff's line (Outcome, its
// outcome landed, pushed, pull-request, none or dry-run), the tip handed off, and two notes.
// Stale is git's line when the fetch of step 1 failed (the land went on from the canonical's
// copy); CanonicalLags is why step 5 left the canonical's default branch behind the landed sha
// (its tracking ref was moved, and Base reads it).
type LandResult struct {
	Outcome
	Sha           string `json:"sha,omitempty"`
	Stale         string `json:"stale,omitempty"`
	CanonicalLags string `json:"canonicalLags,omitempty"`
}

// Land is the one land of ADR 0028 Decision 1, run from the canonical, never opening the tree,
// under the session's bundle lock throughout (one land at a time per session; the reviewer pin,
// `worktree check` and a second land wait for it): (1) the fetch and fast-forward of the
// default branch as at session start (Fetch), a failure noted in Stale, a diverged canonical
// land-canonical-diverged; (2) the bundle and fetch into refs/sessions/<id>, the attribution
// check and the push scan, as the handoff's; (3) on a main project the gate, then `merge-base
// --is-ancestor <remote>/<db> <tip>`, false being land-not-fast-forward with the count; (4) the
// push from the canonical with the askpass helper, never --force, audited `land`, a rejection on
// a main project land-rejected with git's line (and the checkout warning when the remote is
// one); (5) on a main project the canonical's branch and tracking ref fast-forwarded to the
// landed sha (followCanonical). A push project pushes the session branch; a pull-request project
// the same, and opens the request unless one is open (Opened). No commits since the base is
// none. The tree and refs/sessions/<id> stay: the next land hands off what came after.
func (h *Handoff) Land(ctx context.Context, opts LandOptions) (LandResult, error) {
	s := h.Session
	if h.Mode != ModePush && h.Mode != ModePullRequest && h.Mode != ModeMain {
		return LandResult{}, &Error{Code: CodeBadSession, Message: "handoff is " + strconv.Quote(h.Mode) + ", want push, pull-request or main"}
	}
	if err := s.check(); err != nil {
		return LandResult{}, err
	}
	unlock, err := LockBundle(ctx, s.ID)
	if err != nil {
		return LandResult{}, err
	}
	defer unlock()

	var res LandResult
	token, terr := h.token()
	if terr != nil {
		res.Stale = messageOf(terr)
	} else {
		f, err := Fetch(ctx, s, h.Backend.Askpass(), token)
		var ge *Error
		if errors.As(err, &ge) && ge.Code == CodeCanonicalDiverged {
			return LandResult{}, &Error{Code: CodeLandCanonicalDiverged, Message: "Not landed: the host's copy of " +
				s.DefaultBranch + " and " + s.Remote + " have diverged; the owner reconciles it."}
		}
		if err != nil {
			return LandResult{}, err
		}
		if f.Stale {
			res.Stale = f.Error
		}
	}

	c, tip, err := collectLocked(ctx, h.Backend, s, h.Attribution)
	if err == nil {
		err = h.vet(ctx, c)
	}
	if err != nil {
		h.refused(err)
		return LandResult{}, err
	}
	res.Mode, res.Remote, res.Commits, res.Sha = h.Mode, s.Remote, c.Commits, tip
	for _, ch := range c.Checks {
		res.Log = append(res.Log, CommitLine{Hash: ch.Hash, Subject: ch.Subject})
	}
	if c.Commits == 0 {
		res.Outcome.Outcome = OutcomeNone
		if opts.DryRun {
			res.Outcome.Outcome = OutcomeDryRun
		}
		return res, nil
	}

	done := OutcomePushed
	switch h.Mode {
	case ModeMain:
		res.Refspec, done = SessionRef(s.ID)+":refs/heads/"+s.DefaultBranch, OutcomeLanded
		if opts.Gate == nil {
			return LandResult{}, LandNotAllowed(s.DefaultBranch)
		}
		if err := opts.Gate(ctx, tip); err != nil {
			return LandResult{}, err
		}
		if err := fastForwardOnto(ctx, s, tip); err != nil {
			return LandResult{}, err
		}
	case ModePullRequest:
		res.Refspec = SessionRef(s.ID) + ":refs/heads/" + s.Branch()
		if h.Opened == "" {
			pr, err := h.request(ctx)
			if err != nil {
				return LandResult{}, err
			}
			res.Request = &pr
		}
	default:
		res.Refspec = SessionRef(s.ID) + ":refs/heads/" + s.Branch()
	}
	if opts.DryRun {
		res.Outcome.Outcome = OutcomeDryRun
		return res, nil
	}
	if terr != nil {
		return LandResult{}, terr
	}
	if err := h.landPush(ctx, res, token, opts.Claimed, done); err != nil {
		return LandResult{}, err
	}
	res.Outcome.Outcome = done
	switch {
	case h.Mode == ModeMain:
		res.CanonicalLags = followCanonical(ctx, s, tip)
	case res.Request != nil:
		url, err := h.open(ctx, *res.Request, token)
		if err != nil {
			return res, err
		}
		res.Outcome.Outcome, res.URL = OutcomePullRequest, &url
	case h.Mode == ModePullRequest:
		url := h.Opened // the request a land opened: the push moved its branch
		res.Outcome.Outcome, res.URL = OutcomePullRequest, &url
	}
	return res, nil
}

// fastForwardOnto is step 3's check after the gate: <remote>/<db> must be an ancestor of tip,
// else land-not-fast-forward with the number of its commits tip lacks; the bot never rebases. A
// remote that has no such branch yet is no refusal: the push creates it.
func fastForwardOnto(ctx context.Context, s Session, tip string) error {
	tracking := trackingRef(s)
	if _, err := run(ctx, CodeFetch, s.Canonical, nil, "rev-parse", "--verify", "--quiet", tracking+"^{commit}"); err != nil {
		return nil
	}
	ok, err := isAncestor(ctx, s.Canonical, tracking, tip)
	if err != nil || ok {
		return err
	}
	out, err := run(ctx, CodeFetch, s.Canonical, nil, "rev-list", "--count", tip+".."+tracking)
	if err != nil {
		return err
	}
	return &Error{Code: CodeLandNotFastForward, Message: "Not landed: " + s.DefaultBranch + " moved by " + strings.TrimSpace(out) +
		" commits since this session branched. Fetch " + s.Remote + ", rebase this tree onto " + s.Remote + "/" + s.DefaultBranch +
		" (the driver or the runner), review again, then push again."}
}

// landPush is step 4: `git push <remote> <refspec>` from the canonical with the account's
// credential helpers cleared, never --force, audited `land` attempted, then done with the
// outcome or failed with git's line. A rejected main push is land-rejected; a push project's is
// the handoff's handoff-failed-push.
func (h *Handoff) landPush(ctx context.Context, res LandResult, token []byte, claimed, outcome string) error {
	details := map[string]string{"remote": res.Remote, "refspec": res.Refspec, "commits": strconv.Itoa(res.Commits), "sha": res.Sha}
	if claimed != "" {
		details["claimed"] = claimed
	}
	h.recordAs(h.LandActor, audit.KindLand, audit.PhaseAttempted, details)
	_, err := run(ctx, CodeFailedPush, h.Session.Canonical, PushEnv(h.Backend.Askpass(), token),
		"-c", "credential.helper=", "push", "--quiet", res.Remote, res.Refspec)
	if err != nil {
		details["error"] = messageOf(err)
		h.recordAs(h.LandActor, audit.KindLand, audit.PhaseFailed, details)
		if h.Mode != ModeMain {
			return err
		}
		s := h.Session
		msg := "Not landed: push rejected by the remote: " + strings.TrimPrefix(messageOf(err), "git push: ") + ". If " +
			s.DefaultBranch + " moved, push again; if the remote refuses the branch, the owner changes the remote or the project's handoff."
		// A land on a checkout remote repeats config check's inspection (the companion).
		if CheckoutRemote(ctx, s.Canonical, s.Remote, s.DefaultBranch) {
			msg += " " + CheckoutRemoteWarning(s.Remote, s.DefaultBranch)
		}
		return &Error{Code: CodeLandRejected, Message: msg}
	}
	details["outcome"] = outcome
	h.recordAs(h.LandActor, audit.KindLand, audit.PhaseDone, details)
	return nil
}

// followCanonical is step 5 after a done main push, under a deadline of its own: the canonical's
// tracking ref moved to sha unless it already holds it (followTracking), then its default branch
// fast-forwarded to it (FastForward) when the branch is an ancestor. "" when the branch followed;
// else why it lags, the tracking ref moved all the same, so Base reads the landed sha. The tree
// is untouched; nothing is re-collected.
func followCanonical(ctx context.Context, s Session, sha string) string {
	fctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), fastForwardTimeout)
	defer cancel()
	lags := func(why string) string {
		return "the host's copy of " + s.DefaultBranch + " was not moved to " + sha + ": " + why
	}
	if err := followTracking(fctx, s, sha); err != nil {
		return lags(messageOf(err))
	}
	out, err := run(fctx, CodeFetch, s.Canonical, nil, "rev-parse", "--verify", "--quiet", "refs/heads/"+s.DefaultBranch+"^{commit}")
	if err != nil {
		return lags(messageOf(err))
	}
	local := strings.TrimSpace(out)
	if local == sha {
		return ""
	}
	ok, err := isAncestor(fctx, s.Canonical, local, sha)
	switch {
	case err != nil:
		return lags(messageOf(err))
	case !ok:
		return lags("it holds commits the landed one does not")
	}
	if err := FastForward(fctx, s, sha, local); err != nil {
		return lags(messageOf(err))
	}
	return ""
}

// followTracking moves refs/remotes/<remote>/<db> to sha, a fetch's write after a done push: left
// alone when it already holds sha or a descendant of it (git's push or another session's fetch
// on the same project got there first), else `update-ref <ref> <sha> <old>`, so a concurrent
// write is refused, never moved backwards.
func followTracking(ctx context.Context, s Session, sha string) error {
	old := ""
	if out, err := run(ctx, CodeFetch, s.Canonical, nil, "rev-parse", "--verify", "--quiet", trackingRef(s)+"^{commit}"); err == nil {
		old = strings.TrimSpace(out)
	}
	if old == sha {
		return nil
	}
	if old != "" {
		if held, err := isAncestor(ctx, s.Canonical, sha, old); err != nil || held {
			return err
		}
	}
	// An empty old value: the ref must not exist yet.
	_, err := run(ctx, CodeFetch, s.Canonical, nil, "update-ref", trackingRef(s), sha, old)
	return err
}

// LandNotAllowed is the land-not-allowed refusal (the companion's text; wording: owner).
func LandNotAllowed(defaultBranch string) *Error {
	return &Error{Code: CodeLandNotAllowed, Message: "Not landed: landing on " + defaultBranch + " needs a land role in this channel."}
}

// LandUnreviewed is the land-unreviewed refusal for the tip sha, naming the newest reviewer
// row's commitError when there is one (the companion's text; wording: owner).
func LandUnreviewed(sha, commitError string) *Error {
	why := "review: none since " + sha
	if commitError != "" {
		why += "; the last reviewer launch failed: " + commitError
	}
	return &Error{Code: CodeLandUnreviewed, Message: "Not landed: " + why +
		". Commit, launch the reviewer as a call on its own, then push again."}
}

// CheckoutRemote is the inspection behind the companion's second config check warning: the
// remote named in the canonical is a path to a non-bare repository (`rev-parse
// --is-bare-repository` false) whose `symbolic-ref HEAD` is the default branch and whose
// receive.denyCurrentBranch is unset, true or refuse, so git refuses every push to that branch.
// A URL remote, or a path it cannot read, is false.
func CheckoutRemote(ctx context.Context, canonical, remote, defaultBranch string) bool {
	url, err := run(ctx, CodeClone, canonical, nil, "remote", "get-url", remote)
	if err != nil {
		return false
	}
	dir, ok := remotePath(canonical, strings.TrimSpace(url))
	if !ok {
		return false
	}
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		return false
	}
	bare, err := run(ctx, CodeClone, dir, nil, "rev-parse", "--is-bare-repository")
	if err != nil || strings.TrimSpace(bare) != "false" {
		return false
	}
	head, err := run(ctx, CodeClone, dir, nil, "symbolic-ref", "-q", "HEAD")
	if err != nil || strings.TrimSpace(head) != "refs/heads/"+defaultBranch {
		return false
	}
	deny, err := run(ctx, CodeClone, dir, nil, "config", "--get", "receive.denyCurrentBranch")
	if err != nil {
		return true // unset: git's default, refuse
	}
	// git reads a boolean true (true, yes, on, 1, any case) as refuse.
	switch strings.ToLower(strings.TrimSpace(deny)) {
	case "refuse", "true", "yes", "on", "1":
		return true
	}
	return false
}

// CheckoutRemoteWarning is the companion's checkout-remote warning without its `warning:
// <project>: ` prefix (wording: owner).
func CheckoutRemoteWarning(remote, defaultBranch string) string {
	return remote + " is a checkout with " + defaultBranch + " checked out; git refuses every push to it " +
		"(receive.denyCurrentBranch). Lifts: a bare repository beside the checkout that the owner pulls from; " +
		"receive.denyCurrentBranch updateInstead on the checkout (still refused while its tree or index differs " +
		"from HEAD); GitHub as the remote."
}

// remotePath is a remote URL read as a local path, a relative one against the canonical: a
// file:// URL or a plain path; any other scheme, or git's scp form host:path, is not one.
func remotePath(canonical, url string) (string, bool) {
	if p, ok := strings.CutPrefix(url, "file://"); ok {
		if len(p) > 2 && p[0] == '/' && p[2] == ':' {
			p = p[1:] // file:///C:/x
		}
		return p, true
	}
	if strings.Contains(url, "://") {
		return "", false
	}
	if i := strings.Index(url, ":"); i > 1 && !strings.ContainsAny(url[:i], `/\`) {
		return "", false
	}
	if !filepath.IsAbs(url) && !strings.HasPrefix(url, "/") {
		url = filepath.Join(canonical, url)
	}
	return url, true
}
