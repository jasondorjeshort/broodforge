---
date: 2026-10-31
kind: feature
related: [2026-10-29-a-session-remembers-across-weeks.md, 2026-10-30-venue-posts-shaped-for-the-reader.md, 2026-10-30-owner-less-weeks.md]  # the first is the agent's memory across a gap; this is the person's; the second shapes the bot's own lines, which this is one of; the third is a digest for the owner after unattended weeks, this is a per-return line inside a session
---

# A "since you left" line on return

Recorded from the owner's conversation of 2026-10-31, not sanity checked: nobody has read the
anchor against the claim yet.

On a person's first message to a venue after a long gap, the bot answers first with what happened
while they were away: commits landed on the branch, a prompt waiting on them, what was spent, the
session's state. It is built from the store and the audit at no engine cost, before the turn
starts. The vision's "check in rarely, lose nothing" principle applied to the return, not only to
the wait.

Anchor: `internal/discord/text.go` (the bot's own lines); `internal/session/query.go` and
`internal/store/query.go` (what `session show` already assembles); the gap is a clock comparison
against the venue's last message from that person.

## Open

- What counts as a long gap, and whether it is a channel setting.
- A person who was away while another driver worked sees the other's turns summarised; that is
  the point, but it is also a second person's work restated by the bot.
