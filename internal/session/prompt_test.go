// SPDX-License-Identifier: GPL-3.0-or-later

package session

import (
	"encoding/json"
	"strings"
	"testing"
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
	p := SystemPromptAppend("s0001", "Scratch", "code", "Ada Example", []string{"Co-Authored-By: Sam <sam@example.com>"})
	for _, w := range []string{"session s0001 on Scratch", "Reach: code", "one tool call at a time",
		"\nCo-Authored-By: Sam <sam@example.com>\n"} {
		if !strings.Contains(p, w) {
			t.Errorf("system prompt append lacks %q:\n%s", w, p)
		}
	}
	if p := SystemPromptAppend("s0001", "Scratch", "read-only", "Ada", nil); !strings.Contains(p, "no Co-Authored-By lines") {
		t.Errorf("no trailers: %s", p)
	}
}
