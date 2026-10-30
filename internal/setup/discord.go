// SPDX-License-Identifier: GPL-3.0-or-later

package setup

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/jasondorjeshort/broodforge/internal/config"
	"github.com/jasondorjeshort/broodforge/internal/control"
	"github.com/jasondorjeshort/broodforge/internal/credentials"
	"github.com/jasondorjeshort/broodforge/internal/discord"
	"github.com/jasondorjeshort/broodforge/internal/store"
)

// The Discord steps (ADR 0027, Decision 3; decisions/0027-setup-commands.md): token,
// application, guild, channels and role, over discord.SetupAPI built per step from the stored
// token, so the token is not held past the step. Every id is derived; a channel or role setup
// found is never deleted or renamed.

// The owner's texts (decisions/0027-setup-commands.md, The owner's texts), quoted as they stand;
// the role's name is --role's.
const (
	textToken       = "Bot token (Developer Portal, Bot, Reset Token; not shown as you type):"
	textInvite      = "Open this link, pick the server, Authorize: "
	textWaitJoin    = "waiting for the bot to join"
	textRoleAsk     = `Let setup create the role "%s"? This adds Manage Roles to the invite link; setup asks you to remove it again at the end. [Y/n]`
	textRoleNo      = "create a role named %s in Server Settings, Roles, and give it to yourself; setup is waiting"
	textRoleRemove  = "remove Manage Roles from the bot's role in Server Settings, Roles; setup is waiting"
	textChannelsAsk = "This server already has channels. Create Sessions, #%s and #%s-notices in it? [y/N]"
)

// Texts of this pass the companion does not give word for word (for the owner's review).
const (
	textPortal = "no bot token yet: in the Developer Portal (https://discord.com/developers/applications), New Application, " +
		"then Bot, Reset Token; the bot joins a server you own, so make one in Discord first if you have none"
	textRoleAdmin    = "the bot's role has Administrator, which includes Manage Roles; remove Administrator from it in Server Settings, Roles; setup is waiting"
	textIntent       = "in the Developer Portal, Bot, switch on Message Content Intent; setup is waiting"
	textChannelsWait = "create the category Sessions and the text channels #%s and #%s-notices, or rerun with --channels yes; setup is waiting"
	textNoDiscord    = "no Discord"
	textDryRun       = "dry-run"
	textNoFixtures   = "build the binary in a checkout: no fixtures/engine beside bin/"
	categoryName     = "Sessions"
	defaultRole      = "Trusted"
)

// lobbyEntry is the lobby's entry in the project's channels: code reach, the trusted group may
// start, contribute and answer; mode, harness and timeouts are the instance's defaults.
var lobbyEntry = json.RawMessage(`{"reach":"code","roles":{"start":["trusted"],"contribute":["trusted"],"answer":["trusted"]}}`)

func waitFix(step, detail string) Line {
	return Line{Result: ResultFix, Step: step, Detail: detail, Wait: true}
}

// noAnswer is a call Discord did not answer: a fix the poll tries again.
func noAnswer(step string, err error) Line {
	return waitFix(step, "Discord did not answer ("+control.OneLine(err.Error())+"); setup is waiting")
}

// discordFail is a failed call: Discord's refusal ends the run refused-<step>, no answer waits.
func discordFail(step, what string, err error) (Line, error) {
	if se, ok := discord.Refused(err); ok {
		msg := what + ": " + se.Reason
		return fix(step, msg), refused(step, msg)
	}
	return noAnswer(step, err), nil
}

// passed is a step's passing line: ok when it wrote a key in this run, else skip.
func (s *State) passed(step, detail string) Line {
	if s.wrote[step] {
		return ok(step, detail)
	}
	return skip(step, detail)
}

func (s *State) markWrote(step string) {
	if s.wrote == nil {
		s.wrote = map[string]bool{}
	}
	s.wrote[step] = true
}

func (s *State) roleName() string {
	if s.opt.Role != "" {
		return s.opt.Role
	}
	return defaultRole
}

// writeInstance sets one key of the instance file and writes it.
func (s *State) writeInstance(inst *config.Doc, value json.RawMessage, path ...string) error {
	if err := inst.Set("", value, path...); err != nil {
		return toControl(err)
	}
	return config.WriteDoc(s.instancePath(), inst)
}

func refusalText(err error) string {
	if e, ok := credentials.IsRefusal(err); ok {
		return e.Message
	}
	return control.OneLine(err.Error())
}

// botToken is the stored bot token's value; --check reads the store read-only.
func (s *State) botToken(inst *config.Doc, id string) ([]byte, error) {
	key, err := s.keyPath(inst)
	if err != nil {
		return nil, err
	}
	if s.check {
		r, err := store.OpenReadOnly(s.root)
		if err != nil {
			return nil, err
		}
		defer r.Close()
		return credentials.BotToken(r, s.root, key, id)
	}
	st, err := s.openStore()
	if err != nil {
		return nil, err
	}
	return credentials.BotToken(st, s.root, key, id)
}

// api is the Discord calls over the stored token for one step; without a usable token, the
// step's fix line instead.
func (s *State) api(step string, inst *config.Doc) (discord.SetupAPI, *Line, error) {
	id := inst.String("discord", "botCredentialId")
	if id == "" {
		l := fix(step, "no bot token stored; run setup token")
		return nil, &l, nil
	}
	tok, err := s.botToken(inst, id)
	if err != nil {
		l := fix(step, "the bot token "+id+": "+refusalText(err)+"; run setup token")
		return nil, &l, nil
	}
	api, err := s.env.Discord(string(tok))
	clear(tok)
	if err != nil {
		l := fix(step, "the bot token "+id+": "+control.OneLine(err.Error())+"; run setup token")
		return nil, &l, nil
	}
	return api, nil, nil
}

// tryToken asks Discord for the token's application; a value without a token's shape is a
// refusal like Discord's.
func (s *State) tryToken(ctx context.Context, token string) (discord.SetupApp, error) {
	api, err := s.env.Discord(token)
	if err != nil {
		return discord.SetupApp{}, &discord.SetupError{Reason: control.OneLine(err.Error())}
	}
	defer api.Close()
	return api.Application(ctx)
}

// stepToken stores the bot token as the instance's bot-token credential: read hidden on a
// terminal, or from stdin with --yes; skip when the stored one answers the application call;
// a stored token Discord refuses is replaced in the run (a new id, the old one removed).
func stepToken(ctx context.Context, s *State) (Line, error) {
	const step = "token"
	if s.opt.DryRun {
		return skip(step, textDryRun), nil
	}
	if s.env.Discord == nil {
		return skip(step, textNoDiscord), nil
	}
	inst, err := s.instanceDoc()
	if err != nil {
		return Line{}, toControl(err)
	}
	if inst == nil {
		return s.noConfig(step)
	}
	cur := inst.String("discord", "botCredentialId")
	if cur == "" {
		if s.check {
			return fix(step, textPortal), nil
		}
		return s.newToken(ctx, inst, "", false)
	}
	tok, err := s.botToken(inst, cur)
	if err != nil {
		return Line{}, mismatch(step, cur, refusalText(err))
	}
	_, err = s.tryToken(ctx, string(tok))
	clear(tok)
	if err == nil {
		if s.alone && !s.check {
			return s.newToken(ctx, inst, cur, true)
		}
		return s.passed(step, cur), nil
	}
	se, ok := discord.Refused(err)
	if !ok {
		return noAnswer(step, err), nil
	}
	text := "Discord refused the stored token: " + se.Reason
	if s.check {
		return fix(step, text), nil
	}
	// Only a token Discord does not know (401), or a value without a token's shape (tryToken's
	// status 0, no call made), is replaced; any other refusal ends the run with it.
	if se.Status != http.StatusUnauthorized && se.Status != 0 {
		return fix(step, text), refused(step, se.Reason+"; Developer Portal, Bot")
	}
	if err := s.emit(fix(step, text)); err != nil {
		return Line{}, err
	}
	return s.newToken(ctx, inst, cur, false)
}

// newToken reads a token, checks it with Discord, stores it and points discord.botCredentialId
// at it; old, when set, is removed after (ask: `setup token`'s old and new records first). On a
// terminal a refused token is asked again; off one it is read from stdin once, with --yes only.
func (s *State) newToken(ctx context.Context, inst *config.Doc, old string, ask bool) (Line, error) {
	const step = "token"
	tty := s.prompt.tty && !s.opt.Yes
	var portal Line
	if old == "" {
		portal = fix(step, textPortal)
		if tty {
			if err := s.emit(portal); err != nil {
				return Line{}, err
			}
		}
	}
	for {
		var value string
		switch {
		case tty:
			v, err := s.prompt.Hidden(textToken)
			if err != nil {
				return Line{}, err
			}
			value = strings.TrimSpace(v)
		case s.opt.Yes && !s.prompt.tty:
			v, err := s.prompt.All()
			if err != nil {
				return Line{}, err
			}
			value = strings.TrimSpace(v)
		}
		if value == "" {
			return portal, control.NewError("needs-input-"+step, "pass the bot token on stdin with --yes")
		}
		_, verr := s.tryToken(ctx, value)
		if se, ok := discord.Refused(verr); ok {
			text := "Discord refused the token: " + se.Reason
			if tty {
				if err := s.emit(fix(step, text)); err != nil {
					return Line{}, err
				}
				continue
			}
			return fix(step, text), refused(step, se.Reason+"; Developer Portal, Bot, Reset Token")
		}
		// A token Discord did not answer for is stored all the same: the poll asks again with it.
		svc, err := s.service()
		if err != nil {
			return Line{}, err
		}
		rec, err := svc.AddInstance(credentials.KindBotToken, "bot", []byte(value))
		if err != nil {
			return Line{}, toControl(err)
		}
		if ask {
			yes, err := s.replace(step, old, rec.ID)
			if err != nil || !yes {
				_, _ = svc.Remove(rec.ID)
				if err != nil {
					return Line{}, err
				}
				return skip(step, "kept "+old), nil
			}
		}
		if err := s.writeInstance(inst, config.RawValue(rec.ID), "discord", "botCredentialId"); err != nil {
			_, _ = svc.Remove(rec.ID)
			return Line{}, err
		}
		s.markWrote(step)
		detail := rec.ID
		if old != "" && usable(s.st, old) {
			if _, err := svc.Remove(old); err != nil {
				return Line{}, toControl(err)
			}
			detail += "; " + old + " removed"
		}
		if verr != nil {
			return noAnswer(step, verr), nil
		}
		return ok(step, detail), nil
	}
}

