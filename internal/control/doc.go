// SPDX-License-Identifier: GPL-3.0-or-later

// Package control owns the local socket the CLI connects to: a unix socket at
// <root>/data/control.sock on Linux, the named pipe \\.\pipe\broodforge-<instanceName> on
// Windows (docs/design/05-cli-conventions.md, Surface). It carries the wire format (one JSON
// object per line), a registry of commands keyed by "<group>.<command>", a server that serves
// whatever is registered, and the client the CLI uses. Host differences live in the
// _windows.go and _unix.go files of this package and nowhere else in it.
package control
