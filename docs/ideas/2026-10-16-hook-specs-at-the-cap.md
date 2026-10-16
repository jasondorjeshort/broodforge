---
date: 2026-10-16
kind: chore
---

# The hook specs sit at the 300-line cap

Anchor: `docs/design/decisions/0002-hook-specs.md`, exactly 300 lines after ADR 0011 stage 2
(the cap is `.claude/rules/docs.md`). The last edit compressed section 3's failure-reason
sentence to fit; the next edit to any section has to cut elsewhere first. Section 3 (the gate and
the status line) is the largest. Split by topic, as the `adr` skill allows for a sibling file:
`0002-hook-specs-usage.md` [no-check: proposed file] for section 3, with a pointer left behind,
and the harness and script headers that cite "section 3" updated in the same change.

Same area, same pass: `0011-usage-cache-per-session-and-pool-advice.md` is 152 lines, two over
the `adr` skill's 150; trim it in the next edit that touches it.
