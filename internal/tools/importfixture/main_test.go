// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"bytes"
	"os"
	"path/filepath"
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
	}
	r := newRewriter(user)
	for _, c := range cases {
		if got := string(r.rewrite([]byte(c.in))); got != c.want {
			t.Errorf("%s:\n in   %s\n got  %s\n want %s", c.name, c.in, got, c.want)
		}
	}
}

func TestRewriteLeavesOtherTextAlone(t *testing.T) {
	for _, in := range []string{
		`C:\\Users\\alicia\\x`,
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
	if err := importDir(src, dst, user, false); err != nil {
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
		"user survives":            {"turn-1.jsonl": "{\"p\":\"/c/Users/alice/x\"}\n", "fixture.json": recordedFx},
	} {
		src := writeFiles(t, files)
		dst := filepath.Join(t.TempDir(), "out")
		if err := importDir(src, dst, user, false); err == nil {
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
	if err := importDir(src, dst, user, false); err == nil {
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
	in := []byte("{\"a\":1}\n{\"p\":\"/c/Users/alice/x\"}\n")
	err := scanLeak(user, map[string][]byte{"turn-1.jsonl": r.rewrite(in)})
	if err == nil {
		t.Fatal("MSYS form passed the scan")
	}
	clean := map[string][]byte{"turn-1.jsonl": r.rewrite([]byte(`"C:\\Users\\alice\\x"`))}
	if err := scanLeak(user, clean); err != nil {
		t.Fatalf("clean output refused: %v", err)
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
