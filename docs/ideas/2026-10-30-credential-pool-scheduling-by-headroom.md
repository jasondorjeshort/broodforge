---
date: 2026-10-30
kind: feature
related: [2026-10-29-broodforge-works-on-itself-overnight.md, 2026-10-30-owner-less-weeks.md]  # the first is the unattended work this would route; the second is the combined weeks-away vision, which needs this piece
---

# Credential pool scheduling by headroom

A moonshot the owner judged far off at filing (2026-10-30). With P28's per-credential meters
(the `credential_meter` table), Broodforge routes overnight work to whichever subscription has
headroom, across the owner's accounts or a small team's. This extends ADR 0011's pool advice from
"tell the owner another account has room" to "the bot decides". Accounts still change at session
boundaries, never under a running session. Money is the owner's (R-46): the bot never spends past
a cap the owner set.

Anchor: ADR 0011 (`docs/hatchery/decisions/0011-usage-cache-per-session-and-pool-advice.md`, the `usage.pool` advice);
`credential_meter` from P28 (`docs/design/08-implementation-plan-engine-stream.md`).

## Open

- Whose credential may carry whose work, for a team.
- What headroom means near a reset: a nearly full meter that resets in an hour.

Question: whether the owner wants the bot to choose the account, and under what cap.
Answer:
