# Handoff

Coordinator state. Subagents follow their brief, not this file.
Updated: 2026-10-17 by main session (Hatchery coordinator)

## Now

- ADR 0016 (base-owned and project-owned files; one patch-only update; the agreement imported per
  section) is accepted and built, reviewed three times. Not built: migration section 3.4, the subtree
  move of the records (`0016-migration.md`), the owner's yes pending.
- ADR 0017 (queue directories list on demand; `queue-list.js`; a marker line switches the hook) is
  accepted, built and reviewed. ADR 0018 (personal data never leaves the machine: the privacy scan per commit in the guard, the
  pusher and a pre-push hook; identity from the policy) is accepted and built; its code and text are
  reviewed and fixed; the owner's identity is listed, so pushes are allowed again.

## Next

1. ADR 0018: fix what the code review finds; `git config core.hooksPath .claude/hooks/git` in this
   clone once the policy names it; then each sibling carries 0018 back (`0018-privacy-scan.md` 9).
2. Migration section 3.4 after the owner's yes: architect reviews, tester executes, one commit, then 3.5.
3. Each sibling's migration in its own session (`0016-migration.md` section 4, then the 0017 and 0018 lines).
4. Architect answers the critic reviews of proposals 0003 and 0005; critic review of ADR 0006 as a whole.
5. Triage `docs/ideas/` (`triage` skill), the ADR 0012 scan first; then the sibling two-way sync
   in its own session (`docs/ideas/2026-10-13-first-sync-with-the-build-phase-sibling.md`).

## Waiting on user

- Whether to switch commits to the GitHub noreply address (ADR 0018 recommends it; the real one is listed now).
- The subtree move of Hatchery's records (ADR 0016 item 5, migration 3.4): yes, or the architect
  writes the alternative. Nothing else in 0016 depends on it.
- The history rewrite of the 28 commits under the second account's identity, in hatchery, broodforge and
  dystopo (`docs/operator/2026-10-17-rewrite-root-commit-identity.md`): all repositories are private;
  everything through 344416e is pushed; mirror backups are in the kit folder; the owner runs the kit,
  then a tester re-records the cited hashes from the map. The visibility entry is done; delete it.
- ADR 0013 is `proposed`: one word confirms it; `2026-10-16-scheduled-architect-design-pass.md`: yes or no;
  the HOLD deny's pool sentences (ADR 0011) unseen; delete `~/.claude/usage-state.json` once all projects run the new scripts; delete the merged `task/T94` branch on GitHub.

## Open findings

- `docs/findings/2026-10-11-bash-heredoc-apostrophes.md` (list: `queue-list.js findings --status open`)
