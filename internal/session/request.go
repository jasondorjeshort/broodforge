// SPDX-License-Identifier: GPL-3.0-or-later

package session

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jasondorjeshort/broodforge/internal/audit"
	"github.com/jasondorjeshort/broodforge/internal/engine"
	"github.com/jasondorjeshort/broodforge/internal/policy"
	"github.com/jasondorjeshort/broodforge/internal/store"
)

// A subagent's live request (ADR 0029 rules 4 to 8 as amended 2026-11-07;
// 03-hook-contract-prompt-tool.md): on a prompt-mode process the hook answers a subagent's
// ask-cell call `ask`, the engine raises a control_request and waits on stdin, and the bot
// answers it in place through TurnHandle.Answer, the process running on. Each request has its
// prompt, its timer (subagentPromptTimeoutMinutes) and its answer; the turn's first result
// drops whatever is still open, and Stop, end, the idle end and shutdown deny each with
// interrupt true before the backend's interrupt.

// request is one live request, keyed by its request_id in live.requests.
type request struct {
	id                             string // request_id
	toolUseID, tool, target, agent string
	since, until                   string
	timeout                        time.Duration
	timer                          Timer
}

func (r *request) pending() Pending {
	return Pending{ToolUseID: r.toolUseID, Tool: r.tool, Target: r.target, Since: r.since, Until: r.until, Agent: r.agent}
}

// The texts of a live request. Placeholders where marked: not owner-worded.
const (
	// stopDenyMessage is the deny Stop, end, the idle end and shutdown write for each open
	// request, interrupt true (the companion, Stop, end, shutdown). Placeholder.
	stopDenyMessage = "Denied by Broodforge: the session was interrupted"
	// storeFailedDenyMessage is the deny of a request the store could not record (interrupt
	// false). Placeholder.
	storeFailedDenyMessage = "Denied by Broodforge: the request could not be recorded"
	// traceStopDenied is Stop's denied result line (the companion, Stop). Placeholder: the
	// coordinator's, not owner-worded.
	traceStopDenied = "denied: the turn was stopped"
	// causeTurnEnded is a request dropped at the turn's first result (decider instance).
	causeTurnEnded = "the turn ended"
	// traceDroppedTurnEnded is that request's trace line after the call's (the companion, The
	// turn's end with a request open). Placeholder.
	traceDroppedTurnEnded = "dropped: the turn ended before anyone answered"
	// reasonSessionAllow is the at-once allow of a session-allowed tool (ADR 0029 rule 5).
	reasonSessionAllow = "session-allow"
	// promptToolMissingSummary is rule 8's internal-error. Placeholder: the owner's wording
	// (ADR 0029 rule 8, critic's objection 15).
	promptToolMissingSummary = "engine prompt tool: wanted stdio, got none"
	// tracePromptToolMissing is rule 8's trace line naming the lift. Placeholder: the owner's
	// wording (ADR 0029 rule 8).
	tracePromptToolMissing = "no prompt reached anyone: switch the session to policy mode; the owner checks the spawn"
)

// The bypass causes of a result for a call the bot holds no request record for (ADR 0029
// Amendments 2026-11-07 (3)): the cause rides in the decision row's reason, no schema change.
const (
	CauseEngineAnswered = "engine answered"
	CauseEngineAborted  = "engine aborted"
)

// defaultSubagentPromptTimeout is subagentPromptTimeoutMinutes' default, for a channel that
// carries none (config always fills it; a test's channel may not).
const defaultSubagentPromptTimeout = 60 * time.Minute

// subagentPromptTimeout is the channel's bound on a live request.
func (l *live) subagentPromptTimeout() time.Duration {
	if l.ch.SubagentPromptTimeout <= 0 {
		return defaultSubagentPromptTimeout
	}
	return l.ch.SubagentPromptTimeout
}

