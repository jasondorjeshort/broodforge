---
date: 2026-10-31
kind: feature
related: [2026-10-23-session-land-pushes-to-main-mid-session.md, 2026-10-24-a-turn-ending-on-a-tool-call-could-prompt-the-engine-for-a-closing-message.md]  # the first is the landing this announces; the second produces the text it would post
---

# A changelog channel for landings

Recorded from the owner's conversation of 2026-10-31, not sanity checked: nobody has read the
anchor against the claim yet.

Each landing on a project's default branch posts one message to a changelog channel: the session's
closing message in plain words, the driver, the branch, the commits. The community reads what
changed without reading the venue or the git log. The text comes from the closing message the
related idea would produce; without it, the commit subjects.

A channel key for this already exists: `venue.summaryChannel` (`internal/config/types.go`,
`internal/discord/adapter.go`, `docs/design/04-ux.md`) receives each session's end
summary (commit count, PR, transcript). A landing post would add what that does not carry: one
landing may span several sessions, and a mid-session land has no end summary at all. Whether it
reuses that channel or gets a second key is open.

Anchor: `internal/discord/text.go` (the bot's lines); the `main` handoff of the related idea
(the event); `venue.summaryChannel` in `internal/config/types.go`.

## Open

- A landing by direct push outside Broodforge has no session and no closing message; whether the
  channel reports those too.
- The closing message is the engine's text, posted to a public channel: the scan applies.

Question: Should the landing post reuse `venue.summaryChannel` or get a second key, and do you want it in the first landing pass or after?

Recommendation: after the first `main` land (the landing is P34 to P36, `docs/design/08-implementation-plan-landing.md`, not built
yet), reusing `venue.summaryChannel` so a project has one place for what happened, with the post's text taken from the
commit subjects until the closing-message idea exists. The post is the bot's own text in a public channel, so you
word it. A second key can come later if the end summaries and the landings prove too noisy together.
Answer:
