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
session with no status line (`claude -p`, a cloud session) never writes `~/.claude/usage-state/<session_id>.json` [no-check: a per-session file],
so with the base `whenStale: "allow"` its launches are never gated.

## Open

- Whether the override should lift the stale deny too, or the stale deny should name its lift.
- Whether a headless session should read the meters another way or stay ungated on purpose.

Question: should the override lift the stale-data deny too, and should a headless session (no status line) stay ungated on purpose? Still true: `.claude/hooks/usage-gate.js` takes the `whenStale === "deny"` branch (line 157) before it reads the override variable and files (line 180 on). Both decide when launches spend money, so they are yours. I recommend that the override lifts the stale deny (one lift for every deny, as the deny-line finding concluded), and that headless stays ungated but says so in ADR 0006 and on the gate's stderr, since it has no meter to read.
Answer:
