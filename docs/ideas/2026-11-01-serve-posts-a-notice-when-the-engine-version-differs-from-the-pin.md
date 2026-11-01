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

Question: `serve` checks `claude --version` against `engine.version` only at start (`internal/cli/serve.go`, step 5;
`spawn.EngineVersion` and `Preflight()` run from serve start, `internal/cli/spawn.go` and setup), so a drifted engine
stops a start but nothing watches a running serve: `DISABLE_AUTOUPDATER` covers the bot's own processes, and the owner's
interactive `claude` can update the binary under a running serve, after which turns run unchecked. Do you want a check
during a running serve (a version read per turn or on a timer, one notice line naming both versions when they differ)?
The spike's mismatch is not evidence either way: it ran `claude -p` directly under the owner's login, and its 2.1.309
came from the idea's text, not from `engine.version`. Recommendation: yes, a per-session-start read (cheap, one process
and no timer) that posts the notice line and leaves the turn running; refusing turns on a mismatch stays a separate
choice (the Open question above), and the start-time failure stays as it is. This also gives the ADR 0025 and 0028
"engine pin moves" predictions a visible event. P39 does not touch it.
Answer:
