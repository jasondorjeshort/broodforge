// SPDX-License-Identifier: GPL-3.0-or-later

package cli_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jasondorjeshort/broodforge/internal/cli/clitest"
)

// TestProofDiscordRender: P11's render Proof on the binary.
func TestProofDiscordRender(t *testing.T) {
	r := clitest.Run(t, nil, "discord", "render", "--events", "../../tests/discord/golden/burst/events.jsonl", "--clock", "fake", "--expect", "edits<=1/s")
	if r.Code != 0 || r.Stderr != "" || !strings.Contains(r.Stdout, "  create  m5  106  <@1002> Permission needed · turn 1\n") ||
		!strings.HasPrefix(lastLine(r.Stdout), "edits<=1/s  ") {
		t.Errorf("render: exit %d\n%s%s", r.Code, r.Stdout, r.Stderr)
	}
	want(t, clitest.Run(t, nil, "discord", "render", "--events", "../../tests/discord/golden/burst/events.jsonl", "--clock", "slow"), 2, "",
		"error: usage: discord render: --clock slow: want fake or real\n")
}

// TestGoldenFlagsExclusive: --golden stands alone (it sets the events file and the fake clock),
// and a render needs --events or --golden; each is the one usage error, exit 2.
func TestGoldenFlagsExclusive(t *testing.T) {
	const usage = "error: usage: discord render: give --events <jsonl> [--clock fake|real], or --golden <dir> [--update] alone\n"
	const dir = "../../tests/discord/golden/burst"
	for _, args := range [][]string{
		{"--golden", dir, "--events", dir + "/events.jsonl"},
		{"--golden", dir, "--clock", "fake"},
		{"--golden", dir, "--clock", "real"},
		{},
		{"--update"},
		{"--clock", "fake"},
	} {
		want(t, clitest.Run(t, nil, append([]string{"discord", "render"}, args...)...), 2, "", usage)
	}
}

// TestGoldenRender: an unchanged golden renders with the pacing line last; one changed character
// in the blessed outbox is golden-differs, exit 1, whatever --expect asked, with the golden and
// now records naming the file and the line; --update re-blesses it.
func TestGoldenRender(t *testing.T) {
	r := clitest.Run(t, nil, "discord", "render", "--golden", "../../tests/discord/golden/burst", "--expect", "edits<=1/s")
	if r.Code != 0 || r.Stderr != "" || !strings.HasPrefix(lastLine(r.Stdout), "edits<=1/s  ") {
		t.Fatalf("render --golden: exit %d\n%s%s", r.Code, r.Stdout, r.Stderr)
	}
	dir := t.TempDir()
	for _, name := range []string{"events.jsonl", "outbox.jsonl", "transcript-s0001.md", "transcript-s0002.md"} {
		b, err := os.ReadFile(filepath.Join("../../tests/discord/golden/burst", name))
		if err != nil {
			t.Fatal(err)
		}
		if name == "outbox.jsonl" {
			b = []byte(strings.Replace(string(b), "Allowed by Sam Example", "Allowed by Sam Exampel", 1))
		}
		if err := os.WriteFile(filepath.Join(dir, name), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	r = clitest.Run(t, nil, "discord", "render", "--golden", dir, "--expect", "edits<=1/s")
	slash := filepath.ToSlash(dir)
	if r.Code != 1 || r.Stderr != "error: golden-differs: "+slash+": outbox.jsonl; --update, then git diff tests/discord/golden/, shows the whole\n" ||
		!strings.Contains(r.Stdout, "\noutbox.jsonl  14  golden  {\"time\":\"2026-10-16T10:00:06.500Z\",\"op\":\"edit\"") ||
		!strings.Contains(r.Stdout, "\noutbox.jsonl  14  now  {\"time\":\"2026-10-16T10:00:06.500Z\",\"op\":\"edit\"") ||
		strings.Contains(r.Stdout, "  more  ") || strings.Contains(r.Stdout, "edits<=1/s  ") {
		t.Errorf("differs: exit %d\n%s%s", r.Code, r.Stdout, r.Stderr)
	}
	r = clitest.Run(t, nil, "--json", "discord", "render", "--golden", dir)
	if r.Code != 1 || !strings.Contains(r.Stdout, "\n{\"golden\":{\"file\":\"outbox.jsonl\",\"line\":14,") ||
		!strings.Contains(r.Stdout, "\n{\"result\":\"edits\\u003c=1/s\",") || !strings.Contains(r.Stderr, `"code":"golden-differs"`) {
		t.Errorf("differs --json: exit %d\n%s%s", r.Code, r.Stdout, r.Stderr)
	}
	want(t, clitest.Run(t, nil, "discord", "render", "--golden", dir, "--update"), 0, "golden  updated  "+slash+"\n", "")
	if r := clitest.Run(t, nil, "discord", "render", "--golden", dir); r.Code != 0 {
		t.Errorf("after --update: exit %d\n%s", r.Code, r.Stderr)
	}
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
	for _, w := range []string{"  channel  s0001  0  #s0001-ada\n", "  create  m2  351  Session s0001 on Scratch, branch ",
		"  Working · turn 1 · 0:00", "  Session s0001 ended by Ada Example · 0 commits · nothing to push; the tree was removed.\n",
		"  attach  m"} {
		if !strings.Contains(out, w) {
			t.Errorf("outbox lacks %q:\n%s", w, out)
		}
	}
	if r := run("--json", "session", "outbox", "s0001", "--all"); !strings.Contains(r.Stdout, `\nIdle · turn 1 · 0:00 · 52.6k in / 86 out · session 52.6k · Haiku · asks first"`) {
		t.Errorf("no footer in:\n%s", r.Stdout)
	}
	if r := run("session", "transcript", "s0001"); !strings.HasPrefix(r.Stdout, "# Session s0001\n") || !strings.Contains(r.Stdout, "**Ada Example (driver)** · ") {
		t.Errorf("transcript:\n%s", r.Stdout)
	}
	want(t, run("stop"), 0, "stopped  0\n", "")
	<-exit
}

// TestProofDiscordModel: P12a's Proof. A start chooses the model, /model answers with the
// header's line, a switch is the driver's, edits the pinned header and reaches the next turn's
// engine request, and a bad value is bad-args and audited.
func TestProofDiscordModel(t *testing.T) {
	_, env := scratchEnv(t)
	root, _ := scratchNew(t, env)
	run := func(args ...string) clitest.Result {
		return clitest.Run(t, env, append([]string{"--root", root}, args...)...)
	}
	_, exit := startServe(t, env, root)
	if r := run("status", "--wait", "10", "--expect", "ok"); r.Code != 0 {
		t.Fatalf("status: %d %q", r.Code, r.Stderr)
	}
	run("discord", "inject", "command", "--channel", "100", "--user", "1001", "/register name:Ada Example email:ada@example.com")
	run("register", "set-default", "1001", "cred-scratch")
	want(t, run("session", "start", "--channel", "100", "--as", "1001", "--roles", "10", "--model", "gpt"), 1, "",
		"error: bad-args: --model \"gpt\": want opus, sonnet, haiku, fable or a model id they name\n")
	want(t, run("session", "start", "--channel", "100", "--as", "1001", "--roles", "10", "--credential", "cred-scratch", "--model", "sonnet"), 0, "s0001\n", "")
	if r := run("session", "show", "s0001"); !strings.Contains(r.Stdout, "\nmodel: claude-sonnet-5-5  effort: -\nturns: 0\n") {
		t.Errorf("session show:\n%s", r.Stdout)
	}
	inject := func(user, text string) clitest.Result {
		return run("discord", "inject", "command", "--channel", "s0001", "--user", user, "--roles", "10", text)
	}
	want(t, inject("1001", "/model"), 0, "ephemeral  Model: Sonnet (chosen at start). Switch with /model opus, sonnet, haiku or fable.\n", "")
	want(t, inject("1001", "/model fable"), 0, "none  model from turn 1: claude-fable-5-1\n", "")
	want(t, inject("1001", "/session effort max"), 0, "none  effort from turn 1: max\n", "")
	want(t, inject("1001", "/model gpt"), 1, "", "error: bad-args: Not a model: gpt. Models: opus, sonnet, haiku, fable.\n")
	want(t, inject("1001", "/session effort turbo"), 1, "", "error: bad-args: Not an effort: turbo. Efforts: low, medium, high, xhigh, max.\n")
	run("discord", "inject", "command", "--channel", "100", "--user", "1002", "/register name:Sam Example email:sam@example.com")
	want(t, run("session", "model", "s0001", "opus", "--as", "1002"), 1, "", "error: refused: model needs the driver, Ada Example\n")
	// In Discord the same refusal is the owner's wording (04-ux-session-controls.md, The header's menus).
	want(t, inject("1002", "/model opus"), 1, "", "error: refused: Only the driver, Ada Example, can change the model.\n")
	want(t, inject("1002", "/session effort high"), 1, "", "error: refused: Only the driver, Ada Example, can change the effort.\n")
	want(t, inject("1002", "/session mode acts-freely"), 1, "", "error: refused: Only the driver, Ada Example, can change the mode.\n")
	if r := run("session", "tail", "s0001"); !strings.Contains(r.Stdout, "  model-changed  From the next turn: Fable.\n") ||
		!strings.Contains(r.Stdout, "  effort-changed  From the next turn: effort max.\n") {
		t.Errorf("tail:\n%s", r.Stdout)
	}
	out := waitOutbox(t, run, "  create  m4  ")
	if !strings.Contains(out, "  edit  m1  ") {
		t.Errorf("no edit of the header m1:\n%s", out)
	}
	if r := run("--json", "session", "outbox", "s0001", "--all"); !strings.Contains(r.Stdout, `\nModel: Fable (from turn 1). Effort: max.\n`) {
		t.Errorf("the header was not edited to the switch:\n%s", r.Stdout)
	}
	if r := run("audit", "--kind", "model-changed"); !strings.Contains(r.Stdout, `{"from":"claude-sonnet-5-5","to":"claude-fable-5-1"}`) {
		t.Errorf("audit model-changed: %q", r.Stdout)
	}
	if r := run("audit", "--kind", "action-refused", "--json"); !strings.Contains(r.Stdout, `"message":"Not a model: gpt. Models: opus, sonnet, haiku, fable.","reason":"bad-args"`) {
		t.Errorf("audit action-refused: %q", r.Stdout)
	}
	want(t, run("discord", "inject", "message", "--channel", "s0001", "--user", "1001", "--roles", "10", "go"), 0, "none  turn 1 started\n", "")
	waitOutbox(t, run, "  delete  ")
	b, err := os.ReadFile(filepath.Join(root, "sessions", "s0001", "fake", "received-1.json"))
	var req struct{ Model, Effort string }
	if err != nil || json.Unmarshal(b, &req) != nil || req.Model != "claude-fable-5-1" || req.Effort != "max" {
		t.Errorf("received-1.json: %s %v", b, err)
	}
	if r := run("session", "list"); !strings.Contains(r.Stdout, "  code  claude-fable-5-1  1  ") {
		t.Errorf("session list %q", r.Stdout)
	}
	// /session start model:<alias> in the project channel.
	want(t, run("discord", "inject", "command", "--channel", "100", "--user", "1001", "--roles", "10", "/session start model:haiku"), 0,
		"ephemeral  Session s0002 is open in <#s0002>.\n", "")
	if r := run("session", "show", "s0002"); !strings.Contains(r.Stdout, "\nmodel: claude-haiku-5-5  effort: -\n") {
		t.Errorf("session show s0002:\n%s", r.Stdout)
	}
	want(t, run("stop"), 0, "stopped  2\n", "")
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

// TestPanelSurvivesRestart: `panel post` records the panel in the store, so after a restart
// `panel show` prints the same message id and `panel post` edits it in the outbox instead of
// creating a second (03, Configuration; P11 remainder, item 8).
func TestPanelSurvivesRestart(t *testing.T) {
	_, env := scratchEnv(t)
	root, _ := scratchNew(t, env)
	run := func(args ...string) clitest.Result {
		return clitest.Run(t, env, append([]string{"--root", root}, args...)...)
	}
	serve := func() <-chan int {
		_, exit := startServe(t, env, root)
		if r := run("status", "--wait", "10", "--expect", "ok"); r.Code != 0 {
			t.Fatalf("status: %d %q", r.Code, r.Stderr)
		}
		return exit
	}
	exit := serve()
	want(t, run("panel", "show", "100"), 1, "", "error: no-panel: no panel posted in 100\n")
	posted := run("panel", "post", "100")
	id, _, _ := strings.Cut(strings.TrimPrefix(posted.Stdout, "100  "), "  ")
	if posted.Code != 0 || !strings.HasPrefix(id, "m") {
		t.Fatalf("panel post: %d %q %q", posted.Code, posted.Stdout, posted.Stderr)
	}
	want(t, run("stop"), 0, "stopped  0\n", "")
	<-exit

	exit = serve()
	want(t, run("panel", "show", "100"), 0, posted.Stdout, "")
	if r := run("panel", "post", "100"); r.Code != 0 || !strings.HasPrefix(r.Stdout, "100  "+id+"  ") {
		t.Errorf("panel post after the restart: %d %q %q", r.Code, r.Stdout, r.Stderr)
	}
	deadline := time.Now().Add(20 * time.Second)
	out := ""
	for !strings.Contains(out, "  edit  "+id+"  ") && time.Now().Before(deadline) {
		time.Sleep(200 * time.Millisecond)
		out = run("session", "outbox", "-", "--channel", "100").Stdout
	}
	if !strings.Contains(out, "  edit  "+id+"  ") || strings.Contains(out, "  create  ") {
		t.Errorf("the panel after the restart is not one edit of %s:\n%s", id, out)
	}
	want(t, run("stop"), 0, "stopped  0\n", "")
	<-exit
}
