// SPDX-License-Identifier: GPL-3.0-or-later

package discord

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jasondorjeshort/broodforge/internal/config"
	"github.com/jasondorjeshort/broodforge/internal/policy"
	"github.com/jasondorjeshort/broodforge/internal/session"
)

// Codes the adapter refuses with besides the session service's.
const (
	CodeKeyPasted    = "key-pasted"
	CodeLiveInstance = session.CodeLiveInstance
	CodeBadArgs      = session.CodeBadArgs
	CodeNotVenue     = "not-a-venue"
	CodeNotBuilt     = "not-built"
	CodeUnknownChan  = session.CodeUnknownChannel
)

// Config is what the adapter needs from the bot.
type Config struct {
	// Config is the configuration as loaded now (the panel's project and role list).
	Config func() *config.Config
	// Channel is the session service's view of a configured channel.
	Channel func(id string) (session.Channel, bool)
	// Groups is the instance's roleGroups as loaded now.
	Groups func() map[string][]string
	// Clock paces the coalescer; nil is the wall clock.
	Clock session.Clock
	// DryRun is discord.mode dry-run: posts go to the outbox and `discord inject` is served.
	DryRun bool
}

// Adapter is 03's discord component without the gateway (P11): it follows every session's
// events into a View, paces the posts through one Coalescer for the bot and writes them to the
// Outbox, and runs the inbound handlers `discord inject` feeds (command parsing, the key-paste
// scan, the role check, custom ids). P12 puts the gateway behind the same handlers and sink.
type Adapter struct {
	cfg    Config
	svc    *session.Service
	Outbox *Outbox
	co     *Coalescer
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	nextID atomic.Int64

	mu     sync.Mutex
	venues map[string]string // venue channel id to session id
	panels map[string]Panel
}

// Panel is a project channel's panel message (04, The project channel and the panel).
type Panel struct {
	Channel   string `json:"channel"`
	MessageID string `json:"messageId"`
	PostedAt  string `json:"postedAt"`
}

// New returns an adapter; Attach gives it the session service and Run starts its pacing.
func New(cfg Config) *Adapter {
	if cfg.Clock == nil {
		cfg.Clock = session.RealClock{}
	}
	if cfg.Groups == nil {
		cfg.Groups = func() map[string][]string { return nil }
	}
	ctx, cancel := context.WithCancel(context.Background())
	a := &Adapter{cfg: cfg, Outbox: NewOutbox(), ctx: ctx, cancel: cancel, venues: map[string]string{}, panels: map[string]Panel{}}
	a.co = NewCoalescer(func(now time.Time, p Post) { a.Outbox.add(now, p) })
	return a
}

// Attach sets the session service; serve builds the adapter first, because the service's
// Started hook is the adapter's.
func (a *Adapter) Attach(svc *session.Service) { a.svc = svc }

// Run starts the coalescer on the adapter's clock.
func (a *Adapter) Run() {
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		a.co.Run(a.ctx, a.cfg.Clock, a.Outbox.markDone)
	}()
}

// Close stops the coalescer and every follower.
func (a *Adapter) Close() {
	a.cancel()
	a.wg.Wait()
}

func (a *Adapter) newID() string { return "m" + strconv.FormatInt(a.nextID.Add(1), 10) }

// name is a user's registered name, else their mention.
func (a *Adapter) name(user string) string {
	if a.svc != nil {
		if r, ok, err := a.svc.Registered(user); err == nil && ok {
			return r.Name
		}
	}
	return mention(user)
}

// SessionStarted is the session service's Started hook: a View follows the session's events
// from the first, on its own goroutine, until the session closes.
func (a *Adapter) SessionStarted(id string) {
	a.mu.Lock()
	a.venues[id] = id
	a.mu.Unlock()
	v := NewView(id, ViewConfig{NewID: a.newID, Name: a.name, Offer: a.co.Offer})
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		_ = a.svc.Follow(a.ctx, id, 0, func(ev session.Event) (bool, error) {
			v.Handle(ev)
			return false, nil
		})
		a.co.Done(id)
	}()
}

func (a *Adapter) sessionOf(channel string) string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.venues[channel]
}

// Inbound is one message or click as `discord inject` gives it: the roles stand in for the
// guild lookup at the moment of the action (N-18).
type Inbound struct {
	Channel  string
	User     string
	Roles    []string
	Text     string
	CustomID string
}

// Reply is what the bot answers: `ephemeral` with the text only the writer sees, or `none`
// with what happened in place of a reply.
type Reply struct {
	Reply string `json:"reply"`
	Text  string `json:"text"`
}

func ephemeral(text string) Reply { return Reply{Reply: "ephemeral", Text: text} }
func none(what string) Reply      { return Reply{Reply: "none", Text: what} }

func refusal(code, text string) error { return &session.Error{Code: code, Message: text} }

// refuse is a refusal of the adapter's own (a bad command or custom id, a command outside its
// channel, one not built, a live instance), audited `action-refused` like every refusal (04,
// Conventions), with the reason and the reply but never the inbound text.
func (a *Adapter) refuse(in Inbound, action, code, text string) error {
	err := a.svc.Refuse(in.User, a.sessionOf(in.Channel), action, code, code, text)
	var se *session.Error
	if errors.As(err, &se) && se.Code == code {
		return se
	}
	return err
}

// heldRoles is the role ids as the service's --roles: always checked, even when the person
// holds none (a lone comma is an empty list that is still given).
func heldRoles(roles []string) string {
	if len(roles) == 0 {
		return ","
	}
	return strings.Join(roles, ",")
}

// keyPasted is the key-paste scan (03 Data flow, step 1): the text is never submitted, the
// message is deleted where the bot may (P12: it needs the message's id), and the audit event
// carries the refusal, never the text.
func (a *Adapter) keyPasted(in Inbound, action string) error {
	if !HasKey(in.Text) {
		return nil
	}
	return a.refuse(in, action, CodeKeyPasted, keyPastedText)
}

// Message is a plain message arriving in a channel (03 Data flow, step 1): the key-paste scan
// first, then a venue message for its session, or the lobby's hint. A message is text even
// when it starts with `/` (a path): slash commands arrive as interactions (Command).
func (a *Adapter) Message(ctx context.Context, in Inbound) (Reply, error) {
	if !a.cfg.DryRun {
		return Reply{}, a.refuse(in, "message", CodeLiveInstance, "discord inject is for a dry-run instance")
	}
	if err := a.keyPasted(in, "message"); err != nil {
		return Reply{}, err
	}
	if sess := a.sessionOf(in.Channel); sess != "" {
		sub, err := a.svc.Submit(ctx, session.SendRequest{Session: sess, As: in.User, Roles: heldRoles(in.Roles), Text: in.Text})
		if err != nil {
			return Reply{}, a.venueRefusal(sess, err)
		}
		return none("turn " + strconv.Itoa(sub.Turn) + " " + sub.State), nil
	}
	ch, ok := a.cfg.Channel(in.Channel)
	if !ok {
		return none("not a configured channel"), nil
	}
	if _, reg, err := a.svc.Registered(in.User); err != nil {
		return Reply{}, err
	} else if reg && policy.Permitted(in.Roles, ch.Roles.Start, a.cfg.Groups(), false) {
		return ephemeral(lobbyHintText), nil // Owner 2: a plain message never starts a session
	}
	return none("a plain message in the lobby from someone who may not start"), nil
}

// Command is a slash command, which Discord delivers as an interaction (in a dry run, `discord
// inject command` with the command typed as text): the key-paste scan over its options, then
// the command.
func (a *Adapter) Command(ctx context.Context, in Inbound) (Reply, error) {
	if !a.cfg.DryRun {
		return Reply{}, a.refuse(in, "command", CodeLiveInstance, "discord inject is for a dry-run instance")
	}
	if err := a.keyPasted(in, "command"); err != nil {
		return Reply{}, err
	}
	sl, err := ParseSlash(in.Text)
	if err != nil {
		msg := err.Error()
		if errors.Is(err, ErrNotSlash) {
			msg = "a command starts with /"
		}
		return Reply{}, a.refuse(in, "command", CodeBadArgs, msg)
	}
	return a.slash(ctx, in, sl)
}

// venueRefusal is a refused venue action in 04's words.
func (a *Adapter) venueRefusal(sess string, err error) error {
	var se *session.Error
	if !errors.As(err, &se) {
		return err
	}
	switch se.Code {
	case session.CodeUnregistered:
		return refusal(se.Code, notRegisteredText)
	case session.CodeRefused:
		if strings.HasPrefix(se.Message, "contribute ") {
			lobby := ""
			if d, err := a.svc.Show(sess); err == nil {
				lobby = d.Channel
			}
			return refusal(se.Code, contributeRolesText(lobby))
		}
	}
	return se
}

