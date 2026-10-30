---
date: 2026-10-24
kind: feature
related: [2026-10-29-a-session-remembers-across-weeks.md]  # that one writes for the next resume; this one writes for the venue
---

# A turn ending on a tool call could prompt the engine once for a closing message

The owner, 2026-10-24, from a live session (s000a, P12b work): they could not tell whether the session
was still going or had stopped. Turns 4 and 5 each ended with `turn-ended end_turn` and no closing text,
after tool calls; the session then sat in `waiting-for-input` for 21 minutes until the owner asked
"Where are we?".

The venue now says which (owner-approved 2026-10-24, `docs/design/04-ux.md`, The turn loop): the
working header names the current step (`running Bash`, `writing`), and a completed turn's footer says
`Idle`, or `Idle · ended on a tool call, no closing message` when the turn's last entry is a tool call
(`idleLabel` in `internal/discord/text.go`, `ended` in `internal/discord/render.go`). It is reported
only; the agent still leaves such a turn without a word on what it did.

## Open

- Whether a turn whose last step is a tool call should prompt the engine once for a closing message
  (a change to the turn loop, `internal/session/turn.go`), so the venue gets the agent's own summary
  and not only the bot's footer. The owner chose report-only for the first change.

Question: should a turn whose last entry is a tool call cost one more engine process (a fixed
no-tools prompt, like the session-end closing turn) so the venue gets the agent's own closing
text, or stay report-only as chosen on 2026-10-24? Recommendation: stay report-only and try the
free fix first: one sentence in the header the agent reads at start (`04-ux-session-start.md`,
your wording) asking it to end every turn with a line on what it did and what comes next. The
extra process costs a round of about 3 s and tokens on every such turn and would pay even when the
agent simply forgot to speak; revisit this idea if the header line does not change the rate.
Answer:
