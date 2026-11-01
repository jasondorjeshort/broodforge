---
date: 2026-10-31
kind: feature
related: []
---

# The simple-questions channel

Recorded from the owner's conversation of 2026-10-31, not sanity checked: nobody has read the
anchor against the claim yet.

`00-vision.md` calls it the easiest feature and lists it under "Explicitly later"; it has had no
idea file. It is R-19 in `docs/design/01-requirements.md` (status: later). The `read-only` reach
it needs already exists (`Reach` in `internal/config/types.go`, `ReachReadOnly` in
`internal/policy`); what is new is the channel's binding, the small model, and the tighter cap. One channel where anyone with the lowest role asks a question about the code, and a
read-only session on a cheap model answers it, under its own tighter token cap. No worktree
changes, no commits, no handoff: the reach is read-only and the session ends with the answer.

Anchor: `internal/config/types.go` (a channel binds to a project and a reach); the reach a
questions channel would need is the code-read-only one the vision's "Secrets stay secret" principle
describes; `internal/policy/policy.go` (every write denied for that reach).

## Open

- A session per question or one long-lived session per channel; the first is clean, the second is
  cheaper and keeps context.
- The cap is the owner's money: a number per question and per day.
- The thousand-player tier is untrusted: the question is pasted text and gets the scan.

2026-11-01, the owner, on the questions-from-Discord idea (`2026-11-01-questions-answered-from-discord.md`):
this channel is "the sort of thing we want, where a 'simple questions' channel exists with simple in-out answers
from a sandboxed haiku(?) agent that's got an independent briefing that it's talking to players not devs". So:
a Haiku-class model, read-only reach, and a briefing of its own written for players, not the project's `CLAUDE.md`.

Question: Is the simple-questions channel wanted in the next waves, and what are the per-question and per-day cap and the model?
`ReachReadOnly` exists (`internal/policy/policy.go`) and `Reach` is a channel field in `internal/config/types.go`,
so only the binding, the model and the cap are new. Recommend one session per question on a Haiku-class model,
after the landing passes, with a short design section rather than a full proposal; the cap is the owner's money.

Answer:
