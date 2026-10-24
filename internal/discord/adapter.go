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

// The mode values /session mode and the mode menu take (04-ux-session-controls.md, Policy mode).
const (
	modeAsksFirst  = "asks-first"
	modeActsFreely = "acts-freely"
)

// The audit actions of the two routes a switch comes by: a typed command, a menu pick.
const (
	actionCommand = "command"
	actionClick   = "click"
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
	// Credentials serves /credential (04, Credentials); nil answers it not-built.
	Credentials Credentials
}

// Credentials is what /credential needs of the credential store, which the adapter does not
// import (03: discord talks to the session service, config and policy): `serve` adapts the
// credentials service to it. A refusal is a *session.Error with the store's code.
type Credentials interface {
	// Add records a credential of kind for user, audited with user as the actor; value is an
	// API key's, nil for a subscription.
	Add(user, kind, label string, value []byte) (CredentialInfo, error)
	// List is the user's credentials in the order added.
	List(user string) ([]CredentialInfo, error)
	// Remove marks one removed, audited with user as the actor.
	Remove(user, id string) (CredentialInfo, error)
}

// CredentialInfo is a credential as /credential list shows it: never a value.
type CredentialInfo struct {
	ID, Kind, Label, Status, LastUsedAt string
}

// Adapter is 03's discord component without the gateway (P11): it follows every session's
// events into a View, paces the posts through one Coalescer for the bot and writes them to the
// Outbox, and runs the inbound handlers `discord inject` feeds (command parsing, the key-paste
// scan, the role check, custom ids). The live gateway (gateway.go) feeds the same handlers and
// sends every post the outbox records through REST as well.
type Adapter struct {
	cfg    Config
	svc    *session.Service
	Outbox *Outbox
	co     *Coalescer
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	nextID atomic.Int64
	live   atomic.Pointer[Gateway] // set before Run on a live instance

	mu      sync.Mutex
	venues  map[string]string // venue channel id to session id
	venueOf map[string]string // session id to its venue channel id
	ready   map[string]chan struct{}
	ended   map[string]bool // sessions whose session-ended event the renderer saw
	panels  map[string]Panel
	// views is each followed session's View, from SessionStarted until its follow returns (the
	// end or the interruption rendered), for a refused pick's Resync.
	views map[string]*View
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
	a := &Adapter{cfg: cfg, Outbox: NewOutbox(), ctx: ctx, cancel: cancel, venues: map[string]string{},
		venueOf: map[string]string{}, ready: map[string]chan struct{}{}, ended: map[string]bool{}, panels: map[string]Panel{},
		views: map[string]*View{}}
	a.co = NewCoalescer(func(now time.Time, p Post) {
		p = a.Outbox.add(now, p)
		if g := a.live.Load(); g != nil {
			g.send(p)
		}
	})
	return a
}

// Attach sets the session service; serve builds the adapter first, because the service's
// Started hook is the adapter's.
func (a *Adapter) Attach(svc *session.Service) { a.svc = svc }

// Run starts the coalescer on the adapter's clock. A session whose last post is out is done in
// the outbox; on a live instance an ended one's venue is deleted VenueDeleteAfter later (04).
func (a *Adapter) Run() {
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		a.co.Run(a.ctx, a.cfg.Clock, func(id string) {
			a.Outbox.markDone(id)
			a.mu.Lock()
			ended := a.ended[id]
			// The session is done: its start was answered long ago, so its wait channel and
			// its ended mark go (the venue binding stays until the venue is deleted).
			delete(a.ended, id)
			delete(a.ready, id)
			a.mu.Unlock()
			if g := a.live.Load(); g != nil && ended {
				g.scheduleVenueDelete(id, VenueDeleteAfter)
			}
		})
	}()
}

// bindVenue records channel as session id's venue: in a dry run the session id itself, live
// the private channel the gateway created (or found at its start).
func (a *Adapter) bindVenue(id, channel string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.venues[channel] = id
	a.venueOf[id] = channel
	ch := a.ready[id]
	if ch == nil {
		ch = make(chan struct{})
		a.ready[id] = ch
	}
	select {
	case <-ch:
	default:
		close(ch)
	}
}

// unbindVenue forgets a deleted venue.
func (a *Adapter) unbindVenue(id string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if ch, ok := a.venueOf[id]; ok {
		delete(a.venues, ch)
	}
	delete(a.venueOf, id)
}

// venueChannel is session id's venue channel once bound, waiting up to wait for it.
func (a *Adapter) venueChannel(id string, wait time.Duration) string {
	a.mu.Lock()
	ch := a.ready[id]
	if ch == nil {
		ch = make(chan struct{})
		a.ready[id] = ch
	}
	a.mu.Unlock()
	select {
	case <-ch:
	case <-time.After(wait):
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if v, ok := a.venueOf[id]; ok {
		return v
	}
	return id
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
	if a.live.Load() == nil {
		a.bindVenue(id, id) // a dry run's venue is the session id; live, the gateway binds it
	}
	v := NewView(id, ViewConfig{NewID: a.newID, Name: a.name, Offer: a.co.Offer})
	a.mu.Lock()
	a.views[id] = v
	a.mu.Unlock()
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		_ = a.svc.Follow(a.ctx, id, 0, func(ev session.Event) (bool, error) {
			if ev.Type == session.EventSessionEnded {
				a.mu.Lock()
				a.ended[id] = true
				a.mu.Unlock()
			}
			v.Handle(ev)
			return false, nil
		})
		a.mu.Lock()
		delete(a.views, id)
		a.mu.Unlock()
		a.co.Done(id)
	}()
}

// viewOf is the session's View while its follow runs, nil otherwise.
func (a *Adapter) viewOf(id string) *View {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.views[id]
}

func (a *Adapter) sessionOf(channel string) string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.venues[channel]
}

// Inbound is one message or click as `discord inject` gives it: the roles stand in for the
// guild lookup at the moment of the action (N-18). Values is a select menu's pick (one value);
// a button has none.
type Inbound struct {
	Channel  string
	User     string
	Roles    []string
	Text     string
	CustomID string
	Values   []string
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
	rep, _, err := a.message(ctx, in)
	return rep, err
}

// message is a plain message's handler, the gateway's and inject's; queued says the session
// queued it for the next turn (04: the hourglass).
func (a *Adapter) message(ctx context.Context, in Inbound) (rep Reply, queued bool, err error) {
	if err := a.keyPasted(in, "message"); err != nil {
		return Reply{}, false, err
	}
	if sess := a.sessionOf(in.Channel); sess != "" {
		sub, err := a.svc.Submit(ctx, session.SendRequest{Session: sess, As: in.User, Roles: heldRoles(in.Roles), Text: in.Text})
		if err != nil {
			return Reply{}, false, a.venueRefusal(sess, err)
		}
		return none("turn " + strconv.Itoa(sub.Turn) + " " + sub.State), sub.State == session.SubmitQueued, nil
	}
	ch, ok := a.cfg.Channel(in.Channel)
	if !ok {
		return none("not a configured channel"), false, nil
	}
	if _, reg, err := a.svc.Registered(in.User); err != nil {
		return Reply{}, false, err
	} else if reg && policy.Permitted(in.Roles, ch.Roles.Start, a.cfg.Groups(), false) {
		return ephemeral(lobbyHintText), false, nil // Owner 2: a plain message never starts a session
	}
	return none("a plain message in the lobby from someone who may not start"), false, nil
}

// Command is a slash command, which Discord delivers as an interaction (in a dry run, `discord
// inject command` with the command typed as text): the key-paste scan over its options, then
// the command.
func (a *Adapter) Command(ctx context.Context, in Inbound) (Reply, error) {
	if !a.cfg.DryRun {
		return Reply{}, a.refuse(in, "command", CodeLiveInstance, "discord inject is for a dry-run instance")
	}
	sl, err := ParseSlash(in.Text)
	if err != nil {
		if kerr := a.keyPasted(in, "command"); kerr != nil {
			return Reply{}, kerr
		}
		msg := err.Error()
		if errors.Is(err, ErrNotSlash) {
			msg = "a command starts with /"
		}
		return Reply{}, a.refuse(in, "command", CodeBadArgs, msg)
	}
	return a.command(ctx, in, sl)
}

// command runs a parsed slash command, the gateway's interaction or inject's text: the
// key-paste scan over the text first (an interaction's text is its options joined).
func (a *Adapter) command(ctx context.Context, in Inbound, sl Slash) (Reply, error) {
	if err := a.keyPasted(in, "command"); err != nil {
		return Reply{}, err
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
	case session.CodeSessionClosed:
		return refusal(se.Code, sessionEndedText)
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
		model := sl.Options["model"]
		if _, ok := policy.ResolveModel(model); model != "" && !ok {
			return Reply{}, a.refuse(in, "command", CodeBadArgs, notAModelText(model))
		}
		return a.start(ctx, in, in.Channel, model)
	case SlashModel:
		if err := needVenue(); err != nil {
			return Reply{}, err
		}
		if len(sl.Args) == 0 {
			evs, err := a.svc.Events(sess, 0)
			if err != nil {
				return Reply{}, a.venueRefusal(sess, err)
			}
			return ephemeral(modelQueryText(headerOf(evs))), nil
		}
		return a.setModel(in, act, strings.Join(sl.Args, " "), actionCommand)
	case SlashSessionModel:
		if err := needVenue(); err != nil {
			return Reply{}, err
		}
		return a.setModel(in, act, strings.Join(sl.Args, " "), actionCommand)
	case SlashSessionEffort:
		if err := needVenue(); err != nil {
			return Reply{}, err
		}
		return a.setEffort(in, act, strings.Join(sl.Args, " "), actionCommand)
	case SlashSessionTimeout:
		if err := needVenue(); err != nil {
			return Reply{}, err
		}
		timeout := strings.Join(sl.Args, " ")
		if _, _, ok := session.ParseIdleTimeout(timeout); !ok {
			return Reply{}, a.refuse(in, "command", CodeBadArgs, "Not a timeout: "+timeout+". Give "+session.TimeoutText+".")
		}
		ev, err := a.svc.SetIdleTimeout(session.TimeoutRequest{ActRequest: act, Timeout: timeout})
		if err != nil {
			return Reply{}, a.venueRefusal(sess, err)
		}
		return none("idle timeout: " + ev.Result), nil
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
		return a.setMode(in, act, strings.Join(sl.Args, " "), actionCommand)
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
	case SlashCredential:
		return a.credential(in, sl)
	case SlashUsage:
	}
	return Reply{}, a.refuse(in, "command", CodeNotBuilt, "/"+sl.Name+" is not built in the dry-run adapter yet")
}

// Click is a button press or a header menu's pick (03-architecture.md, Data flow: Custom ids):
// the custom id parsed by the one function, then the service's answer, interrupt, end, start or
// switch for the presser.
func (a *Adapter) Click(ctx context.Context, in Inbound) (Reply, error) {
	if !a.cfg.DryRun {
		return Reply{}, a.refuse(in, "click", CodeLiveInstance, "discord inject is for a dry-run instance")
	}
	return a.click(ctx, in)
}

// click is a button press's handler, the gateway's and inject's.
func (a *Adapter) click(ctx context.Context, in Inbound) (Reply, error) {
	cid, err := ParseCustomID(in.CustomID)
	if err != nil {
		// Never the id itself, which the presser shapes: its length and prefix only.
		return Reply{}, a.refuse(in, "click", CodeBadArgs, "a custom id of "+strconv.Itoa(len(in.CustomID))+
			" bytes starting "+strconv.Quote(idPrefix(in.CustomID))+" is "+err.Error())
	}
	// A menu's pick carries exactly one value (Discord's min and max are one); a button none.
	// not 04: both texts.
	if cid.IsMenu() && len(in.Values) != 1 {
		return Reply{}, a.refuse(in, actionClick, CodeBadArgs, "a pick of "+cid.String()+" carries one value, not "+strconv.Itoa(len(in.Values)))
	}
	if !cid.IsMenu() && len(in.Values) != 0 {
		return Reply{}, a.refuse(in, actionClick, CodeBadArgs, cid.String()+" is a button and carries no value")
	}
	act := session.ActRequest{Session: cid.Session, As: in.User, Roles: heldRoles(in.Roles)}
	switch cid.Kind {
	case IDModel:
		return a.setModel(in, act, in.Values[0], actionClick)
	case IDEffort:
		return a.setEffort(in, act, in.Values[0], actionClick)
	case IDMode:
		return a.setMode(in, act, in.Values[0], actionClick)
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
		return a.start(ctx, in, cid.Channel, "")
	}
	return Reply{}, a.refuse(in, "click", CodeBadArgs, "unknown custom id")
}

// setModel is /model <alias>, /session model <alias> and the model menu's pick, action naming
// the route for the audit row of the adapter's own refusal: a value policy does not know is
// refused before the service sees it (04's `Not a model:`); the venue's `From the next turn:`
// line and the header edit come from the model-changed event.
func (a *Adapter) setModel(in Inbound, act session.ActRequest, value, action string) (Reply, error) {
	if _, ok := policy.ResolveModel(value); !ok {
		return Reply{}, a.refuse(in, action, CodeBadArgs, notAModelText(value))
	}
	ev, err := a.svc.SetModel(session.ModelRequest{ActRequest: act, Model: value})
	if err != nil {
		return Reply{}, a.switchRefusal(act.Session, "model", action, err)
	}
	return none("model from turn " + strconv.Itoa(ev.Turn) + ": " + ev.Result), nil
}

// setEffort is /session effort <level> and the effort menu's pick, as setModel.
func (a *Adapter) setEffort(in Inbound, act session.ActRequest, value, action string) (Reply, error) {
	if !policy.ParseEffort(value) {
		return Reply{}, a.refuse(in, action, CodeBadArgs, notAnEffortText(value))
	}
	ev, err := a.svc.SetEffort(session.EffortRequest{ActRequest: act, Effort: value})
	if err != nil {
		return Reply{}, a.switchRefusal(act.Session, "effort", action, err)
	}
	return none("effort from turn " + strconv.Itoa(ev.Turn) + ": " + value), nil
}

// setMode is /session mode asks-first|acts-freely and the mode menu's pick, as setModel.
func (a *Adapter) setMode(in Inbound, act session.ActRequest, value, action string) (Reply, error) {
	mode := ""
	switch value {
	case modeAsksFirst:
		mode = policy.ModePrompt.String()
	case modeActsFreely:
		mode = policy.ModePolicy.String()
	}
	if mode == "" {
		return Reply{}, a.refuse(in, action, CodeBadArgs, "/session mode takes asks-first or acts-freely")
	}
	ev, err := a.svc.SetMode(session.ModeRequest{ActRequest: act, Mode: mode})
	if err != nil {
		return Reply{}, a.switchRefusal(act.Session, "mode", action, err)
	}
	return none("mode from turn " + strconv.Itoa(ev.Turn) + ": " + mode), nil
}

// switchRefusal is a switch the service refused, in 04's words: the driver-only refusal as the
// owner worded it, the rest as venueRefusal has them. After a refused pick on a session still
// open, the header is posted again unchanged, so the picker's menu returns to the true value
// (05-debug-interface-discord.md, The header's menus); a closed session's header is left alone.
func (a *Adapter) switchRefusal(sess, what, action string, err error) error {
	var se *session.Error
	isSession := errors.As(err, &se)
	if action == actionClick && (!isSession || se.Code != session.CodeSessionClosed) {
		if v := a.viewOf(sess); v != nil {
			v.Resync()
		}
	}
	if isSession && se.Code == session.CodeRefused {
		// checkDriver's refusal is `<what> needs the driver, <name>`.
		if name, ok := strings.CutPrefix(se.Message, what+" needs the driver, "); ok {
			return refusal(se.Code, driverOnlyText(name, what))
		}
	}
	return a.venueRefusal(sess, err)
}

// start is the panel's button and /session start [model:<alias>] (04, The project channel and
// the panel): a refusal creates nothing and is answered in 04's words.
func (a *Adapter) start(ctx context.Context, in Inbound, channel, model string) (Reply, error) {
	if _, ok := a.cfg.Channel(channel); !ok {
		return Reply{}, a.refuse(in, "start", CodeUnknownChan, "/session start works in a project channel.") // not 04
	}
	st, err := a.svc.Start(ctx, session.StartRequest{Channel: channel, As: in.User, Roles: heldRoles(in.Roles), Model: model})
	if err == nil {
		wait := time.Duration(0)
		if a.live.Load() != nil {
			wait = venueWait // the gateway creates the private channel through the coalescer
		}
		return ephemeral(startedText(st.ID, a.venueChannel(st.ID, wait))), nil
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
