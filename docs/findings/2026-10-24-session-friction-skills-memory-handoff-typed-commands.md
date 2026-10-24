---
date: 2026-10-24
status: open
area: sessions / process
---

# A live session's friction: the agreement's skills are unreachable, the memory directory is denied, the handoff is not injected, a typed `/effort` reached the model

## Question

What got in the coordinator's way in a live Broodforge session on 2026-10-24 (the one that shipped
P12b), what each item shows about the design or the code, and who decides the fix. Recorded as the
coordinator reported them; one suggestion per item. The background-subagent loss of the same session
has its own finding (`2026-10-24-background-subagents-die-when-the-turn-ends.md`).

## What was tried

The coordinator's reports from the session; a read of `internal/policy/policy.go`,
`internal/hook/facts.go`, `internal/workflow/copy.go`, `internal/workflow/render.go`,
`internal/discord/adapter.go`, `docs/design/03-hook-contract.md`,
`docs/design/04-ux-session-controls.md`, `docs/design/08-implementation-plan-discord.md`, proposal 0025
and its critic review; and the tool and skill list this session's own subagent was given.

## Result

Items 1, 3 and 4 are repeats seen again from `2026-10-23-subagents-in-a-session-end-before-writing.md`
(its lines ~80-92 and ~150-152); `ToolSearch` and `PowerShell` in the catch-all, the memory directory
(item 2) and the P12b live check (item 6) are new.

1. (Repeat.) `Skill`, `ToolSearch` and `PowerShell` denied by reach. The hook's table puts every tool not
   named in a class into `ClassOther` (`internal/policy/policy.go:81-95`), and `ClassOther` is
   `deny` in both reaches (`policy.go:266-269`; `03-hook-contract.md:132`, "anything else,
   `mcp__*` included"). No design doc names `Skill`, `ToolSearch` or `PowerShell`: the deny is the
   catch-all's, by omission, not a decision. The design means skills to run: proposal 0025's option
   D lays the copy's skills out as a plugin so "the `Skill` tool ran it" (critic review V9), and
   the base agreement gives every agent `Skill` in its tools. `ToolSearch` loads a deferred tool's
   schema and reads nothing by itself; this session confirmed its deny with the reach text.
   Two separate causes keep the procedure layer out of a session, and both need a fix: the hook
   deny above for the coordinator, and critic review V8 (a project's `.claude/skills/` does not
   load under `--setting-sources user`) for everyone. A `runner` found the `commit` skill "not in
   the available skill list" and followed the brief instead; this session's subagent sees the
   account's skills and none of the repository's, V8 again. Suggestion: `Skill` and `ToolSearch`
   join the `TodoWrite`/`Agent` row (`allow`; a skill's own calls fire the hook) and `PowerShell`
   the `Bash` row; separately, option D or its successor for V8. Decides: the owner, through the
   proposal loop (the policy table is a design doc; the architect drafts).

2. The memory directory denied as a credential file. The engine writes the coordinator's memory
   under `<config dir>/projects/<encoded cwd>/memory/`; the hook marks any path under the config
   directory a credential path (`internal/hook/facts.go:143`, `l.under(ps.cfg)`), `CellFor`
   denies every class on one (`policy.go:231-235`), and the deny reads "is a credential file"
   (`policy.go:214`; the rule in `03-hook-contract.md:109-113`). The directory holds model-written
   notes, no secret, and the engine chose its place, so the rule is over-broad for that subtree.
   The transcripts sit beside it in `projects/<encoded cwd>/`, and those stay denied. Suggestion:
   carve `projects/*/memory/` out of the credential rule for the read and edit classes, or decide
   that a session has no memory and say so in the header. Decides: the owner (a credential
   boundary, ADR 0022), through the proposal loop.

3. (Repeat.) `docs/HANDOFF.md` "not in the copy". The workflow copy lists `CLAUDE.md` and `.claude` on the
   default branch and nothing else (`internal/workflow/copy.go:50`, ADR 0025 rule 5), so the
   `@docs/HANDOFF.md` import on `CLAUDE.md:6` resolves to a file the copy does not hold; the
   renderer loads nothing and writes a "was not loaded" note into `instructions.md`
   (`internal/workflow/render.go:122-124`), which is the sentence the coordinator read. The listed
   set is deliberate; the handoff's absence is a consequence nobody recorded, and ADR 0025 does
   not mention it. The tree itself holds `docs/HANDOFF.md`, so the model can `Read` it; it is only
   not injected. Suggestion: either list `docs/HANDOFF.md` (and `docs/handoff/`) in the copy, or
   have the project's `CLAUDE.md` tell a session to read it, since the handoff is coordinator state
   and a session's coordinator is who needs it. Decides: ADR 0025 (proposal loop) for the copy's
   set; the project's own `CLAUDE.md` is the coordinator's.

4. (Repeat.) A typed `/effort high` reached the model as chat. A plain venue message is text even when it
   starts with `/`; slash commands arrive as interactions (`internal/discord/adapter.go:321-323`).
   `/effort` is not a registered command (`04-ux-session-controls.md:27-28` has `/model <alias>`
   and `/session effort <level>`), so Discord delivered it as text, the bot submitted it as a turn,
   and the model could do nothing: a built-in Claude Code command is not something the model runs.
   Now that P12b ships an effort menu, suggestion: the bot maps a typed `/effort <level>` (and
   `/model`, already a command) to its own switch, or answers a venue message whose first word is a
   slash-prefixed engine command with an ephemeral "not a command here; use the header's menu"
   instead of submitting it. Decides: the owner (04 wording); a small change once chosen.

5. The review loop held. The design review found that switches succeeded on ended sessions; the
   code review found the non-driver check preceded the closed check and a false doc sentence. Both
   were real; design, review, fixes, commit is worth keeping as it is.

6. P12b's owner step is outstanding: under `serve --discord live`, a non-driver picks a model and
   says whether the menu returns to the true value; the driver picks the option already shown and
   an agent reads `session tail` for a `model-changed`; both results go in a finding
   (`08-implementation-plan-discord.md:114-118`).

## Next

- The owner: items 1 to 4 each name their decider; 1 and 2 are one proposal against
  `03-hook-contract.md`'s table and credential rule, 3 an ADR 0025 amendment or a `CLAUDE.md`
  line, 4 a 04 wording.
- The P12b live check (item 6), then its finding.
- Resolve this finding when each item has either landed or become an idea of its own.
