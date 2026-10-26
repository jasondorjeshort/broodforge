// SPDX-License-Identifier: GPL-3.0-or-later

package discord

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	dg "github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/snowflake/v2"
)

// The calls setup adds to restAPI, on the gateway tests' fake (P22).

func (f *fakeRest) UpdateCurrentApplication(u dg.ApplicationUpdate, _ ...rest.RequestOpt) (*dg.Application, error) {
	f.record("UpdateCurrentApplication", js(json.Marshal(u)))
	return &dg.Application{ID: sf(testApp)}, nil
}
func (f *fakeRest) GetCurrentUserGuilds(bearer string, before, after snowflake.ID, limit int, withCounts bool, _ ...rest.RequestOpt) ([]dg.OAuth2Guild, error) {
	f.record("GetCurrentUserGuilds", "", "bearer="+bearer)
	return []dg.OAuth2Guild{{ID: sf(testGuild), Name: "Test"}}, nil
}
func (f *fakeRest) GetRoles(g snowflake.ID, _ ...rest.RequestOpt) ([]dg.Role, error) {
	f.record("GetRoles", "", g.String())
	return f.roles, nil
}
func (f *fakeRest) CreateRole(g snowflake.ID, c dg.RoleCreate, _ ...rest.RequestOpt) (*dg.Role, error) {
	f.record("CreateRole", js(json.Marshal(c)), g.String())
	return &dg.Role{ID: f.id(), Name: c.Name, Position: 1}, nil
}
func (f *fakeRest) AddMemberRole(g, u, r snowflake.ID, _ ...rest.RequestOpt) error {
	f.record("AddMemberRole", "", g.String(), u.String(), r.String())
	return nil
}

// TestSetupAPI drives every SetupAPI call against the recording fake and checks the request
// each sends: the PATCH ORs 1<<19 into the flags it was given, the guild list uses the bot's
// own token, a role is created with no permissions, a text channel lands under its category,
// and the bot's permissions and highest role come from its roles.
func TestSetupAPI(t *testing.T) {
	ctx := context.Background()
	const botRole = "5001"
	r := &fakeRest{
		roles: []dg.Role{
			{ID: sf(testGuild), Name: "@everyone", Permissions: noticePerms},
			{ID: sf(botRole), Name: "Broodforge", Position: 3, Permissions: categoryPerms | dg.PermissionManageRoles},
			{ID: sf("5002"), Name: "Other", Position: 5},
		},
		memberRoles: []snowflake.ID{sf(botRole)},
	}
	a := &setupAPI{r: r}

	app, err := a.Application(ctx)
	if err != nil || app.ID != testApp || app.BotID != testBot || app.Flags != FlagMessageContent {
		t.Fatalf("Application: %+v %v", app, err)
	}
	if err := a.EnableMessageContent(ctx, 1<<23); err != nil {
		t.Fatal(err)
	}
	if c, ok := r.find("UpdateCurrentApplication", nil); !ok || c.body != `{"flags":`+jsonInt(1<<23|1<<19)+`}` {
		t.Errorf("PATCH body %q", c.body)
	}
	gs, err := a.Guilds(ctx)
	if err != nil || len(gs) != 1 || gs[0].ID != testGuild {
		t.Fatalf("Guilds: %+v %v", gs, err)
	}
	if _, ok := r.find("GetCurrentUserGuilds", func(c restCall) bool { return c.args[0] == "bearer=" }); !ok {
		t.Errorf("GetCurrentUserGuilds not with the bot's token:\n%s", r.dump())
	}
	g, err := a.Guild(ctx, testGuild)
	if err != nil || g.Name != "Test" {
		t.Errorf("Guild: %+v %v", g, err)
	}
	role, err := a.CreateRole(ctx, testGuild, "Trusted")
	if err != nil || role.Name != "Trusted" {
		t.Fatalf("CreateRole: %+v %v", role, err)
	}
	if c, _ := r.find("CreateRole", nil); !strings.Contains(c.body, `"name":"Trusted"`) || !strings.Contains(c.body, `"permissions":"0"`) {
		t.Errorf("CreateRole body %q", c.body)
	}
	cat, err := a.CreateCategory(ctx, testGuild, "Sessions")
	if err != nil {
		t.Fatal(err)
	}
	if c, _ := r.find("CreateGuildChannel", nil); !strings.Contains(c.body, `"name":"Sessions"`) || !strings.Contains(c.body, `"type":4`) {
		t.Errorf("category body %q", c.body)
	}
	if _, err := a.CreateTextChannel(ctx, testGuild, "demo", cat.ID); err != nil {
		t.Fatal(err)
	}
	if _, ok := r.find("CreateGuildChannel", func(c restCall) bool {
		return strings.Contains(c.body, `"name":"demo"`) && strings.Contains(c.body, `"parent_id":"`+cat.ID+`"`)
	}); !ok {
		t.Errorf("no text channel under the category:\n%s", r.dump())
	}
	if err := a.AddMemberRole(ctx, testGuild, "6000", role.ID); err != nil {
		t.Fatal(err)
	}
	if _, ok := r.find("AddMemberRole", func(c restCall) bool { return c.args[1] == "6000" && c.args[2] == role.ID }); !ok {
		t.Errorf("AddMemberRole:\n%s", r.dump())
	}
	held, err := a.MemberRoles(ctx, testGuild, "6000")
	if err != nil || len(held) != 1 || held[0] != botRole {
		t.Errorf("MemberRoles: %v %v", held, err)
	}
	bot, err := a.Bot(ctx, testGuild)
	want := uint64(noticePerms | categoryPerms | dg.PermissionManageRoles)
	if err != nil || bot.Highest != 3 || bot.Permissions != want || bot.ID != testBot {
		t.Errorf("Bot: %+v %v, want permissions %d, highest 3", bot, err, want)
	}
	roles, err := a.Roles(ctx, testGuild)
	if err != nil || len(roles) != 3 || roles[1].Position != 3 {
		t.Errorf("Roles: %+v %v", roles, err)
	}

	// A call Discord refuses is a SetupError with its reason; no answer is not.
	refused := setupErr(&rest.Error{Response: &http.Response{StatusCode: 403, Status: "403 Forbidden"}, Code: 50013, Message: "Missing Permissions"})
	if se, ok := Refused(refused); !ok || se.Status != 403 || se.Reason != "50013: Missing Permissions" {
		t.Errorf("refusal: %v", refused)
	}
	if _, ok := Refused(setupErr(errors.New("dial tcp: no route"))); ok {
		t.Error("a network error read as Discord's refusal")
	}
}

func jsonInt(v int) string { b, _ := json.Marshal(v); return string(b) }

// TestInviteURL is the critic's figure (C7): the operator file's integer, and with Manage Roles
// 1<<28 more.
func TestInviteURL(t *testing.T) {
	without := InviteURL(testApp, false, "")
	if want := "https://discord.com/oauth2/authorize?client_id=" + testApp + "&scope=bot+applications.commands&permissions=2251799813811280"; without != want {
		t.Errorf("invite %s", without)
	}
	with := InviteURL(testApp, true, testGuild)
	if !strings.Contains(with, "&permissions=2251800082246736&guild_id="+testGuild+"&disable_guild_select=true") {
		t.Errorf("invite with Manage Roles %s", with)
	}
}

// TestFakeSetup is the scripted Discord the binary reads: a token it does not accept is 401, the
// invite joins after its reads, a created channel is in the next read.
func TestFakeSetup(t *testing.T) {
	ctx := context.Background()
	const tok = "NTAwMA.abcd.abcdefghijklmnopqrstuvwxyz"
	f := NewFakeSetup(FakeScript{Token: tok, Application: FakeApp{ID: testApp, BotID: testBot},
		Invite: &FakeGuild{ID: testGuild, Name: "Test", OwnerID: "6000",
			Roles:    []FakeRole{{ID: testGuild, Name: "@everyone"}, {ID: "5001", Name: "Broodforge", Position: 2, Permissions: PermissionBits}},
			BotRoles: []string{"5001"}},
		JoinAfter: 1})
	bad, err := f.API("NTAwMA.abcd.zyxwvutsrqponmlkjihgfedcba")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := bad.Application(ctx); err == nil {
		t.Error("the fake took a token it does not accept")
	} else if se, ok := Refused(err); !ok || se.Status != 401 {
		t.Errorf("wrong token: %v", err)
	}
	a, err := f.API(tok)
	if err != nil {
		t.Fatal(err)
	}
	if gs, _ := a.Guilds(ctx); len(gs) != 0 {
		t.Errorf("joined before the invite: %v", gs)
	}
	if gs, _ := a.Guilds(ctx); len(gs) != 1 {
		t.Errorf("not joined after the invite: %v", gs)
	}
	c, err := a.CreateCategory(ctx, testGuild, "Sessions")
	if err != nil {
		t.Fatal(err)
	}
	chans, err := a.Channels(ctx, testGuild)
	if err != nil || len(chans) != 1 || chans[0].ID != c.ID || !chans[0].Category {
		t.Errorf("channels %+v %v", chans, err)
	}
	if _, err := a.CreateRole(ctx, testGuild, "Trusted"); err == nil {
		t.Error("a role created without Manage Roles")
	}
}
