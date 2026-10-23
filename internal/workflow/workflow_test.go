// SPDX-License-Identifier: GPL-3.0-or-later

package workflow_test

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jasondorjeshort/broodforge/internal/workflow"
	"github.com/jasondorjeshort/broodforge/internal/workflow/workflowtest"
)

var both = []string{"PreToolUse", "PostToolUse"}

func render(t *testing.T, canonical string, o workflow.Options) (workflow.Copy, string) {
	t.Helper()
	out := filepath.Join(t.TempDir(), "copy")
	o.Canonical, o.Branch = canonical, "main"
	c, err := workflow.Render(t.Context(), o, out)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	return c, out
}

func read(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return strings.ReplaceAll(string(b), "\r\n", "\n")
}

// TestRenderFixture: the copy of fixtures/workflow/ holds the listed files, and the three
// renders are the ADR's: the import's text and a note for the missing one, the rules in name
// order; two agents under their bare names; the two command hooks named by their scripts, the
// Stop entry absent.
func TestRenderFixture(t *testing.T) {
	c, out := render(t, workflowtest.Canonical(t, nil), workflow.Options{Hooks: both})
	if len(c.Commit) != 40 || c.Date.IsZero() || c.None() || !strings.HasPrefix(c.Line(), c.Commit+" 20") {
		t.Errorf("copy %+v line %q", c, c.Line())
	}
	if strings.Join(c.Listed, " ") != ".claude CLAUDE.md" && strings.Join(c.Listed, " ") != "CLAUDE.md .claude" {
		t.Errorf("listed %q", c.Listed)
	}
	if _, err := os.Stat(filepath.Join(out, "README.md")); !os.IsNotExist(err) {
		t.Errorf("an unlisted file was copied: %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, ".claude", "hooks", "git-guard.js")); err != nil {
		t.Errorf("the hook script was not copied: %v", err)
	}

	instr := read(t, filepath.Join(out, workflow.InstructionsFile))
	order := []string{"Contents of CLAUDE.md:", "House style: @.claude/instructions/house-style.md",
		"note: @.claude/instructions/missing.md in CLAUDE.md:8 was not loaded: .claude/instructions/missing.md is not in the copy",
		"Contents of .claude/instructions/house-style.md:", "Write short sentences.",
		"Contents of .claude/rules/a-style.md:", "Contents of .claude/rules/b-tests.md:", "paths:",
		"Contents of .claude/rules/c-docs.md:"}
	at := 0
	for _, s := range order {
		i := strings.Index(instr[at:], s)
		if i < 0 {
			t.Fatalf("instructions.md lacks %q after offset %d:\n%s", s, at, instr)
		}
		at += i + len(s)
	}
	if strings.Count(instr, "note:") != 1 || strings.Count(instr, "Contents of") != 5 {
		t.Errorf("instructions.md:\n%s", instr)
	}

	var agents map[string]workflow.Agent
	if err := json.Unmarshal([]byte(read(t, filepath.Join(out, workflow.AgentsFile))), &agents); err != nil {
		t.Fatal(err)
	}
	b, ch := agents["builder"], agents["checker"]
	if len(agents) != 2 || b.Model != "opus" || b.Effort != "high" || strings.Join(b.Tools, ",") != "Read,Glob,Grep,Write,Edit,Bash" ||
		!strings.HasPrefix(b.Description, "Builds what the brief asks for: code") || !strings.HasPrefix(b.Prompt, "You build") {
		t.Errorf("builder %+v", b)
	}
	if ch.Description != "Reads a change it did not write and reports what is wrong; never fixes it." ||
		strings.Join(ch.Tools, ",") != "Read,Grep" || ch.Model != "sonnet" || ch.Effort != "" {
		t.Errorf("checker %+v", ch)
	}

	hooks, err := workflow.ReadHooks(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(hooks) != 2 || hooks[0].Name != "git-guard.js" || hooks[0].Event != "PreToolUse" || hooks[0].Matcher != "Bash|PowerShell" ||
		hooks[0].Timeout == nil || *hooks[0].Timeout != 5 || hooks[1].Name != "post-edit.js" || hooks[1].Event != "PostToolUse" ||
		hooks[1].Timeout != nil || !strings.Contains(hooks[1].Command, "${CLAUDE_PROJECT_DIR:-.}") {
		t.Errorf("hooks %+v", hooks)
	}
	raw := read(t, filepath.Join(out, workflow.HooksFile))
	for _, k := range []string{`"event"`, `"matcher"`, `"command"`, `"name"`, `"timeout": null`} {
		if !strings.Contains(raw, k) {
			t.Errorf("hooks.json lacks %s:\n%s", k, raw)
		}
	}
	if len(c.Files) != 3 {
		t.Errorf("files %q", c.Files)
	}
}

// TestRenderNoHooksListed: with "hooks": [] (the default) no hooks.json is written.
func TestRenderNoHooksListed(t *testing.T) {
	c, out := render(t, workflowtest.Canonical(t, nil), workflow.Options{})
	if _, err := os.Stat(filepath.Join(out, workflow.HooksFile)); !os.IsNotExist(err) || len(c.Hooks) != 0 {
		t.Errorf("hooks.json written with no event listed: %v", err)
	}
	if len(c.Files) != 2 {
		t.Errorf("files %q", c.Files)
	}
}

// TestRenderNothingListed: a default branch with neither CLAUDE.md nor .claude is the
// header-alone case: no archive (git archive would exit 128), instructions.md alone and empty.
func TestRenderNothingListed(t *testing.T) {
	canonical := workflowtest.Canonical(t, func(dir string) {
		for _, p := range []string{"CLAUDE.md", ".claude"} {
			if err := os.RemoveAll(filepath.Join(dir, p)); err != nil {
				t.Fatal(err)
			}
		}
	})
	c, out := render(t, canonical, workflow.Options{Hooks: both})
	if !c.None() || c.Line() != "none (main at "+c.Commit+" has no CLAUDE.md or .claude)" {
		t.Errorf("line %q", c.Line())
	}
	entries, _ := os.ReadDir(out)
	if len(entries) != 1 || entries[0].Name() != workflow.InstructionsFile || read(t, filepath.Join(out, workflow.InstructionsFile)) != "" {
		t.Errorf("entries %v", entries)
	}
}

// TestRenderInjectNone: inject none reads nothing and writes an empty instructions.md.
func TestRenderInjectNone(t *testing.T) {
	c, out := render(t, "no-such-canonical", workflow.Options{Inject: workflow.InjectNone, Hooks: both})
	if c.Line() != "none (inject none)" || len(c.Files) != 1 || read(t, filepath.Join(out, workflow.InstructionsFile)) != "" {
		t.Errorf("copy %+v", c)
	}
}

// TestRenderRefusals: an unreadable canonical, a missing branch, a non-empty out and a settings
// file that does not parse while hooks are listed each refuse with their code.
func TestRenderRefusals(t *testing.T) {
	code := func(err error) string {
		var we *workflow.Error
		if errors.As(err, &we) {
			return we.Code
		}
		return "?" + err.Error()
	}
	ctx := t.Context()
	if _, err := workflow.Render(ctx, workflow.Options{Canonical: t.TempDir(), Branch: "main"}, t.TempDir()+"/a"); code(err) != workflow.CodeCanonical {
		t.Errorf("not a repository: %v", err)
	}
	canonical := workflowtest.Canonical(t, nil)
	if _, err := workflow.Render(ctx, workflow.Options{Canonical: canonical, Branch: "trunk"}, t.TempDir()+"/a"); code(err) != workflow.CodeCanonical ||
		!strings.Contains(err.Error(), "has no branch trunk") {
		t.Errorf("no branch: %v", err)
	}
	full := t.TempDir()
	workflowtest.Write(t, full, "stale.txt", "x")
	if _, err := workflow.Render(ctx, workflow.Options{Canonical: canonical, Branch: "main"}, full); code(err) != workflow.CodeOut {
		t.Errorf("non-empty out: %v", err)
	}
	broken := workflowtest.Canonical(t, func(dir string) { workflowtest.Write(t, dir, ".claude/settings.json", "{") })
	if _, err := workflow.Render(ctx, workflow.Options{Canonical: broken, Branch: "main", Hooks: both}, t.TempDir()+"/a"); code(err) != workflow.CodeSettings {
		t.Errorf("broken settings: %v", err)
	}
	if _, err := workflow.Render(ctx, workflow.Options{Canonical: broken, Branch: "main"}, t.TempDir()+"/b"); err != nil {
		t.Errorf("broken settings with no hooks listed: %v", err)
	}
}

// TestHookNamingAndSkips: names come from the first word naming a file, else the first word; a
// repeat is suffixed -2, -3; an agent without a description is skipped with why; a hook of a
// listed event that is not a command, or has no command, refuses the render naming the event
// and the matcher (ADR 0025 rule 6: never a skip), and is ignored when its event is not listed.
func TestHookNamingAndSkips(t *testing.T) {
	const good = `{"hooks":{"PreToolUse":[
 {"matcher":"Bash","hooks":[
  {"type":"command","command":"node \"$CLAUDE_PROJECT_DIR/.claude/hooks/git-guard.js\""},
  {"type":"command","command":"node '.claude/hooks/git-guard.js' --strict"},
  {"type":"command","command":"python3 -m tool"},
  {"type":"command","command":".claude/hooks/check.sh"},
  {"type":"command","command":"python3 other"}]}],
 "PostToolUse":[{"hooks":[{"type":"command","command":"C:/tools/lint.exe --fix","timeout":2.5}]}]}}`
	for _, bad := range []struct{ entry, why string }{
		{`{"type":"prompt","prompt":"Is this safe?"}`, `is a "prompt" hook; only a command hook runs as a child`},
		{`{"type":"command","command":"  "}`, `has no command`},
	} {
		canonical := workflowtest.Canonical(t, func(dir string) {
			workflowtest.Write(t, dir, ".claude/settings.json", strings.Replace(good, `"python3 other"}`, `"python3 other"},`+bad.entry, 1))
		})
		_, err := workflow.Render(t.Context(), workflow.Options{Canonical: canonical, Branch: "main", Hooks: both}, filepath.Join(t.TempDir(), "c"))
		want := `workflow-settings: .claude/settings.json: hooks.PreToolUse.0.hooks.5 (PreToolUse, matcher "Bash") ` + bad.why
		if err == nil || err.Error() != want {
			t.Errorf("got %v\nwant %s", err, want)
		}
		if _, err := workflow.Render(t.Context(), workflow.Options{Canonical: canonical, Branch: "main", Hooks: []string{"PostToolUse"}},
			filepath.Join(t.TempDir(), "c")); err != nil {
			t.Errorf("an unlisted event's hook refused the render: %v", err)
		}
	}
	canonical := workflowtest.Canonical(t, func(dir string) {
		workflowtest.Write(t, dir, ".claude/settings.json", good)
		workflowtest.Write(t, dir, ".claude/agents/broken.md", "---\nname: broken\n---\nNo description.\n")
	})
	c, _ := render(t, canonical, workflow.Options{Hooks: both})
	var names []string
	for _, h := range c.Hooks {
		names = append(names, h.Name)
	}
	if got := strings.Join(names, " "); got != "git-guard.js git-guard.js-2 python3 check.sh python3-2 lint.exe" {
		t.Errorf("names %q", got)
	}
	if last := c.Hooks[len(c.Hooks)-1]; last.Timeout == nil || *last.Timeout != 3 || last.Matcher != "" {
		t.Errorf("last %+v", last)
	}
	var skips []string
	for _, s := range c.Skips {
		skips = append(skips, s.File+": "+s.Why)
	}
	want := ".claude/agents/broken.md: the frontmatter needs a name and a description"
	if got := strings.Join(skips, "; "); got != want {
		t.Errorf("skips %q", got)
	}
}

// TestProbe: a runtime of the list runs --version; a script word needs its file and its
// runtime; anything else must be on PATH; the first failure names the command and the word,
// and the not-started text is the companion's.
func TestProbe(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is not on PATH: " + err.Error())
	}
	dir := t.TempDir()
	workflowtest.Write(t, dir, ".claude/hooks/a.js", "")
	hooks := []workflow.Hook{
		{Event: "PreToolUse", Command: `node "${CLAUDE_PROJECT_DIR:-.}/.claude/hooks/a.js"`, Name: "a.js"},
		{Event: "PreToolUse", Command: `.claude/hooks/a.js`, Name: "a.js-2"},
		{Event: "PreToolUse", Command: `git status`, Name: "git"},
	}
	got, err := workflow.Probe(t.Context(), dir, hooks)
	if err != nil || len(got) != 3 || got[0].Runtime != "node" || got[1].Runtime != "node" || got[2].Runtime != "git" {
		t.Fatalf("probe %+v %v", got, err)
	}
	for _, c := range []struct{ command, word string }{
		{"nosuchruntime x.js", "nosuchruntime"},
		{".claude/hooks/gone.js", ".claude/hooks/gone.js"},
		{`"$CLAUDE_PROJECT_DIR/none.py"`, "$CLAUDE_PROJECT_DIR/none.py"},
	} {
		h := workflow.Hook{Event: "PostToolUse", Command: c.command, Name: "x"}
		got, err := workflow.Probe(t.Context(), dir, append(hooks[:1:1], h))
		var pf *workflow.ProbeFailure
		if !errors.As(err, &pf) || len(got) != 1 || pf.Word != c.word {
			t.Errorf("%s: %+v %v", c.command, got, err)
			continue
		}
		if c.word == "nosuchruntime" {
			want := "Not started: the project's hook nosuchruntime x.js needs nosuchruntime, which is not installed where " +
				"sessions run. The owner installs it, or removes PostToolUse from workflow.hooks in config/projects/p.json."
			if pf.NotStarted("p") != want || err.Error() != "workflow-probe-failed: nosuchruntime x.js needs nosuchruntime" {
				t.Errorf("texts %q / %q", pf.NotStarted("p"), err.Error())
			}
		}
	}
}
