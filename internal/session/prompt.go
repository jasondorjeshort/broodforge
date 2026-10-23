// SPDX-License-Identifier: GPL-3.0-or-later

package session

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jasondorjeshort/broodforge/internal/policy"
)

// summaryRunes is how much of a text or a command a summary or trace target shows (04, The
// turn loop: the first 80 characters of a command; 05: the text's first 80 characters).
const summaryRunes = 80

// FromBlock is one message handed to the agent (R-14, 03 Data flow step 2): a `From:` line of
// the writer's git name and session role, then the text.
func FromBlock(name, role, text string) string {
	return "From: " + name + " (" + role + ")\n" + text
}

// JoinBlocks makes a turn's input: every queued message's block, in order, a blank line apart.
func JoinBlocks(blocks []string) string { return strings.Join(blocks, "\n\n") }

// reachMeaning is 02-domain.md's sentence for each reach.
func reachMeaning(reach string) string {
	if reach == policy.ReachReadOnly.String() {
		return "read the project, write nothing"
	}
	return "change only this session's tree"
}

// Header is what the header of a session's appended prompt states (ADR 0025 rule 1). The engine
// renders the prompt once per conversation (until a compaction), so nothing here changes during
// a session: no mode, no model, no trailer list (rule 10).
type Header struct {
	Session, ProjectName, Reach, DriverName string
	// Copied is whether the project's instructions follow the header: false under inject none
	// or when the default branch has neither CLAUDE.md nor .claude, and the taken-from line is
	// then left out. Branch, Commit and Date are that line's: the default branch and the commit
	// the copy was taken from, with its committer date.
	Copied         bool
	Branch, Commit string
	Date           time.Time
}

// The header's two fixed sentences, the ADR's verbatim (rule 1, the owner's; rule 10, its first
// letter capitalised to open a sentence).
const (
	headerPrecedence = "This session is run by Broodforge. Where the project's instructions below conflict with this header, " +
		"this header wins; it never says how the project's work is done."
	headerTrailerRule = "Every commit carries the Co-Authored-By lines Broodforge's commit-msg hook adds; never remove one."
)

// SystemPromptAppend is the header that opens the appended prompt, every line of it ending in a
// newline and none blank, so the first blank line of instructions.md ends it (ReadHeader): the
// precedence sentence, the session and project, the reach, the driver, one call at a time, the
// trailer rule, and where the project's instructions were taken from. It is the one place the
// header's text lives; `session show --header <id>` prints it as the session got it.
func SystemPromptAppend(h Header) string {
	var b strings.Builder
	b.WriteString(headerPrecedence + "\n")
	b.WriteString("You are working in Broodforge session " + h.Session + " on " + h.ProjectName + ".\n")
	b.WriteString("Reach: " + h.Reach + " (" + reachMeaning(h.Reach) + ").\n")
	b.WriteString("Each message starts with a From: line naming its writer and their role; the driver is " + h.DriverName + ".\n")
	b.WriteString("Make one tool call at a time: calls made together are asked about one by one.\n")
	b.WriteString(headerTrailerRule + "\n")
	if h.Copied {
		b.WriteString("The project's instructions below were taken from " + h.Branch + " at " + h.Commit + " on " +
			h.Date.UTC().Format(time.RFC3339) + ".\n")
	}
	return b.String()
}

// firstRunes is s on one line, cut to n runes with an ellipsis.
func firstRunes(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	r := []rune(s)
	return string(r[:n-1]) + "…"
}

// firstLine is the first non-empty line of s, trimmed.
func firstLine(s string) string {
	for _, l := range strings.Split(s, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			return l
		}
	}
	return ""
}

// Target is a tool call's trace target (04, The turn loop): a path relative to the tree for
// file tools, the first 80 characters of a command, the pattern for a search; "" otherwise.
func Target(tool string, input json.RawMessage, tree string) string {
	var in map[string]json.RawMessage
	if json.Unmarshal(input, &in) != nil {
		return ""
	}
	str := func(key string) string {
		var s string
		if raw, ok := in[key]; ok && json.Unmarshal(raw, &s) == nil {
			return s
		}
		return ""
	}
	switch tool {
	case "Bash", "PowerShell":
		return firstRunes(str("command"), summaryRunes)
	case "Grep", "Glob":
		if p := str("pattern"); p != "" {
			return p
		}
	}
	for _, key := range []string{"file_path", "notebook_path", "path"} {
		if p := str(key); p != "" {
			return relativeToTree(p, tree)
		}
	}
	return ""
}

