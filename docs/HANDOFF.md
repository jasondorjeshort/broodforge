# Handoff

Coordinator state. Subagents follow their brief, not this file.
Updated: 2026-10-14, at the owner-decisions commit on top of 56eea04, by main session

## Now

- `main` is three commits past the decant `ee08842`, all unpushed. Remotes: `origin` is
  Broodforge, `hatchery` is the base. Policy: HOLD at 60% session, 75% weekly, no STOP line.
- Owner is Jason Dorje Short <jdorje@gmail.com>; repo-local git identity set to him. Rhythm this
  phase: commit when a chunk is agreed, push only at the end of a session on the owner's words.
- The design phase is complete: `00` to `06` with companions, proposals and ADRs 0020 to 0022,
  each commit reviewed once with fixes amended. The owner answered every UX decision on
  2026-10-14; they are applied in `04-ux.md` and the config keys in `03-architecture.md`. UX
  choices that may become options later are one idea in `docs/ideas/`.
- Spike claims C25 to C39 in ADRs 0020 and 0022 are untested; nothing runs until they are.

## Next

1. Reviewer reads the owner-decisions commit (small; 03, 04, 05, one idea).
2. Architect: a spike plan for claims C25 to C39 (which host, which claim, what evidence
   settles it, what each refutation would change), as `docs/design/07-spike-plan.md`
   [no-check: being written].
3. Owner runs the operator entries for the droplet and the PC; a tester then writes
   `docs/reference/droplet-host-setup.md` from them [no-check: written when the owner reports].
4. Coder: the engine adapter and the fake engine first (05 drives them without Discord), then
   the hook, then storage; Discord last.
5. Pusher pushes `main` on the owner's words, at session end.

## Waiting on user

- Running the operator entries (`docs/operator/README.md`).
- When to push.

## Open findings

none
