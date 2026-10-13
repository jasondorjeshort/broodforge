---
date: 2026-10-13
kind: feature
---

# Add a SessionStart hook that reports resume context

A SessionStart hook would report the git log, the open-finding count and a usage line when a
session starts, in `.claude/hooks/` beside `usage-gate.js`. Why not now: the `@docs/HANDOFF.md`
import in CLAUDE.md covers continuity with no code. Constraint: never write a baseline on
`compact`, `clear` or `fork`.

Waits: a session resumes without knowing where it is, or the handoff needs data a file cannot hold.
