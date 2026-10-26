// SPDX-License-Identifier: GPL-3.0-or-later

package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jasondorjeshort/broodforge/internal/config"
	"github.com/jasondorjeshort/broodforge/internal/control"
)

// Error codes of the config commands, beside no-config and bad-config.
const (
	CodeUnknownProject = "unknown-project"
	CodeUnknownChannel = "unknown-channel"
	CodeConfigExists   = "config-exists"
)

// configCheck is `broodforge config check (file)` (05-debug-interface.md, Configuration): it
// validates config/ without the bot. Valid: `ok  <projects>  <channels>`, then one
// `warning: <file>: <message>` line per warning (--json: one object with a warnings array).
// Invalid: one `error: bad-config: <file>: <message>` line per problem on stderr, exit 1. A
// configuration that loads is then probed: each project's listed hooks' runtimes
// (probeWorkflows), a missing one a problem of its project file naming the command and the word.
var configCheck = Command{
	Name:    "config check",
	Class:   ClassFile,
	Columns: checkColumns,
	Local:   runConfigCheck,
}

// configShow is `broodforge config show [--effective] [--channel <id>] [--project <id>]
// [--last-good] (file)`: the parsed files (comments removed, keys sorted), one record per file
// as `<path>  <content>`; with --effective each channel with defaults merged in, as the bot
// sees it. --project and --channel narrow either form; --last-good reads
// data/config-last-good/ instead of config/.
var configShow = Command{
	Name:  "config show",
	Class: ClassFile,
	List:  true,
	Flags: []Flag{
		{Name: "effective", Kind: FlagBool, Usage: "each channel with defaults merged in"},
		{Name: "channel", Kind: FlagString, Value: "<id>", Usage: "only this channel, or the file holding it"},
		{Name: "project", Kind: FlagString, Value: "<id>", Usage: "only this project's channels or file"},
		{Name: "last-good", Kind: FlagBool, Usage: "read data/config-last-good/"},
	},
	Local: runConfigShow,
}

// configExport is `broodforge config export [--out <file>] (file)` (ADR 0027; the companion's
// `config export`): config/ as one JSON document with the host-bound keys emptied and no
// secret (config.Export), written to --out, default <root>/data/config-export.json; prints
// `written  <path>`. The configuration must load, as config check reads it.
var configExport = Command{
	Name:  "config export",
	Class: ClassFile,
	Flags: []Flag{
		{Name: "out", Kind: FlagString, Value: "<file>", Usage: "where to write it (default <root>/data/config-export.json)"},
	},
	Local: runConfigExport,
}

// configImport is `broodforge config import <file> (file)`: an export's files written into an
// empty root's config/ (config.Import), one `written  <path>` record each; a root whose
// config/instance.json exists is config-exists, and an export this binary's schema rejects is
// bad-config naming the key. setup --from does the same inside its run.
var configImport = Command{
	Name:  "config import",
	Class: ClassFile,
	Flags: []Flag{
		{Name: "file", Kind: FlagString, Value: "<file>"},
	},
	Local: runConfigImport,
}

func init() {
	commands = append(commands, &configCheck, &configShow, &configExport, &configImport)
	positionals[&configImport] = []string{"file"}
}

var (
	checkColumns       = []string{"result", "projects", "channels"}
	showFileColumns    = []string{"file", "content"}
	showChannelColumns = []string{"id", "project", "reach", "mode", "harness", "idleTimeoutMinutes",
		"promptTimeoutMinutes", "model", "effort", "roles"}
)

// checkResult is what config check prints.
type checkResult struct {
	Result   string   `json:"result"`
	Projects int      `json:"projects"`
	Channels int      `json:"channels"`
	Warnings []string `json:"warnings"`
}

// showFile is one record of config show without --effective.
type showFile struct {
	File    string          `json:"file"`
	Content json.RawMessage `json:"content"`
}

// credentialExists is the id-existence check config needs; a stub until the credential store
// exists (P6).
var credentialExists config.CredentialExists = config.AnyCredential

func runConfigCheck(ctx context.Context, env *Env) error {
	c, err := config.Load(env.Root, credentialExists)
	if err != nil {
		return configFailure(env, err)
	}
	// The runtime probe of each listed hook (ADR 0025 rule 6), only on a configuration that loads.
	probs, err := probeWorkflows(ctx, c)
	if err != nil {
		return err
	}
	if len(probs) > 0 {
		return configFailure(env, &config.InvalidError{Problems: probs})
	}
	warnings := make([]string, len(c.Warnings))
	for i, w := range c.Warnings {
		warnings[i] = w.String()
	}
	result, err := marshal(checkResult{"ok", len(c.Projects), len(c.Channels()), warnings})
	if err != nil {
		return err
	}
	if err := env.Out.Result(result, checkColumns); err != nil {
		return err
	}
	if !env.JSON {
		for _, w := range warnings {
			line, err := marshal("warning: " + w)
			if err != nil {
				return err
			}
			if err := env.Out.Record(line, nil); err != nil {
				return err
			}
		}
	}
	return nil
}

func runConfigShow(_ context.Context, env *Env) error {
	load := config.Load
	if env.Bool("last-good") {
		load = config.LoadLastGood
	}
	c, err := load(env.Root, credentialExists)
	if err != nil {
		return configFailure(env, err)
	}
	var p *config.Project
	if id := env.String("project"); id != "" {
		if p = c.Project(id); p == nil {
			return control.NewError(CodeUnknownProject, "no project "+id+" in "+c.Dir)
		}
	}
	ch := env.String("channel")
	if ch != "" {
		cp := c.ChannelProject(ch)
		if cp == nil || (p != nil && cp != p) {
			where := c.Dir
			if p != nil {
				where = "project " + p.ID
			}
			return control.NewError(CodeUnknownChannel, "no channel "+ch+" in "+where)
		}
		p = cp
	}

	if env.Bool("effective") {
		var rows []config.EffectiveChannel
		switch {
		case ch != "":
			rows = []config.EffectiveChannel{c.Effective(p, ch)}
		case p != nil:
			rows = c.ProjectChannels(p)
		default:
			rows = c.Channels()
		}
		out, err := marshal(rows)
		if err != nil {
			return err
		}
		return env.Out.Result(out, showChannelColumns)
	}

	rows := []showFile{}
	for _, f := range c.Files {
		if p != nil && f.Name != "projects/"+p.ID+".json" {
			continue
		}
		rows = append(rows, showFile{File: f.Path, Content: f.Stripped})
	}
	out, err := marshal(rows)
	if err != nil {
		return err
	}
	return env.Out.Result(out, showFileColumns)
}

var writtenColumns = []string{"result", "path"}

func runConfigExport(_ context.Context, env *Env) error {
	c, err := config.Load(env.Root, credentialExists)
	if err != nil {
		return configFailure(env, err)
	}
	doc, err := config.Export(c)
	if err != nil {
		return configFailure(env, err)
	}
	out := env.String("out")
	if out == "" {
		out = filepath.Join(env.Root, "data", config.ExportName)
		if err := os.MkdirAll(filepath.Dir(out), 0o700); err != nil {
			return control.NewError(control.CodeBadArgs, err.Error())
		}
	}
	if err := os.WriteFile(out, doc, 0o600); err != nil {
		return control.NewError(control.CodeBadArgs, "--out: "+control.OneLine(err.Error()))
	}
	return printWritten(env, out)
}

func runConfigImport(_ context.Context, env *Env) error {
	if p := config.InstancePath(env.Root); fileExists(p) {
		return control.NewError(CodeConfigExists, p+" exists")
	}
	file := env.String("file")
	e, err := config.ReadExport(file)
	var inv *config.InvalidError
	switch {
	case errors.As(err, &inv):
		for i := range inv.Problems { // each problem names the export file, then its part
			inv.Problems[i].File = filepath.ToSlash(file) + ": " + inv.Problems[i].File
		}
		return configFailure(env, err)
	case err != nil:
		return control.NewError(control.CodeBadArgs, control.OneLine(err.Error()))
	}
	wrote, err := config.Import(env.Root, e, config.ImportOptions{})
	if err != nil {
		return configFailure(env, err)
	}
	for _, w := range wrote {
		if err := printWritten(env, filepath.Join(env.Root, filepath.FromSlash(w.File))); err != nil {
			return err
		}
	}
	return nil
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// printWritten is the `written  <path>` record.
func printWritten(env *Env, path string) error {
	b, err := marshal(map[string]string{"result": "written", "path": filepath.ToSlash(path)})
	if err != nil {
		return err
	}
	return env.Out.Record(b, writtenColumns)
}

// configFailure maps a load error to the CLI's codes: no-config for a missing instance file,
// bad-config for anything invalid. Every problem is its own `error: bad-config: ...` line on
// stderr (the wire object per line with --json); the last is the returned error, so the
// command's one error line is the last of them.
func configFailure(env *Env, err error) error {
	var nf *config.NotFoundError
	if errors.As(err, &nf) {
		return control.NewError(CodeNoConfig, nf.Error())
	}
	var inv *config.InvalidError
	if !errors.As(err, &inv) || len(inv.Problems) == 0 {
		return err
	}
	last := len(inv.Problems) - 1
	for _, p := range inv.Problems[:last] {
		e := control.NewError(CodeBadConfig, p.String())
		if env.JSON {
			b, merr := json.Marshal(e)
			if merr != nil {
				return merr
			}
			fmt.Fprintln(env.Stderr, string(b))
			continue
		}
		fmt.Fprintf(env.Stderr, "error: %s: %s\n", e.Code, e.Message)
	}
	return control.NewError(CodeBadConfig, inv.Problems[last].String())
}

// marshal is json.Marshal without HTML escaping, so a path such as <root>/projects prints as
// the file has it.
func marshal[T string | []config.EffectiveChannel | []showFile | checkResult | map[string]string](v T) (json.RawMessage, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimSpace(b.Bytes()), nil
}
