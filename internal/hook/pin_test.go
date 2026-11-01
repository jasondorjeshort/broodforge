// SPDX-License-Identifier: GPL-3.0-or-later

package hook

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/jasondorjeshort/broodforge/internal/git"
)

// The P38 pin (08-implementation-plan-landing.md): on an Agent call it answers allow, the hook
// appends the call's pinned model to the session's trailers file, tests/hook/pin/ copied per case.

const pinDir = "../../tests/hook/pin"

var (
	sonnetLine = git.Identity{Name: "Claude Sonnet 5.5", Email: git.ModelEmail}.Trailer()
	haikuLine  = git.Identity{Name: "Claude Haiku 5.5", Email: git.ModelEmail}.Trailer()
	opusLine   = git.Identity{Name: "Claude Opus 5.5", Email: git.ModelEmail}.Trailer()
)

// pinWorld is a copy of tests/hook/pin/ under a temp dir: sess/tree, sess/home/trailers (one
// Sonnet line) and workflow/agents.json (runner pinned to haiku, builder unpinned).
type pinWorld struct {
	tree, home, workflow, log string
}

func newPinWorld(t *testing.T) pinWorld {
	t.Helper()
	root := t.TempDir()
	w := pinWorld{tree: filepath.Join(root, "sess", "tree"), home: filepath.Join(root, "sess", "home"),
		workflow: filepath.Join(root, "workflow"), log: filepath.Join(root, "sess", "home", "hook.log")}
	for src, dst := range map[string]string{"home/trailers": filepath.Join(w.home, "trailers"),
		"workflow/agents.json": filepath.Join(w.workflow, agentsFile), "tree/README": filepath.Join(w.tree, "README")} {
		b, err := os.ReadFile(filepath.Join(pinDir, src))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dst, b, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return w
}

// run is one pre-tool-use of an Agent call with input, BROODFORGE_WORKFLOW set when workflow,
// and any extra variables; it returns the decision and the trailers file's lines.
func (w pinWorld) run(t *testing.T, input string, workflow bool, extra map[string]string) (string, []string) {
	t.Helper()
	vars := map[string]string{EnvSession: "s0001", EnvMode: "policy", EnvReach: "code", EnvTree: filepath.ToSlash(w.tree),
		EnvCanonical: filepath.ToSlash(w.tree), EnvHookLog: w.log}
	if workflow {
		vars[EnvWorkflow] = filepath.ToSlash(w.workflow)
	}
	for k, v := range extra {
		vars[k] = v
	}
	stdin, err := json.Marshal(engineInput{ToolName: "Agent", ToolInput: json.RawMessage(input), ToolUseID: "toolu_01",
		Cwd: filepath.ToSlash(w.tree), TranscriptPath: "/tmp/cfg/projects/x/s.jsonl"})
	if err != nil {
		t.Fatal(err)
	}
	code, out, errs := run(string(stdin), vars)
	if code != 0 || errs != "" {
		t.Fatalf("exit %d stderr %q", code, errs)
	}
	var o Output
	if err := json.Unmarshal([]byte(out), &o); err != nil {
		t.Fatalf("stdout %q: %v", out, err)
	}
	return o.HookSpecificOutput.PermissionDecision, w.trailers(t)
}

func (w pinWorld) trailers(t *testing.T) []string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(w.home, "trailers"))
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
}

// pinLines are the log's pin lines as `<pin>: <why>`.
func (w pinWorld) pinLines(t *testing.T) []string {
	t.Helper()
	var out []string
	for _, m := range readLog(t, w.log) {
		if _, ok := m["pin"]; ok {
			out = append(out, field(m, "pin")+": "+field(m, "why"))
		}
	}
	return out
}

func TestPinLineIsGitsTrailer(t *testing.T) {
	if PinLine("claude-haiku-5-5") != haikuLine || trailerKey != git.TrailerKey || modelEmail != git.ModelEmail {
		t.Errorf("PinLine %q, want %q", PinLine("claude-haiku-5-5"), haikuLine)
	}
}

func TestPinWritesTheTypesModel(t *testing.T) {
	w := newPinWorld(t)
	d, lines := w.run(t, `{"subagent_type":"runner","description":"d","prompt":"p"}`, true, nil)
	if d != "allow" || strings.Join(lines, "\n") != sonnetLine+"\n"+haikuLine {
		t.Errorf("runner: %s %q", d, lines)
	}
	if got := w.pinLines(t); len(got) != 0 {
		t.Errorf("a written pin logged %q", got)
	}
	// Again: the line is there, nothing is added.
	if _, lines = w.run(t, `{"subagent_type":"runner"}`, true, nil); len(lines) != 2 {
		t.Errorf("a second runner call: %q", lines)
	}
}

func TestPinInputModelOutranksTheType(t *testing.T) {
	w := newPinWorld(t)
	d, lines := w.run(t, `{"subagent_type":"runner","model":"opus"}`, true, nil)
	if d != "allow" || strings.Join(lines, "\n") != sonnetLine+"\n"+opusLine {
		t.Errorf("model opus: %s %q", d, lines)
	}
}

func TestPinNothingPinsIsLogged(t *testing.T) {
	w := newPinWorld(t)
	for _, input := range []string{`{"subagent_type":"general-purpose"}`, `{"subagent_type":"builder"}`} {
		if d, lines := w.run(t, input, true, nil); d != "allow" || strings.Join(lines, "\n") != sonnetLine {
			t.Errorf("%s: %s %q", input, d, lines)
		}
	}
	want := []string{`none: subagent_type "general-purpose" is not in the copy's agents.json`, `none: subagent_type "builder" names no model`}
	if got := w.pinLines(t); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("log pin lines %q, want %q", got, want)
	}
}

func TestPinDenyWritesNothing(t *testing.T) {
	w := newPinWorld(t)
	d, lines := w.run(t, `{"subagent_type":"runner"}`, true, map[string]string{EnvMode: "prompt", EnvAnswer: "toolu_01:deny"})
	if d != "deny" || strings.Join(lines, "\n") != sonnetLine {
		t.Errorf("deny: %s %q", d, lines)
	}
	if got := w.pinLines(t); len(got) != 0 {
		t.Errorf("a deny logged %q", got)
	}
	// A deferred call's resumed run with allow writes it.
	if d, lines = w.run(t, `{"subagent_type":"runner"}`, true, map[string]string{EnvMode: "prompt", EnvAnswer: "toolu_01:allow"}); d != "allow" || len(lines) != 2 {
		t.Errorf("resumed allow: %s %q", d, lines)
	}
}

func TestPinWithoutWorkflow(t *testing.T) {
	w := newPinWorld(t)
	if d, lines := w.run(t, `{"subagent_type":"runner","model":"opus"}`, false, nil); d != "allow" || strings.Join(lines, "\n") != sonnetLine+"\n"+opusLine {
		t.Errorf("no workflow, model opus: %s %q", d, lines)
	}
	if d, lines := w.run(t, `{"subagent_type":"runner"}`, false, nil); d != "allow" || len(lines) != 2 {
		t.Errorf("no workflow, no model: %s %q", d, lines)
	}
	if got := w.pinLines(t); strings.Join(got, "\n") != `none: subagent_type "runner": the session's copy defines no agent` {
		t.Errorf("log pin lines %q", got)
	}
}

// A failed write is one log line and the decision stands.
func TestPinFailedWriteKeepsTheDecision(t *testing.T) {
	w := newPinWorld(t)
	w.log = filepath.Join(filepath.Dir(w.home), "hook.log")
	if err := os.RemoveAll(w.home); err != nil {
		t.Fatal(err)
	}
	vars := map[string]string{EnvSession: "s0001", EnvMode: "policy", EnvReach: "code", EnvTree: filepath.ToSlash(w.tree),
		EnvCanonical: filepath.ToSlash(w.tree), EnvHookLog: w.log, EnvWorkflow: filepath.ToSlash(w.workflow)}
	stdin, _ := json.Marshal(engineInput{ToolName: "Agent", ToolInput: json.RawMessage(`{"subagent_type":"runner"}`),
		ToolUseID: "toolu_01", Cwd: filepath.ToSlash(w.tree), TranscriptPath: "/tmp/cfg/projects/x/s.jsonl"})
	code, out, errs := run(string(stdin), vars)
	if code != 0 || errs != "" || !strings.Contains(out, `"permissionDecision":"allow"`) {
		t.Fatalf("exit %d stdout %q stderr %q", code, out, errs)
	}
	got := w.pinLines(t)
	if len(got) != 1 || !strings.HasPrefix(got[0], "failed: the trailers file: ") {
		t.Errorf("log pin lines %q", got)
	}
}

