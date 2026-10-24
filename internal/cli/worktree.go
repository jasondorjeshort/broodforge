// SPDX-License-Identifier: GPL-3.0-or-later

package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jasondorjeshort/broodforge/internal/audit"
	"github.com/jasondorjeshort/broodforge/internal/config"
	"github.com/jasondorjeshort/broodforge/internal/control"
	"github.com/jasondorjeshort/broodforge/internal/git"
	"github.com/jasondorjeshort/broodforge/internal/policy"
	"github.com/jasondorjeshort/broodforge/internal/spawn"
	"github.com/jasondorjeshort/broodforge/internal/store"
	"github.com/jasondorjeshort/broodforge/internal/workflow"
)

// The worktree commands of 05-debug-interface.md (Git handoff) that P8 builds: check, bundle
// and handoff are writes the bot serves (WorktreeHandlers), because each fetches into the
// canonical; list, forget and adopt are later passes'. git-askpass and git-commit-msg are the
// hidden standalone commands share/askpass.sh, share/askpass.cmd and share/hooks/commit-msg run;
// git-pre-push, which share/hooks/pre-push runs, is in git.go.

// CodeUnknownSession is a session id the store does not hold; a project id the configuration
// lacks is CodeUnknownProject (config.go).
const CodeUnknownSession = "unknown-session"

var commitColumns = []string{"commit", "status"}

// hookColumns are worktree check's hook row in text order: `ok|missing  pre-push`.
var hookColumns = []string{"status", "hook"}

// privacyColumns are a push-scan hit's row: `privacy: <sha>  <class>  <location>`.
var privacyColumns = []string{"privacy:", "class", "location"}

// checkEventColumns tells worktree check's hook row, which has a "hook" key, and a push-scan
// hit, which has a "privacy" key, from its commit lines.
func checkEventColumns(ev json.RawMessage) []string {
	var row struct {
		Hook    *string `json:"hook"`
		Privacy *string `json:"privacy"`
	}
	switch {
	case json.Unmarshal(ev, &row) != nil:
	case row.Hook != nil:
		return hookColumns
	case row.Privacy != nil:
		return privacyColumns
	}
	return commitColumns
}

// worktreeCheck is `broodforge worktree check <id>`: first `ok|missing  pre-push` (the tree
// still runs the bot's pre-push hook, ADR 0025 rule 4; --json `{"hook","status"}`), then bundle,
// verify and fetch, then the attribution check: one `<hash>  ok|<problems>` line per commit since the base, then
// `ok  <commits>`; any failing commit is handoff-refused-attribution, exit 1; else the push scan
// (ADR 0025 rule 7), one `privacy: <sha>  <class>  <location>` line per hit (--json
// `{"privacy","class","location"}`) and handoff-refused-privacy, exit 1; else a missing hook is
// pre-push-missing, exit 1 (handoff is not refused for it).
var worktreeCheck = Command{
	Name:            "worktree check",
	Class:           ClassWrite,
	Flags:           []Flag{{Name: "id", Kind: FlagString, Value: "<session id>"}},
	EventColumns:    commitColumns,
	EventColumnsFor: checkEventColumns,
	Columns:         []string{"result", "commits"},
}

// worktreeBundle is `broodforge worktree bundle <id>`: the bundle step alone, printing
// `<bundle path>  <commits>`, or `none  0` for a session with no commits.
var worktreeBundle = Command{
	Name:    "worktree bundle",
	Class:   ClassWrite,
	Flags:   []Flag{{Name: "id", Kind: FlagString, Value: "<session id>"}},
	Columns: []string{"bundle", "commits"},
}

// worktreeHandoff is `broodforge worktree handoff <id> [--dry-run] [--expect <value>]`: the
// companion's five steps, printing `<outcome>  <remote>  <refspec>  <commits>  <url>
// <request>`; a refused attribution streams the failing commit lines first, a refused push
// scan its `privacy:` lines.
var worktreeHandoff = Command{
	Name:   "worktree handoff",
	Class:  ClassWrite,
	Expect: true,
	Flags: []Flag{
		{Name: "dry-run", Kind: FlagBool, Usage: "stop after the check and print what would be sent"},
		{Name: "id", Kind: FlagString, Value: "<session id>"},
	},
	EventColumns:    commitColumns,
	EventColumnsFor: checkEventColumns,
	Columns:         []string{"outcome", "remote", "refspec", "commits", "url", "request"},
}

