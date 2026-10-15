# Handoff

Coordinator state. Subagents follow their brief, not this file.
Updated: 2026-10-14, at the amended spike-plan commit on top of 8ddc962, by main session

## Now

- `main` is three commits past the decant `ee08842`, all unpushed. Remotes: `origin` is
  Broodforge, `hatchery` is the base. Policy: HOLD at 60% session, 75% weekly, no STOP line.
- Owner is Jason Dorje Short <jdorje@gmail.com>; repo-local git identity set to him. Rhythm this
  phase: commit when a chunk is agreed, push only at the end of a session on the owner's words.
- The design phase is complete: `00` to `06` with companions, proposals and ADRs 0020 to 0022,
  each commit reviewed once with fixes amended. The owner answered every UX decision on
  2026-10-14; they are applied in `04-ux.md` and the config keys in `03-architecture.md`. UX
  choices that may become options later are one idea in `docs/ideas/`.
- Spike claims C25 to C39 in ADRs 0020 and 0022 are untested; `07-spike-plan.md` and its droplet
  companion say how, in the order that hurts most if refuted, reviewed once. Nothing in `src/`
  until the stop claims hold. The session hit HOLD on the 5-hour meter at the end of 2026-10-14.

## Next

1. Pusher pushes `main` on the owner's words, at session end; nothing else is queued before it.
2. Owner completes the device logins the plan names and runs the operator entries; the
   coder runs the PC claims first (C25, C36, S1, C27, C39, S2, C28, C26) in `spike/`
   [no-check: created by the coder], filing one finding per ADR.
3. After the droplet entries: the droplet claims (C30 to C35, C38, C29); a tester writes
   `docs/reference/droplet-host-setup.md` from them [no-check: written when the owner reports].
4. Coder: the engine adapter and the fake engine first (05 drives them without Discord), then
   the hook, then storage; Discord last.

## Waiting on user

- Running the operator entries (`docs/operator/README.md`) and the spike logins; the spike
  spends Haiku one-line turns on the owner's subscription, under the usage gate's HOLD line.
- When to push.

## Open findings

none
