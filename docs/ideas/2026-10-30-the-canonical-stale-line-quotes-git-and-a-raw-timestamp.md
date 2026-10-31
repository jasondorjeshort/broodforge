---
date: 2026-10-30
kind: question
related: [2026-10-30-venue-posts-shaped-for-the-reader.md]
---

# The canonical-stale line quotes git's fatal line and a raw timestamp

P30 (`internal/session/service.go`, `internal/discord/text.go`) puts the fetch's failure into the
venue's first post as 04's `canonical-stale` line. Two parts of that line are the bot's own choice,
not the owner's wording, and the reviewer flagged both:

- `<line>` is git's whole fatal line. In the repro it carried the remote's absolute path, a home path
  with a user segment; against a live remote it carries the remote URL. The same text lands in the
  `session-started` audit row and `status --json`, where it is useful; the venue post is read by
  people and may be in a channel the owner shares.
- `<when>` is the store time of the last successful fetch as an ISO timestamp, or `before the bot
  started` when this bot has made none.

Question: in the venue post, keep git's line as is, map the common failures to a short reason
(`remote unreachable`, `credential refused`, `timed out`) with the full line left to the audit row
and `status --json`, or drop the operand and keep git's wording? And for `<when>`: the timestamp,
a relative form (`3 hours ago`), or nothing? The two `04-ux.md` rows take whatever you choose.
Answer:
