---
date: 2026-10-23
kind: feature
related: [2026-10-23-session-land-pushes-to-main-mid-session.md, 2026-10-29-broodforge-works-on-itself-overnight.md, 2026-10-31-a-changelog-channel-for-landings.md, 2026-10-31-broodforge-deploys-broodforge.md]  # the first lands mid-session; this is the end's handoff; the second runs unattended and accepts through it; the third announces each landing, the fourth builds and restarts the service after one
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

The owner, 2026-10-31: the wait is lifted ("Lift it.").

Plan: decided by ADR 0028 (`../design/decisions/0028-landing-a-session-on-the-default-branch.md`),
which settles the open points: the gate is the channel's `roles.land` (not the project's); the end
on a `main` project is the one `Land` procedure followed by the removal, fast-forward only, no rebase
(a moved default branch is `land-not-fast-forward` and falls back to `handoffFallback`); `config
check` warns rather than refuses (a remote that requires a request is a deferred refusal); the venue
and `worktree handoff` lines are the companion's, wording the owner's; `"merge"` stays bad. Built by
P34 of `../design/08-implementation-plan-landing.md` (`handoff: main`, `handoffFallback`, `roles.land`,
`Land`, `session land`, the warnings) and P36 (the venue lines); P34's commit deletes this file.
