// SPDX-License-Identifier: GPL-3.0-or-later

package hook

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// The P18 Proof's chain half (08-implementation-plan-session-workflow.md): the children of
// tests/hook/chain/ and this repository's own hooks, run by the hook as the engine runs it.

const chainDir = "../../tests/hook/chain"

// childEnviron is the hook's environment for the tests: this process's without BROODFORGE_*,
// and on a host with drive letters CLAUDE_CODE_GIT_BASH_PATH naming a Git Bash, as the spawn
// exports it from engine.gitBashPath; the test skips when there is none.
func childEnviron(t *testing.T) []string {
	t.Helper()
	var out []string
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(strings.ToUpper(kv), "BROODFORGE_") {
			out = append(out, kv)
		}
	}
	if !drives {
		return out
	}
	bash, _ := lookupEnv(out, EnvGitBashPath)
	if bash == "" {
		bash = "C:/Program Files/Git/bin/bash.exe"
	}
	if _, err := os.Stat(bash); err != nil {
		t.Skipf("no Git Bash for the children: set %s (%v)", EnvGitBashPath, err)
	}
	return append(out, EnvGitBashPath+"="+bash)
}

func absDir(t *testing.T, p string) string {
	t.Helper()
	a, err := filepath.Abs(p)
	if err != nil {
		t.Fatal(err)
	}
	return filepath.ToSlash(a)
}

// chainVars is a code-reach session in policy mode over tree, with the copy and the log.
func chainVars(t *testing.T, tree, workflow, log string) map[string]string {
	t.Helper()
	return map[string]string{EnvSession: "s0001", EnvMode: "policy", EnvReach: "code", EnvTree: tree, EnvCanonical: tree,
		EnvHookLog: log, EnvWorkflow: workflow}
}

func bashInput(tree, command string) string {
	in, _ := json.Marshal(map[string]string{"command": command})
	b, _ := json.Marshal(engineInput{HookEventName: EventPreToolUse, ToolName: "Bash", ToolInput: in, ToolUseID: "toolu_01",
		Cwd: tree, TranscriptPath: "/tmp/cfg/projects/x/s.jsonl"})
	return string(b)
}

func readLog(t *testing.T, path string) []map[string]json.RawMessage {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var out []map[string]json.RawMessage
	for _, line := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		var m map[string]json.RawMessage
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("log line %q: %v", line, err)
		}
		out = append(out, m)
	}
	return out
}

func field(m map[string]json.RawMessage, k string) string {
	var s string
	if json.Unmarshal(m[k], &s) != nil {
		return string(m[k])
	}
	return s
}

func TestMatcher(t *testing.T) {
	cases := []struct {
		matcher, tool string
		want, bad     bool
	}{
		{"", "Bash", true, false}, {"*", "Write", true, false}, {"Bash", "Bash", true, false},
		{"Bash", "BashOutput", false, false}, {"Bash", "bash", false, false}, {"Write|Edit", "Edit", true, false},
		{"Write|Edit", "MultiEdit", false, false}, {"Notebook.*", "NotebookEdit", true, false},
		{"mcp__.*", "mcp__x__y", true, false}, {"Bash(", "Bash", false, true},
	}
	for _, c := range cases {
		got, err := matches(c.matcher, c.tool)
		if got != c.want || (err != nil) != c.bad {
			t.Errorf("matches(%q, %q) = %v %v", c.matcher, c.tool, got, err)
		}
	}
}

// TestChainPreToolUse: the exit-2 and JSON denies come back prefixed, with the trace in the
// log; an allow with updatedInput leaves the bot's decision as it was and is logged; read-only
// reach and an unset BROODFORGE_WORKFLOW run no child.
func TestChainPreToolUse(t *testing.T) {
	environ := childEnviron(t)
	tree, wf := absDir(t, chainDir), absDir(t, chainDir)
	type want struct{ decision, reason string }
	cases := []struct {
		command string
		want    want
	}{
		{"deny-exit2", want{"deny", "Denied by Broodforge: refused by the project's hook deny-exit2.sh: the chain fixture refuses deny-exit2. The project's rules on its default branch decide this."}},
		{"deny-json", want{"deny", "Denied by Broodforge: refused by the project's hook deny-json.sh: the chain fixture refuses deny-json. The project's rules on its default branch decide this."}},
		{"rewrite-me", want{"allow", "Broodforge: allowed by this channel's reach"}},
	}
	for _, c := range cases {
		log := filepath.Join(t.TempDir(), "hook.log")
		var out, errb bytes.Buffer
		code := PreToolUse(strings.NewReader(bashInput(tree, c.command)), &out, &errb, env(chainVars(t, tree, wf, log)), environ, time.Now)
		var o Output
		if code != 0 || json.Unmarshal(out.Bytes(), &o) != nil || o.HookSpecificOutput.PermissionDecision != c.want.decision ||
			o.HookSpecificOutput.PermissionDecisionReason != c.want.reason {
			t.Errorf("%s: exit %d stdout %q stderr %q", c.command, code, out.String(), errb.String())
			continue
		}
		lines := readLog(t, log)
		last := lines[len(lines)-1]
		if field(last, "decision") != c.want.decision || field(last, "cell") != "ask" {
			t.Errorf("%s: decision line %v", c.command, last)
		}
		switch c.command {
		case "deny-exit2":
			if tr := field(last, "trace"); tr != "denied: refused by the project's hook deny-exit2.sh: the chain fixture refuses deny-exit2" {
				t.Errorf("trace %q", tr)
			}
			if len(lines) != 2 || field(lines[0], "child") != "deny-exit2.sh" || field(lines[0], "outcome") != "deny" {
				t.Errorf("log %v", lines)
			}
		case "rewrite-me":
			var dropped []string
			for _, l := range lines {
				if field(l, "child") == "allow-updated.sh" {
					_ = json.Unmarshal(l["dropped"], &dropped)
				}
			}
			if !slices.Equal(dropped, []string{"additionalContext", "permissionDecision=allow", "updatedInput"}) {
				t.Errorf("dropped %q in %v", dropped, lines)
			}
			if len(lines) != 5 || last["trace"] != nil {
				t.Errorf("log %v", lines)
			}
		}
	}
	for name, vars := range map[string]map[string]string{
		"read-only": func() map[string]string { v := chainVars(t, tree, wf, ""); v[EnvReach] = "read-only"; return v }(),
		"no copy":   func() map[string]string { v := chainVars(t, tree, wf, ""); delete(v, EnvWorkflow); return v }(),
	} {
		log := filepath.Join(t.TempDir(), "hook.log")
		vars[EnvHookLog] = log
		var out, errb bytes.Buffer
		code := PreToolUse(strings.NewReader(bashInput(tree, "deny-exit2")), &out, &errb, env(vars), environ, time.Now)
		if want := map[string]string{"read-only": `"deny"`, "no copy": `"allow"`}[name]; code != 0 || !strings.Contains(out.String(), want) {
			t.Errorf("%s: exit %d stdout %q", name, code, out.String())
		}
		if lines := readLog(t, log); len(lines) != 1 || lines[0]["child"] != nil {
			t.Errorf("%s: a child ran: %v", name, lines)
		}
	}
}

// writeCopy writes a copy holding hooks.json with the given entries.
func writeCopy(t *testing.T, hooks string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, hooksFile), []byte(hooks), 0o600); err != nil {
		t.Fatal(err)
	}
	return filepath.ToSlash(dir)
}

// TestChainChildSetup: a child gets the engine's stdin bytes unchanged, the tree as its working
// directory and the copy as CLAUDE_PROJECT_DIR; its own timeout below ten seconds is its budget;
// an invalid matcher runs nothing.
func TestChainChildSetup(t *testing.T) {
	environ := childEnviron(t)
	wf := writeCopy(t, `[
 {"event":"PreToolUse","matcher":"Bash(","command":"exit 2","name":"bad","timeout":null},
 {"event":"PreToolUse","matcher":"Read","command":"cat > \"$CLAUDE_PROJECT_DIR/stdin.txt\"; echo here > here.txt","name":"record","timeout":null},
 {"event":"PreToolUse","matcher":"Glob","command":"sleep 3","name":"short","timeout":1}]`)
	tree := absDir(t, t.TempDir())
	stdin := []byte(`{"tool_name":"Read","odd":"  spaced  \u00e9"}` + "\n")
	var lines []ChildLine
	c := Chain{Workflow: wf, Tree: tree, Environ: environ, Log: func(l ChildLine) { lines = append(lines, l) }}
	deny, _, err := c.Pre("Read", stdin)
	if err != nil || deny != nil {
		t.Fatalf("Read: %v %v", deny, err)
	}
	if got, err := os.ReadFile(filepath.Join(wf, "stdin.txt")); err != nil || !bytes.Equal(got, stdin) {
		t.Errorf("stdin %q %v", got, err)
	}
	if _, err := os.Stat(filepath.Join(tree, "here.txt")); err != nil {
		t.Errorf("cwd: %v", err)
	}
	deny, _, err = c.Pre("Glob", nil)
	want := "Denied by Broodforge: the project's hook short timed out after 1 second. The project's authors fix it on the default branch, or the owner removes PreToolUse from workflow.hooks for this project."
	if err != nil || deny == nil || deny.Reason != want {
		t.Errorf("Glob: %v %v", deny, err)
	}
	if len(lines) < 1 || lines[0].Outcome != "skipped" || !strings.Contains(lines[0].Detail, "invalid matcher") {
		t.Errorf("log %+v", lines)
	}
}

// TestPostToolUse: the children's additionalContext and an exit-2 child's stderr come back as
// one additionalContext, exit 0; a tool no matcher names, read-only reach and a bad stdin print
// nothing and still exit 0.
func TestPostToolUse(t *testing.T) {
	environ := childEnviron(t)
	tree, wf := absDir(t, chainDir), absDir(t, chainDir)
	post := func(stdin string, vars map[string]string) (int, string, string) {
		var out, errb bytes.Buffer
		code := PostToolUse(strings.NewReader(stdin), &out, &errb, env(vars), environ, time.Now)
		return code, out.String(), errb.String()
	}
	write := `{"hook_event_name":"PostToolUse","tool_name":"Write","tool_input":{"file_path":"a.go"},"tool_response":{},"tool_use_id":"toolu_02"}`
	log := filepath.Join(t.TempDir(), "hook.log")
	code, out, errs := post(write, chainVars(t, tree, wf, log))
	want := `{"hookSpecificOutput":{"hookEventName":"PostToolUse","additionalContext":"chain fixture: context from context.sh\nchain fixture: stderr from post-exit2.sh"}}` + "\n"
	if code != 0 || out != want || errs != "" {
		t.Errorf("Write: exit %d stdout %q stderr %q", code, out, errs)
	}
	if lines := readLog(t, log); len(lines) != 2 || field(lines[1], "outcome") != "context" {
		t.Errorf("log %v", lines)
	}
	readOnly := chainVars(t, tree, wf, os.DevNull)
	readOnly[EnvReach] = "read-only"
	for name, c := range map[string]struct {
		stdin string
		vars  map[string]string
		errs  string
	}{
		"Read":      {strings.Replace(write, `"Write"`, `"Read"`, 1), chainVars(t, tree, wf, os.DevNull), ""},
		"read-only": {write, readOnly, ""},
		"bad stdin": {"not json", chainVars(t, tree, wf, os.DevNull), "hook: stdin is not a JSON object\n"},
	} {
		if code, out, errs := post(c.stdin, c.vars); code != 0 || out != "" || errs != c.errs {
			t.Errorf("%s: exit %d stdout %q stderr %q", name, code, out, errs)
		}
	}
}

// copyTree copies a directory, for a copy holding this repository's hooks.
func copyTree(t *testing.T, from, to string) {
	t.Helper()
	err := filepath.WalkDir(from, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(from, p)
		dst := filepath.Join(to, rel)
		if d.IsDir() {
			return os.MkdirAll(dst, 0o700)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(dst, b, 0o600)
	})
	if err != nil {
		t.Fatal(err)
	}
}

// TestChainBaseHooksC3 is check C3: this repository's git-guard.js and post-edit.js, under
// Node, as children of the hook, each inside the ten-second child budget and the hook with
// both inside the thirty seconds the engine gives it.
func TestChainBaseHooksC3(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is not on PATH: C3 runs where the base's hooks can run")
	}
	environ := childEnviron(t)
	wf := filepath.ToSlash(t.TempDir())
	copyTree(t, "../../.claude/hooks", filepath.Join(wf, ".claude", "hooks"))
	// The commands of this repository's .claude/settings.json, as the copy renders them.
	hooks := `[
 {"event":"PreToolUse","matcher":"Bash|PowerShell","command":"node \"${CLAUDE_PROJECT_DIR:-.}/.claude/hooks/git-guard.js\"","name":"git-guard.js","timeout":null},
 {"event":"PostToolUse","matcher":"Write|Edit","command":"node \"${CLAUDE_PROJECT_DIR:-.}/.claude/hooks/post-edit.js\"","name":"post-edit.js","timeout":null}]`
	if err := os.WriteFile(filepath.Join(wf, hooksFile), []byte(hooks), 0o600); err != nil {
		t.Fatal(err)
	}
	tree := absDir(t, t.TempDir())
	if err := os.MkdirAll(filepath.Join(tree, "docs"), 0o700); err != nil {
		t.Fatal(err)
	}
	big := filepath.ToSlash(filepath.Join(tree, "docs", "big.md"))
	if err := os.WriteFile(big, []byte(strings.Repeat("line\n", 320)), 0o600); err != nil {
		t.Fatal(err)
	}
	log := filepath.Join(t.TempDir(), "hook.log")
	vars := chainVars(t, tree, wf, log)
	childMs := func(child string) []int64 {
		var ms []int64
		for _, l := range readLog(t, log) {
			if field(l, "child") == child {
				var n int64
				_ = json.Unmarshal(l["ms"], &n)
				ms = append(ms, n)
			}
		}
		return ms
	}
	var hookTotal time.Duration
	for _, c := range []struct{ command, decision string }{{"git status", "allow"}, {"git reset --hard HEAD", "deny"}} {
		var out, errb bytes.Buffer
		start := time.Now()
		code := PreToolUse(strings.NewReader(bashInput(tree, c.command)), &out, &errb, env(vars), environ, time.Now)
		took := time.Since(start)
		hookTotal += took
		var o Output
		if code != 0 || json.Unmarshal(out.Bytes(), &o) != nil || o.HookSpecificOutput.PermissionDecision != c.decision {
			t.Errorf("%s: exit %d stdout %q stderr %q", c.command, code, out.String(), errb.String())
		}
		if c.decision == "deny" && !strings.HasPrefix(o.HookSpecificOutput.PermissionDecisionReason, "Denied by Broodforge: refused by the project's hook git-guard.js: ") {
			t.Errorf("reason %q", o.HookSpecificOutput.PermissionDecisionReason)
		}
		t.Logf("pre-tool-use %q: %s in %v", c.command, o.HookSpecificOutput.PermissionDecision, took.Round(time.Millisecond))
	}
	in, _ := json.Marshal(map[string]string{"file_path": big})
	stdin, _ := json.Marshal(engineInput{HookEventName: EventPostToolUse, ToolName: "Write", ToolInput: in, ToolUseID: "toolu_03", Cwd: tree})
	var out, errb bytes.Buffer
	start := time.Now()
	code := PostToolUse(bytes.NewReader(stdin), &out, &errb, env(vars), environ, time.Now)
	took := time.Since(start)
	hookTotal += took
	if code != 0 || errb.String() != "" {
		t.Errorf("post-tool-use: exit %d stderr %q", code, errb.String())
	}
	t.Logf("post-tool-use Write docs/big.md in %v: %s", took.Round(time.Millisecond), strings.TrimSpace(out.String()))
	for _, child := range []string{"git-guard.js", "post-edit.js"} {
		ms := childMs(child)
		if len(ms) == 0 {
			t.Errorf("%s did not run", child)
		}
		for _, n := range ms {
			if n >= ChildBudget.Milliseconds() {
				t.Errorf("%s took %d ms, over the child budget", child, n)
			}
		}
		t.Logf("C3 %s: %v ms", child, ms)
	}
	if hookTotal >= 30*time.Second {
		t.Errorf("the hook calls took %v together, over thirty seconds", hookTotal)
	}
	t.Logf("C3 the hook, three calls with both children: %v", hookTotal.Round(time.Millisecond))
}

