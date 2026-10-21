**Rewrites the history of this repository and of Hatchery so that every commit carries the owner's git
identity and no commit, message or tree carries the PC user name, or the machine owner's first,
middle or last name, then force-pushes both.** (The owner, 2026-10-17: any reference to those names
in any commit or commit log is rewritten.)

It waits on the owner because the guard denies force pushes to every agent and the mailmap holds
a real email that must never be written into the tree. Evidence and the list of commits:
`docs/findings/2026-10-17-net-diff-privacy-scan-let-named-commits-go-public.md`. The 28 root commits are the
owner's work under the machine's global git identity; both repositories are private until this is done.

Runs as the owner, in Git Bash, from a directory beside the two checkouts. About an hour, most
of it waiting. `git filter-repo` must be installed (`pip install git-filter-repo` or the package).
Do not run it inside a checkout that any Claude session has open.

1. Backups, both repositories. A branch on the remote and a copy on disk, before anything else:

   ```
   git -C <repo> branch backup/pre-rewrite-2026-10-17 main
   git -C <repo> push origin backup/pre-rewrite-2026-10-17
   cp -r <repo> <repo>-backup-2026-10-17
   ```

2. A fresh clone to rewrite in (filter-repo refuses a non-fresh clone):

   ```
   git clone --no-local <repo> <repo>-rewrite && cd <repo>-rewrite
   ```

3. The mailmap, written outside any repository, one line per identity to replace:
   `Owner Name <owner email> <old email>` (the old email is the one the finding describes, the
   new one the owner's). Then the replacements file: `Users/<pc user>==>Users/user`, the
   backslash form, and one line per name of the machine owner, `<name>==>the PC owner` (first,
   middle, last; the finding says which commit carries one). The same file serves messages.
   For Hatchery, check out its other branch first so it is rewritten too:
   `git checkout -b task/T94 origin/task/T94 && git checkout main`.

   ```
   git filter-repo --mailmap <path to mailmap> --replace-text <path to replacements> --replace-message <path to replacements> --force
   ```

4. Check before pushing: `git log --format=%ae%n%ce --all | sort -u` prints only the owner's address
   and the noreply ones; for each of the PC user name and the three names,
   `git log --all -i -S<name>` and `git log --all -i --grep=<name>` print nothing, and so does
   `git grep -i <name> HEAD`. `git log --oneline | wc -l` matches the backup's count.

5. Push the rewritten branch over the old one (filter-repo removes `origin`; add it back):

   ```
   git remote add origin <url> && git push --force origin main
   ```

   For Hatchery also `git push --force origin task/T94`. Do Hatchery first, then this repository, since this one's history includes Hatchery's root.

6. Afterwards: every checkout is re-cloned, not pulled (`docs/hatchery/reference/claude-code-on-windows.md`
   for the git identity settings to restore); a tester pass refreshes the docs that cite commit
   hashes; a request to GitHub support to purge the unreachable objects, with the old root sha;
   the backup branch is deleted on both remotes once the purge is confirmed, and the on-disk copies
   are kept until then; the finding is resolved.

When done: delete this entry and its line in the index, and resolve the finding.
