// SPDX-License-Identifier: GPL-3.0-or-later

package setup

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
)

// The Windows host part of setup (.claude/rules/go.md: host code by nature): the console's
// echo for a hidden answer, the pinned engine's place, the hint of a denied address.

// StdinTerminal reports whether stdin is a console, and the switch that turns its echo off for
// one hidden answer.
func StdinTerminal() (bool, func() (restore func())) {
	h := windows.Handle(os.Stdin.Fd())
	var mode uint32
	if windows.GetConsoleMode(h, &mode) != nil {
		return false, nil
	}
	return true, func() func() {
		if windows.SetConsoleMode(h, mode&^windows.ENABLE_ECHO_INPUT) != nil {
			return nil
		}
		return func() { _ = windows.SetConsoleMode(h, mode) }
	}
}

// engineName is the engine binary on PATH: the .exe, never the npm .cmd shim (ADR 0020).
const engineName = "claude.exe"

// pinnedEngine is %ProgramData%\Broodforge\engine\claude.exe, where setup host puts it.
func pinnedEngine(lookup func(string) (string, bool)) string {
	pd, ok := lookup("ProgramData")
	if !ok || pd == "" {
		return ""
	}
	return filepath.Join(pd, "Broodforge", "engine", "claude.exe")
}

// deniedHint is what the owner does about a denied control address (the companion's PC shell rule).
const deniedHint = "run this from an elevated PowerShell"
