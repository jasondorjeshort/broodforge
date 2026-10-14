# Handoff

Coordinator state. Subagents follow their brief, not this file.
Updated: 2026-10-14, at the amended design-chunk commit on top of ee08842, by main session

## Now

- `main` is unpushed past the decant commit `ee08842`. Remotes: `origin` is Broodforge,
  `hatchery` is the base. Policy: HOLD at 60% session, 75% weekly, no STOP line.
- Owner is Jason Dorje Short <jdorje@gmail.com>; repo-local git identity set to him. Rhythm this
  phase: commit when a chunk is agreed, push only at the end of a session on the owner's words.
- Design chunk committed and reviewed 2026-10-14: `00-vision` (initial, may change),
  `01-requirements`, `02-domain`, `06-platforms`, proposals and ADRs 0020, 0021, 0022 with
  companions, one idea, five operator entries. The reviewer's sixteen findings are fixed and
  amended in; ADR 0022 carries an amendments section and a new spike claim C38.
- `03-architecture.md` and `04-ux.md` are written; `05-debug-interface.md` is being written
  [no-check: in progress]. The owner has 04 for review: fifteen numbered owner decisions at its end.

## Next

1. Runner commits 03, 04, 05 as one commit; reviewer reads it; fixes amend.
2. Owner answers the 04 decisions; tester applies them to 04 and the config keys 04 needs
   (session category, panel message id, transcript channel) to 03.
3. Host setup: the owner runs the operator entries; a tester then writes
   `docs/reference/droplet-host-setup.md` from them [no-check: written when the owner reports].
4. After 05 is reviewed: plan the spike for the ADR claims (C25 to C38) and the first code.
5. Pusher pushes `main` on the owner's words, at session end.

## Waiting on user

- The fifteen owner decisions at the end of `04-ux.md` (venue, buttons, mode names, PR vs push
  default, co-author line form, one driver per session).
- When to push.

## Open findings

none
