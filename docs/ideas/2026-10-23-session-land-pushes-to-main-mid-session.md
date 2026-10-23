---
date: 2026-10-23
kind: feature
related: [2026-10-23-handoff-main-lands-a-session-on-the-default-branch.md]  # that one is the end's handoff; this lands without ending
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

## Open

- A running turn: refuse (`turn-running`), or land at the turn's end.
- What the session's end hands off afterwards: only the commits since the last land.
- The tree rebased between turns: whether `--resume` copes, and whether the agent's picture of
  the files (from earlier turns) misleads it once main's changes are under it.
- Whether `refs/sessions/<id>` moves on a land (ADR 0022's canonical ref).

Waits: the owner's proposal and ADR (2026-10-23: "I'll go through the ADR process later").
