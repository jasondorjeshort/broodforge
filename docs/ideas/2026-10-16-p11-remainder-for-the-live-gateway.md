---
date: 2026-10-16
kind: feature
related: [2026-10-16-p10-remainder-closing-turn-needs-login-interrupt-kill-live-reconcile.md]
---

# P11 remainder: what the dry-run adapter left for P12 and later

P11 landed `internal/discord/` without these. The P10 remainder is the session core; this is the
Discord layer. P12 built the hourglass, deleting a pasted key, the venue's deletion and
`/credential`; the numbers below are the original list's.

5. Texts: notice channel, credential-needs-login, Stop timeout, post-restart notice (R-17). Anchor:
   `internal/discord/text.go`. Done: each has its 04 text rendered from an event. P12 posts the
   start-up report to the notice channel (`startupReport` in `internal/cli/serve.go`) and binds
   an interrupted session's venue again at start (`sweep` in `internal/discord/gateway.go`), so
   the post-restart notice has a channel to go to.
7. `/usage` (P12 built `/credential`). Anchor: the `SlashUsage` case of `slash` in `adapter.go`
   (answers `not-built`). Done: it answers as 04 says.
9. `--clock real` is only exercised by hand. Anchor: `New` in `adapter.go`. Done: a repro in
   `tests/repro/` runs the coalescer on the real clock and checks the edit rate.

2026-10-30: items 1 (the `session-ending` event and the venue's `Ending:` line), 2 (each commit's
hash and subject, the pull request's title, `venue.summaryChannel`), 6 (a subagent's calls
indented in the rendered turn, `tests/discord/golden/subagent-end/events.jsonl`) and 8 (the `panel` store
record, loaded at start) landed; the three above remain, untriaged.

Plan: coder, one pass, items 5, 7 and 9, checked against the tree 2026-10-31. Item 5 is partly built:
the start-up report (`startupReport` in `internal/cli/serve.go`) and the post-restart notice
(`interruptedNoticeText` in `internal/discord/text.go`) exist; what is missing is the Stop timeout line
("Stop took too long; the agent's process was killed. Output tokens for this turn are unknown."), the
credential-needs-login venue line, and the notice channel's config-reload and nightly-backup lines, all
in `docs/design/04-ux.md` (Failure flows, notice channel). Render each from its event in `internal/discord/text.go`
and `render.go`; item 7: the `SlashUsage` case in `adapter.go` still answers `not-built`, make it answer as 04's
Usage section says (own totals, the owner's `all`); item 9: a real-clock repro in `tests/repro/`.
Verify: `discord render --golden` (`docs/design/08-implementation-plan-golden-sessions.md`) and `go test ./internal/discord/...`.
Out of scope: the P10 remainders, wording changes to 04 (the texts are 04's).
