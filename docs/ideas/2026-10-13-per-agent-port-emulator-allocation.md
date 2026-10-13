---
date: 2026-10-13
kind: chore
---

# Brief each parallel agent with its own port and emulator

A downstream project's parallel agents collided on ports and emulators. Briefs should allocate a distinct port
and emulator to each agent, so two agents on one machine never share a device. The allocation
belongs in the brief the `brief` skill writes (`.claude/skills/brief/`).

Recorded 2026-10-11 from a downstream build-phase project; a candidate once proposal 0003 (`docs/design/proposals/0003-wave-workflow.md`) is decided.