// gitAskpass is the hidden `broodforge git-askpass <prompt>`, git's GIT_ASKPASS through the
// share scripts: the answer on stdout, or no-git-credential, exit 1.
var gitAskpass = Command{
	Name:   "git-askpass",
	Class:  ClassStandalone,
	Hidden: true,
	Flags:  []Flag{{Name: "prompt", Kind: FlagString, Value: "<prompt>"}},
	Local:  runGitAskpass,
}

// gitCommitMsg is the hidden `broodforge git-commit-msg <file>` that share/hooks/commit-msg
// runs: it appends the session's missing trailer lines. A failure is one `commit-msg: <what>`
// line on stderr and exit 0, because the hook is a help and the check at handoff the rule.
var gitCommitMsg = Command{
	Name:   "git-commit-msg",
	Class:  ClassStandalone,
	Hidden: true,
	Flags:  []Flag{{Name: "file", Kind: FlagString, Value: "<file>"}},
	Local:  runGitCommitMsg,
}

func init() {
	commands = append(commands, &worktreeCheck, &worktreeBundle, &worktreeHandoff, &gitAskpass, &gitCommitMsg)
	positionals[&worktreeCheck] = []string{"id"}
	positionals[&worktreeBundle] = []string{"id"}
	positionals[&worktreeHandoff] = []string{"id"}
	positionals[&gitAskpass] = []string{"prompt"}
	positionals[&gitCommitMsg] = []string{"file"}
}

func runGitAskpass(_ context.Context, env *Env) error {
	answer, err := git.AskpassAnswer(env.String("prompt"), os.LookupEnv)
	if err != nil {
		return gitError(err)
	}
	_, err = fmt.Fprintln(env.Out.w, answer)
	return err
}

func runGitCommitMsg(ctx context.Context, env *Env) error {
	if err := git.CommitMsg(ctx, env.String("file")); err != nil {
		msg := err.Error()
		var ge *git.Error
		if errors.As(err, &ge) {
			msg = ge.Message
		}
		fmt.Fprintln(env.Stderr, "commit-msg: "+control.OneLine(msg))
	}
	return nil
}

// WorktreeService is the bot side of the worktree commands; `serve` (P10) and the tests'
// harness build one. Until the session service exists it reads the session, driver,
// participant and usage rows itself, as P10's end-of-session sequence will.
type WorktreeService struct {
	Root    string
	Config  *config.Config
	Store   *store.Store
	Audit   *audit.Writer
	Backend spawn.Backend
	// Secret reads a credential's value (credentials.Service.Secret); nil sends none.
	Secret func(id string) ([]byte, error)
	Forge  git.PullRequester // nil: git.GitHub
	API    string            // the forge API root; "": git.GitHubAPI
	// Current, when set, is the configuration as loaded now (`config reload`); else Config.
	Current func() *config.Config
}

// actorHostCLI is the audit actor of a handoff run from the debug CLI.
const actorHostCLI = "host"

// hookLine is worktree check's first row: whether the tree still runs the bot's pre-push hook.
type hookLine struct {
	Hook   string `json:"hook"`
	Status string `json:"status"`
}

type commitLine struct {
	Commit string `json:"commit"`
	Status string `json:"status"`
}

// privacyLine is one push-scan hit: the commit, the class, where; never the value.
type privacyLine struct {
	Privacy  string `json:"privacy"`
	Class    string `json:"class"`
	Location string `json:"location"`
}

type attributionResult struct {
	Result  string `json:"result"`
	Commits int    `json:"commits"`
}

type bundleResult struct {
	Bundle  string `json:"bundle"`
	Commits int    `json:"commits"`
}

func encodeWorktree[T hookLine | commitLine | privacyLine | attributionResult | bundleResult | git.Outcome | endLine](v T) (json.RawMessage, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimSpace(b.Bytes()), nil
}

func emitCommits(call *control.Call, checks []git.CommitCheck) error {
	for _, c := range checks {
		ev, err := encodeWorktree(commitLine{c.Hash, c.Status()})
		if err != nil {
			return err
		}
		if err := call.Emit(ev); err != nil {
			return err
		}
	}
	return nil
}

func emitPrivacy(call *control.Call, hits []git.PrivacyHit) error {
	for _, h := range hits {
		ev, err := encodeWorktree(privacyLine{h.Commit, h.Class, h.Location})
		if err != nil {
			return err
		}
		if err := call.Emit(ev); err != nil {
			return err
		}
	}
	return nil
}

