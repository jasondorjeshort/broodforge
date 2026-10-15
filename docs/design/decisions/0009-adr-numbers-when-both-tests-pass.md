# 0009. ADR numbers when both tests pass

Date: 2026-10-15
Status: accepted

Record: no proposal; a downstream project met the case and settled it for itself; this carries the
rule into the base. Amends ADR 0008, Numbers, for the case below; the rest of 0008 stands.

## Context

ADR 0008 and the `adr` skill tell three trees apart: Hatchery itself (no `hatchery` remote), a
decanted project (`git merge-base HEAD hatchery/main` succeeds) and an adopted one (a
`Hatchery base:` line exists under `docs/design/decisions/`). A decanted project starts at `0020`;
an adopted one keeps its own numbering.

A project that adopted Hatchery and later joined its history to the base, by an ours-merge made
after an update (ADR 0004), passes both tests: `merge-base` succeeds, which says "decanted", and
the `Hatchery base:` line exists, which says "adopted". The skill gave no answer. Taking "decanted"
would renumber nothing but would start a tree that already holds ADRs in `0001` to `0019` at
`0020`; taking "adopted" keeps the numbers it has.

## Decision

- When both tests pass, the `Hatchery base:` line wins: the project numbers as an adopted project.
  Its next ADR is the next free number in its own `decisions/`, proposals included, and nothing is
  renumbered. The "starts at `0020`" rule does not apply to it.
- Why the line and not the merge: the line records how the project took the practice (files copied
  once, patches after); the join merge only keeps `merge-base` current for the next diff and says
  nothing about which numbers the project has used.
- The `adr` skill, step 1, states the tie-break. A decanted project, which has no such line, is
  unchanged.
- Base ADR numbers cited inside `.claude/` mean Hatchery's record, whichever numbering the project
  uses; a project that does not carry the file reads it with
  `git show hatchery/main:docs/design/decisions/<file>`. Out of scope: any change to the update
  procedure itself.

## Consequences

- An adopted project may hold numbers in `0001` to `0019` that are its own, so a base ADR number
  cited in `.claude/` can name a different file in the project's tree. Citing by title or path
  along with the number avoids the confusion.
- The `Hatchery base:` line becomes load-bearing for numbering as well as for updates; a project
  that deletes it falls back to the `merge-base` test.
- Supersede this record if Hatchery widens its range (ADR 0008, Consequences) or changes how
  projects take updates.
