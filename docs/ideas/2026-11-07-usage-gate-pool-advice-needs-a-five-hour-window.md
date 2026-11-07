---
date: 2026-11-07
kind: bug
related: [2026-10-30-credential-pool-scheduling-by-headroom.md]
---

# The usage gate's pool advice skips an account with no five-hour window

Some accounts no longer have a five-hour window (an Anthropic change, the owner 2026-11-07; the
bot's meter and parser now take either shape, ADR 0021's amendment of that date). The base hook
`.claude/hooks/usage-gate.js` judges HOLD on the windows a usage-state file has, so the gate itself
holds up: a file with `seven_day` alone is judged on the weekly meter and the deny text reads
`5h n/a`. Its pool advice (ADR 0011) does not: `reading()` returns null unless both `fiveHour.pct`
and `sevenDay.pct` are present, so an account without the window is never offered as the account
to switch to, which is the account with the most room. `.claude/hooks/test-hooks.js` has no
usage-gate case with a state file lacking `five_hour` (its S52 to S55 cover the status line only).

Base-owned: the fix is a carry-back to Hatchery (`carry-back` skill), with a test case, not an edit
here. The usage-gating agreement's session-meter row also assumes the window exists; a sentence
there saying a missing meter is skipped would close the reading.

## Open

- Whether the status line shows `5h n/a` or drops the segment on such an account.