// try resolves the same pin and writes nothing.
func TestPinTry(t *testing.T) {
	w := newPinWorld(t)
	d, _, pin, err := Try{Tool: "Agent", Input: json.RawMessage(`{"subagent_type":"runner"}`), Mode: "policy", Reach: "code",
		Tree: w.tree, Workflow: w.workflow, Transcript: "/tmp/cfg/projects/x/s.jsonl"}.Run()
	if err != nil || d != "allow" || pin != haikuLine {
		t.Errorf("try: %s %q %v", d, pin, err)
	}
	if lines := w.trailers(t); strings.Join(lines, "\n") != sonnetLine {
		t.Errorf("try wrote %q", lines)
	}
	if _, err := os.Stat(w.log); !os.IsNotExist(err) {
		t.Errorf("try logged: %v", err)
	}
}

// Two hook processes appending at once (parallel Agent calls) keep both lines: the lock file
// serialises their read and rename.
func TestPinConcurrentAppendsKeepBoth(t *testing.T) {
	dir := t.TempDir()
	for round := 0; round < 50; round++ {
		path := filepath.Join(dir, "trailers")
		if err := os.WriteFile(path, []byte(sonnetLine+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		var wg sync.WaitGroup
		errs := make([]error, 2)
		for i, line := range []string{haikuLine, opusLine} {
			wg.Add(1)
			go func() {
				defer wg.Done()
				errs[i] = appendTrailer(path, line)
			}()
		}
		wg.Wait()
		b, err := os.ReadFile(path)
		if err != nil || errs[0] != nil || errs[1] != nil {
			t.Fatalf("round %d: %v %v %v", round, err, errs[0], errs[1])
		}
		if !hasLine(b, haikuLine) || !hasLine(b, opusLine) || !hasLine(b, sonnetLine) {
			t.Fatalf("round %d lost a line: %q", round, b)
		}
		if _, err := os.Stat(path + ".lock"); !os.IsNotExist(err) {
			t.Fatalf("round %d left the lock: %v", round, err)
		}
	}
}

// A relative BROODFORGE_TREE writes nothing (it would land under the hook's working
// directory): one `failed` line, the decision unchanged.
func TestPinRelativeTreeWritesNothing(t *testing.T) {
	w := newPinWorld(t)
	t.Chdir(filepath.Dir(filepath.Dir(w.tree)))
	vars := map[string]string{EnvSession: "s0001", EnvMode: "policy", EnvReach: "code", EnvTree: "sess/tree",
		EnvCanonical: "sess/tree", EnvHookLog: w.log, EnvWorkflow: filepath.ToSlash(w.workflow)}
	stdin, _ := json.Marshal(engineInput{ToolName: "Agent", ToolInput: json.RawMessage(`{"subagent_type":"runner"}`),
		ToolUseID: "toolu_01", TranscriptPath: "/tmp/cfg/projects/x/s.jsonl"})
	code, out, errs := run(string(stdin), vars)
	if code != 0 || errs != "" || !strings.Contains(out, `"permissionDecision":"allow"`) {
		t.Fatalf("exit %d stdout %q stderr %q", code, out, errs)
	}
	if lines := w.trailers(t); strings.Join(lines, "\n") != sonnetLine {
		t.Errorf("a relative tree wrote %q", lines)
	}
	if got := w.pinLines(t); strings.Join(got, "\n") != "failed: tree not absolute" {
		t.Errorf("log pin lines %q", got)
	}
}
