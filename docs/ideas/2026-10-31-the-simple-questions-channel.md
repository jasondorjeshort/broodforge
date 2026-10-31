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
