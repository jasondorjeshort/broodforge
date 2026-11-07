// SPDX-License-Identifier: GPL-3.0-or-later

package engine

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"unicode/utf8"
)

// The raw stream layer: a port of spike/run.ts's parseLine, usageDelta and interruptedRow.
// It reads 03's subset of the engine's stream-json (03-architecture.md, The engine adapter)
// and never refuses a line: an unknown field is ignored, an unknown kind or a line that is not
// JSON becomes an unknown event, and a field of the wrong type is left at its zero value.

type rawLine struct {
	Type              string                                `json:"type"`
	Subtype           string                                `json:"subtype"`
	SessionID         string                                `json:"session_id"`
	Model             string                                `json:"model"`
	Tools             []string                              `json:"tools"`
	Capabilities      []string                              `json:"capabilities"`
	Message           *rawMessage                           `json:"message"`
	ParentToolUseID   *string                               `json:"parent_tool_use_id"`
	ToolResultMeta    []rawMeta                             `json:"tool_result_meta"`
	StopReason        string                                `json:"stop_reason"`
	IsError           *bool                                 `json:"is_error"`
	NumTurns          int                                   `json:"num_turns"`
	DeferredToolUse   *rawDeferred                          `json:"deferred_tool_use"`
	PermissionDenials []json.RawMessage                     `json:"permission_denials"`
	ModelUsage        map[string]map[string]json.RawMessage `json:"modelUsage"`
	Usage             json.RawMessage                       `json:"usage"`
	RateLimitInfo     *rawRateLimit                         `json:"rate_limit_info"`
	PermissionMode    string                                `json:"permissionMode"`
	RequestID         string                                `json:"request_id"`
	Request           *rawControlRequest                    `json:"request"`
}

// rawControlRequest is a control_request line's request, the subset a permission ask carries
// (c42-allow's shape, the 2026-11-07 spike).
type rawControlRequest struct {
	Subtype            string          `json:"subtype"`
	ToolName           string          `json:"tool_name"`
	Input              json.RawMessage `json:"input"`
	ToolUseID          string          `json:"tool_use_id"`
	DecisionReason     json.RawMessage `json:"decision_reason"`
	DecisionReasonType string          `json:"decision_reason_type"`
	AgentID            string          `json:"agent_id"`
}

// rawRateLimit is rate_limit_info's subset (03, the stream subset): the top-level
// utilization and resetsAt, seven_day_overage_included and the other fields are not read.
type rawRateLimit struct {
	Status         string `json:"status"`
	RateLimitType  string `json:"rateLimitType"`
	OverageStatus  string `json:"overageStatus"`
	IsUsingOverage bool   `json:"isUsingOverage"`
	UnifiedWindows struct {
		FiveHour json.RawMessage `json:"five_hour"`
		SevenDay json.RawMessage `json:"seven_day"`
	} `json:"unifiedWindows"`
}

// rawWindow keeps a window's fields raw, so window can tell a field absent or mistyped from a
// zero.
type rawWindow struct {
	Utilization json.RawMessage `json:"utilization"`
	ResetsAt    json.RawMessage `json:"resetsAt"`
}

type rawMessage struct {
	ID      string          `json:"id"`
	Model   string          `json:"model"`
	Usage   rawUsage        `json:"usage"`
	Content json.RawMessage `json:"content"` // an array of blocks, or a string
}

type rawUsage struct {
	InputTokens         int `json:"input_tokens"`
	CacheReadTokens     int `json:"cache_read_input_tokens"`
	CacheCreationTokens int `json:"cache_creation_input_tokens"`
}

type rawBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text"`
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Input     json.RawMessage `json:"input"`
	ToolUseID string          `json:"tool_use_id"`
	Content   json.RawMessage `json:"content"`  // a tool_result's: a string, or text blocks
	IsError   *bool           `json:"is_error"` // absent, not false, on success (S1)
}

type rawMeta struct {
	ID                 string `json:"id"`
	PermissionDecision *struct {
		Source     *string `json:"source"`
		Decision   string  `json:"decision"`
		ReasonType string  `json:"reason_type"`
	} `json:"permission_decision"`
}

type rawDeferred struct {
	ID        string          `json:"id"`
	ToolUseID string          `json:"tool_use_id"`
	Name      string          `json:"name"`
	Input     json.RawMessage `json:"input"`
}

// decode unmarshals data into v as far as it fits: a value of the wrong type leaves its field
// at the zero value instead of failing the line. It fails only on JSON that is not well formed.
func decode(data []byte, v any) bool {
	err := json.Unmarshal(data, v)
	var te *json.UnmarshalTypeError
	return err == nil || errors.As(err, &te)
}

// ParseLine maps one stream-json stdout line to its events: none for a blank line, one per
// tool_result block of a user line, one otherwise. A resumed turn's tool_result may come
// before its init, so the parser keeps no state between lines.
func ParseLine(line []byte) []EngineEvent {
	line = bytes.TrimSpace(line)
	if len(line) == 0 {
		return nil
	}
	var o rawLine
	if !json.Valid(line) || line[0] != '{' || !decode(line, &o) {
		return unknownOf(line, "", "")
	}
	switch {
	case o.Type == "system" && o.Subtype == "init":
		return []EngineEvent{{Kind: KindInit, Init: &InitEvent{
			EngineSessionID: o.SessionID,
			Model:           o.Model,
			Tools:           nonNil(o.Tools),
			Capabilities:    nonNil(o.Capabilities),
			PermissionMode:  o.PermissionMode,
		}}}
	case o.Type == "control_request":
		if p := prompt(&o); p != nil {
			return []EngineEvent{{Kind: KindPrompt, Prompt: p}}
		}
	case o.Type == "assistant" && o.Message != nil:
		return []EngineEvent{{Kind: KindAssistant, Assistant: assistant(&o)}}
	case o.Type == "user" && o.Message != nil:
		if out := toolResults(&o); len(out) > 0 {
			return out
		}
	case o.Type == "result":
		return []EngineEvent{{Kind: KindResult, Result: result(&o)}}
	case o.Type == "rate_limit_event":
		return []EngineEvent{{Kind: KindRateLimit, RateLimit: rateLimit(o.RateLimitInfo)}}
	}
	return unknownOf(line, o.Type, o.Subtype)
}

// unknownOf is the one unknown event of a trimmed line: the type and subtype given, the
// line's length and its first UnknownHeadBytes bytes, cut back to a rune boundary.
func unknownOf(line []byte, typ, subtype string) []EngineEvent {
	head := line
	if len(head) > UnknownHeadBytes {
		n := UnknownHeadBytes
		for i := 0; i < utf8.UTFMax-1 && n > 0 && !utf8.RuneStart(line[n]); i++ {
			n--
		}
		head = line[:n]
	}
	return []EngineEvent{{Kind: KindUnknown, Unknown: &UnknownEvent{
		Type: typ, Subtype: subtype, Bytes: len(line), Head: string(head),
	}}}
}

func assistant(o *rawLine) *AssistantEvent {
	m := o.Message
	ev := &AssistantEvent{
		MessageID: m.ID,
		Model:     m.Model,
		Blocks:    []Block{},
		Usage: Usage{
			Input:      m.Usage.InputTokens,
			CacheRead:  m.Usage.CacheReadTokens,
			CacheWrite: m.Usage.CacheCreationTokens,
		},
	}
	if o.ParentToolUseID != nil {
		ev.ParentToolUseID = *o.ParentToolUseID
	}
	for _, b := range blocks(m.Content) {
		switch b.Type {
		case "text":
			ev.Blocks = append(ev.Blocks, Block{Kind: BlockText, Text: b.Text})
		case "tool_use":
			ev.Blocks = append(ev.Blocks, Block{Kind: BlockToolUse, ID: b.ID, Name: b.Name, Input: b.Input})
		}
		// Other block kinds (thinking) are outside 03's subset and dropped.
	}
	return ev
}

func toolResults(o *rawLine) []EngineEvent {
	decided, decision, reasonType := map[string]string{}, map[string]string{}, map[string]string{}
	for _, m := range o.ToolResultMeta {
		if m.PermissionDecision != nil && m.PermissionDecision.Source != nil {
			decided[m.ID] = *m.PermissionDecision.Source
		}
		if m.PermissionDecision != nil {
			decision[m.ID] = m.PermissionDecision.Decision
			reasonType[m.ID] = m.PermissionDecision.ReasonType
		}
	}
	var out []EngineEvent
	for _, b := range blocks(o.Message.Content) {
		if b.Type != "tool_result" {
			continue
		}
		out = append(out, EngineEvent{Kind: KindToolResult, ToolResult: &ToolResultEvent{
			ToolUseID:          b.ToolUseID,
			IsError:            b.IsError != nil && *b.IsError,
			Text:               resultText(b.Content),
			DecidedBy:          decided[b.ToolUseID],
			ReasonType:         reasonType[b.ToolUseID],
			PermissionDecision: decision[b.ToolUseID],
		}})
	}
	return out
}

func result(o *rawLine) *ResultEvent {
	ev := &ResultEvent{
		StopReason:        o.StopReason,
		IsError:           o.IsError != nil && *o.IsError,
		NumTurns:          o.NumTurns,
		PermissionDenials: o.PermissionDenials,
		ModelUsage:        map[string]ModelUsage{},
		Usage:             o.Usage,
	}
	if ev.PermissionDenials == nil {
		ev.PermissionDenials = []json.RawMessage{}
	}
	if len(ev.Usage) == 0 {
		ev.Usage = json.RawMessage("null")
	}
	for model, fields := range o.ModelUsage {
		u := ModelUsage{}
		for name, raw := range fields {
			var n float64
			if json.Unmarshal(raw, &n) == nil {
				u[name] = n
			}
		}
		ev.ModelUsage[model] = u
	}
	if d := o.DeferredToolUse; d != nil {
		id := d.ID
		if id == "" {
			id = d.ToolUseID
		}
		ev.Deferred = &DeferredTool{ToolUseID: id, Name: d.Name, Input: d.Input}
	}
	return ev
}

// prompt is the event of a control_request line asking whether a tool call may run, nil for
// one that does not parse as such (another subtype, no tool name or tool_use_id), which stays
// unknown. The reason is decision_reason as a string (a hook's `ask`, c42-allow), or as an
// object's "reason" or "message" field (the Auto mode's form, unobserved, C40).
func prompt(o *rawLine) *PromptEvent {
	r := o.Request
	if r == nil || r.Subtype != "can_use_tool" || r.ToolName == "" || r.ToolUseID == "" {
		return nil
	}
	p := &PromptEvent{RequestID: o.RequestID, ToolUseID: r.ToolUseID, Tool: r.ToolName, Input: r.Input,
		DecisionReasonType: r.DecisionReasonType, AgentID: r.AgentID}
	if len(p.Input) == 0 {
		p.Input = json.RawMessage("null")
	}
	var s string
	var obj struct {
		Reason  string `json:"reason"`
		Message string `json:"message"`
	}
	switch {
	case json.Unmarshal(r.DecisionReason, &s) == nil:
		p.Reason = s
	case json.Unmarshal(r.DecisionReason, &obj) == nil:
		p.Reason = obj.Reason
		if p.Reason == "" {
			p.Reason = obj.Message
		}
	}
	return p
}

// rateLimit is the event of a rate_limit_event line; an absent rate_limit_info or field is
// the zero value and an absent window nil, so a line without unifiedWindows still records when
// it was seen.
func rateLimit(r *rawRateLimit) *RateLimitEvent {
	if r == nil {
		return &RateLimitEvent{}
	}
	w := r.UnifiedWindows
	return &RateLimitEvent{
		Status:         r.Status,
		RateLimitType:  r.RateLimitType,
		FiveHour:       window(w.FiveHour),
		SevenDay:       window(w.SevenDay),
		OverageStatus:  r.OverageStatus,
		IsUsingOverage: r.IsUsingOverage,
	}
}

// window is one unifiedWindows entry, nil unless it is an object carrying a utilization or a
// resetsAt of the right type (null, a string, or an object of neither is no window).
func window(raw json.RawMessage) *Window {
	var rw rawWindow
	if len(raw) == 0 || raw[0] != '{' || !decode(raw, &rw) {
		return nil
	}
	var w Window
	hasU := number(rw.Utilization, &w.Utilization)
	hasR := number(rw.ResetsAt, &w.ResetsAt)
	if !hasU && !hasR {
		return nil
	}
	return &w
}

