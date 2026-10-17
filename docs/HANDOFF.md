# Handoff

Coordinator state. Subagents follow their brief, not this file.
Updated: 2026-10-17 by main session (Hatchery coordinator)

## Now

- ADR 0016 (`docs/design/decisions/0016-base-owned-and-project-owned-files.md`, accepted, 35a2a53):
  base-owned files versus project-owned, one patch-only update, the agreement imported per section.
  Under independent review by the `reviewer`; nothing built yet. The owner has not yet seen the
  `docs/hatchery/` subtree move; it is the migration's last step (`0016-migration.md`). [no-check: the subtree does not exist yet]
- Proposal 0017 (`docs/design/proposals/0017-queue-directories-list-on-demand.md`) is `under review`
  with the critic: queue READMEs drop their per-file index; a listing script reads frontmatter.
- Pushed through 48a43e4; everything from 5b17d07 on is not.

## Next

1. Fix what the ADR 0016 review finds (architect for design, tester for wording), then build from
   `0016-migration.md` section 1: `ownership.json`, `ownership.js`, `imports.js`, the policy and
   post-edit changes (coder); the skill rewrites and the agreement files (tester); the subtree last.
2. Architect answers the 0017 critic; `adr` skill for ADR 0017; then its hook, script and skill edits.
   Absorb `docs/ideas/2026-10-16-update-conflict-rule-wipes-mixed-files.md` when 0016 lands.
3. Each sibling's migration in its own session (`0016-migration.md`, per-sibling table).
4. Architect answers the critic reviews of proposals 0003 and 0005; critic review of ADR 0006 as a whole (`docs/design/proposals/README.md`).
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
- `docs/findings/2026-10-17-queue-directory-indexes.md`
