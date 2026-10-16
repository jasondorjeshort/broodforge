// SPDX-License-Identifier: GPL-3.0-or-later

// Package spawn implements SpawnBackend (03-architecture.md, the spawn row): the same-account
// backend, which starts the engine through os/exec with the explicit environment of ADRs 0020
// and 0022 and, on Windows, a kill-on-close job object per engine (ADR 0023); the
// linux-sudo-bwrap backend, held until the droplet rows land (P14); interrupt; harness-create
// and harness-remove; and preflight. Host branching lives in the _windows.go and _unix.go files
// of this package and nowhere else in the bot.
package spawn
