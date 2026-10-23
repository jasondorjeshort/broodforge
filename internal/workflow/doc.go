// SPDX-License-Identifier: GPL-3.0-or-later

// Package workflow makes a session's copy of a project's workflow from the canonical's default
// branch and renders instructions.md, agents.json and hooks.json there, and probes each hook's
// runtime (ADR 0025 rules 5 and 6; the procedure and the probe are its companion's).
package workflow
