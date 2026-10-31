// SPDX-License-Identifier: GPL-3.0-or-later

package session

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jasondorjeshort/broodforge/internal/audit"
	"github.com/jasondorjeshort/broodforge/internal/git"
	"github.com/jasondorjeshort/broodforge/internal/store"
)

// The session's workflow (ADR 0025): the copy made once at start through Config.Workflow (this
// package does not import workflow, 03's session row), the header written at the top of the
// copy's instructions.md, the start-time probe's refusal, and what `session show` reports of
// it: the `workflow:` and `hooks:` lines and the review fact (rule 7).

// WorkflowKey is a project's workflow key as the bot loaded it (03 Configuration): Inject
// "repository" ("" reads the same) or "none"; Hooks, the events the hook chains.
type WorkflowKey struct {
	Inject string
	Hooks  []string
}

// WorkflowRequest is one session's copy: the directory it is made in (absent or empty), and the
// project's canonical, default branch and workflow key.
type WorkflowRequest struct {
	Dir               string // <root>/workflow/<session id>, absolute, slash-separated
	Canonical, Branch string
	Key               WorkflowKey
}

// WorkflowCopy is what the copy made. Instructions is always written (empty when nothing was
// copied); the session puts the header at its top.
type WorkflowCopy struct {
	Line           string // the `workflow:` line's value: `<commit> <date>` or `none (...)`
	Copied         bool   // false under inject none or when the branch has neither CLAUDE.md nor .claude
	Branch, Commit string
	Date           time.Time
	Agents         string   // agents.json, absolute; "" when the copy defines no agent
	Hooks          []string // hooks.json's names, in its order
}

// WorkflowRefusal is a start the copy or its probe refused. Code and Message are the start's
// error (`error: <code>: <message>`); the start's Discord reply is `Could not start a session:
// <message>.`, ephemeral to the starter alone (the owner, 2026-10-23).
type WorkflowRefusal struct {
	Code, Message string
}

func (e *WorkflowRefusal) Error() string { return e.Code + ": " + e.Message }

// CodeNoHeader is `session show --header` for a session without a workflow copy (one started
// before P19, or whose copy is gone).
const CodeNoHeader = "no-header"

// instructionsFile is the copy's rendered prompt, passed as --append-system-prompt-file.
const instructionsFile = "instructions.md"

// stageWorkflow is the start stage of the copy and the probe.
const stageWorkflow = "workflow"

// workflowDir is a session's copy, workflow.Dir's <root>/workflow/<id>.
func workflowDir(root, id string) string { return root + "/workflow/" + id }

// workflowState is a live session's copy: what every engine process is given.
type workflowState struct {
	dir          string
	instructions string   // --append-system-prompt-file, always
	agents       string   // --agents, "" for none
	copied       bool     // BROODFORGE_WORKFLOW is set only then (the companion, step 4)
	line         string   // the `workflow:` line; "" without Config.Workflow
	hooks        []string // the `hooks:` names
}

// makeWorkflow is the copy at start (ADR 0025 rules 5 and 6): Config.Workflow renders the copy
// and probes its hooks (nil makes an empty directory: the header alone), then the header goes at
// the top of instructions.md, a blank line before the project's text. Any failure removes the
// directory. Nothing is copied again during the session; a rule change reaches the next one.
func (s *Service) makeWorkflow(ctx context.Context, ch Channel, gs git.Session, reach string, reg registration) (workflowState, error) {
	dir := workflowDir(s.cfg.Root, gs.ID)
	wf := workflowState{dir: dir, instructions: dir + "/" + instructionsFile}
	var c WorkflowCopy
	if s.cfg.Workflow != nil {
		var err error
		c, err = s.cfg.Workflow(ctx, WorkflowRequest{Dir: dir, Canonical: gs.Canonical,
			Branch: gs.DefaultBranch, Key: ch.Workflow})
		if err != nil {
			return wf, err
		}
	} else if err := os.MkdirAll(filepath.FromSlash(dir), 0o755); err != nil {
		return wf, err
	}
	body, err := os.ReadFile(filepath.FromSlash(wf.instructions))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		_ = os.RemoveAll(filepath.FromSlash(dir))
		return wf, err
	}
	text := SystemPromptAppend(Header{Session: gs.ID, ProjectName: ch.ProjectName, Reach: reach, DriverName: reg.Name,
		Copied: c.Copied, Branch: c.Branch, Commit: c.Commit, Date: c.Date})
	if strings.TrimSpace(string(body)) != "" {
		text += "\n" + string(body)
	}
	if err := os.WriteFile(filepath.FromSlash(wf.instructions), []byte(text), 0o644); err != nil {
		_ = os.RemoveAll(filepath.FromSlash(dir))
		return wf, err
	}
	wf.agents, wf.copied, wf.line, wf.hooks = c.Agents, c.Copied, c.Line, c.Hooks
	return wf, nil
}

// ReadHeader is the header session id got, as its instructions.md holds it: every line up to
// the first blank one (SystemPromptAppend writes none inside it).
func ReadHeader(root, id string) (string, error) {
	b, err := os.ReadFile(filepath.FromSlash(workflowDir(root, id) + "/" + instructionsFile))
	if errors.Is(err, fs.ErrNotExist) {
		return "", &Error{CodeNoHeader, id + " has no workflow copy"}
	}
	if err != nil {
		return "", err
	}
	text := strings.ReplaceAll(string(b), "\r\n", "\n")
	if head, _, ok := strings.Cut(text, "\n\n"); ok {
		return head + "\n", nil
	}
	return text, nil
}

// Review is rule 7's fact: the session's last commit as of the last bundle (refs/sessions/<id>,
// which `worktree check` and the handoff refresh), and the newest `reviewer` subagent call whose
// result's audit row (subagent-result, a result that is not an error) names that commit as the
// one the bot bundled at the reviewer's launch; nil ToolUseID is none. Error, when set, is why the fact could not be
// read (the canonical moved, the configuration does not load): `review: error: <code>`.
type Review struct {
	Commit    string       `json:"commit"`
	ToolUseID *string      `json:"toolUseId"`
	Error     *ReviewError `json:"error,omitempty"`
}

// ReviewError is a review fact that could not be read.
type ReviewError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ReviewFailed is the Review of a fact that could not be read.
func ReviewFailed(code, message string) *Review {
	return &Review{Error: &ReviewError{Code: code, Message: message}}
}

// subagentTools are the names of the engine's subagent tool: Agent, Task on an older pin.
var subagentTools = map[string]bool{"Agent": true, "Task": true}

// reviewerAgent is the subagent type the review fact counts.
const reviewerAgent = "reviewer"

// Results of a subagent-result audit row.
const (
	subagentResultOK    = "ok"
	subagentResultError = "error"
)

// ReviewFact reads rule 7's fact for gs: nil only when the canonical holds no refs/sessions/<id>
// (the session was handed off); a ref that cannot be read is a Review carrying the git error.
// The rows counted are subagent-result events, written when a subagent call's tool result comes
// (never at a person's answer), whose `agent` is reviewer and whose result is ok: a reviewer
// allowed and then interrupted has no such row, and an errored or denied one is not ok. The
// fact holds only when the newest such row's `commit` (Refresh's sha at the launch) equals the
// ref's sha; a row with `commitError` never counts. No time enters the comparison.
func ReviewFact(ctx context.Context, q store.Querier, gs git.Session) (*Review, error) {
	sha, found, err := git.SessionTip(ctx, gs)
	var ge *git.Error
	switch {
	case errors.As(err, &ge):
		return ReviewFailed(ge.Code, ge.Message), nil
	case err != nil:
		return nil, err
	case !found:
		return nil, nil
	}
	r := &Review{Commit: sha}
	var newest struct{ ToolUseID, Commit, CommitError string }
	err = q.Query(`SELECT details FROM audit_event WHERE session_id = ? AND kind = ? ORDER BY seq`,
		[]store.Value{store.Text(gs.ID), store.Text(string(audit.KindSubagentResult))},
		func(row *store.Row) error {
			var d struct{ Tool, Agent, Result, ToolUseID, Commit, CommitError string }
			if json.Unmarshal([]byte(row.Text(0)), &d) != nil {
				return nil
			}
			if subagentTools[d.Tool] && d.Agent == reviewerAgent && d.Result == subagentResultOK && d.ToolUseID != "" {
				newest.ToolUseID, newest.Commit, newest.CommitError = d.ToolUseID, d.Commit, d.CommitError
			}
			return nil
		})
	// The newest ok reviewer row alone names the reviewed commit: an older row's match does not
	// revive a fact a newer review of another commit, or a failed bundle, left void.
	if err == nil && newest.CommitError == "" && newest.Commit != "" && newest.Commit == sha {
		id := newest.ToolUseID
		r.ToolUseID = &id
	}
	return r, err
}

// startedWorkflow is the session-started event's workflow and hooks details, which `session
// show` prints from the store: "" and nil for a session started before P19.
func startedWorkflow(q store.Querier, id string) (*string, []string, error) {
	var line *string
	var hooks []string
	err := q.Query(`SELECT details FROM audit_event WHERE session_id = ? AND kind = ? ORDER BY seq LIMIT 1`,
		[]store.Value{store.Text(id), store.Text(string(audit.KindSessionStarted))}, func(row *store.Row) error {
			var d struct{ Workflow, Hooks string }
			if json.Unmarshal([]byte(row.Text(0)), &d) != nil {
				return nil
			}
			if d.Workflow != "" {
				w := d.Workflow
				line = &w
			}
			if d.Hooks != "" {
				hooks = strings.Split(d.Hooks, hooksSep)
			}
			return nil
		})
	return line, hooks, err
}

// hooksSep joins the hooks: names in the session-started details and on the `hooks:` line.
const hooksSep = ", "

// Inspect is `session show` with what it reads beyond the store: the review fact from the
// canonical (the live session's, else its channel's project; a channel the configuration no
// longer holds is `review: error: unknown-channel`), and with header the header from the copy
// (`session show --header`).
func (s *Service) Inspect(ctx context.Context, id string, header bool) (Detail, error) {
	d, err := s.Show(id)
	if err != nil {
		return d, err
	}
	var gs git.Session
	if l := s.lookup(id); l != nil {
		gs = l.git
	} else if ch, found := s.cfg.Channel(d.Channel); found {
		gs = git.Session{Root: s.cfg.Root, ID: id, Instance: s.cfg.Instance, Canonical: ch.Repo.Canonical,
			Remote: orDefault(ch.Repo.Remote, "origin"), DefaultBranch: orDefault(ch.Repo.DefaultBranch, "main")}
	}
	if gs.ID == "" {
		d.Review = ReviewFailed(CodeUnknownChannel, "the configuration holds no channel "+d.Channel)
	} else if d.Review, err = ReviewFact(ctx, s.cfg.Store, gs); err != nil {
		return d, err
	}
	if header {
		if d.Header, err = ReadHeader(s.cfg.Root, id); err != nil {
			return d, err
		}
	}
	return d, nil
}
