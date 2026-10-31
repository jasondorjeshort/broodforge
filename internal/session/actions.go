// SPDX-License-Identifier: GPL-3.0-or-later

package session

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jasondorjeshort/broodforge/internal/audit"
	"github.com/jasondorjeshort/broodforge/internal/engine"
	"github.com/jasondorjeshort/broodforge/internal/git"
	"github.com/jasondorjeshort/broodforge/internal/policy"
	"github.com/jasondorjeshort/broodforge/internal/store"
)

// Who decided an answered call, beside classify.go's (02-domain.md, Permission decision).
const (
	DecidedByParticipant   = "participant"
	DecidedByPromptTimeout = "prompt-timeout"
	// DecidedByInstance is the bot itself: a call dropped because the session ended or the bot stopped.
	DecidedByInstance = "instance"
	// DecidedByHost is the owner on the host (no --as): an interrupt while a call waits.
	DecidedByHost = "host"
)

// Answer decisions `session answer` takes: 04's three buttons.
const (
	AnswerAllow        = "allow"
	AnswerDeny         = "deny"
	AnswerAllowSession = "allow-session"
)

// More refusal codes.
const (
	CodeNoPending    = "no-pending"
	CodeNotPending   = "not-pending"
	CodeNoRunning    = "no-running-turn"
	CodeHandoffState = "handoff-refused"
	CodeResumeFailed = "resume-failed"
)

// ActRequest is a command acting for a person on a session: --as and --roles (05).
type ActRequest struct {
	Session, As, Roles string
}

// AnswerRequest is `session answer`.
type AnswerRequest struct {
	ActRequest
	ToolUseID, Decision, Reason string
}

// answerRec is how a call was answered, for its result's trace line.
type answerRec struct {
	decision string
	trace    string // what 04's trace line appends: "denied: ...", "" for an allow
}

// open is the live session and the actor's display name, with an unknown or closed session
// refused; a registered --as is required to be one.
func (s *Service) acting(req ActRequest) (*live, string, error) {
	if s.closing.Load() {
		return nil, "", &Error{CodeShuttingDown, "the bot is stopping"}
	}
	l := s.lookup(req.Session)
	if l == nil {
		return nil, "", s.closedSession(req.Session)
	}
	if req.As == "" {
		return l, "the host", nil
	}
	reg, found, err := s.registration(req.As)
	if err != nil {
		return nil, "", err
	}
	if !found {
		return nil, "", s.refuse(req.As, l.id, "act", "unregistered", &Error{CodeUnregistered, req.As + " is not registered"})
	}
	return l, reg.Name, nil
}

// checkAnswerList is the role check for answer, interrupt and end: the channel's `answer`
// list, "driver" meaning the driver (R-25, Owner 8). Unchecked without --roles (05).
func (s *Service) checkAnswerList(l *live, req ActRequest, action string) error {
	if req.Roles == "" {
		return nil
	}
	isDriver := req.As == l.driver.User
	list := l.ch.Roles.Answer
	if policy.Permitted(splitRoles(req.Roles), list, s.groups(), isDriver) {
		return nil
	}
	msg := action + " needs a role in " + strings.Join(list, ",")
	if len(list) == 1 && list[0] == policy.RoleDriver {
		msg = "Only " + l.driver.Name + " (the driver) can " + action + " this." // 04, Permission prompts
	}
	return s.refuse(actorOf(req.As), l.id, action, "roles", &Error{CodeRefused, msg})
}

// checkDriver is mode's and invite's check: the driver only, when --as names someone.
func (s *Service) checkDriver(l *live, req ActRequest, action string) error {
	if req.As == "" || req.As == l.driver.User {
		return nil
	}
	return s.refuse(req.As, l.id, action, "driver", &Error{CodeRefused, action + " needs the driver, " + l.driver.Name})
}

// switchOpenLocked is the check mode's, model's and effort's switches share
// (05-debug-interface-session.md, Session lifecycle): a session that is ending, ended or
// interrupted refuses them with end's code and text, audited with the reason closed, and the
// caller writes nothing. l.mu is held, so the state read is the one the write would follow.
func (s *Service) switchOpenLocked(l *live, act ActRequest, action string) error {
	switch l.state {
	case StateEnding, StateEnded, StateInterrupted:
		return s.refuse(actorOf(act.As), l.id, action, "closed", &Error{CodeSessionClosed, l.id + " is " + l.state})
	}
	return nil
}