// WorktreeHandlers registers worktree.check, worktree.bundle and worktree.handoff.
func WorktreeHandlers(s *WorktreeService) func(*control.Registry) {
	byID := func(f func(ctx context.Context, call *control.Call, id string, dryRun bool) (json.RawMessage, error)) control.Handler {
		return func(ctx context.Context, call *control.Call) (json.RawMessage, error) {
			var a struct {
				ID     string
				DryRun bool
			}
			if err := json.Unmarshal(call.Args, &a); err != nil {
				return nil, control.NewError(control.CodeBadArgs, err.Error())
			}
			if !git.ValidSessionID(a.ID) {
				return nil, control.NewError(control.CodeBadArgs, fmt.Sprintf("%q is not a session id", a.ID))
			}
			return f(ctx, call, a.ID, a.DryRun)
		}
	}
	return func(reg *control.Registry) {
		reg.Handle("worktree.check", byID(s.check))
		reg.Handle("worktree.bundle", byID(s.bundle))
		reg.Handle("worktree.handoff", byID(s.handoff))
	}
}

func (s *WorktreeService) check(ctx context.Context, call *control.Call, id string, _ bool) (json.RawMessage, error) {
	sess, err := s.load(id)
	if err != nil {
		return nil, err
	}
	// The hook's row comes first, before the bundle. A tree the backend cannot read (gone
	// after a handoff) prints no row, so the bundle step's own refusal stays the error.
	prePush, hookErr := s.Backend.WorktreePrePush(sess.git.Dir())
	if hookErr == nil {
		state := "ok"
		if !prePush {
			state = "missing"
		}
		ev, err := encodeWorktree(hookLine{Hook: "pre-push", Status: state})
		if err != nil {
			return nil, err
		}
		if err := call.Emit(ev); err != nil {
			return nil, err
		}
	}
	c, err := git.Collect(ctx, s.Backend, sess.git, sess.attr)
	if err != nil {
		return nil, gitError(err)
	}
	if hookErr != nil {
		return nil, gitError(hookErr)
	}
	if err := emitCommits(call, c.Checks); err != nil {
		return nil, err
	}
	if failing := c.Failing(); len(failing) > 0 {
		return nil, control.NewError(git.CodeRefusedAttribution,
			fmt.Sprintf("%d of %d commits fail the attribution check", len(failing), len(c.Checks)))
	}
	if len(c.Checks) > 0 {
		p, err := s.scanPrivacy(sess, id)
		if err != nil {
			return nil, gitError(err)
		}
		hits, err := git.ScanPrivacy(ctx, sess.git, c.Base, p.WithSession(sess.attr))
		if err != nil {
			return nil, gitError(err)
		}
		if err := emitPrivacy(call, hits); err != nil {
			return nil, err
		}
		if len(hits) > 0 {
			return nil, control.NewError(git.CodeRefusedPrivacy, git.PrivacySummary(hits, len(c.Checks)))
		}
	}
	if !prePush {
		return nil, control.NewError(CodePrePushMissing,
			"the tree's core.hooksPath is not "+s.Backend.HooksDir()+" with its pre-push hook; handoff still runs")
	}
	return encodeWorktree(attributionResult{"ok", len(c.Checks)})
}

func (s *WorktreeService) bundle(ctx context.Context, _ *control.Call, id string, _ bool) (json.RawMessage, error) {
	sess, err := s.load(id)
	if err != nil {
		return nil, err
	}
	b, _, err := git.BundleStep(ctx, s.Backend, sess.git)
	if err != nil {
		return nil, gitError(err)
	}
	if b.Commits == 0 {
		return encodeWorktree(bundleResult{"none", 0})
	}
	return encodeWorktree(bundleResult{b.Path, b.Commits})
}

func (s *WorktreeService) handoff(ctx context.Context, call *control.Call, id string, dryRun bool) (json.RawMessage, error) {
	out, err := s.runHandoff(ctx, id, dryRun, actorHostCLI)
	if err != nil {
		var ge *git.Error
		if errors.As(err, &ge) && ge.Code == git.CodeRefusedAttribution {
			if eerr := emitCommits(call, ge.Commits); eerr != nil {
				return nil, eerr
			}
		}
		if errors.As(err, &ge) && ge.Code == git.CodeRefusedPrivacy && len(ge.Hits) > 0 {
			// The hits are the lines; the error is the count, as worktree check prints it.
			if eerr := emitPrivacy(call, ge.Hits); eerr != nil {
				return nil, eerr
			}
			return nil, control.NewError(ge.Code, git.PrivacySummary(ge.Hits, ge.Checked))
		}
		var ce *control.Error
		if errors.As(err, &ce) {
			return nil, ce
		}
		return nil, gitError(err)
	}
	return encodeWorktree(out)
}

// ActorBot is the audit actor of a handoff the bot runs itself, at the end of a session.
const ActorBot = "instance"

// runHandoff is the companion's five steps for one session: the tree's state and the
// session's outcome recorded as `worktree handoff` records them. A git refusal is returned as
// the *git.Error, its failing commits with it.
func (s *WorktreeService) runHandoff(ctx context.Context, id string, dryRun bool, actor string) (git.Outcome, error) {
	sess, err := s.load(id)
	if err != nil {
		return git.Outcome{}, err
	}
	p, err := s.scanPrivacy(sess, id)
	if err != nil {
		return git.Outcome{}, err
	}
	h := &git.Handoff{Backend: s.Backend, Session: sess.git, Mode: sess.project.HandoffOrDefault(),
		Attribution: sess.attr, Privacy: p, CredentialID: sess.project.Repo.GitCredentialID, Secret: s.Secret,
		Forge: s.Forge, API: s.API, Title: sess.title, Record: s.record, Actor: actor}
	out, err := h.Run(ctx, dryRun)
	if err != nil {
		var ge *git.Error
		if errors.As(err, &ge) && (ge.Code == git.CodeFailedPush || ge.Code == git.CodeFailedPullRequest) {
			s.setWorktreeState(id, "kept")
		}
		return out, err
	}
	if out.Removed {
		s.setWorktreeState(id, store.WorktreeRemoved)
	}
	if out.Outcome == git.OutcomePushed || out.Outcome == git.OutcomePullRequest {
		outcome := sess.git.Branch()
		if out.URL != nil {
			outcome = *out.URL
		}
		_ = s.Store.Write(func(tx *store.Tx) error {
			_, err := tx.Exec("UPDATE session SET end_outcome = ? WHERE id = ?", store.Text(outcome), store.Text(id))
			return err
		})
	}
	return out, nil
}

// setWorktreeState records the tree's state on its row, when the session has one.
func (s *WorktreeService) setWorktreeState(id, state string) {
	_ = s.Store.Write(func(tx *store.Tx) error {
		_, err := tx.Exec("UPDATE worktree SET state = ? WHERE session_id = ? AND state != ?",
			store.Text(state), store.Text(id), store.Text(store.WorktreeRemoved))
		return err
	})
}

func (s *WorktreeService) record(ev audit.Event) error {
	return s.Store.Write(func(tx *store.Tx) error {
		_, err := s.Audit.Append(tx, ev)
		return err
	})
}

type loadedSession struct {
	git     git.Session
	attr    git.Attribution
	privacy git.Privacy
	project *config.Project
	title   string
}

// titleRunes bounds the first message in the fallback pull-request title.
const titleRunes = 60

