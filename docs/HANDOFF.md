# Handoff

Coordinator state. Subagents follow their brief, not this file.
Updated: 2026-10-15, after the Hatchery merge 58307e2, by Hatchery's main session

## Now

- `main` is unpushed past the decant `ee08842`, through the hatchery/main merge `58307e2` (hook
  paths, ADR 0009). `origin` (empty) is Broodforge, `hatchery` the base; `main` no longer tracks
  hatchery/main, so the first push is `git push -u origin main`. HOLD 60% session, 75% weekly.
- Owner is Jason Dorje Short <jdorje@gmail.com>, repo-local git identity. Push only at session
  end on his words; his UX decisions are applied. Design complete: `00` to `07`, ADRs 0020 to 0022.
- The spike is live on the PC: C25 login done, fixtures in place, C25 and C36 confirmed in
  `docs/findings/2026-10-15-spike-claims-for-adr-0020.md` and `...-0022.md`. No engine turn yet.
- The spike config directory holds the PC owner's Claude Max account (her PC), not the owner's; every
  PC claim spends it. Moving to his account means deleting `$CFG` and running C25 again. The
  owner drives the claim runs himself with a Fable agent (the table says `coder`; his call).
- 2026-10-14: a bare `npm install` wiped the global npm folder, restored but for two links
  (`bot-folder`, `project`) the owner must name. Safe form: Windows reference, "npm on this PC".

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

- Whose Claude account the PC claims spend; npm link targets; droplet entries; the push; whether
  the decant `ee08842` (author Sam, before the identity fix) is re-authored before that push.

## Open findings

- `docs/findings/2026-10-15-spike-claims-for-adr-0020.md` (C25 closed, seven rows open)
- `docs/findings/2026-10-15-spike-claims-for-adr-0022.md` (C36 closed, eight rows open)
