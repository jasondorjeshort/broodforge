// SPDX-License-Identifier: GPL-3.0-or-later

package discord

import (
	"bytes"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	dg "github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/snowflake/v2"

	"github.com/jasondorjeshort/broodforge/internal/config"
	"github.com/jasondorjeshort/broodforge/internal/session"
)

// panelStore stands in for the store's panel table (serve's storedPanels and savePanel).
type panelStore struct {
	rows map[string]Panel
	fail error
}

func (s *panelStore) save(p Panel) error {
	if s.fail != nil {
		return s.fail
	}
	s.rows[p.Channel] = p
	return nil
}

func (s *panelStore) all() []Panel {
	var out []Panel
	for _, p := range s.rows {
		out = append(out, p)
	}
	return out
}

// panelAdapter is an adapter as serve builds it at start: the store's panels loaded. Its
// coalescer is flushed by hand (flushAll), not run.
func panelAdapter(st *panelStore) *Adapter {
	cfg := &config.Config{Projects: []config.Project{{ID: "p", Name: "Project P",
		Channels: map[string]config.Channel{testLobby: {Reach: config.ReachCode, Roles: config.Roles{Start: []string{"trusted"}}}}}}}
	return New(Config{Config: func() *config.Config { return cfg }, DryRun: true, Panels: st.all(), SavePanel: st.save,
		Clock: session.NewManualClock(time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC))})
}

func flushAll(a *Adapter) {
	now := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	for range 5 {
		a.co.Flush(now)
		now = now.Add(2 * time.Second)
	}
}

// TestPanelRestart: the panel's message id is recorded per project channel (03, Configuration),
// so a restarted bot shows it and edits it in place instead of posting a second panel, and its
// new messages never take the panel's id (P11 remainder, item 8). A failed save posts nothing.
func TestPanelRestart(t *testing.T) {
	st := &panelStore{rows: map[string]Panel{}}
	a := panelAdapter(st)
	a.newID() // some other message first, so the panel's id is not m1
	first, err := a.PostPanel(testLobby)
	if err != nil {
		t.Fatal(err)
	}
	flushAll(a)
	if st.rows[testLobby] != first || first.MessageID != "m2" {
		t.Fatalf("stored %+v, posted %+v", st.rows, first)
	}
	if posts := a.Outbox.Channel(testLobby); len(posts) != 1 || posts[0].Op != OpCreate || !posts[0].Pinned {
		t.Errorf("first post %+v", posts)
	}

	// The restart: a new adapter on the same store.
	b := panelAdapter(st)
	if p, ok := b.PanelOf(testLobby); !ok || p != first {
		t.Errorf("after the restart PanelOf = %+v %v", p, ok)
	}
	if id := b.newID(); id != "m3" {
		t.Errorf("the restarted bot's next id is %s, which may be the panel's", id)
	}
	again, err := b.PostPanel(testLobby)
	if err != nil {
		t.Fatal(err)
	}
	flushAll(b)
	posts := b.Outbox.Channel(testLobby)
	if again.MessageID != first.MessageID || len(posts) != 1 || posts[0].Op != OpEdit || posts[0].MessageID != first.MessageID {
		t.Errorf("after the restart: panel %+v, posts %+v", again, posts)
	}

	st.fail = errors.New("disk full")
	if _, err := b.PostPanel(testLobby); err == nil {
		t.Error("a failed save was not refused")
	}
	flushAll(b)
	if n := len(b.Outbox.Channel(testLobby)); n != 1 {
		t.Errorf("a failed save posted: %d posts", n)
	}
}

