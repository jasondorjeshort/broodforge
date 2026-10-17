# Handoff

Coordinator state. Subagents follow their brief, not this file.
Updated: 2026-10-17 by main session (Hatchery coordinator)

## Now

- ADR 0016 (base-owned and project-owned files; one patch-only update; the agreement imported per
  section) is accepted and built, reviewed three times. Not built: migration section 3.4, the subtree
  move of the records (`0016-migration.md`), the owner's yes pending.
- ADR 0017 (queue directories list on demand; `queue-list.js`; a marker line switches the hook) is
  accepted, built and reviewed. ADR 0018 (personal data never leaves the machine: one privacy scan per
  commit in the guard, the pusher and a pre-push hook; repo-local identity from the policy) is
  accepted, under review, nothing built; evidence: the leak finding of 2026-10-17.

## Next

1. ADR 0018: fix what its review finds, then the coder brief (`0018-privacy-scan.md` section 7) and
   the tester brief (section 8); one review; then each sibling carries it back.
2. Migration section 3.4 after the owner's yes: architect reviews, tester executes, one commit, then 3.5.
3. Each sibling's migration in its own session (`0016-migration.md` section 4, then the 0017 and 0018 lines).
4. Architect answers the critic reviews of proposals 0003 and 0005; critic review of ADR 0006 as a whole.
5. Triage `docs/ideas/` (`triage` skill), the ADR 0012 scan first; then the sibling two-way sync
   in its own session (`docs/ideas/2026-10-13-first-sync-with-the-build-phase-sibling.md`).

## Waiting on user

- The two operator entries of 2026-10-17: the rewrite of the 28 root commits carrying a third-party
  git identity (starts here, reaches both siblings) and the public repository going private first.
- The subtree move of Hatchery's records (ADR 0016 item 5, migration 3.4): yes, or the architect
  writes the alternative. Nothing else in 0016 depends on it.
- A push: everything after 48a43e4 is unpushed; the pusher goes with the request quoted.
- ADR 0013 is `proposed`: one word confirms it; `2026-10-16-scheduled-architect-design-pass.md`: yes or no.
- The HOLD deny's pool sentences (ADR 0011, The gate) have not been seen by a human; deleting
  `~/.claude/usage-state.json` once every project on this machine runs the new scripts.
- Deletion of the merged `task/T94` branch on GitHub (the guard denies it here; terminal or UI).

## Open findings

- `docs/findings/2026-10-11-bash-heredoc-apostrophes.md` (list: `queue-list.js findings --status open`)
