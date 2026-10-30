// SPDX-License-Identifier: GPL-3.0-or-later

package git

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/jasondorjeshort/broodforge/internal/audit"
	"github.com/jasondorjeshort/broodforge/internal/spawn"
)

// Handoff modes (the project's handoff) and outcomes.
const (
	ModePush        = "push"
	ModePullRequest = "pull-request"

	OutcomeNone        = "none"         // no commits: the tree was removed
	OutcomePushed      = "pushed"       // push: the branch is on the remote
	OutcomePullRequest = "pull-request" // pull-request: pushed and the request opened
	OutcomeDryRun      = "dry-run"      // --dry-run: checked, nothing sent
)

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
	bundle, err := b.WorktreeBundle(s.Worktree(base))
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
	_, _, found, err := SessionTip(ctx, s)
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

// Collect runs steps 1 to 3 (ADR 0022 companion, Handoff at session end): worktree-bundle,
// then in the canonical `git bundle verify` and `git fetch <bundle>
// +<session branch>:refs/sessions/<id>`, then Check. The fetch is forced because the driver may
// have had commits amended after a refused check; the ref is the bot's own. No commits since
// the base sets the ref back to the base. The bot never opens the session's tree.
func Collect(ctx context.Context, b spawn.Backend, s Session, a Attribution) (Collected, error) {
	bundle, base, err := BundleStep(ctx, b, s)
	if err != nil {
		return Collected{}, err
	}
	c := Collected{Base: base, Commits: bundle.Commits}
	if bundle.Commits == 0 {
		_, err := run(ctx, CodeRefusedFetch, s.Canonical, nil, "update-ref", SessionRef(s.ID), base)
		return c, err
	}
	if _, err := run(ctx, CodeRefusedBundleVerify, s.Canonical, nil, "bundle", "verify", "--quiet", bundle.Path); err != nil {
		return c, err
	}
	if _, err := run(ctx, CodeRefusedFetch, s.Canonical, nil, "fetch", "--quiet", "--no-tags", "--no-write-fetch-head",
		bundle.Path, "+refs/heads/"+s.Branch()+":"+SessionRef(s.ID)); err != nil {
		return c, err
	}
	// The check's base is taken again after the fetch: a session that merged the default
	// branch's newer commits forks from it later, and those commits are not the session's.
	if c.Base, err = Base(ctx, s); err != nil {
		return c, err
	}
	if c.Checks, err = Check(ctx, s, c.Base, a); err != nil {
		return c, err
	}
	c.Commits = len(c.Checks)
	return c, nil
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
	// Log is the commits handed off, oldest first, for the end-of-session summary (04, Ending a
	// session); `worktree handoff` does not print it.
	Log []CommitLine `json:"-"`
}

// CommitLine is one commit as the summary lists it: its hash and its subject.
type CommitLine struct {
	Hash    string `json:"hash"`
	Subject string `json:"subject"`
}

// Handoff is one session's handoff: the five steps of the ADR 0022 companion.
type Handoff struct {
	Backend     spawn.Backend
	Session     Session
	Mode        string // ModePush or ModePullRequest
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
	// Record writes one audit event (push, pull-request-opened, handoff-refused); nil records nothing.
	Record func(audit.Event) error
	Actor  string
}

// Run hands the session off. Refusals keep the tree; a done push or request removes it through
// worktree-remove and deletes refs/sessions/<id> (R-35). dryRun stops after the check: the
// attribution check, then, with every commit passing it, the push scan.
func (h *Handoff) Run(ctx context.Context, dryRun bool) (Outcome, error) {
	s := h.Session
	if h.Mode != ModePush && h.Mode != ModePullRequest {
		return Outcome{}, &Error{Code: CodeBadSession, Message: "handoff is " + strconv.Quote(h.Mode) + ", want push or pull-request"}
	}
	c, err := Collect(ctx, h.Backend, s, h.Attribution)
	if err == nil {
		if failing := c.Failing(); len(failing) > 0 {
			err = &Error{Code: CodeRefusedAttribution, Commits: failing,
				Message: strconv.Itoa(len(failing)) + " of " + strconv.Itoa(len(c.Checks)) + " commits fail the attribution check"}
		} else if c.Commits > 0 {
			var hits []PrivacyHit
			if hits, err = ScanPrivacy(ctx, s, c.Base, h.Privacy.WithSession(h.Attribution)); err == nil && len(hits) > 0 {
				err = PrivacyRefusal(hits, c.Commits)
			}
		}
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
	if h.Mode == ModePullRequest {
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
	var token []byte
	if h.CredentialID != "" && h.Secret != nil {
		if token, err = h.Secret(h.CredentialID); err != nil {
			return out, &Error{Code: CodeFailedPush, Message: "git credential " + h.CredentialID + ": " + err.Error()}
		}
	}
	if err := h.push(ctx, out, token); err != nil {
		return out, err
	}
	out.Outcome = OutcomePushed
	if out.Request != nil {
		url, err := h.open(ctx, *out.Request, token)
		if err != nil {
			return out, err
		}
		out.Outcome, out.URL = OutcomePullRequest, &url
	}
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
	if b, err := json.Marshal(details); err == nil {
		h.emit(kind, phase, b)
	}
}

// emit writes the event; an audit failure does not change the handoff's outcome, which has
// already happened outside the store (ADR 0021: attempted before, done or failed after).
func (h *Handoff) emit(kind audit.Kind, phase audit.Phase, b json.RawMessage) {
	if h.Record == nil {
		return
	}
	actor := h.Actor
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