// expiryReason is rule 6's deny reason on expiry (owner-approved 2026-11-07), the count in
// humanDuration's words so a one-minute bound reads `1 minute`.
func expiryReason(d time.Duration) string {
	return "Broodforge: the Discord prompt went unanswered for " + humanDuration(d) + "; stop here and report what you have."
}

// turnIsPrompt is whether the running turn started in prompt mode.
func (l *live) turnIsPrompt() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.turnMode == policy.ModePrompt.String()
}

// requestByToolUseLocked is the open request for a call, nil when none. l.mu is held.
func (l *live) requestByToolUseLocked(toolUseID string) *request {
	for _, rid := range l.requestOrder {
		if r := l.requests[rid]; r != nil && r.toolUseID == toolUseID {
			return r
		}
	}
	return nil
}

// agentOfLocked is a subagent call's agent type: the parent Agent call's subagent_type through
// the call's parent_tool_use_id, else the request's agent_id (ADR 0029 rule 4). l.mu is held.
func (l *live) agentOfLocked(c call, agentID string) string {
	if c.parent != "" {
		if a := l.calls[c.parent].agent; a != "" {
			return a
		}
	}
	return agentID
}

// requestArrived is a prompt event in a prompt turn (ADR 0029 rule 4): a tool the session
// allowed (never a web tool: a session allow does not cover web) is answered allow at once,
// decider participant, reason session-allow; any other call is recorded deferred with decider
// policy, gets its prompt (waiting-for-answer, indented under the Agent line) and its timer,
// and the session waits for an answer while the process runs on. A request after the process
// was stopped, or for a session that is ending, is logged, not acted on.
func (s *Service) requestArrived(l *live, n int, h engine.TurnHandle, p *engine.PromptEvent) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.stopDenied || s.closing.Load() || (l.state != StateRunningTurn && l.state != StateWaitingForAnswer) {
		return
	}
	if _, dup := l.requests[p.RequestID]; dup {
		return
	}
	c, ok := l.calls[p.ToolUseID]
	if !ok {
		c = call{tool: p.Tool, target: Target(p.Tool, p.Input, l.tree)}
		l.calls[p.ToolUseID] = c
	}
	agent := l.agentOfLocked(c, p.AgentID)
	now := s.now()
	stamp := store.FormatTime(now)
	if slices.Contains(l.sessionAllow, c.tool) && policy.ClassOf(c.tool) != policy.ClassWeb {
		if err := h.Answer(p.RequestID, true, "", false); err != nil {
			l.log.add(storeFailed(stamp, n, p.ToolUseID, c, err))
			return
		}
		if err := s.cfg.Store.Write(func(tx *store.Tx) error {
			return s.decisionTx(tx, l, n, p.ToolUseID, c, DecisionAllow, DecidedByParticipant, ActorInstance, "", reasonSessionAllow, "")
		}); err != nil {
			l.log.add(storeFailed(stamp, n, p.ToolUseID, c, err))
		}
		l.answers[p.ToolUseID] = answerRec{decision: DecisionAllow}
		l.log.add(Event{Time: stamp, Type: EventPermissionDecision, Turn: n, ToolUseID: p.ToolUseID, Tool: c.tool, Target: c.target,
			Decision: DecisionAllow, DecidedBy: DecidedByParticipant, Reason: reasonSessionAllow,
			Summary: c.indent(DecisionAllow + " " + TraceLine(c.tool, c.target) + " (" + DecidedByParticipant + ")"), Parent: c.parent, Depth: c.depth})
		return
	}
	if err := s.cfg.Store.Write(func(tx *store.Tx) error {
		return s.decisionTx(tx, l, n, p.ToolUseID, c, DecisionDeferred, DecidedByPolicy, ActorPolicy, "", "", "")
	}); err != nil {
		l.log.add(storeFailed(stamp, n, p.ToolUseID, c, err))
		// Unrecorded, the request would never be timed or answered and the turn would hang
		// with the process held: deny it at once.
		if aerr := h.Answer(p.RequestID, false, storeFailedDenyMessage, false); aerr != nil {
			l.log.add(storeFailed(stamp, n, p.ToolUseID, c, aerr))
		}
		return
	}
	d := l.subagentPromptTimeout()
	r := &request{id: p.RequestID, toolUseID: p.ToolUseID, tool: c.tool, target: c.target, agent: agent, since: stamp,
		until: store.FormatTime(now.Add(d)), timeout: d}
	if l.requests == nil {
		l.requests = map[string]*request{}
	}
	l.requests[r.id] = r
	l.requestOrder = append(l.requestOrder, r.id)
	rid := r.id
	r.timer = s.cfg.Clock.AfterFunc(d, func() { s.requestExpired(l, rid) })
	if l.state == StateRunningTurn {
		l.state = StateWaitingForAnswer
		_ = s.setState(l.id, StateWaitingForAnswer)
		s.armIdleLocked(l)
	}
	l.log.add(Event{Time: stamp, Type: EventPermissionDecision, Turn: n, ToolUseID: p.ToolUseID, Tool: c.tool, Target: c.target,
		Decision: DecisionDeferred, DecidedBy: DecidedByPolicy,
		Summary: c.indent(DecisionDeferred + " " + TraceLine(c.tool, c.target) + " (" + DecidedByPolicy + ")"), Parent: c.parent, Depth: c.depth})
	l.log.add(Event{Time: stamp, Type: EventWaitingForAnswer, Turn: n, ToolUseID: p.ToolUseID, Tool: c.tool, Target: c.target,
		Summary: c.indent(p.ToolUseID + " " + TraceLine(c.tool, c.target)), Text: FullCommand(c.tool, p.Input), Agent: agent,
		Parent: c.parent, Depth: c.depth})
}

