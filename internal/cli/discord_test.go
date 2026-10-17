// SPDX-License-Identifier: GPL-3.0-or-later

package cli_test

import (
	"strings"
	"testing"
	"time"

	"github.com/jasondorjeshort/broodforge/internal/cli/clitest"
)

// TestProofDiscordRender: P11's render Proof on the binary.
func TestProofDiscordRender(t *testing.T) {
	r := clitest.Run(t, nil, "discord", "render", "--events", "../../tests/discord/burst.jsonl", "--clock", "fake", "--expect", "edits<=1/s")
	if r.Code != 0 || r.Stderr != "" || !strings.Contains(r.Stdout, "  create  m5  106  <@1002> Permission needed · turn 1\n") ||
		!strings.HasPrefix(lastLine(r.Stdout), "edits<=1/s  ") {
		t.Errorf("render: exit %d\n%s%s", r.Code, r.Stdout, r.Stderr)
	}
	want(t, clitest.Run(t, nil, "discord", "render", "--events", "../../tests/discord/burst.jsonl", "--clock", "slow"), 2, "",
		"error: usage: discord render: --clock slow: want fake or real\n")
}

func lastLine(s string) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	return lines[len(lines)-1]
}

// TestProofDiscordInject: P11's inject Proof, and a session through the adapter: a pasted key
// is refused and audited without its text, the lobby hint, the panel, a start by its button,
// a venue message, and the outbox's working message, footer and summary.
func TestProofDiscordInject(t *testing.T) {
	_, env := scratchEnv(t)
	root, _ := scratchNew(t, env)
	run := func(args ...string) clitest.Result {
		return clitest.Run(t, env, append([]string{"--root", root}, args...)...)
	}
	want(t, run("discord", "inject", "message", "--channel", "100", "--user", "1001", "hi"), 1, "",
		"error: not-running: no bot at "+socketOf(t, run)+"\n")
	_, exit := startServe(t, env, root)
	if r := run("status", "--wait", "10", "--expect", "ok"); r.Code != 0 {
		t.Fatalf("status: %d %q", r.Code, r.Stderr)
	}
	want(t, run("discord", "inject", "command", "--channel", "100", "--user", "1001", "/register name:Ada Example email:ada@example.com"), 0,
		`ephemeral  Registered. Commits you drive are authored as Ada Example <ada@example.com>.\nNext: add a credential with /credential add, then start a session from a project channel.`+"\n", "")
	run("register", "set-default", "1001", "cred-scratch")

	key := "sk-ant-api03-NOTAREALKEYNOTAREALKEY"
	want(t, run("discord", "inject", "message", "--channel", "100", "--user", "1001", "--roles", "10", key), 1, "",
		"error: key-pasted: Never paste a key in chat. Use /credential add kind:api-key, which opens a private form.\n")
	want(t, run("discord", "inject", "message", "--channel", "100", "--user", "1001", "--roles", "10", "sk"+string(rune(0x200b))+"-ant-api03-HIDDENHIDDEN"), 1, "",
		"error: key-pasted: Never paste a key in chat. Use /credential add kind:api-key, which opens a private form.\n")
	want(t, run("discord", "inject", "message", "--channel", "100", "--user", "1001", "--roles", "10", "the task-ant-colony test fails"), 0,
		"ephemeral  Start a session with the button above or /session start; this channel is the lobby.\n", "")
	want(t, run("discord", "inject", "command", "--channel", "100", "--user", "1001", "/frobnicate now"), 1, "",
		"error: bad-args: no command /frobnicate\n")
	want(t, run("discord", "inject", "click", "--custom-id", "bf:100:start", "--user", "1001", "--roles", "11"), 1, "",
		"error: refused: Your roles do not allow starting a session in <#100>. Ask a server admin.\n")
	want(t, run("discord", "inject", "click", "--custom-id", "bf:100:nope", "--user", "1001"), 1, "",
		"error: bad-args: a custom id of 11 bytes starting \"bf\" is not a Broodforge custom id\n")
	if r := run("audit", "--kind", "action-refused", "--json"); !strings.Contains(r.Stdout, `"reason":"key-pasted"`) ||
		strings.Contains(r.Stdout, "NOTAREAL") || strings.Contains(r.Stdout, "HIDDEN") || strings.Contains(r.Stdout, "nope") ||
		strings.Count(r.Stdout, `"reason":"bad-args"`) != 2 {
		t.Errorf("audit: %q", r.Stdout)
	}
	if r := run("panel", "post", "100"); r.Code != 0 || !strings.HasPrefix(r.Stdout, "100  m") {
		t.Errorf("panel post: %d %q %q", r.Code, r.Stdout, r.Stderr)
	}
	if r := run("panel", "show", "100"); r.Code != 0 || !strings.HasPrefix(r.Stdout, "100  m") {
		t.Errorf("panel show: %d %q %q", r.Code, r.Stdout, r.Stderr)
	}
	if r := run("session", "outbox", "-", "--channel", "100"); !strings.Contains(r.Stdout, "  create  m1  ") ||
		!strings.Contains(r.Stdout, "  Work on Scratch with Claude Code: one session is one branch; ending it pushes the branch. May start: Trusted.\n") {
		t.Errorf("panel in the outbox: %q %q", r.Stdout, r.Stderr)
	}
	want(t, run("discord", "inject", "click", "--custom-id", "bf:100:start", "--user", "1001", "--roles", "10"), 0,
		"ephemeral  Session s0001 is open in <#s0001>.\n", "")
	// A message starting with a path is text for the agent, not a command.
	want(t, run("discord", "inject", "message", "--channel", "s0001", "--user", "1001", "--roles", "10", "/tmp/x fails"), 0,
		"none  turn 1 started\n", "")
	if tail := run("session", "tail", "s0001").Stdout; !strings.Contains(tail, "  turn-started  From: Ada Example (driver) /tmp/x fails\n") {
		t.Errorf("tail:\n%s", tail)
	}
	waitOutbox(t, run, "  delete  ")
	want(t, run("discord", "inject", "click", "--custom-id", "bf:s0001:end", "--user", "1001", "--roles", "10"), 0, "none  s0001 ended\n", "")
	out := waitOutbox(t, run, "  attach  ")
	for _, w := range []string{"  channel  s0001  0  #s0001-ada\n", "  create  m2  339  Session s0001 on Scratch, branch ",
		"  Working · turn 1 · 0:00\n", "  Session s0001 ended by Ada Example · 0 commits · nothing to push; the tree was removed.\n",
		"  attach  m"} {
		if !strings.Contains(out, w) {
			t.Errorf("outbox lacks %q:\n%s", w, out)
		}
	}
	if r := run("--json", "session", "outbox", "s0001", "--all"); !strings.Contains(r.Stdout, `\nDone · turn 1 · 0:00 · 52.6k in / 86 out · session 52.6k · Haiku · asks first"`) {
		t.Errorf("no footer in:\n%s", r.Stdout)
	}
	if r := run("session", "transcript", "s0001"); !strings.HasPrefix(r.Stdout, "# Session s0001\n") || !strings.Contains(r.Stdout, "**Ada Example (driver)** · ") {
		t.Errorf("transcript:\n%s", r.Stdout)
	}
	want(t, run("stop"), 0, "stopped  0\n", "")
	<-exit
}

func waitOutbox(t *testing.T, run func(...string) clitest.Result, line string) string {
	t.Helper()
	deadline := time.Now().Add(20 * time.Second)
	for {
		out := run("session", "outbox", "s0001", "--all").Stdout
		if strings.Contains(out, line) || time.Now().After(deadline) {
			return out
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func socketOf(t *testing.T, run func(...string) clitest.Result) string {
	t.Helper()
	r := run("status")
	return strings.TrimSuffix(strings.TrimPrefix(r.Stderr, "error: not-running: no bot at "), "\n")
}
