// SPDX-License-Identifier: GPL-3.0-or-later

package session

import (
	"strings"

	"github.com/jasondorjeshort/broodforge/internal/engine"
	"github.com/jasondorjeshort/broodforge/internal/policy"
)

// Who decided a call: the permission_decision row's decided_by (02-domain.md, Permission
// decision; 03-hook-contract.md, What the bot sees).
const (
	// DecidedByPolicy is the hook answering from the policy table, both modes.
	DecidedByPolicy = "policy"
	// DecidedByHook is the hook failing (exit 2, `hook:`): the engine blocked the call.
	DecidedByHook = "hook"
	// DecidedByConfig is the engine's own default, the hook bypassed (C39).
	DecidedByConfig = "config"
	// DecidedByUnknown is a result without tool_result_meta's source.
	DecidedByUnknown = "unknown"
)

// Decisions of a permission_decision row. Deferred and dropped are prompt mode's (P10 half b).
const (
	DecisionAllow    = "allow"
	DecisionDeny     = "deny"
	DecisionDeferred = "deferred"
	DecisionDropped  = "dropped"
)

// Stream values of tool_result_meta[].permission_decision.source (C39).
const (
	sourceHook   = "hook"
	sourceConfig = "config"
)

// toolUseErrorPrefix starts a tool's own error, which the engine raises before any hook
// (S2's standalone sleep): no decision row.
const toolUseErrorPrefix = "<tool_use_error>"

// hookFailurePrefix starts the hook's own failure text (03-hook-contract.md, Failure).
const hookFailurePrefix = "hook:"

// ClassKind is what a tool result says about its permission decision.
type ClassKind int

const (
	// ClassNoDecision is the tool's own error: no row.
	ClassNoDecision ClassKind = iota
	// ClassAllowed: the call ran (or the tool failed after an allow).
	ClassAllowed
	// ClassPolicyDenied: the hook denied it with ADR 0020's text.
	ClassPolicyDenied
	// ClassHookFailed: the hook exited 2 with `hook: <what>`; audited action refused.
	ClassHookFailed
	// ClassBypassed: the engine's default decided; audited hook bypassed.
	ClassBypassed
)

// Classified is one tool result's decision.
type Classified struct {
	Kind      ClassKind
	Decision  string // allow or deny; "" for ClassNoDecision
	DecidedBy string
	// Reason is the text after the engine's wrapper, for a deny or a hook failure.
	Reason string
	// Ran is whether the call ran, for a bypassed one (04: "ran without" or "refused by the engine").
	Ran bool
}

// StripWrapper removes what the engine wraps a hook's block or deny reason in before the model
// sees it (ADR 0020, C39 amendment): `PreToolUse:<tool> hook error: `, then an optional
// `[<hook command>]: `, then surrounding whitespace. Text without the wrapper is only trimmed.
// tool is the call's name when known; "" accepts any one-word name, because a resumed turn's
// result arrives without its tool_use, which was the turn before's.
func StripWrapper(text, tool string) string {
	s := text
	if rest, ok := strings.CutPrefix(s, "PreToolUse:"); ok {
		name, after, found := strings.Cut(rest, " hook error: ")
		if found && name != "" && !strings.ContainsAny(name, " \t\r\n") && (tool == "" || name == tool) {
			s = after
			if strings.HasPrefix(s, "[") {
				if i := strings.Index(s, "]: "); i >= 0 {
					s = s[i+len("]: "):]
				}
			}
		}
	}
	return strings.TrimSpace(s)
}

// Classify reads one tool result (03-hook-contract.md, What the bot sees). The prefixes are
// tested on the stripped text, never by a contains-match: the rest of the text is the model's.
func Classify(tr engine.ToolResultEvent, tool string) Classified {
	if tr.IsError && strings.HasPrefix(strings.TrimSpace(tr.Text), toolUseErrorPrefix) {
		return Classified{Kind: ClassNoDecision}
	}
	stripped := StripWrapper(tr.Text, tool)
	if tr.IsError {
		switch {
		case strings.HasPrefix(stripped, policy.DenyPrefix):
			return Classified{Kind: ClassPolicyDenied, Decision: DecisionDeny, DecidedBy: DecidedByPolicy, Reason: stripped}
		case strings.HasPrefix(stripped, hookFailurePrefix):
			return Classified{Kind: ClassHookFailed, Decision: DecisionDeny, DecidedBy: DecidedByHook, Reason: stripped}
		}
	}
	switch tr.DecidedBy {
	case sourceHook:
		// The hook allowed it; an error here is the tool's own.
		return Classified{Kind: ClassAllowed, Decision: DecisionAllow, DecidedBy: DecidedByPolicy}
	case sourceConfig:
		c := Classified{Kind: ClassBypassed, Decision: DecisionAllow, DecidedBy: DecidedByConfig, Ran: !tr.IsError}
		if tr.IsError {
			c.Decision, c.Reason = DecisionDeny, stripped
		}
		return c
	}
	if tr.IsError {
		return Classified{Kind: ClassBypassed, Decision: DecisionDeny, DecidedBy: DecidedByUnknown, Reason: stripped}
	}
	return Classified{Kind: ClassAllowed, Decision: DecisionAllow, DecidedBy: DecidedByUnknown}
}
