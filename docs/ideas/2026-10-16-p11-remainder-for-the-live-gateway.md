---
date: 2026-10-16
kind: feature
related: [2026-10-16-p10-remainder-closing-turn-needs-login-interrupt-kill-live-reconcile.md]
---

# P11 remainder: what the dry-run adapter left for P12 and later

P11 landed `internal/discord/` without these. The P10 remainder is the session core; this is the
Discord layer. P12 built the hourglass, deleting a pasted key, the venue's deletion and
`/credential`; the numbers below are kept so the Plan still reads.

1. Event before the handoff. Anchor: `internal/session/events.go`, `sessionEnded` in
   `internal/discord/render.go`. Done: the summary place says "Ending: checking and handing off N
   commits..." (04, Ending a session).
2. Summary detail. Anchor: `summaryText` in `internal/discord/text.go` (the "not 04" commits line).
   Done: each commit's hash and subject, the pull-request title, and posting to `venue.summaryChannel`.
5. Texts: notice channel, credential-needs-login, Stop timeout, post-restart notice (R-17). Anchor:
   `internal/discord/text.go`. Done: each has its 04 text rendered from an event. P12 posts the
   start-up report to the notice channel (`startupReport` in `internal/cli/serve.go`) and binds
   an interrupted session's venue again at start (`sweep` in `internal/discord/gateway.go`), so
   the post-restart notice has a channel to go to.
6. Indent subagent calls. Anchor: `Event.ToolUseID` in `events.go`; `Handle` in `render.go`. Done:
   events carry the parent tool-use id and a subagent's calls render indented under it.
7. `/usage` (P12 built `/credential`). Anchor: the `SlashUsage` case of `slash` in `adapter.go`
   (answers `not-built`). Done: it answers as 04 says.
8. Store record for the panel. Anchor: `PostPanel`, `PanelOf` in `adapter.go`; the message id lives
   in the bot's memory, so a restart posts a second panel. Done: the id is stored and edited.
9. `--clock real` is only exercised by hand. Anchor: `New` in `adapter.go`. Done: a repro in
   `tests/repro/` runs the coalescer on the real clock and checks the edit rate.

Plan: coder, one pass for items 1, 2, 6 and 8 (no Discord needed). 1: a new event emitted before the
handoff, rendered, with a `session outbox` check in the ending test. 2: carry hashes, subjects and
title in `session.EndInfo`; route by the config key. 6: parent id on the engine event and `Event`,
a fake-engine fixture with a subagent, a render test. 8: a store record keyed by channel, loaded in
`New`, with a restart test. Tests with `-race`.
