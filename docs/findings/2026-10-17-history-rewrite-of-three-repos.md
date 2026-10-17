---
date: 2026-10-17
status: resolved
area: git
---

# Removing a second account's identity, session links and personal data from three repos that share commits

## Question

How to take a second account's identity, `Claude-Session:` links and personal data out of the
pushed history of three repositories (the base and two siblings) that share commits, leaving every
HEAD tree identical, and how to prove it before anything is pushed.

## What was tried

In order:

1. A mailmap, a message callback that strips session-link lines, and `git filter-repo
   --preserve-commit-hashes`. The last flag is why: a commit shared by two repositories gets the
   same new hash in both, so the shared history stays shared. Cost: hashes quoted in messages go
   stale and are re-recorded from the maps.
2. The first base-repo push was audited after the push, and two names turned up in one old blob (the
   mailmap and callback never touch file contents). The GitHub repository was deleted and recreated,
   and the run repeated with `--replace-text` rules for file contents. Audits moved before every
   push; a pushed history that fails the audit costs a recreated repository.
3. A case-insensitive substring scan of every object (blobs, paths, messages, identities, tags) of
   all three repositories, for every name and account-name fragment. It found the second account's
   Windows user name in sibling history and in one sibling's current files. The current files were
   fixed by a normal commit in that sibling first, so HEAD trees stay identical and the rewrite
   rules touch only history; regex rules for files and `--replace-message` rules for messages
   covered the rest.
4. Tags went with `--follow-tags`. GitHub Actions were turned off before the first push of each
   new repository. One release was recreated from saved notes and assets. Secrets were not carried
   (a new repository has none).
5. The git-guard blocked pushes whose history holds pre-existing content hits, so the owner ran
   the pushes in their own terminal.
6. Working directories were switched in place instead of replaced by fresh clones: fetch, tree
   check against the new head, `reset --soft`, tags and remotes re-fetched. This keeps ignored files
   (keys, caches) and works while a session runs in the directory.

## Result

Final heads: the base repository 33b443b, the first sibling e1de16b, the second sibling 3f7bd8e
plus 7 tags. Each run printed the checks the kit lists: one identity, no session-link lines, same
commit count, identical trees, shared commits with identical new hashes, every forbidden word 0,
privacy scan clean. The old repositories remain on GitHub as `<repo>-backup`; mirror clones and
bundles are kept as well.

The kit, the old-to-new commit maps and the written procedure live outside every repository, at
`../history-rewrite`. It holds identities and is never committed or pasted into docs.

Lessons that carry over:

- Search every blob of the old history for names before the first run, not after.
- After a rename, old clones' `origin` points at the new empty repository, and a stale local
  `origin/main` makes the privacy scan skip the old history on a push from such a clone. Never push
  from an old clone (`docs/ideas/2026-10-17-privacy-scan-trusts-stale-tracking-refs.md`).
- New GitHub repositories came out public twice; create with `--private` and check with
  `gh repo view --json visibility`.
- The git-guard denied some read-only audit commands as false positives
  (`docs/ideas/2026-10-17-guard-false-positives-in-audits.md`).

## Next

- Each rewritten sibling, in its own session, re-records its base line and the commit hashes its
  docs cite from the maps. The base repository's own were re-recorded with this finding.
- Regenesis and droplet-management were not rewritten (the owner: mixed authorship is fine there);
  each re-records its base line from `hatchery.commit-map` in its own session: regenesis
  `docs/design/decisions/0003-hatchery-adoption.md` line 5 (6866d8a, new d343b69) and
  droplet-management `docs/design/decisions/0003-adopt-hatchery.md` line 6 (48a43e4, new 2f98979).
- In dystopo, from the review of its name-removal commit: the task worktree row of
  `docs/reference/machines.md` writes a literal `git -C ~\...` command that does not run as written
  (spell it `$env:USERPROFILE\`), and four `~` paths lost whose home they are (ADR 0020 line 9,
  `docs/PLANNING.md` line 574, `docs/TASKS.md` line 740, the soak finding of 2026-10-16 line 18):
  write "the second account's `~\...`".
