# Handoff

Coordinator state. Subagents follow their brief, not this file.
Updated: 2026-10-16 by main session (Hatchery coordinator)

## Now

- origin/main is ab42bd2. Unpushed: 3d5361a (proposal, review, ADR 0014, finding) and this commit
  (ADR 0014 applied: skills, docs rule, post-edit hook with group PH, pusher check 7, ADR lines,
  indexes). Reviewed once, minors fixed. Account jdorje; the gate never denied. Earliest due
  prediction 2027-01-16; no fitness FAIL or SKIP in the last run.
- `../regenesis` (its ADR 0017) spells its map `@docs/handoff/<file>.md`, which resolves from
  `docs/`, so no stream has loaded there since the split; not edited from here (ADR 0014, migration).
  `../broodforge` runs the older base and still reads `~/.claude/usage-state.json`.

## Next

1. The push, on the owner's words (`pusher`); then the owner's answers (Waiting on user).
2. `../regenesis` in its own session: map lines to `@handoff/<file>.md` now; ADRs 0011 to 0014 at its update.
3. The first decant with the README prompt.
4. Architect answers the critic reviews of proposals 0003 and 0005 (`docs/design/proposals/README.md`).
5. Critic review of ADR 0006 as a whole, with the escape-hatch question in `.claude/skills/proposal/brief-critic.md`.
6. Triage `docs/ideas/` (`triage` skill, opening with the ADR 0012 scan): the files with no block,
   `2026-10-16-handoff-import-check-residue.md` among them. Also settle the post-edit warning that
   `docs/design/decisions/README.md` lists no ADRs.
7. Research `docs/ideas/2026-10-15-review-fixes-amend-or-follow-up.md`; check `2026-10-15-statusline-command-cwd-relative.md`.
8. Sibling two-way sync in its own session (`docs/ideas/2026-10-13-first-sync-with-the-build-phase-sibling.md`);
   the open heredoc finding's Next steps (reproduce on Linux; check the changelog).

## Waiting on user

- ADR 0013 is `proposed`: its surfaces list (now with the handoff-paths row); one word confirms it.
- `docs/ideas/2026-10-16-scheduled-architect-design-pass.md`: a scheduled Fable design pass, yes or no.
- The HOLD deny's pool sentences (ADR 0011, The gate) have not been seen by a human.
- Deleting `~/.claude/usage-state.json` once every project on this machine runs the new scripts.
- Deletion of the merged `task/T94` branch on GitHub (the guard denies it here; terminal or UI).

## Open findings

- `docs/findings/2026-10-11-bash-heredoc-apostrophes.md`
