// SPDX-License-Identifier: GPL-3.0-or-later

package policy

import (
	"slices"
	"strings"
)

// Mode is the session's permission mode (ADR 0020): policy or prompt.
type Mode int

const (
	ModePolicy Mode = iota
	ModePrompt
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
	ClassAgent              // TodoWrite, Task
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
	case "TodoWrite", "Task":
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
	CellAsk               // policy: allow; prompt: allow if session-allowed, else defer
	CellWeb               // policy: deny; prompt: defer
	CellDeny              // both modes, and no answer lifts it
)

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
	}
	return "unknown"
}

// Decision is what the hook returns to the engine.
type Decision int

const (
	Allow Decision = iota
	Deny
	Defer
)

func (d Decision) String() string {
	switch d {
	case Allow:
		return "allow"
	case Deny:
		return "deny"
	case Defer:
		return "defer"
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

// Allow and defer reasons; the engine shows them to nobody but the model.
const (
	reasonAllow        = "Broodforge: allowed by this channel's reach"
	reasonSessionAllow = "Broodforge: allowed for this session by a person in it"
	reasonAnswerAllow  = "Broodforge: allowed by a person in this session"
	reasonDefer        = "Broodforge: waiting for a person in this session to answer"
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
func Decide(req Request) Result {
	cell := CellFor(ClassOf(req.Tool), req.Reach, req.Paths)
	if cell == CellDeny {
		for _, p := range req.Paths {
			if p.Credential || p.HoldsCredential {
				return Result{Deny, CredentialDenyText(p.Target, !p.Credential), cell}
			}
		}
		return Result{Deny, DenyText(req.Tool, firstDenied(req), req.Reach), cell}
	}
	if a := req.Answer; a != nil && a.ToolUseID == req.ToolUseID {
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
		return Result{Defer, reasonDefer, cell}
	case CellWeb:
		if req.Mode == ModePolicy {
			return Result{Deny, DenyText(req.Tool, "", req.Reach), cell}
		}
		return Result{Defer, reasonDefer, cell}
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
		{ClassAgent, []string{"TodoWrite", "Task"}},
		{ClassWeb, []string{"WebFetch", "WebSearch"}},
		{ClassEdit, []string{"Edit", "MultiEdit", "Write", "NotebookEdit"}},
		{ClassBash, []string{"Bash"}},
		{ClassOther, nil},
	}
}
