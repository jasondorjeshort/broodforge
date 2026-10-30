---
date: 2026-10-30
kind: feature
---

# Venue posts shaped for the reader

The venue's posts follow reading rules: lead with the next action, numbered steps for a multi-step
ask, the state restated in each post so none needs the one before, lists capped, no preamble. A
person reading on a phone between other things gets the ask first and the detail after.

Anchor: `docs/design/04-ux.md` (the footer, the summary, the Ending line);
`internal/discord/text.go`, where the bot's own lines are built.

## Open

- A session's own text is the engine's; reshaping it means a system prompt, not the renderer.
- The caps and the order are wording, which the owner decides.

Question: whether the owner wants the posts reshaped this way, and which rules apply to a session's own text versus the bot's footer and summary (the recommendation: the bot's own lines first; a session's text is the engine's).
Answer:
