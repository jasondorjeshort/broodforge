// SPDX-License-Identifier: GPL-3.0-or-later

package engine

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// Engine is one way of running a turn (03-architecture.md, The engine adapter): "claude-code"
// or "fake". StartTurn returns an error when the turn cannot start at all (no fixture file, no
// process); everything after the start arrives on the handle's Events.
type Engine interface {
	Name() string
	StartTurn(req TurnRequest) (TurnHandle, error)
}

// TurnRequest is everything one turn needs. The spawn backend joins it with the spawn
// package (P7); the fake records the rest as received-<n>.json.
type TurnRequest struct {
	SessionID       string      `json:"sessionId"`
	EngineSessionID string      `json:"engineSessionId"` // --resume when set
	Cwd             string      `json:"cwd"`             // sessions/<id>/tree
	Delivery        Delivery    `json:"delivery"`
	GitIdentity     GitIdentity `json:"gitIdentity"` // the driver's, all four variables
	HookEnv         []HookVar   `json:"hookEnv"`     // the BROODFORGE_* names of the hook contract
	Input           string      `json:"input"`       // the From-prefixed blocks
	Answer          *TurnAnswer `json:"answer"`
	Model           string      `json:"model"`
	Effort          string      `json:"effort"`
	// SystemPromptFile is --append-system-prompt-file, passed on every process: the session's
	// workflow copy's instructions.md, the header first (ADR 0025 rules 1 and 5). The engine
	// renders it once per conversation, so the file holds nothing that changes in a session.
	SystemPromptFile string `json:"systemPromptFile"`
	// AgentsFile is --agents, the copy's agents.json; "" when the copy defines no agent.
	AgentsFile string `json:"agentsFile"`
}

// Delivery is how the credential reaches the engine: Kind "config-dir" (Path) or "api-key"
// (CredentialID).
type Delivery struct {
	Kind         string `json:"kind"`
	Path         string `json:"path,omitempty"`
	CredentialID string `json:"credentialId,omitempty"`
}

// GitIdentity is the driver's name and email, set as author and committer.
type GitIdentity struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// HookVar is one variable of the hook environment (03-hook-contract.md).
type HookVar struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// TurnAnswer is a person's answer to a deferred call, given on the resume.
type TurnAnswer struct {
	ToolUseID string `json:"toolUseId"`
	Decision  string `json:"decision"` // "allow" | "deny"
}

// TurnHandle is a running turn. Events carries the turn's events, the last of them an exit,
// and is then closed; a reader drains it to the close, or calls Interrupt, after which it may
// stop reading without leaving the turn blocked. Interrupt ends the turn early.
type TurnHandle struct {
	Events    <-chan EngineEvent
	Interrupt func() error
}

// EventKind names the payload an EngineEvent carries; it is the "type" of its JSON form.
type EventKind string

const (
	KindInit       EventKind = "init"
	KindAssistant  EventKind = "assistant"
	KindToolResult EventKind = "tool-result"
	KindResult     EventKind = "result"
	KindExit       EventKind = "exit"
	KindRateLimit  EventKind = "rate-limit"
	KindUnknown    EventKind = "unknown"
)

// EngineEvent is a sum type (ADR 0023, Types and JSON): Kind says which one payload is set.
// Unknown stands for any other line, logged at debug level and never fatal.
type EngineEvent struct {
	Kind       EventKind
	Init       *InitEvent
	Assistant  *AssistantEvent
	ToolResult *ToolResultEvent
	Result     *ResultEvent
	Exit       *ExitEvent
	RateLimit  *RateLimitEvent
	Unknown    *UnknownEvent
}

// UnknownEvent is a line the parser does not map: its type and subtype when it is a JSON
// object (empty otherwise), its byte length and its first UnknownHeadBytes bytes cut back to
// a rune boundary, the whole line only when shorter (the engine's commands_changed runs to
// pages). Invalid UTF-8 in a head marshals as U+FFFD; Bytes stays the line's true length.
// The recording keeps the line.
type UnknownEvent struct {
	Type    string `json:"eventType"`
	Subtype string `json:"subtype"`
	Bytes   int    `json:"bytes"`
	Head    string `json:"head"`
}

// UnknownHeadBytes is the most of an unknown line its event carries.
const UnknownHeadBytes = 200

// InitEvent is the engine's system/init line.
type InitEvent struct {
	EngineSessionID string   `json:"engineSessionId"`
	Model           string   `json:"model"`
	Tools           []string `json:"tools"`
	Capabilities    []string `json:"capabilities"`
}

// AssistantEvent is one assistant line. The engine writes one line per content block, all
// sharing MessageID and Usage (ADR 0020, S1), so a count deduplicates by MessageID first.
type AssistantEvent struct {
	MessageID       string  `json:"messageId"`
	Model           string  `json:"model"`
	ParentToolUseID string  `json:"parentToolUseId,omitempty"`
	Blocks          []Block `json:"blocks"`
	Usage           Usage   `json:"usage"`
}

// ToolResultEvent is one tool_result block of a user line. DecidedBy is the matching
// tool_result_meta entry's permission_decision.source ("hook" or "config"), empty when the
// line carries none (C39).
type ToolResultEvent struct {
	ToolUseID string `json:"toolUseId"`
	IsError   bool   `json:"isError"`
	Text      string `json:"text"`
	DecidedBy string `json:"decidedBy,omitempty"`
}

// ResultEvent is the turn's result line. ModelUsage is cumulative per engine session (S2).
type ResultEvent struct {
	StopReason        string                `json:"stopReason"`
	IsError           bool                  `json:"isError"`
	NumTurns          int                   `json:"numTurns"`
	Deferred          *DeferredTool         `json:"deferred,omitempty"`
	PermissionDenials []json.RawMessage     `json:"permissionDenials"`
	ModelUsage        map[string]ModelUsage `json:"modelUsage"`
	Usage             json.RawMessage       `json:"usage"`
}

// ExitEvent ends every turn. Code is nil when the process did not exit by itself.
type ExitEvent struct {
	Code        *int   `json:"code"`
	Signal      string `json:"signal,omitempty"`
	Interrupted bool   `json:"interrupted"`
}

// RateLimitEvent is a rate_limit_event line's rate_limit_info: the plan meters of the account
// the engine runs as (ADR 0020, Amendments 2026-10-30). An absent field is its zero value.
type RateLimitEvent struct {
	Status         string `json:"status"`
	RateLimitType  string `json:"rateLimitType"`
	FiveHour       Window `json:"fiveHour"`
	SevenDay       Window `json:"sevenDay"`
	OverageStatus  string `json:"overageStatus"`
	IsUsingOverage bool   `json:"isUsingOverage"`
}

// Window is one of unifiedWindows' meters: Utilization a fraction, ResetsAt unix seconds.
type Window struct {
	Utilization float64 `json:"utilization"`
	ResetsAt    int64   `json:"resetsAt"`
}

// Usage is an assistant message's input-side counts.
type Usage struct {
	Input      int `json:"input"`
	CacheRead  int `json:"cacheRead"`
	CacheWrite int `json:"cacheWrite"`
}

// ModelUsage is one model's entry of a result's modelUsage: its numeric fields by the
// engine's names (inputTokens, outputTokens, costUSD, ...); non-numeric fields are dropped.
type ModelUsage map[string]float64

// DeferredTool is the call a tool_deferred result names (the last of its message, C28).
type DeferredTool struct {
	ToolUseID string          `json:"toolUseId"`
	Name      string          `json:"name"`
	Input     json.RawMessage `json:"input"`
}

// BlockKind is an assistant content block's kind; other block kinds (thinking) are dropped.
type BlockKind string

const (
	BlockText    BlockKind = "text"
	BlockToolUse BlockKind = "tool-use"
)

// Block is one assistant content block: Text for text, ID, Name and Input for tool-use.
type Block struct {
	Kind  BlockKind       `json:"kind"`
	Text  string          `json:"text,omitempty"`
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
}

// MarshalJSON writes the event flat, its kind as "type" first and the payload's fields after
// it: {"type":"init","engineSessionId":...}; an unknown event is
// {"type":"unknown","eventType":...,"subtype":...,"bytes":...,"head":...}. HTML characters
// are not escaped, so a tool's text reads as the engine wrote it.
func (e EngineEvent) MarshalJSON() ([]byte, error) {
	var body []byte
	var err error
	switch e.Kind {
	case KindInit:
		body, err = encodeJSON(e.Init)
	case KindAssistant:
		body, err = encodeJSON(e.Assistant)
	case KindToolResult:
		body, err = encodeJSON(e.ToolResult)
	case KindResult:
		body, err = encodeJSON(e.Result)
	case KindExit:
		body, err = encodeJSON(e.Exit)
	case KindRateLimit:
		body, err = encodeJSON(e.RateLimit)
	case KindUnknown:
		body, err = encodeJSON(e.Unknown)
	default:
		return nil, fmt.Errorf("engine event: unknown kind %q", e.Kind)
	}
	if err != nil {
		return nil, err
	}
	kind, err := encodeJSON(string(e.Kind))
	if err != nil {
		return nil, err
	}
	var b bytes.Buffer
	b.WriteString(`{"type":`)
	b.Write(kind)
	if len(body) > 2 && body[0] == '{' {
		b.WriteByte(',')
		b.Write(body[1:])
	} else {
		b.WriteByte('}')
	}
	return b.Bytes(), nil
}
