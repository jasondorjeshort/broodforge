# Design

Design-first: nothing gets built until it is written down here and agreed.

Planned documents (added as we go):

| File | Purpose |
|------|---------|
| `00-vision.md` | What the app is, who it is for, what "done" looks like |
| `01-requirements.md` | Functional and non-functional requirements |
| `02-domain.md` | Core concepts, vocabulary, data model |
| `03-architecture.md` | Components, boundaries, tech choices and why, the promised surfaces with their compatibility promise, and the fitness checks |
| `04-ux.md` | Flows, screens, interaction model |
| `05-debug-interface.md` | How agents drive, observe, and reproduce every feature from a terminal |
| `06-platforms.md` | Target platforms and the language, toolkit, and API chosen for each part, with the sharing strategy between them |
| `proposals/` | Draft proposals for major choices; each gets a critic pass before becoming an ADR |
| `decisions/` | Architecture decision records (ADRs), one per decision |

Major choices (language, toolkit, API, storage, protocol, build system, core dependencies) follow
the proposal, critic, ADR process in `CLAUDE.md`. Technical choices are settled by agents; what
humans see and touch is settled with the user.

Decisions so far:

- [0001 Licensing](decisions/0001-licensing.md): GPL-3.0-or-later by default, LGPL for linkable components.
- [0002 Repository layout](decisions/0002-repository-layout.md): where knowledge, state, procedures,
  rules, and hooks live; skills, permissions, and continuity by `@docs/HANDOFF.md` import, superseded
  in part by 0014 (handoff) and 0016 (the Hatchery records subtree; `CLAUDE.md` as a 40-line
  importing file). Companion [hook specifications](decisions/0002-hook-specs.md) for the
  hooks, and [section 3, the usage gate and status line](decisions/0002-hook-specs-usage.md).
- [0004 Adopting the Hatchery practice](decisions/0004-adopt-hatchery.md): an existing project takes
  the practice as files from `hatchery/main`, records the base commit in its own ADR, and updates by
  three-way patch; merging stays the path for decanted projects. (0003 and 0005 are proposals in
  Hatchery not yet promoted; the numbers stay theirs.) Superseded in part by 0016 (the merging
  sentence, the practice-set table, the fold, the two modes).
- [0006 Hook policy file](decisions/0006-hook-policy-file.md): each project's choices for the hooks
  (who commits and pushes, deletable branches, usage lines, cap exemptions) live in
  `.claude/hooks/policy.json`, not in the scripts; the guard fails closed, the usage gate open.
  Superseded in part by 0016 (the `CLAUDE.md` cap constant; `update.exclude` in the living table).
- [0007 Queues, triage and one review](decisions/0007-queues-triage-and-one-review.md): `docs/ideas/`
  and `docs/operator/` as delete-when-done queues with triage blocks; a `reviewer` agent and one
  independent review per change; amends ADR 0002 where it names `backlog.md`. Superseded in part
  by 0016 (the queue READMEs and `docs/design/README.md` are project-owned).
- [0008 ADR numbers, and what a decant keeps](decisions/0008-adr-numbers-and-what-decant-keeps.md):
  Hatchery uses `0001` to `0019` and a decanted project starts at `0020`; a decant keeps every
  Hatchery ADR and the Windows reference, deletes proposals, findings, ideas and operator entries;
  resolves the deferral in ADR 0004 and amends ADR 0002 where it says only 0001 and 0002 are kept.
  Superseded by 0016.
- [0009 ADR numbers when both tests pass](decisions/0009-adr-numbers-when-both-tests-pass.md): a
  tree that is both joined to the base and holds a `Hatchery base: <sha>` line numbers as adopted; the
  line wins. Amends 0008, Numbers. Superseded in part by 0016 (the tie-break is moot).
- [0010 Update mode when both tests pass](decisions/0010-update-mode-when-both-tests-pass.md): the
  anchored `Hatchery base: <sha>` line means patch mode from that sha whatever `merge-base` says;
  a join merge is a project's recorded departure, not base procedure. Amends 0009 and 0004.
  Superseded in part by 0016 (one mode; the line decides `FROM` only).
- [0011 Usage cache per session, pool advice at HOLD](decisions/0011-usage-cache-per-session-and-pool-advice.md):
  the status line caches the meters per `session_id` with the account as a label, the gate treats a
  changed login as stale, and at HOLD it reports the pool account with room through `usage.pool`;
  rotation happens at session boundaries. Amends 0006 and the hook specs, section 3
  ([usage companion](decisions/0002-hook-specs-usage.md)).
- [0012 Decision practices for long-lived software](decisions/0012-decision-practices-for-long-lived-software.md):
  each project names its promised surfaces (additions free, an incompatible change a major
  choice); dependencies by checklist, a copy under a page keeps its licence; fitness checks with a
  skip entry as the lift; ADR triggers, dated predictions and fitness lines scanned by `triage`,
  the architect called on a hit; four lines in `CLAUDE.md`, the rest in the file that applies.
  Amends the living table of ADR 0001.
- [0013 Hatchery's promised surfaces](decisions/0013-hatchery-promised-surfaces.md): the policy
  file, grant files, the idea grammar, the base line, the usage cache and the hook seams, with
  what is not promised; the reference fitness check (hooks require only Node built-ins) in the
  harness, group `F`. Proposed until the owner confirms the list. Superseded in part by 0016 (five
  rows added by its item 7; the idea-grammar row's record).
- [0014 Handoff layout](decisions/0014-handoff-layout.md): one
  `docs/HANDOFF.md` by default, no hash, the edit rides in the work commit; a per-stream split
  (`@handoff/<file>.md` lines, 40 each, 120 together, no README, no fixed files) the tooling
  accepts and a project switches to at a stated trigger; every `@` line resolved by the hook
  and the pusher. Skill text in [the sibling](decisions/0014-handoff-skill.md). Supersedes in
  part 0002; adds a row to 0013. Superseded in part by 0015 (switch trigger).
- [0015 Handoff shape at decant](decisions/0015-handoff-shape-at-decant.md): the shape is chosen
  by writer count, one question to the owner at decant or adoption (yes: the map plus one stream
  from the first commit; no: the single file); a single-file project switches the day a second
  parallel session opens, not at a conflict count; the map stays the index and the stream
  directory has no README; Hatchery stays single-file; the caps do not move. Supersedes in part 0014.
- [0016 Base-owned and project-owned files](decisions/0016-base-owned-and-project-owned-files.md):
  ownership as data (`.claude/hooks/ownership.json`, one set for every tree); one patch-only update
  from the base-owned set for decanted and adopted projects alike, printed by a helper and run
  verbatim, conflicts base-wins unless a departure-table row, whole failures per path; the rule
  sections as base-owned `.claude/agreement/<section>.md` imported by a 40-line project `CLAUDE.md`
  (loads at most 190; `edit.capExempt` naming `CLAUDE.md` lifts both; `imports.js` and
  `--claude-check`); Hatchery's records in the base-owned subtree `docs/hatchery/`, Hatchery
  numbering from 0016 up, projects from 0001; `update.exclude`. Siblings: [the map](decisions/0016-ownership-map.md),
  [the procedure](decisions/0016-update-procedure.md), [the migration](decisions/0016-migration.md).
  Supersedes 0008; supersedes in part 0002, 0004, 0006, 0007, 0009, 0010, 0013.
  [no-check: the map and the subtree exist once the ADR is implemented]

The rest of `docs/` (handoff, ideas, reference, findings) is mapped in [`../README.md`](../README.md).
