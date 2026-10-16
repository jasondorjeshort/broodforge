// SPDX-License-Identifier: GPL-3.0-or-later

package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jasondorjeshort/broodforge/internal/audit"
	"github.com/jasondorjeshort/broodforge/internal/backup"
	"github.com/jasondorjeshort/broodforge/internal/control"
	"github.com/jasondorjeshort/broodforge/internal/store"
)

// usageCmd is `broodforge usage [--session <id> | --user <id> | --credential <id> | --channel
// <id> | --project <id>] [--since <time>] [--by model|turn]` (05-debug-interface.md, Usage and
// audit queries). By model: one line per model, `<model>  <turns>  <input>  <output>  <cache
// read>  <cache write>  <interrupted turns>`, then the same columns for every model as the
// `total` line, printed even when there is no usage. By turn: `<session>  <turn>  <model>
// <input>  <output>  <cache read>  <cache write>  <interrupted>  <time>`, cut at --limit; --json
// adds the raw reports. A read command: the bot answers when it runs, the store file otherwise.
var usageCmd = Command{
	Name:  "usage",
	Class: ClassRead,
	List:  true,
	Flags: []Flag{
		{Name: "session", Kind: FlagString, Value: "<id>", Usage: "one session's usage"},
		{Name: "user", Kind: FlagString, Value: "<id>", Usage: "the requester's usage"},
		{Name: "credential", Kind: FlagString, Value: "<id>", Usage: "the payer's usage"},
		{Name: "channel", Kind: FlagString, Value: "<id>", Usage: "one channel's usage"},
		{Name: "project", Kind: FlagString, Value: "<id>", Usage: "one project's usage"},
		{Name: "since", Kind: FlagString, Value: "<time>", Usage: "rows recorded at or after this RFC 3339 time"},
		{Name: "by", Kind: FlagString, Default: "model", Value: "model|turn", Usage: "one line per model or per turn"},
	},
	Columns: usageModelColumns,
	Local:   runUsage,
}

func init() { commands = append(commands, &usageCmd) }

var (
	usageModelColumns = []string{"model", "turns", "input", "output", "cacheRead", "cacheWrite", "interruptedTurns"}
	usageTurnColumns  = []string{"session", "turn", "model", "input", "output", "cacheRead", "cacheWrite", "interrupted", "time"}
)

func runUsage(_ context.Context, env *Env) error {
	f := store.UsageFilter{Session: env.String("session"), User: env.String("user"),
		Credential: env.String("credential"), Channel: env.String("channel"), Project: env.String("project")}
	set := 0
	for _, v := range []string{f.Session, f.User, f.Credential, f.Channel, f.Project} {
		if v != "" {
			set++
		}
	}
	if set > 1 {
		return &usageError{"usage: give at most one of --session, --user, --credential, --channel, --project"}
	}
	if s := env.String("since"); s != "" {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return &usageError{"usage: --since " + s + ": want an RFC 3339 time such as 2026-10-16T00:00:00Z"}
		}
		f.Since = store.FormatTime(t)
	}
	by := env.String("by")
	if by != "model" && by != "turn" {
		return &usageError{"usage: --by " + by + ": want model or turn"}
	}
	r, err := openStore(env)
	if err != nil {
		return err
	}
	defer r.Close()

	if by == "turn" {
		rows, err := store.UsageByTurn(r, f)
		if err != nil {
			return err
		}
		out, err := encode(rows)
		if err != nil {
			return err
		}
		return env.Out.Result(out, usageTurnColumns)
	}
	rows, total, err := store.UsageByModel(r, f)
	if err != nil {
		return err
	}
	for _, row := range append(rows, total) {
		line, err := encode(row)
		if err != nil {
			return err
		}
		if err := env.Out.Record(line, usageModelColumns); err != nil {
			return err
		}
	}
	return nil
}

// openStore opens the instance's store read-only for a command running without the bot.
func openStore(env *Env) (*store.Reader, error) {
	r, err := store.OpenReadOnly(env.Root)
	if err != nil {
		return nil, localError(err)
	}
	return r, nil
}

// localError turns a store, audit or backup refusal into the CLI's error line; those packages
// do not import control.
func localError(err error) error {
	var se *store.Error
	if errors.As(err, &se) {
		return control.NewError(se.Code, se.Message)
	}
	return err
}

// notRunning is the write refusal for a command whose form needs the bot.
func notRunning(env *Env) error {
	addr, err := socketAddress(env.Root)
	if err != nil {
		return err
	}
	return control.NewError(control.CodeNotRunning, "no bot at "+addr)
}

// encode is json.Marshal without HTML escaping, so paths and details print as stored.
func encode[T []store.TurnUsage | store.UsageTotal | []store.Finding | backup.SnapshotResult |
	backup.VerifyResult | backup.TableRows | auditText | verifyLine | backup.RehearseResult | store.Scratch |
	scratchLine](v T) (json.RawMessage, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimSpace(b.Bytes()), nil
}

// auditExport is what a snapshot writes as backup/audit.jsonl.
var auditExport backup.Exporter = audit.Export