// stepApplication writes discord.applicationId from the token's application and switches on
// the message-content intent (PATCH /applications/@me, flags ORed with 1<<19) unless 1<<18 or
// 1<<19 is set; a refused PATCH waits for the owner's switch in the portal.
func stepApplication(ctx context.Context, s *State) (Line, error) {
	const step = "application"
	if s.opt.DryRun {
		return skip(step, textDryRun), nil
	}
	if s.env.Discord == nil {
		return skip(step, textNoDiscord), nil
	}
	inst, err := s.instanceDoc()
	if err != nil {
		return Line{}, toControl(err)
	}
	if inst == nil {
		return s.noConfig(step)
	}
	api, l, err := s.api(step, inst)
	if err != nil || l != nil {
		return deref(l), err
	}
	defer api.Close()
	app, err := api.Application(ctx)
	if err != nil {
		return discordFail(step, "the application", err)
	}
	have := inst.String("discord", "applicationId")
	switch {
	case have == app.ID:
	case have != "" && (!s.alone || s.check):
		return Line{}, mismatch(step, "application "+have, app.ID)
	case s.check:
		return fix(step, "discord.applicationId is not set; run setup"), nil
	default:
		if have != "" {
			yes, err := s.replace(step, have, app.ID)
			if err != nil {
				return Line{}, err
			}
			if !yes {
				return skip(step, "kept "+have), nil
			}
		}
		if err := s.writeInstance(inst, config.RawValue(app.ID), "discord", "applicationId"); err != nil {
			return Line{}, err
		}
		s.markWrote(step)
	}
	if app.Flags&(discord.FlagMessageContent|discord.FlagMessageContentLimited) == 0 {
		if s.check {
			return fix(step, "the Message Content intent is off; run setup"), nil
		}
		if err := api.EnableMessageContent(ctx, app.Flags); err != nil {
			if se, ok := discord.Refused(err); ok {
				return waitFix(step, "Discord refused the intent switch ("+se.Reason+"); "+textIntent), nil
			}
			return noAnswer(step, err), nil
		}
		s.markWrote(step)
	}
	return s.passed(step, app.ID+"  message content on"), nil
}

func findGuild(gs []discord.SetupGuild, id string) (discord.SetupGuild, bool) {
	for _, g := range gs {
		if g.ID == id {
			return g, true
		}
	}
	return discord.SetupGuild{}, false
}

// roleAnswer is the role question's answer: --create-role, else asked once, before the invite
// (ask), else the default yes.
func (s *State) roleAnswer(ask bool) (bool, error) {
	if s.createRole != nil {
		return *s.createRole, nil
	}
	v := true
	switch s.opt.CreateRole {
	case "yes":
	case "no":
		v = false
	default:
		if !ask {
			return true, nil
		}
		a, err := s.prompt.YesNo("guild", "create-role", fmt.Sprintf(textRoleAsk, s.roleName()), true)
		if err != nil {
			return false, err
		}
		v = a
	}
	s.createRole = &v
	return v, nil
}

// stepGuild takes the bot's one guild (GET /users/@me/guilds), --guild, or the owner's pick of
// several; in none, the role question, then the invite link and a poll until the bot joins.
func stepGuild(ctx context.Context, s *State) (Line, error) {
	const step = "guild"
	if s.opt.DryRun {
		return skip(step, textDryRun), nil
	}
	if s.env.Discord == nil {
		return skip(step, textNoDiscord), nil
	}
	inst, err := s.instanceDoc()
	if err != nil {
		return Line{}, toControl(err)
	}
	if inst == nil {
		return s.noConfig(step)
	}
	api, l, err := s.api(step, inst)
	if err != nil || l != nil {
		return deref(l), err
	}
	defer api.Close()
	gs, err := api.Guilds(ctx)
	if err != nil {
		return discordFail(step, "the bot's servers", err)
	}
	have := inst.String("discord", "guildId")
	if have != "" {
		if g, ok := findGuild(gs, have); ok {
			return s.passed(step, g.ID+"  "+g.Name), nil
		}
		if !s.alone || s.check {
			return Line{}, mismatch(step, "guild "+have, "the bot is not in it")
		}
	}
	want := s.opt.Guild
	var pick *discord.SetupGuild
	switch {
	case want != "":
		if g, ok := findGuild(gs, want); ok {
			pick = &g
		}
	case len(gs) == 1:
		pick = &gs[0]
	case len(gs) > 1:
		if s.check {
			return fix(step, fmt.Sprintf("the bot is in %d servers; run setup --guild <id>", len(gs))), nil
		}
		if s.prompt.tty && !s.opt.Yes && s.env.Stderr != nil {
			for _, g := range gs {
				fmt.Fprintln(s.env.Stderr, "  "+g.ID+"  "+g.Name)
			}
		}
		id, err := s.prompt.Prompt(step, "guild", "The bot is in these servers; which one (its id)?", "")
		if err != nil {
			return Line{}, err
		}
		g, ok := findGuild(gs, strings.TrimSpace(id))
		if !ok {
			return Line{}, control.NewError("bad-args", quote(id)+" is not a server the bot is in")
		}
		pick = &g
	}
	if pick == nil {
		if s.check {
			if want != "" {
				return fix(step, "the bot is not in server "+want+"; run setup for the invite link"), nil
			}
			return fix(step, "the bot is in no server; run setup for the invite link"), nil
		}
		manage, err := s.roleAnswer(true)
		if err != nil {
			return Line{}, err
		}
		appID := inst.String("discord", "applicationId")
		if appID == "" {
			app, err := api.Application(ctx)
			if err != nil {
				return discordFail(step, "the application", err)
			}
			appID = app.ID
		}
		l := waitFix(step, textInvite+discord.InviteURL(appID, manage, want))
		l.Note = textWaitJoin
		return l, nil
	}
	if s.check {
		return fix(step, "discord.guildId is not set; run setup"), nil
	}
	if have != "" {
		yes, err := s.replace(step, have, pick.ID)
		if err != nil {
			return Line{}, err
		}
		if !yes {
			return skip(step, "kept "+have), nil
		}
	}
	if err := s.writeInstance(inst, config.RawValue(pick.ID), "discord", "guildId"); err != nil {
		return Line{}, err
	}
	s.markWrote(step)
	return ok(step, pick.ID+"  "+pick.Name), nil
}

// slot is one of the channels setup makes: the category, the lobby, the notice channel.
type slot struct {
	label, name string
	category    bool
	have, old   string // the id configured or found; a configured id gone from the guild
	set         bool   // found by name or created in this run: to be written
}

// channelEntries is the project's configured channels by id, comments left out.
func channelEntries(pdoc *config.Doc) map[string]json.RawMessage {
	raw, ok := pdoc.Get("channels")
	if !ok {
		return nil
	}
	var m map[string]json.RawMessage
	if json.Unmarshal(raw, &m) != nil {
		return nil
	}
	for k := range m {
		if strings.HasPrefix(k, config.CommentPrefix) {
			delete(m, k)
		}
	}
	return m
}

// isLobbyEntry reports whether a channel entry is the one setup writes for the lobby.
func isLobbyEntry(raw json.RawMessage) bool {
	var b bytes.Buffer
	return json.Compact(&b, raw) == nil && bytes.Equal(b.Bytes(), lobbyEntry)
}

// dropChannel removes a channel id from the project's channels: `setup channels` dropping an id
// gone from the guild.
func dropChannel(pdoc *config.Doc, id string) error {
	raw, ok := pdoc.Get("channels")
	if !ok {
		return nil
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return err
	}
	delete(m, id)
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return pdoc.Set("", b, "channels")
}

// stepChannels settles the category Sessions, the lobby #<project id> and #<project id>-notices:
// a configured id present in the guild is kept whatever its name, one gone is mismatch-channels;
// with no id, by name, then created (asked first when the guild has channels); the ids written to
// venue.category, channels.<lobby id> and discord.noticeChannelId.
func stepChannels(ctx context.Context, s *State) (Line, error) {
	const step = "channels"
	if s.opt.DryRun {
		return s.dryRunChannels()
	}
	if s.env.Discord == nil {
		return skip(step, textNoDiscord), nil
	}
	inst, err := s.instanceDoc()
	if err != nil {
		return Line{}, toControl(err)
	}
	if inst == nil {
		return s.noConfig(step)
	}
	pdoc, l, err := s.projectDoc(step)
	if err != nil || l != nil {
		return deref(l), err
	}
	guild := inst.String("discord", "guildId")
	if guild == "" {
		return fix(step, "no guild yet; run setup"), nil
	}
	api, l, err := s.api(step, inst)
	if err != nil || l != nil {
		return deref(l), err
	}
	defer api.Close()
	chans, err := api.Channels(ctx, guild)
	if err != nil {
		return discordFail(step, "the channels of guild "+guild, err)
	}
	byID := map[string]discord.SetupChannel{}
	for _, c := range chans {
		byID[c.ID] = c
	}
	id := s.project
	cat := &slot{label: categoryName, name: categoryName, category: true, have: pdoc.String("venue", "category")}
	lobby := &slot{label: "#" + id, name: id}
	notice := &slot{label: "#" + id + "-notices", name: id + "-notices", have: inst.String("discord", "noticeChannelId")}
	gone := func(label, cid string) error {
		if !s.alone || s.check {
			return mismatch(step, "channel "+cid+label, "no such channel in guild "+guild)
		}
		return nil
	}
	// The project's channels: every configured id must be in the guild (`setup channels` drops
	// the gone ones after its old and new records). The lobby among them is the one named
	// #<project id>, else the one holding the entry setup writes; any other is the owner's. The
	// dry-run form's placeholders count as no id: dropped without asking, never a mismatch.
	entries := channelEntries(pdoc)
	keys := slices.Sorted(maps.Keys(entries))
	var goneIDs, placeholders []string
	for _, cid := range keys {
		if config.PlaceholderChannel(cid) {
			placeholders = append(placeholders, cid)
			continue
		}
		c, ok := byID[cid]
		if !ok {
			label := ""
			if isLobbyEntry(entries[cid]) {
				label = " (" + lobby.label + ")"
			}
			if err := gone(label, cid); err != nil {
				return Line{}, err
			}
			goneIDs = append(goneIDs, cid)
			continue
		}
		if lobby.have == "" && c.Text && strings.EqualFold(c.Name, lobby.name) {
			lobby.have = cid
		}
	}
	for _, cid := range keys {
		if _, ok := byID[cid]; ok && lobby.have == "" && isLobbyEntry(entries[cid]) {
			lobby.have = cid
		}
	}
	for _, sl := range []*slot{cat, notice} {
		if sl.have == "" {
			continue
		}
		if _, ok := byID[sl.have]; !ok {
			if err := gone(" ("+sl.label+")", sl.have); err != nil {
				return Line{}, err
			}
			sl.old, sl.have = sl.have, ""
		}
	}
	slots := []*slot{cat, lobby, notice}
	var missing []string
	for _, sl := range slots {
		if sl.have != "" {
			continue
		}
		for _, c := range chans {
			if c.Category == sl.category && (sl.category || c.Text) && strings.EqualFold(c.Name, sl.name) {
				sl.have, sl.set = c.ID, true
				break
			}
		}
		if sl.have == "" {
			missing = append(missing, sl.label)
		}
	}
	if s.check && (len(missing) > 0 || len(placeholders) > 0 || slices.ContainsFunc(slots, func(sl *slot) bool { return sl.set })) {
		return fix(step, "the channels are not configured; run setup"), nil
	}
	if len(missing) > 0 {
		create := true
		if len(chans) > 0 {
			if create, err = s.channelsAnswer(id); err != nil {
				return Line{}, err
			}
		}
		if !create {
			return waitFix(step, fmt.Sprintf(textChannelsWait, id, id)), nil
		}
		for _, sl := range slots {
			if sl.have != "" {
				continue
			}
			var c discord.SetupChannel
			if sl.category {
				c, err = api.CreateCategory(ctx, guild, sl.name)
			} else {
				c, err = api.CreateTextChannel(ctx, guild, sl.name, cat.have)
			}
			if err != nil {
				if se, ok := discord.Refused(err); ok {
					return waitFix(step, "Discord refused creating "+sl.label+" ("+se.Reason+
						"); give the bot Manage Channels in Server Settings, Roles; setup is waiting"), nil
				}
				return noAnswer(step, err), nil
			}
			sl.have, sl.set = c.ID, true
		}
	}
	for _, sl := range slots {
		if sl.old == "" || !sl.set {
			continue
		}
		yes, err := s.replace(step, sl.old, sl.have)
		if err != nil {
			return Line{}, err
		}
		if !yes {
			return skip(step, "kept "+sl.old), nil
		}
	}
	// A gone configured id: the lobby's is replaced by the lobby found or made, any other is
	// dropped (new -), each after its old and new records.
	drop := placeholders // setup wrote them itself: no old and new records, no question
	for _, cid := range goneIDs {
		next := "-"
		if lobby.set && isLobbyEntry(entries[cid]) {
			next = lobby.have
		}
		yes, err := s.replace(step, cid, next)
		if err != nil {
			return Line{}, err
		}
		if !yes {
			return skip(step, "kept "+cid), nil
		}
		drop = append(drop, cid)
	}
	pchanged := len(drop) > 0
	for _, cid := range drop {
		if err := dropChannel(pdoc, cid); err != nil {
			return Line{}, toControl(err)
		}
	}
	if cat.set {
		if err := pdoc.Set("", config.RawValue(cat.have), "venue", "category"); err != nil {
			return Line{}, toControl(err)
		}
		pchanged = true
	}
	if lobby.set {
		if err := pdoc.Set("The lobby setup made: sessions start here, for the trusted role group.", lobbyEntry, "channels", lobby.have); err != nil {
			return Line{}, toControl(err)
		}
		pchanged = true
	}
	if pchanged {
		if err := config.WriteDoc(s.projectPath(s.project), pdoc); err != nil {
			return Line{}, err
		}
		s.markWrote(step)
	}
	if notice.set {
		if err := s.writeInstance(inst, config.RawValue(notice.have), "discord", "noticeChannelId"); err != nil {
			return Line{}, err
		}
		s.markWrote(step)
	}
	var parts []string
	for _, sl := range slots {
		parts = append(parts, sl.label+" "+sl.have)
	}
	return s.passed(step, strings.Join(parts, ", ")), nil
}

// channelsAnswer is --channels, else the question (default no), asked once.
func (s *State) channelsAnswer(project string) (bool, error) {
	switch s.opt.Channels {
	case "yes":
		return true, nil
	case "no":
		return false, nil
	}
	if s.channelsYes != nil {
		return *s.channelsYes, nil
	}
	v, err := s.prompt.YesNo("channels", "channels", fmt.Sprintf(textChannelsAsk, project, project), false)
	if err != nil {
		return false, err
	}
	s.channelsYes = &v
	return v, nil
}

func findRole(roles []discord.SetupRole, match func(discord.SetupRole) bool) (discord.SetupRole, bool) {
	for _, r := range roles {
		if match(r) {
			return r, true
		}
	}
	return discord.SetupRole{}, false
}

// stepRole settles roleGroups.trusted: the configured id, else the role by name, else created
// (the role answer yes and the bot holding Manage Roles) or waited for; the guild's owner is
// given it, refused mismatch-role when it sits at or above the bot's highest role. Then, on
// every run, the bot's Manage Roles: held, the removal fix, polled.
func stepRole(ctx context.Context, s *State) (Line, error) {
	const step = "role"
	if s.opt.DryRun {
		return s.dryRunRole()
	}
	if s.env.Discord == nil {
		return skip(step, textNoDiscord), nil
	}
	inst, err := s.instanceDoc()
	if err != nil {
		return Line{}, toControl(err)
	}
	if inst == nil {
		return s.noConfig(step)
	}
	guild := inst.String("discord", "guildId")
	if guild == "" {
		return fix(step, "no guild yet; run setup"), nil
	}
	api, l, err := s.api(step, inst)
	if err != nil || l != nil {
		return deref(l), err
	}
	defer api.Close()
	name := s.roleName()
	roles, err := api.Roles(ctx, guild)
	if err != nil {
		return discordFail(step, "the roles of guild "+guild, err)
	}
	var trusted []string
	old := ""
	if raw, ok := inst.Get("roleGroups", "trusted"); ok && json.Unmarshal(raw, &trusted) != nil {
		// A present key that is not a list of ids is a disagreement, replaced only by setup role.
		if !s.alone || s.check {
			return Line{}, mismatch(step, "roleGroups.trusted "+string(raw), "not a list of role ids")
		}
		old, trusted = string(raw), nil
	}
	if len(trusted) > 0 && trusted[0] == config.PlaceholderRole {
		// The dry-run form's placeholder counts as no id: replaced without asking.
		trusted = nil
	}
	var role *discord.SetupRole
	if len(trusted) > 0 {
		if r, ok := findRole(roles, func(r discord.SetupRole) bool { return r.ID == trusted[0] }); ok {
			role = &r
		} else if !s.alone || s.check {
			return Line{}, mismatch(step, "role "+trusted[0], "no such role in guild "+guild)
		} else {
			old = trusted[0]
		}
	}
	bot, err := api.Bot(ctx, guild)
	if err != nil {
		return discordFail(step, "the bot in guild "+guild, err)
	}
	manage := bot.Permissions&discord.PermManageRoles != 0
	if role == nil {
		if r, ok := findRole(roles, func(r discord.SetupRole) bool { return strings.EqualFold(r.Name, name) }); ok {
			role = &r
		} else {
			if s.check {
				return fix(step, "no role "+name+" in guild "+guild+"; run setup"), nil
			}
			yes, err := s.roleAnswer(false)
			if err != nil {
				return Line{}, err
			}
			if !yes || !manage {
				return waitFix(step, fmt.Sprintf(textRoleNo, name)), nil
			}
			r, err := api.CreateRole(ctx, guild, name)
			if err != nil {
				if se, ok := discord.Refused(err); ok {
					return waitFix(step, "Discord refused creating "+name+" ("+se.Reason+"); "+fmt.Sprintf(textRoleNo, name)), nil
				}
				return noAnswer(step, err), nil
			}
			role = &r
		}
		if s.check {
			return fix(step, "roleGroups.trusted is not set; run setup"), nil
		}
		if old != "" {
			yes, err := s.replace(step, old, role.ID)
			if err != nil {
				return Line{}, err
			}
			if !yes {
				return skip(step, "kept "+old), nil
			}
		}
		if err := s.writeInstance(inst, config.RawValue([]string{role.ID}), "roleGroups", "trusted"); err != nil {
			return Line{}, err
		}
		s.markWrote(step)
	}
	g, err := api.Guild(ctx, guild)
	if err != nil {
		return discordFail(step, "guild "+guild, err)
	}
	held, err := api.MemberRoles(ctx, guild, g.OwnerID)
	if err != nil {
		return discordFail(step, "the server owner", err)
	}
	if !slices.Contains(held, role.ID) {
		switch {
		case s.check:
			return fix(step, "the server owner does not hold "+name+"; run setup"), nil
		case !manage:
			return waitFix(step, fmt.Sprintf(textRoleNo, name)), nil
		case role.Position >= bot.Highest:
			// Where Discord puts a new role is undocumented: the bot can give only a role below
			// its own highest.
			return fix(step, "move the bot's role above "+name+" in Server Settings, Roles, then run setup role"),
				mismatch(step, "role "+name+" below the bot's highest role",
					fmt.Sprintf("%s at position %d, the bot's highest at %d", name, role.Position, bot.Highest))
		}
		if err := api.AddMemberRole(ctx, guild, g.OwnerID, role.ID); err != nil {
			return discordFail(step, "giving "+name+" to the server owner", err)
		}
		s.markWrote(step)
	}
	if err := s.goLive(inst); err != nil {
		return Line{}, err
	}
	switch {
	case bot.Administrator:
		// Administrator holds Manage Roles whatever the role's own bits: the owner removes it.
		return waitFix(step, textRoleAdmin), nil
	case manage:
		return waitFix(step, textRoleRemove), nil
	}
	return s.passed(step, name+"  "+role.ID+"  held by "+g.OwnerID+s.liveNote), nil
}

// goLive switches discord.mode from dry-run to live once a run that wrote a Discord id has them
// all; the scripted Discord leaves it dry-run.
func (s *State) goLive(inst *config.Doc) error {
	if s.check || inst.String("discord", "mode") != config.DiscordDryRun {
		return nil
	}
	if !slices.ContainsFunc([]string{"token", "application", "guild", "channels", "role"}, func(st string) bool { return s.wrote[st] }) {
		return nil
	}
	for _, k := range []string{"applicationId", "guildId", "botCredentialId", "noticeChannelId"} {
		if inst.String("discord", k) == "" {
			return nil
		}
	}
	if s.env.FakeDiscord {
		s.liveNote = "; discord.mode stays dry-run (scripted Discord)"
		return nil
	}
	if err := s.writeInstance(inst, config.RawValue(config.DiscordLive), "discord", "mode"); err != nil {
		return err
	}
	s.liveNote = "; discord.mode live"
	return nil
}

// discordEchoes is `discord check` for the check step, over the stored token, once the guild is
// set: the guild line, then each channel and the intent, a missing one a failed line.
func (s *State) discordEchoes(ctx context.Context) []Echo {
	const name = "discord check  "
	if s.opt.DryRun || s.env.Discord == nil {
		return nil
	}
	inst, err := s.instanceDoc()
	if err != nil || inst == nil {
		return nil
	}
	guild := inst.String("discord", "guildId")
	if guild == "" || inst.String("discord", "botCredentialId") == "" {
		return nil
	}
	c, err := config.Load(s.root, config.AnyCredential)
	if err != nil {
		return nil // config check's own lines say why
	}
	api, l, err := s.api("check", inst)
	if err != nil {
		return []Echo{{Text: name + "error: " + control.OneLine(err.Error())}}
	}
	if l != nil {
		return []Echo{{Text: name + "error: " + l.Detail}}
	}
	defer api.Close()
	var chans []discord.CheckChannel
	for _, ch := range c.Channels() {
		chans = append(chans, discord.CheckChannel{ID: ch.ID, Reach: ch.Reach})
	}
	if id := c.Instance.Discord.NoticeChannelID; id != "" {
		chans = append(chans, discord.CheckChannel{ID: id, Reach: discord.CheckNotice})
	}
	seen := map[string]bool{}
	for _, p := range c.Projects {
		if cat := p.Venue.Category; cat != "" && !seen[cat] {
			seen[cat] = true
			chans = append(chans, discord.CheckChannel{ID: cat, Reach: discord.CheckCategory})
		}
	}
	res, err := api.Check(ctx, guild, chans)
	if err != nil {
		return []Echo{{Text: name + "error: start-discord: " + control.OneLine(err.Error())}}
	}
	out := []Echo{{OK: true, Text: name + "guild  " + res.Guild + "  " + res.Name}}
	for _, cl := range res.Lines {
		out = append(out, Echo{OK: cl.State == "present" && cl.Perms == "perms:ok",
			Text: name + cl.ID + "  " + cl.State + "  " + cl.Reach + "  " + cl.Perms})
	}
	return out
}

// dryRunEntry is a placeholder channel's entry: code reach, prompt mode, its harness, the
// trusted group may start, contribute and answer.
type dryRunEntry struct {
	Reach   string          `json:"reach"`
	Mode    string          `json:"mode"`
	Harness string          `json:"harness"`
	Roles   json.RawMessage `json:"roles"`
}

// dryRunChannels is the channels step of the dry-run form: with no channel in the project file,
// config.DryRunChannels written, and engine.fakeFixtures, while unset, as fixtures/engine under
// the effective install directory; no such directory is a fix that does not wait, so the run
// ends refused-channels with nothing written. Nothing Discord is called.
func (s *State) dryRunChannels() (Line, error) {
	const step = "channels"
	inst, err := s.instanceDoc()
	if err != nil {
		return Line{}, toControl(err)
	}
	if inst == nil {
		return s.noConfig(step)
	}
	pdoc, l, err := s.projectDoc(step)
	if err != nil || l != nil {
		return deref(l), err
	}
	needChannels := len(channelEntries(pdoc)) == 0
	fixtures := inst.String("engine", "fakeFixtures")
	needFixtures := fixtures == ""
	if needFixtures {
		dir, err := installDir(inst)
		if err != nil {
			msg := "the running binary's path: " + control.OneLine(err.Error())
			return fix(step, msg), control.NewError(control.CodeInternal, msg)
		}
		fixtures = filepath.ToSlash(filepath.Join(dir, "fixtures", "engine"))
		if st, err := os.Stat(fixtures); err != nil || !st.IsDir() {
			return fix(step, textNoFixtures), nil
		}
	}
	if s.check && (needChannels || needFixtures) {
		return fix(step, "the dry-run channels are not configured; run setup --discord dry-run"), nil
	}
	if needChannels {
		roles := json.RawMessage(`{"start":["trusted"],"contribute":["trusted"],"answer":["trusted"]}`)
		for _, c := range config.DryRunChannels {
			b, err := json.Marshal(dryRunEntry{Reach: config.ReachCode, Mode: config.ModePrompt, Harness: c.Harness, Roles: roles})
			if err != nil {
				return Line{}, err
			}
			comment := "A dry-run placeholder (setup --discord dry-run), the " + c.Harness + " harness; a --discord live run replaces it."
			if err := pdoc.Set(comment, b, "channels", c.ID); err != nil {
				return Line{}, toControl(err)
			}
		}
		if err := config.WriteDoc(s.projectPath(s.project), pdoc); err != nil {
			return Line{}, err
		}
		s.markWrote(step)
	}
	if needFixtures {
		if err := s.writeInstance(inst, config.RawValue(fixtures), "engine", "fakeFixtures"); err != nil {
			return Line{}, err
		}
		s.markWrote(step)
	}
	entries := channelEntries(pdoc)
	var parts []string
	for _, id := range slices.Sorted(maps.Keys(entries)) {
		var e struct {
			Harness string `json:"harness"`
		}
		if json.Unmarshal(entries[id], &e) != nil {
			parts = append(parts, id+" ?") // config check names what is wrong with the entry
			continue
		}
		if e.Harness == "" {
			e.Harness = inst.String("defaults", "harness")
		}
		parts = append(parts, strings.TrimSpace(id+" "+e.Harness))
	}
	return s.passed(step, textDryRun+" "+strings.Join(parts, ", ")+"; fixtures "+fixtures), nil
}

// installDir is the effective install directory, as the spawn backend resolves it: paths.install,
// else the parent of the running binary's directory (the checkout, for bin/broodforge). Without
// the binary's path it is an error, never a relative directory.
func installDir(inst *config.Doc) (string, error) {
	if p := inst.String("paths", "install"); p != "" {
		return p, nil
	}
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(filepath.Dir(exe)), nil
}

// dryRunRole is the role step of the dry-run form: an empty roleGroups.trusted becomes the
// placeholder role config.PlaceholderRole; a present list is kept. Nothing Discord is called.
func (s *State) dryRunRole() (Line, error) {
	const step = "role"
	inst, err := s.instanceDoc()
	if err != nil {
		return Line{}, toControl(err)
	}
	if inst == nil {
		return s.noConfig(step)
	}
	var trusted []string
	if raw, ok := inst.Get("roleGroups", "trusted"); ok && json.Unmarshal(raw, &trusted) != nil {
		return Line{}, mismatch(step, "roleGroups.trusted "+string(raw), "not a list of role ids")
	}
	if len(trusted) > 0 {
		return skip(step, textDryRun+" "+strings.Join(trusted, ",")), nil
	}
	if s.check {
		return fix(step, "roleGroups.trusted is not set; run setup --discord dry-run"), nil
	}
	if err := s.writeInstance(inst, config.RawValue([]string{config.PlaceholderRole}), "roleGroups", "trusted"); err != nil {
		return Line{}, err
	}
	s.markWrote(step)
	return ok(step, textDryRun+" "+config.PlaceholderRole), nil
}
