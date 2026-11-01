// SPDX-License-Identifier: GPL-3.0-or-later

package policy

import (
	"encoding/json"
	"strconv"
)

// Pinned is the model a subagent call pins (ADR 0025 rule 10 as amended 2026-10-31): Model is
// the id ResolveModel gives, "" when nothing pins, and Why then says why, for hook.log.
type Pinned struct {
	Model string
	Why   string
}

// Name is the pinned model's display name, "none" when nothing pins: the subagent-result row's
// `model` (05-debug-interface-git.md, Required trailers).
func (p Pinned) Name() string {
	if p.Model == "" {
		return "none"
	}
	return ModelDisplayName(p.Model)
}

// Pin is the model an Agent (or Task) call pins, the one rule the hook and session share: the
// call's tool_input.model first, else the `model` of its subagent_type in agents (the copy's
// agents.json; nil when the session has no copy or the copy defines no agent), through
// ResolveModel. An unpinned, unknown or unresolvable type pins nothing. Pure: the callers read
// the file.
func Pin(toolInput, agents []byte) Pinned {
	var in struct {
		Model        string `json:"model"`
		SubagentType string `json:"subagent_type"`
	}
	_ = json.Unmarshal(toolInput, &in) // a tool_input the hook could not read never reaches here
	if in.Model != "" {
		if id, ok := ResolveModel(in.Model); ok {
			return Pinned{Model: id}
		}
		return Pinned{Why: "tool_input.model " + strconv.Quote(in.Model) + " is no model the alias table knows"}
	}
	typ := strconv.Quote(in.SubagentType)
	if agents == nil {
		return Pinned{Why: "subagent_type " + typ + ": the session's copy defines no agent"}
	}
	var defs map[string]struct {
		Model string `json:"model"`
	}
	if err := json.Unmarshal(agents, &defs); err != nil {
		return Pinned{Why: "agents.json: " + err.Error()}
	}
	def, ok := defs[in.SubagentType]
	switch {
	case !ok:
		return Pinned{Why: "subagent_type " + typ + " is not in the copy's agents.json"}
	case def.Model == "":
		return Pinned{Why: "subagent_type " + typ + " names no model"}
	}
	id, ok := ResolveModel(def.Model)
	if !ok {
		return Pinned{Why: "subagent_type " + typ + " names model " + strconv.Quote(def.Model) + ", which the alias table does not know"}
	}
	return Pinned{Model: id}
}
