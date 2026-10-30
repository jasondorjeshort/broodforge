---
date: 2026-10-30
kind: feature
related: [2026-10-29-the-game-files-its-own-tickets.md, 2026-10-22-discord-layer-trace-for-debugging-the-live-bot.md, 2026-10-30-owner-less-weeks.md]  # the first is the outward sibling (the hosted game reports); this is inward (the bot reports on itself); the second is the trace such a session would read; the third is the combined weeks-away vision
---

# The bot files its own findings

A moonshot the owner judged far off at filing (2026-10-30). On a refusal or warning in `serve.log`,
the bot opens a scratch session that reproduces it against the fake engine and posts a finding
with a repro script. The game-tickets moonshot has software report a bug from outside; this has
Broodforge report its own, with the evidence a `finding` skill file needs (question, what was
tried, result).

Anchor: `serve.log`, the bot's stderr that a run keeps (`docs/design/05-debug-interface-discord-live.md`), and the audit rows the bot already writes;
the `finding` skill; the fake engine (`fixtures/engine/`) as the safe place to reproduce.

## Open

- A warning storm must not open a storm of sessions: a rate limit and a dedupe by signature.
- A log line can carry text from a user; the scan applies before it reaches a prompt.
- Whose credential pays (R-46).

Question: whether the bot may open sessions on itself, and where the finding goes.
Answer:
