// SPDX-License-Identifier: GPL-3.0-or-later

package setup

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/jasondorjeshort/broodforge/internal/control"
)

// The single writer (ADR 0027, Decision 4): `setup`, but not `setup --check`, binds the
// instance's control address before its first write and holds it until it exits, a listener
// beside `serve`, so a bot cannot start on the root while setup writes it (serve fails
// start-control as for any listener) and a second setup is refused setup-running. The address
// is computed from the instance name, so it is bound before config/instance.json exists.

// StatusSetup is the config value of a setup's status reply: `status --expect setup` matches it.
const StatusSetup = "setup"

// binding is the held address.
type binding struct {
	srv  *control.Server
	done chan struct{}
}

func (b *binding) close() {
	_ = b.srv.Close()
	<-b.done
}

// probeTimeout bounds the probe of who answers at the address.
const probeTimeout = 10 * time.Second

// bind refuses when anything answers at the instance's address (running for a bot,
// setup-running for a setup, needs-elevation when the address is denied), then listens there
// with a registry serving `control echo` and `status`.
func bind(ctx context.Context, root, instance string) (*binding, error) {
	addr, err := control.Address(root, instance)
	if err != nil {
		return nil, control.NewError("bad-args", err.Error())
	}
	if err := probe(ctx, addr); err != nil {
		return nil, err
	}
	reg := control.NewRegistry()
	reg.Handle("status", statusHandler(instance))
	srv, err := control.Listen(addr, reg)
	if err != nil {
		// Another listener won the race between the probe and the listen: say which.
		if perr := probe(ctx, addr); perr != nil {
			return nil, perr
		}
		return nil, listenError(addr, err)
	}
	b := &binding{srv: srv, done: make(chan struct{})}
	go func() {
		_ = srv.Serve()
		close(b.done)
	}()
	return b, nil
}

// CodeSocketPath is a control socket path the host cannot bind (over 103 bytes on Linux).
const CodeSocketPath = "socket-path"

// listenError is why the address could not be bound when nothing listens there: a denied
// address is needs-elevation, a socket path too long for the host socket-path; anything else
// is control.Listen's own error. control carries only the message, so it is read.
func listenError(addr string, err error) error {
	var ce *control.Error
	if !asControl(err, &ce) {
		return err
	}
	msg := strings.ToLower(ce.Message)
	switch {
	case strings.Contains(msg, "permission denied") || strings.Contains(msg, "access is denied"):
		return control.NewError(CodeNeedsElev, addr+" is denied; "+deniedHint)
	case strings.HasPrefix(msg, "socket path longer than"):
		return control.NewError(CodeSocketPath, ce.Message+"; choose a shorter --root")
	}
	return ce
}

// probe asks whatever answers at addr for its status: nothing is nil, a setup is
// setup-running, anything else running; an address it may not open is needs-elevation.
func probe(ctx context.Context, addr string) error {
	ctx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()
	res, err := control.Do(ctx, addr, "status", json.RawMessage(`{"wait":0}`), func(json.RawMessage) error { return nil })
	if err != nil {
		var ce *control.Error
		if asControl(err, &ce) {
			switch ce.Code {
			case control.CodeNotRunning:
				return nil
			case control.CodeConnectFailed:
				return control.NewError(CodeNeedsElev, addr+" is denied; "+deniedHint)
			}
		}
		return control.NewError(CodeRunning, runningText(addr))
	}
	var st struct{ Config string }
	if json.Unmarshal(res, &st) == nil && st.Config == StatusSetup {
		return control.NewError(CodeSetupRunning, "another setup holds "+addr+"; finish it or press Ctrl-C there")
	}
	return control.NewError(CodeRunning, runningText(addr))
}

// runningText is the owner's text of the running refusal (decisions/0027-setup-commands.md).
func runningText(addr string) string {
	return "a bot answers at " + addr + "; stop it with broodforge stop in this shell (every open session is interrupted) or use setup --check"
}

// statusHandler is a setup's status reply: `config: setup` and - for every other line.
func statusHandler(instance string) control.Handler {
	return func(context.Context, *control.Call) (json.RawMessage, error) {
		return json.Marshal(struct {
			Config       string           `json:"config"`
			InstanceName string           `json:"instanceName"`
			ConfigFrom   *string          `json:"configFrom"`
			Discord      *json.RawMessage `json:"discord"`
			Sessions     *json.RawMessage `json:"sessions"`
			Worktrees    *json.RawMessage `json:"worktrees"`
			Backup       *json.RawMessage `json:"backup"`
			Engine       *json.RawMessage `json:"engine"`
		}{Config: StatusSetup, InstanceName: instance})
	}
}
