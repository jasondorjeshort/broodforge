// SPDX-License-Identifier: GPL-3.0-or-later

package discord

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	dg "github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/gateway"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/snowflake/v2"

	"github.com/jasondorjeshort/broodforge/internal/audit"
	"github.com/jasondorjeshort/broodforge/internal/config"
	"github.com/jasondorjeshort/broodforge/internal/engine"
	"github.com/jasondorjeshort/broodforge/internal/git"
	"github.com/jasondorjeshort/broodforge/internal/session"
	"github.com/jasondorjeshort/broodforge/internal/store"
)

// The live gateway without Discord (P12): a fake of restAPI records every call with its disgo
// request body, a real session service runs the fake engine, and the adapter's coalescer runs
// on a manual clock the test steps. What only a guild proves is listed in
// 05-debug-interface-discord-live.md.

const (
	testApp   = "4000"
	testGuild = "3000"
	testBot   = "5000"
	testCat   = "900"
	testLobby = "100"
)

type restCall struct {
	name string
	args []string
	body string
}

// fakeRest is restAPI recording each call; ids it hands out count up from 8000.
type fakeRest struct {
	mu       sync.Mutex
	calls    []restCall
	next     uint64
	channels []dg.GuildChannel
	roles    []dg.Role
	noDM     bool
}

func js(b []byte, err error) string {
	if err != nil {
		return "marshal: " + err.Error()
	}
	return string(b)
}

func (f *fakeRest) record(name, body string, args ...string) {
	f.mu.Lock()
	f.calls = append(f.calls, restCall{name, args, body})
	f.mu.Unlock()
}

func (f *fakeRest) id() snowflake.ID {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.next++
	return snowflake.ID(8000 + f.next)
}

func (f *fakeRest) find(name string, match func(restCall) bool) (restCall, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, c := range f.calls {
		if c.name == name && (match == nil || match(c)) {
			return c, true
		}
	}
	return restCall{}, false
}

// count is how many calls were made.
func (f *fakeRest) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

func (f *fakeRest) dump() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	var b strings.Builder
	for _, c := range f.calls {
		b.WriteString(c.name + " " + strings.Join(c.args, " ") + " " + c.body + "\n")
	}
	return b.String()
}

func sf(s string) snowflake.ID { return snowflake.MustParse(s) }

func (f *fakeRest) GetBotApplicationInfo(...rest.RequestOpt) (*dg.Application, error) {
	f.record("GetBotApplicationInfo", "")
	return &dg.Application{ID: sf(testApp), Bot: &dg.User{ID: sf(testBot)}, Flags: dg.ApplicationFlagGatewayMessageContent}, nil
}
func (f *fakeRest) GetGuild(g snowflake.ID, _ bool, _ ...rest.RequestOpt) (*dg.RestGuild, error) {
	f.record("GetGuild", "", g.String())
	return &dg.RestGuild{Guild: dg.Guild{ID: g, Name: "Test"}, Roles: f.roles}, nil
}
func (f *fakeRest) GetMember(g, u snowflake.ID, _ ...rest.RequestOpt) (*dg.Member, error) {
	f.record("GetMember", "", g.String(), u.String())
	return &dg.Member{User: dg.User{ID: u}}, nil
}
func (f *fakeRest) GetGuildChannels(g snowflake.ID, _ ...rest.RequestOpt) ([]dg.GuildChannel, error) {
	f.record("GetGuildChannels", "", g.String())
	return f.channels, nil
}
func (f *fakeRest) SetGuildCommands(a, g snowflake.ID, cmds []dg.ApplicationCommandCreate, _ ...rest.RequestOpt) ([]dg.ApplicationCommand, error) {
	f.record("SetGuildCommands", js(json.Marshal(cmds)), a.String(), g.String())
	return nil, nil
}
func (f *fakeRest) CreateGuildChannel(g snowflake.ID, c dg.GuildChannelCreate, _ ...rest.RequestOpt) (dg.GuildChannel, error) {
	f.record("CreateGuildChannel", js(json.Marshal(c)), g.String())
	tc, _ := c.(dg.GuildTextChannelCreate)
	return textChannel(f.id(), tc.Name, tc.Topic, tc.ParentID), nil
}
func (f *fakeRest) DeleteChannel(c snowflake.ID, _ ...rest.RequestOpt) error {
	f.record("DeleteChannel", "", c.String())
	return nil
}
func (f *fakeRest) CreateMessage(c snowflake.ID, m dg.MessageCreate, _ ...rest.RequestOpt) (*dg.Message, error) {
	f.record("CreateMessage", js(json.Marshal(m)), c.String())
	return &dg.Message{ID: f.id(), ChannelID: c}, nil
}
func (f *fakeRest) UpdateMessage(c, m snowflake.ID, u dg.MessageUpdate, _ ...rest.RequestOpt) (*dg.Message, error) {
	args := []string{c.String(), m.String()}
	for _, file := range u.Files {
		args = append(args, "file:"+file.Name)
	}
	f.record("UpdateMessage", js(json.Marshal(u)), args...)
	return &dg.Message{ID: m, ChannelID: c}, nil
}
func (f *fakeRest) DeleteMessage(c, m snowflake.ID, _ ...rest.RequestOpt) error {
	f.record("DeleteMessage", "", c.String(), m.String())
	return nil
}
func (f *fakeRest) PinMessage(c, m snowflake.ID, _ ...rest.RequestOpt) error {
	f.record("PinMessage", "", c.String(), m.String())
	return nil
}
func (f *fakeRest) AddReaction(c, m snowflake.ID, emoji string, _ ...rest.RequestOpt) error {
	f.record("AddReaction", "", c.String(), m.String(), emoji)
	return nil
}
func (f *fakeRest) CreateDMChannel(u snowflake.ID, _ ...rest.RequestOpt) (*dg.DMChannel, error) {
	f.record("CreateDMChannel", "", u.String())
	if f.noDM {
		return nil, &rest.Error{Code: 50007, Message: "Cannot send messages to this user"}
	}
	var ch dg.DMChannel
	if err := json.Unmarshal([]byte(`{"id":"6000","type":1}`), &ch); err != nil {
		return nil, err
	}
	return &ch, nil
}
func (f *fakeRest) CreateInteractionResponse(i snowflake.ID, token string, r dg.InteractionResponse, _ ...rest.RequestOpt) error {
	f.record("CreateInteractionResponse", js(json.Marshal(r)), i.String(), token)
	return nil
}
func (f *fakeRest) UpdateInteractionResponse(a snowflake.ID, token string, u dg.MessageUpdate, _ ...rest.RequestOpt) (*dg.Message, error) {
	f.record("UpdateInteractionResponse", js(json.Marshal(u)), a.String(), token)
	return &dg.Message{}, nil
}
func (f *fakeRest) DeleteInteractionResponse(a snowflake.ID, token string, _ ...rest.RequestOpt) error {
	f.record("DeleteInteractionResponse", "", a.String(), token)
	return nil
}
func (f *fakeRest) CreateFollowupMessage(a snowflake.ID, token string, m dg.MessageCreate, _ ...rest.RequestOpt) (*dg.Message, error) {
	f.record("CreateFollowupMessage", js(json.Marshal(m)), a.String(), token)
	return &dg.Message{}, nil
}

