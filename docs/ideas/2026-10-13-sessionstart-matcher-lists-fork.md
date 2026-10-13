---
date: 2026-10-13
kind: chore
---

# List fork in any future SessionStart matcher

Any future `SessionStart` matcher list should include `fork`. Claude Code's hooks reference
(https://code.claude.com/docs/en/hooks) lists the SessionStart matchers as `startup`, `resume`,
`clear`, `compact` and `fork`, so a matcher copied from the other lists could drop `fork` by
accident. The repo anchor is `.claude/settings.json`.
