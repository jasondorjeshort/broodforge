// SPDX-License-Identifier: GPL-3.0-or-later

package session

import (
	"encoding/json"
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
	// DecidedByEngine is auto mode's engine classifier deciding an ask-cell call the hook
	// passed (03-hook-contract.md, What the bot sees; Auto mode), and the deferred row of a
	// call the engine's own check asked about (shape A).
	DecidedByEngine = "engine"
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

// onFailurePrefix starts the engine's text when it blocks a hook that timed out or could not
// start under "onFailure": "block" (03-hook-contract.md, What the bot sees): a hook failure.
const onFailurePrefix = "failed; blocking because onFailure"

// OnFailureCause is the `cause` of a hook failure's audit row: for the engine's onFailure block
// its own text, unclassified, since the text does not say whether the hook timed out or did not
// start; "" (no cause) for the hook's own `hook:` failure.
func OnFailureCause(reason string) string {
	if strings.HasPrefix(reason, onFailurePrefix) {
		return reason
	}
	return ""
}

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
	// ClassEngine: auto mode only, the engine's classifier decided an ask-cell call the hook
	// passed; decider engine, never hook bypassed.
	ClassEngine
)

// Classified is one tool result's decision.
type Classified struct {
	Kind      ClassKind
	Decision  string // allow or deny; "" for ClassNoDecision
	DecidedBy string
	// Reason is the text after the engine's wrapper, for a deny or a hook failure; for
	// ClassEngine the stream's raw source value (03-hook-contract.md, What the bot sees).
	Reason string
	// Said is ClassEngine's refusal text after the wrapper, for the trace line.
	Said string
	// Ran is whether the call ran, for a bypassed one (04: "ran without" or "refused by the engine").
	Ran bool
	// Cause is a bypassed call's cause when the stdio prompt tool's sources name it (ADR 0029
	// Amendments 2026-11-07 (3)): CauseEngineAnswered or CauseEngineAborted, also the Reason;
	// "" otherwise.
	Cause string
}

// sourceUserPrefix starts the stdio prompt tool's sources for a call the driver answered on
// stdin (user_temporary, user_reject; 03-hook-contract-prompt-tool.md, What the bot sees).
const sourceUserPrefix = "user_"

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

// AskCell is whether a call falls in the policy table's ask cell for the reach: the shared
// policy function over the tool, the reach and the call's paths resolved against the tree, the
// keys Target reads (03-hook-contract.md, What the bot sees). A path is resolved as the hook
// resolves it (askpath.go: `~` and `/<letter>/` expanded, relative to the tree, links followed
// on the existing prefix both ways); a path outside the tree, a credential file's name, a path
// the hook would refuse as an error or an edit with no path is not ask, and the `reviewer`
// cell never is.
func AskCell(tool string, input json.RawMessage, reach, tree string) bool {
	r, ok := policy.ParseReach(reach)
	if !ok || policy.ReviewerCall(tool, subagentType(tool, input)) {
		return false
	}
	var in map[string]json.RawMessage
	_ = json.Unmarshal(input, &in)
	var paths []policy.Path
	for _, key := range []string{"file_path", "notebook_path", "path"} {
		var p string
		if raw, ok := in[key]; ok && json.Unmarshal(raw, &p) == nil && p != "" {
			under, credential, resolved := askUnderTree(p, tree)
			if !resolved {
				return false
			}
			paths = append(paths, policy.Path{Target: p, UnderTree: under, Credential: credential})
		}
	}
	class := policy.ClassOf(tool)
	if class == policy.ClassEdit && len(paths) == 0 {
		return false
	}
	return policy.CellFor(class, r, paths) == policy.CellAsk
}

// Classify reads one tool result (03-hook-contract.md, What the bot sees). The prefixes are
// tested on the stripped text, never by a contains-match: the rest of the text is the model's.
func Classify(tr engine.ToolResultEvent, tool string) Classified {
	return ClassifyIn(tr, tool, false)
}

// ClassifyIn is Classify with auto mode's one exception (03-hook-contract.md, What the bot
// sees): engineAsk, true only in an auto session on a call whose cell for the session's reach
// is ask (AskCell), reads a result whose decider is not the hook as the engine's own decision
// (ClassEngine), allow when the call ran and deny when the engine refused it. On every other
// call a non-hook decider stays hook bypassed, in every mode.
func ClassifyIn(tr engine.ToolResultEvent, tool string, engineAsk bool) Classified {
	if tr.IsError && strings.HasPrefix(strings.TrimSpace(tr.Text), toolUseErrorPrefix) {
		return Classified{Kind: ClassNoDecision}
	}
	stripped := StripWrapper(tr.Text, tool)
	if tr.IsError {
		switch {
		case strings.HasPrefix(stripped, policy.DenyPrefix):
			return Classified{Kind: ClassPolicyDenied, Decision: DecisionDeny, DecidedBy: DecidedByPolicy, Reason: stripped}
		case strings.HasPrefix(stripped, hookFailurePrefix), strings.HasPrefix(stripped, onFailurePrefix):
			return Classified{Kind: ClassHookFailed, Decision: DecisionDeny, DecidedBy: DecidedByHook, Reason: stripped}
		}
	}
	if engineAsk && tr.DecidedBy != sourceHook {
		// Whether the call ran is the stream's permission_decision.decision: a call the engine
		// accepted that then failed (a red test run) is an allow. Only a line without the field
		// falls back to the error flag.
		c := Classified{Kind: ClassEngine, Decision: DecisionAllow, DecidedBy: DecidedByEngine, Reason: tr.DecidedBy}
		refused := tr.PermissionDecision == engine.PermissionReject || tr.PermissionDecision == "" && tr.IsError
		if refused {
			c.Decision, c.Said = DecisionDeny, stripped
		}
		return c
	}
	// The stdio prompt tool's sources on a call the bot holds no request record for (a record
	// takes the result before it gets here, toolResult): a user_* source is the engine answering
	// without the bot, a config source with reason_type other its abort at a stdin close.
	refused := tr.PermissionDecision == engine.PermissionReject || tr.PermissionDecision == "" && tr.IsError
	switch {
	case strings.HasPrefix(tr.DecidedBy, sourceUserPrefix):
		c := Classified{Kind: ClassBypassed, Decision: DecisionAllow, DecidedBy: DecidedByUnknown, Ran: !refused,
			Reason: CauseEngineAnswered, Cause: CauseEngineAnswered}
		if refused {
			c.Decision = DecisionDeny
		}
		return c
	case tr.DecidedBy == sourceConfig && tr.ReasonType == reasonTypeOther:
		return Classified{Kind: ClassBypassed, Decision: DecisionDeny, DecidedBy: DecidedByConfig,
			Reason: CauseEngineAborted, Cause: CauseEngineAborted}
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
