---
date: 2026-10-31
kind: feature
related: [2026-10-24-a-turn-ending-on-a-tool-call-could-prompt-the-engine-for-a-closing-message.md]  # that one gives the agent a closing message; this one gives it a way to ask
---

# The agent asks the venue with buttons

Recorded from the owner's conversation of 2026-10-31 as a moonshot, not sanity checked: nobody has
read the anchor against the claim yet.

Inside a session the agent can only emit text and tool calls. Give it a small tool set served by
the bot over the control socket: ask the driver a multiple-choice question that renders as buttons
or a select menu, read who is in the session and how much budget is left, and mark its turn as
waiting on a person. "Which of these three do you want?" becomes a click instead of a paragraph,
and an agent that is unsure pauses itself rather than guessing. This reverses a settled choice:
the hook contract denies every `mcp__*` tool and every engine process is launched with an empty
MCP map, so it needs the proposal loop before any code. The engine's own `AskUserQuestion` is
absent under `-p` (`docs/findings/2026-10-15-spike-stream-and-process-claims-for-adr-0020.md`, S1;
`docs/findings/2026-10-16-prior-art-chat-front-ends-for-coding-agents.md`, L3), which is why the
agent has no structured way to ask today.

Anchor: `docs/design/03-hook-contract.md` (the policy row that denies `mcp__*`);
`internal/control/server.go` (the socket the tools would be served over);
`internal/discord/customid.go` (the answers come back as clicks).

## Open

- An MCP server inside the bot, or a tool the hook intercepts and answers itself without the engine
  seeing a server; the second keeps the empty map.
- A question is a permission prompt in another shape: whether it waits the way a prompt waits
  (forever, by the vision) and whether policy mode may answer it.
- The agent reading its own budget invites it to economise in ways the owner did not ask for.

Question: Should the proposal be queued now, or after the landing passes (P33 to P37)?
It reverses the `mcp__*` deny of `docs/design/03-hook-contract.md`, so it needs the proposal loop before code.
Recommend after the landing passes; the proposal should lead with the hook-intercepted shape (a tool the hook
answers itself, no MCP server, the empty map kept), which is the smaller reversal.

Answer:
