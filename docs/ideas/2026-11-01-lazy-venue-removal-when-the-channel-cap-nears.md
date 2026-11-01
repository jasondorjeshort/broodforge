---
date: 2026-11-01
kind: feature
---

# Lazy venue removal when the channel cap nears

An ended session's channel now moves to the category Ended sessions (`venue.endedCategory`,
`docs/design/04-ux-session-end.md`, Ending a session) and is still deleted ten minutes after its
summary (`VenueDeleteAfter`, `internal/discord/gateway.go`, `scheduleVenueDelete`). The driver's
wish was for the channel to stay readable longer: keep ended channels until the room is needed,
then remove the oldest ended first.

What it would take: the timer replaced by a check at each venue creation (and at the start-up
sweep) that counts the channels under the ended category and in the guild; when Discord's cap is
near (50 channels per category, 500 per guild), delete the oldest ended venues, by the session's
end time, until there is room. The ended category fills first, so it sets the real limit.

## Open

- A reopened or resumed session (none exists yet) would have to move its channel back under
  `venue.category`, and lift the send lock if the lock ever holds.
- How near is near: a fixed margin, or a config key.
