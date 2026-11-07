// SPDX-License-Identifier: GPL-3.0-or-later

package engine

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// The prompt-mode fixture check (ADR 0029 rules 8 and 9; 03-hook-contract-prompt-tool.md, What
// the bot sees and Fixtures), over every fixtures/engine/prompt-* directory: a pin that changes
// the stdio prompt tool's protocol fails here, not in a session.

// promptAnswer is the outcome the recording gave one request, read from the run's sent.jsonl
// (the 2026-11-07 spike finding), and so the meta its call's result carries.
type promptAnswer int

const (
	answerAllow     promptAnswer = iota // allow: source user_temporary, decision accept (c42-allow)
	answerDeny                          // deny, interrupt false: source user_reject, decision reject (c42-deny)
	answerAbort                         // the engine's own deny at a stdin close: source config, reason_type other (c43a)
	answerInterrupt                     // deny, interrupt true: no tool_result_meta at all (c49)
	answerNone                          // never answered: the call reaches no result in the turn (c47-kill)
)

// promptAnswers is, per prompt-* fixture and turn, each request's answer in the order the
// requests appear on the stream. A new prompt-* fixture adds its row here.
var promptAnswers = map[string]map[int][]promptAnswer{
	"prompt-ask-allow":         {1: {answerAllow}},
	"prompt-ask-deny":          {1: {answerDeny, answerAllow}},
	"prompt-ask-two":           {1: {answerDeny, answerAllow}},
	"prompt-defer":             {1: {answerAllow, answerAllow}, 2: {}},
	"prompt-ask-background":    {1: {answerAllow}},
	"prompt-ask-abort":         {1: {answerAbort}},
	"prompt-ask-interrupt":     {1: {answerInterrupt}, 2: {answerAllow}},
	"prompt-ask-kill":          {1: {answerNone}, 2: {}},
	"prompt-ask-no-tool":       {1: {}},
	"prompt-ask-session-allow": {1: {answerAllow, answerAllow}},
}

// noToolFixture is rule 8's fixture: a hook `ask` with no request on the stream.
const noToolFixture = "prompt-ask-no-tool"

// hookStdinEntry is the subset of one hook stdin entry the check reads. A field of another type
// than string fails the decode.
type hookStdinEntry struct {
	AgentID   *string `json:"agent_id"`
	AgentType *string `json:"agent_type"`
	ToolName  string  `json:"tool_name"`
	ToolUseID string  `json:"tool_use_id"`
}

// hookStdin is turn n's hook-stdin-<n>.json in dir, none when the file is absent.
func hookStdin(t *testing.T, dir string, n int) []hookStdinEntry {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, "hook-stdin-"+strconv.Itoa(n)+".json"))
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	var entries []hookStdinEntry
	if err := json.Unmarshal(b, &entries); err != nil {
		t.Fatalf("%s hook-stdin-%d.json: %v", dir, n, err)
	}
	return entries
}

func promptFixtureDirs(t *testing.T) []string {
	t.Helper()
	var dirs []string
	for _, dir := range fixtureDirs(t) {
		if strings.HasPrefix(filepath.Base(dir), "prompt-") {
			dirs = append(dirs, dir)
		}
	}
	return dirs
}

// TestPromptFixtures: on every prompt-* fixture, (1) every hook-stdin-<n>.json entry carries
// agent_id and (2) agent_type, both non-empty strings (rule 9); (3) each request's call ends in
// the result meta of the answer the recording gave (What the bot sees: user_temporary,
// user_reject, config with reason_type other, or none on the interrupting deny); (4) every call
// the hook answered `ask` in hook-stdin-<n>.json has its control_request on the stream, under the
// entry's agent_id with decision_reason_type hook, and every subagent's request has its entry;
// (5) in prompt-ask-no-tool, the opposite: no request for such a call (rule 8's invariant). A call
// the hook answered `ask` is one whose result does not carry source hook: the hook's own allow or
// deny is the only answer that does.
func TestPromptFixtures(t *testing.T) {
	seen := map[string]bool{}
	for _, dir := range promptFixtureDirs(t) {
		name := filepath.Base(dir)
		seen[name] = true
		want, ok := promptAnswers[name]
		if !ok {
			t.Errorf("%s: no row in promptAnswers", name)
			continue
		}
		fx, err := ReadFixture(dir)
		if err != nil {
			t.Fatal(err)
		}
		if len(fx.Turns) != len(want) {
			t.Errorf("%s: %d turns, promptAnswers has %d", name, len(fx.Turns), len(want))
		}
		asked := 0
		for _, turn := range fx.Turns {
			var prompts []*PromptEvent
			results := map[string]*ToolResultEvent{}
			for _, ev := range parseTurn(t, dir, turn.N) {
				switch ev.Kind {
				case KindPrompt:
					prompts = append(prompts, ev.Prompt)
				case KindToolResult:
					results[ev.ToolResult.ToolUseID] = ev.ToolResult
				case KindInit, KindAssistant, KindResult, KindExit, KindRateLimit, KindUnknown:
				}
			}

			// (3) Each request's result.
			answers := want[turn.N]
			if len(prompts) != len(answers) {
				t.Errorf("%s turn %d: %d requests, promptAnswers has %d", name, turn.N, len(prompts), len(answers))
			}
			for k := range min(len(prompts), len(answers)) {
				checkPromptResult(t, dir, name, turn.N, prompts[k], results[prompts[k].ToolUseID], answers[k])
			}

			// (1), (2), (4), (5) over the turn's hook stdin.
			byID := map[string]*PromptEvent{}
			for _, p := range prompts {
				byID[p.ToolUseID] = p
			}
			entries := hookStdin(t, dir, turn.N)
			stdinIDs := map[string]bool{}
			for k, e := range entries {
				at := name + " hook-stdin-" + strconv.Itoa(turn.N) + ".json entry " + strconv.Itoa(k+1)
				if e.AgentID == nil || *e.AgentID == "" {
					t.Errorf("%s: no agent_id", at)
					continue
				}
				if e.AgentType == nil || *e.AgentType == "" {
					t.Errorf("%s: no agent_type", at)
				}
				stdinIDs[e.ToolUseID] = true
				if r := results[e.ToolUseID]; r != nil && r.DecidedBy == "hook" {
					continue // the hook allowed or denied it itself
				}
				asked++
				p := byID[e.ToolUseID]
				switch {
				case name == noToolFixture && p != nil:
					t.Errorf("%s: %s %s has a request, want none (rule 8's fixture)", at, e.ToolName, e.ToolUseID)
				case name == noToolFixture:
				case p == nil:
					t.Errorf("%s: the hook asked for %s %s and no control_request came", at, e.ToolName, e.ToolUseID)
				case p.AgentID != *e.AgentID || p.DecisionReasonType != "hook":
					t.Errorf("%s: request agentId %q decisionReasonType %q, want %q and hook", at, p.AgentID, p.DecisionReasonType, *e.AgentID)
				}
			}
			for _, p := range prompts {
				if p.AgentID != "" && !stdinIDs[p.ToolUseID] {
					t.Errorf("%s turn %d: request for %s from agent %s has no hook stdin entry", name, turn.N, p.ToolUseID, p.AgentID)
				}
			}
		}
		if name == noToolFixture && asked == 0 {
			t.Errorf("%s: no call the hook asked for, so rule 8's invariant is not exercised", name)
		}
	}
	for name := range promptAnswers {
		if !seen[name] {
			t.Errorf("promptAnswers names %s, which is not under %s", name, fixturesRoot)
		}
	}
}

// hasRawMeta reports whether any line of turn n's turn-<n>.jsonl in dir carries a tool_result_meta
// entry with the given id, whatever else the entry holds.
func hasRawMeta(t *testing.T, dir string, n int, id string) bool {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, "turn-"+strconv.Itoa(n)+".jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(b), "\n") {
		var l struct {
			Meta []struct {
				ID string `json:"id"`
			} `json:"tool_result_meta"`
		}
		if json.Unmarshal([]byte(line), &l) != nil {
			continue
		}
		for _, m := range l.Meta {
			if m.ID == id {
				return true
			}
		}
	}
	return false
}

// checkPromptResult asserts the result of one request's call against the recording's answer.
func checkPromptResult(t *testing.T, dir, name string, n int, p *PromptEvent, r *ToolResultEvent, a promptAnswer) {
	t.Helper()
	at := name + " turn " + strconv.Itoa(n) + " " + p.ToolUseID
	if a == answerNone {
		if r != nil {
			t.Errorf("%s: a result %+v, want none (the call dropped)", at, r)
		}
		return
	}
	if r == nil {
		t.Errorf("%s: no result for the request's call", at)
		return
	}
	if a == answerInterrupt && hasRawMeta(t, dir, n, p.ToolUseID) {
		t.Errorf("%s: a tool_result_meta entry on the raw turn line, want none (the interrupting deny)", at)
	}
	var source, reasonType, decision string
	switch a {
	case answerAllow:
		source, decision = "user_temporary", PermissionAccept
	case answerDeny:
		source, decision = "user_reject", PermissionReject
	case answerAbort:
		source, reasonType, decision = "config", "other", PermissionReject
	case answerInterrupt:
	case answerNone:
	}
	if r.DecidedBy != source || r.ReasonType != reasonType || r.PermissionDecision != decision {
		t.Errorf("%s: source %q reason_type %q decision %q, want %q %q %q", at, r.DecidedBy, r.ReasonType, r.PermissionDecision, source, reasonType, decision)
	}
}
