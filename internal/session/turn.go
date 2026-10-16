// SPDX-License-Identifier: GPL-3.0-or-later

package session

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/jasondorjeshort/broodforge/internal/audit"
	"github.com/jasondorjeshort/broodforge/internal/engine"
	"github.com/jasondorjeshort/broodforge/internal/git"
	"github.com/jasondorjeshort/broodforge/internal/policy"
	"github.com/jasondorjeshort/broodforge/internal/store"
)

// SendRequest is `session send` (05-debug-interface-session.md).
type SendRequest struct {
	Session, As, Roles, Text string
}

// Submitted is what submit did with the message: the turn it starts or joins, and whether
// that turn started now or the message waits for it. Seq is the session's last event before
// the message, where a follower of this turn starts.
type Submitted struct {
	Turn  int    `json:"turn"`
	State string `json:"state"` // started or queued
	Seq   int64  `json:"-"`
}

// Submitted states.
const (
	SubmitStarted = "started"
	SubmitQueued  = "queued"
)

// closedSession is the refusal for a session the bot no longer holds open.
func (s *Service) closedSession(id string) error {
	state := ""
	err := s.cfg.Store.Query("SELECT state FROM session WHERE id = ?", []store.Value{store.Text(id)}, func(r *store.Row) error {
		state = r.Text(0)
		return nil
	})
	if err != nil {
		return err
	}
	if state == "" {
		return &Error{CodeUnknownSession, "no session " + id}
	}
	return &Error{CodeSessionClosed, id + " is " + state}
}

