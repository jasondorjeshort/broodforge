---
date: 2026-10-17
status: open
area: git / privacy check
---

# A net-diff privacy scan passed commits that still carried the PC user name; the public history holds it, and 28 root commits carry the machine's git identity instead of the owner's

## Question

The pusher's check 5a stopped a push at 09:0x on 2026-10-17 with two hits: the coordinator had
written the PC home path into `docs/HANDOFF.md` (commit `d00bb95`) and into an idea file
(`e32ddef`). The coordinator fixed both with a new commit on top (`9f841b5`), a fresh pusher
passed every check, and four commits went to `origin/main`. The independent review of the new
git-guard rule 20 (the same scan, moved into the hook) then asked whether the push had sent the
name anyway. It had. This finding records what is public, why the check passed, and what the
rewrite must cover.

## What was tried

- `git log -S'Users/<pc user>' origin/main` and the backslash form, on the checkout at `9f841b5`.
- `git log --format=%ae origin/main | sort | uniq -c`, the same for `%ce`, and
  `git merge-base --is-ancestor <sha> hatchery/main` for every commit not authored by the owner.
- The reviewer's scratch repositories: a branch that adds a name in one commit and removes it in
  the next, pushed with an upstream and without one, under the rule as first written.

## Result

- Content: five commits on `origin/main` add lines that carry the PC user name in a path:
  `d3a0b73` and `94e7667` (2026-10-16, git-guard test strings), `e32ddef`, `d00bb95` and
  `9f841b5` (2026-10-17; the last one adds the fixed lines and removes the named ones, so `-S`
  lists it too). The tree at `9f841b5` is clean: `git grep` finds nothing.
- Identity: 28 commits, the Hatchery base's root commits of 2026-10-11 and 10-12, carry the machine
  owner's personal email as author and committer: the owner's own work, committed under the PC's global
  git identity before the repo-local one was set on 2026-10-14 (the owner, 2026-10-17: every commit
  so far is his). All 28 are on `hatchery/main` as well; every
  later commit in both repositories descends from them. The other 151 carry the owner's identity.
- Names: one commit, `48a43e4` (2026-10-16, ADR 0015 and a finding on split timing), adds a line
  carrying the machine owner's first name; no commit message in either repository carries any of
  her names (`git log --all -i --grep`, checked 2026-10-17). The tree at HEAD carries none.
- Why the check passed: check 5a and rule 20 as first written scanned `git diff @{u}..HEAD`, the
  net change between the remote and the tip. A commit that adds a name and a later one that
  removes it leave a clean net diff while both commits are sent. The reviewer reproduced it with
  an upstream and on a fresh branch. Git sends commits; the scan must read each one.
- The repository was made private by the owner on 2026-10-17 after this was reported. Rule 20 is
  being rewritten to scan every outgoing commit (content, message, added names, author and
  committer); `CLAUDE.md` says a hit in unpushed history is amended or squashed, never fixed on top.

## Next

- Owner: the history rewrite of both repositories, after the current work lands. Before it: a
  backup branch at the current tip and a copy of the checkout on disk. Then `git filter-repo`
  with a mailmap for the 28 root commits and a text replacement for the five content commits,
  a force push of both `main` branches by the owner (the guard denies force pushes to agents),
  every checkout re-cloned, the docs that cite commit hashes refreshed, and a GitHub support
  request to purge the old objects.
- Coordinator: once rule 20 scans per commit and the rewrite is done, resolve this finding and
  promote the rule's shape to the hook spec (ADR 0002 already carries the row).