// closeRequestLocked removes an answered or dropped request: its timer stopped, and with none
// left a waiting session runs again (the process never stopped), the idle timer off. l.mu is
// held.
func (s *Service) closeRequestLocked(l *live, r *request) {
	if r.timer != nil {
		r.timer.Stop()
	}
	delete(l.requests, r.id)
	l.requestOrder = slices.DeleteFunc(l.requestOrder, func(id string) bool { return id == r.id })
	if len(l.requests) == 0 && l.state == StateWaitingForAnswer && l.pending == nil {
		l.state = StateRunningTurn
		_ = s.setState(l.id, StateRunningTurn)
		s.disarmIdleLocked(l)
	}
}

// pickRequestLocked is the live request `session answer` names: by --tool-use-id, or the one
// open; with two or more open and none named, bad-args naming them. ok is false when the
// session holds no live request. l.mu is held.
func (l *live) pickRequestLocked(toolUseID string) (*request, bool, error) {
	if len(l.requests) == 0 {
		return nil, false, nil
	}
	if toolUseID != "" {
		if r := l.requestByToolUseLocked(toolUseID); r != nil {
			return r, true, nil
		}
		ids := l.openToolUseIDsLocked()
		return nil, true, &Error{CodeNotPending, toolUseID + " is not a call " + l.id + " waits on (" + strings.Join(ids, ", ") + ")"}
	}
	if len(l.requests) > 1 {
		return nil, true, badArgs("--tool-use-id is required: " + strconv.Itoa(len(l.requests)) + " calls wait: " +
			strings.Join(l.openToolUseIDsLocked(), ", "))
	}
	return l.requests[l.requestOrder[0]], true, nil
}

func (l *live) openToolUseIDsLocked() []string {
	ids := make([]string, 0, len(l.requestOrder))
	for _, rid := range l.requestOrder {
		ids = append(ids, l.requests[rid].toolUseID)
	}
	return ids
}

