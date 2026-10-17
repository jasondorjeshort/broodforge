# Handoff

Coordinator state. Subagents follow their brief, not this file.
Updated: 2026-10-17 by main session (Hatchery coordinator)

## Now

- ADRs 0016, 0017 and 0018 are accepted, built and reviewed; not built: 0016 migration 3.4 (owner's yes).
- The history rewrite is done: hatchery, broodforge and dystopo lost the second account's identity,
  session links and names, each audited before its push to a new private repo (old: `<repo>-backup`)
  and switched in place. Record: `docs/findings/2026-10-17-history-rewrite-of-three-repos.md`; kit,
  maps and procedure: `../history-rewrite` (never committed: it holds identities).

## Next

1. First, triage and fix `docs/ideas/2026-10-17-privacy-scan-trusts-stale-tracking-refs.md`.
2. Each sibling in its own session, regenesis and droplet-management too: the rewrite finding's Next;
   ADR 0018 back (`0018-privacy-scan.md` 9); its 0016 migration (`0016-migration.md` 4, then 0017, 0018).
3. Migration section 3.4 after the owner's yes: architect reviews, tester executes, one commit, then 3.5.
4. Architect answers the critic reviews of proposals 0003 and 0005; critic review of ADR 0006 as a whole.
5. Triage `docs/ideas/` (`triage` skill), the ADR 0012 scan first; then the sibling two-way sync
   in its own session (`docs/ideas/2026-10-13-first-sync-with-the-build-phase-sibling.md`).

## Waiting on user

- Whether to switch commits to the GitHub noreply address (ADR 0018 recommends it); broodforge's
  files carry the real one, which its scan will refuse once it adopts 0018.
- The subtree move of Hatchery's records (ADR 0016 item 5, migration 3.4): yes, or the architect
  writes the alternative. Nothing else in 0016 depends on it.
- After the rewrite: delete `../hatchery-fresh`, `../broodforge-fresh`, `../Dystopo-fresh` (clean
  old-history clones, origin the new repo: the stale-refs hazard); dystopo's Actions off until its
  secrets are back; optional reflog expire and gc; delete the `<repo>-backup` repos and kit backups.
- Regenesis and droplet-management were not rewritten (mixed authorship is fine there); both still
  carry session links in messages and the second account's name in files: rewrite them or not.
- ADR 0013 is `proposed`: one word confirms it; `2026-10-16-scheduled-architect-design-pass.md`: yes or no;
  the HOLD deny's pool sentences (ADR 0011) unseen; delete `~/.claude/usage-state.json` once all projects run the new scripts.

## Open findings

- `docs/findings/2026-10-11-bash-heredoc-apostrophes.md` (list: `queue-list.js findings --status open`)
