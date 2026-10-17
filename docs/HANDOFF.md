# Handoff

Coordinator state. Subagents follow their brief, not this file.
Updated: 2026-10-17 by main session (Hatchery coordinator)

## Now

- Proposal 0016 (`docs/design/proposals/0016-base-owned-and-project-owned-files.md`) is `under review`,
  revised after the critic. Recommendation E: one patch-only update over a base-owned allowlist for
  every project; the agreement as per-section base-owned files imported by a short project
  `CLAUDE.md`; the base keeps its policy file. Evidence: the finding of 2026-10-17.
- Pushed through 48a43e4; the finding (5b17d07) and proposal 0016 (b58a582, 7692c28) are not.

## Next

1. The owner's three answers on 0016 (Waiting on user). Then a short critic pass on recommendation E
   (the critic saw A), the compaction check the revision asks for, and the `adr` skill for ADR 0016
   (amends ADRs 0004, 0006, 0008, 0009, 0013; absorbs `docs/ideas/2026-10-16-update-conflict-rule-wipes-mixed-files.md`).
2. Implementation briefs from ADR 0016: the ownership map and its helper, the update and decant and
   adopt skills, check 7, the agreement split; then each sibling's migration in its own session.
3. Architect answers the critic reviews of proposals 0003 and 0005 (`docs/design/proposals/README.md`).
4. Critic review of ADR 0006 as a whole (escape-hatch question: `.claude/skills/proposal/brief-critic.md`).
5. Triage `docs/ideas/` (`triage` skill, ADR 0012 scan first); research `2026-10-15-review-fixes-amend-or-follow-up.md`.
6. Sibling two-way sync in its own session (`docs/ideas/2026-10-13-first-sync-with-the-build-phase-sibling.md`).

## Waiting on user

- ADR 0013 is `proposed`: its surfaces list (now with the handoff-paths row); one word confirms it.
- `docs/ideas/2026-10-16-scheduled-architect-design-pass.md`: a scheduled Fable design pass, yes or no.
- The HOLD deny's pool sentences (ADR 0011, The gate) have not been seen by a human.
- Deleting `~/.claude/usage-state.json` once every project on this machine runs the new scripts.
- Deletion of the merged `task/T94` branch on GitHub (the guard denies it here; terminal or UI).

## Open findings

- `docs/findings/2026-10-11-bash-heredoc-apostrophes.md`
- `docs/findings/2026-10-17-base-repo-defaults-and-upstream-merges.md`
