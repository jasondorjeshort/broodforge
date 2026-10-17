**Rewrite the Hatchery history so the 28 root commits carry the owner's identity and no session links, publish it as a fresh repository, and re-record each project's base line.** It waits on the owner because git filter-repo is not installed and the guard denies it to every agent in every repository, so the rewrite runs from the owner's own terminal. The reasoning is in `docs/findings/2026-10-17-personal-data-leak-protocol-gap.md` (Result 4) and ADR 0018 (Consequences, the `--preserve-commit-hashes` bullet).

## What the dry run established

- The rewrite kit is in `~/.claude/hatchery-rewrite/`: `run.sh`, the mailmap, the message filter, a preflight script. It is outside the repository because the mailmap holds a personal value.
- The preflight showed one identity after the mailmap and 25 messages losing their session-link lines. The tree at HEAD must come out identical.
- `run.sh` refuses a source with uncommitted changes. It prints the checks: one identity, no trailers, equal commit counts, identical tree, the privacy scan.
- The scan's remaining hits on the old history are exactly the classes the rewrite clears: identity, email on the 28 commits, session links in messages.
- The rewrite runs with `--preserve-commit-hashes`, so a hash quoted in a message is translated only when that commit is in the repository being rewritten. That is why a sibling is rewritten after Hatchery, with Hatchery's map available for the shared base commits.

## Decided (the owner's)

- All repositories are private now.
- Hatchery and Dystopo get a fresh repository: rename the old one on the host, create a new one under the old name, push.
- Broodforge gets the same rewrite.
- Regenesis and droplet-management keep their history; they only re-record their base line from the hash map after fetching the new Hatchery.

## Commands, in order

Before you start: mirror backups of hatchery, broodforge and dystopo are at `~/.claude/hatchery-rewrite/backup/<name>.git`. For any other repository make one first:

```
git clone --mirror <source> ~/.claude/hatchery-rewrite/backup/<name>.git
```

The script refuses a source with uncommitted changes, clones it with `--no-local` into `~/.claude/hatchery-rewrite/<name>` and rewrites only that clone; it never writes to the source repository or to any remote. The old history also survives in the working directories, the renamed GitHub repository and the sibling clones until you delete them.

Account: the owner, any shell with git, Python and pip. Needs first: a clean working tree in each source, nothing unpushed.

```
pip install git-filter-repo
bash ~/.claude/hatchery-rewrite/run.sh <path to this repository>
```

Read the printed checks; all must pass before pushing. Then, with the new repository created empty on the host:

```
git -C ~/.claude/hatchery-rewrite/hatchery remote add origin <new url>
git -C ~/.claude/hatchery-rewrite/hatchery push -u origin main
```

For Dystopo and Broodforge, after Hatchery's run, the same `run.sh` with the sibling's path, then the same two push commands against that sibling's kit directory and its new url.

## After the push

- Replace every clone with a fresh clone, Hatchery's first. Old clones still hold the old objects.
- Regenesis and droplet-management: fetch the new Hatchery and re-record the base line from the hash map.
- A tester re-records the cited hashes from the map: 13 in living documents of this repository.
- The old, renamed repository can be deleted on the host when nothing needs it.

Time: about half an hour for Hatchery; less for each sibling once the kit is proven.

## When done

Delete this file. Update the finding's Next (the rewrite item) and `docs/HANDOFF.md`'s Waiting on user in the same sitting; the coordinator records the new base-line hash wherever a sibling's base line is kept.
