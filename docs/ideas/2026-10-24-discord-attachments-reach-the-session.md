---
date: 2026-10-24
kind: feature
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
