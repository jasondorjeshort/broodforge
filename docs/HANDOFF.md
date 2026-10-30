# Handoff

Coordinator state. Subagents follow their brief, not this file.
Updated: 2026-10-30 by main session (wave built; pushes and the credential step wait on the owner)

## Now

- Account `jdorje`. `origin/main` is `a015245`. Ahead, each reviewed, unpushed: Hatchery update
  (base line `8d6ee28`), dev instance design and P27, importfixture, goLive, triage amendments
  (P28 to P31), lint test, P11 items 1, 2, 6, 8, the departure rows and git-guard idea.
- Dev instance: `%LOCALAPPDATA%\Broodforge\dev`, dry-run for good, bot stopped, no credential yet
  (the owner's step, `docs/operator/2026-10-30-dev-instance-credential.md`).
- `../hatchery` holds `61b6409` (two ideas), unpushed. Five reviewed carry-back branches in
  `../broodforge-wt/` (ADR 0024 names the shas), unpushed, Send step 6 next.

## Next

1. After the owner's credential step: one real turn on the dev root on the owner's quoted word.
2. W6: the owner re-runs `spike/setup-host/` (README) and pastes the lines; the finding records
   the result; then P23 (coder), owner steps, P25.
3. Coder passes, value order: P28 then P29 (`08-implementation-plan-engine-stream.md`, migration
   `0004`); P30, P31 (`08-implementation-plan-canonical.md`); the three p10 remainders; Git Bash
   stayed waiting (repro first); `serve --trace` after P12d.
4. Hatchery: push `61b6409` and the five branches; Land each there (`carry-back` skill); then
   `update-from-hatchery` here and delete the five ideas.
5. The `auto` spike (tester, the owner's login, spends the plan): only on the owner's word.

## Waiting on user

- Pushes: this clone's commits; Hatchery `61b6409`; the five carry-back branches (first pushes of
  branches rooted in Hatchery history). The dev credential step (five minutes, one sign-in).
- Wording: five dry-run setup texts (companion); four Discord strings (`05-debug-interface-discord.md`);
  two canonical rows in `04-ux.md` and refuse-or-warn; subagent text indent. W6: the console window.
- Nine `Question:` blocks in `docs/ideas/`; the `auto` spike go-ahead; setup texts
  (`05-debug-interface-setup.md`); earlier: the `pre-push` line (P15); P12a wording; "Never ends
  for inactivity."; the switch post; the hook no-answer phrase; R-17; `Waits:` blocks; `C:\elsewhere`.

## Open findings

Thirteen (`node .claude/hooks/queue-list.js findings --status open`). Live: `docs/findings/2026-10-26-spike-windows-setup-host-claims.md` (Next 2); the 2026-10-23 and 2026-10-24 session findings; `2026-10-17-net-diff-privacy-scan-let-named-commits-go-public.md` (owner).
