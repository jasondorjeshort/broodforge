---
date: 2026-11-02
kind: feature
related: [2026-10-29-broodforge-works-on-itself-overnight.md, 2026-11-02-worktree-handoff-lands-main-with-the-role-unchecked.md]
---

# An unattended session lands on idle

Moonshot, not sanity checked. ADR 0028 as amended 2026-11-02 judges the end's land by the ender's
roles, and the idle end has nobody's, so an idle end on a `main` project always falls back to the
session branch. That is the safe default and it closes the overnight idea's loop: a session the
owner starts at night and leaves cannot land `main` by finishing; somebody has to come back and
press End or run `session land`.

A grant at start, not a weaker rule: a driver who holds the land role can start a session with a
flag (`/session start ... land-on-idle`, or a project default `"landOnIdle": true` honoured only
when the starter has the role) that records the grant on the session row. The idle end then judges
the land by the starter's roles as they stand at that moment (a role removed overnight revokes it)
and still needs the review fact, so only a tip a reviewer read lands. The audit's `land` row names
the starter as actor with `via: idle-grant`, and the venue summary says the land was granted at
start.

Anchor: `internal/session/actions.go` (the idle end's roles); `internal/session/service.go`
(`StartRequest`); ADR 0028 Decision 5 and Amendments 2026-11-02; `docs/design/04-ux-session-start.md`
(what humans see: owner's call).

## Open

- Whether the grant is per session (a flag at start) or per channel (a role group whose members'
  sessions always land on idle); the first is explicit, the second is quieter.
