---
date: 2026-11-08
kind: feature
related: [2026-11-07-session-tail-agent-column-and-the-agent-rows-order.md]
---

# A venue thread per subagent

Moonshot, not sanity checked. Since P40 the bot knows a subagent's call by `agent_id` (the hook's stdin,
the pending map in `internal/session/request.go`, the `agent <type>` on `session show`'s pending line and
the prompt's first line). Everything a subagent does still lands flat in the session's one channel,
indented under the `Agent` line in `session tail` and nowhere in the venue but its prompt.

The idea: each subagent launch opens a Discord thread under the session channel, named by the agent
type and the turn (`coder · turn 3`), and that thread carries the subagent's prompts and their buttons,
its tool rows and text, its own Stop (deny with `interrupt` true for that agent alone, the coordinator's
turn continuing), and its cost at the end (the `Agent` result's usage). The coordinator's turn reads as a
tree people can watch and steer one branch of; a prompt in the wrong thread is impossible to mis-click.
On the bot's side each thread is one goroutine over one `agent_id`, which is what Go is for; the
`session tail` indentation becomes the thread boundary.

Anchors: `internal/session/request.go` (`Pending.Agent`, the map by `request_id`); `internal/discord`
(the prompt message; threads are not used anywhere yet); `docs/design/04-ux.md` (Permission prompts);
ADR 0029 rule 7 (the foreground header sentence, which a per-agent Stop would soften).

## Open

- Discord's thread limits per channel and the archive rules; whether a thread per subagent is too many
  for a coordinator that launches ten.
- A background subagent's thread outliving the turn meets ADR 0029's drop at `result`; the one-process-
  per-session idea changes that.
- The venue-neutral form: `session tail --agent <id>` is the debug surface either way and could come first.
