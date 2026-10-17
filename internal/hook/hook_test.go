// SPDX-License-Identifier: GPL-3.0-or-later

package hook

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// link makes a directory link at link pointing to target: a symlink, or where symlinks need a
// privilege the test lacks (Windows without Developer Mode), a directory junction, which needs
// none and which filepath.EvalSymlinks does not follow.
func link(t *testing.T, target, at string) {
	t.Helper()
	if err := os.Symlink(target, at); err == nil {
		return
	}
	if out, err := exec.Command("cmd", "/c", "mklink", "/J", at, target).CombinedOutput(); err != nil {
		t.Skipf("cannot make a symlink or a junction: %v %s", err, out)
	}
}

type world struct {
	tree, canon, outside, cfg, transcript string
}

func newWorld(t *testing.T) world {
	t.Helper()
	root := t.TempDir()
	w := world{tree: filepath.Join(root, "tree"), canon: filepath.Join(root, "canon"),
		outside: filepath.Join(root, "outside"), cfg: filepath.Join(root, "cfg")}
	for _, d := range []string{w.tree, w.canon, w.outside, filepath.Join(w.cfg, "projects", "enc")} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	w.transcript = filepath.Join(w.cfg, "projects", "enc", "s.jsonl")
	link(t, w.outside, filepath.Join(w.tree, "out"))
	link(t, w.cfg, filepath.Join(w.tree, "cfglink"))
	return w
}

func (w world) try(tool, mode, reach string, input map[string]string) (string, error) {
	b, err := json.Marshal(input)
	if err != nil {
		return "", err
	}
	d, _, err := Try{Tool: tool, Input: b, Mode: mode, Reach: reach, Tree: w.tree, Canonical: w.canon, Transcript: w.transcript}.Run()
	return d, err
}

func TestLinksOutOfTheTree(t *testing.T) {
	w := newWorld(t)
	cases := []struct {
		tool, reach, path, want string
	}{
		{"Write", "code", filepath.Join(w.tree, "a.go"), "allow"},
		{"Write", "code", filepath.Join(w.tree, "out", "new.go"), "deny"},
		{"Write", "code", "out/new.go", "deny"},
		{"Read", "read-only", filepath.Join(w.tree, "out", "x"), "deny"},
		{"Read", "code", filepath.Join(w.tree, "out", "x"), "allow"},
		{"Read", "code", filepath.Join(w.tree, "cfglink", "settings.json"), "deny"},
		{"Read", "code", filepath.Join(w.cfg, "settings.json"), "deny"},
		{"Grep", "code", filepath.Dir(w.cfg), "deny"},
		{"Grep", "code", w.tree, "allow"},
		// A link then ..: POSIX applies .. to the link's target, so the path is beside it,
		// outside the tree; Windows cleans first and lands in the tree. The stricter wins.
		{"Read", "read-only", w.tree + "/out/../x", "deny"}, // not filepath.Join, which would clean the ..
		{"Write", "code", "out/../a.go", "deny"},
		{"Write", "code", "sub/../a.go", "allow"},
		{"Read", "code", w.tree + "/cfglink/../cfg/settings.json", "deny"},
	}
	for _, c := range cases {
		key := "file_path"
		if c.tool == "Grep" {
			key = "path"
		}
		got, err := w.try(c.tool, "policy", c.reach, map[string]string{key: c.path})
		if err != nil || got != c.want {
			t.Errorf("%s %s %s: %s %v, want %s", c.tool, c.reach, c.path, got, err, c.want)
		}
	}
}

func TestSpellings(t *testing.T) {
	refused := []string{`\\?\C:\x\a`, `//?/C:/x/a`, `\\.\C:\x`, `\??\C:\x`, `//localhost/C$/x`, `\\srv\share\x`,
		`D:secret`, `C:`, `a/.credentials.json::$DATA`, `x:y`}
	for _, p := range refused {
		if err := checkSpelling(p); err == nil && (strings.ContainsRune(p, ':') || filepath.VolumeName(p) != "") {
			t.Errorf("%q: not refused", p)
		}
	}
	for _, p := range []string{`C:\x\a`, `C:/x/a`, `/tmp/x`, `a/b`, `..`} {
		if err := checkSpelling(p); err != nil {
			t.Errorf("%q: refused: %v", p, err)
		}
	}
	for pattern, want := range map[string]string{"C:/Windows/*.ini": "C:/Windows/", "../outside/*": "../outside/",
		"**/*.go": "", "src/a.go": "src/a.go", "/etc/x*/y": "/etc/", `a\b?`: `a\`, "{a,b}/c": ""} {
		if got := globDir(pattern); got != want {
			t.Errorf("globDir(%q) = %q, want %q", pattern, got, want)
		}
	}
	for name, want := range map[string]bool{".credentials.json": true, ".CREDENTIALS.JSON": true,
		".credentials.json. ": true, ".credentials.jsonx": false, "credentials.json": false} {
		if isCredentialName(name) != want {
			t.Errorf("isCredentialName(%q) != %v", name, want)
		}
	}
}

// TestHardLinkToAnotherHarness: under ADR 0022's layout the hook derives the instance root from
// the config directory and knows every harness's credential files by identity.
func TestHardLinkToAnotherHarness(t *testing.T) {
	root := t.TempDir()
	cfg := filepath.Join(root, "harness", "cred-a")
	other := filepath.Join(root, "harness", "cred-b")
	tree := filepath.Join(root, "sessions", "s0001", "tree")
	for _, d := range []string{filepath.Join(cfg, "projects", "x"), other, tree} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	for _, f := range []string{filepath.Join(other, ".credentials.json"), filepath.Join(other, ".claude.json")} {
		if err := os.WriteFile(f, []byte("{}"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Link(filepath.Join(other, ".credentials.json"), filepath.Join(tree, "innocent.txt")); err != nil {
		t.Skipf("no hard links here: %v", err)
	}
	if err := os.Link(filepath.Join(other, ".claude.json"), filepath.Join(tree, "notes.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tree, "plain.txt"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]string{"innocent.txt": "deny", "notes.md": "deny", "plain.txt": "allow"} {
		for _, reach := range []string{"read-only", "code"} {
			d, _, err := Try{Tool: "Read", Input: json.RawMessage(`{"file_path":"` + name + `"}`), Mode: "policy", Reach: reach,
				Tree: tree, Transcript: filepath.Join(cfg, "projects", "x", "s.jsonl")}.Run()
			if err != nil || d != want {
				t.Errorf("%s %s: %s %v, want %s", name, reach, d, err, want)
			}
		}
	}
}

func TestExpand(t *testing.T) {
	sep := string(filepath.Separator)
	cases := map[string]string{"  a.go\t": "a.go", "\ufeff\u00a0a.go\u2028": "a.go", "\u200ba.go": "\u200ba.go", "\u0085a.go": "\u0085a.go", "~": "/h", "~/x": "/h" + sep + "x", `~\x`: "/h" + sep + "x", "/tmp/x": "/tmp/x"}
	if drives {
		cases["/c/Users/user"] = `C:\Users/user`
		cases["/d/"] = `D:\`
		cases["/d"] = "/d"
		cases[`/c\x`] = `/c\x`
		cases["/cc/x"] = "/cc/x"
	} else {
		cases["/c/Users/user"] = "/c/Users/user"
	}
	for in, want := range cases {
		if got, err := expand(in, "/h"); err != nil || got != want {
			t.Errorf("expand(%q) = %q %v, want %q", in, got, err, want)
		}
	}
	for _, in := range []string{"~bf/x", "   ", ""} {
		if _, err := expand(in, "/h"); err == nil {
			t.Errorf("expand(%q): no error", in)
		}
	}
	if _, err := expand("~/x", ""); err == nil {
		t.Error("~ with no home: no error")
	}
}

func TestGlobDirs(t *testing.T) {
	refused := []string{"*/../../outside/*", "{C:/Windows/*,x}", "{..,x}/*", "[.][.]/outside/*", "E:*", "[/]x", "{a,{b,c}}", "{a,b", "a}", "x/../y"}
	for _, p := range refused {
		if _, err := globDirs(p); err == nil {
			t.Errorf("globDirs(%q): not refused", p)
		}
	}
	for p, want := range map[string]string{"src/{a,b}/*.go": "src/a/|src/b/", "**/*.{go,md}": "|", "*": "", "a/*": "a/"} {
		got, err := globDirs(p)
		if err != nil || strings.Join(got, "|") != want {
			t.Errorf("globDirs(%q) = %q %v, want %q", p, got, err, want)
		}
	}
}

func TestLinkedTree(t *testing.T) {
	w := newWorld(t)
	alias := filepath.Join(filepath.Dir(w.tree), "alias")
	link(t, w.tree, alias)
	// The tree named through a link and a path named directly agree once both are canonical.
	d, _, err := Try{Tool: "Write", Input: json.RawMessage(`{"file_path":"` + filepath.ToSlash(filepath.Join(w.tree, "a.go")) + `"}`),
		Mode: "policy", Reach: "code", Tree: alias, Transcript: w.transcript}.Run()
	if err != nil || d != "allow" {
		t.Errorf("linked tree: %s %v", d, err)
	}
}

func env(vars map[string]string) func(string) (string, bool) {
	return func(k string) (string, bool) { v, ok := vars[k]; return v, ok }
}

func baseEnv(log string) map[string]string {
	return map[string]string{EnvSession: "s0001", EnvMode: "prompt", EnvReach: "code", EnvTree: "/tmp/x",
		EnvCanonical: "/tmp/c", EnvHookLog: log}
}

const writeInput = `{"tool_name":"Write","tool_input":{"file_path":"/tmp/x/a.ts","content":""},"tool_use_id":"toolu_01","cwd":"/tmp/x","transcript_path":"/tmp/cfg/projects/x/s.jsonl"}`

var fixed = func() time.Time { return time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC) }

func run(stdin string, vars map[string]string) (int, string, string) {
	var out, errb bytes.Buffer
	code := PreToolUse(strings.NewReader(stdin), &out, &errb, env(vars), fixed)
	return code, out.String(), errb.String()
}

func TestPreToolUseDefer(t *testing.T) {
	log := filepath.Join(t.TempDir(), "hook.log")
	code, out, errs := run(writeInput, baseEnv(log))
	want := `{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"defer","permissionDecisionReason":"Broodforge: waiting for a person in this session to answer"}}` + "\n"
	if code != 0 || out != want || errs != "" {
		t.Fatalf("exit %d\nstdout %q\nstderr %q", code, out, errs)
	}
	b, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(b); got != `{"time":"2026-10-16T12:00:00Z","session":"s0001","tool":"Write","toolUseId":"toolu_01","cell":"ask","decision":"defer"}`+"\n" {
		t.Errorf("log %q", got)
	}
}

func TestPreToolUseAnswer(t *testing.T) {
	vars := baseEnv(os.DevNull)
	vars[EnvAnswer] = "toolu_01:deny"
	vars[EnvAnswerReason] = "the prompt timed out after 5 minutes"
	code, out, _ := run(writeInput, vars)
	if code != 0 || !strings.Contains(out, `"permissionDecision":"deny","permissionDecisionReason":"the prompt timed out after 5 minutes"`) {
		t.Errorf("exit %d stdout %q", code, out)
	}
}

func TestPreToolUseErrorsExit2(t *testing.T) {
	missing := func(name string) map[string]string { v := baseEnv(os.DevNull); delete(v, name); return v }
	cases := []struct {
		name, stdin string
		vars        map[string]string
		stderr      string
	}{
		{"no transcript_path", strings.Replace(writeInput, `,"transcript_path":"/tmp/cfg/projects/x/s.jsonl"`, "", 1), baseEnv(os.DevNull), "hook: no transcript_path\n"},
		{"malformed stdin", `{"tool_name":`, baseEnv(os.DevNull), "hook: stdin is not a JSON object\n"},
		{"empty stdin", ``, baseEnv(os.DevNull), "hook: stdin is not a JSON object\n"},
		{"array stdin", `[1]`, baseEnv(os.DevNull), "hook: stdin is not a JSON object\n"},
		{"no tool_use_id", `{"tool_name":"Bash","tool_input":{},"transcript_path":"/c/projects/x/s.jsonl"}`, baseEnv(os.DevNull), "hook: no tool_use_id\n"},
		{"missing variable", writeInput, missing(EnvTree), "hook: BROODFORGE_TREE is not set\n"},
		{"missing log", writeInput, missing(EnvHookLog), "hook: BROODFORGE_HOOK_LOG is not set\n"},
		{"bad mode", writeInput, func() map[string]string { v := baseEnv(os.DevNull); v[EnvMode] = "auto"; return v }(), "hook: BROODFORGE_MODE is \"auto\", want policy or prompt\n"},
		{"bad answer", writeInput, func() map[string]string { v := baseEnv(os.DevNull); v[EnvAnswer] = "allow"; return v }(), "hook: BROODFORGE_ANSWER is not <tool_use_id>:allow|deny\n"},
		{"unwritable log", writeInput, baseEnv(filepath.Join(t.TempDir(), "no", "such", "dir", "hook.log")), ""},
	}
	for _, c := range cases {
		code, out, errs := run(c.stdin, c.vars)
		if code != 2 || out != "" || !strings.HasPrefix(errs, "hook: ") || (c.stderr != "" && errs != c.stderr) {
			t.Errorf("%s: exit %d stdout %q stderr %q, want exit 2 and %q", c.name, code, out, errs, c.stderr)
		}
	}
}

// TestShippedCases runs tests/hook/cases.jsonl: every case ok, or skip for a link this host
// does not have.
func TestShippedCases(t *testing.T) {
	f, err := os.Open("../../tests/hook/cases.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	results, err := Cases(f, "../../tests/hook")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 {
		t.Fatal("no cases")
	}
	for _, r := range results {
		if r.Result == ResultFail {
			t.Errorf("line %d %s %s %s: got %s, want %s (%s)", r.Line, r.Tool, r.Mode, r.Reach, r.Got, r.Want, r.Note)
		}
	}
}

func TestBadCaseFile(t *testing.T) {
	for _, in := range []string{`{"tool":"Read","expect":"maybe"}`, `not json`} {
		if _, err := Cases(strings.NewReader(in), "."); err == nil {
			t.Errorf("%q: no error", in)
		}
	}
}
