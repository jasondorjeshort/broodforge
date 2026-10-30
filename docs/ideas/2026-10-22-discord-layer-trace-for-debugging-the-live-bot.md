---
date: 2026-10-22
kind: feature
related: [2026-10-16-p10-remainder-closing-turn-needs-login-interrupt-kill-live-reconcile.md, 2026-10-16-p11-remainder-for-the-live-gateway.md, 2026-10-30-the-bot-files-its-own-findings.md]  # the last reads these lines from a scratch session to reproduce a refusal; this one is the trace itself
---

# A trace of the Discord layer, so an agent can watch a live bot

The first live run (P12's proof) had two views: `audit` (decisions, from the store) and
`session tail` (engine events, from the bot). Neither shows the Discord layer itself, which is where
a live problem starts: which interactions and messages arrived, what the key-paste scan decided,
every REST call and its status, gateway reconnects and rate limits. `audit --follow` is the sibling
gap (that idea's 2026-10-22 line); this one is the layer below it.

Shape to consider: a `serve --discord live --trace` flag (anchors: `internal/cli/serve.go`,
`internal/discord/gateway.go`) that writes one line per inbound interaction or message and per REST
call to the bot's stdout or a file under the root, readable by an agent as `trace tail --follow`
over the control pipe like every other read (05-debug-interface.md: every feature ships an
agent-usable way to exercise it). Debug mode only: it refuses to start on a root that is not
scratch-shaped.

Why not a second bot that sees everything: it cannot see ephemeral replies (`/register`,
`/credential`, every refusal), slash-command interactions, button clicks or modal submissions, which
go to the owning application, nor the gateway and REST errors inside the bot's own connection. The
in-app hook shows all of those.

2026-10-24 (architect): ADR 0026 and pass P12d (`docs/design/08-implementation-plan-discord.md`) build
the inbound half of this, always on, not debug-only: one stderr line per inbound message, command, click
and modal with its route and outcome, never text or a raw custom id (`05-debug-interface-discord-live.md`,
Inbound log and held sessions). Still open here: a line per REST call and its status, gateway reconnects
and rate limits, and reading the lines over the control pipe (`trace tail --follow`).

2026-10-30 (architect): the design half is done. `05-debug-interface.md` (Bot lifecycle) has
`serve [--discord live|dry-run] [--trace]`; `05-debug-interface-discord-live.md` (Inbound log) has
the `rest` line, the always-on `gateway` line and the root rule: `--trace` runs on any root, not
scratch only, because the lines carry no text, URL, token or key, the inbound log is already always
on, and a scratch root is dry-run with no REST call to trace, so the constraint above would have
traced nothing. The first Open item is answered by that line shape; the second and third by "the
run's `serve.log` is the read" until `trace tail --follow` is wanted (after P23).

## Open

- Redaction: a modal submission carries an API key and an interaction carries message text; the trace
  records kind, ids, lengths and the scan outcome, never the text or a key (the audit rule: only the
  id and the last four characters).
- Whether the trace is a new event stream on the control pipe or reuses the session-event channel.
- Whether it belongs in 05-debug-interface-discord.md as a new section (a design change goes through
  the proposal loop).

Plan: coder, after P12d lands (the inbound half is P12d; the trace rides its `GatewayConfig.Log`
line): `internal/cli/serve.go` (`--trace`, any root, the architect's decision above);
`internal/discord/gateway.go` (a `restAPI` decorator when flagged, a `Log` line per call `<time>
discord: rest <method> <channel, message or interaction id, or -> -> <HTTP status|error> <ms>`, never
URL or body; always on, `<time> discord: gateway ready|resumed|closed <reason>`);
`internal/discord/gateway_test.go` (a line per call when flagged, none else, no token or text in the
log). Verify: `go test ./internal/discord -run TestLive`, the check line; the owner reads one run's
`serve.log` on the live bot. Not `trace tail --follow` (after P23); not disgo's debug logging (prints
URLs). Delete this file in the pass's commit.
