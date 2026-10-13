---
date: 2026-10-13
kind: question
---

# Reserve ADR number ranges so Hatchery and its projects stop colliding

Hatchery and the projects decanted from it can each write `docs/design/decisions/0003-*.md`, so
both can hold an ADR 0003. The candidate fix is to reserve `0001` to `0019` for Hatchery and start
projects at `0020`, in the `adr` and `decant` skills (`.claude/skills/adr/`, and
`.claude/skills/decant/SKILL.md`). ADR 0004 defers the decision.

## Open

- Whether a reserved range is worth the rule, or a project prefix in the file name does the same job.
