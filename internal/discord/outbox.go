// SPDX-License-Identifier: GPL-3.0-or-later

package discord

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	dg "github.com/disgoorg/disgo/discord"
)

// Op is what one outbox post does: a REST call the gateway would make (P12).
type Op string

const (
	// OpChannel creates a session's private channel (04, Venue A).
	OpChannel Op = "channel"
	// OpCreate posts a new message.
	OpCreate Op = "create"
	// OpEdit replaces a message's content and buttons.
	OpEdit Op = "edit"
	// OpDelete deletes a message (04's delete-and-replace at a turn's end).
	OpDelete Op = "delete"
	// OpAttach attaches a file to a message (the transcript, 04 Owner 11).
	OpAttach Op = "attach"
)

// Button styles, Discord's.
const (
	StylePrimary   = "primary"
	StyleSecondary = "secondary"
	StyleSuccess   = "success"
	StyleDanger    = "danger"
)

// Button is one button of a post: its label and its custom id (03).
type Button struct {
	Label    string `json:"label"`
	CustomID string `json:"customId"`
	Style    string `json:"style"`
}

// Select is one of the header's select menus (04-ux-session-controls.md, The header's menus): a
// string select of one value, its own action row above the button row. Placeholder is "" for a
// menu with an option marked default.
type Select struct {
	CustomID    string   `json:"customId"`
	Placeholder string   `json:"placeholder,omitempty"`
	Options     []Option `json:"options"`
}

// Option is one option of a Select; Default marks the current value.
type Option struct {
	Label   string `json:"label"`
	Value   string `json:"value"`
	Default bool   `json:"default,omitempty"`
}

// Post is one outbox line: `<seq>  <time>  <op>  <message id>  <length>  <first line>`, --json
// every field. Payload is the request body the gateway would send for it, as disgo builds it
// (a MessageCreate or a MessageUpdate), so a dry run shows what P12 sends. Message ids are the
// adapter's own; the gateway maps them to Discord's.
type Post struct {
	Seq       int64           `json:"seq"`
	Time      string          `json:"time"`
	Op        Op              `json:"op"`
	Session   string          `json:"session,omitempty"`
	Channel   string          `json:"channel"`
	MessageID string          `json:"messageId"`
	Length    int             `json:"length"`
	First     string          `json:"first,omitempty"`
	Body      string          `json:"body,omitempty"`
	Buttons   []Button        `json:"buttons,omitempty"`
	Selects   []Select        `json:"selects,omitempty"`
	Pinned    bool            `json:"pinned,omitempty"`
	File      string          `json:"file,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

// OutboxColumns are a post's text columns.
var OutboxColumns = []string{"seq", "time", "op", "messageId", "length", "first"}

// TimeLayout is an outbox time: UTC to the millisecond, so the coalescer's spacing shows.
const TimeLayout = "2006-01-02T15:04:05.000Z07:00"

// finish fills the fields derived from the body: the length, the first line and the payload.
func (p *Post) finish() {
	p.Length = len([]rune(p.Body))
	switch p.Op {
	case OpChannel, OpAttach:
		// First is the channel name or the file name, set by the caller.
	case OpCreate, OpEdit, OpDelete:
		p.First = p.Body
		if i := strings.IndexByte(p.First, '\n'); i >= 0 {
			p.First = p.First[:i]
		}
	}
	p.Payload = payload(*p)
}

// payload is the request body for the post, built with disgo's types (ADR 0023: disgo's types
// appear in this package only); the live gateway sends the same bodies.
func payload(p Post) json.RawMessage {
	var b []byte
	var err error
	switch p.Op {
	case OpCreate:
		b, err = json.Marshal(messageCreate(p))
	case OpEdit:
		b, err = json.Marshal(messageUpdate(p))
	case OpChannel, OpDelete, OpAttach:
		return nil
	}
	if err != nil {
		return nil
	}
	return b
}

// messageCreate is a create post's body: the content, one action row per select menu in the
// post's order, the buttons as one action row under them, and user mentions only (a prompt's
// mention of the driver buzzes; no role or everyone ping).
func messageCreate(p Post) dg.MessageCreate {
	return dg.MessageCreate{Content: p.Body, Components: componentRows(p),
		AllowedMentions: &dg.AllowedMentions{Parse: []dg.AllowedMentionType{dg.AllowedMentionTypeUsers}}}
}

// messageUpdate is an edit post's body: the content and the components, an empty list removing
// them.
func messageUpdate(p Post) dg.MessageUpdate {
	content := p.Body
	rows := componentRows(p)
	if rows == nil {
		rows = []dg.LayoutComponent{}
	}
	return dg.MessageUpdate{Content: &content, Components: &rows}
}

// componentRows are a post's action rows: each select menu its own (min_values and max_values
// left at Discord's one), then the button row.
func componentRows(p Post) []dg.LayoutComponent {
	var rows []dg.LayoutComponent
	for _, s := range p.Selects {
		opts := make([]dg.StringSelectMenuOption, len(s.Options))
		for i, o := range s.Options {
			opts[i] = dg.NewStringSelectMenuOption(o.Label, o.Value).WithDefault(o.Default)
		}
		rows = append(rows, dg.NewActionRow(dg.NewStringSelectMenu(s.CustomID, s.Placeholder, opts...)))
	}
	return append(rows, buttonRows(p.Buttons)...)
}

func buttonRows(buttons []Button) []dg.LayoutComponent {
	var rows []dg.LayoutComponent
	if len(buttons) > 0 {
		bs := make([]dg.InteractiveComponent, len(buttons))
		for i, b := range buttons {
			style := dg.ButtonStylePrimary
			switch b.Style {
			case StyleSecondary:
				style = dg.ButtonStyleSecondary
			case StyleSuccess:
				style = dg.ButtonStyleSuccess
			case StyleDanger:
				style = dg.ButtonStyleDanger
			}
			bs[i] = dg.NewButton(style, b.Label, b.CustomID, "", 0)
		}
		rows = []dg.LayoutComponent{dg.NewActionRow(bs...)}
	}
	return rows
}

// Outbox is the record of every post the coalescer sends, in order, read by `session outbox`
// and `panel show`: the dry run's sink, and on a live instance the log of what the gateway sent. A follower waits on changed, closed and replaced on every append.
type Outbox struct {
	mu      sync.Mutex
	posts   []Post
	done    map[string]bool // sessions whose last post is out
	changed chan struct{}
}

// NewOutbox returns an empty outbox.
func NewOutbox() *Outbox { return &Outbox{done: map[string]bool{}, changed: make(chan struct{})} }

func (o *Outbox) add(now time.Time, p Post) Post {
	o.mu.Lock()
	defer o.mu.Unlock()
	p.Seq = int64(len(o.posts)) + 1
	p.Time = now.UTC().Format(TimeLayout)
	o.posts = append(o.posts, p)
	o.wakeLocked()
	return p
}

func (o *Outbox) wakeLocked() {
	close(o.changed)
	o.changed = make(chan struct{})
}

func (o *Outbox) markDone(id string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.done[id] = true
	o.wakeLocked()
}

// Posts is every post so far, of one session when id is not "".
func (o *Outbox) Posts(id string, since int64) []Post {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.postsLocked(id, since)
}

func (o *Outbox) postsLocked(id string, since int64) []Post {
	out := []Post{}
	for _, p := range o.posts {
		if p.Seq > since && (id == "" || p.Session == id) {
			out = append(out, p)
		}
	}
	return out
}

// Channel is every post to one channel.
func (o *Outbox) Channel(channel string) []Post {
	o.mu.Lock()
	defer o.mu.Unlock()
	out := []Post{}
	for _, p := range o.posts {
		if p.Channel == channel {
			out = append(out, p)
		}
	}
	return out
}

// Follow hands the session's posts after since to fn in order, waiting for new ones, until the
// session's last post is out, fn says stop or ctx ends.
func (o *Outbox) Follow(ctx context.Context, id string, since int64, fn func(Post) (bool, error)) error {
	for {
		o.mu.Lock()
		posts, changed, done := o.postsLocked(id, since), o.changed, o.done[id]
		o.mu.Unlock()
		for _, p := range posts {
			since = p.Seq
			if stop, err := fn(p); err != nil || stop {
				return err
			}
		}
		if done {
			return nil
		}
		select {
		case <-changed:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
