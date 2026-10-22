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
	"github.com/jasondorjeshort/broodforge/internal/store"
)

// statusCmd is `broodforge status [--wait <seconds>] [--expect <value>]` (05-debug-interface.md,
// Bot lifecycle): it waits up to --wait seconds for the socket to answer, then prints
// `config: ok|last-good from <time>`, `discord: <mode> connected|disconnected`, `sessions: <open>
// open, <waiting> waiting-for-answer, <interrupted> interrupted`, `worktrees: <kept> kept with
// unpushed commits`, `backup: last good <age> ago|none`, `engine: <path> <version>`; --json
// nests the same under those keys. The outcome token is the config value, ok or last-good.
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
