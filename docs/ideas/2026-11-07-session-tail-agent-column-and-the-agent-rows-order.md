---
date: 2026-11-07
kind: chore
related: [2026-10-30-two-session-leftovers-a-sixth-store-site-and-the-agent-decision-order.md]
---

# `send --wait` prints a blank field, and the `Agent` decision row lands at the result

Two small observations from the live prompt-mode session on the dev root (2026-11-07, P40e). The older related idea
covers the `Agent` row's order in the transcript renderer; this one is the live row's timing.

1. `session send --wait` prints `waiting-for-answer` with an empty field (four spaces) between the turn number and
   the tool use id, where the design shows `waiting-for-answer  <n>  <tool use id>`. Perhaps a blank agent column.
   Anchor: where `send --wait` prints its result line (`internal/session/query.go`, or the CLI file that calls it).
2. The `allow [Agent] (policy)` row lands at the `Agent` call's result, after the subagent's own rows, not at the
   launch. Anchor: `internal/session/turn.go`, where the `Agent` call's decision row is written.

## Open

- Whether the blank field is the agent column (then print the agent type or drop the column) or a stray separator.
- Whether the `Agent` row should be written at the launch, as the decision is made there.
