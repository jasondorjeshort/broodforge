# 0016. Base-owned and project-owned files: an update touches nothing the project wrote

Date: 2026-10-17
Status: accepted

Record: proposal `../proposals/0016-base-owned-and-project-owned-files.md` (recommendation E with
the agreement half of A; the three owner questions decided 2026-10-17) and
`../proposals/0016-critic-review.md` (two passes, 28 objections, all answered). Evidence:
`../../findings/2026-10-17-base-repo-defaults-and-upstream-merges.md`. A coder implements from the
siblings: [the ownership map](0016-ownership-map.md) (schema, base-owned set, helper, group `O`),
[the update procedure](0016-update-procedure.md) (the command, conflicts, whole failures, the
report, the policy-key order), [the migration](0016-migration.md) (Hatchery, then each sibling).

## Context

Every `update-from-hatchery` conflicted on `.claude/hooks/policy.json` and usually on `CLAUDE.md`:
both mixed base text with project values, and in a decanted project the merge also brought the
base's own state (handoff, queues, findings, proposals, index rows). The finding's survey: wherever
fewer conflicts are reported, upstream and downstream stopped writing the same file. The owner's
reading, accepted: the cause is ownership, not location. Ruled out in the proposal, not reopened:
`rerere` alone, a `default/` directory, `merge=ours` (B), a layered policy (C), the status quo (D),
A's merge-mode repair, deleting the base's `policy.json`, a disjoint number range, citing records
by `<sha>:<path>`.

Verified 2026-10-17 from the official docs: the project-root `CLAUDE.md` and its imports are
re-read and re-injected after compaction
(https://code.claude.com/docs/en/memory#instructions-seem-lost-after-compact); imports nest four
hops, resolve from the importing file, code spans skipped
(https://code.claude.com/docs/en/memory#import-additional-files); an `InstructionsLoaded` hook
fires per loaded file with `load_reason` values including `include` and `compact`, observation
only (https://code.claude.com/docs/en/hooks#instructionsloaded). Not documented: a subagent's
re-read after its own compaction (Deferred). Verified by the critic (its tables): every git, guard,
`policy.js` and MSYS fact in the proposal's Constraints, and the sibling counts.

What each earlier record still covers. ADR 0004: adoption as a recorded copy, the adoption ADR and
its base line, the licence choice, preconditions; superseded: its practice-set table, the fold,
the two-mode step 1, the owner's 2026-10-12 merging sentence. ADR 0008: nothing stands. ADR 0009:
the line wins over `merge-base` and the join is a project's departure; the tie-break is moot and
the `git show hatchery/main:` read is replaced. ADR 0010: the line decides and the join stays a
departure; patch is the only mode, so the line sets `FROM` and nothing else. ADR 0013: every row
stands; rows are added by item 7 since 0013 does not call its table living. ADR 0006: the living
key table is edited in place; its 150-line constant sentence is superseded. ADR 0002: the layout
gains `docs/hatchery/`, `CLAUDE.md` changes shape. ADR 0007: the queue READMEs and
`docs/design/README.md` are project-owned. The rest of each stands.

## Decision

1. Ownership is data. `.claude/hooks/ownership.json`, base-owned, holds four lists of literal
   paths: `baseOwned` (`.claude`, `LICENSE`, `LICENSES`, `docs/hatchery`), `baseExcluded` (base
   paths that are the project's: `.claude/hooks/policy.json`, `.claude/settings.local.json`),
   `mixed` (`.claude/settings.json`, `.claude/hooks/fitness.json`) and `projectOwned`, naming files
   and directories one by one, never `docs/**`. An unlisted path is project-owned; one set for
   every tree, no mode. The helper `node .claude/hooks/ownership.js` (`--command`, `--per-path`,
   `--report`, each `--ref <sha>` with `--exclude`; `--check`) reads the map at `<sha>`, never defaults.
2. One update for every project: `git diff -M <FROM> <sha> -- <base-owned set, excludes> | git
   apply --3way`, printed by the helper, run verbatim by whichever agent runs the skill. `FROM` is
   the base line; a project without one uses `git merge-base HEAD <sha>` once and writes the line
   in its provenance ADR in that update. Merging ends for decanted projects. Conflicts: base wins
   (`git checkout <sha> -- <path>`) unless the departure table has a row (`git checkout HEAD --
   <path>`); a mixed file is hand-merged. A whole failure goes per path, and the base line is
   bumped last, only when every failed path has a disposition.
3. The agreement is base-owned. Each rule section of today's `CLAUDE.md` (every `##` but `Phase`)
   moves verbatim to `.claude/agreement/<section>.md`: `this-agreement.md`, `where-things-live.md`,
   `delegation.md`, `debug-interface.md`, `major-decisions.md`, `usage-gating.md`,
   `model-routing.md`, `one-review.md`, `git.md`, `licensing.md`; each starts with its heading and
   carries no project value (thresholds are the policy file, the branch the `Branch:` line, the
   licence `LICENSE`, the decisions directory the tree's). The project's `CLAUDE.md`: title,
   provenance line, `@docs/HANDOFF.md`, `## Phase`, `Branch: <name>`, one
   `@.claude/agreement/<section>.md` line per file, `## Project rules`; at most 40 lines, loading
   at most 190 at any hop, the handoff chain excepted. A departure replaces a section's line with
   `@.claude/project/<section>.md`, a project-owned file naming the replaced file in backticks and
   the project's ADR; every replacement is a departure row. Overflow goes to `docs/reference/`,
   `.claude/rules/`, a skill or `## Project rules`, never to a free `@` import.
4. Checks. `.claude/hooks/imports.js` is the one parser of `@` imports (line start or after
   whitespace, outside code spans and fences, four hops, each relative to its file). `post-edit.js`
   at the edit and `node .claude/hooks/policy.js --claude-check`, a new pusher check 7 line, print
   the same `FAIL` lines: a section neither imported nor replaced, imported twice, a replacement
   with no section, the 40, the 190, an unresolved import. Only `edit.capExempt` naming `CLAUDE.md`
   exactly lifts the 40 and the 190, together; the caps live in `policy.js` beside `docCap`. The
   agreement's own total is at most 150 (harness `A1`, Hatchery only). `.claude/rules/docs.md`
   loads for `.claude/agreement/**` and `.claude/project/**`.
5. Hatchery's records ship as a base-owned subtree at the same path in every tree, Hatchery
   included: `docs/hatchery/README.md`, `docs/hatchery/decisions/NNNN-*.md` with their index (the
   "Decisions so far" list), `docs/hatchery/reference/claude-code-on-windows.md` with its README;
   nothing else of Hatchery's `docs/` ships. Hatchery numbers there from `0016` up, no ceiling; a
   project numbers its own `docs/design/decisions/` from `0001` or the next free number, proposals
   included. The `adr` skill's one test: no `hatchery` remote writes in `docs/hatchery/decisions/`,
   every other tree in `docs/design/decisions/`; `triage` and `handoff` scan the same directory. A
   project never writes in `docs/hatchery/`: it supersedes a record by citation from its own ADR
   (`supersedes docs/hatchery/decisions/0001-licensing.md for this project`, which then holds ADR
   0001's living table). In base-owned files a bare `ADR NNNN` is Hatchery's record; across the
   boundary a citation carries the path. `edit.anchorExempt` gains `docs/hatchery/*`.
6. The departure table is a living table (path, reason, carry-back) in the project's provenance
   ADR (`NNNN-adopt-hatchery.md` or `NNNN-decanted-from-hatchery.md`, one template). While a row
   exists the project wins on that path; once Hatchery has the change, base wins and the row goes.
   A path the project never wants the base's version of goes in the policy key `update.exclude`
   (patterns, default `[]`); a departure from a hook or skill is a row, never an exclude.
7. Promised surfaces, rows ADR 0013's table takes by this record: the ownership map (the four
   lists, their meaning, the base-owned set; carried by `ownership.json` and the helper's flags;
   readers: the three skills, the harness; moving a path a project may have written into the
   base-owned set is a major choice, a newly shipped base path or a move toward project-owned is
   free); the agreement (the file names under `.claude/agreement/`, their import lines, the
   replacement path `.claude/project/<section>.md`; readers: the loader, `imports.js`,
   `--claude-check`, `post-edit.js`; renaming, splitting or removing a section is incompatible,
   its migration naming the replacement files it orphans); the `Branch:` line (`^Branch: <name>$`
   in `CLAUDE.md`; readers: the agreement's git section, `decant`, `adopt-hatchery`).
   `update.exclude` is an addition under the policy row; nothing existing changes.
8. Debug interface: `docs/hatchery/reference/agreement-loading.md` (verified, observed): the
   sentinel procedure (`claude -p` with one sentinel word per agreement file, main session and a
   subagent, then `--resume`) and the compaction check (interactive `/compact` with a one-time
   `InstructionsLoaded` hook logging `include` and `compact`). The same procedure checks that
   `imports.js` matches the loader; a mismatch is a parser fix.
9. Out of scope: the queue and proposals directories' per-file indexes (proposal 0017); here they are project-owned.

Deferred, each with its trigger: a subagent losing the agreement after its own compaction (a
sentinel lost under a forced compaction: a finding, then a new ADR); a per-sentence override as a
policy key (a project needs one sentence changed, not the section; the replacement file is the
stopgap); per-path application as the default (`git apply --3way` fails whole on most updates); the
per-file 40 dropped for the 190 alone (overflow routed through an `@` import that replaces no
section); the records not shipping, cited by `<sha>:<path>` (projects edit `docs/hatchery/`
routinely); a per-user pool file (the pool key added by hand at every decant and adoption); merge
mode as option A corrected (the owner wants Hatchery's commits in a decanted project's `blame`, or
a cherry-pick carries a base-file fix back through the DAG).

## Consequences

- Nothing the project wrote is touched by an update: the command cannot carry a path outside the
  list, the base's own state never enters a project; one code path, allowed for every caller.
- Costs: shared history for base files ends at the switch (`blame` names the update commit; the
  base's commit is one hop away in `git log hatchery/main -- <path>`); `git apply --3way` fails
  whole, mitigated per path; the base line is bumped by hand; each sibling pays one migration and
  Hatchery one citation pass. To maintain: the map and helper, groups `O` and `A`, `imports.js`,
  `--claude-check`, the provenance template, three subtree indexes.
- The owner has not yet seen the `docs/hatchery/` subtree move. Implementation starts with it
  last: the map, helper, update, agreement files and checks come first; the subtree (item 5, the
  numbering rule, the citation pass) is the final step, so a veto before then costs nothing built
  and reopens objection 15 in a new ADR; items 1 to 4 and 6 to 8 do not depend on it.
- ADR 0004's owner sentence of 2026-10-12 is superseded, reported by the coordinator; ADR 0008 is
  superseded before its own trigger (a next number of 0020) fired. Supersede this record when the
  loader drops or requires approval for an in-tree `@` import (then `.claude/CLAUDE.md` with the
  hook and check 7 pointed at it), or when a Deferred trigger fires.

Predictions (the trigger scan reads these):
- Prediction, check by 2027-04-17: each sibling's next three updates conflicted only on paths in
  its departure table or in `mixed`.
- Prediction, check by 2027-04-17: after each of those updates `git diff --name-only <FROM> <sha>
  -- <the base-owned set>` equalled the paths taken plus the paths the report dispositioned.
- Prediction, check by 2027-04-17: no project has written in `docs/hatchery/`; every supersession
  of a Hatchery record is a citation from a project ADR.
