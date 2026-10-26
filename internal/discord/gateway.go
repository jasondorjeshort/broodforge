// SPDX-License-Identifier: GPL-3.0-or-later

package discord

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	dg "github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/gateway"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/snowflake/v2"

	"github.com/jasondorjeshort/broodforge/internal/policy"
	"github.com/jasondorjeshort/broodforge/internal/session"
)

// The live gateway (P12; 03 Startup 9 and 10; 05-debug-interface-discord-live.md): disgo's
// gateway client for the events and its REST client for every call, behind the handlers
// `discord inject` drives and the sink a dry run writes to the outbox. Every post still goes
// through the mask and the coalescer, lands in the outbox (so `session outbox` reads a live
// instance too), and is then sent by send. Resume, reconnects and the per-route rate limits are
// disgo's.

// restAPI is every REST call the live adapter makes, in one place: the surface a swap to
// arikawa must cover (ADR 0023). disgo's rest.Rest satisfies it; the tests' fake records.
type restAPI interface {
	// The start, `discord check` and `broodforge setup` (setup.go): the bot's application and
	// its flags, the bot's guilds, the guild, its roles and channels, the bot's member, creating
	// a channel (a session's venue too) and a role, assigning a role.
	setupREST
	// The guild's slash commands.
	SetGuildCommands(applicationID snowflake.ID, guildID snowflake.ID, commands []dg.ApplicationCommandCreate, opts ...rest.RequestOpt) ([]dg.ApplicationCommand, error)
	// Venues: a session's private channel (created through setupREST), deleted ten minutes
	// after its summary (04).
	DeleteChannel(channelID snowflake.ID, opts ...rest.RequestOpt) error
	// Posts: the outbox's create, edit, delete and attach, the pinned header and panel, the
	// hourglass on a queued message, and deleting a pasted key.
	CreateMessage(channelID snowflake.ID, messageCreate dg.MessageCreate, opts ...rest.RequestOpt) (*dg.Message, error)
	UpdateMessage(channelID snowflake.ID, messageID snowflake.ID, messageUpdate dg.MessageUpdate, opts ...rest.RequestOpt) (*dg.Message, error)
	DeleteMessage(channelID snowflake.ID, messageID snowflake.ID, opts ...rest.RequestOpt) error
	PinMessage(channelID snowflake.ID, messageID snowflake.ID, opts ...rest.RequestOpt) error
	AddReaction(channelID snowflake.ID, messageID snowflake.ID, emoji string, opts ...rest.RequestOpt) error
	// A reply to a plain message cannot be ephemeral: it goes as a direct message.
	CreateDMChannel(userID snowflake.ID, opts ...rest.RequestOpt) (*dg.DMChannel, error)
	// Interactions: the deferred answer, its edit or deletion, a click's ephemeral follow-up,
	// the API-key modal.
	CreateInteractionResponse(interactionID snowflake.ID, interactionToken string, interactionResponse dg.InteractionResponse, opts ...rest.RequestOpt) error
	UpdateInteractionResponse(applicationID snowflake.ID, interactionToken string, messageUpdate dg.MessageUpdate, opts ...rest.RequestOpt) (*dg.Message, error)
	DeleteInteractionResponse(applicationID snowflake.ID, interactionToken string, opts ...rest.RequestOpt) error
	CreateFollowupMessage(applicationID snowflake.ID, interactionToken string, messageCreate dg.MessageCreate, opts ...rest.RequestOpt) (*dg.Message, error)
}

// intents are the gateway intents the bot asks for: Guilds (the guild and its channels),
// GuildMessages (messages in the lobby and the venues) and MessageContent (their text; a
// privileged intent, switched on in the Developer Portal). DMs are not read: the relay is not
// built. IntentBits is the same as a number, for the operator entry.
const intents = gateway.IntentGuilds | gateway.IntentGuildMessages | gateway.IntentMessageContent

// IntentBits is intents as the integer the Developer Portal and the gateway use.
const IntentBits = uint64(intents)

// Permissions the bot needs, by where (`discord check` names them; the OAuth2 URL grants their
// union, PermissionBits). A venue gets the bot's own overwrite of venuePerms.
var permNames = []struct {
	name string
	bit  dg.Permissions
}{
	{"view", dg.PermissionViewChannel},
	{"send", dg.PermissionSendMessages},
	{"embed", dg.PermissionEmbedLinks},
	{"attach", dg.PermissionAttachFiles},
	{"add-reactions", dg.PermissionAddReactions},
	{"read-history", dg.PermissionReadMessageHistory},
	{"manage-messages", dg.PermissionManageMessages},
	{"pin", dg.PermissionPinMessages},
	{"manage-channels", dg.PermissionManageChannels},
}

const (
	// lobbyPerms: the panel (pinned), the summary with the transcript, deleting a pasted key.
	lobbyPerms = dg.PermissionViewChannel | dg.PermissionSendMessages | dg.PermissionEmbedLinks | dg.PermissionAttachFiles |
		dg.PermissionReadMessageHistory | dg.PermissionManageMessages | dg.PermissionPinMessages
	// noticePerms: the start-up report.
	noticePerms = dg.PermissionViewChannel | dg.PermissionSendMessages
	// venuePerms: everything a session's channel needs, granted to the bot in its overwrite.
	venuePerms = lobbyPerms | dg.PermissionAddReactions
	// categoryPerms: what the bot needs where venues are created, so it may grant itself
	// venuePerms there and delete the channel later.
	categoryPerms = venuePerms | dg.PermissionManageChannels
	// memberPerms: what a venue grants its driver and the roles that may contribute or answer.
	memberPerms = dg.PermissionViewChannel | dg.PermissionSendMessages | dg.PermissionReadMessageHistory |
		dg.PermissionAddReactions | dg.PermissionAttachFiles | dg.PermissionEmbedLinks
)

// PermissionBits is the permission integer of the OAuth2 URL that adds the bot to the server.
const PermissionBits = uint64(categoryPerms)

// VenueDeleteAfter is how long a private channel outlives its summary (04, Ending a session).
const VenueDeleteAfter = 10 * time.Minute

// venueWait bounds how long a start's reply waits for its venue to exist, so it can link it.
const venueWait = 10 * time.Second

// restTimeout bounds one REST call, so a hung request does not hold the coalescer for long.
const restTimeout = 30 * time.Second

// hourglass marks a message queued for the next turn (04, The turn loop).
const hourglass = "⏳"

// GatewayConfig is what the live gateway needs from `serve`.
type GatewayConfig struct {
	// Token is the bot token (the bot-token credential); it is never printed.
	Token string
	// ApplicationID, GuildID and NoticeChannel are discord.applicationId, guildId and
	// noticeChannelId of the instance configuration (NoticeChannel may be "").
	ApplicationID, GuildID, NoticeChannel string
	// Instance is instanceName: a venue's topic is `<instance>:<session id>`, the mark the sweep
	// requires, so a channel this bot did not make (by hand, or another instance's on the same
	// guild) is never bound or deleted.
	Instance string
	// Report is the start-up report for the notice channel (04, operator flow); "" posts none.
	Report string
	// Log receives one `warning: discord: ...` line per failed call; nil discards them.
	Log io.Writer
}

// Gateway is the live half of the adapter: it owns the REST client, the gateway connection,
// and the map from the adapter's message ids and session ids to Discord's.
type Gateway struct {
	a      *Adapter
	cfg    GatewayConfig
	rest   restAPI
	closer func(ctx context.Context) // the REST client's Close; nil in tests
	dial   func(token string, h gateway.EventHandlerFunc, opts ...gateway.ConfigOpt) gateway.Gateway
	logger *slog.Logger
	app    snowflake.ID
	guild  snowflake.ID
	notice snowflake.ID
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// mu guards everything below. gw and bot are written by Open while the control socket
	// already serves `status` and the coalescer may create a venue.
	mu     sync.Mutex
	gw     gateway.Gateway
	bot    snowflake.ID
	msgs   map[string]msgRef       // the adapter's message id to Discord's channel and message
	venues map[string]snowflake.ID // session id to its private channel
	timers map[string]session.Timer
}

type msgRef struct{ channel, message snowflake.ID }

// tokenRE is a Discord bot token's shape: three base64url parts joined by dots, the first the
// bot's id.
var tokenRE = regexp.MustCompile(`^[A-Za-z0-9_\-]+\.[A-Za-z0-9_\-]{4,}\.[A-Za-z0-9_\-]{20,}$`)

// CheckToken refuses a value that is not shaped like a bot token, before anything connects; it
// never repeats the value. It returns the bot's id the token's first part names.
func CheckToken(token string) (string, error) {
	bad := errors.New("the bot-token credential does not hold a Discord bot token (want three parts joined by dots, the first the bot's id in base64)")
	if !tokenRE.MatchString(token) {
		return "", bad
	}
	first := strings.SplitN(token, ".", 2)[0]
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(first, "="))
	if err != nil {
		if raw, err = base64.RawStdEncoding.DecodeString(strings.TrimRight(first, "=")); err != nil {
			return "", bad
		}
	}
	if _, err := snowflake.Parse(string(raw)); err != nil {
		return "", bad
	}
	return string(raw), nil
}

// NewGateway builds the live gateway for a, checks the token's shape and the configured ids,
// and makes a's sink send through it; Open connects. Call it before a.Run.
func NewGateway(a *Adapter, c GatewayConfig) (*Gateway, error) {
	if _, err := CheckToken(c.Token); err != nil {
		return nil, err
	}
	// The Warn level is what keeps the token out of the log: disgo logs the identify payload,
	// which carries the token, and every payload it receives at Debug.
	logger := slog.New(slog.NewTextHandler(logWriter(c.Log), &slog.HandlerOptions{Level: slog.LevelWarn}))
	client := rest.NewClient(c.Token, rest.WithLogger(logger))
	g, err := newGateway(a, c, rest.New(client))
	if err != nil {
		return nil, err
	}
	g.closer, g.logger = client.Close, logger
	a.live.Store(g)
	return g, nil
}

func logWriter(w io.Writer) io.Writer {
	if w == nil {
		return io.Discard
	}
	return w
}

