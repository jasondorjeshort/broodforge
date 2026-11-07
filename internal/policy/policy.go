// SPDX-License-Identifier: GPL-3.0-or-later

package policy

import (
	"slices"
	"strings"
)

// Mode is the session's permission mode (ADR 0020 as amended 2026-11-01): policy, prompt or
// auto, policy mode with the engine's own classifier reviewing the ask cell.
type Mode int

const (
	ModePolicy Mode = iota
	ModePrompt
	ModeAuto
)

// Reach is the channel's reach (03-hook-contract.md, Policy table): read-only or code.
type Reach int

const (
	ReachReadOnly Reach = iota
	ReachCode
)

// ParseMode reads BROODFORGE_MODE's value.
func ParseMode(s string) (Mode, bool) {
	switch s {
	case "policy":
		return ModePolicy, true
	case "prompt":
		return ModePrompt, true
	case "auto":
		return ModeAuto, true
	}
	return ModePolicy, false
}

// ParseReach reads BROODFORGE_REACH's value.
func ParseReach(s string) (Reach, bool) {
	switch s {
	case "read-only":
		return ReachReadOnly, true
	case "code":
		return ReachCode, true
	}
	return ReachReadOnly, false
}

func (m Mode) String() string {
	switch m {
	case ModePolicy:
		return "policy"
	case ModePrompt:
		return "prompt"
	case ModeAuto:
		return "auto"
	}
	return "unknown"
}

func (r Reach) String() string {
	switch r {
	case ReachReadOnly:
		return "read-only"
	case ReachCode:
		return "code"
	}
	return "unknown"
}

// Class is a tool class, one row of the policy table.
type Class int

const (
	ClassRead  Class = iota // Read, Glob, Grep, LS, NotebookRead
	ClassAgent              // TodoWrite, Agent, Task
	ClassWeb                // WebFetch, WebSearch
	ClassEdit               // Edit, MultiEdit, Write, NotebookEdit
	ClassBash               // Bash
	ClassOther              // anything else, mcp__* included
)

// ClassOf is the row a tool name falls in. Names match exactly, as the engine sends them.
func ClassOf(tool string) Class {
	switch tool {
	case "Read", "Glob", "Grep", "LS", "NotebookRead":
		return ClassRead
	case "TodoWrite", "Agent", "Task":
		return ClassAgent
	case "WebFetch", "WebSearch":
		return ClassWeb
	case "Edit", "MultiEdit", "Write", "NotebookEdit":
		return ClassEdit
	case "Bash":
		return ClassBash
	}
	return ClassOther
}

// Cell is one cell of the table, after the path checks.
type Cell int

const (
	CellAllow Cell = iota // both modes
	CellAsk               // policy: allow; prompt: allow if session-allowed, else defer (a subagent's: ask); auto: allow if session-allowed, else pass
	CellWeb               // policy and auto: deny; prompt: defer (a subagent's: allow if session-allowed, else ask)
	CellDeny              // both modes, and no answer lifts it
	// CellReviewer is the `reviewer` cell (ADR 0028): an Agent or Task call whose subagent_type
	// is reviewer, defer in every mode and reach; the bot answers it itself after pinning the
	// tree. One object of the table, never configuration. A subagent's is deny in every mode
	// (ADR 0029 rule 1).
	CellReviewer
)

// ReviewerAgent is the subagent type the `reviewer` cell defers.
const ReviewerAgent = "reviewer"

// ReviewerCall is whether a call falls in the `reviewer` cell: the subagent tool (Agent, Task
// on an older pin) whose tool_input.subagent_type is reviewer. The hook and session both ask
// it, so the call the hook defers is the call the bot pins.
func ReviewerCall(tool, subagentType string) bool {
	return (tool == "Agent" || tool == "Task") && subagentType == ReviewerAgent
}

func (c Cell) String() string {
	switch c {
	case CellAllow:
		return "allow"
	case CellAsk:
		return "ask"
	case CellWeb:
		return "web"
	case CellDeny:
		return "deny"
	case CellReviewer:
		return "reviewer"
	}
	return "unknown"
}

// Decision is what the hook returns to the engine.
type Decision int

const (
	Allow Decision = iota
	Deny
	Defer
	// Pass is auto mode's answer for the ask cell (03-hook-contract.md, Auto mode): no
	// decision, exit 0 with nothing on stdout, so the engine's own classifier decides. C40a
	// (2026-11-07) holds that a silent exit 0 is no decision.
	Pass
	// Ask is a subagent's ask- or web-cell call in prompt mode (ADR 0029 rule 1): the engine
	// routes it to the prompt-mode process's stdio prompt tool, where the bot answers it.
	Ask
)

func (d Decision) String() string {
	switch d {
	case Allow:
		return "allow"
	case Deny:
		return "deny"
	case Defer:
		return "defer"
	case Pass:
		return "pass"
	case Ask:
		return "ask"
	}
	return "unknown"
}

