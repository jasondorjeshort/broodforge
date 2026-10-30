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
the tree, text-typed ones scanned like a pasted message, each path named in the turn's message, a
skipped file named with its reason), with image content blocks left for later?
Recommendation: yes, files only: one mechanism, no `Engine` or `TurnRequest` change (03 sends one
text block per turn), and the engine's Read views an image from a path. The architect first settles
03's data flow, the hook contract's path rule (`Read` under read-only reach is limited to the tree
or the canonical, `../design/03-hook-contract.md` line 127, so `sessions/<id>/files` must join them
and the ADR 0022 companion's bind list), 04's texts and `discord inject message --attach`; then one
coder pass.
Answer:
