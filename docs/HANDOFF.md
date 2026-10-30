# Handoff

Coordinator state. Subagents follow their brief, not this file.
Updated: 2026-10-30 by main session (P28 and P32 built and reviewed; channel-200 credential doc fixed)

## Now

- Account `jdorje`. Ahead of `origin/main`, each reviewed: the channel-200 credential doc fix
  (`4d15450`), P28 (`cb1047c`), P32 without `dev-read` (`73986be`), two ideas (`32e6be2`), this handoff.
- Dev instance: `%LOCALAPPDATA%\Broodforge\dev`, dry-run for good, bot stopped, dev credential
  `cred-2ha7apqowv` usable (confirm `stop` was run); channel 200 as the owner, or `--credential <dev>`.
- Hatchery `main` pushed at `61b6409`. Five carry-back branches pushed (ADR 0024 names the shas).

## Next

1. `dev-read`, the real-engine golden (P32's tail in `08-implementation-plan-golden-sessions.md`): one
   real turn, the third of three, on the owner's quoted go-ahead; the independent read; the owner's
   decision before the commit. The dev engine still loads the account's synced skills.
2. W6: the owner re-runs `spike/setup-host/` (README) and pastes the lines; the finding records
   the result; then P23 (coder), owner steps, P25.
3. Coder passes, value order: P29 (`08-implementation-plan-engine-stream.md`); the transcript
   decision-line idea (`Plan:`, tester); P30, P31 (`08-implementation-plan-canonical.md`); the three
   p10 remainders; Git Bash stayed waiting.
4. Hatchery: Land the five branches there; then `update-from-hatchery` here; delete the seven ideas.
5. The `auto` spike (tester, the owner's login, spends the plan): only on the owner's word.

## Waiting on user

- The go-ahead for the one real turn (`dev-read`).
- Three new `Question:` blocks in `docs/ideas/` (the git-guard gap: `Waits:` or the ADR 0024 row then
  delete; a store-failure event type; the decision-line wording), nine earlier ones; the `auto` spike.
- Wording: dry-run and setup texts (companion, `05-debug-interface-setup.md`); four Discord strings
  (`05-debug-interface-discord.md`); two canonical rows in `04-ux.md` and refuse-or-warn; subagent
  text indent; the `pre-push` line (P15); P12a; "Never ends for inactivity."; the switch post; the
  hook no-answer phrase; R-17; `Waits:` blocks; `C:\elsewhere`; W6's console window.

## Open findings

Thirteen (`node .claude/hooks/queue-list.js findings --status open`). Live: `docs/findings/2026-10-26-spike-windows-setup-host-claims.md` (Next 2); the 2026-10-23 and 2026-10-24 session findings; `2026-10-17-net-diff-privacy-scan-let-named-commits-go-public.md` (owner).
