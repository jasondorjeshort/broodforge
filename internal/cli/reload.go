// SPDX-License-Identifier: GPL-3.0-or-later

package cli

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/jasondorjeshort/broodforge/internal/config"
	"github.com/jasondorjeshort/broodforge/internal/control"
	"github.com/jasondorjeshort/broodforge/internal/credentials"
)

// configReload is `broodforge config reload` (05-debug-interface.md, Configuration): the
// running bot reads config/ again. Valid: it runs on it from then on, rewrites the last-good
// copy and writes configuration-changed with the file hash and the changed keys, and prints
// `reloaded  <changed keys or ->`. Invalid: the old configuration stays, configuration-invalid
// is written, and the problems are the error, `error: bad-config: <file>: <what>; ...`, exit 1.
// A running session keeps the reach and mode it started with (ADR 0021).
var configReload = Command{
	Name:    "config reload",
	Expect:  true,
	Class:   ClassWrite,
	Columns: []string{"result", "changed"},
}

func init() {
	commands = append(commands, &configReload)
	renderers[&configReload] = renderReload
}

// renderReload prints the result as the columns do, then (text) one `warning: <file>: <message>`
// line per warning, as config check does; --json carries them in the `warnings` field.
func renderReload(env *Env, result json.RawMessage) error {
	if err := env.Out.Result(result, configReload.Columns); err != nil {
		return err
	}
	if env.JSON {
		return nil
	}
	var r reloaded
	if err := json.Unmarshal(result, &r); err != nil {
		return err
	}
	for _, w := range r.Warnings {
		line, err := marshal("warning: " + w)
		if err != nil {
			return err
		}
		if err := env.Out.Record(line, nil); err != nil {
			return err
		}
	}
	return nil
}

func warningTexts(c *config.Config) []string {
	out := make([]string, len(c.Warnings))
	for i, w := range c.Warnings {
		out[i] = w.String()
	}
	return out
}

// reload is the bot's config.reload.
func (b *bot) reload(_ context.Context, _ *control.Call) (json.RawMessage, error) {
	c, err := config.Load(b.root, credentials.Exists(b.st))
	if err != nil {
		if _, aerr := b.recordConfig(nil, err, "previous"); aerr != nil {
			return nil, aerr
		}
		var nf *config.NotFoundError
		if errors.As(err, &nf) {
			return nil, control.NewError(CodeNoConfig, nf.Error())
		}
		var inv *config.InvalidError
		if errors.As(err, &inv) {
			lines := make([]string, len(inv.Problems))
			for i, p := range inv.Problems {
				lines[i] = p.String()
			}
			return nil, control.NewError(CodeBadConfig, strings.Join(lines, "; "))
		}
		return nil, err
	}
	running := b.config()
	if c.Instance.InstanceName != running.Instance.InstanceName {
		err := errors.New(config.InstancePath(b.root) + ": instanceName cannot change while the bot runs (its socket is named by it)")
		if _, aerr := b.recordConfig(nil, err, "previous"); aerr != nil {
			return nil, aerr
		}
		return nil, control.NewError(CodeBadConfig, err.Error())
	}
	keys := changedKeys(running, c)
	if restart := restartKeys(keys); len(restart) > 0 {
		err := errors.New(strings.Join(restart, ",") + ": the bot reads these at its start; stop it and serve again")
		if _, aerr := b.recordConfig(nil, err, "previous"); aerr != nil {
			return nil, aerr
		}
		return nil, control.NewError(CodeReloadNeedsRestart, err.Error())
	}
	if len(keys) == 0 && b.lastGoodAt() == nil {
		return json.Marshal(reloaded{Result: "reloaded", Warnings: warningTexts(c)}) // nothing changed: no event, no write
	}
	if keys == nil {
		keys = []string{} // the bot ran on last-good: the same content, now from config/
	}
	changed, err := b.recordConfig(c, nil, "", keys...)
	if err != nil {
		return nil, err
	}
	b.mu.Lock()
	b.cfg, b.lastGood = c, nil
	b.mu.Unlock()
	r := reloaded{Result: "reloaded", Warnings: warningTexts(c)}
	if changed != "" {
		r.Changed = &changed
	}
	return json.Marshal(r)
}

// reloaded is config reload's result; Changed is null when no key changed. Warnings are
// config check's, shaped the same.
type reloaded struct {
	Result   string   `json:"result"`
	Changed  *string  `json:"changed"`
	Warnings []string `json:"warnings"`
}

// CodeReloadNeedsRestart refuses a reload changing what `serve` fixes at its start.
const CodeReloadNeedsRestart = "reload-needs-restart"

// startOnly are instance.json's top-level keys the bot reads only at its start (steps 1 to 8):
// the key file, the gateway, the engine and the fake's fixtures, the spawn backend, the install
// directory, the audit sink and the schema. instanceName is refused on its own above.
var startOnly = map[string]bool{"schemaVersion": true, "credentials": true, "discord": true, "engine": true,
	"spawn": true, "paths": true, "audit": true}

// restartKeys are the changed keys a reload cannot apply.
func restartKeys(changed []string) []string {
	var out []string
	for _, k := range changed {
		if startOnly[k] {
			out = append(out, k)
		}
	}
	return out
}
