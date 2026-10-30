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

Plan: architect, an amendment, no code: the fetch is designed (ADR 0022's companion, Clone at
session start; a row of 03's failure table) before `git.Create` changes. Amend the companion with a
pre-clone step: in the canonical, `git fetch <remote> <default branch>` with the push's askpass
credential (`PushEnv`), then fast-forward `refs/heads/<default branch>`. Settle: `CloneCanonical`
checks the branch out, so a plain `fetch +b:b` is refused; choose `--update-head-ok` (the stale
worktree is unread: the copy uses `git archive <commit>`, the clone `--branch`), `pull --ff-only`,
or the remote-tracking ref. A diverged local branch is a refusal. On fetch failure, start from the
local branch, audit it and say so in the venue's first line (that line and refuse-or-warn are the
owner's, `04-ux.md`); `status --json` gains the lag per project. Then 03's row,
`05-debug-interface-git.md` (on a scratch remote) and the coder pass in `08-implementation-plan.md`.
