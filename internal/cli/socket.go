// SPDX-License-Identifier: GPL-3.0-or-later

package cli

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

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
	result, err := control.Do(ctx, addr, cmd.wireCmd(), args, func(ev json.RawMessage) error {
		return env.Out.Event(ev, cmd.EventColumns)
	})
	if err != nil {
		return err
	}
	return env.Out.Result(result, cmd.Columns)
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

func instancePath(root string) string { return filepath.Join(root, "config", "instance.json") }

// instanceName reads the one key the socket address needs from config/instance.json, so the
// pipe name is the same on every host and no path is hashed. Until P4's config package exists
// this is the whole read: no validation beyond the key, no last-good fallback.
func instanceName(root string) (string, error) {
	p := instancePath(root)
	b, err := os.ReadFile(p)
	if err != nil {
		return "", control.NewError(CodeNoConfig, err.Error())
	}
	var cfg struct {
		InstanceName string `json:"instanceName"`
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		return "", control.NewError(CodeBadConfig, p+": "+err.Error())
	}
	if cfg.InstanceName == "" {
		return "", control.NewError(CodeBadConfig, p+": no instanceName")
	}
	return cfg.InstanceName, nil
}
