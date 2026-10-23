// SPDX-License-Identifier: GPL-3.0-or-later

package config_test

import (
	"os"
	"strings"
	"testing"

	"github.com/jasondorjeshort/broodforge/internal/config"
)

// readMachine is the real reader, kept before TestMain replaces it.
var readMachine = config.MachineLiterals

// TestMain empties the machine literals, so no test here depends on the machine's user and host
// names.
func TestMain(m *testing.M) {
	config.MachineLiterals = func() []config.MachineLiteral { return nil }
	os.Exit(m.Run())
}

// The real reader: a bare user name (no domain prefix), then the host name.
func TestMachineLiteralsReader(t *testing.T) {
	got := readMachine()
	if len(got) != 2 || got[0].What != config.OSUserName || got[0].Value == "" || strings.Contains(got[0].Value, `\`) ||
		got[1].What != config.HostName || got[1].Value == "" {
		t.Errorf("%d literals; a backslash in the user name: %v", len(got), len(got) > 0 && strings.Contains(got[0].Value, `\`))
	}
}
