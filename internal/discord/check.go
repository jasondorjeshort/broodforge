// SPDX-License-Identifier: GPL-3.0-or-later

package discord

import (
	"context"
	"fmt"
	"strings"

	dg "github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/snowflake/v2"
)

// `discord check` (05-debug-interface-discord.md, The Discord adapter): the bot token against
// the guild, over REST only (no gateway session), every configured channel present and the
// bot's permissions in it computed from the guild's roles and the channel's overwrites.

// Reaches of a checked channel besides a project channel's own (code, read-only).
const (
	CheckNotice   = "notice"
	CheckCategory = "category"
	CheckIntent   = "intent"
)

// CheckChannel is one channel to check: a project channel with its reach, the notice channel,
// or a project's venue.category.
type CheckChannel struct {
	ID, Reach string
}

// CheckLine is one result line: `<id>  present|missing  <reach>  perms:ok|missing:<names>`.
type CheckLine struct {
	ID    string `json:"id"`
	State string `json:"state"`
	Reach string `json:"reach"`
	Perms string `json:"perms"`
}

// CheckColumns are a line's text columns.
var CheckColumns = []string{"id", "state", "reach", "perms"}

// CheckResult is the guild and one line per channel, then the message-content intent's line.
type CheckResult struct {
	Guild   string      `json:"guild"`
	Name    string      `json:"name"`
	Lines   []CheckLine `json:"lines"`
	Missing int         `json:"missing"`
}

// Check connects with token once, checks guild and channels, and disconnects. The token is
// checked for its shape first and never printed.
func Check(ctx context.Context, token, guild string, channels []CheckChannel) (CheckResult, error) {
	if _, err := CheckToken(token); err != nil {
		return CheckResult{}, err
	}
	client := rest.NewClient(token)
	defer client.Close(ctx)
	return check(ctx, rest.New(client), guild, channels)
}

func check(ctx context.Context, r setupREST, guild string, channels []CheckChannel) (CheckResult, error) {
	gid, err := snowflake.Parse(guild)
	if err != nil {
		return CheckResult{}, fmt.Errorf("guild %q is not a Discord id", guild)
	}
	opt := rest.WithCtx(ctx)
	app, err := r.GetBotApplicationInfo(opt)
	if err != nil {
		return CheckResult{}, fmt.Errorf("the bot's application: %w", err)
	}
	if app.Bot == nil {
		return CheckResult{}, fmt.Errorf("the token's application has no bot user")
	}
	g, err := r.GetGuild(gid, false, opt)
	if err != nil {
		return CheckResult{}, fmt.Errorf("guild %s: %w", gid, err)
	}
	member, err := r.GetMember(gid, app.Bot.ID, opt)
	if err != nil {
		return CheckResult{}, fmt.Errorf("the bot in guild %s: %w", gid, err)
	}
	chans, err := r.GetGuildChannels(gid, opt)
	if err != nil {
		return CheckResult{}, fmt.Errorf("the channels of guild %s: %w", gid, err)
	}
	byID := map[string]dg.GuildChannel{}
	for _, c := range chans {
		byID[c.ID().String()] = c
	}
	res := CheckResult{Guild: gid.String(), Name: g.Name}
	for _, want := range channels {
		line := CheckLine{ID: want.ID, State: "missing", Reach: want.Reach, Perms: "-"}
		if c, ok := byID[want.ID]; ok {
			line.State = "present"
			have := channelPerms(gid, g.Roles, member.RoleIDs, app.Bot.ID, c.PermissionOverwrites())
			line.Perms = permsLine(have, neededPerms(want.Reach))
		}
		if line.State != "present" || line.Perms != "perms:ok" {
			res.Missing++
		}
		res.Lines = append(res.Lines, line)
	}
	intent := CheckLine{ID: "message-content", State: "present", Reach: CheckIntent, Perms: "perms:ok"}
	if !app.Flags.Has(dg.ApplicationFlagGatewayMessageContent) && !app.Flags.Has(dg.ApplicationFlagGatewayMessageContentLimited) {
		intent.State, intent.Perms = "missing", "-"
		res.Missing++
	}
	res.Lines = append(res.Lines, intent)
	return res, nil
}

// neededPerms is what the bot needs in a channel of the given reach.
func neededPerms(reach string) dg.Permissions {
	switch reach {
	case CheckNotice:
		return noticePerms
	case CheckCategory:
		return categoryPerms
	}
	return lobbyPerms
}

// permsLine is `perms:ok`, or `missing:` and the names of what need lacks in have.
func permsLine(have, need dg.Permissions) string {
	var missing []string
	for _, p := range permNames {
		if need.Has(p.bit) && !have.Has(p.bit) {
			missing = append(missing, p.name)
		}
	}
	if len(missing) == 0 {
		return "perms:ok"
	}
	return "missing:" + strings.Join(missing, ",")
}

// channelPerms is the bot's permissions in a channel, Discord's computation: the @everyone
// role's (its id is the guild's) and the bot's roles' guild permissions, everything for an
// administrator, then the channel's @everyone overwrite, its role overwrites together, and the
// bot's own member overwrite.
func channelPerms(guild snowflake.ID, roles []dg.Role, memberRoles []snowflake.ID, bot snowflake.ID, ows dg.PermissionOverwrites) dg.Permissions {
	held := map[snowflake.ID]bool{guild: true}
	for _, id := range memberRoles {
		held[id] = true
	}
	var base dg.Permissions
	for _, r := range roles {
		if held[r.ID] {
			base = base.Add(r.Permissions)
		}
	}
	if base.Has(dg.PermissionAdministrator) {
		return dg.PermissionsAll
	}
	if ow, ok := ows.Role(guild); ok {
		base = base.Remove(ow.Deny).Add(ow.Allow)
	}
	var allow, deny dg.Permissions
	for _, id := range memberRoles {
		if ow, ok := ows.Role(id); ok {
			allow, deny = allow.Add(ow.Allow), deny.Add(ow.Deny)
		}
	}
	base = base.Remove(deny).Add(allow)
	if ow, ok := ows.Member(bot); ok {
		base = base.Remove(ow.Deny).Add(ow.Allow)
	}
	return base
}
