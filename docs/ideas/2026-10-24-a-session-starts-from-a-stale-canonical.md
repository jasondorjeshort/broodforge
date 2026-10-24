---
date: 2026-10-24
kind: bug
related: [2026-10-23-session-land-pushes-to-main-mid-session.md]
---

# A session starts from whatever the canonical's default branch was last set to

`git.Create` (`internal/git/create.go`) branches a session from the canonical's local
`refs/heads/<defaultBranch>`; nothing in `internal/` fetches the remote into it, and no doc names a
step that does. On the live root the canonical's `main` stayed at 61cccdc (2026-10-23) while the
remote moved on: s000a started four commits behind (without b3baa39 and 3aaf308, the s0008 finding
it was then partly re-deriving), and after s000a's work was cherry-picked to the remote the next
session would start twelve behind, without P12b or the base-36 id fix it is built on.

Fetch and fast-forward the canonical's default branch at session start (refuse or warn when it has
diverged), so a session starts from what the remote has. The handoff's fetch is the precedent.

## Open

- Fetch failure (offline, credential): start from the stale branch and say so in the venue, or refuse.
- Whether `status` shows the canonical's lag behind the remote.
