// SPDX-License-Identifier: GPL-3.0-or-later

package setup_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/jasondorjeshort/broodforge/internal/config"
	"github.com/jasondorjeshort/broodforge/internal/discord"
	"github.com/jasondorjeshort/broodforge/internal/setup"
)

// The Discord steps against the scripted Discord (08-implementation-plan-setup.md, P22): a fresh
// guild the bot joins through the invite, a guild with channels, the role present or made by the
// owner, the PATCH refused, a role landing above the bot.

const (
	tokA    = "NTAwMA.abcd.abcdefghijklmnopqrstuvwxyz" // the bot 5000's token the fake accepts
	tokB    = "NTAwMA.efgh.zyxwvutsrqponmlkjihgfedcba" // a reset one
	app     = "4000"
	bot     = "5000"
	guildID = "3000"
	owner   = "6000"
	botRole = "5001"
)

// scriptedGuild is a guild of the owner's with the bot's role holding the invite's permissions.
func scriptedGuild(id, name string, manageRoles bool) discord.FakeGuild {
	perms := discord.PermissionBits
	if manageRoles {
		perms |= discord.PermManageRoles
	}
	return discord.FakeGuild{ID: id, Name: name, OwnerID: owner,
		Roles:    []discord.FakeRole{{ID: id, Name: "@everyone"}, {ID: botRole, Name: "Broodforge", Position: 2, Permissions: perms}},
		BotRoles: []string{botRole}, DropManageRolesAfter: 1}
}

// fresh is an application whose bot is in no server yet: the invite joins on the poll after the
// link, with Manage Roles, which the owner removes on the poll after the removal fix.
func fresh() discord.FakeScript {
	g := scriptedGuild(guildID, "Test", true)
	return discord.FakeScript{Token: tokA, Application: discord.FakeApp{ID: app, BotID: bot}, Invite: &g, JoinAfter: 1}
}

func withFake(f *fixture, s discord.FakeScript) *fixture {
	f.fake = discord.NewFakeSetup(s)
	return f
}

// edit replaces the fake's world with fn's change to it, as the owner's hand in the server.
func (f *fixture) edit(fn func(s *discord.FakeScript)) {
	s := f.fake.Script()
	fn(&s)
	f.fake = discord.NewFakeSetup(s)
}

func line(out, prefix string) string {
	for _, l := range strings.Split(out, "\n") {
		if strings.HasPrefix(l, prefix) {
			return l
		}
	}
	return ""
}

func calls(f *fixture, prefix string) int {
	n := 0
	for _, c := range f.fake.Script().Calls {
		if strings.HasPrefix(c, prefix) {
			n++
		}
	}
	return n
}

func TestDiscordSteps(t *testing.T) {
	f := withFake(newFixture(t), fresh())
	out, c := f.run(tokA + "\n")
	if c != "" {
		t.Fatalf("setup refused %s:\n%s", c, out)
	}
	want := []string{"ok root", "ok config", "ok key", "ok engine", "ok token", "ok application", "fix guild", "ok guild",
		"ok channels", "fix role", "ok role", "skip git-credential", "ok project", "ok defaults"}
	if got := results(out); strings.Join(got[:len(want)], ",") != strings.Join(want, ",") {
		t.Fatalf("lines:\n%s\nwant %v", out, want)
	}
	if l := line(out, "fix  guild  "); l != "fix  guild  Open this link, pick the server, Authorize: https://discord.com/oauth2/authorize?client_id=4000&scope=bot+applications.commands&permissions=2251800082246736" {
		t.Errorf("invite line %q", l)
	}
	if l := line(out, "fix  role  "); l != "fix  role  remove Manage Roles from the bot's role in Server Settings, Roles; setup is waiting" {
		t.Errorf("removal line %q", l)
	}
	for _, w := range []string{"ok  application  4000  message content on", "ok  guild  3000  Test", "ok  channels  Sessions 9001, #demo 9002, #demo-notices 9003",
		"ok  role  Trusted  9004  held by 6000; discord.mode live", "ok  check  discord check  guild  3000  Test", "ok  check  discord check  9002  present  code  perms:ok",
		"ok  check  discord check  message-content  present  intent  perms:ok"} {
		if !strings.Contains(out, w+"\n") {
			t.Errorf("no %q:\n%s", w, out)
		}
	}
	if strings.Contains(out, tokA) {
		t.Error("the token is printed")
	}
	cfg, err := config.Load(f.root, config.AnyCredential)
	if err != nil {
		t.Fatalf("config after setup: %v", err)
	}
	d := cfg.Instance.Discord
	if d.Mode != config.DiscordLive || d.ApplicationID != app || d.GuildID != guildID || d.NoticeChannelID != "9003" ||
		!strings.HasPrefix(d.BotCredentialID, "cred-") || strings.Join(cfg.Instance.RoleGroups["trusted"], ",") != "9004" {
		t.Errorf("discord config %+v, trusted %v", d, cfg.Instance.RoleGroups["trusted"])
	}
	p := cfg.Projects[0]
	if p.Venue.Category != "9001" || p.Channels["9002"].Reach != config.ReachCode || strings.Join(p.Channels["9002"].Roles.Start, ",") != "trusted" {
		t.Errorf("project %+v", p)
	}
	s := f.fake.Script()
	if strings.Join(s.Calls, ",") != "UpdateCurrentApplication flags=524288,CreateGuildChannel Sessions,CreateGuildChannel demo,"+
		"CreateGuildChannel demo-notices,CreateRole Trusted,AddMemberRole 6000 9004" {
		t.Errorf("writes to Discord: %v", s.Calls)
	}
	if s.Guilds[0].Channels[1].Parent != "9001" || s.Guilds[0].Channels[2].Parent != "9001" {
		t.Errorf("the channels are not under the category: %+v", s.Guilds[0].Channels)
	}

	// The rerun: every Discord step skip, nothing written to Discord or config/.
	before := snapshot(t, f.root)
	out, c = f.run("")
	if c != "" {
		t.Fatalf("rerun refused %s:\n%s", c, out)
	}
	for _, st := range []string{"token", "application", "guild", "channels", "role"} {
		if !strings.Contains(out, "skip  "+st+"  ") {
			t.Errorf("rerun: %s is not skip:\n%s", st, out)
		}
	}
	if after := snapshot(t, f.root); after != before {
		t.Error("the rerun changed config/")
	}
	if n := len(f.fake.Script().Calls); n != len(s.Calls) {
		t.Errorf("the rerun wrote to Discord: %v", f.fake.Script().Calls[len(s.Calls):])
	}

	t.Run("stored token refused", func(t *testing.T) {
		old := cfg.Instance.Discord.BotCredentialID
		f.edit(func(s *discord.FakeScript) { s.Token = tokB })
		out, c := f.run(tokB + "\n")
		if c != "" {
			t.Fatalf("refused %s:\n%s", c, out)
		}
		if l := line(out, "fix  token  "); l != "fix  token  Discord refused the stored token: 401: Unauthorized" {
			t.Errorf("refusal line %q", l)
		}
		l := line(out, "ok  token  ")
		cfg, _ := config.Load(f.root, config.AnyCredential)
		now := cfg.Instance.Discord.BotCredentialID
		if now == old || l != "ok  token  "+now+"; "+old+" removed" {
			t.Errorf("token line %q, id %s was %s", l, now, old)
		}
	})

	t.Run("live root refuses the scripted Discord", func(t *testing.T) {
		var out syncBuffer
		env := f.env(strings.NewReader(""), &out)
		env.FakeDiscord = true
		if c := code(setup.Run(context.Background(), f.opts(), env)); c != setup.CodeLiveInstance {
			t.Errorf("code %q:\n%s", c, out.String())
		}
	})
}

func TestDiscordStepsRefusals(t *testing.T) {
	t.Run("no token on stdin", func(t *testing.T) {
		f := withFake(newFixture(t), fresh())
		out, c := f.run("")
		if c != "needs-input-token" || !strings.Contains(out, "fix  token  no bot token yet: in the Developer Portal (https://discord.com/developers/applications), New Application, then Bot, Reset Token") {
			t.Errorf("code %q:\n%s", c, out)
		}
	})
	t.Run("a token Discord refuses", func(t *testing.T) {
		f := withFake(newFixture(t), fresh())
		out, c := f.run(tokB + "\n")
		if c != "refused-token" || line(out, "fix  token  ") != "fix  token  Discord refused the token: 401: Unauthorized" {
			t.Errorf("code %q:\n%s", c, out)
		}
	})
	t.Run("message content already on", func(t *testing.T) {
		s := fresh()
		s.Application.Flags = discord.FlagMessageContent
		f := withFake(newFixture(t), s)
		if out, c := f.run(tokA + "\n"); c != "" {
			t.Fatalf("refused %s:\n%s", c, out)
		}
		if n := calls(f, "UpdateCurrentApplication"); n != 0 {
			t.Errorf("%d PATCH calls with the flag set", n)
		}
	})
	t.Run("PATCH refused", func(t *testing.T) {
		s := fresh()
		s.Application.RefusePatch = true
		f := withFake(newFixture(t), s)
		out, c := f.run(tokA + "\n")
		if c != "timeout-application" || !strings.Contains(out, "fix  application  Discord refused the intent switch (20012: You are not authorized to perform this action on this application); in the Developer Portal, Bot, switch on Message Content Intent; setup is waiting\n") {
			t.Errorf("code %q:\n%s", c, out)
		}
	})
	t.Run("several guilds", func(t *testing.T) {
		s := fresh()
		s.Invite = nil
		s.Guilds = []discord.FakeGuild{scriptedGuild(guildID, "Test", true), scriptedGuild("3100", "Other", true)}
		f := withFake(newFixture(t), s)
		out, c := f.run(tokA + "\n")
		if c != "needs-input-guild" {
			t.Fatalf("code %q:\n%s", c, out)
		}
		out, c = f.run("", func(o *setup.Options) { o.Guild = "3100" })
		if c != "" || !strings.Contains(out, "ok  guild  3100  Other\n") {
			t.Errorf("--guild 3100: code %q:\n%s", c, out)
		}
	})
}

func TestChannels(t *testing.T) {
	t.Run("renamed channel is skip, deleted is mismatch", func(t *testing.T) {
		f := withFake(newFixture(t), fresh())
		if out, c := f.run(tokA + "\n"); c != "" {
			t.Fatalf("refused %s:\n%s", c, out)
		}
		f.edit(func(s *discord.FakeScript) { s.Guilds[0].Channels[0].Name = "Work" })
		out, c := f.run("")
		if c != "" || !strings.Contains(out, "skip  channels  Sessions 9001, #demo 9002, #demo-notices 9003\n") {
			t.Errorf("renamed: code %q:\n%s", c, out)
		}
		f.edit(func(s *discord.FakeScript) { s.Guilds[0].Channels = s.Guilds[0].Channels[:2] })
		out, c = f.run("")
		if c != "mismatch-channels" {
			t.Fatalf("deleted: code %q:\n%s", c, out)
		}
		if l := line(out, "fix  channels  "); l != "fix  channels  config has channel 9003 (#demo-notices), found no such channel in guild 3000; run setup channels to replace it" {
			t.Errorf("deleted: %q", l)
		}
		// setup channels finds by name or creates, then rewrites the id.
		out, c = f.run("", func(o *setup.Options) { o.Step = "channels"; o.Channels = "yes" })
		if c != "" || !strings.Contains(out, "old  9003\nnew  9005\n") || !strings.Contains(out, "ok  channels  Sessions 9001, #demo 9002, #demo-notices 9005\n") {
			t.Errorf("setup channels: code %q:\n%s", c, out)
		}
		if n := calls(f, "Delete"); n != 0 {
			t.Errorf("%d deletes", n)
		}
	})
	t.Run("a guild with channels and --channels no", func(t *testing.T) {
		s := fresh()
		s.Invite.Channels = []discord.FakeChannel{{ID: "700", Name: "Text Channels", Category: true}, {ID: "701", Name: "general", Parent: "700"}}
		f := withFake(newFixture(t), s)
		out, c := f.run(tokA + "\n")
		if c != "timeout-channels" || !strings.Contains(out, "fix  channels  create the category Sessions and the text channels #demo and #demo-notices, or rerun with --channels yes; setup is waiting\n") {
			t.Errorf("code %q:\n%s", c, out)
		}
		if n := calls(f, "CreateGuildChannel"); n != 0 {
			t.Errorf("%d channels created under --channels no", n)
		}
		out, c = f.run("", func(o *setup.Options) { o.Channels = "yes" })
		if c != "" || calls(f, "CreateGuildChannel") != 3 {
			t.Errorf("--channels yes: code %q:\n%s", c, out)
		}
	})
	t.Run("channels found by name", func(t *testing.T) {
		s := fresh()
		s.Invite.Channels = []discord.FakeChannel{{ID: "800", Name: "sessions", Category: true}, {ID: "801", Name: "demo"}, {ID: "802", Name: "demo-notices", Parent: "800"}}
		f := withFake(newFixture(t), s)
		out, c := f.run(tokA + "\n")
		if c != "" || !strings.Contains(out, "ok  channels  Sessions 800, #demo 801, #demo-notices 802\n") || calls(f, "CreateGuildChannel") != 0 {
			t.Errorf("code %q:\n%s", c, out)
		}
	})
}

func TestRole(t *testing.T) {
	t.Run("Manage Roles absent: the owner makes the role", func(t *testing.T) {
		s := fresh()
		g := scriptedGuild(guildID, "Test", false)
		g.OwnerMakesRole, g.OwnerMakesAfter = "Trusted", 2
		s.Invite = &g
		f := withFake(newFixture(t), s)
		out, c := f.run(tokA+"\n", func(o *setup.Options) { o.CreateRole = "no" })
		if c != "" {
			t.Fatalf("refused %s:\n%s", c, out)
		}
		if !strings.Contains(line(out, "fix  guild  "), "&permissions=2251799813811280") {
			t.Errorf("the invite carries Manage Roles:\n%s", out)
		}
		if l := line(out, "fix  role  "); l != "fix  role  create a role named Trusted in Server Settings, Roles, and give it to yourself; setup is waiting" {
			t.Errorf("no-branch line %q", l)
		}
		if !strings.Contains(out, "ok  role  Trusted  ") || calls(f, "CreateRole") != 0 || calls(f, "AddMemberRole") != 0 {
			t.Errorf("role:\n%s\n%v", out, f.fake.Script().Calls)
		}
	})
	t.Run("Manage Roles kept: the removal fix on every run", func(t *testing.T) {
		s := fresh()
		s.Invite.DropManageRolesAfter = 0
		f := withFake(newFixture(t), s)
		out, c := f.run(tokA + "\n")
		const removal = "fix  role  remove Manage Roles from the bot's role in Server Settings, Roles; setup is waiting\n"
		if c != "timeout-role" || !strings.Contains(out, removal) {
			t.Fatalf("code %q:\n%s", c, out)
		}
		out, c = f.run("")
		if c != "timeout-role" || !strings.Contains(out, removal) {
			t.Errorf("second run: code %q:\n%s", c, out)
		}
		out, c = f.run("", func(o *setup.Options) { o.Check = true })
		if c != setup.CodeCheckFailed || !strings.Contains(out, removal) {
			t.Errorf("--check: code %q:\n%s", c, out)
		}
	})
	t.Run("a role landing above the bot", func(t *testing.T) {
		s := fresh()
		s.Invite.RoleAbove = true
		f := withFake(newFixture(t), s)
		out, c := f.run(tokA + "\n")
		if c != "mismatch-role" || line(out, "fix  role  ") != "fix  role  move the bot's role above Trusted in Server Settings, Roles, then run setup role" ||
			calls(f, "AddMemberRole") != 0 {
			t.Errorf("code %q:\n%s", c, out)
		}
	})
	t.Run("a configured role gone", func(t *testing.T) {
		f := withFake(newFixture(t), fresh())
		if out, c := f.run(tokA + "\n"); c != "" {
			t.Fatalf("refused %s:\n%s", c, out)
		}
		f.edit(func(s *discord.FakeScript) { s.Guilds[0].Roles = s.Guilds[0].Roles[:2] })
		if out, c := f.run(""); c != "mismatch-role" {
			t.Errorf("code %q:\n%s", c, out)
		}
	})
}

// setKey sets one key of a config file of the root, as the owner's hand edit.
func (f *fixture) setKey(file, value string, path ...string) {
	f.t.Helper()
	p := f.root + "/config/" + file
	d, _, err := config.ReadDoc(p)
	if err != nil || d == nil {
		f.t.Fatalf("read %s: %v", p, err)
	}
	if err := d.Set("", []byte(value), path...); err != nil {
		f.t.Fatal(err)
	}
	if err := config.WriteDoc(p, d); err != nil {
		f.t.Fatal(err)
	}
}

// TestChannelsOwnersChannels: a configured channel that is not the lobby is the owner's. One
// gone from the guild is mismatch-channels, lifted by setup channels (old, new -, then dropped);
// one present never stands in for the lobby.
func TestChannelsOwnersChannels(t *testing.T) {
	const readOnly = `{"reach":"read-only","roles":{"start":["trusted"]}}`
	t.Run("a configured channel gone, then setup channels", func(t *testing.T) {
		f := withFake(newFixture(t), fresh())
		if out, c := f.run(tokA + "\n"); c != "" {
			t.Fatalf("refused %s:\n%s", c, out)
		}
		f.setKey("projects/demo.json", readOnly, "channels", "9999")
		out, c := f.run("")
		if c != "mismatch-channels" || !strings.Contains(out, "fix  channels  config has channel 9999, found no such channel in guild 3000; run setup channels to replace it\n") {
			t.Fatalf("code %q:\n%s", c, out)
		}
		out, c = f.run("", func(o *setup.Options) { o.Step = "channels"; o.Channels = "yes" })
		if c != "" || !strings.Contains(out, "old  9999\nnew  -\n") || !strings.Contains(out, "ok  channels  Sessions 9001, #demo 9002, #demo-notices 9003\n") {
			t.Fatalf("setup channels: code %q:\n%s", c, out)
		}
		out, c = f.run("")
		if c != "" || !strings.Contains(out, "skip  channels  ") {
			t.Errorf("the next run: code %q:\n%s", c, out)
		}
		if n := calls(f, "CreateGuildChannel"); n != 3 {
			t.Errorf("%d channels created", n)
		}
	})
	t.Run("the owner's channel is not the lobby", func(t *testing.T) {
		s := fresh()
		s.Invite.Channels = []discord.FakeChannel{{ID: "701", Name: "general"}}
		f := withFake(newFixture(t), s)
		// The first run stops at channels (--channels no): the project file exists by then.
		if _, c := f.run(tokA + "\n"); c != "timeout-channels" {
			t.Fatalf("first run: %q", c)
		}
		f.setKey("projects/demo.json", readOnly, "channels", "701")
		out, c := f.run("", func(o *setup.Options) { o.Channels = "yes" })
		if c != "" || !strings.Contains(out, "ok  channels  Sessions 9001, #demo 9002, #demo-notices 9003\n") {
			t.Fatalf("code %q:\n%s", c, out)
		}
		cfg, err := config.Load(f.root, config.AnyCredential)
		if err != nil {
			t.Fatal(err)
		}
		ch := cfg.Projects[0].Channels
		if ch["701"].Reach != config.ReachReadOnly || ch["9002"].Reach != config.ReachCode || len(ch) != 2 {
			t.Errorf("channels %+v", ch)
		}
	})
}

// TestDiscordAnswers: a 5xx is no answer (the waiting fix, then the poll passes), a stored
// token Discord refuses otherwise than 401 is not replaced.
func TestDiscordAnswers(t *testing.T) {
	f := withFake(newFixture(t), fresh())
	if out, c := f.run(tokA + "\n"); c != "" {
		t.Fatalf("refused %s:\n%s", c, out)
	}
	cfg, _ := config.Load(f.root, config.AnyCredential)
	id := cfg.Instance.Discord.BotCredentialID
	f.edit(func(s *discord.FakeScript) {
		s.Fail = map[string][]int{"GetBotApplicationInfo": {502}, "GetCurrentUserGuilds": {502}}
	})
	out, c := f.run("")
	if c != "" || !strings.Contains(out, "fix  token  Discord did not answer (Discord answered 502: Bad Gateway); setup is waiting\nskip  token  "+id+"\n") ||
		!strings.Contains(out, "fix  guild  Discord did not answer (Discord answered 502: Bad Gateway); setup is waiting\nskip  guild  3000  Test\n") {
		t.Errorf("502: code %q:\n%s", c, out)
	}
	f.edit(func(s *discord.FakeScript) { s.Fail = map[string][]int{"GetBotApplicationInfo": {403}} })
	out, c = f.run(tokB + "\n")
	cfg, _ = config.Load(f.root, config.AnyCredential)
	if c != "refused-token" || cfg.Instance.Discord.BotCredentialID != id || !strings.Contains(out, "fix  token  Discord refused the stored token: 403: Forbidden\n") {
		t.Errorf("403: code %q, id %s:\n%s", c, cfg.Instance.Discord.BotCredentialID, out)
	}
}

// TestDiscordGuards: the scripted Discord on an unreadable instance file is refused, not
// assumed dry-run; --json keeps the waiting note off stderr; a malformed roleGroups.trusted is
// mismatch-role, replaced by setup role.
func TestDiscordGuards(t *testing.T) {
	t.Run("unreadable instance file", func(t *testing.T) {
		f := withFake(newFixture(t), fresh())
		if err := os.MkdirAll(f.root+"/config", 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(f.root+"/config/instance.json", []byte("{not json"), 0o600); err != nil {
			t.Fatal(err)
		}
		var out syncBuffer
		env := f.env(strings.NewReader(""), &out)
		env.FakeDiscord = true
		if c := code(setup.Run(context.Background(), f.opts(), env)); c != "bad-config" {
			t.Errorf("code %q:\n%s", c, out.String())
		}
	})
	t.Run("--json stderr", func(t *testing.T) {
		f := withFake(newFixture(t), fresh())
		var out, errb syncBuffer
		env := f.env(strings.NewReader(tokA), &out)
		env.Stderr = &errb
		if c := code(setup.Run(context.Background(), f.opts(func(o *setup.Options) { o.JSON = true }), env)); c != "" {
			t.Fatalf("code %q:\n%s", c, out.String())
		}
		if strings.Contains(errb.String(), "waiting for the bot to join") || !strings.Contains(out.String(), `"step":"guild"`) {
			t.Errorf("stderr %q", errb.String())
		}
	})
	t.Run("malformed roleGroups.trusted", func(t *testing.T) {
		f := withFake(newFixture(t), fresh())
		if out, c := f.run(tokA + "\n"); c != "" {
			t.Fatalf("refused %s:\n%s", c, out)
		}
		f.setKey("instance.json", `"9004"`, "roleGroups", "trusted")
		out, c := f.run("")
		if c != "mismatch-role" || !strings.Contains(out, `fix  role  config has roleGroups.trusted "9004", found not a list of role ids; run setup role to replace it`) {
			t.Fatalf("code %q:\n%s", c, out)
		}
		out, c = f.run("", func(o *setup.Options) { o.Step = "role" })
		if c != "" || !strings.Contains(out, "old  \"9004\"\nnew  9004\n") {
			t.Errorf("setup role: code %q:\n%s", c, out)
		}
	})
	t.Run("a bot with Administrator", func(t *testing.T) {
		s := fresh()
		s.Invite.Roles[1].Permissions |= 1 << 3
		f := withFake(newFixture(t), s)
		out, c := f.run(tokA + "\n")
		if c != "timeout-role" || line(out, "fix  role  ") != "fix  role  the bot's role has Administrator, which includes Manage Roles; remove Administrator from it in Server Settings, Roles; setup is waiting" {
			t.Errorf("code %q:\n%s", c, out)
		}
	})
}
