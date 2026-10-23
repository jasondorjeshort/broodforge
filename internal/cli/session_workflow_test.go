// SPDX-License-Identifier: GPL-3.0-or-later

package cli_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jasondorjeshort/broodforge/internal/cli/clitest"
	"github.com/jasondorjeshort/broodforge/internal/git/gittest"
	"github.com/jasondorjeshort/broodforge/internal/workflow/workflowtest"
)

// The P19 Proof (08-implementation-plan-session-workflow.md) on the built binary: a scratch root
// whose canonical is fixtures/workflow/ committed on main, a bot serving it, sessions against
// the fake engine.

// sessionRoot makes a scratch root on the fixture (edit changes it before the commit), writes
// the workflow key, serves it and registers 1001 with cred-scratch. It returns the root and a
// runner against it.
func sessionRoot(t *testing.T, edit func(dir string), key string) (string, func(...string) clitest.Result) {
	t.Helper()
	_, env := scratchEnv(t)
	root, _ := scratchNew(t, env, "--project", workflowtest.Canonical(t, edit))
	setWorkflow(t, root, key)
	run := func(args ...string) clitest.Result {
		return clitest.Run(t, env, append([]string{"--root", root}, args...)...)
	}
	startServe(t, env, root)
	if r := run("status", "--wait", "10", "--expect", "ok"); r.Code != 0 {
		t.Fatalf("status: %d %q", r.Code, r.Stderr)
	}
	if r := run("register", "1001", "--name", "Ada Example", "--email", "ada@example.com"); r.Code != 0 {
		t.Fatalf("register: %q", r.Stderr)
	}
	return filepath.ToSlash(root), run
}

// receivedOf reads a session's received-<n>.json as raw keys.
func receivedOf(t *testing.T, root, id string) map[string]json.RawMessage {
	t.Helper()
	var keys map[string]json.RawMessage
	if err := json.Unmarshal([]byte(readFile(t, filepath.Join(root, "sessions", id, "fake", "received-1.json"))), &keys); err != nil {
		t.Fatal(err)
	}
	return keys
}