// TestPanelRestartLive: on a live instance the gateway records the Discord id of the panel it
// created, and after a restart edits that message; a stored panel without one (a dry run's, or
// one whose create never reached Discord) is created anew.
func TestPanelRestartLive(t *testing.T) {
	st := &panelStore{rows: map[string]Panel{}}
	live := func() (*Adapter, *fakeRest) {
		a := panelAdapter(st)
		r := &fakeRest{}
		g, err := newGateway(a, GatewayConfig{ApplicationID: testApp, GuildID: testGuild, Instance: "test"}, r)
		if err != nil {
			t.Fatal(err)
		}
		a.live.Store(g)
		return a, r
	}
	a, r := live()
	if _, err := a.PostPanel(testLobby); err != nil {
		t.Fatal(err)
	}
	flushAll(a)
	created, ok := r.find("CreateMessage", nil)
	if !ok {
		t.Fatalf("no create:\n%s", r.dump())
	}
	stored := st.rows[testLobby]
	if stored.DiscordID != "8001" || created.args[0] != testLobby {
		t.Fatalf("stored %+v after %v", stored, created.args)
	}

	b, r2 := live()
	if _, err := b.PostPanel(testLobby); err != nil {
		t.Fatal(err)
	}
	flushAll(b)
	if c, ok := r2.find("UpdateMessage", nil); !ok || c.args[0] != testLobby || c.args[1] != "8001" {
		t.Errorf("the restarted bot did not edit the panel:\n%s", r2.dump())
	}
	if _, ok := r2.find("CreateMessage", nil); ok {
		t.Errorf("the restarted bot posted a second panel:\n%s", r2.dump())
	}

	st.rows[testLobby] = Panel{Channel: testLobby, MessageID: "m1", PostedAt: "2026-10-29T00:00:00Z"}
	c, r3 := live()
	p, err := c.PostPanel(testLobby)
	if err != nil {
		t.Fatal(err)
	}
	flushAll(c)
	if _, ok := r3.find("CreateMessage", nil); !ok || p.MessageID == "m1" || st.rows[testLobby].DiscordID == "" {
		t.Errorf("a stored panel without a Discord id: panel %+v, stored %+v\n%s", p, st.rows[testLobby], r3.dump())
	}
}

// goneRest is the fake REST client after someone deleted message gone by hand: an edit of it is
// Discord's Unknown Message.
type goneRest struct {
	*fakeRest
	gone snowflake.ID
}

func (r *goneRest) UpdateMessage(c, m snowflake.ID, u dg.MessageUpdate, opts ...rest.RequestOpt) (*dg.Message, error) {
	if m == r.gone {
		r.record("UpdateMessage", "unknown", c.String(), m.String())
		return nil, &rest.Error{Response: &http.Response{StatusCode: http.StatusNotFound, Status: "404 Not Found"},
			Code: rest.JSONErrorCodeUnknownMessage, Message: "Unknown Message"}
	}
	return r.fakeRest.UpdateMessage(c, m, u, opts...)
}

// TestPanelDeletedByHand: a live panel whose message was deleted in Discord answers the edit with
// Unknown Message; the stored Discord id is cleared, so the next panel post creates it anew. A
// failed save of a created panel's Discord id is logged.
func TestPanelDeletedByHand(t *testing.T) {
	st := &panelStore{rows: map[string]Panel{testLobby: {Channel: testLobby, MessageID: "m1", PostedAt: "x", DiscordID: "7001"}}}
	a := panelAdapter(st)
	r := &goneRest{fakeRest: &fakeRest{}, gone: sf("7001")}
	var log bytes.Buffer
	g, err := newGateway(a, GatewayConfig{ApplicationID: testApp, GuildID: testGuild, Instance: "test", Log: &log}, r)
	if err != nil {
		t.Fatal(err)
	}
	a.live.Store(g)
	if _, err := a.PostPanel(testLobby); err != nil {
		t.Fatal(err)
	}
	flushAll(a)
	if p := st.rows[testLobby]; p.DiscordID != "" || p.MessageID != "m1" {
		t.Fatalf("after Unknown Message the store holds %+v\n%s", p, r.dump())
	}
	if _, ok := r.find("CreateMessage", nil); ok {
		t.Fatalf("created before the next post:\n%s", r.dump())
	}
	p, err := a.PostPanel(testLobby)
	if err != nil {
		t.Fatal(err)
	}
	flushAll(a)
	if c, ok := r.find("CreateMessage", nil); !ok || c.args[0] != testLobby || p.MessageID == "m1" || st.rows[testLobby].DiscordID == "" {
		t.Errorf("the next post: panel %+v, stored %+v\n%s", p, st.rows[testLobby], r.dump())
	}

	// The create's Discord id cannot be saved: a warning, the post itself done.
	st.rows = map[string]Panel{}
	b := panelAdapter(st)
	log.Reset()
	g2, err := newGateway(b, GatewayConfig{ApplicationID: testApp, GuildID: testGuild, Instance: "test", Log: &log}, &fakeRest{})
	if err != nil {
		t.Fatal(err)
	}
	b.live.Store(g2)
	if _, err := b.PostPanel(testLobby); err != nil {
		t.Fatal(err)
	}
	st.fail = errors.New("disk full")
	flushAll(b)
	if !strings.Contains(log.String(), "warning: discord: panel m1: disk full") {
		t.Errorf("log %q", log.String())
	}
}
