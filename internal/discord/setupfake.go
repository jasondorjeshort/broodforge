// SPDX-License-Identifier: GPL-3.0-or-later

package discord

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"slices"
	"strconv"
	"sync"

	dg "github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/snowflake/v2"
)

// The scripted Discord of `broodforge setup` without Discord (08-implementation-plan-setup.md,
// P22): setupREST over a JSON script of the application and the guilds, so setup's steps, the
// permission arithmetic and `discord check` run as they do against Discord. The binary reads a
// script file named by BROODFORGE_SETUP_FAKE_DISCORD (never on a live instance) and writes every
// change back to it, so a rerun sees what the first run created; tests hold one in memory.

// FakeScript is the scripted world. Counters advance on every read and make the owner's hand
// steps happen after a number of polls.
type FakeScript struct {
	// Token is the one token the fake accepts; "" accepts any token of the right shape.
	Token       string      `json:"token,omitempty"`
	Application FakeApp     `json:"application"`
	Guilds      []FakeGuild `json:"guilds"`
	// Invite is the guild the bot joins once the owner opens the invite: on the read of the
	// bot's guilds after JoinAfter reads.
	Invite    *FakeGuild `json:"invite,omitempty"`
	JoinAfter int        `json:"joinAfter,omitempty"`
	// GuildReads counts the reads of the bot's guilds.
	GuildReads int `json:"guildReads,omitempty"`
	// Next is the last id handed out; new ids count up from it.
	Next uint64 `json:"next,omitempty"`
	// Calls records every write as `<call> <args>`.
	Calls []string `json:"calls,omitempty"`
	// Fail queues HTTP statuses by call name (GetBotApplicationInfo, ...): the call's next
	// answers are those errors, one each, before the script answers again.
	Fail map[string][]int `json:"fail,omitempty"`
}

// FakeApp is the application behind the token.
type FakeApp struct {
	ID    string `json:"id"`
	BotID string `json:"botId"`
	Flags uint64 `json:"flags,omitempty"`
	// RefusePatch refuses PATCH /applications/@me, as Discord does for some applications.
	RefusePatch bool `json:"refusePatch,omitempty"`
}

// FakeGuild is one guild with its channels, roles and the two members setup reads.
type FakeGuild struct {
	ID       string        `json:"id"`
	Name     string        `json:"name"`
	OwnerID  string        `json:"ownerId"`
	Channels []FakeChannel `json:"channels,omitempty"`
	// Roles: @everyone's id is the guild's.
	Roles      []FakeRole `json:"roles,omitempty"`
	BotRoles   []string   `json:"botRoles,omitempty"`
	OwnerRoles []string   `json:"ownerRoles,omitempty"`
	// RoleAbove puts a role the bot creates above the bot's highest role.
	RoleAbove bool `json:"roleAbove,omitempty"`
	// OwnerMakesRole is a role the owner creates and takes by hand once the roles were read
	// OwnerMakesAfter times.
	OwnerMakesRole  string `json:"ownerMakesRole,omitempty"`
	OwnerMakesAfter int    `json:"ownerMakesAfter,omitempty"`
	RoleReads       int    `json:"roleReads,omitempty"`
	// DropManageRolesAfter: the owner removes Manage Roles from the bot's roles once the bot's
	// member was read this many times; 0 never.
	DropManageRolesAfter int `json:"dropManageRolesAfter,omitempty"`
	BotReads             int `json:"botReads,omitempty"`
}

// FakeChannel is a channel; a text channel unless Category.
type FakeChannel struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Category bool   `json:"category,omitempty"`
	Parent   string `json:"parent,omitempty"`
}

// FakeRole is a role.
type FakeRole struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Position    int    `json:"position"`
	Permissions uint64 `json:"permissions"`
}

// FakeSetup is a FakeScript behind a lock, written back to its file after each call.
type FakeSetup struct {
	mu   sync.Mutex
	path string
	s    FakeScript
}

// NewFakeSetup holds s in memory.
func NewFakeSetup(s FakeScript) *FakeSetup { return &FakeSetup{s: s} }

// LoadFakeSetup reads the script at path; every change is written back to it.
func LoadFakeSetup(path string) (*FakeSetup, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	f := &FakeSetup{path: path}
	if err := json.Unmarshal(b, &f.s); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return f, nil
}

// Script is a copy of the world as it stands.
func (f *FakeSetup) Script() FakeScript {
	f.mu.Lock()
	defer f.mu.Unlock()
	b, _ := json.Marshal(f.s)
	var c FakeScript
	_ = json.Unmarshal(b, &c)
	return c
}

// API is SetupAPI over the fake with token, its shape checked as NewSetupAPI checks it.
func (f *FakeSetup) API(token string) (SetupAPI, error) {
	if _, err := CheckToken(token); err != nil {
		return nil, err
	}
	return &setupAPI{r: &fakeREST{f: f, token: token}}, nil
}

// fakeREST is setupREST over the script.
type fakeREST struct {
	f     *FakeSetup
	token string
}

// do runs the call named name on the script under the lock: the next status Fail queues for it
// first, then a token the script does not accept is 401, else fn; the script is written back.
func (r *fakeREST) do(name string, fn func(s *FakeScript) error) error {
	f := r.f
	f.mu.Lock()
	defer f.mu.Unlock()
	var err error
	if q := f.s.Fail[name]; len(q) > 0 {
		f.s.Fail[name] = q[1:]
		err = fakeErr(q[0], 0, strconv.Itoa(q[0])+": "+http.StatusText(q[0]))
	} else if f.s.Token != "" && r.token != f.s.Token {
		return fakeErr(http.StatusUnauthorized, 0, "401: Unauthorized")
	} else {
		err = fn(&f.s)
	}
	if f.path != "" {
		b, merr := json.MarshalIndent(f.s, "", "  ")
		if merr != nil {
			return merr
		}
		if werr := os.WriteFile(f.path, append(b, '\n'), 0o600); werr != nil {
			return werr
		}
	}
	return err
}

func fakeErr(status, code int, msg string) error {
	return &rest.Error{Response: &http.Response{StatusCode: status, Status: strconv.Itoa(status) + " " + http.StatusText(status)},
		Code: rest.JSONErrorCode(code), Message: msg}
}

func errMissingPermissions() error {
	return fakeErr(http.StatusForbidden, 50013, "Missing Permissions")
}

func (s *FakeScript) id() string {
	if s.Next == 0 {
		s.Next = 9000
	}
	s.Next++
	return strconv.FormatUint(s.Next, 10)
}

func (s *FakeScript) guild(id snowflake.ID) (*FakeGuild, error) {
	for i := range s.Guilds {
		if s.Guilds[i].ID == id.String() {
			return &s.Guilds[i], nil
		}
	}
	return nil, fakeErr(http.StatusNotFound, 10004, "Unknown Guild")
}

func sfID(s string) snowflake.ID {
	v, _ := snowflake.Parse(s)
	return v
}

// botPerms is the bot's guild permissions and highest role position in g.
func (s *FakeScript) botPerms(g *FakeGuild) (dg.Permissions, int) {
	roles, held := g.dgRoles(), []snowflake.ID{}
	high := 0
	for _, id := range g.BotRoles {
		held = append(held, sfID(id))
		for _, r := range g.Roles {
			if r.ID == id && r.Position > high {
				high = r.Position
			}
		}
	}
	return channelPerms(sfID(g.ID), roles, held, sfID(s.Application.BotID), nil), high
}

func (g *FakeGuild) dgRoles() []dg.Role {
	out := make([]dg.Role, 0, len(g.Roles))
	for _, r := range g.Roles {
		out = append(out, dg.Role{ID: sfID(r.ID), GuildID: sfID(g.ID), Name: r.Name, Position: r.Position, Permissions: dg.Permissions(r.Permissions)})
	}
	return out
}

func (g *FakeGuild) channel(c FakeChannel) (dg.GuildChannel, error) {
	t, parent := 0, "null"
	if c.Category {
		t = int(dg.ChannelTypeGuildCategory)
	}
	if c.Parent != "" {
		parent = strconv.Quote(c.Parent)
	}
	raw := `{"id":` + strconv.Quote(c.ID) + `,"type":` + strconv.Itoa(t) + `,"guild_id":` + strconv.Quote(g.ID) +
		`,"name":` + strconv.Quote(c.Name) + `,"parent_id":` + parent + `}`
	var u dg.UnmarshalChannel
	if err := json.Unmarshal([]byte(raw), &u); err != nil {
		return nil, err
	}
	gc, ok := u.Channel.(dg.GuildChannel)
	if !ok {
		return nil, fmt.Errorf("channel %s is not a guild channel", c.ID)
	}
	return gc, nil
}

func (r *fakeREST) GetBotApplicationInfo(...rest.RequestOpt) (*dg.Application, error) {
	var app *dg.Application
	err := r.do("GetBotApplicationInfo", func(s *FakeScript) error {
		app = &dg.Application{ID: sfID(s.Application.ID), Bot: &dg.User{ID: sfID(s.Application.BotID)},
			Flags: dg.ApplicationFlags(s.Application.Flags)}
		return nil
	})
	return app, err
}

func (r *fakeREST) UpdateCurrentApplication(u dg.ApplicationUpdate, _ ...rest.RequestOpt) (*dg.Application, error) {
	var app *dg.Application
	err := r.do("UpdateCurrentApplication", func(s *FakeScript) error {
		flags := uint64(0)
		if u.Flags != nil {
			flags = uint64(*u.Flags)
		}
		s.Calls = append(s.Calls, "UpdateCurrentApplication flags="+strconv.FormatUint(flags, 10))
		if s.Application.RefusePatch {
			return fakeErr(http.StatusForbidden, 20012, "You are not authorized to perform this action on this application")
		}
		if u.Flags != nil {
			s.Application.Flags = flags
		}
		app = &dg.Application{ID: sfID(s.Application.ID), Flags: dg.ApplicationFlags(s.Application.Flags)}
		return nil
	})
	return app, err
}

func (r *fakeREST) GetCurrentUserGuilds(bearer string, _, _ snowflake.ID, _ int, _ bool, _ ...rest.RequestOpt) ([]dg.OAuth2Guild, error) {
	var out []dg.OAuth2Guild
	err := r.do("GetCurrentUserGuilds", func(s *FakeScript) error {
		if bearer != "" {
			return fakeErr(http.StatusUnauthorized, 0, "401: Unauthorized")
		}
		s.GuildReads++
		if s.Invite != nil && s.GuildReads > s.JoinAfter {
			s.Guilds = append(s.Guilds, *s.Invite)
			s.Invite = nil
		}
		for _, g := range s.Guilds {
			out = append(out, dg.OAuth2Guild{ID: sfID(g.ID), Name: g.Name})
		}
		return nil
	})
	return out, err
}

func (r *fakeREST) GetGuild(id snowflake.ID, _ bool, _ ...rest.RequestOpt) (*dg.RestGuild, error) {
	var out *dg.RestGuild
	err := r.do("GetGuild", func(s *FakeScript) error {
		g, err := s.guild(id)
		if err != nil {
			return err
		}
		out = &dg.RestGuild{Guild: dg.Guild{ID: id, Name: g.Name, OwnerID: sfID(g.OwnerID)}, Roles: g.dgRoles()}
		return nil
	})
	return out, err
}

func (r *fakeREST) GetRoles(id snowflake.ID, _ ...rest.RequestOpt) ([]dg.Role, error) {
	var out []dg.Role
	err := r.do("GetRoles", func(s *FakeScript) error {
		g, err := s.guild(id)
		if err != nil {
			return err
		}
		g.RoleReads++
		if g.OwnerMakesRole != "" && g.RoleReads > g.OwnerMakesAfter {
			rid := s.id()
			g.Roles = append(g.Roles, FakeRole{ID: rid, Name: g.OwnerMakesRole, Position: 1})
			g.OwnerRoles = append(g.OwnerRoles, rid)
			g.OwnerMakesRole = ""
		}
		out = g.dgRoles()
		return nil
	})
	return out, err
}

func (r *fakeREST) CreateRole(id snowflake.ID, c dg.RoleCreate, _ ...rest.RequestOpt) (*dg.Role, error) {
	var out *dg.Role
	err := r.do("CreateRole", func(s *FakeScript) error {
		g, err := s.guild(id)
		if err != nil {
			return err
		}
		s.Calls = append(s.Calls, "CreateRole "+c.Name)
		perms, high := s.botPerms(g)
		if !perms.Has(dg.PermissionManageRoles) {
			return errMissingPermissions()
		}
		pos := 1
		if g.RoleAbove {
			pos = high + 1
		}
		fr := FakeRole{ID: s.id(), Name: c.Name, Position: pos}
		if c.Permissions != nil {
			fr.Permissions = uint64(*c.Permissions)
		}
		g.Roles = append(g.Roles, fr)
		out = &dg.Role{ID: sfID(fr.ID), Name: fr.Name, Position: fr.Position, Permissions: dg.Permissions(fr.Permissions)}
		return nil
	})
	return out, err
}

func (r *fakeREST) GetMember(id snowflake.ID, user snowflake.ID, _ ...rest.RequestOpt) (*dg.Member, error) {
	var out *dg.Member
	err := r.do("GetMember", func(s *FakeScript) error {
		g, err := s.guild(id)
		if err != nil {
			return err
		}
		var roles []string
		switch user.String() {
		case s.Application.BotID:
			g.BotReads++
			if g.DropManageRolesAfter > 0 && g.BotReads > g.DropManageRolesAfter {
				for i := range g.Roles {
					if slices.Contains(g.BotRoles, g.Roles[i].ID) {
						g.Roles[i].Permissions &^= uint64(dg.PermissionManageRoles)
					}
				}
			}
			roles = g.BotRoles
		case g.OwnerID:
			roles = g.OwnerRoles
		default:
			return fakeErr(http.StatusNotFound, 10007, "Unknown Member")
		}
		out = &dg.Member{User: dg.User{ID: user}}
		for _, rid := range roles {
			out.RoleIDs = append(out.RoleIDs, sfID(rid))
		}
		return nil
	})
	return out, err
}

func (r *fakeREST) AddMemberRole(id snowflake.ID, user snowflake.ID, role snowflake.ID, _ ...rest.RequestOpt) error {
	return r.do("AddMemberRole", func(s *FakeScript) error {
		g, err := s.guild(id)
		if err != nil {
			return err
		}
		s.Calls = append(s.Calls, "AddMemberRole "+user.String()+" "+role.String())
		perms, high := s.botPerms(g)
		pos := -1
		for _, fr := range g.Roles {
			if fr.ID == role.String() {
				pos = fr.Position
			}
		}
		switch {
		case pos < 0:
			return fakeErr(http.StatusNotFound, 10011, "Unknown Role")
		case !perms.Has(dg.PermissionManageRoles) || pos >= high:
			return errMissingPermissions()
		case user.String() != g.OwnerID:
			return fakeErr(http.StatusNotFound, 10007, "Unknown Member")
		}
		if !slices.Contains(g.OwnerRoles, role.String()) {
			g.OwnerRoles = append(g.OwnerRoles, role.String())
		}
		return nil
	})
}

func (r *fakeREST) GetGuildChannels(id snowflake.ID, _ ...rest.RequestOpt) ([]dg.GuildChannel, error) {
	var out []dg.GuildChannel
	err := r.do("GetGuildChannels", func(s *FakeScript) error {
		g, err := s.guild(id)
		if err != nil {
			return err
		}
		for _, c := range g.Channels {
			gc, err := g.channel(c)
			if err != nil {
				return err
			}
			out = append(out, gc)
		}
		return nil
	})
	return out, err
}

func (r *fakeREST) CreateGuildChannel(id snowflake.ID, c dg.GuildChannelCreate, _ ...rest.RequestOpt) (dg.GuildChannel, error) {
	var out dg.GuildChannel
	err := r.do("CreateGuildChannel", func(s *FakeScript) error {
		g, err := s.guild(id)
		if err != nil {
			return err
		}
		fc := FakeChannel{}
		if cat, ok := c.(dg.GuildCategoryChannelCreate); ok {
			fc.Name, fc.Category = cat.Name, true
		} else if txt, ok := c.(dg.GuildTextChannelCreate); ok {
			fc.Name = txt.Name
			if txt.ParentID != 0 {
				fc.Parent = txt.ParentID.String()
			}
		} else {
			return fakeErr(http.StatusBadRequest, 50035, "Invalid Form Body")
		}
		s.Calls = append(s.Calls, "CreateGuildChannel "+fc.Name)
		if perms, _ := s.botPerms(g); !perms.Has(dg.PermissionManageChannels) {
			return errMissingPermissions()
		}
		fc.ID = s.id()
		g.Channels = append(g.Channels, fc)
		out, err = g.channel(fc)
		return err
	})
	return out, err
}
