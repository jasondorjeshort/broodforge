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
- Proposal 0028 (landing from a session, the review gate) `under review`; the critic (Fable) was running
  at this write. Branches here, the owner's to delete: `broodforge/dev/s0005`, `broodforge/dev-live/s0002`
  (landed as `87e90fa`), `s0004`, `s0005` (nothing of theirs: the rebase artefact in the finding).

## Next

1. Proposal 0028: read `0028-critic-review.md`, brief the architect (step 5), the `adr` skill, then the
   coder passes the ADR names.
2. Live-run bugs and texts, the owner's words first: `docs/ideas/2026-10-31-the-first-live-run-texts-name-missing-or-wrong-commands.md`
   (item 3's wrong refusal is a code bug: tester), `2026-10-31-ending-a-session-with-uncommitted-changes-loses-them.md`.
3. W6: the owner re-runs `spike/setup-host/` (README) and pastes the lines; then P23, P25.
4. Coder passes, value order: the three p10 remainders; `2026-10-30-the-review-bundle-should-run-in-the-pretooluse-decision.md` (0028 takes it as R-a).
5. Hatchery: its `main` is at `efe0bba`; the five carry-back branches sit on this repo's `origin`,
   unlanded there; `update-from-hatchery` landed here as `3af3c62`. Land them; delete the seven ideas.

## Waiting on user

- `Question:` blocks in `docs/ideas/` (today: end-without-handoff/abort, uncommitted-at-end, the
  reviewer-model trailer; 0028's owner decisions once the critic is in); the `auto` spike (spends the plan). The handoff import in sessions:
  no answer, stays as is. Wording: the five live-run texts; 0028's venue lines; dry-run and setup texts;
  four Discord strings; two canonical rows and refuse-or-warn; subagent indent; the `pre-push` line;
  P12a; "Never ends for inactivity."; the switch post; the hook no-answer phrase; R-17; `C:\elsewhere`; W6's console window.

## Open findings

Thirteen (`node .claude/hooks/queue-list.js findings --status open`). Live: `docs/findings/2026-10-26-spike-windows-setup-host-claims.md` (Next 3); the 2026-10-23 and 2026-10-24 session findings (the handoff import gap seen again today); `2026-10-17-net-diff-privacy-scan-let-named-commits-go-public.md` (owner).
