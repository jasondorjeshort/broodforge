// SPDX-License-Identifier: GPL-3.0-or-later

package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jasondorjeshort/broodforge/internal/control/controltest"
)

func run(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := Run(args, strings.NewReader(""), &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func check(t *testing.T, args []string, wantCode int, wantStdout, wantStderr string) {
	t.Helper()
	code, stdout, stderr := run(t, args...)
	if code != wantCode || stdout != wantStdout || stderr != wantStderr {
		t.Errorf("broodforge %q:\n exit %d, want %d\n stdout %q, want %q\n stderr %q, want %q",
			args, code, wantCode, stdout, wantStdout, stderr, wantStderr)
	}
}

func TestNoArgsIsUsage(t *testing.T) {
	code, stdout, stderr := run(t)
	if code != ExitUsage || stdout != "" || !strings.HasPrefix(stderr, "usage: broodforge ") {
		t.Fatalf("exit %d stdout %q stderr %q", code, stdout, stderr)
	}
	if strings.Contains(stderr, "control echo") {
		t.Errorf("usage lists the hidden control echo: %q", stderr)
	}
}

func TestHelp(t *testing.T) {
	code, stdout, _ := run(t, "--help")
	if code != ExitOK || !strings.HasPrefix(stdout, "usage: broodforge ") {
		t.Errorf("--help: exit %d stdout %q", code, stdout)
	}
	check(t, []string{"control", "echo", "-h"}, ExitOK,
		"usage: broodforge control echo [--text <s>] [--events <n>] [--fail <code>] [--expect <value>]\n", "")
}

func TestUsageErrors(t *testing.T) {
	t.Setenv(RootEnv, "")
	check(t, []string{"--root", "x", "nope"}, ExitUsage, "", "error: usage: unknown command \"nope\"\n")
	check(t, []string{"--root", "x", "control", "nope"}, ExitUsage, "", "error: usage: unknown command \"control nope\"\n")
	check(t, []string{"--bogus"}, ExitUsage, "", "error: usage: flag provided but not defined: -bogus\n")
	check(t, []string{"--root", "x", "control", "echo", "--bogus"}, ExitUsage, "",
		"error: usage: control echo: flag provided but not defined: -bogus\n")
	check(t, []string{"--root", "x", "control", "echo", "--events", "x"}, ExitUsage, "",
		"error: usage: control echo: invalid value \"x\" for flag -events: parse error\n")
	check(t, []string{"--root", "x", "control", "echo", "extra"}, ExitUsage, "",
		"error: usage: control echo: unexpected argument \"extra\"\n")
	check(t, []string{"control", "echo"}, ExitUsage, "",
		"error: usage: no instance root: pass --root <path> or set BROODFORGE_ROOT\n")
	check(t, []string{"--json", "control", "echo"}, ExitUsage, "",
		`{"code":"usage","message":"no instance root: pass --root \u003cpath\u003e or set BROODFORGE_ROOT"}`+"\n")
}

func TestEchoInProcess(t *testing.T) {
	h := controltest.Start(t)
	check(t, []string{"--root", h.Root, "control", "echo"}, ExitOK, "ok\n", "")
	check(t, []string{"control", "echo", "--root", h.Root, "--json", "--text", "hi there"}, ExitOK, `{"text":"hi there"}`+"\n", "")
	check(t, []string{"--root", h.Root, "control", "echo", "--events", "2", "--text", "a\tb"}, ExitOK, "a\\tb  1\na\\tb  2\na\\tb\n", "")
	check(t, []string{"--root", h.Root, "--json", "control", "echo", "--events", "2"}, ExitOK,
		`{"seq":1,"text":"ok"}`+"\n"+`{"seq":2,"text":"ok"}`+"\n"+`{"text":"ok"}`+"\n", "")
	check(t, []string{"--root", h.Root, "control", "echo", "--text", ""}, ExitOK, "\"\"\n", "")
	t.Setenv(RootEnv, h.Root)
	check(t, []string{"control", "echo", "--expect", "ok"}, ExitOK, "ok\n", "")
}

func TestExpect(t *testing.T) {
	h := controltest.Start(t)
	r := []string{"--root", h.Root, "control", "echo"}
	check(t, append(r, "--expect", "ok"), ExitOK, "ok\n", "")
	check(t, append(r, "--expect", "nope"), ExitFailure, "ok\n", "error: expect: wanted nope, got ok\n")
	check(t, append(r, "--events", "1", "--text", "x", "--expect", "x"), ExitOK, "x  1\nx\n", "")
	// A refusal: the outcome token is the error code; a match exits 0.
	check(t, append(r, "--fail", "busy"), ExitFailure, "", "error: busy: echo was asked to fail\n")
	check(t, append(r, "--fail", "busy", "--expect", "busy"), ExitOK, "", "")
	check(t, append(r, "--fail", "busy", "--expect", "ok"), ExitFailure, "", "error: expect: wanted ok, got busy\n")
	check(t, []string{"--json", "--root", h.Root, "control", "echo", "--fail", "busy"}, ExitFailure, "",
		`{"code":"busy","message":"echo was asked to fail"}`+"\n")
	check(t, append(r, "--fail", "Bad:Code"), ExitFailure, "",
		"error: bad-args: fail must be one kebab-case code, got \"Bad:Code\"\n")
}

func TestNotRunningAndConfigErrors(t *testing.T) {
	root, _, addr := controltest.Root(t)
	check(t, []string{"--root", root, "control", "echo"}, ExitFailure, "", "error: not-running: no bot at "+addr+"\n")
	check(t, []string{"--root", root, "control", "echo", "--expect", "not-running"}, ExitOK, "", "")

	empty := t.TempDir()
	code, _, stderr := run(t, "--root", empty, "control", "echo")
	if code != ExitFailure || !strings.HasPrefix(stderr, "error: no-config: ") || strings.Count(stderr, "\n") != 1 {
		t.Errorf("no config: exit %d stderr %q", code, stderr)
	}
	write := func(body string) string {
		dir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dir, "config"), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "config", "instance.json"), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		return dir
	}
	for _, body := range []string{`{}`, `{"instanceName":"a/b"}`, `not json`} {
		dir := write(body)
		code, _, stderr := run(t, "--root", dir, "control", "echo")
		if code != ExitFailure || !strings.HasPrefix(stderr, "error: bad-config: "+instancePath(dir)+": ") {
			t.Errorf("%s: exit %d stderr %q", body, code, stderr)
		}
	}
}

func TestWireArgsAndCamel(t *testing.T) {
	for in, want := range map[string]string{"events": "events", "dry-run": "dryRun", "older-than-days": "olderThanDays"} {
		if got := camel(in); got != want {
			t.Errorf("camel(%q) = %q, want %q", in, got, want)
		}
	}
	env := &Env{values: map[string]*flagValue{}}
	inv := &invocation{}
	fs, err := inv.flagSet(&controlEcho, env)
	if err != nil {
		t.Fatal(err)
	}
	if err := fs.Parse([]string{"--events", "3"}); err != nil {
		t.Fatal(err)
	}
	got, err := controlEcho.wireArgs(env)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != `{"events":3,"fail":"","text":"ok"}` {
		t.Errorf("wireArgs = %s", got)
	}
	if controlEcho.wireCmd() != "control.echo" {
		t.Errorf("wireCmd = %s", controlEcho.wireCmd())
	}
}

func TestCommandTable(t *testing.T) {
	seen := map[string]bool{}
	for _, c := range commands {
		if seen[c.Name] {
			t.Errorf("command %q twice", c.Name)
		}
		seen[c.Name] = true
		if (c.Class == ClassFile || c.Class == ClassStandalone) && c.Local == nil {
			t.Errorf("%s: class %d needs Local", c.Name, c.Class)
		}
		for _, f := range c.Flags {
			if f.Name == "root" || f.Name == "json" || f.Name == "expect" || f.Name == "limit" || f.Name == "all" {
				t.Errorf("%s: flag --%s is shared, not the command's", c.Name, f.Name)
			}
		}
	}
}
