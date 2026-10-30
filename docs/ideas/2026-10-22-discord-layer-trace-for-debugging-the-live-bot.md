---
date: 2026-10-22
kind: feature
related: [2026-10-16-p10-remainder-closing-turn-needs-login-interrupt-kill-live-reconcile.md, 2026-10-16-p11-remainder-for-the-live-gateway.md]
---

# A debug-only trace of the Discord layer, so an agent can watch a live bot

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

## Open

- Redaction: a modal submission carries an API key and an interaction carries message text; the trace
  records kind, ids, lengths and the scan outcome, never the text or a key (the audit rule: only the
  id and the last four characters).
- Whether the trace is a new event stream on the control pipe or reuses the session-event channel.
- Whether it belongs in 05-debug-interface-discord.md as a new section (a design change goes through
  the proposal loop).

Plan: two sittings. Architect amends `05-debug-interface.md` (the `serve` line at 22 gains
`--trace`) and `05-debug-interface-discord-live.md` (Inbound log), deciding whether `--trace`
refuses a non-scratch root (the idea's constraint) or runs on any root (no text, URL or token). Then
the coder, after P12d lands (the inbound half is P12d; the trace rides it): `internal/cli/serve.go`
(`--trace`, root rule per the amendment); `internal/discord/gateway.go` (a `restAPI` decorator when
flagged, a `GatewayConfig.Log` line per call: `<time> discord: rest <method name> <channel or
message id, or -> -> <HTTP status|error> <ms>`, never URL or body; always on, `discord: gateway
ready|resumed|closed <reason>`); `gateway_test.go` (a line per call flagged, none else, no token or
text). Verify: `go test ./internal/discord -run TestLive`, check line. Not `trace tail --follow`
(after P23); not disgo's debug logging (prints URLs).