func newGateway(a *Adapter, c GatewayConfig, r restAPI) (*Gateway, error) {
	g := &Gateway{a: a, cfg: c, rest: r, dial: gateway.New, msgs: map[string]msgRef{}, venues: map[string]snowflake.ID{},
		timers: map[string]session.Timer{}}
	var err error
	if g.app, err = snowflake.Parse(c.ApplicationID); err != nil {
		return nil, fmt.Errorf("discord.applicationId %q is not a Discord id", c.ApplicationID)
	}
	if g.guild, err = snowflake.Parse(c.GuildID); err != nil {
		return nil, fmt.Errorf("discord.guildId %q is not a Discord id", c.GuildID)
	}
	if c.NoticeChannel != "" {
		if g.notice, err = snowflake.Parse(c.NoticeChannel); err != nil {
			return nil, fmt.Errorf("discord.noticeChannelId %q is not a Discord id", c.NoticeChannel)
		}
	}
	g.ctx, g.cancel = context.WithCancel(context.Background())
	return g, nil
}

func (g *Gateway) warn(what string, err error) {
	if g.cfg.Log != nil && err != nil {
		fmt.Fprintf(g.cfg.Log, "warning: discord: %s: %s\n", what, oneLine(err.Error()))
	}
}

func oneLine(s string) string { return strings.Join(strings.Fields(s), " ") }

// Open is 03's startup step 9: the application behind the token, the guild's slash commands,
// the venues of earlier runs, then the gateway connection; and step 10's start-up report to
// the notice channel. A failure is the caller's start-discord refusal.
func (g *Gateway) Open(ctx context.Context) error {
	if err := g.prepare(ctx); err != nil {
		return err
	}
	opts := []gateway.ConfigOpt{gateway.WithIntents(intents)}
	if g.logger != nil {
		opts = append(opts, gateway.WithLogger(g.logger))
	}
	gw := g.dial(g.cfg.Token, g.onEvent, opts...)
	g.mu.Lock()
	g.gw = gw
	g.mu.Unlock()
	if err := gw.Open(ctx); err != nil {
		return fmt.Errorf("gateway: %w", err)
	}
	g.report()
	return nil
}

// prepare is Open without the websocket: what the tests drive against the fake REST client.
func (g *Gateway) prepare(ctx context.Context) error {
	opt := rest.WithCtx(ctx)
	app, err := g.rest.GetBotApplicationInfo(opt)
	if err != nil {
		return fmt.Errorf("the bot's application: %w", err)
	}
	if app.Bot == nil {
		return errors.New("the token's application has no bot user")
	}
	if app.ID != g.app {
		return fmt.Errorf("the token is application %s, discord.applicationId says %s", app.ID, g.app)
	}
	if !app.Flags.Has(dg.ApplicationFlagGatewayMessageContent) && !app.Flags.Has(dg.ApplicationFlagGatewayMessageContentLimited) {
		return errors.New("the Message Content intent is off: switch it on under Bot, Privileged Gateway Intents")
	}
	g.mu.Lock()
	g.bot = app.Bot.ID
	g.mu.Unlock()
	if _, err := g.rest.SetGuildCommands(g.app, g.guild, Commands(), opt); err != nil {
		return fmt.Errorf("registering the slash commands in guild %s: %w", g.guild, err)
	}
	return g.sweep(ctx)
}

// report posts the start-up report through the coalescer, so the outbox shows it too.
func (g *Gateway) report() {
	if g.notice == 0 || g.cfg.Report == "" {
		return
	}
	g.a.co.Offer(Post{Op: OpCreate, Channel: g.notice.String(), MessageID: g.a.newID(), Body: Mask(g.cfg.Report)})
}

// Connected reports whether the gateway session is ready (status's `discord: live connected`).
func (g *Gateway) Connected() bool {
	g.mu.Lock()
	gw := g.gw
	g.mu.Unlock()
	return gw != nil && gw.Status() == gateway.StatusReady
}

// botID is the bot's user id, known once Open has read the application.
func (g *Gateway) botID() snowflake.ID {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.bot
}

// Close disconnects the gateway and waits for the handlers it started.
func (g *Gateway) Close(ctx context.Context) {
	g.mu.Lock()
	gw := g.gw
	g.mu.Unlock()
	if gw != nil {
		gw.Close(ctx)
	}
	g.cancel()
	g.wg.Wait()
	g.mu.Lock()
	for id, t := range g.timers {
		t.Stop()
		delete(g.timers, id)
	}
	g.mu.Unlock()
	if g.closer != nil {
		g.closer(ctx)
	}
}

// venueTopic is the topic a venue is created with, the mark the sweep finds it by.
func (g *Gateway) venueTopic(id string) string { return g.cfg.Instance + ":" + id }

// sweep finds the venues earlier runs of this instance created: text channels under a
// project's venue.category whose topic is `<instance>:<session id>`. An open or interrupted
// session's is bound again, so its messages and /session end reach it; an ended one's is deleted
// VenueDeleteAfter after the session ended, now if that has passed (04: also across a restart).
// A channel without the mark, or whose session the store does not know, is left alone; when
// more than one channel carries a session's mark, none is bound or deleted and a warning names
// them.
func (g *Gateway) sweep(ctx context.Context) error {
	cats := map[snowflake.ID]bool{}
	if c := g.a.cfg.Config; c != nil {
		for _, p := range c().Projects {
			if id, err := snowflake.Parse(p.Venue.Category); err == nil {
				cats[id] = true
			}
		}
	}
	if len(cats) == 0 {
		return nil
	}
	chans, err := g.rest.GetGuildChannels(g.guild, rest.WithCtx(ctx))
	if err != nil {
		return fmt.Errorf("the guild's channels: %w", err)
	}
	now := g.a.cfg.Clock.Now()
	marked := map[string][]snowflake.ID{}
	for _, ch := range chans {
		parent := ch.ParentID()
		tc, ok := ch.(dg.GuildTextChannel)
		if !ok || parent == nil || !cats[*parent] || tc.Topic() == nil {
			continue
		}
		id, ok := strings.CutPrefix(*tc.Topic(), g.cfg.Instance+":")
		if !ok || g.cfg.Instance == "" || !session.ValidSessionID(id) {
			continue
		}
		marked[id] = append(marked[id], ch.ID())
	}
	sessions := make([]string, 0, len(marked))
	for id := range marked {
		sessions = append(sessions, id)
	}
	sort.Strings(sessions)
	for _, id := range sessions {
		chs := ids(marked[id])
		if len(chs) > 1 {
			g.warn("the venue of "+id, fmt.Errorf("channels %s all carry its topic; none is bound or deleted", strings.Join(chs, ", ")))
			continue
		}
		d, err := g.a.svc.Show(id)
		if err != nil {
			continue
		}
		ch := marked[id][0]
		g.mu.Lock()
		g.venues[d.ID] = ch
		g.mu.Unlock()
		if d.State != session.StateEnded {
			g.a.bindVenue(d.ID, ch.String())
			continue
		}
		left := VenueDeleteAfter
		if d.EndedAt != nil {
			if t, err := time.Parse(time.RFC3339Nano, *d.EndedAt); err == nil {
				left = t.Add(VenueDeleteAfter).Sub(now)
			}
		}
		g.scheduleVenueDelete(d.ID, left)
	}
	return nil
}

// scheduleVenueDelete deletes session id's venue after d on the adapter's clock.
func (g *Gateway) scheduleVenueDelete(id string, d time.Duration) {
	if d <= 0 {
		g.deleteVenue(id)
		return
	}
	t := g.a.cfg.Clock.AfterFunc(d, func() { g.deleteVenue(id) })
	g.mu.Lock()
	if old := g.timers[id]; old != nil {
		old.Stop()
	}
	g.timers[id] = t
	g.mu.Unlock()
}

func (g *Gateway) deleteVenue(id string) {
	g.mu.Lock()
	ch, ok := g.venues[id]
	delete(g.venues, id)
	delete(g.timers, id)
	g.mu.Unlock()
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(g.ctx, restTimeout)
	defer cancel()
	g.warn("deleting the venue of "+id, g.rest.DeleteChannel(ch, rest.WithCtx(ctx), rest.WithReason("session "+id+" ended")))
	g.a.unbindVenue(id)
}

// send is the live sink: one outbox post as the REST call it stands for. A failure is logged;
// the post stays in the outbox.
func (g *Gateway) send(p Post) {
	ctx, cancel := context.WithTimeout(g.ctx, restTimeout)
	defer cancel()
	opt := rest.WithCtx(ctx)
	what := string(p.Op) + " " + p.MessageID
	switch p.Op {
	case OpChannel:
		g.warn("creating the venue of "+p.Session, g.createVenue(p, opt))
	case OpCreate:
		ch, err := g.channelOf(p)
		if err != nil {
			g.warn(what, err)
			return
		}
		m, err := g.rest.CreateMessage(ch, messageCreate(p), opt)
		if err != nil {
			g.warn(what, err)
			return
		}
		g.mu.Lock()
		g.msgs[p.MessageID] = msgRef{ch, m.ID}
		g.mu.Unlock()
		if p.Pinned {
			g.warn("pin "+p.MessageID, g.rest.PinMessage(ch, m.ID, opt))
		}
	case OpEdit:
		if ref, ok := g.ref(p.MessageID, what); ok {
			_, err := g.rest.UpdateMessage(ref.channel, ref.message, messageUpdate(p), opt)
			g.warn(what, err)
		}
	case OpDelete:
		if ref, ok := g.ref(p.MessageID, what); ok {
			g.warn(what, g.rest.DeleteMessage(ref.channel, ref.message, opt))
			g.mu.Lock()
			delete(g.msgs, p.MessageID)
			g.mu.Unlock()
		}
	case OpAttach:
		if ref, ok := g.ref(p.MessageID, what); ok {
			_, err := g.rest.UpdateMessage(ref.channel, ref.message, attachUpdate(p), opt)
			g.warn(what, err)
		}
	}
}

// attachUpdate is an attach post's body: the file added to the message (04, Owner 11).
func attachUpdate(p Post) dg.MessageUpdate {
	return dg.MessageUpdate{Files: []*dg.File{dg.NewFile(p.File, "", strings.NewReader(p.Body))}}
}

func (g *Gateway) ref(id, what string) (msgRef, bool) {
	g.mu.Lock()
	ref, ok := g.msgs[id]
	g.mu.Unlock()
	if !ok {
		g.warn(what, errors.New("the message was never created"))
	}
	return ref, ok
}

// channelOf is the Discord channel of a post: a session's venue when the post names the
// session id as its channel (the View's venue), else the channel id it names.
func (g *Gateway) channelOf(p Post) (snowflake.ID, error) {
	if p.Session != "" && p.Channel == p.Session {
		g.mu.Lock()
		ch, ok := g.venues[p.Session]
		g.mu.Unlock()
		if !ok {
			return 0, errors.New(p.Session + " has no venue")
		}
		return ch, nil
	}
	return snowflake.Parse(p.Channel)
}

// createVenue creates a session's private channel (04, Venue A) under its project's
// venue.category, visible to the driver, the bot and the roles of the channel's contribute and
// answer lists, and binds it.
func (g *Gateway) createVenue(p Post, opt rest.RequestOpt) error {
	d, err := g.a.svc.Show(p.Session)
	if err != nil {
		return err
	}
	create, err := g.venueCreate(p, d.Channel, d.Driver)
	if err != nil {
		return err
	}
	ch, err := g.rest.CreateGuildChannel(g.guild, create, opt)
	if err != nil {
		return err
	}
	g.mu.Lock()
	g.venues[p.Session] = ch.ID()
	g.mu.Unlock()
	g.a.bindVenue(p.Session, ch.ID().String())
	return nil
}

// venueCreate is the private channel's request body for session p.Session started from lobby
// by driver.
func (g *Gateway) venueCreate(p Post, lobby, driver string) (dg.GuildTextChannelCreate, error) {
	c := g.a.cfg.Config()
	proj := c.ChannelProject(lobby)
	if proj == nil {
		return dg.GuildTextChannelCreate{}, errors.New("no project has channel " + lobby)
	}
	parent, err := snowflake.Parse(proj.Venue.Category)
	if err != nil {
		return dg.GuildTextChannelCreate{}, fmt.Errorf("project %s: venue.category %q is not a Discord id", proj.ID, proj.Venue.Category)
	}
	driverID, err := snowflake.Parse(driver)
	if err != nil {
		return dg.GuildTextChannelCreate{}, fmt.Errorf("driver %q is not a Discord id", driver)
	}
	e := c.Effective(proj, lobby)
	roles := roleIDs(append(append([]string{}, e.Roles.Contribute...), e.Roles.Answer...), g.a.cfg.Groups())
	return dg.GuildTextChannelCreate{
		Name:                 strings.TrimPrefix(p.First, "#"),
		Topic:                g.venueTopic(p.Session),
		ParentID:             parent,
		PermissionOverwrites: venueOverwrites(g.guild, g.botID(), driverID, roles),
	}, nil
}

// roleIDs resolves a role list (group names, raw role ids, "driver") to role ids, sorted and
// without repeats; "driver" is the driver's own overwrite, not a role.
func roleIDs(list []string, groups map[string][]string) []snowflake.ID {
	seen := map[snowflake.ID]bool{}
	add := func(s string) {
		if id, err := snowflake.Parse(s); err == nil {
			seen[id] = true
		}
	}
	for _, e := range list {
		if e == policy.RoleDriver {
			continue
		}
		if ids, ok := groups[e]; ok {
			for _, id := range ids {
				add(id)
			}
			continue
		}
		add(e)
	}
	out := make([]snowflake.ID, 0, len(seen))
	for id := range seen {
		out = append(out, id)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// venueOverwrites hide the channel from everyone (the @everyone role's id is the guild's) and
// open it to the bot, the driver and the roles given.
func venueOverwrites(guild, bot, driver snowflake.ID, roles []snowflake.ID) []dg.PermissionOverwrite {
	out := []dg.PermissionOverwrite{
		dg.RolePermissionOverwrite{RoleID: guild, Deny: dg.PermissionViewChannel},
		// Manage Channels too, so deleting the venue does not hang on a guild-wide grant.
		dg.MemberPermissionOverwrite{UserID: bot, Allow: venuePerms | dg.PermissionManageChannels},
		dg.MemberPermissionOverwrite{UserID: driver, Allow: memberPerms},
	}
	for _, r := range roles {
		out = append(out, dg.RolePermissionOverwrite{RoleID: r, Allow: memberPerms})
	}
	return out
}

// onEvent is disgo's event handler: a message or an interaction runs on its own goroutine, so
// a slow handler (a start clones a repository) never holds the gateway's read loop.
func (g *Gateway) onEvent(_ gateway.Gateway, t gateway.EventType, _ int, ev gateway.EventData) {
	if t == gateway.EventTypeMessageCreate {
		if m, ok := ev.(gateway.EventMessageCreate); ok {
			g.spawn(func() { g.onMessage(m.Message) })
		}
	} else if t == gateway.EventTypeMessageUpdate {
		if m, ok := ev.(gateway.EventMessageUpdate); ok {
			g.spawn(func() { g.onMessageEdit(m.Message) })
		}
	} else if t == gateway.EventTypeInteractionCreate {
		if i, ok := ev.(gateway.EventInteractionCreate); ok {
			g.spawn(func() { g.onInteraction(i.Interaction) })
		}
	}
}

func (g *Gateway) spawn(f func()) {
	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		f()
	}()
}

func ids(list []snowflake.ID) []string {
	out := make([]string, len(list))
	for i, id := range list {
		out[i] = id.String()
	}
	return out
}

// onMessage is a guild message (03 Data flow, step 1) through the adapter's message handler:
// the bot's own and other bots' messages, webhooks and other guilds are ignored. A pasted key
// is deleted where the bot may; a queued message gets the hourglass; a reply or a refusal goes
// to the writer privately.
func (g *Gateway) onMessage(m dg.Message) {
	in, ok := g.inboundMessage(m)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(g.ctx, restTimeout)
	defer cancel()
	opt := rest.WithCtx(ctx)
	rep, queued, err := g.a.message(g.ctx, in)
	var se *session.Error
	switch {
	case errors.As(err, &se):
		if se.Code == CodeKeyPasted {
			g.warn("deleting a pasted key", g.rest.DeleteMessage(m.ChannelID, m.ID, opt))
		}
		g.tell(m.Author.ID, m.ChannelID, se.Message, opt)
	case err != nil:
		g.warn("message", err)
	case rep.Reply == "ephemeral":
		g.tell(m.Author.ID, m.ChannelID, rep.Text, opt)
	case queued:
		g.warn("hourglass", g.rest.AddReaction(m.ChannelID, m.ID, hourglass, opt))
	}
}

// inboundMessage is a guild message as the handlers take it; false for one the bot ignores:
// its own and other bots' messages, webhooks, other guilds.
func (g *Gateway) inboundMessage(m dg.Message) (Inbound, bool) {
	if m.Author.Bot || m.WebhookID != nil || m.GuildID == nil || *m.GuildID != g.guild {
		return Inbound{}, false
	}
	in := Inbound{Channel: m.ChannelID.String(), User: m.Author.ID.String(), Text: m.Content}
	if m.Member != nil {
		in.Roles = ids(m.Member.RoleIDs)
	}
	return in, true
}

// onMessageEdit is an edited message: only the key-paste scan runs (04: a key pasted anywhere
// is refused and deleted), since an edit is never submitted again.
func (g *Gateway) onMessageEdit(m dg.Message) {
	in, ok := g.inboundMessage(m)
	if !ok {
		return
	}
	err := g.a.keyPasted(in, "message-edit")
	var se *session.Error
	if !errors.As(err, &se) {
		g.warn("message edit", err)
		return
	}
	ctx, cancel := context.WithTimeout(g.ctx, restTimeout)
	defer cancel()
	opt := rest.WithCtx(ctx)
	g.warn("deleting a pasted key", g.rest.DeleteMessage(m.ChannelID, m.ID, opt))
	g.tell(m.Author.ID, m.ChannelID, se.Message, opt)
}

// tell answers a plain message privately: Discord has no ephemeral reply to a message, so it
// goes as a direct message, and in the channel with a mention when the writer takes none.
func (g *Gateway) tell(user, channel snowflake.ID, text string, opt rest.RequestOpt) {
	body := dg.MessageCreate{Content: Mask(text), AllowedMentions: &dg.AllowedMentions{Users: []snowflake.ID{user}}}
	if dm, err := g.rest.CreateDMChannel(user, opt); err == nil {
		if _, err := g.rest.CreateMessage(dm.ID(), body, opt); err == nil {
			return
		}
	}
	body.Content = Mask(mention(user.String()) + " " + text)
	_, err := g.rest.CreateMessage(channel, body, opt)
	g.warn("a reply to "+user.String(), err)
}

// inbound is an interaction's caller and channel, with the roles held at that moment (N-18).
func inbound(i dg.Interaction) Inbound {
	in := Inbound{User: i.User().ID.String()}
	if ch := i.Channel(); ch.MessageChannel != nil {
		in.Channel = ch.ID().String()
	}
	if mem := i.Member(); mem != nil {
		in.Roles = ids(mem.RoleIDs)
	}
	return in
}

// ephemeralBody is an ephemeral interaction answer's body, masked, mentioning nobody.
func ephemeralBody(text string) dg.MessageCreate {
	return dg.MessageCreate{Content: Mask(text), Flags: dg.MessageFlagEphemeral, AllowedMentions: &dg.AllowedMentions{}}
}

// replyText is what the person sees of a handler's answer: an ephemeral reply's text, a
// refusal's message; "" for none (the effect shows in the venue).
func replyText(rep Reply, err error) (string, error) {
	var se *session.Error
	if errors.As(err, &se) {
		return se.Message, nil
	}
	if err != nil {
		return "", err
	}
	if rep.Reply == "ephemeral" {
		return rep.Text, nil
	}
	return "", nil
}

// onInteraction answers a slash command, a button, a header menu's pick or the API-key modal
// (03-architecture.md, Data flow: Custom ids) through the adapter's handlers. Every answer is deferred first (Discord wants one
// within three seconds; a start clones a repository), except the API-key form, which must be
// the first answer.
func (g *Gateway) onInteraction(i dg.Interaction) {
	ctx, cancel := context.WithTimeout(g.ctx, restTimeout)
	defer cancel()
	opt := rest.WithCtx(ctx)
	in := inbound(i)
	if gid := i.GuildID(); gid == nil || *gid != g.guild {
		g.warn("interaction", g.rest.CreateInteractionResponse(i.ID(), i.Token(), dg.InteractionResponse{
			Type: dg.InteractionResponseTypeCreateMessage, Data: ephemeralBody(otherGuildText)}, opt))
		return
	}
	t := i.Type()
	if t == dg.InteractionTypeApplicationCommand {
		ac, ok := i.(dg.ApplicationCommandInteraction)
		if !ok {
			return
		}
		data, ok := ac.Data.(dg.SlashCommandInteractionData)
		if !ok {
			return
		}
		sl, text, ok := slashOf(data)
		if !ok {
			g.answer(i, opt, Reply{}, g.a.refuse(in, "command", CodeBadArgs, "no command "+data.CommandPath()))
			return
		}
		in.Text = text
		if sl.Kind == SlashCredential && len(sl.Args) > 0 && sl.Args[0] == "add" && sl.Options["kind"] == kindAPIKey {
			rep, err := g.a.command(g.ctx, in, sl)
			if err == nil && rep.Reply == ReplyModal {
				g.warn("modal", g.rest.CreateInteractionResponse(i.ID(), i.Token(), dg.InteractionResponse{
					Type: dg.InteractionResponseTypeModal, Data: apiKeyModal(rep.Text)}, opt))
				return
			}
			text, err := replyText(rep, err)
			g.warn("credential add", err)
			g.warn("credential add", g.rest.CreateInteractionResponse(i.ID(), i.Token(), dg.InteractionResponse{
				Type: dg.InteractionResponseTypeCreateMessage, Data: ephemeralBody(text)}, opt))
			return
		}
		g.ack(i, opt, dg.InteractionResponseTypeDeferredCreateMessage)
		rep, err := g.a.command(g.ctx, in, sl)
		g.answer(i, opt, rep, err)
	} else if t == dg.InteractionTypeComponent {
		ci, ok := i.(dg.ComponentInteraction)
		if !ok {
			return
		}
		in.CustomID = ci.Data.CustomID()
		if sel, ok := ci.Data.(dg.StringSelectMenuInteractionData); ok {
			in.Values = sel.Values // a header menu's pick (P12b)
		}
		g.ack(i, opt, dg.InteractionResponseTypeDeferredUpdateMessage)
		rep, err := g.a.click(g.ctx, in)
		text, err := replyText(rep, err)
		g.warn("click", err)
		if text != "" {
			_, err := g.rest.CreateFollowupMessage(g.app, i.Token(), ephemeralBody(text), opt)
			g.warn("click follow-up", err)
		}
	} else if t == dg.InteractionTypeModalSubmit {
		mi, ok := i.(dg.ModalSubmitInteraction)
		if !ok || !IsAPIKeyModal(mi.Data.CustomID) {
			return
		}
		in.CustomID = mi.Data.CustomID
		g.ack(i, opt, dg.InteractionResponseTypeDeferredCreateMessage)
		rep, err := g.a.apiKeySubmitted(in, mi.Data.Text(APIKeyField))
		g.answer(i, opt, rep, err)
	}
}

// ack acknowledges an interaction: a deferred ephemeral answer, or for a click a deferred
// update that leaves the message as it is.
func (g *Gateway) ack(i dg.Interaction, opt rest.RequestOpt, t dg.InteractionResponseType) {
	resp := dg.InteractionResponse{Type: t}
	if t == dg.InteractionResponseTypeDeferredCreateMessage {
		resp.Data = dg.MessageCreate{Flags: dg.MessageFlagEphemeral}
	}
	g.warn("deferring an interaction", g.rest.CreateInteractionResponse(i.ID(), i.Token(), resp, opt))
}

// answer replaces a deferred answer with the handler's text, or deletes it when there is none.
func (g *Gateway) answer(i dg.Interaction, opt rest.RequestOpt, rep Reply, err error) {
	text, err := replyText(rep, err)
	g.warn("interaction", err)
	if text == "" {
		g.warn("interaction", g.rest.DeleteInteractionResponse(g.app, i.Token(), opt))
		return
	}
	content := Mask(text)
	_, err = g.rest.UpdateInteractionResponse(g.app, i.Token(), dg.MessageUpdate{Content: &content, AllowedMentions: &dg.AllowedMentions{}}, opt)
	g.warn("interaction", err)
}

// apiKeyModal is 04's form: one field, `Anthropic API key`.
func apiKeyModal(customID string) dg.ModalCreate {
	input := dg.NewShortTextInput(APIKeyField).WithRequired(true).WithMinLength(8).WithMaxLength(400)
	return dg.NewModalCreate(customID, apiKeyModalTitle, dg.NewLabel(apiKeyFieldLabel, input))
}

// slashOf is a slash command interaction as the Slash the handlers take, and its option values
// joined for the key-paste scan; false for a command this bot did not register.
func slashOf(d dg.SlashCommandInteractionData) (Slash, string, bool) {
	sub := ""
	if d.SubCommandName != nil {
		sub = *d.SubCommandName
	}
	opts := map[string]string{}
	var texts []string
	for name, o := range d.Options {
		v := strings.Trim(string(o.Value), `"`)
		if o.Type == dg.ApplicationCommandOptionTypeString {
			if s, ok := d.OptString(name); ok {
				v = s
			}
		}
		opts[name] = v
		texts = append(texts, v)
	}
	sort.Strings(texts)
	text := strings.Join(texts, " ")
	switch d.CommandName() {
	case "register":
		return Slash{Kind: SlashRegister, Name: "register", Options: opts}, text, sub == ""
	case "model":
		s := Slash{Kind: SlashModel, Name: "model", Options: map[string]string{}}
		if v, ok := opts["model"]; ok {
			s.Args = []string{v}
		}
		return s, text, sub == ""
	case "usage":
		return Slash{Kind: SlashUsage, Name: "usage", Options: opts}, text, sub == ""
	case "credential":
		return Slash{Kind: SlashCredential, Name: "credential", Args: []string{sub}, Options: opts}, text, sub != ""
	case "session":
		s := Slash{Name: "session " + sub, Options: map[string]string{}}
		switch sub {
		case "start":
			s.Kind = SlashSessionStart
			if v, ok := opts["model"]; ok {
				s.Options["model"] = v
			}
		case "stop":
			s.Kind = SlashSessionStop
		case "end":
			s.Kind = SlashSessionEnd
		case "mode":
			s.Kind, s.Args = SlashSessionMode, []string{opts["mode"]}
		case "model":
			s.Kind, s.Args = SlashSessionModel, []string{opts["model"]}
		case "effort":
			s.Kind, s.Args = SlashSessionEffort, []string{opts["effort"]}
		case "timeout":
			s.Kind, s.Args = SlashSessionTimeout, []string{opts["timeout"]}
		case "invite":
			s.Kind, s.Options["user"] = SlashSessionInvite, opts["user"]
		default:
			return Slash{}, text, false
		}
		return s, text, true
	}
	return Slash{}, text, false
}

// Commands are 04's slash commands as registered in the guild (Owner 1): /register,
// /session start|stop|end|mode|model|effort|timeout|invite, /credential add|list|remove|default|login,
// /usage, /model. The model and effort options are choice lists of policy's aliases and levels.
func Commands() []dg.ApplicationCommandCreate {
	str := func(name, desc string, max int, choices ...string) dg.ApplicationCommandOptionString {
		o := dg.ApplicationCommandOptionString{Name: name, Description: desc, Required: true}
		if max > 0 {
			o.MaxLength = &max
		}
		for _, c := range choices {
			o.Choices = append(o.Choices, dg.ApplicationCommandOptionChoiceString{Name: c, Value: c})
		}
		return o
	}
	sub := func(name, desc string, opts ...dg.ApplicationCommandOption) dg.ApplicationCommandOption {
		return dg.ApplicationCommandOptionSubCommand{Name: name, Description: desc, Options: opts}
	}
	id := str("id", "the credential id, cred-...", 40)
	model := str("model", "the model", 0, policy.ModelAliases()...)
	optional := func(o dg.ApplicationCommandOptionString) dg.ApplicationCommandOptionString {
		o.Required = false
		return o
	}
	return []dg.ApplicationCommandCreate{
		dg.SlashCommandCreate{Name: "register", Description: "Register the name and email your commits are authored as",
			Options: []dg.ApplicationCommandOption{str("name", "your name as git writes it", 100), str("email", "your email as git writes it", 254)}},
		dg.SlashCommandCreate{Name: "session", Description: "Start, stop, end or steer a Claude Code session",
			Options: []dg.ApplicationCommandOption{
				sub("start", "start a session in this project channel", optional(model)),
				sub("stop", "stop the running turn"),
				sub("end", "end this session and hand off its commits"),
				sub("mode", "switch from the next turn", str("mode", "asks first or acts freely", 0, "asks-first", "acts-freely")),
				sub("model", "switch the model from the next turn", model),
				sub("effort", "switch the effort from the next turn", str("effort", "the effort", 0, policy.Efforts...)),
				sub("timeout", "end this session after this long without activity",
					str("timeout", "never, minutes, or a duration such as 2h", 20)),
				sub("invite", "let someone into this session", dg.ApplicationCommandOptionUser{Name: "user", Description: "who", Required: true}),
			}},
		dg.SlashCommandCreate{Name: "credential", Description: "Your Claude credentials",
			Options: []dg.ApplicationCommandOption{
				sub("add", "add a credential", str("kind", "a Claude subscription or an API key", 0, kindSubscription, kindAPIKey), str("label", "a name for your list", MaxLabel)),
				sub("list", "list your credentials"),
				sub("remove", "remove a credential", id),
				sub("default", "use a credential for the sessions you start", id),
				sub("login", "log a subscription in", id),
			}},
		dg.SlashCommandCreate{Name: "usage", Description: "Your token usage"},
		dg.SlashCommandCreate{Name: "model", Description: "This session's model, or switch it from the next turn",
			Options: []dg.ApplicationCommandOption{optional(model)}},
	}
}