// textChannel is a guild text channel; topic "" is none.
func textChannel(id snowflake.ID, name, topic string, parent snowflake.ID) dg.GuildChannel {
	t := "null"
	if topic != "" {
		t = `"` + topic + `"`
	}
	raw := `{"id":"` + id.String() + `","type":0,"guild_id":"` + testGuild + `","name":"` + name + `","topic":` + t + `,"parent_id":"` + parent.String() + `"}`
	var c dg.UnmarshalChannel
	if err := json.Unmarshal([]byte(raw), &c); err != nil {
		panic(err)
	}
	return c.Channel.(dg.GuildChannel)
}

// memCredentials is /credential's store, in memory.
type memCredentials struct {
	mu   sync.Mutex
	recs []CredentialInfo
	vals map[string]string
	user map[string]string
}

func (m *memCredentials) Add(user, kind, label string, value []byte) (CredentialInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c := CredentialInfo{ID: "cred-" + strconv.Itoa(len(m.recs)+1), Kind: kind, Label: label, Status: "usable"}
	if kind == kindSubscription {
		c.Status = "needs-login"
	}
	m.recs = append(m.recs, c)
	m.vals[c.ID], m.user[c.ID] = string(value), user
	return c, nil
}

func (m *memCredentials) List(user string) ([]CredentialInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []CredentialInfo
	for _, c := range m.recs {
		if m.user[c.ID] == user {
			out = append(out, c)
		}
	}
	return out, nil
}

func (m *memCredentials) Remove(user, id string) (CredentialInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, c := range m.recs {
		if c.ID == id {
			m.recs[i].Status = "removed"
			return m.recs[i], nil
		}
	}
	return CredentialInfo{}, &session.Error{Code: "unknown-credential", Message: "no credential " + id}
}

type liveHarness struct {
	t     *testing.T
	st    *store.Store
	svc   *session.Service
	a     *Adapter
	g     *Gateway
	rest  *fakeRest
	clock *session.ManualClock
	creds *memCredentials
	log   *syncLog
}

// delayedBasic is fixtures/engine/basic with its lines delay apart, so a turn is still running
// when a second message arrives.
func delayedBasic(t *testing.T, delay time.Duration) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "basic")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	for n := 1; n <= 2; n++ {
		b, err := os.ReadFile(engine.TurnFile("../../fixtures/engine/basic", n))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(engine.TurnFile(dir, n), b, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	fx := `{"claim":"test","delayMs":` + strconv.Itoa(int(delay/time.Millisecond)) + `,"turns":[{"n":1,"exit":0},{"n":2,"exit":0}]}`
	if err := os.WriteFile(filepath.Join(dir, engine.FixtureFile), []byte(fx), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

func newLiveHarness(t *testing.T, leftovers ...dg.GuildChannel) *liveHarness {
	t.Helper()
	return newLiveHarnessHeld(t, nil, leftovers...)
}

// newLiveHarnessHeld is newLiveHarness as serve starts after a restart (ADR 0026): held writes the
// store's earlier rows (and what an earlier run left under the root), then the service holds
// every interrupted session before prepare's sweep, and the coalescer starts after it.
func newLiveHarnessHeld(t *testing.T, held func(root string, tx *store.Tx) error, leftovers ...dg.GuildChannel) *liveHarness {
	t.Helper()
	root := filepath.ToSlash(t.TempDir())
	st, err := store.Create(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if err := st.Write(func(tx *store.Tx) error {
		_, err := tx.Exec(`INSERT INTO credential (id, kind, provider, user_id, label, status, added_at, store_ref)
VALUES ('cred-x', 'api-key', 'anthropic', NULL, 'x', 'usable', '2026-10-21T00:00:00Z', 'cred-x')`)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	fixture := delayedBasic(t, 150*time.Millisecond)
	prompt := config.ModePolicy
	cfg := &config.Config{
		Instance: config.Instance{InstanceName: "test", RoleGroups: map[string][]string{"trusted": {"10"}},
			Defaults: config.Defaults{Mode: config.ModePolicy, Harness: config.HarnessFake, IdleTimeoutMinutes: 60, PromptTimeoutMinutes: 5}},
		Projects: []config.Project{{ID: "p", Name: "Project P", Venue: config.Venue{Category: testCat},
			Channels: map[string]config.Channel{testLobby: {Reach: config.ReachCode, Mode: &prompt,
				Roles: config.Roles{Start: []string{"trusted"}, Contribute: []string{"trusted", "20"}, Answer: []string{"driver"}}}}}},
	}
	roles := session.Roles{Start: []string{"trusted"}, Contribute: []string{"trusted", "20"}, Answer: []string{"driver"}}
	channel := func(id string) (session.Channel, bool) {
		if id != testLobby {
			return session.Channel{}, false
		}
		return session.Channel{ID: id, Project: "p", ProjectName: "Project P", Reach: "code", Mode: "policy", Harness: "fake",
			Roles: roles, Repo: session.Repo{Canonical: root + "/canonical", Remote: "origin", DefaultBranch: "main"}}, true
	}
	groups := func() map[string][]string { return cfg.Instance.RoleGroups }
	h := &liveHarness{t: t, st: st, rest: &fakeRest{channels: leftovers},
		clock: session.NewManualClock(time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)),
		creds: &memCredentials{vals: map[string]string{}, user: map[string]string{}}, log: &syncLog{}}
	h.a = New(Config{Config: func() *config.Config { return cfg }, Channel: channel, Groups: groups, Clock: h.clock, Credentials: h.creds})
	h.svc = session.New(session.Config{Root: root, Instance: "test", Groups: groups, Channel: channel, Started: h.a.SessionStarted,
		Fixtures: filepath.Dir(fixture),
		Engine:   func(_, _, received string) (engine.Engine, error) { return engine.NewFake(fixture, received) },
		CreateTree: func(_ context.Context, s git.Session) (string, error) {
			return "base", os.MkdirAll(filepath.FromSlash(s.Dir()+"/tree"), 0o750)
		},
		Store: st, Audit: audit.NewWriter(nil), Clock: session.NewManualClock(time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC))})
	h.a.Attach(h.svc)
	if h.g, err = newGateway(h.a, GatewayConfig{ApplicationID: testApp, GuildID: testGuild, NoticeChannel: "300", Instance: "test", Log: h.log,
		Report: "Broodforge test started: engine fake, 1 projects."}, h.rest); err != nil {
		t.Fatal(err)
	}
	h.a.live.Store(h.g)
	if held != nil {
		if err := st.Write(func(tx *store.Tx) error { return held(root, tx) }); err != nil {
			t.Fatal(err)
		}
		if _, err := h.svc.LoadInterrupted(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if err := h.g.prepare(context.Background()); err != nil {
		t.Fatal(err)
	}
	h.a.Run()
	h.g.report()
	t.Cleanup(func() {
		h.g.Close(context.Background())
		h.a.Close()
	})
	return h
}

// until steps the adapter's clock a second at a time until ok holds.
func (h *liveHarness) until(what string, ok func() bool) {
	h.t.Helper()
	deadline := time.Now().Add(20 * time.Second)
	for !ok() {
		if time.Now().After(deadline) {
			h.t.Fatalf("timed out waiting for %s; REST calls:\n%s", what, h.rest.dump())
		}
		h.clock.Advance(time.Second)
		time.Sleep(5 * time.Millisecond)
	}
}

func (h *liveHarness) saw(name string, match func(restCall) bool) func() bool {
	return func() bool { _, ok := h.rest.find(name, match); return ok }
}

// content is a recorded body's content field, decoded.
func content(c restCall) string {
	var b struct{ Content string }
	_ = json.Unmarshal([]byte(c.body), &b)
	return b.Content
}

func bodyHas(s string) func(restCall) bool {
	return func(c restCall) bool { return strings.Contains(content(c), s) }
}

func interaction(t *testing.T, raw string) dg.Interaction {
	t.Helper()
	i, err := dg.UnmarshalInteraction([]byte(raw))
	if err != nil {
		t.Fatalf("interaction %s: %v", raw, err)
	}
	return i
}

// slashJSON is a slash command interaction from user 1001 with roles 10 in channel.
func slashJSON(id, channel, data string) string {
	return `{"id":"` + id + `","application_id":"` + testApp + `","type":2,"token":"tok-` + id + `","version":1,"guild_id":"` + testGuild +
		`","channel":{"id":"` + channel + `","type":0,"guild_id":"` + testGuild + `","name":"c"},` +
		`"member":{"user":{"id":"1001","username":"ada"},"roles":["10"]},"data":` + data + `}`
}

func TestIntentsAndPermissionBits(t *testing.T) {
	if IntentBits != 1|1<<9|1<<15 {
		t.Errorf("intents %d, want Guilds|GuildMessages|MessageContent = 33281", IntentBits)
	}
	if want := uint64(1<<4 | 1<<6 | 1<<10 | 1<<11 | 1<<13 | 1<<14 | 1<<15 | 1<<16 | 1<<51); PermissionBits != want {
		t.Errorf("permissions %d, want %d", PermissionBits, want)
	}
	if got := permsLine(noticePerms, categoryPerms); got != "missing:embed,attach,add-reactions,read-history,manage-messages,pin,manage-channels" {
		t.Errorf("permsLine: %s", got)
	}
}

func TestCheckToken(t *testing.T) {
	good := "NTAwMA.Gabcde.abcdefghijklmnopqrstuvwxyz0123" // the first part is base64 of 5000
	if id, err := CheckToken(good); err != nil || id != testBot {
		t.Errorf("good token: %q %v", id, err)
	}
	for _, bad := range []string{"", "placeholder", "a.b.c", "bm90LWFuLWlk.Gabcde.abcdefghijklmnopqrstuvwxyz0123"} {
		_, err := CheckToken(bad)
		if err == nil {
			t.Errorf("%q passed", bad)
		} else if bad != "" && strings.Contains(err.Error(), bad) {
			t.Errorf("the refusal repeats the value: %v", err)
		}
	}
}

func TestCommandsAndSlashOf(t *testing.T) {
	b, err := json.Marshal(Commands())
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{`"name":"register"`, `"name":"session"`, `"name":"credential"`, `"name":"usage"`,
		`"name":"mode"`, `"value":"acts-freely"`, `"name":"invite"`, `"type":6`, `"value":"api-key"`, `"name":"default"`, `"name":"login"`} {
		if !strings.Contains(string(b), s) {
			t.Errorf("commands lack %s: %s", s, b)
		}
	}
	cases := []struct {
		data string
		kind SlashKind
		name string
		args []string
		opts map[string]string
	}{
		{`{"id":"1","name":"register","type":1,"options":[{"name":"name","type":3,"value":"Ada Example"},{"name":"email","type":3,"value":"ada@example.com"}]}`,
			SlashRegister, "register", nil, map[string]string{"name": "Ada Example", "email": "ada@example.com"}},
		{`{"id":"1","name":"session","type":1,"options":[{"name":"mode","type":1,"options":[{"name":"mode","type":3,"value":"acts-freely"}]}]}`,
			SlashSessionMode, "session mode", []string{"acts-freely"}, map[string]string{}},
		{`{"id":"1","name":"session","type":1,"options":[{"name":"invite","type":1,"options":[{"name":"user","type":6,"value":"1002"}]}]}`,
			SlashSessionInvite, "session invite", nil, map[string]string{"user": "1002"}},
		{`{"id":"1","name":"credential","type":1,"options":[{"name":"add","type":1,"options":[{"name":"kind","type":3,"value":"api-key"},{"name":"label","type":3,"value":"mine"}]}]}`,
			SlashCredential, "credential", []string{"add"}, map[string]string{"kind": "api-key", "label": "mine"}},
	}
	for _, c := range cases {
		i := interaction(t, slashJSON("1", testLobby, c.data)).(dg.ApplicationCommandInteraction)
		sl, _, ok := slashOf(i.SlashCommandInteractionData())
		if !ok || sl.Kind != c.kind || sl.Name != c.name || strings.Join(sl.Args, " ") != strings.Join(c.args, " ") {
			t.Errorf("%s: %+v %v", c.data, sl, ok)
		}
		for k, v := range c.opts {
			if sl.Options[k] != v {
				t.Errorf("%s: option %s = %q, want %q", c.data, k, sl.Options[k], v)
			}
		}
	}
}

func TestVenueOverwrites(t *testing.T) {
	ows := venueOverwrites(sf(testGuild), sf(testBot), sf("1001"), roleIDs([]string{"trusted", "20", "driver", "trusted"},
		map[string][]string{"trusted": {"10"}}))
	b, err := json.Marshal(ows)
	if err != nil {
		t.Fatal(err)
	}
	want := `[{"type":0,"id":"3000","allow":"0","deny":"1024"},{"type":1,"id":"5000","allow":"2251799813811280","deny":"0"},` +
		`{"type":1,"id":"1001","allow":"117824","deny":"0"},{"type":0,"id":"10","allow":"117824","deny":"0"},{"type":0,"id":"20","allow":"117824","deny":"0"}]`
	if string(b) != want {
		t.Errorf("overwrites\n got %s\nwant %s", b, want)
	}
	// The bot's computed permissions in a venue: @everyone may not view; the bot's overwrite gives venuePerms.
	have := channelPerms(sf(testGuild), []dg.Role{{ID: sf(testGuild), Permissions: dg.PermissionViewChannel | dg.PermissionSendMessages}},
		nil, sf(testBot), dg.PermissionOverwrites(ows))
	if permsLine(have, categoryPerms) != "perms:ok" {
		t.Errorf("venue perms %d", have)
	}
	if got := permsLine(channelPerms(sf(testGuild), []dg.Role{{ID: sf(testGuild), Permissions: dg.PermissionViewChannel}}, nil, sf(testBot),
		dg.PermissionOverwrites{dg.RolePermissionOverwrite{RoleID: sf(testGuild), Deny: dg.PermissionViewChannel}}), noticePerms); got != "missing:view,send" {
		t.Errorf("denied view: %s", got)
	}
}

// TestLiveSession drives a session the way Discord would: commands registered, the panel's
// start, the venue created and pinned, messages through the key-paste scan, the hourglass, End,
// and the venue deleted ten minutes after the summary.
func TestLiveSession(t *testing.T) {
	old := textChannel(sf("7777"), "s0099-old", "test:s0099", sf(testCat)) // a session the store does not know: left alone
	h := newLiveHarness(t, old)
	if c, ok := h.rest.find("SetGuildCommands", nil); !ok || c.args[0] != testApp || c.args[1] != testGuild || !strings.Contains(c.body, `"name":"credential"`) {
		t.Fatalf("commands: %+v\n%s", c, h.rest.dump())
	}
	h.until("the start-up report", h.saw("CreateMessage", func(c restCall) bool {
		return c.args[0] == "300" && strings.Contains(c.body, "Broodforge test started")
	}))

	text := func(token string) string {
		c, _ := h.rest.find("UpdateInteractionResponse", func(c restCall) bool { return c.args[1] == token })
		return c.body
	}

	// /register, deferred ephemeral, then the reply.
	h.g.onInteraction(interaction(t, slashJSON("11", testLobby,
		`{"id":"1","name":"register","type":1,"options":[{"name":"name","type":3,"value":"Ada Example"},{"name":"email","type":3,"value":"ada@example.com"}]}`)))
	if c, ok := h.rest.find("CreateInteractionResponse", func(c restCall) bool { return c.args[1] == "tok-11" }); !ok || c.body != `{"type":5,"data":{"flags":64}}` {
		t.Errorf("register defer: %+v", c)
	}
	if !strings.Contains(text("tok-11"), "Registered. Commits you drive are authored as Ada Example") {
		t.Errorf("register reply: %s", text("tok-11"))
	}
	if _, err := h.svc.SetDefault("1001", "cred-x", session.ActorHost); err != nil {
		t.Fatal(err)
	}

	// The panel's Start session button: the venue under the category, then the reply naming it.
	done := make(chan struct{})
	go func() {
		defer close(done)
		h.g.onInteraction(interaction(t, `{"id":"12","application_id":"`+testApp+`","type":3,"token":"tok-12","version":1,"guild_id":"`+testGuild+
			`","channel":{"id":"100","type":0,"guild_id":"`+testGuild+`","name":"c"},"member":{"user":{"id":"1001","username":"ada"},"roles":["10"]},`+
			`"message":{"id":"1","channel_id":"100","author":{"id":"5000"}},"data":{"custom_id":"bf:100:start","component_type":2}}`))
	}()
	h.until("the venue", h.saw("CreateGuildChannel", nil))
	<-done
	venue, _ := h.rest.find("CreateGuildChannel", nil)
	if !strings.Contains(venue.body, `"name":"s0001-ada"`) || !strings.Contains(venue.body, `"topic":"test:s0001"`) || !strings.Contains(venue.body, `"parent_id":"900"`) ||
		!strings.Contains(venue.body, `{"type":0,"id":"3000","allow":"0","deny":"1024"}`) || !strings.Contains(venue.body, `"id":"20"`) {
		t.Errorf("venue body: %s", venue.body)
	}
	venueID := h.a.venueChannel("s0001", 0)
	if venueID == "s0001" {
		t.Fatal("the venue was not bound")
	}
	if c, ok := h.rest.find("CreateFollowupMessage", bodyHas("Session s0001 is open in <#"+venueID+">")); !ok {
		t.Errorf("start follow-up: %+v\n%s", c, h.rest.dump())
	}
	h.until("the pinned header", h.saw("PinMessage", func(c restCall) bool { return c.args[0] == venueID }))

	// A message in the venue starts turn 1; a second one during it is queued: the hourglass.
	msg := func(id, content string) dg.Message {
		g := sf(testGuild)
		return dg.Message{ID: sf(id), ChannelID: sf(venueID), GuildID: &g, Author: dg.User{ID: sf("1001")},
			Member: &dg.Member{RoleIDs: []snowflake.ID{sf("10")}}, Content: content}
	}
	h.g.onMessage(msg("21", "Read hello.txt and reply with its one word."))
	h.g.onMessage(msg("22", "And say it again."))
	if c, ok := h.rest.find("AddReaction", nil); !ok || c.args[1] != "22" || c.args[2] != hourglass {
		t.Errorf("hourglass: %+v\n%s", c, h.rest.dump())
	}
	h.until("turn 1's footer", h.saw("CreateMessage", func(c restCall) bool {
		return c.args[0] == venueID && strings.Contains(c.body, "Idle · turn 1")
	}))
	h.until("a working message deleted", h.saw("DeleteMessage", func(c restCall) bool { return c.args[0] == venueID }))

	// A pasted key: deleted, the refusal by direct message, nothing submitted.
	h.g.onMessage(msg("23", "my key is sk-ant-api03-abcdefgh"))
	if _, ok := h.rest.find("DeleteMessage", func(c restCall) bool { return c.args[1] == "23" }); !ok {
		t.Errorf("the pasted key was not deleted\n%s", h.rest.dump())
	}
	if c, ok := h.rest.find("CreateMessage", func(c restCall) bool { return c.args[0] == "6000" }); !ok ||
		!strings.Contains(c.body, "Never paste a key in chat") || strings.Contains(c.body, "sk-ant") {
		t.Errorf("key refusal DM: %+v", c)
	}

	// /credential add kind:api-key answers with the modal; its submission stores the key.
	h.g.onInteraction(interaction(t, slashJSON("13", venueID,
		`{"id":"1","name":"credential","type":1,"options":[{"name":"add","type":1,"options":[{"name":"kind","type":3,"value":"api-key"},{"name":"label","type":3,"value":"mine"}]}]}`)))
	if c, ok := h.rest.find("CreateInteractionResponse", func(c restCall) bool { return c.args[1] == "tok-13" }); !ok ||
		!strings.Contains(c.body, `"type":9`) || !strings.Contains(c.body, `"custom_id":"bfm:api-key:mine"`) || !strings.Contains(c.body, `"label":"Anthropic API key"`) {
		t.Errorf("modal: %+v", c)
	}
	h.g.onInteraction(interaction(t, `{"id":"14","application_id":"`+testApp+`","type":5,"token":"tok-14","version":1,"guild_id":"`+testGuild+
		`","channel":{"id":"`+venueID+`","type":0,"guild_id":"`+testGuild+`","name":"c"},"member":{"user":{"id":"1001","username":"ada"},"roles":["10"]},`+
		`"data":{"custom_id":"bfm:api-key:mine","components":[{"type":1,"components":[{"type":4,"custom_id":"key","value":"sk-ant-test-0000a7f2"}]}]}}`))
	if got := text("tok-14"); !strings.Contains(got, "Stored cred-1 (API key, ends in ...a7f2).") {
		t.Errorf("modal reply: %s\n%s", got, h.rest.dump())
	}
	if h.creds.vals["cred-1"] != "sk-ant-test-0000a7f2" {
		t.Errorf("stored %q", h.creds.vals["cred-1"])
	}
	h.g.onInteraction(interaction(t, slashJSON("15", venueID, `{"id":"1","name":"credential","type":1,"options":[{"name":"list","type":1}]}`)))
	if got := text("tok-15"); !strings.Contains(got, "cred-1 · API key · mine · usable · never used") {
		t.Errorf("list: %s", got)
	}

	// End session from the header: the summary in the lobby with the transcript, then the
	// venue goes ten minutes later.
	h.g.onInteraction(interaction(t, `{"id":"16","application_id":"`+testApp+`","type":3,"token":"tok-16","version":1,"guild_id":"`+testGuild+
		`","channel":{"id":"`+venueID+`","type":0,"guild_id":"`+testGuild+`","name":"c"},"member":{"user":{"id":"1001","username":"ada"},"roles":["10"]},`+
		`"message":{"id":"1","channel_id":"`+venueID+`","author":{"id":"5000"}},"data":{"custom_id":"bf:s0001:end","component_type":2}}`))
	h.until("the transcript", h.saw("UpdateMessage", func(c restCall) bool {
		return c.args[0] == testLobby && len(c.args) > 2 && c.args[2] == "file:s0001.md"
	}))
	if _, ok := h.rest.find("DeleteChannel", nil); ok {
		t.Fatal("the venue went before ten minutes")
	}
	h.until("the venue deleted", h.saw("DeleteChannel", func(c restCall) bool { return c.args[0] == venueID }))
	if _, ok := h.rest.find("DeleteChannel", func(c restCall) bool { return c.args[0] == "7777" }); ok {
		t.Error("an unknown session's channel was deleted")
	}
}

// TestLiveHeld: after a restart an interrupted session is held before the sweep (ADR 0026); the
// sweep binds its venue, then the coalescer sends the notice there, with no `has no venue`
// warning; a typed switch and a venue message get the interrupted refusal, and End session hands
// it off: the summary in the venue and the lobby, the transcript opening with the restart.
func TestLiveHeld(t *testing.T) {
	const venueID = "7400"
	h := newLiveHarnessHeld(t, func(root string, tx *store.Tx) error {
		// The tree the earlier run cloned is still there.
		if err := os.MkdirAll(filepath.Join(root, "sessions", "s0001", "tree"), 0o750); err != nil {
			return err
		}
		for _, q := range []string{
			`INSERT INTO registration (user_id, git_name, git_email, display_name, registered_at) VALUES ('1001', 'Ada Example', 'ada@example.com', '', '2026-10-21T00:00:00Z')`,
			`INSERT INTO session (rowid, id, channel_id, project_id, driver_id, state, harness, credential_id, mode, reach, started_at, ended_at)
VALUES (1, 's0001', '100', 'p', '1001', 'interrupted', 'fake', 'cred-x', 'policy', 'code', '2026-10-21T11:00:00Z', '2026-10-21T11:30:00Z')`,
			`INSERT INTO participant (session_id, user_id, role, joined_at) VALUES ('s0001', '1001', 'driver', '2026-10-21T11:00:00Z')`,
		} {
			if _, err := tx.Exec(q); err != nil {
				return err
			}
		}
		return nil
	}, textChannel(sf(venueID), "s0001-ada", "test:s0001", sf(testCat)))
	h.until("the notice", h.saw("CreateMessage", func(c restCall) bool {
		return c.args[0] == venueID && strings.HasPrefix(content(c), "Broodforge restarted; this session was interrupted. Its tree is kept. ")
	}))
	if strings.Contains(h.log.String(), "has no venue") {
		t.Errorf("a post went out before the sweep bound the venue: %s", h.log.String())
	}
	if _, ok := h.rest.find("CreateGuildChannel", nil); ok {
		t.Errorf("a venue was created for a held session\n%s", h.rest.dump())
	}

	// /session model in the venue and a plain message: the interrupted refusal, not `has ended`.
	h.g.onInteraction(interaction(t, slashJSON("21", venueID,
		`{"id":"1","name":"session","type":1,"options":[{"name":"model","type":1,"options":[{"name":"model","type":3,"value":"sonnet"}]}]}`)))
	if c, ok := h.rest.find("UpdateInteractionResponse", func(c restCall) bool { return c.args[1] == "tok-21" }); !ok ||
		!strings.Contains(c.body, interruptedRefusalText) {
		t.Errorf("/session model on a held session: %+v\n%s", c, h.rest.dump())
	}
	g := sf(testGuild)
	h.g.onMessage(dg.Message{ID: sf("61"), ChannelID: sf(venueID), GuildID: &g, Author: dg.User{ID: sf("1001")}, Content: "hi",
		Member: &dg.Member{RoleIDs: []snowflake.ID{sf("10")}}})
	if c, ok := h.rest.find("CreateMessage", func(c restCall) bool { return c.args[0] == "6000" }); !ok || content(c) != interruptedRefusalText {
		t.Errorf("a venue message on a held session: %+v", c)
	}

	// End session from the header the old process pinned.
	h.g.onInteraction(interaction(t, `{"id":"22","application_id":"`+testApp+`","type":3,"token":"tok-22","version":1,"guild_id":"`+testGuild+
		`","channel":{"id":"`+venueID+`","type":0,"guild_id":"`+testGuild+`","name":"c"},"member":{"user":{"id":"1001","username":"ada"},"roles":["10"]},`+
		`"message":{"id":"1","channel_id":"`+venueID+`","author":{"id":"5000"}},"data":{"custom_id":"bf:s0001:end","component_type":2}}`))
	h.until("the summary in the venue", h.saw("CreateMessage", func(c restCall) bool {
		return c.args[0] == venueID && strings.HasPrefix(content(c), "Session s0001 ended by Ada Example")
	}))
	h.until("the transcript", h.saw("UpdateMessage", func(c restCall) bool {
		return c.args[0] == testLobby && len(c.args) > 2 && c.args[2] == "file:s0001.md"
	}))
	if d, _ := h.svc.Show("s0001"); d.State != session.StateEnded {
		t.Errorf("after End session: %s", d.State)
	}
	if strings.Contains(h.log.String(), "warning:") {
		t.Errorf("warnings: %s", h.log.String())
	}
}

// TestSweepDeletesAnEndedVenue: a venue whose session ended before the bot started goes once
// ten minutes have passed since the end (04, also across a restart).
func TestSweepDeletesAnEndedVenue(t *testing.T) {
	h := newLiveHarness(t)
	if err := h.st.Write(func(tx *store.Tx) error {
		_, err := tx.Exec(`INSERT INTO registration (user_id, git_name, git_email, display_name, registered_at) VALUES ('1001', 'A', 'a@example.com', '', '2026-10-21T00:00:00Z')`)
		if err != nil {
			return err
		}
		_, err = tx.Exec(`INSERT INTO session (id, channel_id, project_id, driver_id, state, harness, credential_id, mode, reach, started_at, ended_at)
VALUES ('s0002', '100', 'p', '1001', 'ended', 'fake', 'cred-x', 'policy', 'code', '2026-10-21T11:00:00Z', '2026-10-21T11:55:00Z')`)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	h.rest.channels = []dg.GuildChannel{textChannel(sf("7100"), "s0002-a", "test:s0002", sf(testCat))}
	if err := h.g.sweep(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, ok := h.rest.find("DeleteChannel", nil); ok {
		t.Fatal("deleted before ten minutes from the end")
	}
	h.until("the ended venue deleted", h.saw("DeleteChannel", func(c restCall) bool { return c.args[0] == "7100" }))
}

// TestSweepTakesBase36IDs: a session id with a letter (s000a, the first after s0009) is a
// session id to the sweep: an open one's venue is bound again, an ended one's deleted.
func TestSweepTakesBase36IDs(t *testing.T) {
	h := newLiveHarness(t)
	if err := h.st.Write(func(tx *store.Tx) error {
		for _, q := range []string{
			`INSERT INTO registration (user_id, git_name, git_email, display_name, registered_at) VALUES ('1001', 'A', 'a@example.com', '', '2026-10-21T00:00:00Z')`,
			`INSERT INTO session (id, channel_id, project_id, driver_id, state, harness, credential_id, mode, reach, started_at)
VALUES ('s000a', '100', 'p', '1001', 'waiting-for-input', 'fake', 'cred-x', 'policy', 'code', '2026-10-21T11:00:00Z')`,
			`INSERT INTO session (id, channel_id, project_id, driver_id, state, harness, credential_id, mode, reach, started_at, ended_at)
VALUES ('s000b', '100', 'p', '1001', 'ended', 'fake', 'cred-x', 'policy', 'code', '2026-10-21T11:00:00Z', '2026-10-21T11:00:00Z')`,
		} {
			if _, err := tx.Exec(q); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	h.rest.channels = []dg.GuildChannel{
		textChannel(sf("7400"), "s000a-a", "test:s000a", sf(testCat)),
		textChannel(sf("7401"), "s000b-a", "test:s000b", sf(testCat)),
	}
	if err := h.g.sweep(context.Background()); err != nil {
		t.Fatal(err)
	}
	if ch := h.a.venueChannel("s000a", time.Second); ch != "7400" {
		t.Errorf("s000a's venue bound to %q, want 7400\n%s", ch, h.log.String())
	}
	h.until("s000b's venue deleted", h.saw("DeleteChannel", func(c restCall) bool { return c.args[0] == "7401" }))
	if _, ok := h.rest.find("DeleteChannel", func(c restCall) bool { return c.args[0] == "7400" }); ok {
		t.Error("the open session's venue was deleted")
	}
}

// TestRefusalWithoutDMs: a writer who takes no direct messages gets the refusal in the
// channel, mentioning them.
func TestRefusalWithoutDMs(t *testing.T) {
	h := newLiveHarness(t)
	h.until("the start-up report", h.saw("CreateMessage", nil))
	h.rest.noDM = true
	g := sf(testGuild)
	h.g.onMessage(dg.Message{ID: sf("31"), ChannelID: sf(testLobby), GuildID: &g, Author: dg.User{ID: sf("1001")}, Content: "sk-ant-x1234567"})
	if c, ok := h.rest.find("CreateMessage", func(c restCall) bool { return c.args[0] == testLobby }); !ok ||
		!strings.Contains(content(c), "<@1001> Never paste a key") {
		t.Errorf("channel refusal: %+v\n%s", c, h.rest.dump())
	}
	// The bot's own messages are never read.
	n := h.rest.count()
	h.g.onMessage(dg.Message{ID: sf("32"), ChannelID: sf(testLobby), GuildID: &g, Author: dg.User{ID: sf(testBot), Bot: true}, Content: "sk-ant-x1234567"})
	if h.rest.count() != n {
		t.Error("a bot message was handled")
	}
}

// TestOnEventRoutes: disgo's event handler hands messages and interactions on; others pass.
func TestOnEventRoutes(t *testing.T) {
	h := newLiveHarness(t)
	g := sf(testGuild)
	h.g.onEvent(nil, gateway.EventTypeMessageCreate, 1, gateway.EventMessageCreate{Message: dg.Message{ID: sf("41"),
		ChannelID: sf(testLobby), GuildID: &g, Author: dg.User{ID: sf("1001")}, Content: "sk-ant-x1234567"}})
	h.g.onEvent(nil, gateway.EventTypeTypingStart, 2, gateway.EventTypingStart{})
	h.g.wg.Wait()
	if _, ok := h.rest.find("DeleteMessage", func(c restCall) bool { return c.args[1] == "41" }); !ok {
		t.Errorf("message event not handled\n%s", h.rest.dump())
	}
}

// TestCheck: `discord check` over the fake: present and permitted, a permission short, absent,
// and the intent.
func TestCheck(t *testing.T) {
	r := &fakeRest{roles: []dg.Role{{ID: sf(testGuild), Permissions: lobbyPerms}},
		channels: []dg.GuildChannel{textChannel(sf(testLobby), "lobby", "", sf(testCat)), textChannel(sf(testCat), "sessions", "", 0)}}
	res, err := check(context.Background(), r, testGuild, []CheckChannel{{testLobby, "code"}, {"300", CheckNotice}, {testCat, CheckCategory}})
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(res)
	want := `{"guild":"3000","name":"Test","lines":[{"id":"100","state":"present","reach":"code","perms":"perms:ok"},` +
		`{"id":"300","state":"missing","reach":"notice","perms":"-"},` +
		`{"id":"900","state":"present","reach":"category","perms":"missing:add-reactions,manage-channels"},` +
		`{"id":"message-content","state":"present","reach":"intent","perms":"perms:ok"}],"missing":2}`
	if string(b) != want {
		t.Errorf("check\n got %s\nwant %s", b, want)
	}
	if _, err := check(context.Background(), r, "nope", nil); err == nil {
		t.Error("a guild that is not an id passed")
	}
}

// syncLog is the gateway's warning log, safe to read while it writes.
type syncLog struct {
	mu sync.Mutex
	b  strings.Builder
}

func (l *syncLog) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.Write(p)
}

func (l *syncLog) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.String()
}

// TestSweepNeedsTheMark: the sweep takes only a channel whose topic is this instance's mark for
// the session, and binds or deletes none when two carry the same mark.
func TestSweepNeedsTheMark(t *testing.T) {
	h := newLiveHarness(t)
	if err := h.st.Write(func(tx *store.Tx) error {
		for _, q := range []string{
			`INSERT INTO registration (user_id, git_name, git_email, display_name, registered_at) VALUES ('1001', 'A', 'a@example.com', '', '2026-10-21T00:00:00Z')`,
			`INSERT INTO session (id, channel_id, project_id, driver_id, state, harness, credential_id, mode, reach, started_at, ended_at)
VALUES ('s0002', '100', 'p', '1001', 'ended', 'fake', 'cred-x', 'policy', 'code', '2026-10-21T11:00:00Z', '2026-10-21T11:00:00Z')`,
			`INSERT INTO session (id, channel_id, project_id, driver_id, state, harness, credential_id, mode, reach, started_at)
VALUES ('s0003', '100', 'p', '1001', 'interrupted', 'fake', 'cred-x', 'policy', 'code', '2026-10-21T11:00:00Z')`,
		} {
			if _, err := tx.Exec(q); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	h.rest.channels = []dg.GuildChannel{
		textChannel(sf("7200"), "s0002", "", sf(testCat)),                // made by hand: no mark
		textChannel(sf("7201"), "s0002-a", "other:s0002", sf(testCat)),   // another instance's
		textChannel(sf("7300"), "s0003-real", "test:s0003", sf(testCat)), // two claim s0003
		textChannel(sf("7301"), "s0003-x", "test:s0003", sf(testCat)),
	}
	if err := h.g.sweep(context.Background()); err != nil {
		t.Fatal(err)
	}
	h.clock.Advance(VenueDeleteAfter + time.Minute)
	time.Sleep(50 * time.Millisecond)
	if _, ok := h.rest.find("DeleteChannel", nil); ok {
		t.Errorf("a channel without this instance's mark was deleted\n%s", h.rest.dump())
	}
	if ch := h.a.venueChannel("s0003", 0); ch != "s0003" {
		t.Errorf("s0003 was bound to %s with two channels claiming it", ch)
	}
	if !strings.Contains(h.log.String(), "warning: discord: the venue of s0003: channels 7300, 7301 all carry its topic") {
		t.Errorf("no warning: %q", h.log.String())
	}
}

// TestMessageEditScanned: an edit that adds a key is refused and deleted like a new message
// (04); an edit without one does nothing.
func TestMessageEditScanned(t *testing.T) {
	h := newLiveHarness(t)
	h.until("the start-up report", h.saw("CreateMessage", nil))
	g := sf(testGuild)
	edit := func(id, content string) {
		h.g.onEvent(nil, gateway.EventTypeMessageUpdate, 1, gateway.EventMessageUpdate{Message: dg.Message{ID: sf(id),
			ChannelID: sf(testLobby), GuildID: &g, Author: dg.User{ID: sf("1001")}, Content: content}})
		h.g.wg.Wait()
	}
	n := h.rest.count()
	edit("51", "fixed a typo")
	if h.rest.count() != n {
		t.Errorf("a plain edit made calls\n%s", h.rest.dump())
	}
	edit("52", "oops _sk-ant-api03-abcdefgh_")
	if _, ok := h.rest.find("DeleteMessage", func(c restCall) bool { return c.args[1] == "52" }); !ok {
		t.Errorf("the edited key was not deleted\n%s", h.rest.dump())
	}
	if c, ok := h.rest.find("CreateMessage", func(c restCall) bool { return c.args[0] == "6000" }); !ok || !strings.Contains(content(c), "Never paste a key") {
		t.Errorf("edit refusal: %+v", c)
	}
}

// fakeGateway is disgo's gateway without a socket: Open makes it ready.
type fakeGateway struct {
	gateway.Gateway
	mu     sync.Mutex
	status gateway.Status
}

func (f *fakeGateway) Open(context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.status = gateway.StatusReady
	return nil
}

func (f *fakeGateway) Status() gateway.Status {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.status
}

func (f *fakeGateway) Close(context.Context) {}

// TestOpenWhileServing: Open writes the connection and the bot's id while `status` reads
// Connected and the coalescer may create a venue; run under -race.
func TestOpenWhileServing(t *testing.T) {
	h := newLiveHarness(t)
	h.g.dial = func(string, gateway.EventHandlerFunc, ...gateway.ConfigOpt) gateway.Gateway { return &fakeGateway{} }
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case <-stop:
				return
			default:
				_ = h.g.Connected()
				_ = h.g.botID()
			}
		}
	}()
	err := h.g.Open(context.Background())
	close(stop)
	<-done
	if err != nil || !h.g.Connected() || h.g.botID() != sf(testBot) {
		t.Errorf("Open: %v, connected %v, bot %s", err, h.g.Connected(), h.g.botID())
	}
}
