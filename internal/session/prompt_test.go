// SPDX-License-Identifier: GPL-3.0-or-later

package session

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestTarget(t *testing.T) {
	tree := "/r/sessions/s0001/tree"
	long := strings.Repeat("x", 100)
	for _, c := range []struct {
		tool, input, want string
	}{
		{"Read", `{"file_path":"/r/sessions/s0001/tree/src/a.go"}`, "src/a.go"},
		{"Edit", `{"file_path":"/elsewhere/a.go"}`, "/elsewhere/a.go"},
		{"Grep", `{"pattern":"schemaVersion","path":"src/"}`, "schemaVersion"},
		{"Glob", `{"pattern":"**/*.go"}`, "**/*.go"},
		{"Bash", `{"command":"npm test"}`, "npm test"},
		{"Bash", `{"command":"` + long + `"}`, strings.Repeat("x", 79) + "…"},
		{"NotebookEdit", `{"notebook_path":"/r/sessions/s0001/tree/n.ipynb"}`, "n.ipynb"},
		{"TodoWrite", `{"todos":[]}`, ""},
		{"Read", `not json`, ""},
	} {
		if got := Target(c.tool, json.RawMessage(c.input), tree); got != c.want {
			t.Errorf("Target(%s, %s) = %q, want %q", c.tool, c.input, got, c.want)
		}
	}
}

// TestResultLine checks the trace lines against 04's wording (The turn loop; Permission prompts).
func TestResultLine(t *testing.T) {
	reachDeny := Classified{Kind: ClassPolicyDenied, Decision: DecisionDeny, DecidedBy: DecidedByPolicy,
		Reason: "Denied by Broodforge: Edit on ../utopia/config.ini is outside this channel's reach (code); nobody in this session can allow it; the channel's configuration sets reach"}
	for _, c := range []struct {
		name, tool, target string
		isError            bool
		text               string
		c                  Classified
		want               string
	}{
		{"allowed", "Read", "src/config/load.ts", false, "", Classified{Kind: ClassAllowed}, "[Read] src/config/load.ts"},
		{"reach deny", "Edit", "../utopia/config.ini", true, "", reachDeny,
			"[Edit] ../utopia/config.ini -> denied: outside this channel's reach (code). Nobody in this session can allow it."},
		{"hook failed", "Bash", "npm test", true, "", Classified{Kind: ClassHookFailed},
			"[Bash] npm test -> blocked: the permission check failed on the host; the agent continues without it."},
		{"bypassed ran", "Bash", "echo ok", false, "", Classified{Kind: ClassBypassed, DecidedBy: DecidedByConfig, Ran: true},
			"[Bash] echo ok -> ran without Broodforge's check: the permission check timed out on the host."},
		{"bypassed refused", "Write", "note.txt", true, "", Classified{Kind: ClassBypassed, DecidedBy: DecidedByUnknown},
			"[Write] note.txt -> refused by the engine without Broodforge's check: the permission check failed on the host."},
		{"tool error", "Read", "a.txt", true, "File does not exist.\nmore", Classified{Kind: ClassAllowed},
			"[Read] a.txt -> File does not exist."},
		{"tool's own error", "Bash", "sleep 30", true, "<tool_use_error>Blocked: standalone sleep 30.</tool_use_error>",
			Classified{Kind: ClassNoDecision}, "[Bash] sleep 30 -> Blocked: standalone sleep 30."},
	} {
		if got := ResultLine(c.tool, c.target, "code", c.isError, c.text, c.c); got != c.want {
			t.Errorf("%s: got %q\nwant %q", c.name, got, c.want)
		}
	}
}

func TestBlocksAndPrompt(t *testing.T) {
	in := JoinBlocks([]string{FromBlock("Ada Example", RoleDriver, "fix it"), FromBlock("Sam", RoleContributor, "and test")})
	want := "From: Ada Example (driver)\nfix it\n\nFrom: Sam (contributor)\nand test"
	if in != want {
		t.Errorf("JoinBlocks = %q, want %q", in, want)
	}
}

// TestHeader: the header of ADR 0025 rule 1, whole: the owner's precedence sentence first,
// verbatim, then the session's facts, one call at a time, rule 10's trailer rule with no list,
// and the taken-from line; no blank line inside it (ReadHeader cuts at the first).
func TestHeader(t *testing.T) {
	date := time.Date(2026, 10, 23, 9, 30, 0, 0, time.UTC)
	got := SystemPromptAppend(Header{Session: "s0001", ProjectName: "Scratch", Reach: "code", DriverName: "Ada Example",
		Copied: true, Branch: "main", Commit: "0123456789abcdef0123456789abcdef01234567", Date: date})
	want := "This session is run by Broodforge. Where the project's instructions below conflict with this header, this header wins; it never says how the project's work is done.\n" +
		"You are working in Broodforge session s0001 on Scratch.\n" +
		"Reach: code (change only this session's tree).\n" +
		"Each message starts with a From: line naming its writer and their role; the driver is Ada Example.\n" +
		"Make one tool call at a time: calls made together are asked about one by one.\n" +
		"Every commit carries the Co-Authored-By lines Broodforge's commit-msg hook adds; never remove one.\n" +
		"The project's instructions below were taken from main at 0123456789abcdef0123456789abcdef01234567 on 2026-10-23T09:30:00Z.\n"
	if got != want {
		t.Errorf("header\n got %q\nwant %q", got, want)
	}
	// Nothing copied: no taken-from line; no list of trailers either way.
	alone := SystemPromptAppend(Header{Session: "s0002", ProjectName: "Scratch", Reach: "read-only", DriverName: "Ada"})
	if strings.Contains(alone, "taken from") || !strings.HasSuffix(alone, "never remove one.\n") ||
		!strings.Contains(alone, "Reach: read-only (read the project, write nothing).\n") {
		t.Errorf("header alone:\n%s", alone)
	}
	for _, h := range []string{got, alone} {
		if strings.Contains(h, "\n\n") || strings.Contains(h, "Co-Authored-By: ") {
			t.Errorf("a blank line or a trailer line in the header:\n%s", h)
		}
	}
}
