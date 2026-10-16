---
date: 2026-10-16
kind: chore
---

# Adopt step 7 leaves dangling pointers in an adopted project

The checkout list in `.claude/skills/adopt-hatchery/SKILL.md` step 7 brings in docs whose
pointers target files the list does not bring:

- `docs/design/decisions/0002-repository-layout.md` line 3 points to `0014-handoff-layout.md`.
- `docs/design/decisions/0002-hook-specs-usage.md` line 58 points to
  `docs/design/decisions/0006-hook-policy-file.md`.
- `docs/design/proposals/README.md` has rows naming proposals, and ADRs 0012 and 0014, that are
  not checked out; step 7 resets the `ideas/`, `findings/` and `reference/` tables but not this one.

(The pointer from `0002-hook-specs.md` to `0002-hook-specs-usage.md` was fixed on 2026-10-16 by
adding the file to the list.)

## Open

Question for ADR 0004's owner: add those files to the list, or mark the pointers as Hatchery-tree
references (read with `git show hatchery/main:<path>`, as `0002-repository-layout.md` already says
of its proposals)?
