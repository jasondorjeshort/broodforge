// SPDX-License-Identifier: GPL-3.0-or-later

package discord

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	dg "github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/snowflake/v2"
)

// The Discord half of `broodforge setup` (ADR 0027, Decision 3; decisions/0027-setup-commands.md,
// the steps token, application, guild, channels and role): SetupAPI is what internal/setup takes,
// in plain values so setup imports no Discord library. It is one implementation over setupREST,
// disgo's rest.Rest for a real token or the scripted fake of setupfake.go, so the permission
// arithmetic and `discord check` are the same code in both.

// setupREST is the REST calls setup and `discord check` make: part of restAPI, the one place a
// library swap must cover.
type setupREST interface {
	GetBotApplicationInfo(opts ...rest.RequestOpt) (*dg.Application, error)
	UpdateCurrentApplication(applicationUpdate dg.ApplicationUpdate, opts ...rest.RequestOpt) (*dg.Application, error)
	GetCurrentUserGuilds(bearerToken string, before snowflake.ID, after snowflake.ID, limit int, withCounts bool, opts ...rest.RequestOpt) ([]dg.OAuth2Guild, error)
	GetGuild(guildID snowflake.ID, withCounts bool, opts ...rest.RequestOpt) (*dg.RestGuild, error)
	GetRoles(guildID snowflake.ID, opts ...rest.RequestOpt) ([]dg.Role, error)
	CreateRole(guildID snowflake.ID, createRole dg.RoleCreate, opts ...rest.RequestOpt) (*dg.Role, error)
	GetMember(guildID snowflake.ID, userID snowflake.ID, opts ...rest.RequestOpt) (*dg.Member, error)
	AddMemberRole(guildID snowflake.ID, userID snowflake.ID, roleID snowflake.ID, opts ...rest.RequestOpt) error
	GetGuildChannels(guildID snowflake.ID, opts ...rest.RequestOpt) ([]dg.GuildChannel, error)
	CreateGuildChannel(guildID snowflake.ID, guildChannelCreate dg.GuildChannelCreate, opts ...rest.RequestOpt) (dg.GuildChannel, error)
}

// Application flags setup reads and writes: the message-content intent, switched on in the
// portal (1<<18) or by PATCH /applications/@me (1<<19, the limited form an unverified bot holds).
const (
	FlagMessageContent        = uint64(dg.ApplicationFlagGatewayMessageContent)
	FlagMessageContentLimited = uint64(dg.ApplicationFlagGatewayMessageContentLimited)
)

// PermManageRoles is Manage Roles, which the invite carries only when setup may create the role.
const PermManageRoles = uint64(dg.PermissionManageRoles)

// SetupApp is the application behind the token.
type SetupApp struct {
	ID, BotID string
	Flags     uint64
}

// SetupGuild is a guild the bot is in, with its owner.
type SetupGuild struct {
	ID, Name, OwnerID string
}

// SetupChannel is one channel of a guild.
type SetupChannel struct {
	ID, Name, ParentID string
	Category, Text     bool
}

// SetupRole is one role of a guild.
type SetupRole struct {
	ID, Name    string
	Position    int
	Permissions uint64
}

// SetupBot is the bot's member in a guild: its guild permissions (Discord's computation over its
// roles and @everyone; everything for an administrator) and its highest role's position.
type SetupBot struct {
	ID          string
	Permissions uint64
	Highest     int
	// Administrator: a role of the bot (or @everyone) has Administrator, which holds every
	// permission, Manage Roles among them.
	Administrator bool
}

// SetupAPI is the Discord calls `broodforge setup` makes, built for one step and closed after it,
// so the token is not held past the step.
type SetupAPI interface {
	// Application is GET /oauth2/applications/@me: the token's application and its bot.
	Application(ctx context.Context) (SetupApp, error)
	// EnableMessageContent is PATCH /applications/@me with flags ORed with 1<<19.
	EnableMessageContent(ctx context.Context, flags uint64) error
	// Guilds is GET /users/@me/guilds with the bot's own token.
	Guilds(ctx context.Context) ([]SetupGuild, error)
	// Guild is GET /guilds/{id}: its name and owner_id.
	Guild(ctx context.Context, guild string) (SetupGuild, error)
	Channels(ctx context.Context, guild string) ([]SetupChannel, error)
	CreateCategory(ctx context.Context, guild, name string) (SetupChannel, error)
	CreateTextChannel(ctx context.Context, guild, name, parent string) (SetupChannel, error)
	Roles(ctx context.Context, guild string) ([]SetupRole, error)
	CreateRole(ctx context.Context, guild, name string) (SetupRole, error)
	// MemberRoles is a member's role ids (GET /guilds/{id}/members/{user}).
	MemberRoles(ctx context.Context, guild, user string) ([]string, error)
	AddMemberRole(ctx context.Context, guild, user, role string) error
	// Bot is the bot's member: its guild permissions and highest role.
	Bot(ctx context.Context, guild string) (SetupBot, error)
	// Check is `discord check` over the same connection.
	Check(ctx context.Context, guild string, channels []CheckChannel) (CheckResult, error)
	Close()
}

// NewSetupAPI is SetupAPI over disgo's REST client with token, its shape checked first.
func NewSetupAPI(token string) (SetupAPI, error) {
	if _, err := CheckToken(token); err != nil {
		return nil, err
	}
	client := rest.NewClient(token)
	return &setupAPI{r: rest.New(client), close: func() { client.Close(context.Background()) }}, nil
}

type setupAPI struct {
	r     setupREST
	close func()
}

func (a *setupAPI) Close() {
	if a.close != nil {
		a.close()
	}
}

// SetupError is a call Discord answered with an error: its reason, as Discord words it.
type SetupError struct {
	Status int // the HTTP status, 0 when unknown
	Reason string
}

func (e *SetupError) Error() string { return e.Reason }

// Refused reports whether err is Discord's answer (a SetupError) rather than no answer.
func Refused(err error) (*SetupError, bool) {
	var se *SetupError
	ok := errors.As(err, &se)
	return se, ok
}

// setupErr turns disgo's REST error into a SetupError; anything else (no answer) passes.
func setupErr(err error) error {
	if err == nil {
		return nil
	}
	var re *rest.Error
	if !errors.As(err, &re) {
		return err
	}
	se := &SetupError{}
	if re.Response != nil {
		se.Status = re.Response.StatusCode
		se.Reason = re.Response.Status
	}
	if re.Message != "" {
		se.Reason = re.Message
		if re.Code != 0 {
			se.Reason = strconv.Itoa(int(re.Code)) + ": " + re.Message
		}
	}
	if se.Reason == "" {
		se.Reason = "Discord refused the call"
	}
	// A server error or a rate limit is Discord not answering yet, not a refusal: the step waits.
	if se.Status >= http.StatusInternalServerError || se.Status == http.StatusTooManyRequests {
		return &unavailableError{reason: se.Reason}
	}
	return se
}

// unavailableError is a 5xx or 429: no answer, for Refused.
type unavailableError struct{ reason string }

func (e *unavailableError) Error() string { return "Discord answered " + e.reason }

func parseID(what, id string) (snowflake.ID, error) {
	v, err := snowflake.Parse(id)
	if err != nil {
		return 0, fmt.Errorf("%s %q is not a Discord id", what, id)
	}
	return v, nil
}

func (a *setupAPI) Application(ctx context.Context) (SetupApp, error) {
	app, err := a.r.GetBotApplicationInfo(rest.WithCtx(ctx))
	if err != nil {
		return SetupApp{}, setupErr(err)
	}
	out := SetupApp{ID: app.ID.String(), Flags: uint64(app.Flags)}
	if app.Bot != nil {
		out.BotID = app.Bot.ID.String()
	}
	return out, nil
}

func (a *setupAPI) EnableMessageContent(ctx context.Context, flags uint64) error {
	f := dg.ApplicationFlags(flags | FlagMessageContentLimited)
	_, err := a.r.UpdateCurrentApplication(dg.ApplicationUpdate{Flags: &f}, rest.WithCtx(ctx))
	return setupErr(err)
}

func (a *setupAPI) Guilds(ctx context.Context) ([]SetupGuild, error) {
	gs, err := a.r.GetCurrentUserGuilds("", 0, 0, 200, false, rest.WithCtx(ctx))
	if err != nil {
		return nil, setupErr(err)
	}
	out := make([]SetupGuild, 0, len(gs))
	for _, g := range gs {
		out = append(out, SetupGuild{ID: g.ID.String(), Name: g.Name})
	}
	return out, nil
}

func (a *setupAPI) Guild(ctx context.Context, guild string) (SetupGuild, error) {
	gid, err := parseID("guild", guild)
	if err != nil {
		return SetupGuild{}, err
	}
	g, err := a.r.GetGuild(gid, false, rest.WithCtx(ctx))
	if err != nil {
		return SetupGuild{}, setupErr(err)
	}
	return SetupGuild{ID: g.ID.String(), Name: g.Name, OwnerID: g.OwnerID.String()}, nil
}

func setupChannel(c dg.GuildChannel) SetupChannel {
	out := SetupChannel{ID: c.ID().String(), Name: c.Name(), Category: c.Type() == dg.ChannelTypeGuildCategory,
		Text: c.Type() == dg.ChannelTypeGuildText}
	if p := c.ParentID(); p != nil && *p != 0 {
		out.ParentID = p.String()
	}
	return out
}

func (a *setupAPI) Channels(ctx context.Context, guild string) ([]SetupChannel, error) {
	gid, err := parseID("guild", guild)
	if err != nil {
		return nil, err
	}
	cs, err := a.r.GetGuildChannels(gid, rest.WithCtx(ctx))
	if err != nil {
		return nil, setupErr(err)
	}
	out := make([]SetupChannel, 0, len(cs))
	for _, c := range cs {
		out = append(out, setupChannel(c))
	}
	return out, nil
}

func (a *setupAPI) CreateCategory(ctx context.Context, guild, name string) (SetupChannel, error) {
	gid, err := parseID("guild", guild)
	if err != nil {
		return SetupChannel{}, err
	}
	c, err := a.r.CreateGuildChannel(gid, dg.GuildCategoryChannelCreate{Name: name}, rest.WithCtx(ctx), rest.WithReason("broodforge setup"))
	if err != nil {
		return SetupChannel{}, setupErr(err)
	}
	return setupChannel(c), nil
}

func (a *setupAPI) CreateTextChannel(ctx context.Context, guild, name, parent string) (SetupChannel, error) {
	gid, err := parseID("guild", guild)
	if err != nil {
		return SetupChannel{}, err
	}
	create := dg.GuildTextChannelCreate{Name: name}
	if parent != "" {
		if create.ParentID, err = parseID("category", parent); err != nil {
			return SetupChannel{}, err
		}
	}
	c, err := a.r.CreateGuildChannel(gid, create, rest.WithCtx(ctx), rest.WithReason("broodforge setup"))
	if err != nil {
		return SetupChannel{}, setupErr(err)
	}
	return setupChannel(c), nil
}

func setupRole(r dg.Role) SetupRole {
	return SetupRole{ID: r.ID.String(), Name: r.Name, Position: r.Position, Permissions: uint64(r.Permissions)}
}

func (a *setupAPI) Roles(ctx context.Context, guild string) ([]SetupRole, error) {
	gid, err := parseID("guild", guild)
	if err != nil {
		return nil, err
	}
	rs, err := a.r.GetRoles(gid, rest.WithCtx(ctx))
	if err != nil {
		return nil, setupErr(err)
	}
	out := make([]SetupRole, 0, len(rs))
	for _, r := range rs {
		out = append(out, setupRole(r))
	}
	return out, nil
}

// CreateRole creates a role with no permissions of its own: it only names the people the
// configuration trusts.
func (a *setupAPI) CreateRole(ctx context.Context, guild, name string) (SetupRole, error) {
	gid, err := parseID("guild", guild)
	if err != nil {
		return SetupRole{}, err
	}
	none := dg.Permissions(0)
	r, err := a.r.CreateRole(gid, dg.RoleCreate{Name: name, Permissions: &none}, rest.WithCtx(ctx), rest.WithReason("broodforge setup"))
	if err != nil {
		return SetupRole{}, setupErr(err)
	}
	return setupRole(*r), nil
}

func (a *setupAPI) MemberRoles(ctx context.Context, guild, user string) ([]string, error) {
	gid, err := parseID("guild", guild)
	if err != nil {
		return nil, err
	}
	uid, err := parseID("user", user)
	if err != nil {
		return nil, err
	}
	m, err := a.r.GetMember(gid, uid, rest.WithCtx(ctx))
	if err != nil {
		return nil, setupErr(err)
	}
	out := make([]string, 0, len(m.RoleIDs))
	for _, id := range m.RoleIDs {
		out = append(out, id.String())
	}
	return out, nil
}

func (a *setupAPI) AddMemberRole(ctx context.Context, guild, user, role string) error {
	gid, err := parseID("guild", guild)
	if err != nil {
		return err
	}
	uid, err := parseID("user", user)
	if err != nil {
		return err
	}
	rid, err := parseID("role", role)
	if err != nil {
		return err
	}
	return setupErr(a.r.AddMemberRole(gid, uid, rid, rest.WithCtx(ctx), rest.WithReason("broodforge setup")))
}

func (a *setupAPI) Bot(ctx context.Context, guild string) (SetupBot, error) {
	gid, err := parseID("guild", guild)
	if err != nil {
		return SetupBot{}, err
	}
	opt := rest.WithCtx(ctx)
	app, err := a.r.GetBotApplicationInfo(opt)
	if err != nil {
		return SetupBot{}, setupErr(err)
	}
	if app.Bot == nil {
		return SetupBot{}, errors.New("the token's application has no bot user")
	}
	m, err := a.r.GetMember(gid, app.Bot.ID, opt)
	if err != nil {
		return SetupBot{}, setupErr(err)
	}
	roles, err := a.r.GetRoles(gid, opt)
	if err != nil {
		return SetupBot{}, setupErr(err)
	}
	held := map[snowflake.ID]bool{}
	for _, id := range m.RoleIDs {
		held[id] = true
	}
	bot := SetupBot{ID: app.Bot.ID.String(), Permissions: uint64(channelPerms(gid, roles, m.RoleIDs, app.Bot.ID, nil))}
	for _, r := range roles {
		if (held[r.ID] || r.ID == gid) && r.Permissions.Has(dg.PermissionAdministrator) {
			bot.Administrator = true
		}
		if held[r.ID] && r.Position > bot.Highest {
			bot.Highest = r.Position
		}
	}
	return bot, nil
}

func (a *setupAPI) Check(ctx context.Context, guild string, channels []CheckChannel) (CheckResult, error) {
	return check(ctx, a.r, guild, channels)
}

// InviteURL is the OAuth2 link that adds the bot to a server: PermissionBits, ORed with Manage
// Roles when setup may create the role; a guild id preselects that server.
func InviteURL(applicationID string, manageRoles bool, guild string) string {
	perms := PermissionBits
	if manageRoles {
		perms |= PermManageRoles
	}
	q := "client_id=" + url.QueryEscape(applicationID) + "&scope=bot+applications.commands&permissions=" + strconv.FormatUint(perms, 10)
	if guild != "" {
		q += "&guild_id=" + url.QueryEscape(guild) + "&disable_guild_select=true"
	}
	return "https://discord.com/oauth2/authorize?" + q
}
