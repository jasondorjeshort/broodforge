---
date: 2026-10-23
kind: feature
---

# `/session land` puts a session's commits on main without ending it

Work reaches the remote only at the end of a session (`internal/git/handoff.go`, `Run`). Under
"check in rarely; lose nothing" (00-vision, 2026-10-23) a session lives for days, so its finished
work would wait on an end nobody wants yet. The owner wants a faster loop: land what is done, keep
going.

`/session land` in the venue, `session land <id>` on the CLI, the driver's (and the same gate as
the `main` handoff). It runs that handoff's steps on the commits so far: check, rebase onto the
default branch, fast-forward push; the session stays open, its tree rebased onto what it pushed,
so the next turn builds on main. A conflict or a rejected push refuses with the reason and leaves
the tree and the session as they were.

The owner (2026-10-23) sees this as a different thing from the handoff form, not a part of it.

The owner, 2026-10-24, after s000a's P12b commits sat only in its tree under Temp: a push by the
session's own runner, after the session's own independent review, should be an acceptable handoff,
so the session's agents can land work too, not only the driver. Today the tree refuses every push
until the end (P15, `docs/design/03-architecture.md`, end of session).

## Open

- A running turn: refuse (`turn-running`), or land at the turn's end.
- What the session's end hands off afterwards: only the commits since the last land.
- The tree rebased between turns: whether `--resume` copes, and whether the agent's picture of
  the files (from earlier turns) misleads it once main's changes are under it.
- Whether `refs/sessions/<id>` moves on a land (ADR 0022's canonical ref).
- An agent's land: how the bot knows the review happened (an audit record, a trailer), and
  whether the runner calls the same gated land path or a push the tree's hook lets through.

The owner, 2026-10-31, after the first `dev-live` sessions: the wait is lifted ("Lift it."). The push
word should work mid-session, with a review phase by a subagent before the push, and `pull-request`
or a direct push per project setting.

Plan: decided by ADR 0028 (`../design/decisions/0028-landing-a-session-on-the-default-branch.md`),
which settles the open points: a running turn does not block a land (only committed, reviewed
commits move); the end hands off only the commits after the landed sha (`refs/sessions/<id>` stays,
`Base` becomes the landed sha); the tree is never rebased by the bot (a moved default branch is the
session's to rebase, review again, land again); the review is P31's fact pinned at the reviewer
call's `defer`; an agent's land is the push word (`git push` as a call on its own) deferred by the
hook's `land` cell and run by the bot, the driver process's until the subagent probe passes. Built
by P34 (`Land`, `session land`), P35 (the `land` cell, the push word's parse and resume), P36
(`/session land` in the venue) and P37 (the subagent probe) of
`../design/08-implementation-plan-landing.md`; P36's commit deletes this file.
