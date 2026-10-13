---
date: 2026-10-13
kind: feature
---

# Block a session end once when HANDOFF.md is stale

A Stop hook would block once per session when work happened and `docs/HANDOFF.md` is unchanged,
keeping its state in a `.claude/state/` directory. Why not now: three edge cases are unresolved
(objection 2 in `docs/design/proposals/0002-critic-review.md`), and each block costs a model turn. [no-check: not yet created]

Waits: three or more sessions end with a stale handoff despite the usage-gate text and the `commit`
skill.