// relativeToTree shows a path under the tree relative to it, slash-separated; any other path
// as written.
func relativeToTree(p, tree string) string {
	sp := filepath.ToSlash(p)
	t := strings.TrimSuffix(filepath.ToSlash(tree), "/")
	if t != "" && len(sp) > len(t) && sp[len(t)] == '/' && strings.EqualFold(sp[:len(t)], t) {
		return sp[len(t)+1:]
	}
	return sp
}

// TraceLine is 04's trace line for a call: `[<tool>] <target>`.
func TraceLine(tool, target string) string {
	if target == "" {
		return "[" + tool + "]"
	}
	return "[" + tool + "] " + target
}

// The outcomes 04 appends to a trace line (The turn loop; Permission prompts).
const (
	traceHookFailed = "blocked: the permission check failed on the host; the agent continues without it."
	traceRanWithout = "ran without Broodforge's check: "
	traceRefusedBy  = "refused by the engine without Broodforge's check: "
	causeTimedOut   = "the permission check timed out on the host."
	causeFailed     = "the permission check failed on the host."
	reachDenyMarker = " is outside this channel's reach ("
)

// NoAnswerCause is the cause a trace line gives when the hook gave the engine no answer and
// the engine's own default decided (decided by config, C39). Which of 04's two phrasings the
// owner wants is pending: this one function holds the choice for every place the line is shown
// (session tail, the Discord posts, the transcript), so one edit flips them all; return
// causeFailed for the other.
func NoAnswerCause() string { return causeTimedOut }

// BypassLine is the trace line for a call the engine decided without Broodforge's check:
// outcome, then cause (04, Permission prompts). A hook exit the engine ignored (decided by
// unknown) is always "failed".
func BypassLine(trace string, ran bool, decidedBy string) string {
	cause := causeFailed
	if decidedBy == DecidedByConfig {
		cause = NoAnswerCause()
	}
	if ran {
		return trace + " -> " + traceRanWithout + cause
	}
	return trace + " -> " + traceRefusedBy + cause
}

// ResultLine is a tool result's trace line: the call's line, with its outcome appended only
// when it is not a plain success (04: a result is appended only when it is an error; a
// bypassed check is always shown).
func ResultLine(tool, target, reach string, isError bool, text string, c Classified) string {
	line := TraceLine(tool, target)
	switch c.Kind {
	case ClassPolicyDenied:
		if strings.Contains(c.Reason, reachDenyMarker) {
			return line + " -> denied: outside this channel's reach (" + reach + "). Nobody in this session can allow it."
		}
		return line + " -> denied: " + strings.TrimSpace(strings.TrimPrefix(c.Reason, policy.DenyPrefix))
	case ClassHookFailed:
		return line + " -> " + traceHookFailed
	case ClassBypassed:
		return BypassLine(line, c.Ran, c.DecidedBy)
	case ClassNoDecision, ClassAllowed:
	}
	if isError {
		return line + " -> " + firstRunes(firstLine(stripToolUseError(StripWrapper(text, tool))), summaryRunes)
	}
	return line
}

// stripToolUseError drops the tags the engine puts around a tool's own error
// (`<tool_use_error>...</tool_use_error>`, S2), so the trace shows the sentence.
func stripToolUseError(s string) string {
	s = strings.TrimSpace(s)
	if rest, ok := strings.CutPrefix(s, toolUseErrorPrefix); ok {
		s = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(rest), "</tool_use_error>"))
	}
	return s
}

// FullCommand is a shell call's whole command, which a permission prompt shows after the mask
// (04, Permission prompts: `Command: <command>`); "" for any other tool.
func FullCommand(tool string, input json.RawMessage) string {
	if tool != "Bash" && tool != "PowerShell" {
		return ""
	}
	var in struct {
		Command string `json:"command"`
	}
	if json.Unmarshal(input, &in) != nil {
		return ""
	}
	return in.Command
}
