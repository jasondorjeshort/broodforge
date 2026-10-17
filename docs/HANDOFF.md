# Handoff

Coordinator state. Subagents follow their brief, not this file.
Updated: 2026-10-17 by main session (Hatchery coordinator)

## Now

- ADR 0016 (base-owned and project-owned files, one patch-only update, the agreement imported per
  section) is accepted and built in Hatchery: hooks, skills, templates, the rule file, the README,
  and `CLAUDE.md` split into `.claude/agreement/`. Reviewed three times; fixes landed. Not built:
  migration section 3.4, the subtree move of the records (`0016-migration.md`), owner's yes pending.
- ADR 0017 (queue directories list on demand: no per-file index in ideas, operator, findings;
  `queue-list.js`; a marker line switches the hook) is accepted and built; its review is in progress.

## Next

1. Fix what the ADR 0017 review finds (tester); commit via the runner.
2. Migration section 3.4 after the owner's yes: the architect reviews, the tester executes, one
   commit; then the citation pass (3.5).
3. Each sibling's migration in its own session: `0016-migration.md` section 4 (one update per
   sibling, in the first-update order), then ADR 0017's per-sibling line.
4. Architect answers the critic reviews of proposals 0003 and 0005 (`docs/design/proposals/README.md`);
   critic review of ADR 0006 as a whole (escape-hatch question: `.claude/skills/proposal/brief-critic.md`).
5. Triage `docs/ideas/` (`triage` skill), the ADR 0012 scan first; then the sibling two-way sync
   in its own session (`docs/ideas/2026-10-13-first-sync-with-the-build-phase-sibling.md`).

## Waiting on user

- The subtree move of Hatchery's records (ADR 0016 item 5, migration 3.4): yes, or the architect
  writes the alternative. Nothing else in 0016 depends on it.
- A push: everything after 48a43e4 is unpushed; the pusher goes with the request quoted.
- ADR 0013 is `proposed` (superseded in part by 0016 and 0017): one word confirms the rest.
- `docs/ideas/2026-10-16-scheduled-architect-design-pass.md`: a scheduled Fable design pass, yes or no.
- The HOLD deny's pool sentences (ADR 0011, The gate) have not been seen by a human; deleting
  `~/.claude/usage-state.json` once every project on this machine runs the new scripts.
- Deletion of the merged `task/T94` branch on GitHub (the guard denies it here; terminal or UI).

## Open findings

- `docs/findings/2026-10-11-bash-heredoc-apostrophes.md` (list: `queue-list.js findings --status open`)
