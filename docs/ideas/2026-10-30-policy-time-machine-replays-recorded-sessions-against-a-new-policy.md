---
date: 2026-10-30
kind: feature
related: [2026-10-30-owner-less-weeks.md]  # that one combines ideas; this is a check on policy changes alone
---

# Policy time machine: replay recorded sessions against a new policy

A moonshot. Replay the tool calls of every recorded session on file through the policy layer (not the
renderer, which `discord render --golden` already covers) and diff the resulting decisions against the
recorded ones. A changed rule then shows what it would have allowed or denied in every real session,
at no engine cost, before the owner accepts it.

Anchor: the goldens in `tests/discord/golden/` and the decision code in `internal/policy/policy.go`.
