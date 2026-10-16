// SPDX-License-Identifier: GPL-3.0-or-later

package cli

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/jasondorjeshort/broodforge/internal/audit"
	"github.com/jasondorjeshort/broodforge/internal/store"
)

// auditCmd is `broodforge audit [--since-seq <n>] [--kind <kind>[,<kind>]] [--session <id>]
// [--limit 20] [--follow]` (05-debug-interface.md, Usage and audit queries; --all as on every
// list, 05-cli-conventions.md): `<seq>  <time>  <kind>  <phase or ->  <actor>  <session or ->
// <details, first 80 chars>` per event in seq order; --json prints every column. Without --since-seq the newest --limit events,
// with it the first --limit after that seq. An empty log prints nothing, exit 0. --follow
// streams new events and needs the bot; without one it is not-running.
var auditCmd = Command{
	Name:  "audit",
	Class: ClassRead,
	List:  true,
	Flags: []Flag{
		{Name: "since-seq", Kind: FlagInt, Value: "<n>", Usage: "only events after this seq"},
		{Name: "kind", Kind: FlagString, Value: "<kind>[,<kind>]", Usage: "only these kinds"},
		{Name: "session", Kind: FlagString, Value: "<id>", Usage: "only this session's events"},
		{Name: "follow", Kind: FlagBool, Usage: "stream new events (needs the bot)"},
	},
	Columns:      auditColumns,
	EventColumns: auditColumns,
	Local:        runAudit,
}

func init() { commands = append(commands, &auditCmd) }

var auditColumns = []string{"seq", "time", "kind", "phase", "actor", "session", "details"}

// auditDetailsWidth is how much of the details text mode shows.
const auditDetailsWidth = 80

// auditText is an event as text mode prints it: details cut to its first 80 characters.
type auditText struct {
	Seq     int64   `json:"seq"`
	Time    string  `json:"time"`
	Kind    string  `json:"kind"`
	Phase   *string `json:"phase"`
	Actor   string  `json:"actor"`
	Session *string `json:"session"`
	Details string  `json:"details"`
}

func runAudit(_ context.Context, env *Env) error {
	f := audit.Filter{SinceSeq: int64(env.Int("since-seq")), Session: env.String("session"), Limit: env.Out.limit}
	if f.SinceSeq < 0 {
		return &usageError{"audit: --since-seq must not be negative"}
	}
	if k := env.String("kind"); k != "" {
		for _, s := range strings.Split(k, ",") {
			s = strings.TrimSpace(s)
			if !audit.ValidKind(s) {
				return &usageError{"audit: --kind " + s + ": a kind is kebab-case, such as session-started"}
			}
			f.Kinds = append(f.Kinds, audit.Kind(s))
		}
	}
	if env.Bool("follow") {
		return notRunning(env)
	}
	r, err := openStore(env)
	if err != nil {
		return err
	}
	defer r.Close()
	events, err := audit.Query(r, f)
	if err != nil {
		return err
	}
	for _, ev := range events {
		line, err := ev.MarshalJSON()
		if err == nil && !env.JSON {
			line, err = encode(textEvent(ev))
		}
		if err != nil {
			return err
		}
		if err := env.Out.Record(line, auditColumns); err != nil {
			return err
		}
	}
	return nil
}

func textEvent(ev audit.Event) auditText {
	t := auditText{Seq: ev.Seq, Time: store.FormatTime(ev.Time), Kind: string(ev.Kind), Actor: ev.Actor,
		Details: firstRunes(string(ev.Details), auditDetailsWidth)}
	if ev.Phase != audit.PhaseNone {
		p := string(ev.Phase)
		t.Phase = &p
	}
	if ev.Session != "" {
		t.Session = &ev.Session
	}
	return t
}

func firstRunes(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	i := 0
	for j := range s {
		if i == n {
			return s[:j]
		}
		i++
	}
	return s
}