// answerRequestLocked answers a live request (ADR 0029 rules 5 and 6): allow, the call running
// with its input unchanged; deny with interrupt false and the reason (04's text when none, the
// expiry's on a timeout); allow-session also adds the tool to the session's one list. The row
// and its event carry the decider; the result, when it comes, is published with the answer's
// trace (the record-first rule). l.mu is held.
func (s *Service) answerRequestLocked(l *live, r *request, decision, decidedBy, actor, user, who, reason string) error {
	if l.running == nil {
		return &Error{CodeNotPending, r.toolUseID + " is no longer open"}
	}
	c := l.calls[r.toolUseID]
	allow := decision != AnswerDeny
	dec, msg, trace := DecisionAllow, "", ""
	if !allow {
		dec, msg = DecisionDeny, reason
		switch {
		case decidedBy == DecidedByPromptTimeout:
			// The venue's line is 04's: the timed-out prompt's trace, the bound in its words.
			trace = "denied: the prompt timed out after " + humanDuration(r.timeout) + "."
		case reason != "":
			trace = "denied by " + who + ": " + reason
		default:
			trace = "denied by " + who
			msg = policy.AnswerDenyText
		}
	}
	if err := l.running.Answer(r.id, allow, msg, false); err != nil {
		return &Error{CodeNotPending, r.toolUseID + " is no longer open: " + err.Error()}
	}
	err := s.cfg.Store.Write(func(tx *store.Tx) error {
		return s.decisionTx(tx, l, l.turn, r.toolUseID, c, dec, decidedBy, actor, user, reason, "")
	})
	if err != nil {
		l.log.add(storeFailed(s.stamp(), l.turn, r.toolUseID, c, err))
	}
	if decision == AnswerAllowSession && !slices.Contains(l.sessionAllow, c.tool) {
		l.sessionAllow = append(l.sessionAllow, c.tool)
	}
	l.answers[r.toolUseID] = answerRec{decision: dec, trace: trace}
	s.closeRequestLocked(l, r)
	l.log.add(Event{Time: s.stamp(), Type: EventPermissionDecision, Turn: l.turn, User: user, ToolUseID: r.toolUseID,
		Tool: c.tool, Target: c.target, Decision: dec, DecidedBy: decidedBy, Reason: reason,
		Summary: c.indent(dec + " " + TraceLine(c.tool, c.target) + " (" + decidedBy + ")"), Parent: c.parent, Depth: c.depth})
	return nil
}

// requestExpired is a live request's bound (ADR 0029 rule 6): deny, interrupt false, the
// expiry reason, decider prompt-timeout; the turn goes on.
func (s *Service) requestExpired(l *live, rid string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	r := l.requests[rid]
	if r == nil || s.closing.Load() || l.stopDenied {
		return
	}
	_ = s.answerRequestLocked(l, r, AnswerDeny, DecidedByPromptTimeout, ActorInstance, "", "", expiryReason(r.timeout))
}

// dropRequestsLocked is the turn's first result, or an exit without one, with requests open (ADR
// 0029 Amendments 2026-11-07 (1)): each is recorded dropped, decider instance, cause the turn
// ended, with its trace line; its prompt loses its buttons and its timer stops. The adapter has
// already denied it with interrupt true and killed the process. l.mu is held.
func (s *Service) dropRequestsLocked(l *live, n int) {
	for _, rid := range slices.Clone(l.requestOrder) {
		r := l.requests[rid]
		c := l.calls[r.toolUseID]
		stamp := s.stamp()
		if err := s.cfg.Store.Write(func(tx *store.Tx) error {
			return s.decisionTx(tx, l, n, r.toolUseID, c, DecisionDropped, DecidedByInstance, ActorInstance, "", causeTurnEnded, "")
		}); err != nil {
			l.log.add(storeFailed(stamp, n, r.toolUseID, c, err))
		}
		l.answers[r.toolUseID] = answerRec{decision: DecisionDropped, trace: traceDroppedTurnEnded}
		s.closeRequestLocked(l, r)
		l.log.add(Event{Time: stamp, Type: EventPermissionDecision, Turn: n, ToolUseID: r.toolUseID, Tool: c.tool, Target: c.target,
			Decision: DecisionDropped, DecidedBy: DecidedByInstance, Reason: causeTurnEnded,
			Summary: c.indent(DecisionDropped + " " + TraceLine(c.tool, c.target) + " (" + DecidedByInstance + ")"), Parent: c.parent, Depth: c.depth})
		l.log.add(Event{Time: stamp, Type: EventToolResult, Turn: n, ToolUseID: r.toolUseID, Tool: c.tool, Target: c.target, IsError: true,
			Summary: c.indent(TraceLine(c.tool, c.target) + " -> " + traceDroppedTurnEnded), Parent: c.parent, Depth: c.depth})
	}
}

// stopRequestsLocked is Stop, end, the idle end and shutdown with requests open (ADR 0029 rule
// 7 as amended): each answered deny with interrupt true, then recorded (Stop: deny with its
// actor, reason interrupted; the others: dropped, decider instance, with 05's cause), its
// prompt answered and its timer stopped. The process is then the caller's to interrupt at
// once; stopDenied makes its result, requests and defers logged, not acted on, and the turn end
// interrupted on the exit. l.mu is held.
func (s *Service) stopRequestsLocked(l *live, decision, decidedBy, actor, user, reason string) {
	if len(l.requests) == 0 || l.running == nil {
		return
	}
	n := l.turn
	l.stopDenied = true
	rowReason := reason
	if decision == DecisionDropped {
		rowReason = reason + " before an answer"
	}
	for _, rid := range slices.Clone(l.requestOrder) {
		r := l.requests[rid]
		c := l.calls[r.toolUseID]
		stamp := s.stamp()
		if err := l.running.Answer(r.id, false, stopDenyMessage, true); err != nil {
			l.log.add(storeFailed(stamp, n, r.toolUseID, c, err))
		}
		if err := s.cfg.Store.Write(func(tx *store.Tx) error {
			return s.decisionTx(tx, l, n, r.toolUseID, c, decision, decidedBy, actor, user, rowReason, "")
		}); err != nil {
			l.log.add(storeFailed(stamp, n, r.toolUseID, c, err))
		}
		rec := answerRec{decision: decision}
		if decision == DecisionDeny {
			rec.trace = traceStopDenied
		}
		l.answers[r.toolUseID] = rec
		s.closeRequestLocked(l, r)
		l.log.add(Event{Time: stamp, Type: EventPermissionDecision, Turn: n, User: user, ToolUseID: r.toolUseID, Tool: c.tool,
			Target: c.target, Decision: decision, DecidedBy: decidedBy, Reason: reason,
			Summary: c.indent(decision + " " + TraceLine(c.tool, c.target) + " (" + decidedBy + ")"), Parent: c.parent, Depth: c.depth})
	}
	if l.state == StateWaitingForAnswer && l.pending == nil {
		l.state = StateRunningTurn
		_ = s.setState(l.id, StateRunningTurn)
	}
}

// requestResultLocked is a result for a call whose request is still open: the engine answered
// it itself (a stdin close's abort, c43a), so the request closes on the bot's record, decider
// instance, the cause in the reason, and the result is published with that trace (the
// record-first rule, ADR 0029 Amendments 2026-11-07 (3)). l.mu is held.
func (s *Service) requestResultLocked(l *live, n int, r *request, tr engine.ToolResultEvent) answerRec {
	c := l.calls[r.toolUseID]
	cause := CauseEngineAnswered
	if tr.DecidedBy == sourceConfig && tr.ReasonType == reasonTypeOther {
		cause = CauseEngineAborted
	}
	dec := DecisionAllow
	if tr.PermissionDecision == engine.PermissionReject || tr.PermissionDecision == "" && tr.IsError {
		dec = DecisionDeny
	}
	stamp := s.stamp()
	if err := s.cfg.Store.Write(func(tx *store.Tx) error {
		return s.decisionTx(tx, l, n, r.toolUseID, c, dec, DecidedByInstance, ActorInstance, "", cause, tr.DecidedBy)
	}); err != nil {
		l.log.add(storeFailed(stamp, n, r.toolUseID, c, err))
	}
	s.closeRequestLocked(l, r)
	l.log.add(Event{Time: stamp, Type: EventPermissionDecision, Turn: n, ToolUseID: r.toolUseID, Tool: c.tool, Target: c.target,
		Decision: dec, DecidedBy: DecidedByInstance, Reason: cause,
		Summary: c.indent(dec + " " + TraceLine(c.tool, c.target) + " (" + DecidedByInstance + ")"), Parent: c.parent, Depth: c.depth})
	trace := ""
	if dec == DecisionDeny {
		trace = "denied: " + cause
	}
	return answerRec{decision: dec, trace: trace}
}

