// SPDX-License-Identifier: GPL-3.0-or-later

package policy

import "strings"

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

// modelNames maps a model id prefix to the display name the trailers use (03-architecture.md,
// Attribution). The longest matching prefix wins; an id no prefix matches is used verbatim.
var modelNames = map[string]string{
	"claude-opus-5-5":   "Claude Opus 5.5",
	"claude-sonnet-5-5": "Claude Sonnet 5.5",
	"claude-haiku-5-5":  "Claude Haiku 5.5",
}

// ModelDisplayName is the display name for a model id, such as "Claude Haiku 5.5" for
// "claude-haiku-5-5" or a dated "claude-haiku-5-5-20261001".
func ModelDisplayName(id string) string {
	best, name := 0, id
	for prefix, n := range modelNames {
		if len(prefix) > best && (id == prefix || strings.HasPrefix(id, prefix+"-")) {
			best, name = len(prefix), n
		}
	}
	return name
}
