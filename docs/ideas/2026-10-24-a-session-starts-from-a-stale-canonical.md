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
  Settled 2026-10-30: start from the local branch, audit it, say so in the venue; the line's wording
  is the owner's (`04-ux.md`, `canonical-stale`, pending).
- Whether `status` shows the canonical's lag behind the remote. Settled 2026-10-30: `status --json`
  shows the last fetch per project (`branch`, `commit`, `fetched`, `error`), not a lag count, since a
  failed fetch leaves the remote-tracking ref as stale as the branch.

2026-10-30 (architect): designed. ADR 0022 companion, Clone at session start, step 0 (written in
place, the living spec; Amendments 2026-10-30): the fetch goes to the remote-tracking ref with the
push's askpass credential, then the branch is fast-forwarded by `merge --ff-only` while checked out
and `update-ref` otherwise (not `fetch --update-head-ok`, which leaves the checkout incoherent; not
`pull`, which needs an upstream and runs a merge); a local branch ahead of the remote is left alone
and starts (`canonical` `ahead <n>`, no venue line: the owner's commits are the owner's, and
`tests/repro/full-session.sh` commits into the canonical without pushing); diverged, neither
containing the other, refuses the start, `canonical-diverged`; a failed fetch (a thirty-second
deadline; a missing git credential counts as one, `credential-missing`) starts from the local
branch. The fetch comes before the workflow copy, so a
session's rules are the remote's too. Also `03-architecture.md` (the failure row; the Git section's
base sentence), `05-debug-interface-git.md` (The fetch at session start), `05-debug-interface.md`
(`status --json`'s `projects`), `04-ux.md` (two rows, pending the owner).

Plan: coder, pass P30 of `../design/08-implementation-plan-canonical.md`, which is the brief:
`git.Fetch` in `internal/git/create.go`, the fetch first in `Start` (`internal/session/service.go`),
the `canonical-diverged` refusal, the `session-started` details, `status --json`'s `projects`, the
venue line from the start facts, `tests/repro/stale-canonical.sh` [no-check: written by P30] with its
`tests/README.md` row, the `Fetch` tests. The owner's, pending in `04-ux.md`: the `canonical-stale` text and whether divergence
refuses (the design's default) or warns; the coder builds the default and the text as drafted, marked
pending. Delete this file in the pass's commit.
