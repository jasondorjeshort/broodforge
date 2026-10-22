// SPDX-License-Identifier: GPL-3.0-or-later

package cli_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jasondorjeshort/broodforge/internal/cli/clitest"
)

// P12 without a guild: the bot token's path into the credential store, then `discord check`
// and `serve --discord live` refusing cleanly before anything connects, the token unprinted.
func TestProofBotTokenAndLiveRefusals(t *testing.T) {
	_, env := scratchEnv(t)
	root, _ := scratchNew(t, env)
	run := func(args ...string) clitest.Result {
		return clitest.Run(t, env, append([]string{"--root", root}, args...)...)
	}
	_, exit := startServe(t, env, root)
	if r := run("status", "--wait", "10", "--expect", "ok"); r.Code != 0 {
		t.Fatalf("status: %d %q %q", r.Code, r.Stdout, r.Stderr)
	}

	const token = "placeholder-token-value"
	r := runStdin(t, env, token+"\n", "--root", root, "credential", "add", "--instance", "--kind", "bot-token", "--label", "bot")
	id := strings.TrimSuffix(r.Stdout, "\n")
	if r.Code != 0 || r.Stderr != "" || !credID.MatchString(id) {
		t.Fatalf("add --instance: exit %d stdout %q stderr %q", r.Code, r.Stdout, r.Stderr)
	}
	if r := run("credential", "list"); r.Code != 0 || !strings.Contains(r.Stdout, id+"  bot-token  -  bot  usable  -\n") {
		t.Errorf("list: %q", r.Stdout)
	}
	r = run("credential", "show", id, "--json")
	var rec struct {
		User     *string
		Provider string
	}
	if r.Code != 0 || strings.Contains(r.Stdout, token) || json.Unmarshal([]byte(r.Stdout), &rec) != nil || rec.User != nil || rec.Provider != "discord" {
		t.Errorf("show --json: %d %q", r.Code, r.Stdout)
	}
	want(t, run("credential", "login", id), 1, "", "error: bad-kind: "+id+" is the instance's bot-token credential; it has no login\n")
	want(t, run("credential", "login", id, "--relay"), 1, "", "error: not-built: credential login --relay: the direct-message login relay (ADR 0022) is not built yet; run credential login "+id+" and finish the login on the host\n")
	want(t, runStdin(t, env, token, "--root", root, "credential", "add", "--instance", "--user", "1001", "--kind", "bot-token", "--label", "b"), 2, "",
		"error: usage: credential add: --instance and --user are exclusive\n")
	want(t, runStdin(t, env, token, "--root", root, "credential", "add", "--user", "1001", "--kind", "bot-token", "--label", "b"), 1, "",
		"error: bad-kind: a bot-token credential belongs to the instance: add it with --instance, not --user\n")
	want(t, runStdin(t, env, "", "--root", root, "credential", "add", "--instance", "--kind", "bot-token", "--label", "b"), 2, "",
		"error: usage: credential add: --kind bot-token reads its value from stdin, 1 to 65536 bytes\n")
	if r := run("stop"); r.Code != 0 {
		t.Fatalf("stop: %d %q", r.Code, r.Stderr)
	}
	<-exit

	// The ids the live mode needs, the bot token among them.
	path := filepath.Join(root, "config", "instance.json")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var inst map[string]json.RawMessage
	if err := json.Unmarshal(b, &inst); err != nil {
		t.Fatal(err)
	}
	inst["discord"] = json.RawMessage(`{"mode":"dry-run","applicationId":"4000","guildId":"3000","botCredentialId":"` + id + `","noticeChannelId":"300"}`)
	if b, err = json.Marshal(inst); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
	if r := run("config", "check"); r.Code != 0 || !strings.HasPrefix(r.Stdout, "ok  ") {
		t.Errorf("config check: %d %q %q", r.Code, r.Stdout, r.Stderr)
	}

	refusal := "error: start-discord: the bot-token credential does not hold a Discord bot token (want three parts joined by dots, the first the bot's id in base64)\n"
	want(t, run("discord", "check", "--guild", "1"), 1, "", refusal)
	want(t, run("serve", "--discord", "live"), 1, "", refusal)
	if r := run("audit", "--limit", "100"); strings.Contains(r.Stdout, token) {
		t.Error("the audit log holds the token")
	}
}
