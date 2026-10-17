**Rewrite the Hatchery history so the 28 root commits carry the owner's identity and no session links, then force-push it and re-record each project's base line.** It waits on the owner because the guard denies force-pushes for every agent, so the push of a rewritten history is a command from the owner's own terminal, and because whether to rewrite at all is the owner's call. The reasoning is in `docs/findings/2026-10-17-personal-data-leak-protocol-gap.md` (Result 4) and ADR 0018, whose paragraph on the owner's separate steps reads: "the history rewrite of the 28 root commits and the trailers, which starts at Hatchery, reaches every project that cloned it and re-records each base line".

## What the rewrite touches (classes and counts, no values)

- 28 root commits of 2026-10-11 and 2026-10-12, the root included, carry a third-party git author and committer identity: the machine's global identity before the owner set theirs on 2026-10-15. All are on origin/main.
- 25 commit messages carry a session-link trailer.
- The two sibling checkouts that cloned Hatchery carry the same 28 commits (and the trailers), so a rewrite starts here; each sibling's recorded base line (a sha of this history) changes and is re-recorded afterwards.
- Not covered here: the two first names on one line of `docs/findings/2026-10-16-handoff-split-default.md`. The coordinator replaces those in an ordinary commit; the rewrite does not need to.

## Decide first (the owner's)

1. Rewrite or leave the history as is. Leaving it is a choice: the identity stays public for as long as the repository is.
2. Whether to go private before rewriting: see `docs/operator/2026-10-17-hatchery-repository-visibility.md`. Rewriting a public repository does not unpublish clones, forks or cached views; going private first stops new copies while the rewrite is prepared.
3. The order against the visibility step.

## Sketch (untested, an agent can refine it; nothing here was run)

Account: the owner, in a terminal on the machine holding the checkouts, any shell with git and `git-filter-repo` installed. Needs first: a clean working tree, no unpushed work, and a mirror backup kept outside the checkout until the rewrite is verified.

```
# 1. Backup, outside the working tree.
git clone --mirror <hatchery-remote-url> <backup-dir>/hatchery-before.git

# 2. Work on a fresh clone, because filter-repo wants one and drops the remote.
git clone <hatchery-remote-url> <work-dir>/hatchery-rewrite
cd <work-dir>/hatchery-rewrite

# 3. Mailmap: old third-party identity to the owner's. One line, owner fills the placeholders.
#    Format: <new name> <new email> <old name> <old email>   (written to <work-dir>/mailmap)
git filter-repo --mailmap <work-dir>/mailmap \
  --message-callback 'import re; return re.sub(rb"(?m)^Claude-Session:.*\n?", b"", message)'

# 4. Check before pushing.
git log --all --format='%an|%ae|%cn|%ce' | sort -u      # only the owner's identity remains
git log --all --format=%B | grep -c '^Claude-Session:'   # 0
git rev-list --all --count                               # the same count as the backup

# 5. Push the rewritten history (the guard denies this for agents; this is the owner's terminal).
git remote add origin <hatchery-remote-url>
git push --force --all origin
git push --force --tags origin
```

Expect every sha to change from the root on, so the commit shas quoted in docs and findings go stale; the quoted ones are history, leave them.

## After the push

- Replace the local Hatchery checkout with a fresh clone of the rewritten remote (or `git fetch` and `git reset --hard` to it, from the owner's terminal); old clones still hold the old objects.
- In each of the two sibling checkouts: their history contains the old 28 commits, so a decanted project keeps them until it is itself rewritten (same tool, same mailmap, from the same root) and its base line is re-recorded to the new Hatchery sha. Each project's session does that through `update-from-hatchery`; do not rewrite a sibling that has unpushed work without its owner's say.
- GitHub keeps old commits reachable by sha and in cached views until its support purges them; a rewrite alone does not remove them. Ask GitHub support to run garbage collection and purge cached views if the old identity must be gone from the host.
- Check it took: the three commands under step 4, run again in the fresh clone.

Time: about half an hour for Hatchery; more for each sibling.

## When done

Delete this file. Update the finding's Next (the rewrite item) and `docs/HANDOFF.md`'s Waiting on user in the same sitting; the coordinator records the new base-line sha wherever a sibling's base line is kept.
