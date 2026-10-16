---
date: 2026-10-13
kind: chore
---

# Give the debug-interface design doc a hooks section

When `docs/design/05-debug-interface.md` is written, it should have a hooks section pointing at
`.claude/hooks/test-hooks.js`, which has `--only`, `--verbose` and `--input <file> <hook>` modes.
The doc does not exist yet, so the section waits on the doc being written. [no-check: not yet created]

2026-10-16: the dependency check's debug interface is `--only deps --verbose` (ADR 0013); it
belongs in that section when the doc is written.