// Submit is SessionService.submit (03 Data flow, step 2): the writer is checked, joins as a
// contributor on their first accepted message (R-13), and the message is one From block. An
// idle session starts a turn; a busy one queues the message for the next (ADR 0020).
func (s *Service) Submit(ctx context.Context, req SendRequest) (Submitted, error) {
	actor := actorOf(req.As)
	if s.closing.Load() {
		return Submitted{}, &Error{CodeShuttingDown, "the bot is stopping"}
	}
	l := s.lookup(req.Session)
	if l == nil {
		return Submitted{}, s.closedSession(req.Session)
	}
	if req.As == "" {
		return Submitted{}, badArgs("session send needs --as <user id>: every message is handed to the agent by its writer's name")
	}
	if strings.TrimSpace(req.Text) == "" {
		return Submitted{}, badArgs("session send needs a message")
	}
	reg, found, err := s.registration(req.As)
	if err != nil {
		return Submitted{}, err
	}
	if !found {
		return Submitted{}, s.refuse(actor, l.id, "send", "unregistered", &Error{CodeUnregistered, req.As + " is not registered"})
	}
	isDriver := req.As == l.driver.User
	if req.Roles != "" && !isDriver && !policy.Permitted(splitRoles(req.Roles), l.ch.Roles.Contribute, s.cfg.RoleGroups, false) {
		return Submitted{}, s.refuse(actor, l.id, "send", "roles",
			&Error{CodeRefused, "contribute needs a role in " + strings.Join(l.ch.Roles.Contribute, ",")})
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	// Again under l.mu, which Shutdown takes after setting closing to see whether a turn runs:
	// a turn either starts before that look, and is interrupted, or not at all.
	if s.closing.Load() {
		return Submitted{}, &Error{CodeShuttingDown, "the bot is stopping"}
	}
	switch l.state {
	case StateWaitingForInput, StateRunningTurn, StateWaitingForAnswer:
	default:
		return Submitted{}, &Error{CodeSessionClosed, l.id + " is " + l.state}
	}
	seq := l.log.last()
	role, joined := l.participants[req.As]
	if !joined {
		role = RoleContributor
		now := s.stamp()
		if err := s.cfg.Store.Write(func(tx *store.Tx) error {
			_, err := tx.Exec("INSERT INTO participant (session_id, user_id, role, joined_at) VALUES (?, ?, ?, ?)",
				store.Text(l.id), store.Text(req.As), store.Text(role), store.Text(now))
			return err
		}); err != nil {
			return Submitted{}, err
		}
		l.participants[req.As] = role
		// 04, Starting and the first message.
		l.log.add(Event{Time: now, Type: EventParticipantJoined, User: req.As, Summary: reg.Name + " joined as a contributor."})
	}
	block := FromBlock(reg.Name, role, req.Text)
	if l.state == StateWaitingForInput {
		n, err := s.startTurnLocked(l, []queued{{user: req.As, block: block}})
		if err != nil {
			return Submitted{}, err
		}
		return Submitted{Turn: n, State: SubmitStarted, Seq: seq}, nil
	}
	l.queue = append(l.queue, queued{user: req.As, block: block})
	l.log.add(Event{Time: s.stamp(), Type: EventMessageQueued, Turn: l.turn + 1, User: req.As,
		Summary: firstRunes(req.Text, summaryRunes)})
	return Submitted{Turn: l.turn + 1, State: SubmitQueued, Seq: seq}, nil
}

// trailers are the co-author lines a commit made now carries (03 Git, Attribution): the
// other participants in join order, then every model with usage, by display name.
func (s *Service) trailers(l *live) ([]string, error) {
	var lines []string
	err := s.cfg.Store.Query(`SELECT r.git_name, r.git_email FROM participant p JOIN registration r ON r.user_id = p.user_id
WHERE p.session_id = ? AND p.user_id != ? ORDER BY p.joined_at, p.rowid`,
		[]store.Value{store.Text(l.id), store.Text(l.driver.User)}, func(r *store.Row) error {
			lines = append(lines, git.Identity{Name: r.Text(0), Email: r.Text(1)}.Trailer())
			return nil
		})
	if err != nil {
		return nil, err
	}
	err = s.cfg.Store.Query(`SELECT model FROM usage_record WHERE session_id = ?
    AND input_tokens + output_tokens + cache_read_tokens + cache_write_tokens > 0
GROUP BY model ORDER BY min(recorded_at), model`, []store.Value{store.Text(l.id)}, func(r *store.Row) error {
		lines = append(lines, git.Identity{Name: policy.ModelDisplayName(r.Text(0)), Email: git.ModelEmail}.Trailer())
		return nil
	})
	return lines, err
}

// startTurnLocked starts the next turn from the messages given; l.mu is held. The turn row is
// written (started) before the engine starts (03 Data flow, step 3); the turn's goroutine
// owns its events from then on.
func (s *Service) startTurnLocked(l *live, msgs []queued) (int, error) {
	n := l.turn + 1
	blocks := make([]string, len(msgs))
	for i, m := range msgs {
		blocks[i] = m.block
	}
	input := JoinBlocks(blocks)
	lines, err := s.trailers(l)
	if err != nil {
		return 0, s.turnNotStarted(l, n, msgs, err)
	}
	if err := git.WriteTrailers(l.git.Dir(), lines); err != nil {
		return 0, s.turnNotStarted(l, n, msgs, err)
	}
	now := s.stamp()
	// The requester is the first block's writer (R-44); every writer is in the input's From lines.
	if err := s.cfg.Store.Write(func(tx *store.Tx) error {
		if _, err := tx.Exec("INSERT INTO turn (session_id, seq, requester_id, input, started_at) VALUES (?, ?, ?, ?, ?)",
			store.Text(l.id), store.Int(int64(n)), store.Text(msgs[0].user), store.Text(input), store.Text(now)); err != nil {
			return err
		}
		_, err := tx.Exec("UPDATE session SET state = ? WHERE id = ?", store.Text(StateRunningTurn), store.Text(l.id))
		return err
	}); err != nil {
		return 0, s.turnNotStarted(l, n, msgs, err)
	}
	l.turn, l.state = n, StateRunningTurn
	l.log.add(Event{Time: now, Type: EventTurnStarted, Turn: n, User: msgs[0].user, Summary: firstRunes(input, summaryRunes)})
	req := engine.TurnRequest{
		SessionID:       l.id,
		EngineSessionID: l.engineSessionID,
		Cwd:             l.tree,
		Delivery:        l.delivery,
		GitIdentity:     engine.GitIdentity{Name: l.driver.Name, Email: l.driver.Email},
		HookEnv: []engine.HookVar{
			{Name: envSession, Value: l.id},
			{Name: envMode, Value: l.mode},
			{Name: envReach, Value: l.reach},
			{Name: envTree, Value: l.tree},
			{Name: envCanonical, Value: l.git.Canonical},
			{Name: envHookLog, Value: l.home + "/hook.log"},
		},
		Input:              input,
		Model:              l.ch.Model,
		Effort:             l.ch.Effort,
		SystemPromptAppend: SystemPromptAppend(l.id, l.ch.ProjectName, l.reach, l.driver.Name, lines),
	}
	h, err := l.eng.StartTurn(req)
	if err != nil {
		ended := s.endTurnLocked(l, n, nil, err.Error())
		s.auditTurnFailure(l, n, msgs, err)
		l.state = StateWaitingForInput
		_ = s.setState(l.id, StateWaitingForInput)
		l.log.add(ended)
		return n, nil
	}
	l.running = &h
	done := make(chan struct{})
	l.turnDone = done
	go s.runTurn(l, n, h, done)
	return n, nil
}

// turnNotStarted is a turn that failed before its row was written: it publishes turn-ended
// with the failure, so a `send --wait` on it returns, audits it, and returns err. l.mu is held.
func (s *Service) turnNotStarted(l *live, n int, msgs []queued, err error) error {
	s.auditTurnFailure(l, n, msgs, err)
	if l.state == StateRunningTurn { // the turn before it, when it was queued
		l.state = StateWaitingForInput
		_ = s.setState(l.id, StateWaitingForInput)
	}
	l.log.add(Event{Time: s.stamp(), Type: EventTurnEnded, Turn: n, Result: ResultError,
		Summary: "the turn could not start: " + firstRunes(err.Error(), summaryRunes)})
	return err
}

// auditTurnFailure writes action-refused for a turn that did not start, actor its requester.
func (s *Service) auditTurnFailure(l *live, n int, msgs []queued, err error) {
	_ = s.cfg.Store.Write(func(tx *store.Tx) error {
		_, aerr := s.cfg.Audit.Append(tx, audit.Event{Kind: audit.KindActionRefused, Actor: actorOf(msgs[0].user), Session: l.id,
			Details: details("action", "turn", "turn", strconv.Itoa(n), "reason", "turn-start", "message", err.Error())})
		return aerr
	})
}

func (s *Service) setState(id, state string) error {
	return s.cfg.Store.Write(func(tx *store.Tx) error {
		_, err := tx.Exec("UPDATE session SET state = ? WHERE id = ?", store.Text(state), store.Text(id))
		return err
	})
}

// call is a tool call the turn has seen, for its result's trace line and decision row.
type call struct {
	tool, target string
}

// runTurn reads one turn's engine events to the end (03 Data flow, steps 4 to 7) and then
// starts the next queued turn, if any. It is the only goroutine reading this handle.
func (s *Service) runTurn(l *live, n int, h engine.TurnHandle, done chan struct{}) {
	defer close(done)
	var endEv *Event // the turn-ended event, published once the state is settled
	for ev := range h.Events {
		switch ev.Kind {
		case engine.KindInit:
			if ev.Init != nil && ev.Init.EngineSessionID != "" {
				s.setEngineSession(l, ev.Init.EngineSessionID)
			}
		case engine.KindAssistant:
			if ev.Assistant == nil {
				continue
			}
			for _, b := range ev.Assistant.Blocks {
				switch b.Kind {
				case engine.BlockText:
					if strings.TrimSpace(b.Text) != "" {
						l.log.add(Event{Time: s.stamp(), Type: EventText, Turn: n, Summary: firstRunes(b.Text, summaryRunes), Text: b.Text})
					}
				case engine.BlockToolUse:
					c := call{tool: b.Name, target: Target(b.Name, b.Input, l.tree)}
					l.mu.Lock()
					l.calls[b.ID] = c
					l.mu.Unlock()
					l.log.add(Event{Time: s.stamp(), Type: EventToolCall, Turn: n, ToolUseID: b.ID, Tool: c.tool, Target: c.target,
						Summary: TraceLine(c.tool, c.target)})
				}
			}
		case engine.KindToolResult:
			if ev.ToolResult != nil {
				s.toolResult(l, n, *ev.ToolResult)
			}
		case engine.KindResult:
			if ev.Result != nil && endEv == nil {
				l.mu.Lock()
				e := s.endTurnLocked(l, n, ev.Result, "")
				l.mu.Unlock()
				endEv = &e
			}
		case engine.KindExit:
			if endEv == nil {
				l.mu.Lock()
				e := s.endTurnLocked(l, n, nil, "")
				l.mu.Unlock()
				endEv = &e
			}
		case engine.KindUnknown:
		}
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if endEv == nil {
		e := s.endTurnLocked(l, n, nil, "")
		endEv = &e
	}
	l.running, l.turnDone = nil, nil
	// turn-ended goes out once the session's state is settled, so whoever acts on it (a
	// `send --wait` sending again) finds the session waiting, or the queued turn started.
	switch {
	case l.state != StateRunningTurn || s.closing.Load():
		// Interrupted by shutdown meanwhile, or about to be: Shutdown marks it.
		l.log.add(*endEv)
	case len(l.queue) > 0:
		msgs := l.queue
		l.queue = nil
		l.log.add(*endEv)
		_, _ = s.startTurnLocked(l, msgs) // a failure publishes its own turn-ended (turnNotStarted)
	default:
		l.state = StateWaitingForInput
		_ = s.setState(l.id, StateWaitingForInput)
		l.log.add(*endEv)
	}
}

func (s *Service) setEngineSession(l *live, id string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.engineSessionID == id {
		return
	}
	l.engineSessionID = id
	_ = s.cfg.Store.Write(func(tx *store.Tx) error {
		_, err := tx.Exec("UPDATE session SET engine_session_id = ? WHERE id = ?", store.Text(id), store.Text(l.id))
		return err
	})
}

// toolResult writes the call's permission decision row and audit events in one transaction,
// then publishes the decision and the result (03 Data flow, step 5). A tool's own error gets no
// row (03-hook-contract.md).
func (s *Service) toolResult(l *live, n int, tr engine.ToolResultEvent) {
	l.mu.Lock()
	c := l.calls[tr.ToolUseID]
	l.mu.Unlock()
	cl := Classify(tr, c.tool)
	now := s.stamp()
	line := ResultLine(c.tool, c.target, l.reach, tr.IsError, tr.Text, cl)
	if cl.Kind != ClassNoDecision {
		summary := TraceLine(c.tool, c.target)
		actor := ActorPolicy
		switch cl.DecidedBy {
		case DecidedByHook:
			actor = ActorHook
		case DecidedByConfig, DecidedByUnknown:
			actor = ActorInstance
		}
		err := s.cfg.Store.Write(func(tx *store.Tx) error {
			// A resumed call's final answer replaces its deferred row (P10 half b); the audit log keeps both.
			if _, err := tx.Exec(`INSERT INTO permission_decision (session_id, request_id, turn_seq, tool, summary, decided_by, user_id, decision, reason, decided_at)
VALUES (?, ?, ?, ?, ?, ?, NULL, ?, ?, ?)
ON CONFLICT (session_id, request_id) DO UPDATE SET turn_seq = excluded.turn_seq, decided_by = excluded.decided_by,
    decision = excluded.decision, reason = excluded.reason, decided_at = excluded.decided_at`,
				store.Text(l.id), store.Text(tr.ToolUseID), store.Int(int64(n)), store.Text(c.tool), store.Text(summary),
				store.Text(cl.DecidedBy), store.Text(cl.Decision), store.OptText(cl.Reason), store.Text(now)); err != nil {
				return err
			}
			if _, err := s.cfg.Audit.Append(tx, audit.Event{Kind: audit.KindPermissionDecision, Actor: actor, Session: l.id,
				Details: details("turn", strconv.Itoa(n), "toolUseId", tr.ToolUseID, "tool", c.tool, "target", c.target,
					"decision", cl.Decision, "decidedBy", cl.DecidedBy, "source", tr.DecidedBy, "reason", cl.Reason)}); err != nil {
				return err
			}
			switch cl.Kind {
			case ClassHookFailed:
				_, err := s.cfg.Audit.Append(tx, audit.Event{Kind: audit.KindActionRefused, Actor: ActorHook, Session: l.id,
					Details: details("action", "tool-call", "reason", "hook-failed", "toolUseId", tr.ToolUseID, "tool", c.tool,
						"message", cl.Reason)})
				return err
			case ClassBypassed:
				outcome, cause := "refused", "failed"
				if cl.Ran {
					outcome = "ran"
				}
				if cl.DecidedBy == DecidedByConfig {
					cause = "timed-out"
				}
				_, err := s.cfg.Audit.Append(tx, audit.Event{Kind: audit.KindHookBypassed, Actor: ActorInstance, Session: l.id,
					Details: details("toolUseId", tr.ToolUseID, "tool", c.tool, "outcome", outcome, "cause", cause,
						"text", firstRunes(tr.Text, summaryRunes))})
				return err
			case ClassNoDecision, ClassAllowed, ClassPolicyDenied:
			}
			return nil
		})
		if err != nil {
			l.log.add(Event{Time: now, Type: EventPermissionDecision, Turn: n, ToolUseID: tr.ToolUseID, Tool: c.tool,
				Summary: "store: " + err.Error()})
		} else {
			l.log.add(Event{Time: now, Type: EventPermissionDecision, Turn: n, ToolUseID: tr.ToolUseID, Tool: c.tool,
				Target: c.target, Decision: cl.Decision, DecidedBy: cl.DecidedBy, Reason: cl.Reason,
				Summary: cl.Decision + " " + summary + " (" + cl.DecidedBy + ")"})
		}
	}
	l.log.add(Event{Time: now, Type: EventToolResult, Turn: n, ToolUseID: tr.ToolUseID, Tool: c.tool, Target: c.target,
		IsError: tr.IsError, Summary: line})
}

// endTurnLocked records the turn's end (03 Data flow, step 7) and returns its turn-ended
// event, which the caller publishes once the session's state is settled; l.mu is held. A nil
// result is an exit without one: the turn was interrupted.
//
// TODO(P10 half b): the usage rows (the modelUsage delta against the last result row of the
// engine session, or the deduplicated assistant counts of an interrupted turn) go into this
// transaction, before turn-ended is published (N-21); a deferred result moves the session to
// waiting-for-answer with the deferred and dropped rows instead of back to waiting-for-input.
func (s *Service) endTurnLocked(l *live, n int, r *engine.ResultEvent, startErr string) Event {
	res, reason := ResultInterrupted, ResultInterrupted
	switch {
	case startErr != "":
		res, reason = ResultError, "engine: "+startErr
	case r == nil:
	case r.Deferred != nil:
		res, reason = ResultDeferred, r.StopReason
	case r.IsError:
		res, reason = ResultError, r.StopReason
	default:
		res, reason = ResultCompleted, r.StopReason
	}
	now := s.stamp()
	err := s.cfg.Store.Write(func(tx *store.Tx) error {
		_, err := tx.Exec("UPDATE turn SET ended_at = ?, result = ?, harness_ref = ? WHERE session_id = ? AND seq = ?",
			store.Text(now), store.Text(res), store.OptText(l.engineSessionID), store.Text(l.id), store.Int(int64(n)))
		return err
	})
	if err != nil {
		reason = errors.Join(errors.New(reason), err).Error()
	}
	return Event{Time: now, Type: EventTurnEnded, Turn: n, Result: res, Summary: reason}
}

// Follow hands the session's events after seq to fn in order, waiting for new ones, until fn
// says stop, the session closes or ctx ends (`session tail --follow`, `send --wait|--tail`).
func (s *Service) Follow(ctx context.Context, id string, seq int64, fn func(Event) (bool, error)) error {
	l := s.lookup(id)
	if l == nil {
		return s.closedSession(id)
	}
	return l.log.follow(ctx, seq, fn)
}

// Events returns the session's events after seq. They live in the bot's memory: a session
// held by an earlier run of the bot has none, and is unknown-session or session-closed.
func (s *Service) Events(id string, seq int64) ([]Event, error) {
	l := s.lookup(id)
	if l == nil {
		return nil, s.closedSession(id)
	}
	evs, _, _ := l.log.since(seq)
	return evs, nil
}
