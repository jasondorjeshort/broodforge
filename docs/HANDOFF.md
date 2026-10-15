# Handoff

Coordinator state. Subagents follow their brief, not this file.
Updated: 2026-10-14, at the handoff commit on top of f59c195, by main session

## Now

- `main` is seven commits past the decant `ee08842`, all unpushed, each reviewed once with fixes
  amended. Remotes: `origin` is Broodforge, `hatchery` is the base. HOLD 60% session, 75% weekly.
- Owner is Jason Dorje Short <jdorje@gmail.com>; repo-local git identity set to him. Push only
  at the end of a session on the owner's words; the owner's UX decisions are applied.
- Design complete: `00` to `07` with companions, ADRs 0020 to 0022. The spike harness is built
  under `spike/` (Node 22 here) and its offline checks hold; no real engine turn has run.
- The spike waits on the owner's device login into the spike config directory
  (`docs/operator/2026-10-14-spike-login-on-the-pc.md`). Then the coder runs the PC claims.
- 2026-10-14: a bare `npm install` wiped the owner's global npm folder; restored from the cache
  at the pinned versions except two local links (`bot-folder`, `project`) whose targets the
  owner must name. The safe form is in `docs/reference/claude-code-on-windows.md`, "npm on this PC".

## Next

1. Owner runs the spike login and names the two npm links; runner relinks them if asked.
2. Coder runs the PC claims in `07-spike-plan.md` order (C25, C36, S1, C27, C39, S2, C28, C26),
   one short Haiku turn each, fixtures under `$TEMP/bf-spike`; files one finding per ADR.
3. Owner runs the droplet operator entries; coder runs the droplet claims (`07-spike-droplet.md`);
   tester writes `docs/reference/droplet-host-setup.md` [no-check: after the owner reports].
4. Spike exit criterion met: architect records refutations as ADR amendments; coder starts
   `src/` with the engine adapter and the fake engine (05), then the hook, storage, Discord last.
5. Pusher pushes `main` on the owner's words, at session end.

## Waiting on user

- The spike login on the PC; the two npm link targets; the droplet operator entries; when to push.

## Open findings

none