func TestProofSessionWorkflow(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is not on PATH, which the fixture's hooks need: " + err.Error())
	}
	root, run := sessionRoot(t, nil, `{"hooks":["PreToolUse","PostToolUse"]}`)
	commit := headOf(t, filepath.Join(root, "projects", "scratch"))
	copyDir := root + "/workflow/s0001"
	want(t, run("session", "start", "--channel", "100", "--as", "1001", "--roles", "10", "--credential", "cred-scratch",
		"--mode", "policy", "--fixture", "basic"), 0, "s0001\n", "")
	want(t, run("session", "send", "s0001", "--as", "1001", "--wait", "Read hello.txt."), 0, "turn-ended  1  completed\n", "")

	// received-1.json: the file flag and --agents, never the text flag; BROODFORGE_WORKFLOW the copy.
	keys := receivedOf(t, root, "s0001")
	t.Logf("received-1.json args: %s", keys["args"])
	var args []string
	_ = json.Unmarshal(keys["args"], &args)
	if strings.Join(args, " ") != "--append-system-prompt-file "+copyDir+"/instructions.md --agents "+copyDir+"/agents.json" {
		t.Errorf("args %q", args)
	}
	if !strings.Contains(string(keys["hookEnv"]), `"name": "BROODFORGE_WORKFLOW",`+"\n"+`      "value": "`+copyDir+`"`) {
		t.Errorf("hookEnv %s", keys["hookEnv"])
	}

	// session show: the three lines.
	r := run("session", "show", "s0001")
	t.Logf("$ broodforge session show s0001\n%s%s", r.Stdout, r.Stderr)
	for _, w := range []string{"\nworkflow: " + commit + " 20", "\nhooks: git-guard.js, post-edit.js\n", "\nreview: none since " + commit + "\n"} {
		if !strings.Contains(r.Stdout, w) {
			t.Errorf("session show lacks %q", w)
		}
	}
	// --header: the header alone, as instructions.md opens; the project's text follows it there.
	r = run("session", "show", "--header", "s0001")
	t.Logf("$ broodforge session show --header s0001\n%s%s", r.Stdout, r.Stderr)
	instr := readFile(t, copyDir+"/instructions.md")
	if r.Code != 0 || !strings.HasPrefix(r.Stdout, "This session is run by Broodforge. Where the project's instructions below conflict") ||
		!strings.Contains(r.Stdout, "\nThe project's instructions below were taken from main at "+commit+" on 20") ||
		!strings.HasSuffix(r.Stdout, "Z.\n") || !strings.HasPrefix(instr, r.Stdout+"\nContents of CLAUDE.md:\n") {
		t.Errorf("--header: exit %d\n%s\ninstructions.md:\n%s", r.Code, r.Stdout, instr)
	}
	// Without the bot the store and the canonical give the same lines.
	want(t, run("stop"), 0, "stopped  1\n", "")
	if r := run("session", "show", "s0001"); !strings.Contains(r.Stdout, "\nhooks: git-guard.js, post-edit.js\nreview: none since "+commit+"\n") {
		t.Errorf("session show without the bot:\n%s%s", r.Stdout, r.Stderr)
	}
	// A moved canonical is an error on the line, the rest of the record still printed; --json
	// carries git's message.
	canonical := filepath.Join(root, "projects", "scratch")
	if err := os.Rename(canonical, canonical+"-moved"); err != nil {
		t.Fatal(err)
	}
	r = run("session", "show", "s0001")
	t.Logf("$ broodforge session show s0001 (canonical moved)\n%s%s", r.Stdout, r.Stderr)
	if r.Code != 0 || !strings.HasSuffix(r.Stdout, "\nhooks: git-guard.js, post-edit.js\nreview: error: session-ref\n") {
		t.Errorf("moved canonical: exit %d\n%s%s", r.Code, r.Stdout, r.Stderr)
	}
	if r := run("--json", "session", "show", "s0001"); !strings.Contains(r.Stdout, `"review":{"commit":"","toolUseId":null,"error":{"code":"session-ref","message":"git for-each-ref: `) {
		t.Errorf("moved canonical --json: %s", r.Stdout)
	}
}

// TestProofSessionInjectNone: inject none copies nothing: `workflow: none (inject none)`, the
// file flag with the header alone, no --agents, no BROODFORGE_WORKFLOW; a subagent's calls are
// indented under its Agent line in `session tail`.
func TestProofSessionInjectNone(t *testing.T) {
	root, run := sessionRoot(t, nil, `{"inject":"none"}`)
	canonical := filepath.Join(root, "projects", "scratch")
	commit := headOf(t, canonical)
	want(t, run("session", "start", "--channel", "100", "--as", "1001", "--roles", "10", "--credential", "cred-scratch",
		"--mode", "policy", "--fixture", "subagent"), 0, "s0001\n", "")
	if r := run("session", "show", "s0001"); !strings.HasSuffix(r.Stdout, "\nreview: none since "+commit+"\n") {
		t.Errorf("before the review:\n%s", r.Stdout)
	}
	// The review fact counts a row strictly after the commit's committer second: wait past it.
	ct, err := strconv.ParseInt(strings.TrimSpace(gittest.Git(t, canonical, "show", "-s", "--format=%ct", commit)), 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Until(time.Unix(ct+1, 0).Add(100 * time.Millisecond)))
	want(t, run("session", "send", "s0001", "--as", "1001", "--wait", "Review it."), 0, "turn-ended  1  completed\n", "")
	keys := receivedOf(t, root, "s0001")
	var args []string
	_ = json.Unmarshal(keys["args"], &args)
	if strings.Join(args, " ") != "--append-system-prompt-file "+root+"/workflow/s0001/instructions.md" ||
		strings.Contains(string(keys["hookEnv"]), "BROODFORGE_WORKFLOW") {
		t.Errorf("args %q hookEnv %s", args, keys["hookEnv"])
	}
	instr := readFile(t, root+"/workflow/s0001/instructions.md")
	if r := run("session", "show", "--header", "s0001"); r.Stdout != instr || strings.Contains(instr, "taken from") ||
		strings.Contains(instr, "Contents of") {
		t.Errorf("header alone: %q\ninstructions.md %q", r.Stdout, instr)
	}
	// The reviewer's Agent call came after the session's last commit (here the base).
	r := run("session", "show", "s0001")
	t.Logf("$ broodforge session show s0001\n%s", r.Stdout)
	if !strings.HasSuffix(r.Stdout, "\nworkflow: none (inject none)\nhooks: -\nreview: toolu_sub_agent after "+commit+"\n") {
		t.Errorf("session show:\n%s", r.Stdout)
	}
	tail := run("session", "tail", "s0001").Stdout
	t.Logf("$ broodforge session tail s0001\n%s", tail)
	for _, w := range []string{"  tool-call  [Agent]\n", "  tool-call    [Read] hello.txt\n",
		"  permission-decision    allow [Read] hello.txt (policy)\n", "  text    No findings.\n", "  tool-result  [Agent]\n"} {
		if !strings.Contains(tail, w) {
			t.Errorf("session tail lacks %q", w)
		}
	}
}

