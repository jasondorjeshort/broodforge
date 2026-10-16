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
