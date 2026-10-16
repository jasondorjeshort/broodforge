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
  rules, and hooks live; skills, permissions, and continuity by `@docs/HANDOFF.md` import. Companion
  [hook specifications](decisions/0002-hook-specs.md) for the hooks, and
  [section 3, the usage gate and status line](decisions/0002-hook-specs-usage.md).
- [0004 Adopting the Hatchery practice](decisions/0004-adopt-hatchery.md): an existing project takes
  the practice as files from `hatchery/main`, records the base commit in its own ADR, and updates by
  three-way patch; merging stays the path for decanted projects. (0003 and 0005 are proposals in
  Hatchery not yet promoted; the numbers stay theirs.)
- [0006 Hook policy file](decisions/0006-hook-policy-file.md): each project's choices for the hooks
  (who commits and pushes, deletable branches, usage lines, cap exemptions) live in
  `.claude/hooks/policy.json`, not in the scripts; the guard fails closed, the usage gate open.
- [0007 Queues, triage and one review](decisions/0007-queues-triage-and-one-review.md): `docs/ideas/`
  and `docs/operator/` as delete-when-done queues with triage blocks; a `reviewer` agent and one
  independent review per change; amends ADR 0002 where it names `backlog.md`.
- [0008 ADR numbers, and what a decant keeps](decisions/0008-adr-numbers-and-what-decant-keeps.md):
  Hatchery uses `0001` to `0019` and a decanted project starts at `0020`; a decant keeps every
  Hatchery ADR and the Windows reference, deletes proposals, findings, ideas and operator entries;
  resolves the deferral in ADR 0004 and amends ADR 0002 where it says only 0001 and 0002 are kept.
- [0009 ADR numbers when both tests pass](decisions/0009-adr-numbers-when-both-tests-pass.md): a
  tree that is both joined to the base and holds a `Hatchery base: <sha>` line numbers as adopted; the
  line wins. Amends 0008, Numbers.
- [0010 Update mode when both tests pass](decisions/0010-update-mode-when-both-tests-pass.md): the
  anchored `Hatchery base: <sha>` line means patch mode from that sha whatever `merge-base` says;
  a join merge is a project's recorded departure, not base procedure. Amends 0009 and 0004.
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
  harness, group `F`. Proposed until the owner confirms the list.

The rest of `docs/` (handoff, ideas, reference, findings) is mapped in [`../README.md`](../README.md).