// permissionDeniedToolUse and permissionDeniedByHook read a system/permission_denied line's head (the parser keeps it as an
// unknown event, its first UnknownHeadBytes): the call and whether the hook decided it.
var (
	permissionDeniedToolUse = regexp.MustCompile(`"tool_use_id":"([^"]+)"`)
	permissionDeniedByHook  = regexp.MustCompile(`"decision_reason_type":"hook"`)
)

// hookAskedOf is the tool_use_id a system/permission_denied line names when the hook decided
// it (ADR 0029 rule 8: the engine without the prompt tool turns a hook `ask` into that line);
// "" for any other event.
func hookAskedOf(ev engine.EngineEvent) string {
	u := ev.Unknown
	if ev.Kind != engine.KindUnknown || u == nil || u.Type != "system" || u.Subtype != "permission_denied" ||
		!permissionDeniedByHook.MatchString(u.Head) {
		return ""
	}
	if m := permissionDeniedToolUse.FindStringSubmatch(u.Head); m != nil {
		return m[1]
	}
	return ""
}

// hookLogAsked is whether the session's hook.log holds an `ask` decision for the call (ADR
// 0029 rule 8); a missing or unreadable log is no.
func (l *live) hookLogAsked(toolUseID string) bool {
	f, err := os.Open(filepath.FromSlash(l.home + "/hook.log"))
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		var line struct {
			ToolUseID string `json:"toolUseId"`
			Decision  string `json:"decision"`
		}
		if json.Unmarshal(sc.Bytes(), &line) == nil && line.ToolUseID == toolUseID && line.Decision == policy.Ask.String() {
			return true
		}
	}
	return false
}

// promptToolMissing is rule 8's detector firing for a hook-answered ask that reached its result
// with no request seen: the engine-prompt-tool-missing audit row and the internal-error event;
// it returns the call's trace line naming the lift, which the result is published with.
func (s *Service) promptToolMissing(l *live, n int, toolUseID string) string {
	l.mu.Lock()
	c := l.calls[toolUseID]
	l.mu.Unlock()
	now := s.stamp()
	if err := s.cfg.Store.Write(func(tx *store.Tx) error {
		_, err := s.cfg.Audit.Append(tx, audit.Event{Kind: audit.KindEnginePromptToolMissing, Actor: ActorInstance, Session: l.id,
			Details: details("turn", strconv.Itoa(n), "toolUseId", toolUseID, "tool", c.tool, "agent", l.agentOf(c))})
		return err
	}); err != nil {
		l.log.add(storeFailed(now, n, toolUseID, c, err))
	}
	l.log.add(Event{Time: now, Type: EventInternalError, Turn: n, ToolUseID: toolUseID, Tool: c.tool, Target: c.target,
		Parent: c.parent, Depth: c.depth, Summary: promptToolMissingSummary})
	return TraceLine(c.tool, c.target) + " -> " + tracePromptToolMissing
}

// agentOf is agentOfLocked taking l.mu itself.
func (l *live) agentOf(c call) string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.agentOfLocked(c, "")
}

// reasonTypeOther is tool_result_meta's reason_type on a request the engine aborted at a stdin
// close (c43a).
const reasonTypeOther = "other"
