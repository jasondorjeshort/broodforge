# Handoff

Coordinator state. Subagents follow their brief, not this file.
Updated: 2026-10-15, after C25 and C36 on the PC, by main session

## Now

- `main` is seven commits past the decant `ee08842`, all unpushed. Two new findings and two edited
  index/operator files are uncommitted. Remotes: `origin` is Broodforge, `hatchery` is the base.
  HOLD 60% session, 75% weekly; at 2026-10-15 09:06 the meters read 11% and 68%.
- Owner is Jason Dorje Short <jdorje@gmail.com>; repo-local git identity set to him. Push only at
  the end of a session on the owner's words; the owner's UX decisions are applied.
- Design complete: `00` to `07` with companions, ADRs 0020 to 0022.
- The spike is live on the PC. The owner ran the C25 login; the fixtures exist (`tree` on `main`,
  `canon` on `base`, `cfg/settings.json`, `spike-hook.json` in mode `allow`). C25 and C36 are both
  confirmed, recorded in `docs/findings/2026-10-15-spike-claims-for-adr-0020.md` and
  `...-for-adr-0022.md`. No real engine turn has run yet.
- The spike config directory holds the Claude Max account belonging to the PC owner, whose PC this is,
  not the owner's. Every PC claim spends that account. The owner may move the spike to his own
  account; that means deleting `$CFG` and running C25 again.
- The owner will drive the claim runs himself with a Fable agent. The routing table's own choice for
  claim runs is `coder` (Opus 5.5); this is the owner's call and costs his money, so it stands.
- 2026-10-14: a bare `npm install` wiped the owner's global npm folder; restored from the cache at
  the pinned versions except two local links (`bot-folder`, `project`) whose targets the owner must
  name. The safe form is in `docs/reference/claude-code-on-windows.md`, "npm on this PC".

## Next

1. Owner decides whose subscription the PC claims spend. Nothing that spends an engine turn runs
   until then; the PC is otherwise ready for S1.
2. The remaining PC claims in `07-spike-plan.md` order: S1, C27, C39, S2, C28, C26. Fixtures are in
   place, so each one is `$RUN` against `$CFG` and `$TREE` with the hook mode set per claim.
3. Owner runs the droplet operator entries and names the two npm link targets.
4. Droplet claims (`07-spike-droplet.md`); tester writes `docs/reference/droplet-host-setup.md`
   [no-check: after the owner reports].
5. Spike exit criterion met: architect records refutations as ADR amendments, and weighs C36's
   open question about the `.claude.json` backup exclusion. Then coder starts `src/` with the engine
   adapter and the fake engine (05), then the hook, storage, Discord last.
6. Pusher pushes `main` on the owner's words, at session end.

## Waiting on user

- Whose Claude account the PC claims spend; the two npm link targets; the droplet operator entries;
  when to push.

## Open findings

- `docs/findings/2026-10-15-spike-claims-for-adr-0020.md` (C25 closed, seven rows open)
- `docs/findings/2026-10-15-spike-claims-for-adr-0022.md` (C36 closed, eight rows open)
