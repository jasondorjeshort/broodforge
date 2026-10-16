---
date: 2026-10-15
kind: question
related: [2026-10-15-usage-cache-not-keyed-by-account.md, 2026-10-13-fable-per-model-usage-gate.md]
---

# What HOLD should mean when the owner rotates between several accounts

Anchor: `.claude/hooks/policy.json` (`usage` block) and `.claude/hooks/usage-gate.js`.

The gate judges the meters of the account that is logged in now. With an account switcher such as
claude-swap (`cswap auto`), an owner with several accounts moves to the one with the most
quota when the active one reaches a threshold (cswap default 90%, set by `autoswitch.threshold`).
The base HOLD is 50%. So the gate stops delegation long before the switcher would move on, while
the other accounts still have most of their quota. The two thresholds were never set against each
other.

Shapes, none chosen:

- Leave the gate per account; a project that uses a switcher raises its HOLD in its own policy
  file to at or above the switcher's threshold.
- The gate reads how much room the whole set of accounts has (`cswap list --json` gives every account's
  5h, 7d and per-model readings) and holds only when no account has room.
- Unchanged: HOLD stays a pacing rule per account, whatever the switcher does.

This decides what the owner spends, so it is the owner's call (CLAUDE.md, Major decisions).
