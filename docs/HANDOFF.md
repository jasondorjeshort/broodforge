# Handoff

Coordinator state. Subagents follow their brief, not this file.
Updated: 2026-10-13, at commit 7c57730 plus the wrap-up commit, by main session (Hatchery coordinator)

## Now

- Usage gate: the deny line is optional and the base ships none; the override file lifts HOLD per
  meter; `usage.overrideEnv` names a session variable (ADR 0006, living key table). Pushed at
  7c57730. The owner's override file lifts HOLD on the weekly meter, machine-wide, until the reset
  on 2026-10-16 00:00 EDT. Root cause and sibling traps: `docs/findings/2026-10-13-usage-gate-deny-line-trap.md`.
- Build-phase sibling (`../dystopo`): its PR #90 adopts the base gate; Hatchery PR #1 from it is
  merged, so its worktree `../hatchery-wt/usage-gate` and the branch `task/T94` here can go.
- Second sibling (`../regenesis`): updated to 7c57730 on `master` by patch plus an ours-merge
  (2ce5ccd, 0ad0d5b), every deny line null. Unpushed and not yet reviewed there; its copies of
  base ADRs 0002 and 0006 are still the 44f09a2 versions.

## Next

1. Architect answers the critic reviews of proposals 0003 and 0005 (`docs/design/proposals/README.md`),
   revises or promotes; this decides the `wave` skill and the verify-agent question.
2. Critic review of ADR 0006 as a whole, with the escape-hatch question now in
   `.claude/skills/proposal/brief-critic.md`.
3. Triage `docs/ideas/` (`triage` skill): three new gate ideas from the finding above, plus the
   files with no block.
4. Regenesis: independent review and push of its update, in its own session.
5. Adoption of the owner's other existing project.

## Waiting on user

- Push of the wrap-up commit (this file, the finding, three ideas, two skill fixes).
- Removal of the merged `task/T94` branch and its worktree here.
- The `Answer:` on the verify-agent idea, or leave it to proposal 0003.

## Open findings

- `docs/findings/2026-10-11-bash-heredoc-apostrophes.md`
