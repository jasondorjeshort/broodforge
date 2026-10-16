# Handoff

Coordinator state. Subagents follow their brief, not this file.
Updated: 2026-10-16 by main session (Hatchery coordinator)

## Now

- Everything is pushed: origin/main matches HEAD (pusher check 7 reads `edit.capExempt`; ADR 0006's
  deferred item closed). This session: ADR 0015, the handoff shape chosen at decant, reviewed, committing.
- `../regenesis`: the check 7 blocker is gone, master is at its origin, the finished handoffs worktree is
  removed; its policy names its legacy docs one by one; its map plus streams total 146 lines.
- Scans of 2026-10-16 filed seven ideas. `../broodforge` (decanted, live session) and
  `../droplet-management` (adopted at 9d90716, never updated) still read `~/.claude/usage-state.json`;
  `../dystopo` (adopted, patch mode) updates cleanly only after the patch-update ideas are fixed.

## Next

1. The owner's answers (Waiting on user); the siblings' own fixes from the scans, each in its own session.
2. `../regenesis`: at its next `update-from-hatchery`, take ADR 0015 with 0011 to 0014, drop
   `../regenesis/docs/handoff/README.md`, and close streams until map plus streams are at most 120 (146 now).
3. The first decant with the README prompt.
4. Architect answers the critic reviews of proposals 0003 and 0005 (`docs/design/proposals/README.md`).
5. Critic review of ADR 0006 as a whole, with the escape-hatch question in `.claude/skills/proposal/brief-critic.md`.
6. Triage `docs/ideas/` (`triage` skill, opening with the ADR 0012 scan): the eight untriaged files
   of 2026-10-16 first; also the post-edit warning that `docs/design/decisions/README.md` lists no ADRs.
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