// TestProofSessionProbeRefused: a listed hook whose runtime is missing refuses the start with
// workflow-probe-failed (through Discord, an ephemeral reply to the starter and no post), the
// refusal in the audit, and nothing of the session left but its ended row.
func TestProofSessionProbeRefused(t *testing.T) {
	edit := func(dir string) {
		p := filepath.Join(dir, ".claude", "settings.json")
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		s := strings.Replace(string(b), `node \"${CLAUDE_PROJECT_DIR:-.}/.claude/hooks/git-guard.js\"`, "nosuchruntime x.js", 1)
		if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	root, run := sessionRoot(t, edit, `{"hooks":["PreToolUse"]}`)
	r := run("session", "start", "--channel", "100", "--as", "1001", "--roles", "10", "--credential", "cred-scratch", "--fixture", "basic")
	t.Logf("$ broodforge session start ...\n%s%s", r.Stdout, r.Stderr)
	want(t, r, 1, "", "error: workflow-probe-failed: nosuchruntime x.js needs nosuchruntime\n")
	// Through Discord the starter alone is told, ephemerally (the owner, 2026-10-23); nothing is
	// posted to the channel or anywhere else.
	run("register", "set-default", "1001", "cred-scratch")
	r = run("discord", "inject", "click", "--custom-id", "bf:100:start", "--user", "1001", "--roles", "10")
	t.Logf("$ broodforge discord inject click --custom-id bf:100:start ...\n%s%s", r.Stdout, r.Stderr)
	want(t, r, 1, "", "error: workflow-probe-failed: Could not start a session: nosuchruntime x.js needs nosuchruntime.\n")
	time.Sleep(1500 * time.Millisecond) // past the coalescer's pacing, so a post would be out
	want(t, run("session", "outbox", "-", "--channel", "100"), 0, "", "")
	r = run("--json", "audit", "--kind", "action-refused", "--session", "s0001")
	t.Logf("$ broodforge audit --kind action-refused --session s0001\n%s", r.Stdout)
	if !strings.Contains(r.Stdout, `"action":"start","message":"nosuchruntime x.js needs nosuchruntime","reason":"workflow"`) {
		t.Errorf("audit: %s%s", r.Stdout, r.Stderr)
	}
	if r := run("session", "show", "s0001"); !strings.HasPrefix(r.Stdout, "session: s0001  ended  ") || !strings.Contains(r.Stdout, "\nworktree: -\n") {
		t.Errorf("session show: %s%s", r.Stdout, r.Stderr)
	}
	for _, p := range []string{root + "/workflow/s0001", root + "/sessions/s0001"} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("%s survived the refusal: %v", p, err)
		}
	}
}
