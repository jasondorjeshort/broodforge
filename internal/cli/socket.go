// SPDX-License-Identifier: GPL-3.0-or-later

package cli

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jasondorjeshort/broodforge/internal/config"
	"github.com/jasondorjeshort/broodforge/internal/control"
)

// socketCall sends the command over the control socket of env.Root and prints each event as
// it arrives, then the result.
func socketCall(ctx context.Context, cmd *Command, env *Env) error {
	addr, err := socketAddress(env.Root)
	if err != nil {
		return err
	}
	args, err := cmd.wireArgs(env)
	if err != nil {
		return err
	}
	if extra := extraArgs[cmd]; extra != nil {
		if args, err = addArgs(args, env, extra); err != nil {
			return err
		}
	}
	onEvent := func(ev json.RawMessage) error {
		cols := cmd.EventColumns
		if cmd.EventColumnsFor != nil {
			cols = cmd.EventColumnsFor(ev)
		}
		return env.Out.Event(ev, cols)
	}
	result, err := control.Do(ctx, addr, cmd.wireCmd(), args, onEvent)
	var ce *control.Error
	if errors.As(err, &ce) && ce.Code == control.CodeNotRunning {
		// config/ may name another instance than the bot runs as (an edit not yet reloaded,
		// or one the bot refused): the bot listens under its last-good copy's name.
		if alt, ok := lastGoodAddress(env.Root, addr); ok {
			if r, aerr := control.Do(ctx, alt, cmd.wireCmd(), args, onEvent); !isNotRunning(aerr) {
				result, err = r, aerr
			}
		}
	}
	if err != nil {
		return err
	}
	if render := renderers[cmd]; render != nil {
		return render(env, result)
	}
	return env.Out.Result(result, cmd.Columns)
}

// renderers prints, per command, a result whose text form is not one record of Columns: one
// object as several labelled lines (`status`, `session show`), the same object in --json. A
// command file fills it from its own init().
var renderers = map[*Command]func(env *Env, result json.RawMessage) error{}

// extraArgs adds, per command, request args that come from somewhere other than a flag: a
// credential value from stdin, which is never an argument (05-debug-interface.md,
// Credentials). A command file fills it from its own init().
var extraArgs = map[*Command]func(*Env) (map[string]string, error){}

func addArgs(args json.RawMessage, env *Env, extra func(*Env) (map[string]string, error)) (json.RawMessage, error) {
	more, err := extra(env)
	if err != nil || len(more) == 0 {
		return args, err
	}
	m := map[string]json.RawMessage{}
	if err := json.Unmarshal(args, &m); err != nil {
		return nil, err
	}
	for k, v := range more {
		b, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		m[k] = b
	}
	return json.Marshal(m)
}

// socketAddress is the control address of the instance at root.
func socketAddress(root string) (string, error) {
	name, err := instanceName(root)
	if err != nil {
		return "", err
	}
	addr, err := control.Address(root, name)
	if err != nil {
		return "", control.NewError(CodeBadConfig, instancePath(root)+": "+err.Error())
	}
	return addr, nil
}

func instancePath(root string) string { return config.InstancePath(root) }

// instanceName is the one key the socket address needs, read through the config package so
// the pipe name is the same on every host and no path is hashed. It reads only that key (a
// client still reaches a bot running on its last-good copy after a bad edit, ADR 0021) and
// keeps the codes of 05-cli-conventions.md: no-config without the file, bad-config otherwise.
func instanceName(root string) (string, error) {
	name, err := config.InstanceName(root)
	var nf *config.NotFoundError
	var inv *config.InvalidError
	switch {
	case err == nil:
		return name, nil
	case errors.As(err, &nf):
		return "", control.NewError(CodeNoConfig, nf.Error())
	case errors.As(err, &inv):
		return "", control.NewError(CodeBadConfig, inv.Error())
	}
	return "", err
}

func isNotRunning(err error) bool {
	var ce *control.Error
	return errors.As(err, &ce) && ce.Code == control.CodeNotRunning
}

// lastGoodAddress is the control address under the last-good copy's instanceName, when it
// differs from addr.
func lastGoodAddress(root, addr string) (string, bool) {
	name, err := config.LastGoodInstanceName(root)
	if err != nil {
		return "", false
	}
	alt, err := control.Address(root, name)
	if err != nil || alt == addr {
		return "", false
	}
	return alt, true
}
