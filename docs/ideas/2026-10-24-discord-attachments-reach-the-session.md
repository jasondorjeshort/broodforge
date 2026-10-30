---
date: 2026-10-24
kind: feature
related: [2026-10-29-the-game-files-its-own-tickets.md]  # that one is a report posted by software; this one carries a person's attachment
---

# Discord attachments reach the session

The owner, 2026-10-24, from a live session: a file attached to a message in the venue never reaches
the agent. `internal/discord/gateway.go` builds the `Inbound` from `m.Content` alone, so the
message's attachments (name, size, content type, CDN URL) are dropped there, and the agent sees only
the text, with no sign that anything was attached.

Want: the bot downloads each attachment into a session-owned directory (beside the tree, not inside
it, so nothing lands in a commit by accident), and the message handed to the agent names the saved
paths so it can read them with its usual tools.

## Open

- The key-paste scan runs on message text today; an attachment needs the same scan before it is
  saved, and the same handling (delete the message, refuse) when it holds a key.
- Size and type limits, and what the agent is told when a file is skipped.
- Where the files live (the session directory under the root) and when they are removed (session
  end, like the tree).
- Images: saved as files the agent can read, or passed as image content blocks on the engine stream.
- The debug interface: the fake gateway's inbound message gains attachments, so a test can drive one.

Question: Do you want attachments in v1 as files only (saved under the session's directory beside
the tree, text-typed ones scanned like a pasted message with the same delete-and-refuse on a key,
every type kept under Discord's own upload limit so nothing a person could post is refused for
size alone, each saved path named to the agent in the turn's message, a skipped file named with its
reason), with image content blocks on the engine stream left for later?
Recommendation: yes, files only: one mechanism for every type, no `Engine` or `TurnRequest`
change (`../design/03-architecture.md` sends one text block per turn), and the engine's own Read
tool views an image from a path. It is a design change before any code, so the answer routes to
the architect first: 03's data flow (step 1), the hook contract's path rule (`Read` under code
reach is allowed only under the tree or the canonical today, `../design/03-hook-contract.md`, so
`sessions/<id>/files` must join the allowed prefixes and the ADR 0022 companion's bind list), 04's
texts for a skipped and a refused file (the owner's), and `discord inject message --attach <path>`
in `../design/05-debug-interface-discord.md`; then one coder pass. The related moonshot (the game
files its own tickets) waits on this.
Answer:
