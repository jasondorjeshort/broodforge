# Handoff

Coordinator state. Subagents follow their brief, not this file.
Updated: 2026-10-18 by main session (Hatchery coordinator)

## Now

- Four commits ahead of origin, reviewed, unpushed, waiting on the owner's word to push: ADR 0019
  (carry-back channel), its build (`carry-back` skill, triage step, group CB), the best-practices
  review, and the first landing (PR 1 of this repository, `ae161c7`). The push closes PR 1 with
  the scanned comment at `<scratchpad>/pr1-close-comment.md`; the branch delete is the owner's.
  The session's account was not read from the status line; the gate named none.

## Next

1. Pusher: push `main`, then the PR 1 close (carry-back skill, the PR 1 paragraph).
2. Triage `docs/ideas/` (`triage` skill): the three 2026-10-18 ideas first; the sibling-sync idea
   of 2026-10-13 is stale (that sibling now holds the base's hooks); the ADR 0012 scan.
3. In `../dystopo`: its handoff and departure row for T117 record the landing `ae161c7`; its next
   `update-from-hatchery` takes the base's version (ADR 0019 item 7).
4. Each sibling in its own session, regenesis and droplet-management too: the rewrite finding's Next;
   ADR 0018 back (`0018-privacy-scan.md` 9); its 0016 migration (`0016-migration.md` 4, then 0017, 0018).
5. Migration 3.4 after the owner's yes (architect reviews, tester executes), then 3.5; then the
   architect answers the critic reviews of proposals 0003 and 0005; critic review of ADR 0006 as a whole.

## Waiting on user

- Whether to switch commits to the GitHub noreply address (ADR 0018 recommends it); broodforge's
  files carry the real one, which its scan will refuse once it adopts 0018.
- The subtree move of Hatchery's records (ADR 0016 item 5, migration 3.4): yes, or the architect
  writes the alternative. Until then a carry-back may edit a record only beside a base change (ADR 0019).
- After the rewrite: dystopo's Actions are on but its four signing secrets are not set (`gh secret set`
  before its next `v*` tag); delete the `<repo>-backup` repos and the kit's backups when satisfied.
- ADR 0013 is `proposed`: one word confirms it; `2026-10-16-scheduled-architect-design-pass.md`: yes or no;
  the HOLD deny's pool sentences (ADR 0011) unseen; delete `~/.claude/usage-state.json` once all projects run the new scripts.

## Open findings

- `docs/findings/2026-10-11-bash-heredoc-apostrophes.md` (list: `queue-list.js findings --status open`)
