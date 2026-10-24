---
date: 2026-10-24
kind: feature
---

# A turn that ends without a closing message looks like a stall

The owner, 2026-10-24, from a live session (s000a, P12b work): they could not tell whether the session
was still going or had stopped, and nothing in the venue said which. Both readings were reasonable
from the channel alone.

What the record shows. Turns 4 and 5 each ended with `turn-ended end_turn` and no closing text: turn 4
after two `Grep` calls (12:15 local), turn 5 mid-way through a proof script (12:48). The venue did post
`internal/discord/render.go` `ended`'s delete-and-replace with the footer (`Done · turn 4 · 10:21 · ...`),
but the message above the footer is the last tool call, so the channel reads as work interrupted, and
`Done` reads the same whether the agent finished or simply stopped calling tools. The session then sat in
`waiting-for-input` for 21 minutes until the owner asked "Where are we?", which started turn 6 and made it
`running-turn` again. Telling the three states apart took `session show`, `session tail --all` and a
process listing from a terminal, none of which the owner had open in the venue.

Want: when a turn ends, the venue says what state the session is in and what it needs, in the channel
where the owner is looking. At least:

- Whether the turn's last event was agent text or a tool call, so "ended mid-task, no closing message"
  is stated rather than inferred from a footer.
- That the session is idle and waiting for the next message, as against running, in words beside the
  footer (`labelDone` in `internal/discord/text.go` covers both "finished" and "stopped talking").

## Open

- Whether `Done` splits into labels (finished with a closing message; ended without one), or a line is
  added above the footer. The footer wording is the owner's.
- Whether the bot, not the agent, should state it: the agent did not write a closing message, so the
  line has to come from the renderer (`ended` in `internal/discord/render.go`), from the event stream.
- A way to ask "is it running?" from the venue without a terminal: the pinned header
  (`session show --header`) could carry `running` or `waiting for you` and since when.
- Whether a turn whose last step is a tool call should prompt the engine once for a closing message
  (a change to the turn loop, `internal/session/turn.go`), or only be reported.
- The debug interface: a fixture that ends a turn on a tool call, so a test of `ended` can drive it.
