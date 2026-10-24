// SPDX-License-Identifier: GPL-3.0-or-later

package discord

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/jasondorjeshort/broodforge/internal/session"
)

// The header's menus (P12b; 04-ux-session-controls.md and 05-debug-interface-discord.md, The
// header's menus).

// selectsLine is a post's menus as `<custom id>[<placeholder>]=<default value>` joined by
// spaces, "-" for a menu with no default.
func selectsLine(p Post) string {
	var parts []string
	for _, s := range p.Selects {
		def := "-"
		for _, o := range s.Options {
			if o.Default {
				def = o.Value
			}
		}
		parts = append(parts, s.CustomID+"["+s.Placeholder+"]="+def)
	}
	return strings.Join(parts, " ")
}

// lastHeaderEdit is the last edit of the header message id in posts.
func lastHeaderEdit(t *testing.T, posts []Post, header string) Post {
	t.Helper()
	for i := len(posts) - 1; i >= 0; i-- {
		if posts[i].Op == OpEdit && posts[i].MessageID == header {
			return posts[i]
		}
	}
	t.Fatalf("no edit of the header %s in %+v", header, posts)
	return Post{}
}

// TestSelectHeaderMenus: the create carries the three menus in order with 04's placeholders and
// options, each switch's header edit marks the new value and keeps the others, a mode-changed
// edits the header as a model switch does, and the end's edit carries no components.
func TestSelectHeaderMenus(t *testing.T) {
	v, posts := newTestView()
	st := *testStart
	st.Mode, st.ChannelMode = "policy", "policy"
	v.Handle(ev(session.EventSessionStarted, 0, func(e *session.Event) { e.Start = &st }))
	create := (*posts)[1]
	if create.Op != OpCreate || !create.Pinned {
		t.Fatalf("header %+v", create)
	}
	if got, want := selectsLine(create), "bf:s0001:model[Model: the engine's own]=- bf:s0001:effort[Effort: the engine's own]=- "+
		"bf:s0001:mode[]=acts-freely"; got != want {
		t.Errorf("create's menus\n got %s\nwant %s", got, want)
	}
	var labels []string
	for _, s := range create.Selects {
		for _, o := range s.Options {
			labels = append(labels, o.Label+"="+o.Value)
		}
	}
	if got, want := strings.Join(labels, ","), "Opus=opus,Sonnet=sonnet,Haiku=haiku,Fable=fable,"+
		"low=low,medium=medium,high=high,xhigh=xhigh,max=max,asks first=asks-first,acts freely (within reach)=acts-freely"; got != want {
		t.Errorf("options\n got %s\nwant %s", got, want)
	}
	if len(create.Buttons) != 1 || create.Buttons[0].CustomID != "bf:s0001:end" {
		t.Errorf("create's buttons %+v", create.Buttons)
	}
	// The disgo payload: one action row per menu, in order, then the button row.
	var body struct {
		Components []struct {
			Type       int
			Components []struct {
				Type      int
				CustomID  string `json:"custom_id"`
				MinValues *int   `json:"min_values"`
				MaxValues *int   `json:"max_values"`
			}
		}
	}
	if err := json.Unmarshal(payload(create), &body); err != nil {
		t.Fatal(err)
	}
	var rows []string
	for _, r := range body.Components {
		for _, c := range r.Components {
			rows = append(rows, c.CustomID)
			if c.Type == 3 && (c.MinValues != nil && *c.MinValues != 1 || c.MaxValues != nil && *c.MaxValues != 1) {
				t.Errorf("%s min/max %v %v", c.CustomID, c.MinValues, c.MaxValues)
			}
		}
		if len(r.Components) != 1 {
			t.Errorf("a row of %d components", len(r.Components))
		}
	}
	if got := strings.Join(rows, " "); got != "bf:s0001:model bf:s0001:effort bf:s0001:mode bf:s0001:end" {
		t.Errorf("payload rows %s: %s", got, payload(create))
	}

	v.Handle(ev(session.EventEffortChanged, time.Second, func(e *session.Event) {
		e.Turn, e.Result, e.Summary = 1, "high", "From the next turn: effort high."
	}))
	if got := selectsLine(lastHeaderEdit(t, *posts, create.MessageID)); got !=
		"bf:s0001:model[Model: the engine's own]=- bf:s0001:effort[]=high bf:s0001:mode[]=acts-freely" {
		t.Errorf("after effort high: %s", got)
	}
	// A dated id belongs to its prefix's entry.
	v.Handle(ev(session.EventModelChanged, 2*time.Second, func(e *session.Event) {
		e.Turn, e.Result, e.Summary = 1, "claude-sonnet-5-5-20261001", "From the next turn: Sonnet."
	}))
	if got := selectsLine(lastHeaderEdit(t, *posts, create.MessageID)); got !=
		"bf:s0001:model[]=sonnet bf:s0001:effort[]=high bf:s0001:mode[]=acts-freely" {
		t.Errorf("after sonnet: %s", got)
	}
	n := len(*posts)
	v.Handle(ev(session.EventModeChanged, 3*time.Second, func(e *session.Event) { e.Turn, e.Result = 1, "prompt" }))
	if (*posts)[n].Op != OpCreate || !strings.HasPrefix((*posts)[n].Body, "From the next turn: asks first.") {
		t.Errorf("mode switch post %+v", (*posts)[n])
	}
	edit := lastHeaderEdit(t, (*posts)[n:], create.MessageID)
	if got := selectsLine(edit); got != "bf:s0001:model[]=sonnet bf:s0001:effort[]=high bf:s0001:mode[]=asks-first" {
		t.Errorf("after asks-first: %s", got)
	}
	if !strings.Contains(edit.Body, "\nMode: asks first (from turn 1).") || len(edit.Buttons) != 1 {
		t.Errorf("mode edit %+v", edit)
	}
	v.Handle(ev(session.EventSessionEnded, 4*time.Second, func(e *session.Event) {
		e.End = &session.EndInfo{Outcome: "nothing to push", By: "1001"}
	}))
	end := lastHeaderEdit(t, *posts, create.MessageID)
	if len(end.Selects) != 0 || len(end.Buttons) != 0 || !strings.Contains(end.Body, "Session s0001 on Broodforge") ||
		!strings.Contains(string(payload(end)), `"components":[]`) {
		t.Errorf("the end's header edit %+v %s", end, payload(end))
	}
}

// TestSelectModelPlaceholder: a model the map does not name marks no option and shows
// `Model: <id>`.
func TestSelectModelPlaceholder(t *testing.T) {
	v, posts := newTestView()
	st := *testStart
	st.Model, st.ChannelModel, st.Effort = "claude-opus-4-1-20250805", "claude-opus-4-1-20250805", "max"
	v.Handle(ev(session.EventSessionStarted, 0, func(e *session.Event) { e.Start = &st }))
	if got := selectsLine((*posts)[1]); got != "bf:s0001:model[Model: claude-opus-4-1-20250805]=- bf:s0001:effort[]=max bf:s0001:mode[]=asks-first" {
		t.Errorf("menus %s", got)
	}
}

// TestSelectResyncAndEnd: a Resync and a session-ended on one View, in either order, leave the
// header's last edit without components; a Resync before the end re-posts the header as it was.
func TestSelectResyncAndEnd(t *testing.T) {
	ended := ev(session.EventSessionEnded, time.Second, func(e *session.Event) { e.End = &session.EndInfo{Outcome: "nothing to push"} })
	for _, order := range []string{"resync-first", "end-first"} {
		t.Run(order, func(t *testing.T) {
			v, posts := newTestView()
			v.Handle(ev(session.EventSessionStarted, 0, func(e *session.Event) { e.Start = testStart }))
			create := (*posts)[1]
			if order == "resync-first" {
				v.Resync()
				re := (*posts)[len(*posts)-1]
				if re.Op != OpEdit || re.MessageID != create.MessageID || re.Body != create.Body ||
					selectsLine(re) != selectsLine(create) || len(re.Buttons) != 1 {
					t.Errorf("resync %+v", re)
				}
				v.Handle(ended)
			} else {
				v.Handle(ended)
				n := len(*posts)
				v.Resync()
				if len(*posts) != n {
					t.Errorf("a resync after the end posted %+v", (*posts)[n:])
				}
			}
			last := lastHeaderEdit(t, *posts, create.MessageID)
			if len(last.Selects) != 0 || len(last.Buttons) != 0 {
				t.Errorf("the header's last edit %+v", last)
			}
		})
	}
	// An interruption closes the View too.
	v, posts := newTestView()
	v.Handle(ev(session.EventSessionStarted, 0, func(e *session.Event) { e.Start = testStart }))
	v.Handle(ev(session.EventSessionInterrupted, time.Second, nil))
	n := len(*posts)
	v.Resync()
	if len(*posts) != n {
		t.Errorf("a resync after the interruption posted %+v", (*posts)[n:])
	}
}

// TestCoalesceSelects: an edit merged into a queued create or edit carries its components, the
// later ones winning: a switch's edit into the queued create, the end's edit into a queued
// switch's edit.
func TestCoalesceSelects(t *testing.T) {
	var sent []Post
	co := NewCoalescer(func(_ time.Time, p Post) { sent = append(sent, p) })
	t0 := time.Date(2026, 10, 24, 10, 0, 0, 0, time.UTC)
	menus := func(effort string) []Select {
		return []Select{{CustomID: "bf:s1:effort", Options: []Option{{Label: "high", Value: "high", Default: effort == "high"},
			{Label: "max", Value: "max", Default: effort == "max"}}}}
	}
	end := []Button{{CustomID: "bf:s1:end", Label: "End session", Style: "danger"}}
	// A switch inside the start's second: one create carrying the switch's menus.
	co.Offer(Post{Session: "s1", Op: OpCreate, MessageID: "m1", Body: "header", Buttons: end, Selects: menus("")})
	co.Offer(Post{Session: "s1", Op: OpEdit, MessageID: "m1", Body: "header high", Buttons: end, Selects: menus("high")})
	co.Flush(t0)
	if len(sent) != 1 || sent[0].Op != OpCreate || sent[0].Body != "header high" || selectsLine(sent[0]) != "bf:s1:effort[]=high" {
		t.Fatalf("the create %+v", sent)
	}
	// An end inside a switch's second: one edit, no components.
	co.Offer(Post{Session: "s1", Op: OpEdit, MessageID: "m1", Body: "header max", Buttons: end, Selects: menus("max")})
	co.Offer(Post{Session: "s1", Op: OpEdit, MessageID: "m1", Body: "header ended"})
	for now := t0.Add(time.Second); co.Pending() > 0; now = now.Add(time.Second) {
		co.Flush(now)
	}
	if len(sent) != 2 || sent[1].Op != OpEdit || sent[1].Body != "header ended" || len(sent[1].Selects) != 0 || len(sent[1].Buttons) != 0 {
		t.Errorf("the merged edit %+v", sent[1:])
	}
}

// componentsLine is a recorded message body's components as `<custom id>=<default value>`, the
// rows joined by " | " ("-" for a menu with no default, a button bare).
func componentsLine(t *testing.T, body string) string {
	t.Helper()
	var b struct {
		Components []struct {
			Components []struct {
				Type     int
				CustomID string `json:"custom_id"`
				Options  []struct {
					Value   string
					Default bool
				}
			}
		}
	}
	if err := json.Unmarshal([]byte(body), &b); err != nil {
		t.Fatalf("%s: %v", body, err)
	}
	var rows []string
	for _, r := range b.Components {
		var cs []string
		for _, c := range r.Components {
			if c.Type != 3 {
				cs = append(cs, c.CustomID)
				continue
			}
			def := "-"
			for _, o := range c.Options {
				if o.Default {
					def = o.Value
				}
			}
			cs = append(cs, c.CustomID+"="+def)
		}
		rows = append(rows, strings.Join(cs, " "))
	}
	return strings.Join(rows, " | ")
}

// all is every recorded call of name that match accepts, in order.
func (f *fakeRest) all(name string, match func(restCall) bool) []restCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []restCall
	for _, c := range f.calls {
		if c.name == name && match(c) {
			out = append(out, c)
		}
	}
	return out
}

// pickJSON is a string select interaction (a header menu's pick) from user in channel.
func pickJSON(id, channel, user, customID, value string) string {
	return `{"id":"` + id + `","application_id":"` + testApp + `","type":3,"token":"tok-` + id + `","version":1,"guild_id":"` + testGuild +
		`","channel":{"id":"` + channel + `","type":0,"guild_id":"` + testGuild + `","name":"c"},"member":{"user":{"id":"` + user +
		`","username":"u` + user + `"},"roles":["10"]},"message":{"id":"1","channel_id":"` + channel + `","author":{"id":"5000"}},` +
		`"data":{"custom_id":"` + customID + `","component_type":3,"values":["` + value + `"]}}`
}

// TestSelectLive: a recorded string-select interaction through onInteraction against the fake
// restAPI. The driver's pick is acknowledged with the deferred update, then the header's
// UpdateMessage marks the new value; a non-driver's pick gets the owner's ephemeral refusal as
// the follow-up, then an UpdateMessage of the unchanged header.
func TestSelectLive(t *testing.T) {
	h, venueID := liveHeader(t)
	isHeader := func(c restCall) bool {
		return c.args[0] == venueID && strings.HasPrefix(content(c), "Session s0001 on Project P")
	}
	create, ok := h.rest.find("CreateMessage", isHeader)
	if !ok {
		t.Fatalf("no header\n%s", h.rest.dump())
	}
	if got, want := componentsLine(t, create.body), "bf:s0001:model=- | bf:s0001:effort=- | bf:s0001:mode=acts-freely | bf:s0001:end"; got != want {
		t.Errorf("the header's components\n got %s\nwant %s", got, want)
	}

	// The driver picks effort high.
	h.g.onInteraction(interaction(t, pickJSON("31", venueID, "1001", "bf:s0001:effort", "high")))
	if c, ok := h.rest.find("CreateInteractionResponse", func(c restCall) bool { return c.args[1] == "tok-31" }); !ok || c.body != `{"type":6}` {
		t.Errorf("the pick's ack: %+v", c)
	}
	h.until("the header's edit", h.saw("UpdateMessage", func(c restCall) bool { return isHeader(c) && strings.Contains(content(c), "Effort: high.") }))
	edits := h.rest.all("UpdateMessage", isHeader)
	edit := edits[len(edits)-1]
	if got, want := componentsLine(t, edit.body), "bf:s0001:model=- | bf:s0001:effort=high | bf:s0001:mode=acts-freely | bf:s0001:end"; got != want {
		t.Errorf("the edit's components\n got %s\nwant %s", got, want)
	}
	if _, ok := h.rest.find("CreateFollowupMessage", func(c restCall) bool { return c.args[1] == "tok-31" }); ok {
		t.Error("a successful pick answered the picker")
	}
	h.until("the switch's post", h.saw("CreateMessage", func(c restCall) bool {
		return c.args[0] == venueID && content(c) == "From the next turn: effort high."
	}))

	// Bob, not the driver, picks max: the refusal only he sees, then the header unchanged.
	h.g.onInteraction(interaction(t, pickJSON("32", venueID, "1002", "bf:s0001:effort", "max")))
	if c, ok := h.rest.find("CreateFollowupMessage", func(c restCall) bool { return c.args[1] == "tok-32" }); !ok ||
		content(c) != "Only the driver, Ada Example, can change the effort." || !strings.Contains(c.body, `"flags":64`) {
		t.Errorf("the non-driver's follow-up: %+v\n%s", c, h.rest.dump())
	}
	h.until("the re-sync", func() bool { return len(h.rest.all("UpdateMessage", isHeader)) > len(edits) })
	resync := h.rest.all("UpdateMessage", isHeader)[len(edits)]
	if resync.body != edit.body {
		t.Errorf("the re-sync changed the header\n got %s\nwant %s", resync.body, edit.body)
	}
	if h.saw("CreateMessage", func(c restCall) bool { return c.args[0] == venueID && strings.Contains(content(c), "effort max") })() {
		t.Error("a refused pick posted in the venue")
	}
}

