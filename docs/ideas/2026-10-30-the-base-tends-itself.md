---
date: 2026-10-30
kind: feature
related: [2026-10-29-broodforge-works-on-itself-overnight.md]  # that one is this project's unattended work; this one is the base's
---

# The base tends itself

A moonshot the owner judged far off at filing (2026-10-30). Hatchery becomes a Broodforge project:
a session there wakes on a sibling's `carry-back/*` push, runs the `carry-back` skill's Land with
its own review, and opens the update in each sibling as a session. On 2026-10-30, five carry-backs took
about fifteen agent rounds of coordination, each a hand-off between sessions the owner started.

Anchor: `.claude/skills/carry-back/SKILL.md`; the `update-from-hatchery` skill; a push-triggered
session start has no entry point yet (`internal/session/service.go` starts from a venue command).

## Open

- Hatchery needs a root, a venue and a credential of its own.
- The one-review rule holds: the base's reviewer, not the sibling's, reads each landing.
- Landing without the owner touches base-owned files; the owner may want to keep that step.

Question: whether Hatchery becomes a Broodforge project, and whether the base may land a carry-back without the owner.
Answer:
