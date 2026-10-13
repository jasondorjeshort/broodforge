---
date: 2026-10-13
kind: question
---

# The override file lifts HOLD for every project on the machine

`~/.claude/usage-gate-override.json` is read by every project's `usage-gate.js`, so a lift asked for
one project (ADR 0006, The HOLD override) lifts HOLD everywhere until `until`. On 2026-10-13 a lift
written for the base also lifted the two siblings. The ADR accepts this and asks the reason to name
the scope; nothing enforces it. A `projects` list in the file (checkout basenames or paths) would
let the owner lift one project, with the absent key meaning all, as `meters` does today.

## Open

- Whether the owner wants per-project lifts at all, or always lifts the machine.
