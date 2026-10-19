---
date: 2026-10-17
kind: bug
related: [../findings/2026-10-17-history-rewrite-of-three-repos.md]
---

# The privacy scan trusts local remote-tracking refs, so a push from an old clone scans nothing

`.claude/hooks/privacy-scan.js` (ADR 0018) computes its range as `rev-list <refs> --not
--remotes=<target>`. That excludes every commit a local remote-tracking ref already holds, on the
assumption that the remote has them. It also excludes `--remotes=hatchery`, so a sibling with a
stale `hatchery/main` skips the same way.

After a remote is renamed and a new empty repository is created under the old name, an old clone's
`origin` points at the empty repository while its local `origin/main` still holds the old head. A
push of the whole old history then scans zero commits, and the old history uploads unscanned.
Found during the 2026-10-17 history rewrite.

Reproduce: clone a repository, rename it on the remote, create an empty one in its place, push
from the old clone; the scan reports an empty range.

Suggested fix: in `--pre-push`, use the remote shas git passes on stdin and exclude only those; in
the guard, ask the remote for its real heads (`git ls-remote <target>`) and exclude only those. A
remote that cannot be reached denies, as a range that cannot be computed already does.

Plan: `coder`, in `.claude/hooks/privacy-scan.js` (the range, lines 385-393) and `docs/hatchery/decisions/0018-privacy-scan.md`. `--pre-push` excludes only the non-zero remote shas git sends on stdin, never `--remotes=<target>` or `--remotes=hatchery`. The guard and CLI paths ask the remote (`git ls-remote <target>`) for its heads and exclude only those present locally; an unreachable remote is `cannot` (exit 2, deny), as an uncomputable range already is. PV harness cases: this file's repro (old clone, empty remote, stale `origin/main`) scans every commit; an ordinary up-to-date push still scans only the new ones. Verify: `node .claude/hooks/test-hooks.js`. Out of scope: the never list and the classes.
