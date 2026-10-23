// SPDX-License-Identifier: GPL-3.0-or-later

package cli_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jasondorjeshort/broodforge/internal/cli/clitest"
	"github.com/jasondorjeshort/broodforge/internal/git/gittest"
	"github.com/jasondorjeshort/broodforge/internal/workflow"
	"github.com/jasondorjeshort/broodforge/internal/workflow/workflowtest"
)

// The P17 Proof (08-implementation-plan-session-workflow.md), on the built binary against a
// scratch root whose canonical is fixtures/workflow/ committed on main.

// setWorkflow writes the workflow key into the scratch root's project file; raw "" removes it.
func setWorkflow(t *testing.T, root, raw string) {
	t.Helper()
	p := filepath.Join(root, "config", "projects", "scratch.json")
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	delete(m, "workflow")
	if raw != "" {
		m["workflow"] = json.RawMessage(raw)
	}
	if b, err = json.Marshal(m); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, b, 0o600); err != nil {
		t.Fatal(err)
	}
}

func headOf(t *testing.T, dir string) string {
	t.Helper()
	return strings.TrimSpace(gittest.Git(t, dir, "rev-parse", "HEAD"))
}

func TestProofWorkflowRender(t *testing.T) {
	_, env := scratchEnv(t)
	root, _ := scratchNew(t, env, "--project", workflowtest.Canonical(t, nil))
	canonical := filepath.Join(root, "projects", "scratch")
	commit := headOf(t, canonical)
	setWorkflow(t, root, `{"hooks":["PreToolUse","PostToolUse"]}`)
	out := filepath.ToSlash(filepath.Join(t.TempDir(), "copy"))

	r := clitest.Run(t, env, "--root", root, "workflow", "render", "--project", "scratch", "--out", out)
	t.Logf("$ broodforge workflow render --project scratch --out <tmp>/copy\n%s%s", r.Stdout, r.Stderr)
	lines := strings.Split(r.Stdout, "\n")
	if r.Code != 0 || r.Stderr != "" || len(lines) != 5 || !strings.HasPrefix(lines[0], "workflow: "+commit+" 20") ||
		strings.Join(lines[1:], "\n") != "wrote  "+out+"/instructions.md\nwrote  "+out+"/agents.json\nwrote  "+out+"/hooks.json\n" {
		t.Fatalf("render: exit %d\n%s%s", r.Code, r.Stdout, r.Stderr)
	}
	instr := readFile(t, filepath.Join(out, "instructions.md"))
	t.Logf("instructions.md:\n%s", instr)
	for _, s := range []string{"Write short sentences.", "note: @.claude/instructions/missing.md in CLAUDE.md:8 was not loaded",
		"Contents of .claude/rules/a-style.md:\n"} {
		if !strings.Contains(instr, s) {
			t.Errorf("instructions.md lacks %q", s)
		}
	}
	if a, b, c := strings.Index(instr, "a-style.md:"), strings.Index(instr, "b-tests.md:"), strings.Index(instr, "c-docs.md:"); !(a < b && b < c) {
		t.Errorf("rules out of name order:\n%s", instr)
	}
	var agents map[string]json.RawMessage
	if err := json.Unmarshal([]byte(readFile(t, filepath.Join(out, "agents.json"))), &agents); err != nil || len(agents) != 2 ||
		agents["builder"] == nil || agents["checker"] == nil {
		t.Errorf("agents.json: %v %v", agents, err)
	}
	var hooks []workflow.Hook
	if err := json.Unmarshal([]byte(readFile(t, filepath.Join(out, "hooks.json"))), &hooks); err != nil || len(hooks) != 2 ||
		hooks[0].Name != "git-guard.js" || hooks[1].Name != "post-edit.js" {
		t.Errorf("hooks.json: %+v %v", hooks, err)
	}

	// An out that holds the last render is refused, so nothing stale survives into the next.
	want(t, clitest.Run(t, env, "--root", root, "workflow", "render", "--project", "scratch", "--out", out), 1, "",
		"error: workflow-out: "+out+" is not empty\n")

	// "hooks": [] writes no hooks.json.
	setWorkflow(t, root, `{"hooks":[]}`)
	out2 := filepath.ToSlash(filepath.Join(t.TempDir(), "copy"))
	r = clitest.Run(t, env, "--root", root, "workflow", "render", "--project", "scratch", "--out", out2)
	if r.Code != 0 || strings.Contains(r.Stdout, "hooks.json") {
		t.Errorf("hooks []: exit %d\n%s%s", r.Code, r.Stdout, r.Stderr)
	}
	if _, err := os.Stat(filepath.Join(out2, "hooks.json")); !os.IsNotExist(err) {
		t.Errorf("hooks.json written with hooks []: %v", err)
	}

	// --json: one object per line.
	r = clitest.Run(t, env, "--root", root, "--json", "workflow", "render", "--project", "scratch", "--out", filepath.Join(t.TempDir(), "j"))
	if r.Code != 0 || !strings.HasPrefix(r.Stdout, `{"workflow":"`+commit+` 20`) || !strings.Contains(r.Stdout, `"branch":"main","commit":"`+commit+`"`) ||
		!strings.Contains(r.Stdout, `{"action":"wrote","file":"`) {
		t.Errorf("--json: %s%s", r.Stdout, r.Stderr)
	}

	// config check refuses an event other than the two, naming it.
	setWorkflow(t, root, `{"hooks":["PreToolUse","Stop"]}`)
	want(t, clitest.Run(t, env, "config", "check", "--root", root), 1, "",
		"error: bad-config: "+filepath.ToSlash(root)+"/config/projects/scratch.json: workflow.hooks.1: \"Stop\", want PreToolUse or PostToolUse\n")

	// inject none copies nothing.
	setWorkflow(t, root, `{"inject":"none"}`)
	out3 := filepath.ToSlash(filepath.Join(t.TempDir(), "copy"))
	want(t, clitest.Run(t, env, "--root", root, "workflow", "render", "--project", "scratch", "--out", out3), 0,
		"workflow: none (inject none)\nwrote  "+out3+"/instructions.md\n", "")

	want(t, clitest.Run(t, env, "--root", root, "workflow", "render", "--project", "nope", "--out", out3), 1, "",
		"error: unknown-project: no project nope in "+filepath.ToSlash(root)+"/config\n")
	r = clitest.Run(t, env, "--root", root, "workflow", "render", "--project", "scratch")
	if r.Code != 2 || r.Stderr != "error: usage: workflow render: --out is required\n" {
		t.Errorf("no --out: exit %d stderr %q", r.Code, r.Stderr)
	}
}

// A canonical with neither CLAUDE.md nor .claude prints the none line and writes
// instructions.md alone.
func TestProofWorkflowRenderNothingListed(t *testing.T) {
	_, env := scratchEnv(t)
	root, _ := scratchNew(t, env) // the generated canonical: one README on main
	commit := headOf(t, filepath.Join(root, "projects", "scratch"))
	setWorkflow(t, root, `{"hooks":["PreToolUse"]}`)
	out := filepath.ToSlash(filepath.Join(t.TempDir(), "copy"))
	want(t, clitest.Run(t, env, "--root", root, "workflow", "render", "--project", "scratch", "--out", out), 0,
		"workflow: none (main at "+commit+" has no CLAUDE.md or .claude)\nwrote  "+out+"/instructions.md\n", "")
	entries, err := os.ReadDir(out)
	if err != nil || len(entries) != 1 {
		t.Errorf("out holds %v %v", entries, err)
	}
}

func TestProofWorkflowProbe(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is not on PATH: " + err.Error())
	}
	_, env := scratchEnv(t)
	root, _ := scratchNew(t, env, "--project", workflowtest.Canonical(t, nil))
	setWorkflow(t, root, `{"hooks":["PreToolUse","PostToolUse"]}`)
	r := clitest.Run(t, env, "--root", root, "workflow", "probe", "--project", "scratch")
	t.Logf("$ broodforge workflow probe --project scratch\n%s%s", r.Stdout, r.Stderr)
	want(t, r, 0, "ok  git-guard.js  node\nok  post-edit.js  node\n", "")
	want(t, clitest.Run(t, env, "config", "check", "--root", root), 0, "ok  1  2\n", "")
	r = clitest.Run(t, env, "--root", root, "--json", "workflow", "probe", "--project", "scratch")
	if r.Code != 0 || !strings.HasPrefix(r.Stdout, `{"event":"PreToolUse","matcher":"Bash|PowerShell","command":"node \"${CLAUDE_PROJECT_DIR:-.}/.claude/hooks/git-guard.js\"","name":"git-guard.js","timeout":5,"runtime":"node"}`+"\n") {
		t.Errorf("--json: %s%s", r.Stdout, r.Stderr)
	}

	// A command whose runtime is missing: the not-started line, the error, exit 1.
	canonical := filepath.Join(root, "projects", "scratch")
	settings := readFile(t, filepath.Join(canonical, ".claude", "settings.json"))
	gittest.Commit(t, gittest.Driver, nil, canonical, ".claude/settings.json",
		strings.Replace(settings, `node \"${CLAUDE_PROJECT_DIR:-.}/.claude/hooks/post-edit.js\"`, "nosuchruntime x.js", 1), "A hook nobody can run")
	r = clitest.Run(t, env, "--root", root, "workflow", "probe", "--project", "scratch")
	t.Logf("$ broodforge workflow probe --project scratch\n%s%s", r.Stdout, r.Stderr)
	want(t, r, 1, "ok  git-guard.js  node\nNot started: the project's hook nosuchruntime x.js needs nosuchruntime, which is not "+
		"installed where sessions run. The owner installs it, or removes PostToolUse from workflow.hooks in config/projects/scratch.json.\n",
		"error: workflow-probe-failed: nosuchruntime x.js needs nosuchruntime\n")
	// config check probes the same way (ADR 0025 rule 6), naming the command and the word; with
	// the event no longer listed it passes.
	want(t, clitest.Run(t, env, "config", "check", "--root", root), 1, "",
		"error: bad-config: "+filepath.ToSlash(root)+"/config/projects/scratch.json: workflow.hooks: the PostToolUse hook "+
			"nosuchruntime x.js needs nosuchruntime, which is not installed here\n")
	setWorkflow(t, root, `{"hooks":["PreToolUse"]}`)
	want(t, clitest.Run(t, env, "config", "check", "--root", root), 0, "ok  1  2\n", "")
	// The probe leaves no copy behind in the temporary directory.
	for _, kv := range env {
		if dir, ok := strings.CutPrefix(kv, "TMP="); ok {
			if m, _ := filepath.Glob(filepath.Join(dir, "broodforge-workflow-*")); len(m) != 0 {
				t.Errorf("probe left %v", m)
			}
		}
	}
}

func readFile(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return strings.ReplaceAll(string(b), "\r\n", "\n")
}
