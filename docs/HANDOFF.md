# Handoff

Coordinator state. Subagents follow their brief, not this file.
Updated: 2026-10-15, after the Hatchery merge 0da3f38, by main session

## Now

- `main` is unpushed past the decant `ee08842`, through the C25/C36 findings `4121c44` and the
  merge of hatchery/main `0da3f38` (scoped override grants, ADR 0006). `origin` is Broodforge,
  `hatchery` the base. HOLD 60% session, 75% weekly; at 09:30 the meters read 17% and 70%.
- Owner is Jason Dorje Short <jdorje@gmail.com>, repo-local git identity. Push only at session
  end on his words; his UX decisions are applied. Design complete: `00` to `07`, ADRs 0020 to 0022.
- The spike is live on the PC: C25 login done, fixtures in place, C25 and C36 confirmed in
  `docs/findings/2026-10-15-spike-claims-for-adr-0020.md` and `...-0022.md`. No engine turn yet.
- The spike config directory holds the PC owner's Claude Max account (her PC), not the owner's; every
  PC claim spends it. Moving to his account means deleting `$CFG` and running C25 again. The
  owner drives the claim runs himself with a Fable agent (the table says `coder`; his call).
- 2026-10-14: a bare `npm install` wiped the global npm folder; restored except two local links
  (`bot-folder`, `project`) whose targets the owner must name. Safe form:
  `docs/reference/claude-code-on-windows.md`, "npm on this PC".

## Next

1. Owner decides whose subscription the PC claims spend; nothing spends an engine turn until then.
2. Remaining PC claims in `07-spike-plan.md` order: S1, C27, C39, S2, C28, C26, each `$RUN` against
   `$CFG` and `$TREE` with the hook mode set per claim.
3. Owner runs the droplet operator entries and names the two npm link targets.
4. Droplet claims (`07-spike-droplet.md`); tester then writes a droplet host-setup reference.
5. Spike exit met: architect records refutations as ADR amendments and weighs C36's `.claude.json`
   backup question; then coder starts `src/` (engine adapter, fake engine, hook, storage, Discord).
6. Pusher pushes `main` on the owner's words, at session end.

## Waiting on user

- Whose Claude account the PC claims spend; the npm link targets; the droplet entries; the push.

## Open findings

- `docs/findings/2026-10-15-spike-claims-for-adr-0020.md` (C25 closed, seven rows open)
- `docs/findings/2026-10-15-spike-claims-for-adr-0022.md` (C36 closed, eight rows open)
