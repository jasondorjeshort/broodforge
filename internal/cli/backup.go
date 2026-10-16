// SPDX-License-Identifier: GPL-3.0-or-later

package cli

import (
	"context"
	"strconv"

	"github.com/jasondorjeshort/broodforge/internal/backup"
)

// backupSnapshot is `broodforge backup snapshot [--expect ok] (file)`: the ADR 0021
// companion's snapshot from outside the bot (no checkpoint). It prints `ok  <snapshot path>
// <audit events exported>`; a failed integrity check deletes the snapshot and is
// `error: integrity-check: <result>`, exit 1.
var backupSnapshot = Command{
	Name:    "backup snapshot",
	Class:   ClassFile,
	Expect:  true,
	Columns: snapshotColumns,
	Local:   runBackupSnapshot,
}

// backupVerify is `broodforge backup verify [--path <file>] [--expect ok] (file)`: opens the
// snapshot backup/latest names (or --path) read-only and prints `ok`, `migration  <id>`, then
// `<table>  <rows>` per table; --json prints one object. Not ok is `error: integrity-check:
// <result>`, exit 1.
var backupVerify = Command{
	Name:   "backup verify",
	Class:  ClassFile,
	Expect: true,
	Flags: []Flag{
		{Name: "path", Kind: FlagString, Value: "<file>", Usage: "the file to verify (default: the snapshot backup/latest names)"},
	},
	Local: runBackupVerify,
}

func init() { commands = append(commands, &backupSnapshot, &backupVerify) }

var snapshotColumns = []string{"result", "snapshot", "auditEvents"}

func runBackupSnapshot(_ context.Context, env *Env) error {
	res, err := backup.Snapshot(env.Root, auditExport)
	if err != nil {
		return localError(err)
	}
	out, err := encode(res)
	if err != nil {
		return err
	}
	return env.Out.Result(out, snapshotColumns)
}

// verifyLine is one text line of verify after the first.
type verifyLine struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

var verifyColumns = []string{"name", "value"}

func runBackupVerify(_ context.Context, env *Env) error {
	res, err := backup.Verify(env.Root, env.String("path"))
	if err != nil {
		return localError(err)
	}
	if env.JSON {
		out, err := encode(res)
		if err != nil {
			return err
		}
		return env.Out.Result(out, nil)
	}
	lines := []verifyLine{{Name: res.Result}, {Name: "migration", Value: strconv.FormatInt(res.Migration, 10)}}
	for _, t := range res.Tables {
		lines = append(lines, verifyLine{Name: t.Table, Value: strconv.FormatInt(t.Rows, 10)})
	}
	for i, l := range lines {
		cols := verifyColumns
		if i == 0 {
			cols = verifyColumns[:1]
		}
		out, err := encode(l)
		if err != nil {
			return err
		}
		if err := env.Out.Record(out, cols); err != nil {
			return err
		}
	}
	return nil
}