// load reads what a handoff needs: the session and its project's repo, the driver's
// registration, the other participants in join order (a contributor joins on their first
// accepted message, R-13, so joined_at is when they first wrote), and every model with
// non-zero usage, by policy's display name, from the start of its first turn (the usage row's
// time only where no turn row exists).
func (s *WorktreeService) load(id string) (loadedSession, error) {
	var ls loadedSession
	var projectID, driverID, firstInput string
	found := false
	err := s.Store.Query(`SELECT s.project_id, s.driver_id, r.git_name, r.git_email,
    coalesce((SELECT input FROM turn WHERE session_id = s.id ORDER BY seq LIMIT 1), '')
FROM session s JOIN registration r ON r.user_id = s.driver_id WHERE s.id = ?`, []store.Value{store.Text(id)},
		func(r *store.Row) error {
			found = true
			projectID, driverID, firstInput = r.Text(0), r.Text(1), r.Text(4)
			ls.attr.Driver = git.Identity{Name: r.Text(2), Email: r.Text(3)}
			return nil
		})
	if err != nil {
		return ls, err
	}
	if !found {
		return ls, control.NewError(CodeUnknownSession, "no session "+id)
	}
	if ls.project = s.config().Project(projectID); ls.project == nil {
		return ls, control.NewError(CodeUnknownProject, id+" names project "+projectID+", which the configuration does not define")
	}
	err = s.Store.Query(`SELECT r.git_name, r.git_email, p.joined_at FROM participant p
JOIN registration r ON r.user_id = p.user_id
WHERE p.session_id = ? AND p.user_id != ? ORDER BY p.joined_at, p.rowid`,
		[]store.Value{store.Text(id), store.Text(driverID)}, func(r *store.Row) error {
			t, err := time.Parse(store.TimeLayout, r.Text(2))
			if err != nil {
				return err
			}
			ls.attr.Participants = append(ls.attr.Participants, git.Contributor{Identity: git.Identity{Name: r.Text(0), Email: r.Text(1)}, Since: t})
			return nil
		})
	if err != nil {
		return ls, err
	}
	// A model is a co-author from the start of its first turn: its usage row is written at
	// the turn's end, after the commits the turn made (03-architecture.md, Attribution).
	err = s.Store.Query(`SELECT u.model, min(coalesce(t.started_at, u.recorded_at)) FROM usage_record u
LEFT JOIN turn t ON t.session_id = u.session_id AND t.seq = u.turn_seq
WHERE u.session_id = ? AND u.input_tokens + u.output_tokens + u.cache_read_tokens + u.cache_write_tokens > 0
GROUP BY u.model ORDER BY 2, 1`, []store.Value{store.Text(id)}, func(r *store.Row) error {
		t, err := time.Parse(store.TimeLayout, r.Text(1))
		if err != nil {
			return err
		}
		ls.attr.Models = append(ls.attr.Models, git.Contributor{
			Identity: git.Identity{Name: policy.ModelDisplayName(r.Text(0)), Email: git.ModelEmail}, Since: t})
		return nil
	})
	if err != nil {
		return ls, err
	}
	ls.privacy = instancePrivacy(s.config().Instance.Privacy)
	repo := ls.project.Repo
	ls.git = git.Session{Root: s.Root, ID: id, Instance: s.config().Instance.InstanceName, Canonical: repo.Canonical,
		Remote: orDefault(repo.Remote, "origin"), DefaultBranch: orDefault(repo.DefaultBranch, "main")}
	ls.title = "Session " + id
	if firstInput != "" {
		ls.title += ": " + truncateRunes(strings.Join(strings.Fields(firstInput), " "), titleRunes)
	}
	return ls, nil
}

// scanPrivacy is the push scan's lists for a session: the instance's, extended by the privacy
// section of the session's workflow copy when it has one (ADR 0025 rule 7). Only the paths that
// scan read it, so a malformed copy never fails `worktree bundle`.
func (s *WorktreeService) scanPrivacy(sess loadedSession, id string) (git.Privacy, error) {
	return sess.privacy.WithCopy(workflow.Dir(s.Root, id))
}

// instancePrivacy is the push scan's data from config/instance.json and the machine's own
// literals, the OS user and host names (ADR 0025 rule 7); the
// scan adds the session's identities (git.Privacy.WithSession). config check refuses an
// identity that does not parse, so none is dropped here from a loaded configuration.
func instancePrivacy(p config.Privacy) git.Privacy {
	out := git.Privacy{Never: p.Never, Allow: p.Allow}
	for _, m := range config.MachineLiterals() {
		out.Machine = append(out.Machine, m.Value)
	}
	for _, id := range p.Identities {
		if name, email, ok := config.ParseIdentity(id); ok {
			out.Identities = append(out.Identities, git.Identity{Name: name, Email: email})
		}
	}
	return out
}

func orDefault(s, d string) string {
	if s == "" {
		return d
	}
	return s
}

func truncateRunes(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	r := []rune(s)
	return string(r[:n-1]) + "…"
}

// gitError maps a git or spawn refusal to the CLI's error line.
func gitError(err error) error {
	var ge *git.Error
	if errors.As(err, &ge) {
		return control.NewError(ge.Code, control.OneLine(ge.Message))
	}
	return spawnError(err)
}

func (s *WorktreeService) config() *config.Config {
	if s.Current != nil {
		return s.Current()
	}
	return s.Config
}

// Commits is session id's commit count over its base, read through the backend from its tree
// without a bundle (git.Commits): the session service's Config.Commits, a held session's notice
// after a restart (ADR 0026).
func (s *WorktreeService) Commits(ctx context.Context, id string) (*int, error) {
	sess, err := s.load(id)
	if err != nil {
		return nil, err
	}
	n, err := git.Commits(ctx, s.Backend, sess.git)
	if err != nil {
		return nil, err
	}
	return &n, nil
}

// Handoff is the handoff of session id as `worktree handoff` runs it, for the end of session
// (the session service's Handoff): the outcome, or the git refusal with its failing commits.
func (s *WorktreeService) Handoff(ctx context.Context, id string) (git.Outcome, error) {
	return s.runHandoff(ctx, id, false, ActorBot)
}
