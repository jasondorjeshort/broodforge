// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The samples use a made-up PC user; the real one must not appear in the repository.
const user = "alice"

func TestRewrite(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"forward slashes, spike root",
			`"cwd":"C:/Users/alice/AppData/Local/Temp/bf-spike/tree"`,
			`"cwd":"/home/user/bf-spike/tree"`},
		{"JSON-escaped, spike root",
			`"file_path":"C:\\Users\\alice\\AppData\\Local\\Temp\\bf-spike\\tree\\hello.txt"`,
			`"file_path":"/home/user/bf-spike/tree/hello.txt"`},
		{"nested JSON, home; a shorter backslash run ends the token",
			`"stdin":"{\"cwd\":\"C:\\\\Users\\\\alice\\\\Documents\\\\src\\\\x\\\\hook.js\\\"}"`,
			`"stdin":"{\"cwd\":\"/home/user/Documents/src/x/hook.js\\\"}"`},
		{"raw backslashes, home",
			`C:\Users\alice\notes.txt`,
			`/home/user/notes.txt`},
		{"home rule for paths outside the spike root",
			`"--mcp-config","C:/Users/alice/Documents/src/broodforge/spike/share/empty/mcp.json"`,
			`"--mcp-config","/home/user/Documents/src/broodforge/spike/share/empty/mcp.json"`},
		{"a JSON escape ends a forward-slash token",
			`LOG=C:/Users/alice/AppData/Local/Temp/bf-spike/hook.log\nTREE=C:/Users/alice/AppData/Local/Temp/bf-spike/tree\n`,
			`LOG=/home/user/bf-spike/hook.log\nTREE=/home/user/bf-spike/tree\n`},
		{"a JSON escape ends an escaped token",
			`"stdout":"C:\\Users\\alice\\x.txt\n"`,
			`"stdout":"/home/user/x.txt\n"`},
		{"trailing separator and the spike slug",
			`"auto":"C:\\Users\\alice\\AppData\\Local\\Temp\\bf-spike\\cfg\\projects\\C--Users-alice-AppData-Local-Temp-bf-spike-tree\\memory\\"`,
			`"auto":"/home/user/bf-spike/cfg/projects/-home-user-bf-spike-tree/memory/"`},
		{"slug inside a home path, prose after",
			`written to: C:\\Users\\alice\\AppData\\Local\\Temp\\claude\\C--Users-alice-AppData-Local-Temp-bf-spike-tree\\0f\\tasks\\b.output. You`,
			`written to: /home/user/AppData/Local/Temp/claude/-home-user-bf-spike-tree/0f/tasks/b.output. You`},
		{"home slug",
			`projects/C--Users-alice-Documents-src-broodforge/`,
			`projects/-home-user-Documents-src-broodforge/`},
		{"bracket and backtick close a token",
			"[node C:/Users/alice/Documents/s/hook.js]: hook (`C:/Users/alice/Documents/s/hook.js`, exit 2)",
			"[node /home/user/Documents/s/hook.js]: hook (`/home/user/Documents/s/hook.js`, exit 2)"},
		{"case-insensitive drive and segments",
			`c:\\users\\Alice\\appdata\\local\\temp\\BF-SPIKE\\tree`,
			`/home/user/bf-spike/tree`},
		{"engine record: a credential config dir and the tree slug under home",
			`"auto":"C:\\Users\\alice\\bf-owner\\harness\\cred-x1\\projects\\C--Users-alice-bf-p7-tree\\memory\\"`,
			`"auto":"/home/user/bf-owner/harness/cred-x1/projects/-home-user-bf-p7-tree/memory/"`},
		{"a segment that only starts like the spike root is under home",
			`C:/Users/alice/AppData/Local/Temp/bf-spike2/x`,
			`/home/user/AppData/Local/Temp/bf-spike2/x`},
		{"ADR 0029's spike root, path and slug alike",
			`"transcript_path":"C:\\Users\\alice\\.claude\\projects\\C--Users-alice-AppData-Local-Temp-bf-spike-0029-c42-allow-tree\\s.jsonl","cwd":"C:\\Users\\alice\\AppData\\Local\\Temp\\bf-spike-0029\\c42-allow\\tree"`,
			`"transcript_path":"/home/user/.claude/projects/-home-user-bf-spike-c42-allow-tree/s.jsonl","cwd":"/home/user/bf-spike/c42-allow/tree"`},
	}
	r := newRewriter(user)
	for _, c := range cases {
		if got := string(r.rewrite([]byte(c.in))); got != c.want {
			t.Errorf("%s:\n in   %s\n got  %s\n want %s", c.name, c.in, got, c.want)
		}
	}
}

func TestRewriteMSYSHome(t *testing.T) {
	for in, want := range map[string]string{
		`/c/Users/alice/x`:                             `/home/user/x`,
		`{"command":"cd /c/Users/alice/bf-p7/tree"}`:   `{"command":"cd /home/user/bf-p7/tree"}`,
		`/c/Users/alice/AppData/Local/Temp/bf-spike/t`: `/home/user/bf-spike/t`,
		`/c/Users/alice`:                               `/home/user`,
	} {
		if got := string(newRewriter(user).rewrite([]byte(in))); got != want {
			t.Errorf("%s\n got  %s\n want %s", in, got, want)
		}
	}
	for _, in := range []string{`/mnt/c/Users/alice/x`, `a/c/Users/alice/x`} {
		if got := string(newRewriter(user).rewrite([]byte(in))); got != in {
			t.Errorf("changed %s to %s", in, got)
		}
	}
}

// TestRewriteAfterJSONEscape: a home path that begins right after a JSON escape (\n, \t) is rewritten, not left to the refusal.
func TestRewriteAfterJSONEscape(t *testing.T) {
	for in, want := range map[string]string{
		`"stdout":"x\nC:\\Users\\alice\\y"`: `"stdout":"x\n/home/user/y"`,
		`"stdout":"x\t/c/Users/alice/y"`:    `"stdout":"x\t/home/user/y"`,
	} {
		if got := string(newRewriter(user).rewrite([]byte(in))); got != want {
			t.Errorf("%s\n got  %s\n want %s", in, got, want)
		}
	}
}

func TestRewriteLeavesOtherTextAlone(t *testing.T) {
	for _, in := range []string{
		`C:\\Users\\user\\x`,
		`C:\\Users\\bob\\alice\\x`,
		`"powershell_path":"C:\\Windows\\System32\\WindowsPowerShell\\v1.0\\powershell.exe"`,
		`"messaging_socket_path":"\\\\.\\pipe\\LOCAL\\cc-msg-1"`,
		`C--Users-alicia-x and xC--Users-alice-y`,
		`"text":"Examples:\nC:"`,
		`ABC:/Users/alice/x`,
	} {
		if got := string(newRewriter(user).rewrite([]byte(in))); got != in {
			t.Errorf("changed %s\n to %s", in, got)
		}
	}
}

func TestRewriteKeepsLines(t *testing.T) {
	in := []byte("{\"type\":\"system\",\"cwd\":\"C:\\\\Users\\\\alice\\\\AppData\\\\Local\\\\Temp\\\\bf-spike\\\\tree\"}\n" +
		"{\"type\":\"result\",\"result\":\"ok\"}\n")
	want := []byte("{\"type\":\"system\",\"cwd\":\"/home/user/bf-spike/tree\"}\n" +
		"{\"type\":\"result\",\"result\":\"ok\"}\n")
	if got := newRewriter(user).rewrite(in); !bytes.Equal(got, want) {
		t.Errorf("got %q\nwant %q", got, want)
	}
}

func TestSourcePattern(t *testing.T) {
	m := srcPattern.FindStringSubmatch("C:/Users/alice/AppData/Local/Temp/bf-spike/fx/basic")
	if m == nil || m[1] != "alice" || m[2] != "basic" {
		t.Fatalf("got %v", m)
	}
	if srcPattern.MatchString("/home/alice/bf-spike/fx/basic") {
		t.Fatal("matched a non-PC path")
	}
}

func TestUserFromPath(t *testing.T) {
	for in, want := range map[string]string{
		"C:/Users/alice/AppData/Local/Temp/bf-spike/fx/basic": "alice",
		"C:/Users/alice/bf-p7/rec":                            "alice",
		"d:/users/alice/x":                                    "alice",
	} {
		if got, err := userFromPath(in); err != nil || got != want {
			t.Errorf("%s: got %q, %v", in, got, err)
		}
	}
	for _, in := range []string{"/home/alice/rec", "C:/Users", "C:/tmp/rec"} {
		if got, err := userFromPath(in); err == nil {
			t.Errorf("%s: read user %q", in, got)
		}
	}
}

// writeFiles writes name to content pairs into a new temporary directory.
func writeFiles(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

const recordedFx = "{\n  \"claim\": \"recorded\",\n  \"engineVersion\": \"2.1.296\",\n  \"recorded\": \"2026-10-17\",\n" +
	"  \"turns\": [\n    {\n      \"n\": 1,\n      \"exit\": 0,\n      \"engineSessionId\": \"s1\"\n    }\n  ]\n}\n"

func TestImportRecordedDirectory(t *testing.T) {
	turn := "{\"type\":\"system\",\"cwd\":\"C:\\\\Users\\\\alice\\\\bf-p7\\\\tree\"," +
		"\"auto\":\"C:\\\\Users\\\\alice\\\\bf-owner\\\\cred\\\\projects\\\\C--Users-alice-bf-p7-tree\\\\memory\\\\\"}\n" +
		"{\"type\":\"result\",\"result\":\"hello\"}\n"
	src := writeFiles(t, map[string]string{"turn-1.jsonl": turn, "fixture.json": recordedFx, "received-1.json": "{}"})
	dst := filepath.Join(t.TempDir(), "recorded-read")
	if err := importDir(src, dst, "", user, false); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dst, "turn-1.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	want := "{\"type\":\"system\",\"cwd\":\"/home/user/bf-p7/tree\"," +
		"\"auto\":\"/home/user/bf-owner/cred/projects/-home-user-bf-p7-tree/memory/\"}\n" +
		"{\"type\":\"result\",\"result\":\"hello\"}\n"
	if string(got) != want {
		t.Errorf("turn-1.jsonl:\n got  %s\n want %s", got, want)
	}
	fx, err := os.ReadFile(filepath.Join(dst, "fixture.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(fx) != recordedFx {
		t.Errorf("fixture.json changed:\n%s", fx)
	}
	left, _ := os.ReadDir(dst)
	if len(left) != 2 {
		t.Errorf("wrote %v, want turn-1.jsonl and fixture.json only", left)
	}
}

func TestImportRecordedRefusals(t *testing.T) {
	turn := "{\"type\":\"result\"}\n"
	for name, files := range map[string]map[string]string{
		"no fixture.json":          {"turn-1.jsonl": turn},
		"turn list does not match": {"turn-1.jsonl": turn, "turn-2.jsonl": turn, "fixture.json": recordedFx},
		"meta for some turns only": {"turn-1.jsonl": turn, "turn-2.jsonl": turn, "meta-1.json": "{}", "fixture.json": recordedFx},
		"user survives":            {"turn-1.jsonl": "{\"p\":\"/mnt/c/Users/alice/x\"}\n", "fixture.json": recordedFx},
	} {
		src := writeFiles(t, files)
		dst := filepath.Join(t.TempDir(), "out")
		if err := importDir(src, dst, "", user, false); err == nil {
			t.Errorf("%s: accepted", name)
		}
		if _, err := os.Stat(dst); err == nil {
			t.Errorf("%s: wrote %s", name, dst)
		}
	}
}

func TestMetasBesideFixtureJSONRefused(t *testing.T) {
	src := writeFiles(t, map[string]string{"turn-1.jsonl": "{}\n", "meta-1.json": "{}", "fixture.json": recordedFx})
	dst := filepath.Join(t.TempDir(), "out")
	if err := importDir(src, dst, "", user, false); err == nil {
		t.Fatal("meta-1.json and fixture.json together accepted")
	}
	if _, err := os.Stat(dst); err == nil {
		t.Fatalf("wrote %s", dst)
	}
}

func TestOtherUsersHomeRefused(t *testing.T) {
	r := newRewriter(user)
	for _, in := range []string{
		`C:/Users/bob/x`,
		`C:\Users\bob\x`,
		`"p":"C:\\Users\\bob\\x"`,
		`"stdin":"{\"p\":\"C:\\\\Users\\\\bob\\\\x\"}"`,
		`/c/Users/bob/x`,
		`projects/C--Users-bob-tree/`,
	} {
		out := map[string][]byte{"turn-1.jsonl": r.rewrite([]byte("{}\n" + in + "\n"))}
		if err := scanLeak(user, out); err == nil {
			t.Errorf("%s passed the scan", in)
		}
	}
	for _, in := range []string{`"/home/user/bf-p7/tree"`, `-home-user-bf-p7-tree`, `the users. Then`, `MyUsers/x`} {
		if err := scanLeak(user, map[string][]byte{"turn-1.jsonl": []byte(in)}); err != nil {
			t.Errorf("%s refused: %v", in, err)
		}
	}
}

func TestDestinationInsideSpikeRootRefused(t *testing.T) {
	src := "C:/Users/alice/AppData/Local/Temp/bf-spike/fx/basic"
	for _, dst := range []string{
		src,
		"C:/Users/alice/AppData/Local/Temp/bf-spike/fx/BASIC",
		"C:/Users/alice/AppData/Local/Temp/bf-spike/fx/other",
		"C:/Users/alice/AppData/Local/Temp/bf-spike",
	} {
		if err := checkDst(src, dst); err == nil {
			t.Errorf("accepted %s", dst)
		}
	}
	if err := checkDst(src, "C:/Users/alice/AppData/Local/Temp/bf-spike2/x"); err != nil {
		t.Errorf("refused a sibling of the spike root: %v", err)
	}
}

func TestSurvivingUserNameRefused(t *testing.T) {
	r := newRewriter(user)
	in := []byte("{\"a\":1}\n{\"p\":\"/mnt/c/Users/alice/x\"}\n")
	err := scanLeak(user, map[string][]byte{"turn-1.jsonl": r.rewrite(in)})
	if err == nil {
		t.Fatal("WSL form passed the scan")
	}
	for _, ok := range []string{`"C:\\Users\\alice\\x"`, "{\"p\":\"/c/Users/alice/x\"}\n"} {
		clean := map[string][]byte{"turn-1.jsonl": r.rewrite([]byte(ok))}
		if err := scanLeak(user, clean); err != nil {
			t.Fatalf("clean output %s refused: %v", ok, err)
		}
	}
}

func TestDestinationRefusedWhenNotEmpty(t *testing.T) {
	dst := t.TempDir()
	if err := prepareDst(dst, false); err != nil {
		t.Fatalf("empty dst: %v", err)
	}
	for _, f := range []string{"turn-1.jsonl", "fixture.json", "keep.txt"} {
		if err := os.WriteFile(filepath.Join(dst, f), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := prepareDst(dst, false); err == nil {
		t.Fatal("non-empty dst accepted without --force")
	}
	if err := prepareDst(dst, true); err != nil {
		t.Fatalf("--force: %v", err)
	}
	left, _ := os.ReadDir(dst)
	if len(left) != 1 || left[0].Name() != "keep.txt" {
		t.Fatalf("after --force: %v", left)
	}
}

// TestEventsRewritesEverySessionID: a captured session-tail file of s0002 holds no s0002 token
// after --events (the summaries, start.branch, end.branch, a custom id and a worktree path in a
// tool target), its PC home paths are the placeholder, read from the file without --user, and
// every other byte is kept (s00021 and xs0002 are not the token; the comment line's is).
func TestEventsRewritesEverySessionID(t *testing.T) {
	const tmpl = `# {ID} captured
{"seq":1,"time":"2026-10-30T10:00:00Z","type":"session-started","summary":"Session {ID} on Scratch, branch broodforge/scratch/{ID}, driver Ada Example.","user":"1001","start":{"channel":"100","branch":"broodforge/scratch/{ID}","driver":"1001","driverName":"Ada Example"}}
{"seq":2,"time":"2026-10-30T10:00:01Z","type":"tool-call","summary":"[Read] {HOME}/AppData/Local/Temp/bf-scratch-1/sessions/{ID}/tree/hello.txt","tool":"Read","target":"{ESC}\\AppData\\Local\\Temp\\bf-scratch-1\\sessions\\{ID}\\tree\\hello.txt"}
{"seq":3,"time":"2026-10-30T10:00:02Z","type":"waiting-for-answer","summary":"bf:{ID}:toolu_1:allow #{ID}-ada s00021 xs0002"}
{"seq":4,"time":"2026-10-30T10:00:03Z","type":"session-ended","summary":"Session {ID} ended.","end":{"branch":"broodforge/scratch/{ID}","by":"Ada Example"}}
`
	fill := func(id, home, esc string) string {
		return strings.NewReplacer("{ID}", id, "{HOME}", home, "{ESC}", esc).Replace(tmpl)
	}
	path := filepath.Join(t.TempDir(), "events.jsonl")
	// The target is the JSON-escaped form `session tail --json` writes: two backslashes a separator.
	if err := os.WriteFile(path, []byte(fill("s0002", "C:/Users/"+user, `C:\\Users\\`+user)), 0o644); err != nil {
		t.Fatal(err)
	}
	if code := run([]string{"--events", path}); code != 0 {
		t.Fatalf("exit %d", code)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Replace(fill("s0001", "/home/user", "/home/user"), `/home/user\\AppData\\Local\\Temp\\bf-scratch-1\\sessions\\s0001\\tree\\hello.txt`,
		"/home/user/AppData/Local/Temp/bf-scratch-1/sessions/s0001/tree/hello.txt", 1)
	if string(got) != want {
		t.Errorf("rewrite\n got %s\nwant %s", got, want)
	}
	if strings.Contains(strings.NewReplacer("s00021", "", "xs0002", "").Replace(string(got)), "s0002") {
		t.Errorf("an s0002 token survives:\n%s", got)
	}

	// A file with no PC home path gets no home rewrite and is not an error.
	plain := filepath.Join(t.TempDir(), "events.jsonl")
	if err := os.WriteFile(plain, []byte(`{"type":"session-started","summary":"Session s0003 on X, branch b/s0003."}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if code := run([]string{"--events", plain}); code != 0 {
		t.Fatalf("no home path: exit %d", code)
	}
	if b, _ := os.ReadFile(plain); string(b) != `{"type":"session-started","summary":"Session s0001 on X, branch b/s0001."}`+"\n" {
		t.Errorf("no home path: %s", b)
	}
	if code := run([]string{"--events", plain, "a", "b"}); code != 2 {
		t.Errorf("--events with <src> <dst>: exit %d, want 2", code)
	}
}

// TestEventsRewritesAShortMintedCredentialID: newID is base36 of 48 random bits, so about a third
// of minted ids are nine characters or fewer; the one the session-started line names still
// becomes cred-dev everywhere, and cred-scratch stays.
func TestEventsRewritesAShortMintedCredentialID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	body := `{"type":"session-started","summary":"Session s0002 on X.","start":{"credential":"cred-1x2y3z4w5"}}` + "\n" +
		`{"type":"post","summary":"Credential: cred-1x2y3z4w5 (Claude subscription), not cred-scratch"}` + "\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if code := run([]string{"--events", path}); code != 0 {
		t.Fatalf("exit %d", code)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"type":"session-started","summary":"Session s0001 on X.","start":{"credential":"cred-dev"}}` + "\n" +
		`{"type":"post","summary":"Credential: cred-dev (Claude subscription), not cred-scratch"}` + "\n"
	if string(got) != want {
		t.Errorf("rewrite\n got %s\nwant %s", got, want)
	}
}

// TestEventsRewritesMintedCredentialID: the id the session-started line names, in an event and in
// a post, becomes cred-dev; cred-scratch, another id and longer or embedded look-alikes stay.
func TestEventsRewritesMintedCredentialID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	body := `{"type":"session-started","summary":"Session s0002 on X.","start":{"credential":"cred-2ha7apqowv"}}` + "\n" +
		`{"type":"post","summary":"Credential cred-2ha7apqowv, also cred-scratch, cred-other12345, cred-2ha7apqowvx, xcred-2ha7apqowv"}` + "\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if code := run([]string{"--events", path}); code != 0 {
		t.Fatalf("exit %d", code)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"type":"session-started","summary":"Session s0001 on X.","start":{"credential":"cred-dev"}}` + "\n" +
		`{"type":"post","summary":"Credential cred-dev, also cred-scratch, cred-other12345, cred-2ha7apqowvx, xcred-2ha7apqowv"}` + "\n"
	if string(got) != want {
		t.Errorf("rewrite\n got %s\nwant %s", got, want)
	}
}

// TestEventsReadsUserFromAnyHomeForm: without --user, the PC user comes from the file's first home
// path in the JSON-escaped form `session tail --json` writes as well as with forward slashes, and
// a user name may hold an s; the rewrite then leaves no Users path, and no .tmp file is left.
func TestEventsReadsUserFromAnyHomeForm(t *testing.T) {
	bs := strings.Repeat(string(rune(92)), 2)
	for name, home := range map[string]string{
		"escaped backslashes": "C:" + bs + "Users" + bs + "alice" + bs + "x.txt",
		"forward slashes":     "C:/Users/alice/x.txt",
	} {
		path := filepath.Join(t.TempDir(), "events.jsonl")
		body := `{"type":"session-started","summary":"Session s0002 on X."}` + "\n" +
			`{"type":"tool-call","summary":"[Read] x","target":"` + home + `"}` + "\n"
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		if code := run([]string{"--events", path}); code != 0 {
			t.Errorf("%s: exit %d, want 0", name, code)
		}
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(b), "Users") {
			t.Errorf("%s: a Users path survives: %s", name, b)
		}
		if _, err := os.Stat(path + ".tmp"); err == nil {
			t.Errorf("%s: the temporary file is left", name)
		}
	}
}

// runDir writes a spike run directory named name (a key of claims) under a fresh temporary
// directory and returns its path.
func runDir(t *testing.T, name string, files map[string]string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), name)
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for f, content := range files {
		if err := os.WriteFile(filepath.Join(dir, f), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

const (
	runHookStarted  = `{"type":"system","subtype":"hook_started","hook_name":"SessionStart:startup","session_id":"e1"}` + "\n"
	runHookResponse = `{"type":"system","subtype":"hook_response","output":"delete C:\\Users\\alice\\.claude\\plugin-flag to stop","session_id":"e1"}` + "\n"
	runInit         = `{"type":"system","subtype":"init","cwd":"C:\\Users\\alice\\AppData\\Local\\Temp\\bf-spike-0029\\c42-allow\\tree","session_id":"e1","skills":["my-plugin:my-plugin","debug"],"plugins":[{"name":"my-plugin","path":"C:\\Users\\alice\\.claude\\plugins\\cache\\my-plugin\\my-plugin\\0.1.0","source":"my-plugin@my-plugin"},{"name":"cc-plugin-telemetry","path":"builtin"}],"claude_code_version":"2.1.319"}` + "\n"
	runAssistant    = `{"type":"assistant","message":{"id":"m1"},"timestamp":"2026-11-07T18:23:07.004Z"}` + "\n"
	runRequest      = `{"type":"control_request","request_id":"r1","request":{"subtype":"can_use_tool","tool_name":"Bash","tool_use_id":"t2","agent_id":"a1"}}` + "\n"
	runResult       = `{"type":"result","subtype":"success","stop_reason":"end_turn"}` + "\n"
	runStdinAgent   = `{"transcript_path":"C:\\Users\\alice\\.claude\\projects\\C--Users-alice-AppData-Local-Temp-bf-spike-0029-c42-allow-tree\\e1.jsonl","tool_name":"Agent","tool_use_id":"t1"}`
	runStdinBash    = `{"transcript_path":"C:\\Users\\alice\\.claude\\projects\\C--Users-alice-AppData-Local-Temp-bf-spike-0029-c42-allow-tree\\e1.jsonl","agent_id":"a1","agent_type":"probe","tool_name":"Bash","tool_use_id":"t2"}`
)

// TestImportRunDirectories: a spike run directory and its --turn2 import as two turns: the plugin's
// hook_ lines dropped and named in fixture.json, every other byte kept but for the rewrite, the
// subagent's hook stdin entries as hook-stdin-<n>.json, fixture.json from the streams and driver.log.
func TestImportRunDirectories(t *testing.T) {
	src := runDir(t, "c42-allow", map[string]string{
		"stream.jsonl":     runHookStarted + runHookResponse + runInit + runAssistant + runRequest + runResult,
		"hook-stdin.jsonl": runStdinAgent + "\n" + runStdinBash + "\n",
		"driver.log":       "[0.1s] system/init\n[9.0s] exit code=0 signal=null session=e1\n",
	})
	src2 := runDir(t, "c42-allow-resume", map[string]string{
		"stream.jsonl":     runHookStarted + runInit + runAssistant + runResult,
		"hook-stdin.jsonl": runStdinAgent + "\n",
	})
	dst := filepath.Join(t.TempDir(), "prompt-ask-allow")
	if code := run([]string{"--user", user, "--turn2", src2, src, dst}); code != 0 {
		t.Fatalf("exit %d", code)
	}
	read := func(name string) string {
		b, err := os.ReadFile(filepath.Join(dst, name))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	init := `{"type":"system","subtype":"init","cwd":"/home/user/bf-spike/c42-allow/tree","session_id":"e1","skills":["account-plugin:account-plugin","debug"],"plugins":[{"name":"account-plugin","path":"/home/user/.claude/plugins/cache/account-plugin/account-plugin/0.1.0","source":"account-plugin@account-plugin"},{"name":"cc-plugin-telemetry","path":"builtin"}],"claude_code_version":"2.1.319"}` + "\n"
	if got, want := read("turn-1.jsonl"), init+runAssistant+runRequest+runResult; got != want {
		t.Errorf("turn-1.jsonl:\n got  %s\n want %s", got, want)
	}
	if got, want := read("turn-2.jsonl"), init+runAssistant+runResult; got != want {
		t.Errorf("turn-2.jsonl:\n got  %s\n want %s", got, want)
	}
	wantStdin := "[\n" + `{"transcript_path":"/home/user/.claude/projects/-home-user-bf-spike-c42-allow-tree/e1.jsonl","agent_id":"a1","agent_type":"probe","tool_name":"Bash","tool_use_id":"t2"}` + "\n]\n"
	if got := read("hook-stdin-1.json"); got != wantStdin {
		t.Errorf("hook-stdin-1.json:\n got  %s\n want %s", got, wantStdin)
	}
	if _, err := os.Stat(filepath.Join(dst, "hook-stdin-2.json")); err == nil {
		t.Error("hook-stdin-2.json written for a turn with no subagent call")
	}
	wantFx := `{
  "claim": "C42",
  "engineVersion": "2.1.319",
  "recorded": "2026-11-07",
  "turns": [
    {
      "n": 1,
      "exit": 0,
      "engineSessionId": "e1"
    },
    {
      "n": 2,
      "exit": 0,
      "engineSessionId": "e1"
    }
  ],
  "dropped": [
    "system/hook_started",
    "system/hook_response"
  ]
}
`
	if got := read("fixture.json"); got != wantFx {
		t.Errorf("fixture.json:\n%s\nwant\n%s", got, wantFx)
	}
	left, _ := os.ReadDir(dst)
	if len(left) != 4 {
		t.Errorf("wrote %v, want two turns, hook-stdin-1.json and fixture.json", left)
	}
}

func TestImportRunRefusals(t *testing.T) {
	stream := runInit + runAssistant + runResult
	for name, c := range map[string]struct {
		run   string
		files map[string]string
		turn2 bool
	}{
		"no claim for the run":          {"c99", map[string]string{"stream.jsonl": stream}, false},
		"no init":                       {"c42-allow", map[string]string{"stream.jsonl": runAssistant + runResult}, false},
		"no timestamp":                  {"c42-allow", map[string]string{"stream.jsonl": runInit + runResult}, false},
		"no driver.log and no result":   {"c47-kill", map[string]string{"stream.jsonl": runInit + runAssistant}, false},
		"driver.log without exit":       {"c42-allow", map[string]string{"stream.jsonl": stream, "driver.log": "[1s] user\n"}, false},
		"hook stdin not JSON":           {"c42-allow", map[string]string{"stream.jsonl": stream, "hook-stdin.jsonl": "{x\n"}, false},
		"user survives in hook stdin":   {"c42-allow", map[string]string{"stream.jsonl": stream, "hook-stdin.jsonl": `{"agent_id":"a","p":"/mnt/c/Users/alice/x"}` + "\n"}, false},
		"user survives the hook drop":   {"c42-allow", map[string]string{"stream.jsonl": stream + `{"type":"system","subtype":"x","p":"/mnt/c/Users/alice"}` + "\n"}, false},
		"account plugin name survives":  {"c42-allow", map[string]string{"stream.jsonl": stream + `{"type":"user","text":"my-plugins"}` + "\n"}, false},
		"--turn2 on a turn-file source": {"c42-allow", map[string]string{"turn-1.jsonl": stream, "fixture.json": recordedFx}, true},
	} {
		src := runDir(t, c.run, c.files)
		dst := filepath.Join(t.TempDir(), "out")
		turn2 := ""
		if c.turn2 {
			turn2 = src
		}
		if err := importDir(src, dst, turn2, user, false); err == nil {
			t.Errorf("%s: accepted", name)
		}
		if _, err := os.Stat(dst); err == nil {
			t.Errorf("%s: wrote %s", name, dst)
		}
	}
}

// TestImportRecordedDropsHookLines: a recorded directory's hook_ lines go too, and its
// fixture.json gains dropped.
func TestImportRecordedDropsHookLines(t *testing.T) {
	src := writeFiles(t, map[string]string{"turn-1.jsonl": runHookStarted + "{\"type\":\"result\"}\n", "fixture.json": recordedFx})
	dst := filepath.Join(t.TempDir(), "out")
	if err := importDir(src, dst, "", user, false); err != nil {
		t.Fatal(err)
	}
	turn, _ := os.ReadFile(filepath.Join(dst, "turn-1.jsonl"))
	fx, _ := os.ReadFile(filepath.Join(dst, "fixture.json"))
	if string(turn) != "{\"type\":\"result\"}\n" || !bytes.Contains(fx, []byte(`"dropped": [`)) || !bytes.Contains(fx, []byte(`"system/hook_started"`)) {
		t.Errorf("turn %q, fixture.json %s", turn, fx)
	}
}

func TestDestinationInsideRunSpikeRootRefused(t *testing.T) {
	src := "C:/Users/alice/AppData/Local/Temp/bf-spike-0029/c42-allow"
	for _, dst := range []string{src, "C:/Users/alice/AppData/Local/Temp/bf-spike-0029/c49", "C:/Users/alice/AppData/Local/Temp/bf-spike-0029"} {
		if err := checkDst(src, dst); err == nil {
			t.Errorf("accepted %s", dst)
		}
	}
}
