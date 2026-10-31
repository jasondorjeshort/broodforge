// SPDX-License-Identifier: GPL-3.0-or-later

package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/jasondorjeshort/broodforge/internal/backup"
	"github.com/jasondorjeshort/broodforge/internal/config"
	"github.com/jasondorjeshort/broodforge/internal/control"
	"github.com/jasondorjeshort/broodforge/internal/session"
	"github.com/jasondorjeshort/broodforge/internal/setup"
	"github.com/jasondorjeshort/broodforge/internal/store"
)

// statusCmd is `broodforge status [--wait <seconds>] [--expect <value>]` (05-debug-interface.md,
// Bot lifecycle): it waits up to --wait seconds for the socket to answer, then prints
// `config: ok|last-good from <time>`, `discord: <mode> connected|disconnected`, `sessions: <open>
// open, <waiting> waiting-for-answer, <interrupted> interrupted`, `worktrees: <kept> kept with
// unpushed commits`, `backup: last good <age> ago|none`, `engine: <path> <version>`; --json
// nests the same under those keys and adds projects, the last fetch at a session start per
// project (05-debug-interface-git.md, The fetch at session start). The outcome token is the
// config value, ok or last-good.
var statusCmd = Command{
	Name:   "status",
	Class:  ClassRead,
	Expect: true,
	Flags: []Flag{
		{Name: "wait", Kind: FlagInt, Value: "<seconds>", Usage: "wait this long for the bot to answer"},
	},
}

func init() {
	commands = append(commands, &statusCmd)
	renderers[&statusCmd] = renderStatus
	statusCmd.Local = runStatusWait // here, not in the literal: it refers to statusCmd
}

// statusPoll is how often status --wait asks again.
const statusPoll = 200 * time.Millisecond

// runStatusWait runs when the first ask found no bot: it asks again until --wait runs out.
func runStatusWait(ctx context.Context, env *Env) error {
	deadline := time.Now().Add(time.Duration(env.Int("wait")) * time.Second)
	for {
		if !time.Now().Before(deadline) {
			return notRunning(env)
		}
		time.Sleep(statusPoll)
		err := socketCall(ctx, &statusCmd, env)
		var ce *control.Error
		if errors.As(err, &ce) && (ce.Code == control.CodeNotRunning || ce.Code == control.CodeConnectFailed) {
			continue
		}
		return err
	}
}

// statusResult is the bot's status.
type statusResult struct {
	Config string `json:"config"` // ok or last-good
	// InstanceName is the name the bot runs as, which status compares with config/'s.
	InstanceName string  `json:"instanceName"`
	ConfigFrom   *string `json:"configFrom"`
	Discord      struct {
		Mode      string `json:"mode"`
		Connected bool   `json:"connected"`
	} `json:"discord"`
	Sessions  session.Counts `json:"sessions"`
	Worktrees struct {
		Kept int64 `json:"kept"`
	} `json:"worktrees"`
	Backup struct {
		LastGood   *string `json:"lastGood"`
		AgeSeconds *int64  `json:"ageSeconds"`
	} `json:"backup"`
	Engine struct {
		Path    string `json:"path"`
		Version string `json:"version"`
	} `json:"engine"`
	// Projects is --json's alone: the text form has no line for it.
	Projects []statusProject `json:"projects"`
}

// statusProject is one configured project's canonical as the last fetch at a session start
// left it: fetched is the last fetch that succeeded and error the last attempt's line, both
// null when there is none; commit is "" and ahead 0 before the first start since the bot began.
type statusProject struct {
	ID        string `json:"id"`
	Canonical struct {
		Branch  string  `json:"branch"`
		Commit  string  `json:"commit"`
		Fetched *string `json:"fetched"`
		Ahead   int     `json:"ahead"`
		Error   *string `json:"error"`
	} `json:"canonical"`
}

func (b *bot) status(_ context.Context, _ *control.Call) (json.RawMessage, error) {
	var r statusResult
	r.Config = "ok"
	if lg := b.lastGoodAt(); lg != nil {
		r.Config = "last-good"
		t := store.FormatTime(*lg)
		r.ConfigFrom = &t
	}
	// Dry-run has no gateway to lose: its sink is attached for the bot's life (P11 renders into it).
	// Live is connected while the gateway session is ready.
	r.Discord.Mode, r.Discord.Connected = b.discord, b.gateway == nil || b.gateway.Connected()
	var err error
	if r.Sessions, err = session.CountSessions(b.st); err != nil {
		return nil, err
	}
	if err := b.st.Query("SELECT count(*) FROM worktree WHERE state = 'kept'", nil, func(row *store.Row) error {
		r.Worktrees.Kept = row.Int(0)
		return nil
	}); err != nil {
		return nil, err
	}
	if p, err := backup.Latest(b.root); err == nil {
		if st, err := os.Stat(filepath.FromSlash(p)); err == nil {
			t := store.FormatTime(st.ModTime())
			age := int64(time.Since(st.ModTime()).Seconds())
			r.Backup.LastGood, r.Backup.AgeSeconds = &t, &age
		}
	}
	c := b.config()
	r.Engine.Path, r.Engine.Version = c.Instance.Engine.Path, c.Instance.Engine.Version
	r.InstanceName = c.Instance.InstanceName
	r.Projects = []statusProject{}
	for _, p := range c.Projects {
		sp := statusProject{ID: p.ID}
		sp.Canonical.Branch = p.Repo.DefaultBranch
		if sp.Canonical.Branch == "" {
			sp.Canonical.Branch = "main" // as the session's git.Session reads an unset one
		}
		if f, ok := b.svc.LastFetch(p.ID); ok {
			sp.Canonical.Commit, sp.Canonical.Ahead = f.Commit, f.Ahead
			if !f.Fetched.IsZero() {
				t := store.FormatTime(f.Fetched)
				sp.Canonical.Fetched = &t
			}
			if f.Error != "" {
				e := f.Error
				sp.Canonical.Error = &e
			}
		}
		r.Projects = append(r.Projects, sp)
	}
	return json.Marshal(r)
}

// renderStatus prints the six lines, or the object with --json.
func renderStatus(env *Env, raw json.RawMessage) error {
	var r statusResult
	if err := json.Unmarshal(raw, &r); err != nil {
		return err
	}
	env.Out.SetToken(r.Config)
	if env.JSON {
		return env.Out.Record(raw, nil)
	}
	if r.Config == setup.StatusSetup {
		// A setup holds the address (ADR 0027, Decision 4): `config: setup` and - for the rest.
		for _, l := range []string{"config: setup", "discord: -", "sessions: -", "worktrees: -", "backup: -", "engine: -"} {
			if _, err := fmt.Fprintln(env.Out.w, l); err != nil {
				return err
			}
		}
		return nil
	}
	cfg := "config: ok"
	if r.Config != "ok" {
		cfg = "config: last-good from " + orDash(deref(r.ConfigFrom))
	}
	conn := "disconnected"
	if r.Discord.Connected {
		conn = "connected"
	}
	bk := "backup: none"
	if r.Backup.AgeSeconds != nil {
		bk = "backup: last good " + age(*r.Backup.AgeSeconds) + " ago"
	}
	lines := []string{
		cfg,
		"discord: " + r.Discord.Mode + " " + conn,
		fmt.Sprintf("sessions: %d open, %d waiting-for-answer, %d interrupted", r.Sessions.Open, r.Sessions.WaitingForAnswer, r.Sessions.Interrupted),
		fmt.Sprintf("worktrees: %d kept with unpushed commits", r.Worktrees.Kept),
		bk,
		"engine: " + orDash(r.Engine.Path) + " " + orDash(r.Engine.Version),
	}
	// config/ naming another instance than the bot runs as (it answered under its last-good
	// name, socket.go): said after the config line.
	if cur, err := config.PlainInstanceName(env.Root); err == nil && r.InstanceName != "" && cur != r.InstanceName {
		lines = append(lines[:1], append([]string{"instance-name: " + cur + " (bot runs as " + r.InstanceName + ")"}, lines[1:]...)...)
	}
	for _, l := range lines {
		if _, err := fmt.Fprintln(env.Out.w, l); err != nil {
			return err
		}
	}
	return nil
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// age is a duration in its largest whole unit: 45s, 12m, 3h, 2d.
func age(seconds int64) string {
	switch {
	case seconds < 60:
		return strconv.FormatInt(seconds, 10) + "s"
	case seconds < 3600:
		return strconv.FormatInt(seconds/60, 10) + "m"
	case seconds < 86400:
		return strconv.FormatInt(seconds/3600, 10) + "h"
	}
	return strconv.FormatInt(seconds/86400, 10) + "d"
}
