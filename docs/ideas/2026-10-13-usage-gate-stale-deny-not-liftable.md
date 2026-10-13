---
date: 2026-10-13
kind: bug
related: [2026-10-13-malformed-policy-falls-back-silently.md]
---

# The stale-data deny cannot be lifted, and a headless session is not gated at all

In `.claude/hooks/usage-gate.js` the `whenStale: "deny"` branch runs before the override file and
the override variable are read, so a project that denies on stale data has no mid-session lift when
the status line stops writing: the same shape as the deny-line trap
(`docs/findings/2026-10-13-usage-gate-deny-line-trap.md`). The reverse case is also unguarded: a
session with no status line (`claude -p`, a cloud session) never writes `~/.claude/usage-state.json`,
so with the base `whenStale: "allow"` its launches are never gated.

## Open

- Whether the override should lift the stale deny too, or the stale deny should name its lift.
- Whether a headless session should read the meters another way or stay ungated on purpose.
