// SPDX-License-Identifier: GPL-3.0-or-later

package cli_test

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/jasondorjeshort/broodforge/internal/cli/clitest"
)

// The P10 half (a) Proof on the built binary: serve on a scratch root, register, one
// policy-mode turn against fixtures/engine/basic through the fake, the reads, stop.

// startServe runs `serve` on root in the background and returns its stdout and its exit.
func startServe(t *testing.T, env []string, root string, args ...string) (*bytes.Buffer, <-chan int) {
	t.Helper()
	stdout, _, exit := startServeBuffers(t, env, root, args...)
	return stdout, exit
}

// startServeBuffers is startServe that also returns stderr, which is safe to read only once the
// exit channel has delivered.
func startServeBuffers(t *testing.T, env []string, root string, args ...string) (*bytes.Buffer, *bytes.Buffer, <-chan int) {
	t.Helper()
	cmd := exec.Command(clitest.Binary(t), append([]string{"--root", root, "serve"}, args...)...)
	cmd.Env = clitest.Environ(env)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	exit := make(chan int, 1)
	go func() {
		err := cmd.Wait()
		code := 0
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		} else if err != nil {
			code = -1
		}
		t.Logf("serve exited %d\nstdout: %q\nstderr: %q", code, stdout.String(), stderr.String())
		exit <- code
	}()
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
	})
	return &stdout, &stderr, exit
}

// TestServePrintsConfigWarnings: a timeout above a year is capped, and serve says so on stderr
// in the form config check uses.
func TestServePrintsConfigWarnings(t *testing.T) {
	_, env := scratchEnv(t)
	root, _ := scratchNew(t, env)
	project := root + "/config/projects/scratch.json"
	good, err := os.ReadFile(project)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(project, []byte(strings.Replace(string(good), `"promptTimeoutMinutes": null`, `"promptTimeoutMinutes": 525601`, 1)), 0o600); err != nil {
		t.Fatal(err)
	}
	_, stderr, exit := startServeBuffers(t, env, root)
	if r := clitest.Run(t, env, "--root", root, "status", "--wait", "10", "--expect", "ok"); r.Code != 0 {
		t.Fatalf("status: %d %q", r.Code, r.Stderr)
	}
	want(t, clitest.Run(t, env, "--root", root, "stop"), 0, "stopped  0\n", "")
	<-exit
	wantLine := "warning: " + project + ": channels.100.promptTimeoutMinutes 525601 is more than a year; capped at 525600\n"
	if !strings.Contains(stderr.String(), wantLine) {
		t.Errorf("serve stderr %q lacks %q", stderr.String(), wantLine)
	}
}

func TestProofServeSession(t *testing.T) {
	_, env := scratchEnv(t)
	root, name := scratchNew(t, env)
	run := func(args ...string) clitest.Result {
		return clitest.Run(t, env, append([]string{"--root", root}, args...)...)
	}

	want(t, run("serve", "--discord", "live"), 1, "",
		"error: start-discord: live needs discord.applicationId, discord.guildId, discord.botCredentialId in config/instance.json\n")
	stdout, exit := startServe(t, env, root)
	if r := run("status", "--wait", "10", "--expect", "ok"); r.Code != 0 || !strings.HasPrefix(r.Stdout, "config: ok\ndiscord: dry-run connected\nsessions: 0 open,") {
		t.Fatalf("status: exit %d %q %q", r.Code, r.Stdout, r.Stderr)
	}
	r := run("serve")
	if r.Code != 1 || !strings.HasPrefix(r.Stderr, "error: start-control: another bot owns ") || !strings.Contains(r.Stderr, name) {
		t.Errorf("a second serve: exit %d %q", r.Code, r.Stderr)
	}

	if r := run("register", "1001", "--name", "Ada Example", "--email", "ada@example.com"); r.Code != 0 ||
		!strings.HasPrefix(r.Stdout, "1001  Ada Example  ada@example.com  ") || !strings.HasSuffix(r.Stdout, "  -\n") {
		t.Errorf("register: %d %q %q", r.Code, r.Stdout, r.Stderr)
	}
	if r := run("register", "list"); r.Code != 0 || !strings.HasPrefix(r.Stdout, "1001  Ada Example  ada@example.com  ") {
		t.Errorf("register list: %q", r.Stdout)
	}
	want(t, run("session", "start", "--channel", "100", "--as", "1001", "--roles", "11", "--credential", "cred-scratch", "--expect", "refused"), 0, "", "")
	want(t, run("session", "start", "--channel", "100", "--as", "1001", "--roles", "10", "--credential", "cred-scratch",
		"--mode", "policy", "--fixture", "basic"), 0, "s0001\n", "")
	want(t, run("session", "send", "s0001", "--as", "1001", "--wait", "Read hello.txt and reply with its one word."), 0,
		"turn-ended  1  completed\n", "")

	tail := run("session", "tail", "s0001").Stdout
	for _, w := range []string{"  tool-call  [Read] /home/user/bf-spike/tree/hello.txt\n",
		"  permission-decision  allow [Read] /home/user/bf-spike/tree/hello.txt (policy)\n", "  text  hello\n", "  turn-ended  end_turn\n"} {
		if !strings.Contains(tail, w) {
			t.Errorf("session tail lacks %q:\n%s", w, tail)
		}
	}
	if r := run("session", "show", "s0001"); !strings.HasPrefix(r.Stdout, "session: s0001  waiting-for-input  100  scratch  1001  policy  code  fake\n") ||
		!strings.Contains(r.Stdout, "\nparticipants: 1001  driver  ") || !strings.Contains(r.Stdout, "\npending: -\n") {
		t.Errorf("session show:\n%s", r.Stdout)
	}
	if r := run("session", "list"); !strings.HasPrefix(r.Stdout, "s0001  waiting-for-input  100  1001  policy  code  -  1  ") {
		t.Errorf("session list (the store through a running bot): %q %q", r.Stdout, r.Stderr)
	}
	if r := run("audit", "--kind", "permission-decision", "--json"); !strings.Contains(r.Stdout, `"decidedBy":"policy","decision":"allow"`) {
		t.Errorf("audit: %q", r.Stdout)
	}
	// --tail streams the second turn (basic's resumed turn) and prints the start line last.
	r = run("session", "send", "s0001", "--as", "1001", "--tail", "Say it again.")
	if r.Code != 0 || !strings.Contains(r.Stdout, "  turn-started  From: Ada Example (driver) Say it again.\n") ||
		!strings.HasSuffix(r.Stdout, "  turn-ended  end_turn\nturn  2  started\n") {
		t.Errorf("send --tail: exit %d\n%s%s", r.Code, r.Stdout, r.Stderr)
	}

	want(t, run("stop"), 0, "stopped  1\n", "")
	select {
	case code := <-exit:
		if code != 0 {
			t.Errorf("serve exited %d", code)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("serve did not exit after stop")
	}
	if !strings.HasPrefix(stdout.String(), "serving  ") {
		t.Errorf("serve printed %q", stdout.String())
	}
	if r := run("status"); r.Code != 1 || !strings.HasPrefix(r.Stderr, "error: not-running: ") {
		t.Errorf("status after stop: %d %q", r.Code, r.Stderr)
	}
	if r := run("session", "tail", "s0001"); r.Code != 1 || !strings.HasPrefix(r.Stderr, "error: not-running: ") {
		t.Errorf("tail without the bot: %d %q", r.Code, r.Stderr)
	}
	if r := run("session", "show", "s0001"); !strings.HasPrefix(r.Stdout, "session: s0001  interrupted  ") {
		t.Errorf("show from the store file: %q %q", r.Stdout, r.Stderr)
	}
	// A second start reconciles nothing new and stops again.
	_, exit = startServe(t, env, root)
	want(t, run("status", "--wait", "10", "--expect", "ok"), 0, "config: ok\ndiscord: dry-run connected\nsessions: 0 open, 0 waiting-for-answer, 1 interrupted\nworktrees: 0 kept with unpushed commits\nbackup: none\nengine: - -\n", "")
	want(t, run("stop"), 0, "stopped  0\n", "")
	if code := <-exit; code != 0 {
		t.Errorf("second serve exited %d", code)
	}
}

// TestProofConfigReload: a valid reload writes configuration-changed; an invalid one keeps the
// running configuration, writes configuration-invalid and exits 1 with the problem.
func TestProofConfigReload(t *testing.T) {
	_, env := scratchEnv(t)
	root, _ := scratchNew(t, env)
	run := func(args ...string) clitest.Result {
		return clitest.Run(t, env, append([]string{"--root", root}, args...)...)
	}
	_, exit := startServe(t, env, root)
	want(t, run("status", "--wait", "10", "--expect", "ok"), 0,
		"config: ok\ndiscord: dry-run connected\nsessions: 0 open, 0 waiting-for-answer, 0 interrupted\nworktrees: 0 kept with unpushed commits\nbackup: none\nengine: - -\n", "")
	want(t, run("config", "reload"), 0, "reloaded  -\n", "")
	project := root + "/config/projects/scratch.json"
	good, err := os.ReadFile(project)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(project, []byte(strings.Replace(string(good), `"handoff": "push"`, `"handoff": "push", "bogus": 1`, 1)), 0o600); err != nil {
		t.Fatal(err)
	}
	if r := run("config", "reload"); r.Code != 1 || !strings.HasPrefix(r.Stderr, "error: bad-config: ") || !strings.Contains(r.Stderr, "bogus") {
		t.Errorf("an invalid reload: %d %q", r.Code, r.Stderr)
	}
	if r := run("audit", "--kind", "configuration-invalid", "--json"); !strings.Contains(r.Stdout, `"using":"previous"`) {
		t.Errorf("no configuration-invalid event: %q", r.Stdout)
	}
	want(t, run("status", "--expect", "ok"), 0,
		"config: ok\ndiscord: dry-run connected\nsessions: 0 open, 0 waiting-for-answer, 0 interrupted\nworktrees: 0 kept with unpushed commits\nbackup: none\nengine: - -\n", "")
	if err := os.WriteFile(project, []byte(strings.Replace(string(good), `"reach": "code"`, `"reach": "read-only"`, 1)), 0o600); err != nil {
		t.Fatal(err)
	}
	if r := run("config", "reload", "--expect", "reloaded"); r.Code != 0 || r.Stdout != "reloaded  projects/scratch.json\n" {
		t.Errorf("reload: %d %q %q", r.Code, r.Stdout, r.Stderr)
	}
	// The first reload above changed nothing: it wrote no event (start, then this reload: 2).
	if r := run("audit", "--kind", "configuration-changed", "--json"); strings.Count(r.Stdout, "\n") != 2 || !strings.Contains(r.Stdout, `"changed":"projects/scratch.json"`) {
		t.Errorf("configuration-changed events: %q", r.Stdout)
	}
	// The reloaded channel 100 is read-only now: a new session reports it.
	if r := run("register", "1001", "--name", "Ada", "--email", "ada@example.com"); r.Code != 0 {
		t.Fatalf("register: %q", r.Stderr)
	}
	if r := run("session", "start", "--channel", "100", "--as", "1001", "--credential", "cred-scratch"); r.Stdout != "s0001\n" {
		t.Fatalf("start: %q %q", r.Stdout, r.Stderr)
	}
	want(t, run("session", "reach-report", "s0001"), 0,
		"Read,Glob,Grep,LS,NotebookRead  allow  deny\nTodoWrite,Agent,Task  allow  allow\nWebFetch,WebSearch  web  web\nEdit,MultiEdit,Write,NotebookEdit  deny  deny\nBash  deny  deny\n*  deny  deny\nbinds: -\n", "")
	want(t, run("stop"), 0, "stopped  1\n", "")
	if code := <-exit; code != 0 {
		t.Errorf("serve exited %d", code)
	}
}

// TestReloadCarriesConfigWarnings: a reload that caps a timeout says so, as lines in text and in
// the warnings field of --json.
func TestReloadCarriesConfigWarnings(t *testing.T) {
	_, env := scratchEnv(t)
	root, _ := scratchNew(t, env)
	run := func(args ...string) clitest.Result {
		return clitest.Run(t, env, append([]string{"--root", root}, args...)...)
	}
	_, exit := startServe(t, env, root)
	if r := run("status", "--wait", "10", "--expect", "ok"); r.Code != 0 {
		t.Fatalf("status: %d %q", r.Code, r.Stderr)
	}
	project := root + "/config/projects/scratch.json"
	good, err := os.ReadFile(project)
	if err != nil {
		t.Fatal(err)
	}
	set := func(minutes string) {
		t.Helper()
		body := strings.Replace(string(good), `"promptTimeoutMinutes": null`, `"promptTimeoutMinutes": `+minutes, 1)
		if err := os.WriteFile(project, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	set("525601")
	want(t, run("config", "reload"), 0, "reloaded  projects/scratch.json\nwarning: "+project+
		": channels.100.promptTimeoutMinutes 525601 is more than a year; capped at 525600\n", "")
	set("525602")
	r := run("--json", "config", "reload")
	if !strings.Contains(r.Stdout, `"warnings":["`+project+`: channels.100.promptTimeoutMinutes 525602 is more than a year; capped at 525600"]`) {
		t.Errorf("reload --json: %q", r.Stdout)
	}
	want(t, run("stop"), 0, "stopped  0\n", "")
	<-exit
}

// TestSessionStartNotesACappedTimeoutOnlyOnSuccess: the note goes to stderr once the start worked.
func TestSessionStartNotesACappedTimeoutOnlyOnSuccess(t *testing.T) {
	_, env := scratchEnv(t)
	root, _ := scratchNew(t, env)
	run := func(args ...string) clitest.Result {
		return clitest.Run(t, env, append([]string{"--root", root}, args...)...)
	}
	_, exit := startServe(t, env, root)
	if r := run("status", "--wait", "10", "--expect", "ok"); r.Code != 0 {
		t.Fatalf("status: %d %q", r.Code, r.Stderr)
	}
	if r := run("register", "1001", "--name", "Ada", "--email", "ada@example.com"); r.Code != 0 {
		t.Fatalf("register: %q", r.Stderr)
	}
	start := func(channel string) clitest.Result {
		return run("session", "start", "--channel", channel, "--as", "1001", "--credential", "cred-scratch", "--idle-timeout-seconds", "31536001")
	}
	if r := start("999"); r.Code == 0 || strings.Contains(r.Stderr, "capped") {
		t.Errorf("a failed start: %d %q", r.Code, r.Stderr)
	}
	want(t, start("100"), 0, "s0001\n", "idle timeout capped at a year\n")
	want(t, run("stop"), 0, "stopped  1\n", "")
	<-exit
}

// TestProofReloadRefusals: config/ naming another instance is refused by the bot, which the
// client still reaches under its last-good name, and status says so; a key read only at start
// is refused reload-needs-restart. Both write configuration-invalid and change nothing.
func TestProofReloadRefusals(t *testing.T) {
	_, env := scratchEnv(t)
	root, name := scratchNew(t, env)
	run := func(args ...string) clitest.Result {
		return clitest.Run(t, env, append([]string{"--root", root}, args...)...)
	}
	_, exit := startServe(t, env, root)
	if r := run("status", "--wait", "10", "--expect", "ok"); r.Code != 0 {
		t.Fatalf("status: %q", r.Stderr)
	}
	inst := root + "/config/instance.json"
	good, err := os.ReadFile(inst)
	if err != nil {
		t.Fatal(err)
	}
	edit := func(from, to string) {
		t.Helper()
		if err := os.WriteFile(inst, []byte(strings.Replace(string(good), from, to, 1)), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	edit(`"instanceName": "`+name+`"`, `"instanceName": "renamed-instance"`)
	if r := run("config", "reload"); r.Code != 1 || !strings.Contains(r.Stderr, "error: bad-config: ") || !strings.Contains(r.Stderr, "instanceName cannot change") {
		t.Errorf("reload of a new instanceName: %d %q", r.Code, r.Stderr)
	}
	if r := run("status"); r.Code != 0 || !strings.HasPrefix(r.Stdout, "config: ok\ninstance-name: renamed-instance (bot runs as "+name+")\n") {
		t.Errorf("status through the last-good name: %d %q %q", r.Code, r.Stdout, r.Stderr)
	}
	edit(`"fakeFixtures": "`, `"fakeFixtures": "/elsewhere`)
	if r := run("config", "reload", "--expect", "reload-needs-restart"); r.Code != 0 {
		t.Errorf("reload of engine.fakeFixtures: %d %q", r.Code, r.Stderr)
	}
	if r := run("audit", "--kind", "configuration-invalid", "--json"); strings.Count(r.Stdout, "\n") != 2 ||
		!strings.Contains(r.Stdout, "instanceName cannot change") || !strings.Contains(r.Stdout, "engine: the bot reads these at its start") {
		t.Errorf("configuration-invalid events: %q", r.Stdout)
	}
	if err := os.WriteFile(inst, good, 0o600); err != nil {
		t.Fatal(err)
	}
	want(t, run("config", "reload"), 0, "reloaded  -\n", "")
	want(t, run("stop"), 0, "stopped  0\n", "")
	if code := <-exit; code != 0 {
		t.Errorf("serve exited %d", code)
	}
}
