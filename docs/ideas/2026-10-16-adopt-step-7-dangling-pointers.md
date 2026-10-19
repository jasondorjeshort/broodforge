---
date: 2026-10-16
kind: chore
related: [2026-10-16-base-only-sentences-survive-decant-and-adopt.md]
---

# Adopt step 7 leaves dangling pointers in an adopted project

The checkout list in `.claude/skills/adopt-hatchery/SKILL.md` step 7 brings in docs whose
pointers target files the list does not bring:

- `docs/design/decisions/0002-repository-layout.md` says "Superseded in part by 0014 (`0014-handoff-layout.md`)", a file the list does not bring.
- `docs/design/decisions/0002-hook-specs-usage.md` line 58 points to
  `docs/design/decisions/0006-hook-policy-file.md`.
- `docs/design/proposals/README.md` has rows naming proposals, and ADRs 0012 and 0014, that are
  not checked out; step 7 resets the `ideas/`, `findings/` and `reference/` tables but not this one.

## Open

Question for ADR 0004's owner: add those files to the list, or mark the pointers as Hatchery-tree
references (read with `git show hatchery/main:<path>`, as `0002-repository-layout.md` already says
of its proposals)?

Plan: `tester`. Two of the three pointers are gone: step 7 no longer copies ADRs, and the records live under `docs/hatchery/` (base-owned, moved by every update). Left: step 7 copies `docs/design/proposals/README.md` with its table rows naming proposals the project lacks. In `.claude/skills/adopt-hatchery/SKILL.md` step 7, reset that table to its empty row as the step already does for `reference/`; check whether `decant` keeps the rows and do the same there. Verify: run the step's checkout on a scratch copy and confirm no row names a missing proposal; `node .claude/hooks/test-hooks.js`. Out of scope: ADR 0004.
