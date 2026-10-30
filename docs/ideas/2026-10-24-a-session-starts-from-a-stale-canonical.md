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

Plan: architect, one pass, an amendment and no code: session start is a sequence in ADR 0022's
companion (Clone at session start) and a row of 03's failure table, so the fetch is designed
before `git.Create` changes. Amend the companion with a step before the clone: in the canonical,
`git fetch <remote> <default branch>` with the askpass credential as the push carries it
(`PushEnv`), then fast-forward `refs/heads/<default branch>`. Settle the mechanism: the canonical
is a non-bare clone with that branch checked out (`CloneCanonical`), so a plain `fetch +b:b` is
refused; choose between `--update-head-ok` (the worktree goes stale, which nothing reads: the copy
uses `git archive <commit>`, the clone `--branch`), `pull --ff-only`, or basing on the
remote-tracking ref. A diverged local branch is a refusal (the bot never commits there). Recommend
on fetch failure: start from the local branch, audit it, and say so in the venue's first line,
with that line and the refuse-or-warn choice marked for the owner in `04-ux.md`; `status --json`
gains the lag per project. Then 03's row, `05-debug-interface-git.md` (drive and reproduce against
a scratch remote) and the coder pass in `08-implementation-plan.md`.
