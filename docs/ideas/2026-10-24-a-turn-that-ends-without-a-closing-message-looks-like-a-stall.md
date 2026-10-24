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
the footer (`Done · turn 4 · 10:21 · ...`), so it is there; it is not clear enough. The message above it
is the last tool call, and the footer looks much like the working line it replaced, so a running session
and an ended one read alike. `Done` is the same whether the agent finished or simply stopped calling
tools. The session then sat in `waiting-for-input` for 21 minutes until the owner asked "Where are we?",
which started turn 6 and made it `running-turn` again. Telling the states apart took `session show`,
`session tail --all` and a process listing from a terminal, none of which the owner had open in the venue.

The status line is the bot's, not the agent's: `workingHeader`, `waitingHeader` and `footer`
(`internal/discord/text.go`) are built by the renderer (`internal/discord/render.go`) from the session's
events. So it can state the session's real state whatever the agent did or did not write, and it should
be a lot more useful than it is.

Want: the session's state in the channel's bottom-most line, which is where the owner looks. During a
turn that line is the working message, `workingHeader` ("Working · turn 5 · 10:30") with the Stop button
(`stopButton`, `internal/discord/render.go`); the header plus the button say "running, and you can stop
it". After the turn the working message is deleted and the last line is the static footer. The same line
carries the state, and running and ended must look plainly different: while a turn runs it says what it
is doing; at the turn's end it says the session is idle and waiting for the next message, and whether the
turn ended on agent text or on a tool call ("ended on a tool call, no closing message"). No new message,
no new button.

## Open

- Whether the footer itself carries it (`Done` splits into finished and ended-without-closing-message,
  `labelDone` in `internal/discord/text.go`), or the footer is followed by a state line that the next
  turn's working message replaces. The wording is the owner's.
- Whether the bot, not the agent, should state it: the agent did not write a closing message, so the
  line has to come from the renderer (`ended` in `internal/discord/render.go`), from the event stream.
- Whether a turn whose last step is a tool call should prompt the engine once for a closing message
  (a change to the turn loop, `internal/session/turn.go`), or only be reported.
- The debug interface: a fixture that ends a turn on a tool call, so a test of `ended` can drive it.
