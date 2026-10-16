// SPDX-License-Identifier: GPL-3.0-or-later

package cli

import (
	"context"

	"github.com/jasondorjeshort/broodforge/internal/store"
)

// reconcileCmd is `broodforge reconcile [--report-only]` (05-debug-interface.md, Bot
// lifecycle): ADR 0021's reconciliation on demand. The bot applies it; --report-only works on
// the store file without the bot and writes nothing. One line per finding, `<kind>  <subject>
// <session or ->  <detail or ->`: `interrupt  <session id>  <id>  <state>` for a session in a
// running state, `missing  <worktree path>  <session>  <missing since or ->` for an absent
// worktree, `present  <path>  <session>  <missing since>` for one that is back,
// `unrecorded  <directory under sessions/>  -  -` for a directory no session names. A clean
// store prints nothing, exit 0. Without --report-only and without the bot, not-running.
var reconcileCmd = Command{
	Name:  "reconcile",
	Class: ClassRead,
	List:  true,
	Flags: []Flag{
		{Name: "report-only", Kind: FlagBool, Usage: "list what reconciliation would do; write nothing"},
	},
	Columns: reconcileColumns,
	Local:   runReconcile,
}

func init() { commands = append(commands, &reconcileCmd) }

var reconcileColumns = []string{"kind", "subject", "session", "detail"}

func runReconcile(_ context.Context, env *Env) error {
	if !env.Bool("report-only") {
		return notRunning(env)
	}
	r, err := openStore(env)
	if err != nil {
		return err
	}
	defer r.Close()
	findings, err := store.Reconcile(r, env.Root)
	if err != nil {
		return err
	}
	out, err := encode(findings)
	if err != nil {
		return err
	}
	return env.Out.Result(out, reconcileColumns)
}
