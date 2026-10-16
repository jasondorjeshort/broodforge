// SPDX-License-Identifier: GPL-3.0-or-later

package cli

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jasondorjeshort/broodforge/internal/config"
	"github.com/jasondorjeshort/broodforge/internal/control"
	"github.com/jasondorjeshort/broodforge/internal/spawn"
)

// spawnPreflight is `broodforge spawn preflight [--expect ok] (file)` (05-debug-interface.md,
// Spawn preflight and reach): the configured backend's checks without a session. Each passing
// check prints `ok  <check>  <detail>`; each failing one is `error: preflight-<check>:
// <detail>` on stderr, the last of them the command's error line, exit 1. The same-account
// checks are backend, engine-path, engine-version (the binary's --version, its one engine
// run), git-bash (Windows), share and account; linux-sudo-bwrap is `error:
// preflight-backend: held`.
var spawnPreflight = Command{
	Name:    "spawn preflight",
	Class:   ClassFile,
	Expect:  true,
	Columns: preflightColumns,
	Local:   runSpawnPreflight,
}

func init() { commands = append(commands, &spawnPreflight) }

var preflightColumns = []string{"result", "check", "detail"}

type preflightLine struct {
	Result string `json:"result"`
	Check  string `json:"check"`
	Detail string `json:"detail"`
}

func runSpawnPreflight(_ context.Context, env *Env) error {
	c, err := config.Load(env.Root, credentialExists)
	if err != nil {
		return configFailure(env, err)
	}
	backend, err := spawn.New(spawnConfig(c))
	if err != nil {
		return spawnError(err)
	}
	var failed []*control.Error
	for _, ch := range backend.Preflight() {
		if !ch.OK {
			failed = append(failed, control.NewError(ch.Code(), ch.Detail))
			continue
		}
		b, err := json.Marshal(preflightLine{"ok", ch.Name, ch.Detail})
		if err != nil {
			return err
		}
		if err := env.Out.Record(b, preflightColumns); err != nil {
			return err
		}
	}
	if len(failed) == 0 {
		return nil
	}
	for _, e := range failed[:len(failed)-1] {
		if env.JSON {
			b, err := json.Marshal(e)
			if err != nil {
				return err
			}
			fmt.Fprintln(env.Stderr, string(b))
			continue
		}
		fmt.Fprintf(env.Stderr, "error: %s: %s\n", e.Code, e.Message)
	}
	return failed[len(failed)-1]
}
