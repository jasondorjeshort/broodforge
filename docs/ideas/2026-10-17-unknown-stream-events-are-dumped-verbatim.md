---
date: 2026-10-17
kind: bug
related: [2026-10-16-rate-limit-events-carry-plan-usage-per-credential.md]
---

# Unknown stream events are dumped verbatim, and the engine now sends large ones

The owner's first real P7 turn (2026-10-17, engine 2.1.296, a one-word Read) printed about two
pages of JSON for a two-line answer. The bulk was two `system` / `commands_changed` events, each
carrying the name and full description of every slash command and synced skill on the account,
emitted before the first assistant message. `ParseLine` (`internal/engine/parse.go`) maps any line
it does not know to one `KindUnknown` event whose payload is the whole line, and `engine run`
streams it as `{"type":"unknown","raw":"<line>"}`. The P7 fixtures were recorded before the engine
sent this event, so no test sees it. The `rate_limit_event` line is the third unknown in the same
turn and has its own idea (`related`).

What it would take: keep the line in the recording (`engine record` writes raw stdout and must
stay verbatim) but shrink the event. Either `Unknown` carries `type` and `subtype` and a prefix of
the line, with the length, or `commands_changed` becomes a named kind that is parsed to a count
and otherwise dropped from the stream. Text mode should print one short line per unknown event.
Anchors: `internal/engine/parse.go` (`ParseLine`), `internal/engine/events.go` (`KindUnknown`,
`MarshalJSON`), `internal/cli/engine.go` (`engine run` output), `docs/design/03-architecture.md`
line 131 ("Everything else is `unknown`"), `parse_test.go` (the unknown-line table).

## Open

- Whether a fixture recorded on 2.1.296 with the event in it should become a parser golden, so
  the shrink is tested against the real shape.

Plan: architect, an amendment and no code. `EngineEvent.Unknown` is a shape in
`docs/design/03-architecture.md` and `engine run` prints it "exactly as the adapter parsed them"
(`05-debug-interface.md`), so the shrink is an amendment first. Shape: `Unknown *struct{ Type,
Subtype string; Bytes int; Head string }`, `Head` the line's first 200 bytes, JSON
`{"type":"unknown","eventType":..,"subtype":..,"bytes":..,"head":..}`; the recording stays verbatim
because `ClaudeCode.Raw` sees the line before the parser. No `commands_changed` kind: the bot reads
nothing in it. Amend 03 (the struct line, the `unknown` sentence) and 05 (the one-line form), then
the coder pass to `08-implementation-plan.md`: `parse.go`, `events.go`, `parse_test.go`, goldens
regenerated (`-run TestGolden -update`, diff read), and a real-size `commands_changed` line in the
next `engine record` fixture (`recorded-read` has none). Not `rate_limit_event` (own idea).
