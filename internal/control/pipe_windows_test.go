// SPDX-License-Identifier: GPL-3.0-or-later

package control_test

import (
	"strings"
	"testing"

	"github.com/jasondorjeshort/broodforge/internal/control"
	"github.com/jasondorjeshort/broodforge/internal/control/controltest"
)

// A pipe that exists but refuses the caller is connect-failed, not not-running: here the
// \\localhost form of a running bot's pipe, which PIPE_REJECT_REMOTE_CLIENTS refuses.
func TestRefusedPipeIsConnectFailed(t *testing.T) {
	h := controltest.Start(t)
	remote := strings.Replace(h.Addr, `\\.\`, `\\localhost\`, 1)
	_, err := control.Do(timeout(t), remote, control.CmdEcho, nil, nil)
	ce := wantCode(t, err, control.CodeConnectFailed)
	if !strings.HasPrefix(ce.Message, "cannot connect to "+remote+": ") {
		t.Errorf("message = %q", ce.Message)
	}
	_, err = control.Dial(timeout(t), remote)
	wantCode(t, err, control.CodeConnectFailed)
}