// number reads raw into v when it is a JSON number v's type holds; false, v untouched, for an
// absent field, null, or another type.
func number[T float64 | int64](raw json.RawMessage, v *T) bool {
	var n T
	if len(raw) == 0 || string(raw) == "null" || json.Unmarshal(raw, &n) != nil {
		return false
	}
	*v = n
	return true
}

// blocks reads a content field as blocks; a string or anything else is no blocks.
func blocks(content json.RawMessage) []rawBlock {
	var bs []rawBlock
	if len(content) == 0 || content[0] != '[' || !decode(content, &bs) {
		return nil
	}
	return bs
}

// resultText is a tool_result's content as text: the string itself, or its text blocks
// joined by newlines.
func resultText(content json.RawMessage) string {
	var s string
	if len(content) > 0 && content[0] == '"' && json.Unmarshal(content, &s) == nil {
		return s
	}
	var parts []string
	for _, b := range blocks(content) {
		if b.Type == "text" {
			parts = append(parts, b.Text)
		}
	}
	return strings.Join(parts, "\n")
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// encodeJSON is v as compact JSON without HTML escaping (events.go's MarshalJSON).
func encodeJSON(v any) ([]byte, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(b.Bytes(), "\n"), nil
}

// notSummed are modelUsage fields that are limits, not counts.
var notSummed = map[string]bool{"contextWindow": true, "maxOutputTokens": true}

// UsageDelta is a turn's usage record per model (03, Usage delta): the result's cumulative
// modelUsage minus the baseline, clamped at zero, limits left out. The baseline is the
// cumulative of the last result of the same engine session, never an interrupted turn's row
// (S2); nil for the session's first result.
func UsageDelta(cur, baseline map[string]ModelUsage) map[string]ModelUsage {
	delta := map[string]ModelUsage{}
	for model, u := range cur {
		d := ModelUsage{}
		for f, v := range u {
			if notSummed[f] {
				continue
			}
			d[f] = max(0, v-baseline[model][f])
		}
		delta[model] = d
	}
	return delta
}

// InterruptedRow is the record of a turn with no result (03, Usage delta): the assistant
// counts deduplicated by message id (one line per content block shares one usage, S1),
// summed per model as inputTokens, cacheReadInputTokens and cacheCreationInputTokens; output
// is unknown and left out.
func InterruptedRow(events []EngineEvent) map[string]ModelUsage {
	seen := map[string]*AssistantEvent{}
	for _, ev := range events {
		if ev.Kind == KindAssistant && ev.Assistant != nil {
			seen[ev.Assistant.MessageID] = ev.Assistant
		}
	}
	row := map[string]ModelUsage{}
	for _, a := range seen {
		model := a.Model
		if model == "" {
			model = "(no model)"
		}
		r := row[model]
		if r == nil {
			r = ModelUsage{}
			row[model] = r
		}
		r["inputTokens"] += float64(a.Usage.Input)
		r["cacheReadInputTokens"] += float64(a.Usage.CacheRead)
		r["cacheCreationInputTokens"] += float64(a.Usage.CacheWrite)
	}
	return row
}

// TurnUsage is one turn's usage as 03 records it.
type TurnUsage struct {
	// Delta is the turn's record: UsageDelta against the baseline, or InterruptedRow.
	Delta map[string]ModelUsage
	// Cumulative is the result's modelUsage, the baseline of the session's next turn; nil
	// for an interrupted turn, whose row stands outside the baseline (S2).
	Cumulative  map[string]ModelUsage
	Interrupted bool
}

// UsageOf reads a turn's events against the baseline (the previous TurnUsage's Cumulative
// that was not nil, for the same engine session). The last result of the turn counts.
func UsageOf(events []EngineEvent, baseline map[string]ModelUsage) TurnUsage {
	var res *ResultEvent
	for _, ev := range events {
		if ev.Kind == KindResult && ev.Result != nil {
			res = ev.Result
		}
	}
	if res == nil {
		return TurnUsage{Delta: InterruptedRow(events), Interrupted: true}
	}
	return TurnUsage{Delta: UsageDelta(res.ModelUsage, baseline), Cumulative: res.ModelUsage}
}
