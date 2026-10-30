// SPDX-License-Identifier: GPL-3.0-or-later

package session

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strconv"
	"strings"

	"github.com/jasondorjeshort/broodforge/internal/audit"
	"github.com/jasondorjeshort/broodforge/internal/engine"
	"github.com/jasondorjeshort/broodforge/internal/git"
	"github.com/jasondorjeshort/broodforge/internal/policy"
	"github.com/jasondorjeshort/broodforge/internal/store"
)

// A turn is one participant input and everything the agent does until it next waits for input
// (02-domain.md, Turn). In prompt mode it may take several engine processes: one that ends
// deferred, then one resumed with the answer (ADR 0020, C27), and so on. Each process is
// numbered k within its turn; each of its results is one usage record per model.

// SendRequest is `session send` (05-debug-interface-session.md).
type SendRequest struct {
	Session, As, Roles, Text string
}

// Submitted is what submit (or answer) did: the turn it starts, joins or resumes, and how. Seq
// is the session's last event before it, where a follower of this turn starts.
type Submitted struct {
	Turn  int    `json:"turn"`
	State string `json:"state"` // started, queued or resumed
	Seq   int64  `json:"-"`
}

// Submitted states.
const (
	SubmitStarted = "started"
	SubmitQueued  = "queued"
	SubmitResumed = "resumed"
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
// idle session starts a turn; a busy one, or one waiting for an answer, queues the message for
// the next turn (ADR 0020).
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
	if req.Roles != "" && !isDriver && !policy.Permitted(splitRoles(req.Roles), l.ch.Roles.Contribute, s.groups(), false) {
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
		n, err := s.startTurnLocked(l, []queued{{user: req.As, block: block, text: req.Text}})
		if err != nil {
			return Submitted{}, err
		}
		return Submitted{Turn: n, State: SubmitStarted, Seq: seq}, nil
	}
	l.queue = append(l.queue, queued{user: req.As, block: block, text: req.Text})
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
// written (started) before the engine starts (03 Data flow, step 3). The mode is the one at
// the turn's start: a switch takes effect at the next turn (ADR 0020).
func (s *Service) startTurnLocked(l *live, msgs []queued) (int, error) {
	n := l.turn + 1
	blocks := make([]string, len(msgs))
	for i, m := range msgs {
		blocks[i] = m.block
	}
	input := JoinBlocks(blocks)
	now := s.stamp()
	// The requester is the first block's writer (R-44); every writer is in the input's From lines.
	if err := s.cfg.Store.Write(func(tx *store.Tx) error {
		if _, err := tx.Exec("INSERT INTO turn (session_id, seq, requester_id, input, started_at) VALUES (?, ?, ?, ?, ?)",
			store.Text(l.id), store.Int(int64(n)), store.Text(msgs[0].user), store.Text(input), store.Text(now)); err != nil {
			return err
		}
		if _, err := tx.Exec("UPDATE session SET state = ? WHERE id = ?", store.Text(StateRunningTurn), store.Text(l.id)); err != nil {
			return err
		}
		return s.turnModelLocked(tx, l)
	}); err != nil {
		return 0, s.turnNotStarted(l, n, msgs[0].user, err)
	}
	l.turn, l.state, l.turnMode, l.requester, l.proc = n, StateRunningTurn, l.mode, msgs[0].user, 0
	l.turnIn, l.turnOut, l.turnOutUnknown, l.turnModels = 0, 0, false, map[string]int64{}
	// Text is the whole input, From lines included: the transcript shows every message (04).
	l.log.add(Event{Time: now, Type: EventTurnStarted, Turn: n, User: msgs[0].user, Summary: firstRunes(input, summaryRunes), Text: input})
	_ = s.startProcessLocked(l, input, nil, "") // a failure ends the turn and publishes it
	return n, nil
}

// startProcessLocked starts the turn's next engine process: the first with the input, a
// resume with the answer in the hook environment and no stdin line (C27). l.mu is held. A
// process that cannot start ends the turn as an error.
func (s *Service) startProcessLocked(l *live, input string, answer *engine.TurnAnswer, reason string) error {
	n := l.turn
	l.proc++
	s.disarmIdleLocked(l)
	l.state = StateRunningTurn
	lines, err := s.trailers(l)
	if err == nil {
		err = git.WriteTrailers(l.git.Dir(), lines)
	}
	if err != nil {
		s.failTurnLocked(l, err)
		return err
	}
	env := []engine.HookVar{
		{Name: envSession, Value: l.id},
		{Name: envMode, Value: l.turnMode},
		{Name: envReach, Value: l.reach},
		{Name: envTree, Value: l.tree},
		{Name: envCanonical, Value: l.git.Canonical},
		{Name: envHookLog, Value: l.home + "/hook.log"},
	}
	if len(l.sessionAllow) > 0 {
		env = append(env, engine.HookVar{Name: envSessionAllow, Value: strings.Join(l.sessionAllow, ",")})
	}
	if answer != nil && reason != "" {
		env = append(env, engine.HookVar{Name: envAnswerReason, Value: reason})
	}
	// The copy, for the hook's chain: absent under inject none or when nothing was copied.
	if l.wf.copied {
		env = append(env, engine.HookVar{Name: envWorkflow, Value: l.wf.dir})
	}
	req := engine.TurnRequest{
		SessionID:       l.id,
		EngineSessionID: l.engineSessionID,
		Cwd:             l.tree,
		Delivery:        l.delivery,
		GitIdentity:     engine.GitIdentity{Name: l.driver.Name, Email: l.driver.Email},
		HookEnv:         env,
		Input:           input,
		Answer:          answer,
		Model:           l.turnModel,
		Effort:          l.turnEffort,
		// The header and the project's instructions, written once at start (ADR 0025): the
		// trailer lines reach commits through the trailers file and the commit-msg hook alone.
		SystemPromptFile: l.wf.instructions,
		AgentsFile:       l.wf.agents,
	}
	h, err := l.eng.StartTurn(req)
	if err != nil {
		s.failTurnLocked(l, err)
		return err
	}
	l.running = &h
	done := make(chan struct{})
	l.turnDone = done
	go s.runProcess(l, n, l.proc, h, done)
	return nil
}

// failTurnLocked ends the running turn as an error when its process could not start: the turn
// row, an audit event, the state back to waiting, then turn-ended. l.mu is held.
func (s *Service) failTurnLocked(l *live, err error) {
	ended := s.endTurnLocked(l, l.turn, nil, engine.TurnUsage{}, "engine: "+err.Error())
	s.auditTurnFailure(l, l.turn, l.requester, err)
	s.settleLocked(l, ended)
}

// turnNotStarted is a turn that failed before its row was written: it publishes turn-ended
// with the failure, so a `send --wait` on it returns, audits it, and returns err. l.mu is held.
func (s *Service) turnNotStarted(l *live, n int, user string, err error) error {
	s.auditTurnFailure(l, n, user, err)
	if l.state == StateRunningTurn { // the turn before it, when it was queued
		l.state = StateWaitingForInput
		_ = s.setState(l.id, StateWaitingForInput)
		s.armIdleLocked(l)
	}
	l.log.add(Event{Time: s.stamp(), Type: EventTurnEnded, Turn: n, Result: ResultError,
		Summary: "the turn could not start: " + firstRunes(err.Error(), summaryRunes)})
	return err
}

// auditTurnFailure writes action-refused for a turn that did not start, actor its requester.
func (s *Service) auditTurnFailure(l *live, n int, user string, err error) {
	_ = s.cfg.Store.Write(func(tx *store.Tx) error {
		_, aerr := s.cfg.Audit.Append(tx, audit.Event{Kind: audit.KindActionRefused, Actor: actorOf(user), Session: l.id,
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

// call is a tool call the session has seen, for its result's trace line, its decision row and
// the dropped rule (the calls of one message, C28).
type call struct {
	tool, target, message string
	// agent is a subagent call's subagent_type (the review fact, ADR 0025 rule 7); parent and
	// depth place a subagent's own calls under it (parent_tool_use_id).
	agent  string
	parent string
	depth  int
}

// indent is s at the call's depth (session tail, 04's trace lines).
func (c call) indent(s string) string { return Indent(c.depth, s) }

// subagentType is a subagent call's input subagent_type, "" for any other tool.
func subagentType(tool string, input json.RawMessage) string {
	if !subagentTools[tool] {
		return ""
	}
	var in struct {
		SubagentType string `json:"subagent_type"`
	}
	_ = json.Unmarshal(input, &in)
	return in.SubagentType
}

// depthUnder is the depth of a text or a call whose parent_tool_use_id is parent: one below
// the parent call, 0 at the top; a parent the session has not seen counts as a top-level call.
func (l *live) depthUnder(parent string) int {
	if parent == "" {
		return 0
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.calls[parent].depth + 1
}

// runProcess reads one engine process's events to the end (03 Data flow, steps 4 to 7) and
// hands them to processEnded. It is the only goroutine reading this handle.
func (s *Service) runProcess(l *live, n, k int, h engine.TurnHandle, done chan struct{}) {
	defer close(done)
	var evs []engine.EngineEvent
	results := map[string]bool{}
	esid := ""
	for ev := range h.Events {
		evs = append(evs, ev)
		switch ev.Kind {
		case engine.KindInit:
			if ev.Init != nil && ev.Init.EngineSessionID != "" {
				esid = ev.Init.EngineSessionID
				s.setEngineSession(l, esid)
			}
		case engine.KindAssistant:
			if ev.Assistant == nil {
				continue
			}
			parent := ev.Assistant.ParentToolUseID
			depth := l.depthUnder(parent)
			for _, b := range ev.Assistant.Blocks {
				switch b.Kind {
				case engine.BlockText:
					if strings.TrimSpace(b.Text) != "" {
						l.log.add(Event{Time: s.stamp(), Type: EventText, Turn: n, Summary: Indent(depth, firstRunes(b.Text, summaryRunes)),
							Text: b.Text, Parent: parent, Depth: depth})
					}
				case engine.BlockToolUse:
					c := call{tool: b.Name, target: Target(b.Name, b.Input, l.tree), message: ev.Assistant.MessageID,
						agent: subagentType(b.Name, b.Input), parent: parent, depth: depth}
					l.mu.Lock()
					l.calls[b.ID] = c
					l.mu.Unlock()
					l.log.add(Event{Time: s.stamp(), Type: EventToolCall, Turn: n, ToolUseID: b.ID, Tool: c.tool, Target: c.target,
						Summary: c.indent(TraceLine(c.tool, c.target)), Parent: parent, Depth: depth})
				}
			}
		case engine.KindToolResult:
			if ev.ToolResult != nil {
				results[ev.ToolResult.ToolUseID] = true
				s.toolResult(l, n, *ev.ToolResult)
			}
		case engine.KindRateLimit:
			if ev.RateLimit != nil {
				s.saveMeter(l, n, *ev.RateLimit)
			}
		case engine.KindResult, engine.KindExit, engine.KindUnknown:
		}
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.running, l.turnDone = nil, nil
	if esid == "" {
		esid = l.engineSessionID
	}
	s.processEndedLocked(l, n, k, esid, evs, results)
}

// lastResult is the process's result line, nil when it ended without one.
func lastResult(evs []engine.EngineEvent) *engine.ResultEvent {
	var r *engine.ResultEvent
	for _, ev := range evs {
		if ev.Kind == engine.KindResult && ev.Result != nil {
			r = ev.Result
		}
	}
	return r
}

// processEndedLocked records a process's end. A result that defers a call keeps the turn
// open, waiting for an answer; anything else ends the turn. Its usage is measured against the
// engine session's last result, never an interrupted row (S2). l.mu is held.
func (s *Service) processEndedLocked(l *live, n, k int, esid string, evs []engine.EngineEvent, results map[string]bool) {
	u := engine.UsageOf(evs, l.baselines[esid])
	r := lastResult(evs)
	if u.Cumulative != nil {
		l.baselines[esid] = u.Cumulative
	}
	if r != nil && r.Deferred != nil && l.state == StateRunningTurn && l.interruptedBy == "" && !s.closing.Load() {
		s.deferLocked(l, n, k, esid, u, r, results)
		return
	}
	ended := s.endTurnLocked(l, n, r, u, "", usageRef{esid: esid, k: k})
	if ended.Usage != nil {
		ended.Usage.Exit = exitOf(evs)
	}
	s.settleLocked(l, ended)
}

// settleLocked moves the session on after its turn ended, then publishes turn-ended, so
// whoever acts on it (a `send --wait` sending again) finds the session waiting for input or
// the queued turn started. l.mu is held.
func (s *Service) settleLocked(l *live, ended Event) {
	if l.interruptedBy != "" {
		ended.User, l.interruptedBy = l.interruptedBy, ""
	}
	switch {
	case l.state != StateRunningTurn || s.closing.Load():
		// Ending, or interrupted by shutdown meanwhile, or about to be: end and Shutdown mark it.
		l.log.add(ended)
	case len(l.queue) > 0:
		msgs := l.queue
		l.queue = nil
		l.log.add(ended)
		_, _ = s.startTurnLocked(l, msgs) // a failure publishes its own turn-ended (turnNotStarted)
	default:
		l.state = StateWaitingForInput
		_ = s.setState(l.id, StateWaitingForInput)
		s.armIdleLocked(l)
		l.log.add(ended)
	}
}

// deferLocked is a process that ended deferring a call (prompt mode): its usage, the
// `deferred` row for the call the result names and a `dropped` row for every other call of
// that message without a result (C28), then waiting-for-answer with the prompt and idle
// timers running. l.mu is held.
func (s *Service) deferLocked(l *live, n, k int, esid string, u engine.TurnUsage, r *engine.ResultEvent, results map[string]bool) {
	d := r.Deferred
	c, ok := l.calls[d.ToolUseID]
	if !ok {
		c = call{tool: d.Name, target: Target(d.Name, d.Input, l.tree)}
		l.calls[d.ToolUseID] = c
	}
	var dropped []string
	for id, other := range l.calls {
		if id != d.ToolUseID && other.message != "" && other.message == c.message && !results[id] {
			dropped = append(dropped, id)
		}
	}
	sortStrings(dropped)
	now := s.now()
	stamp := store.FormatTime(now)
	err := s.cfg.Store.Write(func(tx *store.Tx) error {
		if err := s.writeUsage(tx, l, n, u, r, usageRef{esid: esid, k: k}); err != nil {
			return err
		}
		if err := s.decisionTx(tx, l, n, d.ToolUseID, c, DecisionDeferred, DecidedByPolicy, ActorPolicy, "", "", ""); err != nil {
			return err
		}
		for _, id := range dropped {
			reason := "the engine kept only the last call of its message"
			if err := s.decisionTx(tx, l, n, id, l.calls[id], DecisionDropped, DecidedByPolicy, ActorPolicy, "", reason, ""); err != nil {
				return err
			}
		}
		_, err := tx.Exec("UPDATE session SET state = ? WHERE id = ?", store.Text(StateWaitingForAnswer), store.Text(l.id))
		return err
	})
	if err != nil {
		s.failTurnLocked(l, err)
		return
	}
	l.countUsageLocked(u)
	for _, id := range append([]string{d.ToolUseID}, dropped...) {
		dc := l.calls[id]
		dec := DecisionDropped
		if id == d.ToolUseID {
			dec = DecisionDeferred
		}
		l.log.add(Event{Time: stamp, Type: EventPermissionDecision, Turn: n, ToolUseID: id, Tool: dc.tool, Target: dc.target,
			Decision: dec, DecidedBy: DecidedByPolicy, Summary: dc.indent(dec + " " + TraceLine(dc.tool, dc.target) + " (" + DecidedByPolicy + ")")})
	}
	// Never is a year (effectiveTimeout): the deadline is always real.
	until := store.FormatTime(now.Add(effectiveTimeout(l.promptTimeout)))
	l.pending = &Pending{ToolUseID: d.ToolUseID, Tool: c.tool, Target: c.target, Since: stamp, Until: until,
		Dropped: len(dropped)}
	l.state = StateWaitingForAnswer
	s.armPromptLocked(l)
	s.armIdleLocked(l)
	l.log.add(Event{Time: stamp, Type: EventWaitingForAnswer, Turn: n, ToolUseID: d.ToolUseID, Tool: c.tool, Target: c.target,
		Summary: d.ToolUseID + " " + TraceLine(c.tool, c.target), Text: FullCommand(c.tool, d.Input)})
}

func sortStrings(s []string) { sort.Strings(s) }

// decisionTx writes one permission_decision row and its audit event in tx. A call that comes
// back (a deferred call answered on the resume) replaces its row; the audit log keeps both.
func (s *Service) decisionTx(tx *store.Tx, l *live, n int, id string, c call, decision, decidedBy, actor, user, reason, source string) error {
	if _, err := tx.Exec(`INSERT INTO permission_decision (session_id, request_id, turn_seq, tool, summary, decided_by, user_id, decision, reason, decided_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (session_id, request_id) DO UPDATE SET turn_seq = excluded.turn_seq, decided_by = excluded.decided_by,
    user_id = excluded.user_id, decision = excluded.decision, reason = excluded.reason, decided_at = excluded.decided_at`,
		store.Text(l.id), store.Text(id), store.Int(int64(n)), store.Text(c.tool), store.Text(TraceLine(c.tool, c.target)),
		store.Text(decidedBy), store.OptText(user), store.Text(decision), store.OptText(reason), store.Text(s.stamp())); err != nil {
		return err
	}
	_, err := s.cfg.Audit.Append(tx, audit.Event{Kind: audit.KindPermissionDecision, Actor: actor, Session: l.id, Registration: user,
		Details: details("turn", strconv.Itoa(n), "toolUseId", id, "tool", c.tool, "target", c.target,
			"decision", decision, "decidedBy", decidedBy, "source", source, "reason", reason, "agent", c.agent)})
	return err
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
// row (03-hook-contract.md). A call a person or the prompt timeout answered has its row and
// event already (answerLocked): only its result is published, with the answer's outcome.
func (s *Service) toolResult(l *live, n int, tr engine.ToolResultEvent) {
	l.mu.Lock()
	c := l.calls[tr.ToolUseID]
	ans, answered := l.answers[tr.ToolUseID]
	delete(l.answers, tr.ToolUseID)
	l.mu.Unlock()
	now := s.stamp()
	s.subagentResult(l, n, tr, c)
	if answered {
		line := TraceLine(c.tool, c.target)
		if ans.trace != "" {
			line += " -> " + ans.trace
		}
		l.log.add(Event{Time: now, Type: EventToolResult, Turn: n, ToolUseID: tr.ToolUseID, Tool: c.tool, Target: c.target,
			IsError: tr.IsError, Summary: c.indent(line), Parent: c.parent, Depth: c.depth})
		return
	}
	cl := Classify(tr, c.tool)
	line := ResultLine(c.tool, c.target, l.reach, tr.IsError, tr.Text, cl)
	if cl.Kind != ClassNoDecision {
		actor := ActorPolicy
		switch cl.DecidedBy {
		case DecidedByHook:
			actor = ActorHook
		case DecidedByConfig, DecidedByUnknown:
			actor = ActorInstance
		}
		err := s.cfg.Store.Write(func(tx *store.Tx) error {
			if err := s.decisionTx(tx, l, n, tr.ToolUseID, c, cl.Decision, cl.DecidedBy, actor, "", cl.Reason, tr.DecidedBy); err != nil {
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
			l.log.add(storeFailed(now, n, tr.ToolUseID, c, err))
		} else {
			l.log.add(Event{Time: now, Type: EventPermissionDecision, Turn: n, ToolUseID: tr.ToolUseID, Tool: c.tool,
				Target: c.target, Decision: cl.Decision, DecidedBy: cl.DecidedBy, Reason: cl.Reason,
				Summary: c.indent(cl.Decision + " " + TraceLine(c.tool, c.target) + " (" + cl.DecidedBy + ")"), Parent: c.parent, Depth: c.depth})
		}
	}
	l.log.add(Event{Time: now, Type: EventToolResult, Turn: n, ToolUseID: tr.ToolUseID, Tool: c.tool, Target: c.target,
		IsError: tr.IsError, Summary: c.indent(line), Parent: c.parent, Depth: c.depth})
}

// usageRef names a process's usage records: harness_ref is `<session id>/<engine session
// id>/<turn>.<k>/<model>`, unique instance-wide (the column is) even when two sessions replay one
// fixture, whose engine session ids repeat.
type usageRef struct {
	esid string
	k    int
}

// usageRaw is a usage record's raw column: what the engine reported for the model, so
// `usage --by turn --json` can be laid beside the fixture (05, Usage and audit queries).
type usageRaw struct {
	ModelUsage  engine.ModelUsage `json:"modelUsage,omitempty"`
	Usage       json.RawMessage   `json:"usage,omitempty"`
	Delta       engine.ModelUsage `json:"delta"`
	Interrupted bool              `json:"interrupted,omitempty"`
}

// writeUsage writes one usage record per model with a non-zero delta (R-44, N-21), in tx.
// An interrupted process's record has no output count and is flagged (ADR 0020; S2).
func (s *Service) writeUsage(tx *store.Tx, l *live, n int, u engine.TurnUsage, r *engine.ResultEvent, ref usageRef) error {
	models := make([]string, 0, len(u.Delta))
	for m := range u.Delta {
		models = append(models, m)
	}
	sortStrings(models)
	esid := ref.esid
	if esid == "" {
		esid = "-" // no init seen: the process never started its engine session
	}
	for _, model := range models {
		d := u.Delta[model]
		in, out := int64(d["inputTokens"]), int64(d["outputTokens"])
		cr, cw := int64(d["cacheReadInputTokens"]), int64(d["cacheCreationInputTokens"])
		if in+out+cr+cw == 0 {
			continue
		}
		raw := usageRaw{Delta: d, Interrupted: u.Interrupted}
		if r != nil {
			raw.ModelUsage, raw.Usage = r.ModelUsage[model], r.Usage
		}
		b, err := json.Marshal(raw)
		if err != nil {
			return err
		}
		interrupted := int64(0)
		if u.Interrupted {
			interrupted = 1
		}
		if _, err := tx.Exec(`INSERT INTO usage_record (session_id, turn_seq, harness_ref, requester_id, credential_id, channel_id,
    project_id, harness, model, input_tokens, output_tokens, cache_read_tokens, cache_write_tokens, interrupted, raw, recorded_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			store.Text(l.id), store.Int(int64(n)), store.Text(l.id+"/"+esid+"/"+strconv.Itoa(n)+"."+strconv.Itoa(ref.k)+"/"+model),
			store.OptText(l.requester), store.Text(l.credential), store.Text(l.ch.ID), store.Text(l.ch.Project), store.Text(l.ch.Harness),
			store.Text(model), store.Int(in), store.Int(out), store.Int(cr), store.Int(cw), store.Int(interrupted),
			store.Text(string(b)), store.Text(s.stamp())); err != nil {
			return err
		}
	}
	return nil
}

// endTurnLocked records the turn's end (03 Data flow, step 7): the turn row and the last
// process's usage records in one transaction, committed before turn-ended is published (N-21).
// It returns the turn-ended event for the caller to publish once the session's state is
// settled; l.mu is held. A nil result is an exit without one: the turn was interrupted.
func (s *Service) endTurnLocked(l *live, n int, r *engine.ResultEvent, u engine.TurnUsage, startErr string, refs ...usageRef) Event {
	res, reason := ResultInterrupted, ResultInterrupted
	switch {
	case startErr != "":
		res, reason = ResultError, startErr
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
		for _, ref := range refs {
			if err := s.writeUsage(tx, l, n, u, r, ref); err != nil {
				return err
			}
		}
		_, err := tx.Exec("UPDATE turn SET ended_at = ?, result = ?, harness_ref = ? WHERE session_id = ? AND seq = ?",
			store.Text(now), store.Text(res), store.OptText(l.engineSessionID), store.Text(l.id), store.Int(int64(n)))
		return err
	})
	if err != nil {
		reason = errors.Join(errors.New(reason), err).Error()
	} else if len(refs) > 0 {
		l.countUsageLocked(u)
	}
	l.pending = nil
	s.disarmPromptLocked(l)
	usage := &TurnUsage{In: l.turnIn, Out: l.turnOut, OutUnknown: l.turnOutUnknown, SessionIn: l.sessIn, SessionOut: l.sessOut,
		Model: topModel(l.turnModels, l.turnModel), Mode: l.turnMode}
	return Event{Time: now, Type: EventTurnEnded, Turn: n, Result: res, Summary: reason, Usage: usage}
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

// countUsageLocked adds one engine process's usage to the running turn's and the session's
// counts, the footer's figures (04): in is input plus cache reads and writes. l.mu is held.
func (l *live) countUsageLocked(u engine.TurnUsage) {
	for model, d := range u.Delta {
		in := int64(d["inputTokens"] + d["cacheReadInputTokens"] + d["cacheCreationInputTokens"])
		out := int64(d["outputTokens"])
		l.turnIn += in
		l.turnOut += out
		l.sessIn += in
		l.sessOut += out
		l.turnModels[model] += in + out
		l.sessionModels[model] += in + out
	}
	if u.Interrupted {
		l.turnOutUnknown = true
	}
}

// topModel is the model with the most tokens, the one a footer names; fallback when none ran.
func topModel(models map[string]int64, fallback string) string {
	best, name := int64(-1), fallback
	keys := make([]string, 0, len(models))
	for m := range models {
		keys = append(keys, m)
	}
	sortStrings(keys)
	for _, m := range keys {
		if models[m] > best {
			best, name = models[m], m
		}
	}
	return name
}

// exitOf is the process's exit code as text, "" when the stream has no exit or no code.
func exitOf(evs []engine.EngineEvent) string {
	for i := len(evs) - 1; i >= 0; i-- {
		if evs[i].Kind == engine.KindExit && evs[i].Exit != nil && evs[i].Exit.Code != nil {
			return strconv.Itoa(*evs[i].Exit.Code)
		}
	}
	return ""
}

// subagentResult writes the subagent-result row of a subagent call's tool result (ADR 0025
// rule 7, the review fact): which subagent, and whether its result was an error (a denial
// included). It is written for every result of the subagent tool, a person's answer or not; a
// call that never returns a result has none. A store failure is shown as toolResult shows one.
func (s *Service) subagentResult(l *live, n int, tr engine.ToolResultEvent, c call) {
	if !subagentTools[c.tool] {
		return
	}
	result := subagentResultOK
	if tr.IsError {
		result = subagentResultError
	}
	err := s.cfg.Store.Write(func(tx *store.Tx) error {
		_, err := s.cfg.Audit.Append(tx, audit.Event{Kind: audit.KindSubagentResult, Actor: ActorInstance, Session: l.id,
			Details: details("turn", strconv.Itoa(n), "toolUseId", tr.ToolUseID, "tool", c.tool, "agent", c.agent, "result", result)})
		return err
	})
	if err != nil {
		l.log.add(storeFailed(s.stamp(), n, tr.ToolUseID, c, err))
	}
}

// saveMeter records a rate-limit event as the session's credential's meter, replacing the one
// it had, in its own short write transaction (ADR 0021 Amendments 2026-10-30). A reading, not
// an action: no audit event. A store failure is shown as subagentResult shows one, never the
// turn's end.
func (s *Service) saveMeter(l *live, n int, r engine.RateLimitEvent) {
	err := store.SaveMeter(s.cfg.Store, store.MeterRow{Credential: l.credential, SeenAt: s.stamp(), Session: l.id,
		Status: r.Status, RateLimitType: r.RateLimitType,
		FiveHourUtilization: r.FiveHour.Utilization, FiveHourResetsAt: r.FiveHour.ResetsAt,
		SevenDayUtilization: r.SevenDay.Utilization, SevenDayResetsAt: r.SevenDay.ResetsAt,
		OverageStatus: r.OverageStatus, IsUsingOverage: r.IsUsingOverage})
	if err != nil {
		l.log.add(storeFailed(s.stamp(), n, "", call{}, err))
	}
}

// storeFailed is the internal-error event of a store write failing mid-turn: the failure does not end the turn.
// c and toolUseID are the call it happened in, zero outside one.
func storeFailed(at string, n int, toolUseID string, c call, err error) Event {
	return Event{Time: at, Type: EventInternalError, Turn: n, ToolUseID: toolUseID, Tool: c.tool, Target: c.target,
		Parent: c.parent, Depth: c.depth, Summary: "store: " + err.Error()}
}