// ParseDecision reads an answer's decision: only allow or deny can be an answer.
func ParseDecision(s string) (Decision, bool) {
	switch s {
	case "allow":
		return Allow, true
	case "deny":
		return Deny, true
	}
	return Allow, false
}

// Path is one path of a tool call as the hook resolved it: the facts the table reads. The
// hook canonicalises (it touches the filesystem); policy only reads the facts.
type Path struct {
	// Target is the path as shown in the deny text, slash-separated.
	Target         string
	UnderTree      bool
	UnderCanonical bool
	// Credential is a file named .credentials.json or anything under the config directory;
	// HoldsCredential a recursive search whose root holds the config directory.
	Credential      bool
	HoldsCredential bool
}

// Answer is a person's answer for one tool_use_id (BROODFORGE_ANSWER).
type Answer struct {
	ToolUseID string
	Decision  Decision
	// Reason is BROODFORGE_ANSWER_REASON, returned verbatim on a deny; "" for the default.
	Reason string
}

// Request is everything the decision reads.
type Request struct {
	Tool      string
	ToolUseID string
	Mode      Mode
	Reach     Reach
	Paths     []Path
	Answer    *Answer
	// SessionAllow lists the tools a person allowed for the session (BROODFORGE_SESSION_ALLOW).
	SessionAllow []string
	// SubagentType is a subagent call's tool_input.subagent_type, "" for any other call.
	SubagentType string
	// Subagent is whether a subagent made the call (the hook input's agent_id set, ADR 0029
	// rule 1); false is the driver process's own call, decided as before ADR 0029.
	Subagent bool
}

// Result is the decision with its reason and the cell it came from.
type Result struct {
	Decision Decision
	Reason   string
	Cell     Cell
}

// DenyPrefix starts every deny reason of the policy, the prefix session classifies by
// (03-hook-contract.md, What the bot sees).
const DenyPrefix = "Denied by Broodforge:"

// DenyText is ADR 0020's reach deny: `Denied by Broodforge: <tool> on <target> is outside this
// channel's reach (<reach>); ...`. A call with no path names the tool alone.
func DenyText(tool, target string, r Reach) string {
	what := tool
	if target != "" {
		what += " on " + target
	}
	return DenyPrefix + " " + what + " is outside this channel's reach (" + r.String() +
		"); nobody in this session can allow it; the channel's configuration sets reach"
}

// CredentialDenyText is the deny for a credential path, in every reach and mode; a search
// root above the config directory holds one.
func CredentialDenyText(target string, holds bool) string {
	if holds {
		return DenyPrefix + " " + target + " holds a credential file"
	}
	return DenyPrefix + " " + target + " is a credential file"
}

// AnswerDenyText is the deny reason of a person's deny when session gave none.
const AnswerDenyText = DenyPrefix + " a person in this session denied this call"

// reviewerSubagentDenyText is the deny of a subagent's `reviewer` launch, in every mode (ADR
// 0029 rule 1; the owner's wording, 2026-11-07).
const reviewerSubagentDenyText = DenyPrefix + " in this session the reviewer is launched by the driver process; " +
	"report and let it launch one."

// Allow and defer reasons; the engine shows them to nobody but the model.
const (
	reasonAllow         = "Broodforge: allowed by this channel's reach"
	reasonSessionAllow  = "Broodforge: allowed for this session by a person in it"
	reasonAnswerAllow   = "Broodforge: allowed by a person in this session"
	reasonDefer         = "Broodforge: waiting for a person in this session to answer"
	reasonReviewer      = "Broodforge: the bot records the tree before the reviewer starts"
	reasonReviewerAllow = "Broodforge: the bot recorded the tree; the reviewer starts"
	// reasonPass is a pass's reason: hook.log's and `hook try`'s only, since a pass prints
	// nothing to the engine.
	reasonPass = "Broodforge: inside this channel's reach; the engine's own check decides"
	// reasonAsk is an ask's reason (ADR 0029 rule 1; the owner's wording, 2026-11-07): the
	// model reads it only on a fall-through, when no prompt reached a person.
	reasonAsk = "Broodforge: asked a person in this session; if this reads as a refusal, no prompt reached them: say so and stop"
)

// CellFor is the table (03-hook-contract.md, Policy table) after the path checks: a credential
// path is deny in every class, reach and mode.
func CellFor(class Class, r Reach, paths []Path) Cell {
	for _, p := range paths {
		if p.Credential || p.HoldsCredential {
			return CellDeny
		}
	}
	switch class {
	case ClassRead:
		if r == ReachCode {
			return CellAllow
		}
		for _, p := range paths {
			if !p.UnderTree && !p.UnderCanonical {
				return CellDeny
			}
		}
		return CellAllow
	case ClassAgent:
		return CellAllow
	case ClassWeb:
		return CellWeb
	case ClassEdit:
		if r == ReachReadOnly {
			return CellDeny
		}
		for _, p := range paths {
			if !p.UnderTree {
				return CellDeny
			}
		}
		return CellAsk
	case ClassBash:
		if r == ReachReadOnly {
			return CellDeny
		}
		return CellAsk
	case ClassOther:
		return CellDeny
	}
	return CellDeny
}

// Decide is the decision order (03-hook-contract.md, Decision): a deny cell first, so an answer
// cannot lift a reach deny; then a pending answer for this tool_use_id; then the cell by mode.
// The `reviewer` cell replaces the row's cell after the path checks, before the answer, so a
// resume with the bot's allow returns allow and a first call defers in every mode.
//
// A subagent's call (ADR 0029 rule 1): the `reviewer` cell is deny in every mode, after the
// reach and credential denies and before any answer, since the bot never answers a subagent's
// launch; in prompt mode the ask and web cells are ask, a session-allowed tool allow. Every
// other cell, and every call in policy or auto mode, decides as the driver process's own.
func Decide(req Request) Result {
	cell := CellFor(ClassOf(req.Tool), req.Reach, req.Paths)
	if cell != CellDeny && ReviewerCall(req.Tool, req.SubagentType) {
		cell = CellReviewer
	}
	if cell == CellDeny {
		for _, p := range req.Paths {
			if p.Credential || p.HoldsCredential {
				return Result{Deny, CredentialDenyText(p.Target, !p.Credential), cell}
			}
		}
		return Result{Deny, DenyText(req.Tool, firstDenied(req), req.Reach), cell}
	}
	if cell == CellReviewer && req.Subagent {
		return Result{Deny, reviewerSubagentDenyText, cell}
	}
	if a := req.Answer; a != nil && a.ToolUseID == req.ToolUseID {
		if a.Decision == Allow && cell == CellReviewer {
			return Result{Allow, reasonReviewerAllow, cell}
		}
		if a.Decision == Allow {
			return Result{Allow, reasonAnswerAllow, cell}
		}
		reason := a.Reason
		if reason == "" {
			reason = AnswerDenyText
		}
		return Result{Deny, reason, cell}
	}
	switch cell {
	case CellAllow:
		return Result{Allow, reasonAllow, cell}
	case CellAsk:
		if req.Mode == ModePolicy {
			return Result{Allow, reasonAllow, cell}
		}
		if slices.Contains(req.SessionAllow, req.Tool) {
			return Result{Allow, reasonSessionAllow, cell}
		}
		if req.Mode == ModeAuto {
			return Result{Pass, reasonPass, cell}
		}
		if req.Subagent {
			return Result{Ask, reasonAsk, cell}
		}
		return Result{Defer, reasonDefer, cell}
	case CellWeb:
		if req.Mode == ModePolicy || req.Mode == ModeAuto {
			return Result{Deny, DenyText(req.Tool, "", req.Reach), cell}
		}
		if req.Subagent {
			// A session allow does not cover web (the policy table): always ask.
			return Result{Ask, reasonAsk, cell}
		}
		return Result{Defer, reasonDefer, cell}
	case CellReviewer:
		return Result{Defer, reasonReviewer, cell}
	case CellDeny:
	}
	return Result{Deny, DenyText(req.Tool, "", req.Reach), CellDeny}
}

// firstDenied is the target the reach deny names: the first path outside what the class may
// touch, else the first path.
func firstDenied(req Request) string {
	for _, p := range req.Paths {
		if !p.UnderTree && (ClassOf(req.Tool) == ClassEdit || !p.UnderCanonical) {
			return p.Target
		}
	}
	if len(req.Paths) > 0 {
		return req.Paths[0].Target
	}
	return ""
}

// SplitList reads a comma-separated list such as BROODFORGE_SESSION_ALLOW, dropping empties.
func SplitList(s string) []string {
	var out []string
	for _, f := range strings.Split(s, ",") {
		if f = strings.TrimSpace(f); f != "" {
			out = append(out, f)
		}
	}
	return out
}

// ClassRow is one row of the policy table as `session reach-report` prints it: the class and
// the tool names ClassOf puts in it (none for ClassOther, which is every other name).
type ClassRow struct {
	Class Class
	Tools []string
}

// ClassRows is the table's rows in its order (03-hook-contract.md, Policy table).
func ClassRows() []ClassRow {
	return []ClassRow{
		{ClassRead, []string{"Read", "Glob", "Grep", "LS", "NotebookRead"}},
		{ClassAgent, []string{"TodoWrite", "Agent", "Task"}},
		{ClassWeb, []string{"WebFetch", "WebSearch"}},
		{ClassEdit, []string{"Edit", "MultiEdit", "Write", "NotebookEdit"}},
		{ClassBash, []string{"Bash"}},
		{ClassOther, nil},
	}
}
