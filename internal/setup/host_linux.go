// SPDX-License-Identifier: GPL-3.0-or-later

package setup

import (
	"os"

	"golang.org/x/sys/unix"
)

// The Linux host part of setup (.claude/rules/go.md: host code by nature): the terminal's echo
// for a hidden answer, the pinned engine's place, the hint of a denied address.

// StdinTerminal reports whether stdin is a terminal, and the switch that turns its echo off
// for one hidden answer.
func StdinTerminal() (bool, func() (restore func())) {
	fd := int(os.Stdin.Fd())
	if _, err := unix.IoctlGetTermios(fd, unix.TCGETS); err != nil {
		return false, nil
	}
	return true, func() func() {
		t, err := unix.IoctlGetTermios(fd, unix.TCGETS)
		if err != nil {
			return nil
		}
		old := *t
		t.Lflag &^= unix.ECHO
		if unix.IoctlSetTermios(fd, unix.TCSETS, t) != nil {
			return nil
		}
		return func() { _ = unix.IoctlSetTermios(fd, unix.TCSETS, &old) }
	}
}

// engineName is the engine binary on PATH.
const engineName = "claude"

// pinnedEngine is /opt/broodforge/engine/claude, where the droplet's setup host puts it.
func pinnedEngine(func(string) (string, bool)) string { return "/opt/broodforge/engine/claude" }

// deniedHint is what the owner does about a denied control address.
const deniedHint = "run this with sudo"
