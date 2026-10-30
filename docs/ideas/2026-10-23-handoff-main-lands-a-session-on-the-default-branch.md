---
date: 2026-10-23
kind: feature
related: [2026-10-23-session-land-pushes-to-main-mid-session.md, 2026-10-29-broodforge-works-on-itself-overnight.md]  # the first lands mid-session; this is the end's handoff; the second runs unattended and accepts through it
---

# A `main` handoff lands a session on the default branch

`handoff` is `pull-request` or `push` (`internal/config/types.go`, `internal/git/handoff.go`
`Run`), and `push` stops at the session branch (`refs/sessions/<id>:refs/heads/<session
branch>`): the owner then merges by hand, and with no `forge` set no request is opened either.
The owner wants a faster, less collaborative path straight onto main for projects only they use.

A third value, `"handoff": "main"`, per project (only the owner writes config). At the end:
1. The same bundle, fetch and attribution check as now.
2. Rebase the session's commits onto `<remote>/<defaultBranch>`, usually a no-op since the
   session started there.
3. Push `refs/heads/<defaultBranch>` as a fast-forward, never a force.
4. On a conflict or a rejected push (main moved, branch protection), fall back to `push`'s
   session branch and say so in the venue; nothing is lost (00-vision, 2026-10-23).

A gate beyond the channel's `start` roles: the driver must also be in a named role group, so a
channel later opened to a wider tier (Utopia's) does not let everyone who can start a session write
main. The owner agreed to the shape, the fallback and the gate (2026-10-23).

## Open

- The gate's key: a role group named on the project (`landRoles`), or the channel's.
- What the venue's end line and `worktree handoff` print for a land and for a fallback.
- `config check` refusing `main` on a project whose `forge` protects the branch, or leaving it to
  the push's rejection.
- `config_test.go` uses `"merge"` as its bad `handoff` value; it stays bad.

Waits: the owner's proposal and ADR (2026-10-23: "I'll go through the ADR process later").
