// SPDX-License-Identifier: GPL-3.0-or-later

package cli

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/jasondorjeshort/broodforge/internal/backup"
	"github.com/jasondorjeshort/broodforge/internal/config"
	"github.com/jasondorjeshort/broodforge/internal/control"
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

// backupRehearse is `broodforge backup rehearse [--path <file>] [--keep] [--expect ok] (file)`:
// restores the snapshot (default the one backup/latest names) into a new scratch root with the
// instance's config/, migrates it, runs reconciliation report-only and reads every table
// (decisions/0021-backup-and-restore.md, Rehearse). It prints verify's lines from the restored
// copy, then `reconcile  <n> missing`, then `root  <path>` when --keep kept the root; --json one
// object. The root is deleted on success unless --keep; on failure it is kept and named on
// stderr (`scratch root kept: <path>`) before the error line.
var backupRehearse = Command{
	Name:   "backup rehearse",
	Class:  ClassFile,
	Expect: true,
	Flags: []Flag{
		{Name: "path", Kind: FlagString, Value: "<file>", Usage: "the snapshot to restore (default: the one backup/latest names)"},
		{Name: "keep", Kind: FlagBool, Usage: "keep the scratch root on success too"},
	},
	Local: runBackupRehearse,
}

func init() { commands = append(commands, &backupSnapshot, &backupVerify, &backupRehearse) }

func runBackupRehearse(_ context.Context, env *Env) error {
	res, kept, err := backup.Rehearse(backup.Rehearsal{Root: env.Root, Path: env.String("path"),
		Keep: env.Bool("keep"), CopyConfig: config.CopyForScratch, RemoveTree: removeTree})
	if err != nil {
		if kept != "" {
			fmt.Fprintln(env.Stderr, keptLine+kept)
		}
		var nf *config.NotFoundError
		if errors.As(err, &nf) {
			return control.NewError(CodeNoConfig, "no "+nf.Path)
		}
		return scratchError(err)
	}
	if env.JSON {
		out, err := encode(res)
		if err != nil {
			return err
		}
		return env.Out.Result(out, nil)
	}
	lines := verifyLines(res.VerifyResult)
	lines = append(lines, verifyLine{Name: "reconcile", Value: strconv.Itoa(res.ReconcileMissing) + " missing"})
	if res.Root != "" {
		lines = append(lines, verifyLine{Name: "root", Value: res.Root})
	}
	return printVerifyLines(env, lines)
}

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
	return printVerifyLines(env, verifyLines(res))
}

// verifyLines are verify's text lines: the result, the migration id, then rows per table.
func verifyLines(res backup.VerifyResult) []verifyLine {
	lines := []verifyLine{{Name: res.Result}, {Name: "migration", Value: strconv.FormatInt(res.Migration, 10)}}
	for _, t := range res.Tables {
		lines = append(lines, verifyLine{Name: t.Table, Value: strconv.FormatInt(t.Rows, 10)})
	}
	return lines
}

// printVerifyLines prints the lines as records, the first (the result) as one column.
func printVerifyLines(env *Env, lines []verifyLine) error {
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
