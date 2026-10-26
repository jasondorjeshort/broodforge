// SPDX-License-Identifier: GPL-3.0-or-later

package setup

import (
	"errors"
	"testing"

	"github.com/jasondorjeshort/broodforge/internal/control"
)

// TestListenError: a bind that fails with nothing listening says why, never `running`.
func TestListenError(t *testing.T) {
	for _, c := range []struct {
		err  error
		code string
	}{
		{control.NewError(control.CodeStartControl, "listen unix /srv/x/data/control.sock: bind: permission denied"), CodeNeedsElev},
		{control.NewError(control.CodeStartControl, `\.\pipe\broodforge-x: Access is denied.`), CodeNeedsElev},
		{control.NewError(control.CodeStartControl, "socket path longer than 103 bytes: /a/b"), CodeSocketPath},
		{control.NewError(control.CodeStartControl, "mkdir /srv/x/data: read-only file system"), control.CodeStartControl},
		{errors.New("plain"), ""},
	} {
		err := listenError("addr", c.err)
		var ce *control.Error
		got := ""
		if errors.As(err, &ce) {
			got = ce.Code
		}
		if got != c.code || got == CodeRunning {
			t.Errorf("%v: code %q, want %q", c.err, got, c.code)
		}
	}
}
