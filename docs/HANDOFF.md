# Handoff

Coordinator state. Subagents follow their brief, not this file.
Updated: 2026-10-31 by main session (milestone 2 reached; proposal 0028 under review)

## Now

- Account `jdorje`. Pushed at `38ebc03`. Ahead: this commit. A pusher stopped on `behind 2` today (a parallel
  session pushed the policy and Hatchery commits); the rebase rewrote hashes under the live bot (the finding).
- Milestone 2 reached, 2026-10-31:
  `docs/findings/2026-10-31-milestone-2-reached-the-dev-live-bot-answers-in-the-owners-server.md`.
  Live instance `%LOCALAPPDATA%\Broodforge\dev-live` (the owner's application and server, `handoff`
  `push`, remote this checkout, `serve` by hand): the owner drives it from Discord; agents never touch
  it. `dev`: dry-run, bot stopped, schema 4, `cred-2ha7apqowv`, four real turns granted, none spent.
- ADR 0028 recorded (landing a session on the default branch; P33 to P37, `08-implementation-plan-landing.md`).
  Dev-live's remote moves to GitHub (owner: `setup git-credential`, re-point the canonical's `origin`).
  Branches here, the owner's to delete: `broodforge/dev/s0005`, `broodforge/dev-live/{s0002,s0004,s0005,s000b}` (all landed or empty).

## Next

1. P33 (coder): the `reviewer` cell and the pin at the defer; then P34 to P37 in order
   (`08-implementation-plan-landing.md`). P33's real turn only on the owner's word.
2. Live-run bugs and texts, the owner's words first: `docs/ideas/2026-10-31-the-first-live-run-texts-name-missing-or-wrong-commands.md`
   (item 3's wrong refusal is a code bug: tester), `2026-10-31-ending-a-session-with-uncommitted-changes-loses-them.md`.
3. W6: the owner re-runs `spike/setup-host/` (README) and pastes the lines; then P23, P25.
4. Coder passes, value order: the three p10 remainders (its idea); Git Bash stayed waiting.
5. Hatchery: its `main` is at `efe0bba`; the five carry-back branches sit on this repo's `origin`,
   unlanded there; `update-from-hatchery` landed here as `3af3c62`. Land them; delete the seven ideas.

## Waiting on user

- `Question:` blocks in `docs/ideas/` (today: end-without-handoff/abort, uncommitted-at-end, the
  reviewer-model trailer, needed before the first `main` land; ADR 0028's "Waits on the owner"); the `auto` spike (spends the plan). The handoff import in sessions:
  no answer, stays as is. Wording: the five live-run texts; 0028's venue lines; dry-run and setup texts;
  four Discord strings; two canonical rows and refuse-or-warn; subagent indent; the `pre-push` line;
  P12a; "Never ends for inactivity."; the switch post; the hook no-answer phrase; R-17; `C:\elsewhere`; W6's console window.

## Open findings

Thirteen (`node .claude/hooks/queue-list.js findings --status open`). Live: `docs/findings/2026-10-26-spike-windows-setup-host-claims.md` (Next 3); the 2026-10-23 and 2026-10-24 session findings (the handoff import gap seen again today); `2026-10-17-net-diff-privacy-scan-let-named-commits-go-public.md` (owner).
