---
date: 2026-10-26
kind: chore
---

# O1's failure lift sends a project toward the wrong fix

In this checkout, harness case O1 (`.claude/hooks/test-hooks.js`, group O) failed with `FAIL O1
ownership: <every path under cmd/, internal/, fixtures/, share/, spike/, tests/, go.mod, go.sum>
matches no list in .claude/hooks/ownership.json; lift: classify it`.

The real cause was that the clone had no `hatchery` remote: decant step 2 adds it, and a fresh clone
carries only `origin`. O1, A1 and CB1 are skipped where a `hatchery` remote exists, because
`ownership.js --check` is for Hatchery's own tree (`docs/hatchery/decisions/0016-ownership-map.md`).
Adding the remote turned the suite green (769 passed, 0 failed, 3 skipped).

The lift "classify it" sent an agent toward editing the base-owned `.claude/hooks/ownership.json`
with project paths. That is a departure, and it would not have worked: updates read the map from the
base commit.

## Proposal for Hatchery

A carry-back request, no diff:

- O1's FAIL lift also says that in a project the fix is `git remote add hatchery <url>`, not
  classification; possibly A1 and CB1 likewise.
- The decant, or a session-start check, warns when a decanted project (one with a `Hatchery base:`
  line) has no `hatchery` remote.

Plan: no diff from here (`carry-back` skill: a request without a diff is filed by Hatchery's coordinator
as a Hatchery idea; the handoff routes it so). Hatchery's coordinator (a session in `../hatchery`) files
it in Hatchery's `docs/ideas/` from the two bullets above, `kind: chore`, with Hatchery's anchors: the O1
`contract(...)` FAIL line in `.claude/hooks/test-hooks.js` and its sentence under "Harness group `O`" in
`docs/hatchery/decisions/0016-ownership-map.md` for the lift (suggested: `lift: in Hatchery classify it;
in a decanted project git remote add hatchery <url>, decant step 2`; A1 and CB1 likewise), and for the
warning the `decant` skill's checks step and the harness's own O1 line as candidate homes, Hatchery's
triage choosing. Verify: the file is on `hatchery/main` (`git fetch hatchery; git ls-tree --name-only
hatchery/main docs/ideas/`); then the runner deletes this file. Out of scope: any edit to this project's
hooks or `ownership.json`; the lift text comes back by `update-from-hatchery`.
