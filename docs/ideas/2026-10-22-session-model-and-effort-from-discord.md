---
date: 2026-10-22
kind: feature
related: [2026-10-14-ux-choices-that-could-become-options.md]
---

# `/session model` and `/session effort`, in the next pass

R-29 (a person changes a session's model or effort from Discord) is `later` in
`docs/design/01-requirements.md` and `docs/design/04-ux.md` (the line "Later, as an option"). The
owner, 2026-10-22, after the first live run: it is essential for the next pass, because building
Broodforge inside Broodforge needs the model switched without editing JSON. Today only a channel's
`model` and `effort` and `defaults.model` set it (`internal/config/types.go`, `Channel`), read at
session start.

Anchors: `internal/discord/gateway.go` (the command registration; `/session` subcommands),
`internal/discord/command.go` (the parse), `internal/session/` (the per-process `Model, Effort` of
03's engine row, ADR 0020), `docs/design/05-debug-interface-discord.md` (the `command` kind of
`discord inject`, which must reach the new subcommands). Shape from the design as it stands: driver
only, effective from the next turn, one `/session model|effort` pair; the debug interface drives it
through `discord inject command` like every other command.

## Direction (owner, 2026-10-22)

- Choose at start: `/session start model:opus`, an alias or name resolved through `policy`'s model
  map (03, the display-name map), falling back to the channel's `model`, then `defaults.model`, then
  the engine's own. An unknown name is refused at the command, not at the spawn.
- Inside a session, `/model` and `/rename` behave as they do in the Claude Code command line: `/model`
  shows the current model and sets another from the next turn, `/rename` renames the session and its
  venue channel (`venueName` in `internal/discord/render.go`). A message that starts with `/` is
  always text in this design (05-debug-interface-discord.md), so these are registered slash commands,
  valid in a venue only (`not-a-venue` elsewhere); `/session model|effort` stays the same command
  under its longer name.
- Longer term, the pinned message at the top of a venue (today the header with the End button,
  `internal/discord/render.go`) shows live status: usage for the account, usage for this session or
  channel, and the model; the channel's lobby panel could show the project's totals. It is edited in
  place (the store record for the panel is item 8 of the P11 remainder idea) and needs `/usage`
  (04-ux.md, Usage) built first. Not in the next pass; the owner says when.

## Open

- Wording of the reply and the header line that shows the current model (what humans see, so the
  owner's call).
- Values: free text checked by the engine, or a list of display names from `policy`'s model map,
  or both (choices in the slash command).
- Whether a change is also written to the channel config or stays on the session (the design says
  the session).
- A bad value must be refused before the next turn starts, not by an engine failure.

Plan: architect, a small pass: promote R-29 from `later` to `v1` in `01-requirements.md`, replace
the "Later" line of `04-ux.md` with the rules for `/session start model:`, `/session model|effort`
and `/model`, and add them to `05-debug-interface-discord.md`; then coder, one pass: register them,
thread the model and effort into the session's next spawn, refuse bad values, a header line, tests
(`-race`) and the `discord inject command` path. `/rename` and the pinned status message are the
Direction's later items, not this pass. The Open items above go to the owner first; the coder starts when
they are answered.