// TestSelectClosedByNonDriver: the closed check comes before the driver check, so a non-driver's
// late pick on an ended session gets 04's `This session has ended.` as the ephemeral follow-up,
// not the driver-only refusal, and the ended header is not edited again.
func TestSelectClosedByNonDriver(t *testing.T) {
	h, venueID := liveHeader(t)
	isHeader := func(c restCall) bool {
		return c.args[0] == venueID && strings.HasPrefix(content(c), "Session s0001 on Project P")
	}
	if _, err := h.svc.End(context.Background(), session.EndRequest{ActRequest: session.ActRequest{Session: "s0001", As: "1001"}}); err != nil {
		t.Fatal(err)
	}
	h.until("the end's header edit", h.saw("UpdateMessage", func(c restCall) bool {
		return isHeader(c) && strings.Contains(c.body, `"components":[]`)
	}))
	edits := len(h.rest.all("UpdateMessage", isHeader))

	h.g.onInteraction(interaction(t, pickJSON("41", venueID, "1002", "bf:s0001:effort", "max")))
	if c, ok := h.rest.find("CreateFollowupMessage", func(c restCall) bool { return c.args[1] == "tok-41" }); !ok ||
		content(c) != "This session has ended." || !strings.Contains(c.body, `"flags":64`) {
		t.Errorf("the non-driver's follow-up: %+v\n%s", c, h.rest.dump())
	}
	if n := len(h.rest.all("UpdateMessage", isHeader)); n != edits {
		t.Errorf("%d header edits after the late pick, %d before", n, edits)
	}
}

// liveHeader is a live harness with 1001 (the driver) and 1002 registered and a session s0001
// started by 1001's press of the panel's button: the harness and the venue's channel id, once the
// header is pinned.
func liveHeader(t *testing.T) (*liveHarness, string) {
	t.Helper()
	h := newLiveHarness(t)
	for _, u := range [][2]string{{"1001", "Ada Example"}, {"1002", "Bob Example"}} {
		if _, err := h.svc.Register(u[0], u[1], strings.ToLower(strings.Fields(u[1])[0])+"@example.com", session.ActorHost); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := h.svc.SetDefault("1001", "cred-x", session.ActorHost); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		h.g.onInteraction(interaction(t, `{"id":"30","application_id":"`+testApp+`","type":3,"token":"tok-30","version":1,"guild_id":"`+testGuild+
			`","channel":{"id":"100","type":0,"guild_id":"`+testGuild+`","name":"c"},"member":{"user":{"id":"1001","username":"ada"},"roles":["10"]},`+
			`"message":{"id":"1","channel_id":"100","author":{"id":"5000"}},"data":{"custom_id":"bf:100:start","component_type":2}}`))
	}()
	h.until("the venue", h.saw("CreateGuildChannel", nil))
	<-done
	venueID := h.a.venueChannel("s0001", 0)
	h.until("the pinned header", h.saw("PinMessage", func(c restCall) bool { return c.args[0] == venueID }))
	return h, venueID
}
