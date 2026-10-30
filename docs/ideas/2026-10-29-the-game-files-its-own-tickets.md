---
date: 2026-10-29
kind: feature
related: [2026-10-24-discord-attachments-reach-the-session.md, 2026-10-30-the-bot-files-its-own-findings.md]  # the first carries a person's attachment; this one is a report posted by software; the last is the inward sibling, the bot reporting on itself
---

# The game files its own tickets

A moonshot the owner judged far off at filing (2026-10-29). The hosted game's error handler, or a
player's in-game bug report, posts to a Discord channel with the repro attached (log excerpt,
stack, the command that failed), and a person of a high enough role presses one button to open a
session on it. The session starts with the attachment already in its worktree and the report as
its first prompt, so the person types nothing. This is R-48's open-a-channel-on-request shape with
the request coming from software, and the R-47 test loop in reverse: the game reports in, rather
than the bot running tests out.

Anchor: the gateway builds an `Inbound` from message text alone in `internal/discord/gateway.go`
(the attachments idea is the prerequisite); the open-a-session button would be a new custom id in
`internal/discord/customid.go`; the posting side lives in the game's repository, not here.

## Open

- A bot-posted report is not a person: who is the driver and whose credential pays, the clicker's
  or a channel-level one?
- Reports from the thousand-player tier are untrusted text; they need the same scan as pasted
  text, and a rate limit per reporter.
- Whether the trigger is a webhook the game calls or a channel the bot watches.

Waits: the first usable version (`00-vision.md`, "The first usable version") has shipped; the owner, 2026-10-29.
