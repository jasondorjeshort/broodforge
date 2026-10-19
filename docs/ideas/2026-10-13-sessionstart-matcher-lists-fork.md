---
date: 2026-10-13
kind: chore
---

# List fork in any future SessionStart matcher

Any future `SessionStart` matcher list should include `fork`. Claude Code's hooks reference
(https://code.claude.com/docs/en/hooks) lists the SessionStart matchers as `startup`, `resume`,
`clear`, `compact` and `fork`, so a matcher copied from the other lists could drop `fork` by
accident. The repo anchor is `.claude/settings.json`.

Question: fold this into ADR 0002's deferred SessionStart item and delete the file? `.claude/settings.json` has no SessionStart hook, so there is nothing to change; the note belongs beside the trigger that would build one (`docs/hatchery/decisions/0002-repository-layout.md`, "Deferred, with triggers"), where whoever builds it will read it. I recommend folding it in as one clause.
Answer: 2026-10-19, fold it into ADR 0002 deferred SessionStart item as one clause and delete this file.
