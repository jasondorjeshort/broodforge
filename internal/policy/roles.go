// SPDX-License-Identifier: GPL-3.0-or-later

package policy

import (
	"regexp"
	"strings"
)

// RoleDriver in an `answer` list means the session's driver only (03-architecture.md, R-25).
const RoleDriver = "driver"

// Permitted is the role check (03-architecture.md, Registration and participants): a pure
// function of the roles a person holds at that moment, the channel's list for the action
// (`start`, `contribute` or `answer`), the instance's roleGroups and whether the person is the
// session's driver. An entry of the list is a group name, a raw role id, or "driver".
func Permitted(held, list []string, groups map[string][]string, isDriver bool) bool {
	has := map[string]bool{}
	for _, r := range held {
		has[r] = true
	}
	for _, entry := range list {
		if entry == RoleDriver {
			if isDriver {
				return true
			}
			continue
		}
		if ids, ok := groups[entry]; ok {
			for _, id := range ids {
				if has[id] {
					return true
				}
			}
			continue
		}
		if has[entry] {
			return true
		}
	}
	return false
}

// models maps a model id prefix to the display name the trailers use (03-architecture.md,
// Attribution). The longest matching prefix wins; an id no prefix matches is used verbatim.
// The order is the alias list's (04-ux-session-controls.md, Model and effort): an alias is the lowercase second word
// of a display name, and the first entry with that word owns it.
var models = []struct{ id, name string }{
	{"claude-opus-5-5", "Claude Opus 5.5"},
	{"claude-sonnet-5-5", "Claude Sonnet 5.5"},
	{"claude-haiku-5-5", "Claude Haiku 5.5"},
	{"claude-fable-5-1", "Claude Fable 5.1"},
}

// matchModel is the entry an id belongs to: the longest prefix equal to the id or followed by
// a dash ("claude-haiku-5-5-20261001"); false for none.
func matchModel(id string) (int, bool) {
	best, at := 0, -1
	for i, m := range models {
		if len(m.id) > best && (id == m.id || strings.HasPrefix(id, m.id+"-")) {
			best, at = len(m.id), i
		}
	}
	return at, at >= 0
}

// ModelDisplayName is the display name for a model id, such as "Claude Haiku 5.5" for
// "claude-haiku-5-5" or a dated "claude-haiku-5-5-20261001".
func ModelDisplayName(id string) string {
	if i, ok := matchModel(id); ok {
		return models[i].name
	}
	return id
}

// aliasOf is a display name's alias: "sonnet" for "Claude Sonnet 5.5".
func aliasOf(name string) string {
	f := strings.Fields(name)
	if len(f) < 2 {
		return ""
	}
	return strings.ToLower(f[1])
}

// ModelAliases are the aliases a person may type, in the map's order: opus, sonnet, haiku,
// fable.
func ModelAliases() []string {
	var out []string
	seen := map[string]bool{}
	for _, m := range models {
		if a := aliasOf(m.name); a != "" && !seen[a] {
			seen[a] = true
			out = append(out, a)
		}
	}
	return out
}

// dateSuffixRE is the one suffix a typed id may carry after the map's id: a release date.
var dateSuffixRE = regexp.MustCompile(`^-[0-9]{8}$`)

// ResolveModel is the model id a person's value names (05-debug-interface-discord.md, Model
// and effort): an alias (in any case) is its entry's id; a full id the map holds, bare or with a
// date suffix (`claude-haiku-5-5-20261001`), passes as typed; anything else is false, so nothing
// unknown reaches the engine's --model.
func ResolveModel(v string) (string, bool) {
	for _, m := range models {
		if a := aliasOf(m.name); a != "" && strings.EqualFold(v, a) {
			return m.id, true
		}
	}
	for _, m := range models {
		if rest, ok := strings.CutPrefix(v, m.id); ok && (rest == "" || dateSuffixRE.MatchString(rest)) {
			return v, true
		}
	}
	return "", false
}

// Efforts are the effort levels a session may set, in order (04-ux-session-controls.md, Model
// and effort): the pinned engine's --effort list (`claude --help`: low, medium, high, xhigh,
// max), in its order; xhigh was added by the owner, 2026-10-24.
var Efforts = []string{"low", "medium", "high", "xhigh", "max"}

// ParseEffort reports whether v is one of Efforts.
func ParseEffort(v string) bool {
	for _, e := range Efforts {
		if v == e {
			return true
		}
	}
	return false
}
