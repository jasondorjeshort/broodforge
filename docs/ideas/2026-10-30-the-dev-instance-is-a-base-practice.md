---
date: 2026-10-30
kind: chore
---

# The dev instance is a base practice

Milestone 1 of this project, an agent-drivable dev instance and a project `run` skill, should be a
Hatchery contract every decanted project ships from day one, with the `debug-interface` skill
naming it. The base already says every feature ships a way for an agent to drive it; it does not
say that the whole app needs a launch procedure an agent can follow, so each project finds this
out late.

Anchor: `.claude/skills/run/SKILL.md`; `docs/design/05-debug-interface-dev-instance.md`; the
`decant` skill (which would ask for the run skill's stub).

## Open

- What the contract requires: start, drive, observe, stop, from a terminal, with no GUI.
- Whether the base ships a skeleton `run` skill or only the contract text.

Plan: carry-back (a request with no diff) after P32 lands, since golden sessions complete the dev-instance contract (dev root, run skill, goldens).
