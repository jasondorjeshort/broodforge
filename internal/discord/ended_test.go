// SPDX-License-Identifier: GPL-3.0-or-later

package discord

import (
	"strconv"
	"strings"
	"testing"
	"time"

	dg "github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"

	"github.com/jasondorjeshort/broodforge/internal/config"
	"github.com/jasondorjeshort/broodforge/internal/session"
)

// TestViewMovesEnded: an ended session's venue is moved under venue.endedCategory and locked,
// after the summaries; with no category, neither (04-ux-session-end.md, Ending a session).
func TestViewMovesEnded(t *testing.T) {
	run := func(cat string) []Post {
		var posts []Post
		n := 0
		v := NewView("s0001", ViewConfig{NewID: func() string { n++; return "m" + strconv.Itoa(n) }, Name: mention,
			Offer: func(p Post) { posts = append(posts, p) }, EndedCategory: func(id, lobby string) string {
				if id != "s0001" || lobby != "100" {
					t.Errorf("ended category asked for %q, %q", id, lobby)
				}
				return cat
			}})
		v.Handle(ev(session.EventSessionStarted, 0, func(e *session.Event) { e.Start = testStart }))
		v.Handle(ev(session.EventSessionEnded, time.Minute, func(e *session.Event) {
			e.End = &session.EndInfo{By: "Jason", Outcome: "none"}
		}))
		return posts
	}
	posts := run("910")
	n := len(posts)
	if n < 3 || posts[n-3].Op != OpAttach {
		t.Fatalf("the move is not after the lobby's transcript: %+v", posts)
	}
	move, lock := posts[n-2], posts[n-1]
	move.finish()
	lock.finish()
	if move.Op != OpMove || move.Channel != "s0001" || move.MessageID != "s0001" || move.First != "910" ||
		string(move.Payload) != `{"parent_id":"910"}` {
		t.Errorf("move %+v payload %s", move, move.Payload)
	}
	if lock.Op != OpLock || lock.Channel != "s0001" || lock.First != lockFirst || lock.Payload != nil {
		t.Errorf("lock %+v", lock)
	}
	for _, p := range run("") {
		if p.Op == OpMove || p.Op == OpLock {
			t.Errorf("no category, yet %+v", p)
		}
	}
	// A category that is not a Discord id still goes to the outbox, with no payload; the live
	// gateway's send refuses it with a warning.
	posts = run("Ended sessions")
	if p := posts[len(posts)-2]; p.Op != OpMove || payload(p) != nil {
		t.Errorf("a non-id category: %+v", p)
	}
}

// TestLockedOverwrites: Send Messages denied to every overwrite but the bot's member one, the
// rest of each kept, and an @everyone deny added when the channel has none.
func TestLockedOverwrites(t *testing.T) {
	guild, bot, driver, role := snowflake.ID(3000), snowflake.ID(5000), snowflake.ID(1001), snowflake.ID(10)
	got := lockedOverwrites([]dg.PermissionOverwrite{
		dg.MemberPermissionOverwrite{UserID: bot, Allow: venuePerms | dg.PermissionManageChannels},
		dg.MemberPermissionOverwrite{UserID: driver, Allow: memberPerms},
		dg.RolePermissionOverwrite{RoleID: role, Allow: memberPerms, Deny: dg.PermissionAttachFiles},
	}, guild, bot)
	want := []dg.PermissionOverwrite{
		dg.MemberPermissionOverwrite{UserID: bot, Allow: venuePerms | dg.PermissionManageChannels},
		dg.MemberPermissionOverwrite{UserID: driver, Allow: memberPerms &^ dg.PermissionSendMessages, Deny: dg.PermissionSendMessages},
		dg.RolePermissionOverwrite{RoleID: role, Allow: memberPerms &^ dg.PermissionSendMessages,
			Deny: dg.PermissionAttachFiles | dg.PermissionSendMessages},
		dg.RolePermissionOverwrite{RoleID: guild, Deny: dg.PermissionSendMessages},
	}
	if len(got) != len(want) {
		t.Fatalf("got %+v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("overwrite %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
	// The venue's own overwrites (an @everyone one among them): none is added.
	if got := lockedOverwrites(venueOverwrites(guild, bot, driver, nil), guild, bot); len(got) != 3 {
		t.Errorf("a venue's lock: %+v", got)
	}
}

// TestEndedCategoryWarns: an empty venue.endedCategory beside a venue.category is "" and one
// warning line naming the session, the project and the setup step; with neither, no line.
func TestEndedCategoryWarns(t *testing.T) {
	var log strings.Builder
	cfg := &config.Config{Projects: []config.Project{{ID: "p", Channels: map[string]config.Channel{"100": {}}}}}
	a := New(Config{Config: func() *config.Config { return cfg }, Log: &log})
	defer a.Close()
	// The dry-run form: no category at all, nothing to say.
	if got := a.endedCategory("s0001", "100"); got != "" || log.Len() != 0 {
		t.Errorf("no categories: %q, log %q", got, log.String())
	}
	cfg.Projects[0].Venue.Category = "900"
	if got := a.endedCategory("s0001", "100"); got != "" {
		t.Errorf("empty key: %q", got)
	}
	want := "warning: discord: moving the venue of s0001: project p has no venue.endedCategory; " +
		"the channel stays under venue.category (broodforge setup channels makes the category)\n"
	if log.String() != want {
		t.Errorf("log %q", log.String())
	}
	log.Reset()
	cfg.Projects[0].Venue.EndedCategory = "910"
	if got := a.endedCategory("s0001", "100"); got != "910" || log.Len() != 0 {
		t.Errorf("set: %q, log %q", got, log.String())
	}
}
