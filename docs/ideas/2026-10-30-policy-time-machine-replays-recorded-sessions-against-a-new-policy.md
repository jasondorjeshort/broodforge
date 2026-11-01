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

Question: Do you want a replay harness for policy changes now?

Recommendation: not in this wave. The goldens (`tests/discord/golden/*/events.jsonl`) carry each `tool-call`'s tool and a
summarised `target`, the session's mode and reach on `session-started`, and a few recorded `permission-decision`s, but
not the resolved paths nor the tool input that `policy.Request` (`internal/policy/policy.go`) takes, and only four
small sessions exist. A harness over the path-free classes (Bash, web, mode-only cells) would be one tester pass; the
file and path rows need a recording format first. Few sessions are on file yet, so the diff would say little; it earns
its place once real-turn fixtures exist.
Answer:
