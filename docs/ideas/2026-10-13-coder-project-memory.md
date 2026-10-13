---
date: 2026-10-13
kind: chore
---

# Give the coder agent a project memory scope

The `coder` agent definition (`.claude/agents/coder.md`) could set `memory: project` so that
environment facts persist across its runs. Why not now: that is a second, unindexed knowledge
location next to `docs/`, which the docs conventions exist to prevent.

Waits: the same environment fact appears in two findings, or coder briefs keep restating it.