// slash runs one of 04's slash commands.
func (a *Adapter) slash(ctx context.Context, in Inbound, sl Slash) (Reply, error) {
	sess := a.sessionOf(in.Channel)
	act := session.ActRequest{Session: sess, As: in.User, Roles: heldRoles(in.Roles)}
	needVenue := func() error {
		if sess == "" {
			return a.refuse(in, "command", CodeNotVenue, "/"+sl.Name+" works in a session's channel.") // not 04
		}
		return nil
	}
	switch sl.Kind {
	case SlashRegister:
		old, had, err := a.svc.Registered(in.User)
		if err != nil {
			return Reply{}, err
		}
		r, err := a.svc.Register(in.User, sl.Options["name"], sl.Options["email"], in.User)
		if err != nil {
			return Reply{}, err
		}
		if had {
			return ephemeral(updatedText(old.Name, old.Email, r.Name, r.Email)), nil
		}
		return ephemeral(registeredText(r.Name, r.Email)), nil
	case SlashSessionStart:
		if sess != "" {
			return Reply{}, a.refuse(in, "command", CodeNotVenue, "/session start works in a project channel.") // not 04
		}
		return a.start(ctx, in, in.Channel)
	case SlashSessionStop:
		if err := needVenue(); err != nil {
			return Reply{}, err
		}
		ev, err := a.svc.Interrupt(ctx, act)
		if err != nil {
			return Reply{}, a.venueRefusal(sess, err)
		}
		return none("turn " + strconv.Itoa(ev.Turn) + " " + ev.Result), nil
	case SlashSessionEnd:
		if err := needVenue(); err != nil {
			return Reply{}, err
		}
		if _, err := a.svc.End(ctx, session.EndRequest{ActRequest: act}); err != nil {
			return Reply{}, a.venueRefusal(sess, err)
		}
		return none(sess + " ended"), nil
	case SlashSessionMode:
		if err := needVenue(); err != nil {
			return Reply{}, err
		}
		mode := ""
		if len(sl.Args) == 1 {
			switch sl.Args[0] {
			case "asks-first":
				mode = policy.ModePrompt.String()
			case "acts-freely":
				mode = policy.ModePolicy.String()
			}
		}
		if mode == "" {
			return Reply{}, a.refuse(in, "command", CodeBadArgs, "/session mode takes asks-first or acts-freely")
		}
		ev, err := a.svc.SetMode(session.ModeRequest{ActRequest: act, Mode: mode})
		if err != nil {
			return Reply{}, a.venueRefusal(sess, err)
		}
		return none("mode from turn " + strconv.Itoa(ev.Turn) + ": " + mode), nil
	case SlashSessionInvite:
		if err := needVenue(); err != nil {
			return Reply{}, err
		}
		user, ok := UserOption(sl.Options["user"])
		if !ok {
			return Reply{}, a.refuse(in, "command", CodeBadArgs, "/session invite takes user:@<name>")
		}
		r, err := a.svc.Invite(session.InviteRequest{ActRequest: act, User: user})
		if err != nil {
			return Reply{}, a.venueRefusal(sess, err)
		}
		return ephemeral(r.Name + " was invited."), nil // not 04
	case SlashCredential, SlashUsage:
	}
	return Reply{}, a.refuse(in, "command", CodeNotBuilt, "/"+sl.Name+" is not built in the dry-run adapter yet")
}

// Click is a button press (03 Data flow, Button custom ids): the custom id parsed by the one
// function, then the service's answer, interrupt, end or start for the presser.
func (a *Adapter) Click(ctx context.Context, in Inbound) (Reply, error) {
	if !a.cfg.DryRun {
		return Reply{}, a.refuse(in, "click", CodeLiveInstance, "discord inject is for a dry-run instance")
	}
	cid, err := ParseCustomID(in.CustomID)
	if err != nil {
		// Never the id itself, which the presser shapes: its length and prefix only.
		return Reply{}, a.refuse(in, "click", CodeBadArgs, "a custom id of "+strconv.Itoa(len(in.CustomID))+
			" bytes starting "+strconv.Quote(idPrefix(in.CustomID))+" is "+err.Error())
	}
	act := session.ActRequest{Session: cid.Session, As: in.User, Roles: heldRoles(in.Roles)}
	switch cid.Kind {
	case IDAnswer:
		sub, err := a.svc.Answer(ctx, session.AnswerRequest{ActRequest: act, ToolUseID: cid.ToolUseID, Decision: cid.Decision})
		if err != nil {
			return Reply{}, a.venueRefusal(cid.Session, err)
		}
		return none("turn " + strconv.Itoa(sub.Turn) + " " + sub.State), nil
	case IDStop:
		ev, err := a.svc.Interrupt(ctx, act)
		if err != nil {
			return Reply{}, a.venueRefusal(cid.Session, err)
		}
		return none("turn " + strconv.Itoa(ev.Turn) + " " + ev.Result), nil
	case IDEnd:
		if _, err := a.svc.End(ctx, session.EndRequest{ActRequest: act}); err != nil {
			return Reply{}, a.venueRefusal(cid.Session, err)
		}
		return none(cid.Session + " ended"), nil
	case IDStart:
		return a.start(ctx, in, cid.Channel)
	}
	return Reply{}, a.refuse(in, "click", CodeBadArgs, "unknown custom id")
}

// start is the panel's button and /session start (04, The project channel and the panel):
// a refusal creates nothing and is answered in 04's words.
func (a *Adapter) start(ctx context.Context, in Inbound, channel string) (Reply, error) {
	if _, ok := a.cfg.Channel(channel); !ok {
		return Reply{}, a.refuse(in, "start", CodeUnknownChan, "/session start works in a project channel.") // not 04
	}
	st, err := a.svc.Start(ctx, session.StartRequest{Channel: channel, As: in.User, Roles: heldRoles(in.Roles)})
	if err == nil {
		return ephemeral(startedText(st.ID)), nil
	}
	var se *session.Error
	if !errors.As(err, &se) {
		return Reply{}, err
	}
	switch se.Code {
	case session.CodeUnregistered:
		return Reply{}, refusal(se.Code, notRegisteredText)
	case session.CodeRefused:
		return Reply{}, refusal(se.Code, startRolesText(channel))
	case session.CodeNoCredential:
		return Reply{}, refusal(se.Code, couldNotStartText("you have no default credential; add one with /credential add")) // not 04
	case session.CodeCredentialStatus:
		id, status, _ := strings.Cut(se.Message, " is ")
		reason := "your default credential " + id + " needs a login"
		if status == "removed" {
			reason = id + " was removed; set another with /credential default"
		}
		return Reply{}, refusal(se.Code, couldNotStartText(reason))
	case "worktree-create":
		return Reply{}, refusal(se.Code, couldNotStartText("cloning the repository failed on the host: "+firstOf(splitLines(se.Message))))
	}
	return Reply{}, refusal(se.Code, couldNotStartText(se.Message))
}

// PostPanel posts or refreshes a project channel's panel (04; `panel post <channel id>`): the
// first post creates and pins it, a later one edits it in place.
func (a *Adapter) PostPanel(channel string) (Panel, error) {
	c := a.cfg.Config()
	p := c.ChannelProject(channel)
	if p == nil {
		return Panel{}, refusal(CodeUnknownChan, "no channel "+channel)
	}
	e := c.Effective(p, channel)
	body := panelText(p.Name, p.HandoffOrDefault(), e.Roles.Start)
	buttons := []Button{{Label: "Start session", CustomID: CustomID{Kind: IDStart, Channel: channel}.String(), Style: StylePrimary}}
	a.mu.Lock()
	defer a.mu.Unlock()
	panel, ok := a.panels[channel]
	op := OpEdit
	if !ok {
		panel, op = Panel{Channel: channel, MessageID: a.newID()}, OpCreate
	}
	panel.PostedAt = a.cfg.Clock.Now().UTC().Format(time.RFC3339)
	a.panels[channel] = panel
	a.co.Offer(Post{Op: op, Channel: channel, MessageID: panel.MessageID, Body: Mask(body), Buttons: buttons, Pinned: op == OpCreate})
	return panel, nil
}

// PanelOf is the channel's panel, false when none was posted while the bot runs.
func (a *Adapter) PanelOf(channel string) (Panel, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	p, ok := a.panels[channel]
	return p, ok
}

// Transcript is the session's transcript from its events so far (`session transcript`).
func (a *Adapter) Transcript(id string) (string, error) {
	evs, err := a.svc.Events(id, 0)
	if err != nil {
		return "", err
	}
	v := NewView(id, ViewConfig{Name: a.name})
	for _, ev := range evs {
		if ev.Type == session.EventSessionStarted {
			v.start = ev.Start
		}
	}
	return Transcript(id, evs, v.who), nil
}

// idPrefix is what a refusal may show of a custom id: up to its first colon, at most 8 bytes.
func idPrefix(s string) string {
	if i := strings.IndexByte(s, ':'); i >= 0 {
		s = s[:i]
	}
	if len(s) > 8 {
		s = s[:8]
	}
	return strings.ToValidUTF8(s, "")
}
