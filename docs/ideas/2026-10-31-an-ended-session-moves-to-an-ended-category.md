---
date: 2026-10-31
kind: feature
related: [2026-10-31-end-a-session-without-handing-off.md]
---

# An ended session's channel moves to an "Ended sessions" category

Not sanity checked. When the driver ends a session its channel stays under the category Sessions
(`venue.category`, made by `internal/setup/discord.go`'s channels step) until the channel is
removed, which takes a while. In that period the ended channels sit among the live ones and
distract. The driver's wording: move the channel to another category, "ended sessions", on
`/session end`; removal could then be lazier, held back until the guild's channel cap is near.

What it would take: a second category beside Sessions (a setup slot like the first, with its own
config key beside `venue.category`); the end path sets the channel's parent to it (a
channel-edit REST call, `ManageChannels` already in `categoryPerms`); removal moves from a timer
to "oldest ended first, when the cap needs the room". The difference from the related discard idea:
that one is about not handing off, this one is about where the channel waits afterwards.

## Open

- Does the lazy removal belong in this idea or in a later one? The cap is Discord's 500 channels per
  guild and 50 per category, so the ended category fills first and sets the real limit.
- A moved channel keeps its history and the driver's read access; whether the bot should also
  lock it (no send) so a stray message does not look like it reached a live session.
- A resumed or reopened session would have to move back.