// TestChainWaitDelay: a child that exits while a background job it started still holds its
// pipes is judged by its exit and its output, not read as one that did not start.
func TestChainWaitDelay(t *testing.T) {
	environ := childEnviron(t)
	wf := writeCopy(t, `[
 {"event":"PreToolUse","matcher":"Bash","command":"sleep 4 & exit 0","name":"bg-allow","timeout":null},
 {"event":"PreToolUse","matcher":"Glob","command":"echo held >&2; sleep 4 & exit 2","name":"bg-deny","timeout":null},
 {"event":"PostToolUse","matcher":"Write","command":"echo '{\"hookSpecificOutput\":{\"hookEventName\":\"PostToolUse\",\"additionalContext\":\"kept\"}}'; sleep 4 & exit 0","name":"bg-post","timeout":null}]`)
	c := Chain{Workflow: wf, Tree: absDir(t, t.TempDir()), Environ: environ}
	if deny, _, err := c.Pre("Bash", nil); err != nil || deny != nil {
		t.Errorf("exit 0 with a background job: %+v %v", deny, err)
	}
	deny, _, err := c.Pre("Glob", nil)
	if err != nil || deny == nil || !strings.Contains(deny.Reason, "refused by the project's hook bg-deny: held. ") {
		t.Errorf("exit 2 with a background job: %+v %v", deny, err)
	}
	if got, err := c.Post("Write", nil); err != nil || !slices.Equal(got, []string{"kept"}) {
		t.Errorf("post with a background job: %q %v", got, err)
	}
}

// TestChainBudgetTexts: a child over its own budget is named with that budget; one the chain's
// twenty seconds stopped, or never let start, says so instead.
func TestChainBudgetTexts(t *testing.T) {
	environ := childEnviron(t)
	wf := writeCopy(t, `[{"event":"PreToolUse","matcher":"Bash","command":"sleep 3","name":"slow","timeout":8}]`)
	t0 := time.Now()
	clockAfter := func(offset time.Duration) func() time.Time {
		calls := 0
		return func() time.Time {
			calls++
			if calls == 1 {
				return t0 // the chain's start
			}
			return t0.Add(offset).Add(time.Since(t0))
		}
	}
	cases := map[time.Duration]string{
		19 * time.Second: "Denied by Broodforge: the project's hook slow was stopped: the project's hooks exceeded twenty seconds together before slow finished.",
		21 * time.Second: "Denied by Broodforge: the project's hook slow did not run: the project's hooks exceeded twenty seconds together before slow ran.",
	}
	for offset, want := range cases {
		c := Chain{Workflow: wf, Tree: absDir(t, t.TempDir()), Environ: environ, now: clockAfter(offset)}
		deny, _, err := c.Pre("Bash", nil)
		if err != nil || deny == nil || deny.Reason != want {
			t.Errorf("chain at %v: %+v %v", offset, deny, err)
		}
	}
	if got := overBudgetText("slow", 8, EventPreToolUse); !strings.Contains(got, "timed out after 8 seconds.") {
		t.Errorf("own budget %q", got)
	}
}

// TestChainJSONReason: a JSON deny's reason is taken as an exit-2 reason is, its first line,
// capped at maxReason bytes.
func TestChainJSONReason(t *testing.T) {
	environ := childEnviron(t)
	long := strings.Repeat("a", 2*maxReason)
	wf := writeCopy(t, `[
 {"event":"PreToolUse","matcher":"Bash","command":"printf '%s' '{\"hookSpecificOutput\":{\"permissionDecision\":\"deny\",\"permissionDecisionReason\":\"first line\\nsecond line\"}}'","name":"multi","timeout":null},
 {"event":"PreToolUse","matcher":"Glob","command":"echo '{\"hookSpecificOutput\":{\"permissionDecision\":\"deny\",\"permissionDecisionReason\":\"`+long+`\"}}'","name":"long","timeout":null}]`)
	c := Chain{Workflow: wf, Tree: absDir(t, t.TempDir()), Environ: environ}
	deny, trace, err := c.Pre("Bash", nil)
	if err != nil || deny == nil || !strings.Contains(deny.Reason, "hook multi: first line. The project's") || trace != "denied: refused by the project's hook multi: first line" {
		t.Errorf("multi-line: %+v %q %v", deny, trace, err)
	}
	deny, _, err = c.Pre("Glob", nil)
	if err != nil || deny == nil || !strings.Contains(deny.Reason, "hook long: "+long[:maxReason]+". The project's") {
		t.Errorf("long: %+v %v", deny, err)
	}
}

// TestChainWorkflowDir: BROODFORGE_WORKFLOW naming no directory is a hook error, exit 2; a
// directory without hooks.json runs no child.
func TestChainWorkflowDir(t *testing.T) {
	tree := absDir(t, t.TempDir())
	vars := chainVars(t, tree, filepath.Join(t.TempDir(), "gone"), os.DevNull)
	var out, errb bytes.Buffer
	code := PreToolUse(strings.NewReader(bashInput(tree, "x")), &out, &errb, env(vars), nil, time.Now)
	if code != 2 || out.Len() != 0 || !strings.HasPrefix(errb.String(), "hook: "+EnvWorkflow+" names no directory: ") {
		t.Errorf("missing copy: exit %d stdout %q stderr %q", code, out.String(), errb.String())
	}
	if deny, _, err := (Chain{Workflow: t.TempDir(), Tree: tree}).Pre("Bash", nil); deny != nil || err != nil {
		t.Errorf("copy without hooks.json: %+v %v", deny, err)
	}
}

// TestChainReachGate: a call the bot allows in read-only reach runs no child; the same call in
// code reach runs the matching child, which denies.
func TestChainReachGate(t *testing.T) {
	environ := childEnviron(t)
	tree, wf := absDir(t, chainDir), absDir(t, chainDir)
	read := `{"tool_name":"Read","tool_input":{"file_path":"deny-exit2.txt"},"tool_use_id":"toolu_04","cwd":"` + tree +
		`","transcript_path":"/tmp/cfg/projects/x/s.jsonl"}`
	for reach, want := range map[string]string{"read-only": "allow", "code": "deny"} {
		log := filepath.Join(t.TempDir(), "hook.log")
		vars := chainVars(t, tree, wf, log)
		vars[EnvReach] = reach
		var out, errb bytes.Buffer
		code := PreToolUse(strings.NewReader(read), &out, &errb, env(vars), environ, time.Now)
		if code != 0 || !strings.Contains(out.String(), `"permissionDecision":"`+want+`"`) {
			t.Errorf("%s: exit %d stdout %q stderr %q", reach, code, out.String(), errb.String())
		}
		lines := readLog(t, log)
		children := len(lines) - 1
		if (reach == "read-only") != (children == 0) || (reach == "code" && field(lines[0], "child") != "deny-exit2.sh-2") {
			t.Errorf("%s: log %v", reach, lines)
		}
	}
}
