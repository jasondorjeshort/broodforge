---
date: 2026-10-31
kind: feature
related: [2026-10-14-ux-choices-that-could-become-options.md]  # that one lists venue options; a mirror is one more
---

# Spectator mirror of a session

Recorded from the owner's conversation of 2026-10-31, not sanity checked: nobody has read the
anchor against the claim yet.

A driver opts a session into a read-only mirror in a public channel: every venue post is also
posted there, buttons stripped, nothing read back. People with no AI experience learn what a
session looks like by watching one happen, and the community sees the project being built. The
venue stays private and the record; the mirror is a second copy of the same posts.

Anchor: `internal/discord/outbox.go` and `internal/discord/render.go` (a second target for the
rendered posts); the opt-in is a session control beside those of
`docs/design/04-ux-session-controls.md`.

## Open

- A mirror doubles the REST calls per turn against the same rate limits.
- Pasted text and tool output can carry a secret the private venue was trusted with; the mirror
  needs the same scan as a push, or shows only the bot's own lines and the agent's text.
- Whether a spectator may react or ask, and where that goes.
