---
date: 2026-11-01
kind: feature
related: [2026-10-30-engine-upgrade-canary-replays-prompts-on-the-new-engine.md]  # that one replays prompts after an upgrade; this only says an upgrade happened
---

# `serve` posts a notice when the engine version differs from the pin

Recorded from the owner's conversation of 2026-11-01, not sanity checked: nobody has read the
anchor against the claim yet.

The auto-mode spike of 2026-10-31 found the engine on the PC at 2.1.316 while the project's recorded
pin was 2.1.309, and nothing had said so. On start, `serve` runs `claude --version`, compares it with
the pin, and posts one line to the notice channel when they differ, naming both. The ADR 0025 and
0028 predictions keyed on "the engine pin moves" then have a visible trigger, and the upgrade canary
has an event to run on.

Anchor: `internal/cli/serve.go` (the start-up report, where the line would join); `internal/setup/steps.go`
(where the pin is recorded); `internal/engine/claudecode.go` (the adapter that runs the engine).

## Open

- Whether a differing version should also be refused for real turns until the owner accepts it.
- Where the pin lives today and whether `setup engine` is what moves it.