// openCheck is switchOpenLocked taking l.mu itself: the closed check answer, interrupt, invite
// and timeout run first of all their checks (ADR 0026), so nothing but end acts on a held
// session and a closed session's refusal is session-closed whoever asks.
func (s *Service) openCheck(l *live, act ActRequest, action string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return s.switchOpenLocked(l, act, action)
}

// Answer is SessionService.answer: the pending call's decision, by a participant the channel
// lets answer, then the turn resumes (03 Data flow, Prompt mode). allow-session also adds the
// tool to BROODFORGE_SESSION_ALLOW for the session's later calls (04, Owner 6).
func (s *Service) Answer(ctx context.Context, req AnswerRequest) (Submitted, error) {
	switch req.Decision {
	case AnswerAllow, AnswerDeny, AnswerAllowSession:
	default:
		return Submitted{}, badArgs("answer " + quote(req.Decision) + ": want allow, deny or allow-session")
	}
	// The hook hands the agent a reason with a deny only (BROODFORGE_ANSWER_REASON); an allow's
	// reason would reach nobody, so it is refused rather than dropped.
	if req.Reason != "" && req.Decision != AnswerDeny {
		return Submitted{}, badArgs("--reason goes with deny: the agent sees no reason for an allow")
	}
	l, who, err := s.acting(req.ActRequest)
	if err != nil {
		return Submitted{}, err
	}
	if err := s.openCheck(l, req.ActRequest, "answer"); err != nil {
		return Submitted{}, err
	}
	if err := s.checkAnswerList(l, req.ActRequest, "answer"); err != nil {
		return Submitted{}, err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if s.closing.Load() {
		return Submitted{}, &Error{CodeShuttingDown, "the bot is stopping"}
	}
	if l.pending == nil {
		return Submitted{}, &Error{CodeNoPending, l.id + " is " + l.state + " and waits on no call"}
	}
	if req.ToolUseID != "" && req.ToolUseID != l.pending.ToolUseID {
		return Submitted{}, &Error{CodeNotPending, req.ToolUseID + " is not the call " + l.id + " waits on (" + l.pending.ToolUseID + ")"}
	}
	seq := l.log.last()
	if err := s.answerLocked(l, req.Decision, DecidedByParticipant, actorOf(req.As), req.As, who, req.Reason); err != nil {
		return Submitted{}, err
	}
	return Submitted{Turn: l.turn, State: SubmitResumed, Seq: seq}, nil
}

// answerLocked records the answer to the pending call (its row and audit event, the decider a
// participant or the prompt timeout) and resumes the turn with it. l.mu is held.
func (s *Service) answerLocked(l *live, decision, decidedBy, actor, user, who, reason string) error {
	p := l.pending
	c := l.calls[p.ToolUseID]
	dec := DecisionAllow
	trace := ""
	if decision == AnswerDeny {
		dec = DecisionDeny
		switch {
		case decidedBy == DecidedByPromptTimeout:
			trace = "denied: " + reason + "." // 04: `-> denied: the prompt timed out after 5 minutes.`
		case reason != "":
			trace = "denied by " + who + ": " + reason
		default:
			trace = "denied by " + who
		}
	}
	err := s.cfg.Store.Write(func(tx *store.Tx) error {
		return s.decisionTx(tx, l, l.turn, p.ToolUseID, c, dec, decidedBy, actor, user, reason, "")
	})
	if err != nil {
		return err
	}
	if decision == AnswerAllowSession && !slices.Contains(l.sessionAllow, c.tool) {
		l.sessionAllow = append(l.sessionAllow, c.tool)
	}
	l.answers[p.ToolUseID] = answerRec{decision: dec, trace: trace}
	l.log.add(Event{Time: s.stamp(), Type: EventPermissionDecision, Turn: l.turn, User: user, ToolUseID: p.ToolUseID,
		Tool: c.tool, Target: c.target, Decision: dec, DecidedBy: decidedBy, Reason: reason,
		Summary: c.indent(dec + " " + TraceLine(c.tool, c.target) + " (" + decidedBy + ")")})
	l.pending = nil
	s.disarmPromptLocked(l)
	hookReason := ""
	if dec == DecisionDeny {
		hookReason = reason
	}
	if err := s.startProcessLocked(l, "", &engine.TurnAnswer{ToolUseID: p.ToolUseID, Decision: dec}, hookReason); err != nil {
		// The turn ended as an error (failTurnLocked); the answer stands in the audit log.
		return &Error{CodeResumeFailed, "the answer is recorded but the turn could not resume: " + err.Error()}
	}
	return nil
}

// Interrupt is SessionService.interrupt (04, Stop): the running turn's engine is interrupted
// through the backend and the turn ends; it returns the turn-ended event. Its usage is the
// deduplicated assistant counts, never a baseline (S2).
func (s *Service) Interrupt(ctx context.Context, req ActRequest) (Event, error) {
	l, _, err := s.acting(req)
	if err != nil {
		return Event{}, err
	}
	if err := s.openCheck(l, req, "interrupt"); err != nil {
		return Event{}, err
	}
	if err := s.checkAnswerList(l, req, "interrupt"); err != nil {
		return Event{}, err
	}
	l.mu.Lock()
	if l.state == StateWaitingForAnswer && l.pending != nil {
		// Stop while a call waits (the coordinator's default until the owner decides): the
		// call is denied with the reason `interrupted` and the turn ends, without a resume.
		defer l.mu.Unlock()
		p, n := l.pending, l.turn
		c := l.calls[p.ToolUseID]
		decidedBy := DecidedByParticipant
		if req.As == "" {
			decidedBy = DecidedByHost
		}
		if err := s.cfg.Store.Write(func(tx *store.Tx) error {
			return s.decisionTx(tx, l, n, p.ToolUseID, c, DecisionDeny, decidedBy, actorOf(req.As), req.As, ResultInterrupted, "")
		}); err != nil {
			return Event{}, err
		}
		l.log.add(Event{Time: s.stamp(), Type: EventPermissionDecision, Turn: n, User: req.As, ToolUseID: p.ToolUseID,
			Tool: c.tool, Target: c.target, Decision: DecisionDeny, DecidedBy: decidedBy, Reason: ResultInterrupted,
			Summary: c.indent(DecisionDeny + " " + TraceLine(c.tool, c.target) + " (" + decidedBy + ")")})
		ended := s.endTurnLocked(l, n, nil, engine.TurnUsage{}, "")
		ended.User = actorOf(req.As)
		// settleLocked moves a turn that ran on: back to waiting for input, or the queued turn.
		l.state = StateRunningTurn
		s.settleLocked(l, ended)
		return ended, nil
	}
	if l.running == nil || l.state != StateRunningTurn {
		l.mu.Unlock()
		return Event{}, &Error{CodeNoRunning, l.id + " is " + l.state}
	}
	h, n, seq := *l.running, l.turn, l.log.last()
	l.interruptedBy = actorOf(req.As)
	l.mu.Unlock()
	if err := h.Interrupt(); err != nil {
		return Event{}, err
	}
	var ended Event
	err = l.log.follow(ctx, seq, func(ev Event) (bool, error) {
		if ev.Type == EventTurnEnded && ev.Turn == n {
			ended = ev
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return Event{}, err
	}
	return ended, nil
}

// ModeRequest is `session mode`.
type ModeRequest struct {
	ActRequest
	Mode string
}

// SetMode is SessionService.setMode: the driver's switch, effective at the next turn (ADR 0020).
func (s *Service) SetMode(req ModeRequest) (Event, error) {
	if _, ok := policy.ParseMode(req.Mode); !ok {
		return Event{}, badArgs("mode " + quote(req.Mode) + ": want prompt or policy")
	}
	l, _, err := s.acting(req.ActRequest)
	if err != nil {
		return Event{}, err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := s.switchOpenLocked(l, req.ActRequest, "mode"); err != nil {
		return Event{}, err
	}
	if err := s.checkDriver(l, req.ActRequest, "mode"); err != nil {
		return Event{}, err
	}
	from := l.mode
	err = s.cfg.Store.Write(func(tx *store.Tx) error {
		if _, err := tx.Exec("UPDATE session SET mode = ? WHERE id = ?", store.Text(req.Mode), store.Text(l.id)); err != nil {
			return err
		}
		_, err := s.cfg.Audit.Append(tx, audit.Event{Kind: audit.KindModeChanged, Actor: actorOf(req.As), Session: l.id,
			Details: details("from", from, "to", req.Mode)})
		return err
	})
	if err != nil {
		return Event{}, err
	}
	l.mode = req.Mode
	// 04-ux-session-controls.md, Policy mode: the names people see are `asks first` and `acts freely`.
	shown := "asks first"
	if req.Mode == policy.ModePolicy.String() {
		shown = "acts freely"
	}
	return l.log.add(Event{Time: s.stamp(), Type: EventModeChanged, Turn: l.turn + 1, User: req.As, Result: req.Mode,
		Summary: "From the next turn: " + shown + "."}), nil
}

// InviteRequest is `session invite`.
type InviteRequest struct {
	ActRequest
	User string
}

// Invite is SessionService.invite: the driver grants the venue to a registered user. They
// become a participant on their first message (R-13), so a co-author line is never owed for
// someone who only was invited. Their roles are Discord's to check (P12).
func (s *Service) Invite(req InviteRequest) (Registration, error) {
	l, _, err := s.acting(req.ActRequest)
	if err != nil {
		return Registration{}, err
	}
	if err := s.openCheck(l, req.ActRequest, "invite"); err != nil {
		return Registration{}, err
	}
	if err := s.checkDriver(l, req.ActRequest, "invite"); err != nil {
		return Registration{}, err
	}
	reg, err := ShowRegistration(s.cfg.Store, req.User)
	if err != nil {
		return Registration{}, err
	}
	err = s.cfg.Store.Write(func(tx *store.Tx) error {
		_, err := s.cfg.Audit.Append(tx, audit.Event{Kind: audit.KindParticipantInvited, Actor: actorOf(req.As), Session: l.id,
			Registration: req.User, Details: details("user", req.User)})
		return err
	})
	if err != nil {
		return Registration{}, err
	}
	l.log.add(Event{Time: s.stamp(), Type: EventParticipantInvited, User: req.User, Summary: reg.Name + " was invited."})
	return reg, nil
}

// EndRequest is `session end`.
type EndRequest struct {
	ActRequest
	NoClosingTurn bool
}

// End is SessionService.end (03, End of session): a running turn is interrupted, a pending
// call dropped, then the handoff. A refused handoff (bundle, verify, fetch, attribution) keeps
// the session open, waiting for input, so the driver can have the commits amended; anything
// else ends it, a failed push keeping the tree. On a held interrupted session (ADR 0026) it is
// the one action accepted: the escapes first (heldEscape), else the same handoff, a refusal
// returning the session to interrupted.
func (s *Service) End(ctx context.Context, req EndRequest) (git.Outcome, error) {
	l, who, err := s.acting(req.ActRequest)
	if err != nil {
		return git.Outcome{}, err
	}
	if err := s.checkAnswerList(l, req.ActRequest, "end"); err != nil {
		return git.Outcome{}, err
	}
	summary := "Session " + l.id + " ended by " + who + "." // 04, Ending a session
	return s.end(ctx, l, actorOf(req.As), who, "end", summary, noGen)
}

// end runs the end-of-session sequence for an actor and a reason (end, idle).
//
// The closing turn that writes the pull request's title and body (03; 04 Owner 12) is not run:
// it needs a `closing` fixture for the fake and P11's summary to show it; the fallback title
// stands (P11).
// noGen is end's idle generation for a caller that is not the idle timer.
const noGen = -1

func (s *Service) end(ctx context.Context, l *live, actor, who, reason, summary string, idleGen int) (git.Outcome, error) {
	l.mu.Lock()
	switch l.state {
	case StateEnded, StateEnding:
		l.mu.Unlock()
		return git.Outcome{}, &Error{CodeSessionClosed, l.id + " is " + l.state}
	case StateInterrupted:
		// Only a held session is ended, and never by the idle timer, which it does not run.
		if !l.held || idleGen != noGen {
			l.mu.Unlock()
			return git.Outcome{}, &Error{CodeSessionClosed, l.id + " is " + l.state}
		}
	}
	// The idle timer's end is checked under the same lock it acts in: a message or an answer
	// since the tick (a new generation, or a turn running) means there was activity after all.
	if idleGen != noGen && (idleGen != l.idleGen || (l.state != StateWaitingForInput && l.state != StateWaitingForAnswer)) {
		l.mu.Unlock()
		return git.Outcome{}, &Error{CodeSessionClosed, l.id + " had activity: the idle timeout is void"}
	}
	before := l.state
	l.state = StateEnding
	_ = s.setState(l.id, StateEnding)
	s.disarmIdleLocked(l)
	s.disarmPromptLocked(l)
	s.discardLocked(l, actor, endCause(reason))
	h := l.running
	done := l.turnDone
	l.mu.Unlock()
	if h != nil {
		_ = h.Interrupt()
	}
	if done != nil {
		select {
		case <-done:
		case <-ctx.Done():
			return git.Outcome{}, ctx.Err()
		}
	}
	var out git.Outcome
	if before == StateInterrupted {
		if esc, ok := s.heldEscape(ctx, l); ok {
			s.closeSession(l, actor, reason, before, esc.Outcome, summary, esc, who)
			return esc, nil
		}
	}
	if s.cfg.Handoff != nil {
		s.ending(ctx, l)
		out, err := s.cfg.Handoff(ctx, l.id)
		if err != nil {
			var ge *git.Error
			if errors.As(err, &ge) && strings.HasPrefix(ge.Code, CodeHandoffState) && before == StateInterrupted {
				// Refused on a held session: back to interrupted, no timer, no new turn; no agent
				// can amend the commits, so 04's sentence names the owner's retry.
				l.mu.Lock()
				l.state = StateInterrupted
				_ = s.setState(l.id, StateInterrupted)
				text := strings.TrimSuffix(ge.Message, ".") + ". " + HeldHandoffSentence(l.id)
				l.log.add(Event{Time: s.stamp(), Type: EventHandoffRefused, Result: ge.Code, Summary: firstRunes(ge.Message, summaryRunes), Text: text})
				l.mu.Unlock()
				return out, err
			}
			if errors.As(err, &ge) && strings.HasPrefix(ge.Code, CodeHandoffState) {
				// Refused: the session stays open; the driver may amend and end again. An idle
				// end tries once: the timer waits for the next activity, not the next period.
				l.mu.Lock()
				l.state = StateWaitingForInput
				_ = s.setState(l.id, StateWaitingForInput)
				if reason != "idle" {
					s.armIdleLocked(l)
				}
				l.log.add(Event{Time: s.stamp(), Type: EventHandoffRefused, Result: ge.Code, Summary: firstRunes(ge.Message, summaryRunes), Text: ge.Message})
				l.mu.Unlock()
				return out, err
			}
			s.closeSession(l, actor, reason, before, "failed: "+errCode(err), summary, out, who)
			return out, err
		}
		s.closeSession(l, actor, reason, before, out.Outcome, summary, out, who)
		return out, nil
	}
	out.Outcome = git.OutcomeNone
	s.closeSession(l, actor, reason, before, out.Outcome, summary, out, who)
	return out, nil
}

// ending publishes session-ending before the handoff runs: the tree's commit count over its base
// through Config.Commits, nil when there is none or it cannot be read.
func (s *Service) ending(ctx context.Context, l *live) {
	var n *int
	if s.cfg.Commits != nil {
		if c, err := s.cfg.Commits(ctx, l.id); err == nil {
			n = c
		}
	}
	summary := "commits unknown"
	if n != nil {
		summary = strconv.Itoa(*n) + " commits"
		if *n == 1 {
			summary = "1 commit"
		}
	}
	l.log.add(Event{Time: s.stamp(), Type: EventSessionEnding, Summary: summary, Commits: n})
}

// OutcomeTreeMissing is the end of a held session whose tree is gone while refs/sessions/<id> is
// still in the canonical: no handoff, the ref left for the owner (ADR 0026).
const OutcomeTreeMissing = "tree-missing"

// HeldHandoffSentence ends a refused handoff's text on a held session (04, Failure flows:
// handoff-refused, interrupted session; owner-approved 2026-10-24).
func HeldHandoffSentence(id string) string {
	return "The session was interrupted, so no agent can amend them; the owner can fix the tree and retry with broodforge worktree handoff " + id + "."
}

// heldEscape is end's two escapes on a held session, in 05's order, so every held session can be
// ended (ADR 0026): refs/sessions/<id> gone from the canonical (a worktree handoff finished
// earlier) ends with the outcome that handoff recorded in end_outcome, a URL as pull-request with
// the URL, a branch as pushed, null as none; else the tree gone ends tree-missing, the ref
// present or the canonical unreadable (a handoff would refuse the bundle on every attempt).
// Neither runs a handoff; the commit count is unknown, the remote and the mode the project's.
// false when neither applies: then the handoff runs and reports what it finds.
func (s *Service) heldEscape(ctx context.Context, l *live) (git.Outcome, bool) {
	mode := l.ch.Repo.Handoff
	_, ref, err := git.SessionTip(ctx, l.git)
	if err == nil && !ref {
		var stored *string
		if err := s.cfg.Store.Query("SELECT end_outcome FROM session WHERE id = ?", []store.Value{store.Text(l.id)},
			func(r *store.Row) error {
				if !r.IsNull(0) {
					v := r.Text(0)
					stored = &v
				}
				return nil
			}); err != nil {
			return git.Outcome{}, false
		}
		out := git.Outcome{Outcome: git.OutcomeNone, Mode: mode, Remote: l.git.Remote, Commits: CommitsUnknown}
		switch {
		case stored == nil:
		case strings.Contains(*stored, "://"):
			// The handoff records a URL in pull-request mode and the branch in push mode.
			out.Outcome, out.Mode, out.URL = git.OutcomePullRequest, git.ModePullRequest, stored
		default:
			out.Outcome, out.Mode = git.OutcomePushed, git.ModePush
		}
		return out, true
	}
	state := ""
	if err := s.cfg.Store.Query("SELECT state FROM worktree WHERE session_id = ? AND parent_path IS NULL ORDER BY created_at LIMIT 1",
		[]store.Value{store.Text(l.id)}, func(r *store.Row) error { state = r.Text(0); return nil }); err != nil {
		return git.Outcome{}, false
	}
	if treeGone(l.tree, state) {
		return git.Outcome{Outcome: OutcomeTreeMissing, Mode: mode, Remote: l.git.Remote, Commits: CommitsUnknown}, true
	}
	return git.Outcome{}, false
}

// endCause is the reason a dropped call or message names.
func endCause(reason string) string {
	switch reason {
	case "idle":
		return "the session ended: no activity"
	case "shutdown":
		return "the bot stopped"
	}
	return "the session ended"
}

// discardLocked closes what an ending session leaves open (end, idle, stop): a call waiting
// for an answer is recorded `dropped` with the cause and its turn ends (`turn-ended`, result
// interrupted); every queued message is dropped with a `message-dropped` event and an
// action-refused audit row. l.mu is held.
func (s *Service) discardLocked(l *live, actor, cause string) {
	if p := l.pending; p != nil {
		c := l.calls[p.ToolUseID]
		if err := s.cfg.Store.Write(func(tx *store.Tx) error {
			return s.decisionTx(tx, l, l.turn, p.ToolUseID, c, DecisionDropped, DecidedByInstance, ActorInstance, "", cause+" before an answer", "")
		}); err != nil {
			l.log.add(storeFailed(s.stamp(), l.turn, p.ToolUseID, c, err))
		}
		l.log.add(Event{Time: s.stamp(), Type: EventPermissionDecision, Turn: l.turn, ToolUseID: p.ToolUseID, Tool: c.tool,
			Target: c.target, Decision: DecisionDropped, DecidedBy: DecidedByInstance, Reason: cause,
			Summary: c.indent(DecisionDropped + " " + TraceLine(c.tool, c.target) + " (" + DecidedByInstance + ")")})
		ended := s.endTurnLocked(l, l.turn, nil, engine.TurnUsage{}, "")
		ended.Summary = cause
		l.log.add(ended)
	}
	for _, q := range l.queue {
		if err := s.cfg.Store.Write(func(tx *store.Tx) error {
			_, err := s.cfg.Audit.Append(tx, audit.Event{Kind: audit.KindActionRefused, Actor: actor, Session: l.id, Registration: q.user,
				Details: details("action", "message", "reason", "message-dropped", "message", cause, "text", firstRunes(q.text, summaryRunes))})
			return err
		}); err != nil {
			l.log.add(storeFailed(s.stamp(), l.turn, "", call{}, err))
		}
		l.log.add(Event{Time: s.stamp(), Type: EventMessageDropped, User: q.user, Reason: cause,
			Summary: cause + ": " + firstRunes(q.text, summaryRunes)})
	}
	l.queue = nil
}

func errCode(err error) string {
	var ge *git.Error
	if errors.As(err, &ge) {
		return ge.Code
	}
	return "error"
}

// closeSession marks the session ended with its audit event and closes its log.
func (s *Service) closeSession(l *live, actor, reason, before, outcome, summary string, out git.Outcome, by string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	info := s.endInfo(l, reason, outcome, out, by)
	now := s.stamp()
	_ = s.cfg.Store.Write(func(tx *store.Tx) error {
		if _, err := tx.Exec("UPDATE session SET state = ?, ended_at = ?, end_outcome = coalesce(end_outcome, ?) WHERE id = ?",
			store.Text(StateEnded), store.Text(now), store.Text(outcome), store.Text(l.id)); err != nil {
			return err
		}
		_, err := s.cfg.Audit.Append(tx, audit.Event{Kind: audit.KindSessionEnded, Actor: actor, Session: l.id,
			Details: details("reason", reason, "outcome", outcome, "state", before)})
		return err
	})
	l.state = StateEnded
	l.log.add(Event{Time: now, Type: EventSessionEnded, Result: outcome, Summary: summary, End: info})
	l.log.close()
}

// The timers. Each arm bumps a generation, so a timer that fires after a disarm or a re-arm
// does nothing; the callbacks take l.mu themselves.

func (s *Service) armIdleLocked(l *live) {
	s.disarmIdleLocked(l)
	gen := l.idleGen
	l.idleTimer = s.cfg.Clock.AfterFunc(effectiveTimeout(l.idleTimeout), func() { s.idleFired(l, gen) })
}

func (s *Service) disarmIdleLocked(l *live) {
	l.idleGen++
	if l.idleTimer != nil {
		l.idleTimer.Stop()
		l.idleTimer = nil
	}
}

// idleFired is the idle timeout (R-15): the end-of-session sequence, actor the instance.
func (s *Service) idleFired(l *live, gen int) {
	l.mu.Lock()
	stale := gen != l.idleGen || (l.state != StateWaitingForInput && l.state != StateWaitingForAnswer)
	idle := effectiveTimeout(l.idleTimeout)
	l.mu.Unlock()
	if stale || s.closing.Load() {
		return
	}
	// 04, Failure flows (session-ended).
	summary := "Session " + l.id + " ended: no activity for " + humanDuration(idle) + "."
	_, _ = s.end(context.Background(), l, ActorInstance, "", "idle", summary, gen)
}

func (s *Service) armPromptLocked(l *live) {
	s.disarmPromptLocked(l)
	gen := l.promptGen
	l.promptTimer = s.cfg.Clock.AfterFunc(effectiveTimeout(l.promptTimeout), func() { s.promptFired(l, gen) })
}

func (s *Service) disarmPromptLocked(l *live) {
	l.promptGen++
	if l.promptTimer != nil {
		l.promptTimer.Stop()
		l.promptTimer = nil
	}
}

// promptFired is the prompt timeout (R-53, Owner 16): the call is denied with 04's reason,
// which the hook returns verbatim (BROODFORGE_ANSWER_REASON), and the turn finishes.
func (s *Service) promptFired(l *live, gen int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if gen != l.promptGen || l.pending == nil || s.closing.Load() {
		return
	}
	reason := "the prompt timed out after " + humanDuration(effectiveTimeout(l.promptTimeout))
	_ = s.answerLocked(l, AnswerDeny, DecidedByPromptTimeout, ActorInstance, "", "", reason)
}

// humanDuration is a timeout in 04's words: `5 minutes`, `1 minute`, `90 seconds`.
func humanDuration(d time.Duration) string {
	unit, n := "second", int64(d/time.Second)
	if d >= time.Minute && d%time.Minute == 0 {
		unit, n = "minute", int64(d/time.Minute)
	}
	if n != 1 {
		unit += "s"
	}
	return strconv.FormatInt(n, 10) + " " + unit
}

// ReachRow is one policy row of `session reach-report`: the tools, the cell for a path under
// the tree or the canonical, and the cell for one elsewhere (03-hook-contract.md, Policy table).
type ReachRow struct {
	Tools     []string `json:"tools"`
	InTree    string   `json:"inTree"`
	Elsewhere string   `json:"elsewhere"`
}

// Bind is one bind of the spawn's sandbox: none until the linux-sudo-bwrap backend (P14).
type Bind struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Mode   string `json:"mode"`
}

// ReachReport is the contract's column for the session's reach beside the bind list.
type ReachReport struct {
	Session string     `json:"session"`
	Reach   string     `json:"reach"`
	Policy  []ReachRow `json:"policy"`
	Binds   []Bind     `json:"binds"`
}

// Reach is `session reach-report <id>`, read from the store and policy's table.
func Reach(q store.Querier, id string) (ReachReport, error) {
	d, err := Show(q, id)
	if err != nil {
		return ReachReport{}, err
	}
	r, ok := policy.ParseReach(d.Reach)
	if !ok {
		return ReachReport{}, &Error{CodeBadArgs, id + " has reach " + d.Reach}
	}
	rep := ReachReport{Session: id, Reach: d.Reach, Binds: []Bind{}}
	in := []policy.Path{{Target: "x", UnderTree: true}}
	out := []policy.Path{{Target: "x"}}
	for _, row := range policy.ClassRows() {
		tools := row.Tools
		if tools == nil {
			tools = []string{"*"}
		}
		rep.Policy = append(rep.Policy, ReachRow{Tools: tools, InTree: policy.CellFor(row.Class, r, in).String(),
			Elsewhere: policy.CellFor(row.Class, r, out).String()})
	}
	return rep, nil
}

// endInfo is the summary's facts for a session ending (04, Ending a session): who, how long,
// the turns and tokens, the branch and its handoff, and the people and models credited (the
// trailers' names, 03 Attribution). l.mu is held.
func (s *Service) endInfo(l *live, reason, outcome string, out git.Outcome, by string) *EndInfo {
	info := &EndInfo{By: by, Reason: reason, StartedAt: l.startedAt, Turns: l.turn, In: l.sessIn, Out: l.sessOut,
		Model: topModel(l.sessionModels, l.model), Branch: l.git.Branch(), Outcome: outcome, Commits: out.Commits,
		Author: l.driver.Name, CoAuthors: []string{}}
	// The opened request's web URL (04's `pull request #42 <url>`); before it opens, as on a
	// failed request, the API URL it was posted to.
	switch {
	case out.URL != nil:
		info.URL = *out.URL
	case out.Request != nil:
		info.URL = out.Request.URL
	}
	if out.Request != nil {
		info.Title = out.Request.Body.Title
	}
	for _, c := range out.Log {
		info.Log = append(info.Log, CommitLine{Hash: c.Hash, Subject: c.Subject})
	}
	lines, err := s.trailers(l)
	if err == nil {
		for _, line := range lines {
			name := strings.TrimSpace(strings.TrimPrefix(line, git.TrailerKey+":"))
			if i := strings.LastIndex(name, " <"); i > 0 {
				name = name[:i]
			}
			info.CoAuthors = append(info.CoAuthors, name)
		}
	}
	return info
}
