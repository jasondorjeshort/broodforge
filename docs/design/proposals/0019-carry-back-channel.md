# 0019. The carry-back channel: how a sibling's change reaches the base

Date: 2026-10-18 (revised the same day after the critic review)
Status: promoted to ADR 0019

## Question

ADR 0016 says how base changes reach a sibling (one patch-only update from the base-owned set) and
leaves the other direction to a column: a departure row's `Carry-back` is "an idea file in the
project, the Hatchery commit that took the change, or `none`". ADR 0018 says there is no PR flow in
the base and names "the first project that adopts one" as the trigger to revisit. On 2026-10-18 the
build-phase sibling opened PR 1 in Hatchery's repository; the owner called that "by our design".
Decided here: the transport upstream, who writes into Hatchery's repository, how a change lands,
how one review applies, whether the PR-text scan precedes the first landing, what the row records.

## Constraints that matter here

- How PR 1 travelled (observed): a fresh clone of Hatchery under the sibling's worktrees folder,
  one remote `origin` at Hatchery's URL, no `core.hooksPath`, the sibling's placeholder identity,
  one commit on `749377e`, pushed as `task/T117-...` by `git -C <clone> push origin` under the
  sibling's guard, whose rule 3 matches the branch and never reads the remote; not a fork.
- The scan (ADR 0018) reads `<refs> --not --remotes=<target>`: a commit already on `origin` is
  never scanned again. Observed: Hatchery's scan refuses PR 1's commit scanned alone (two
  `identity` hits: the sibling's role placeholder at a reserved `.invalid` address) and reports
  `clean, 0 commits` for a push of `main` after a merge of it. A server-side merge is no push.
- Published once, published for good: `refs/pull/<n>/head` is read-only, survives branch deletion
  and a force-push, and only GitHub Support removes it (the critic's sources). The sibling's scan,
  under the sibling's lists, is the only gate on what a push into Hatchery's repository publishes.
- Hatchery's repository was deleted and recreated on 2026-10-17 (the rewrite finding, item 2); a
  recreated repository has no PRs or issues and restarts their numbers. A sha is the durable record.
- The owner's own `Waits:` on `docs/ideas/2026-10-13-github-issues-for-intake.md` (a second
  contributor, parallel coordinator worktrees, or public users) is not an agent's to take.
- One review per change; a pushed commit is never amended; the agreement is edited in Hatchery
  only; a major choice is a proposal; living docs never name the project a practice came from.
- `git apply --3way` stages what it applies and fails whole on a file `main` has since deleted or
  a binary without `--binary` (the critic's experiment); the pipe runs in Bash. `git merge` is a
  commit-making subcommand (rule 4: the runner may, the coder may not). `git fetch <url> <branch>`
  writes `FETCH_HEAD` and no ref or remote (observed): a mistaken merge of it is outgoing to the scan.

## Options

### A. Idea file only (ADR 0016 as written)

The sibling writes `docs/ideas/<date>-<slug>.md` in its own tree and names it in the departure
row; Hatchery's coordinator reads sibling checkouts at a sync session and rebuilds the change.
Nothing foreign ever moves; two of PR 1's three changes were already queued here this way
(`2026-10-16-anchor-check-rejects-elided-paths.md`, `2026-10-15-statusline-command-cwd-relative.md`).
Cost: latency (days, there) and a second build of a diff that exists. Exit cost: none. Moving parts: none.

### B. Same-account branch and PR into Hatchery's repository (what PR 1 did)

The sibling pushes a branch into Hatchery's repository with the owner's one account and opens a
PR; Hatchery lands it. A true fork-and-PR is the GitHub norm and needs no rule on Hatchery's
side; the same-account form is what exists. `gh pr list` is the queue; the body is the account;
the number a citation. Cost: a foreign commit is published in Hatchery's repository before
Hatchery reads it, for the life of the repository; a branch cut from the sibling's `main` (the
natural move in its own tree) publishes the sibling's whole history there under the sibling's
lists; a remote-name rule cannot close that (a push to a URL, a renamed remote, a bare `git
push`, a clone with `["*"]` pushing `main`), only a URL-keyed guard rule with implicit-remote
resolution and a base check shipped to every sibling, plus a GitHub ruleset the owner's plan may
not include. Exit cost: closing PRs leaves `refs/pull/*`; removing what one carried means Support
or recreating the repository, which drops every PR number cited anywhere. Moving parts: `gh`
both sides, a guard rule, a pusher exception, a policy row, a skill.

### C. Patch from a branch in the sibling's repository, a remote per sibling

As E below, but Hatchery adds each sibling as a remote. Cost: a tracking ref per sibling branch,
which the scan treats as published, so a mistaken merge is invisible to it; a remote list to
maintain. Exit cost: drop the remotes. Moving parts: the remotes, a skill.

### D. The update run in reverse

`git diff <base line> <sibling main> -- <baseOwned> <mixed>` applied to Hatchery: every departure
at once, `none` rows included; one patch cannot be reviewed as one change. Right as a report of
rows that lack a carry-back or edits that drifted without a row. Debian DEP-3 is the precedent for
the record: `Forwarded` for where it went, `Applied-Upstream` for the landing sha. Exit cost: none.

### E. Sibling-hosted branch cut from the base, fetched by URL, landed as a fresh commit

The sibling cuts `carry-back/<slug>` from a named `hatchery/main` sha in a worktree, makes the one
change there, and pushes it to its own `origin` under its own guard and scan; its departure row
names the branch. Hatchery fetches by URL into `FETCH_HEAD`, applies the diff, reviews the staged
result, commits under its own identity and pushes `main` through its own guard, scan and pusher.
Nothing foreign enters Hatchery's repository; no guard rule, pusher exception or policy row; the
same landing serves a true fork. A branch cut from the wrong base publishes nothing new (the
sibling's history is already on its own `origin`) and the landing refuses it. The queue: the
departure rows of the sibling checkouts beside Hatchery, read by a `triage` step; the account: the
commit message, and a Hatchery issue after `--text` once the owner lifts the `Waits:` above (a
sibling agent may be the second contributor; the owner's call). Cost: no GitHub diff page or
review thread (no agent here reads one; findings go back as an idea file or, later, an issue
comment). Exit cost: delete the branches in the siblings. Moving parts: one skill, one triage step.

## Recommendation

Changed after the critic review: E, not B. The first draft recommended B with a fresh-commit
landing; objections 1 and 2 showed B's safety resting on a client rule that would not have
stopped the push that made PR 1 and on a branch base nothing fixed, with a permanent failure
mode. E keeps the landing and removes both. B returns only if the owner wants the GitHub PR page
as the reading surface, and then only with the URL-keyed rule, the base check and the ruleset.

Which channel for which change:

- A built change to a base-owned or mixed path: E. It may edit a record where the docs rule
  requires it (the hook specs, a key table) and never adds an ADR, edits `.claude/agreement/` or
  changes a decision; a sibling's name in a living table is a review finding (PR 1, ADR 0006).
- A request without a diff, a question, or a protocol change (PR 1's batch critic pass): an idea
  file in Hatchery's `docs/ideas/`, filed by Hatchery's coordinator and triaged like any other; a
  rule change is then a proposal here, never landed from a branch. An incompatible change to a
  promised surface (ADR 0013): proposal and ADR first. One change per branch (one row or one bug).

Send (the `carry-back` skill, sibling side): `git fetch hatchery`; `git worktree add
<worktrees>/carry-back-<slug> -b carry-back/<slug> <hatchery/main sha>`; the change, committed with
`Carry-back of <path>: <reason>` and the base sha in the body; before the push, `git rev-list
carry-back/<slug> --not --remotes=hatchery` lists only the worktree's commits (else the skill
prints the rebuild command); the push to its own `origin` (`git.push.branches` must match; the base
default does); the row's `Carry-back` becomes `carry-back/<slug> at <sha>`. A rule 20 hit stops
here; the lift is the sibling owner's lists, from its main session after asking the user (ADR 0018).

Land (Hatchery side): a `triage` step lists pending rows (`Carry-back` naming a branch, no landing
sha) across the sibling checkouts the handoff names; the handoff's Next carries them. The coder
runs `git fetch <sibling-url> carry-back/<slug>`, checks that `git merge-base main FETCH_HEAD` is
the base sha the message names and that `git rev-list FETCH_HEAD --not main` holds only that
branch's commits (else back to the sibling with the reason), then in Bash `git diff -M --binary
<base> FETCH_HEAD | git apply --3way`; a whole failure (a file `main` deleted, `does not exist in
index`) is applied per path as the update procedure does, the dropped path named in the landing
message. The reviewer reads the staged landing (`git diff --cached`) and the fetched messages: the
one review, for Hatchery's constraints (ADR 0013, sibling assumptions in fixtures and thresholds,
the docs rule, claims in the message); fixes go to the coder's working tree, never to the
sibling's branch. The runner runs the hook harness, commits once (subject from the message; body:
the account's substance, `Carry-back: <sibling repo> carry-back/<slug> at <sha>`, the branch's
`Co-Authored-By` lines) and deletes the `docs/ideas/` files the landing completes (PR 1: the two
above); the pusher pushes `main`. The sibling's next update takes the base's version and deletes
the row (ADR 0016, step 5); until then the row holds the branch, then the landing sha the
coordinator relays. Hatchery keeps no table.

The ADR 0018 trigger: fired, not a blocker for the first landing. Under E no PR text leaves the
machine; an issue or comment, once the owner allows issues, goes through `--text` from a file
(`--body-file`), a skill step and no hook (`gh` is outside the guard; a `gh` parser is a new scope
with its own trigger, the first text with a hit). The landing commit is scanned on Hatchery's push
like any commit, which is why it is a fresh commit. ADR 0018's "Not covered" paragraph says so.

PR 1: landed by this procedure from `origin/task/T117-...` as it stands (already published; base
`749377e`; clean under `--text`), as one commit this once. The pusher closes the PR with a comment
naming the landing sha, through `--text`; the branch delete is the owner's (rule 2: "ask the
user"). No further PR into Hatchery's repository.

The ADR carries two predictions (ADR 0012 form): by 2027-04-18 no commit under `refs/heads/*` of
Hatchery's repository has an author or committer outside `privacy.identities`; every carry-back
sent before 2027-01-18 is landed or declined, its row gone within two updates. Deferred: D's
report flag; a Hatchery issue per carry-back, trigger: the owner's `Waits:` lifted.

First step: the ADR; the `carry-back` skill (send and land); the triage step; the 0018 amendment;
then PR 1 lands.

## What would change this recommendation

- The owner wants the GitHub PR page as the reading surface: B, tightened as above.
- A sibling whose repository Hatchery cannot read: a `git bundle` or `format-patch` file through a
  channel the owner names; the landing is unchanged.
- Claude Code hooks able to see `gh` commands: the `--text` step becomes a rule.
- A planned rewrite or recreation of Hatchery's repository: nothing in E depends on GitHub objects.

## Claims to verify

1. A branch cut from a `hatchery/main` sha in a sibling worktree, pushed to the sibling's `origin`,
   has `git rev-list <branch> --not --remotes=hatchery` equal to its own commits (the send check).
2. `git apply --3way` stages what it applies, so `git diff --cached` shows the landing (git docs:
   `--3way` implies `--index`; not run here).
3. DEP-3: `Forwarded` for where a patch went, `Applied-Upstream` for the upstream commit; `-M
   --binary` with `--3way` conflicts per file on edits and both-added paths and fails whole on a
   file `main` deleted; PR 1 passed the sibling's guard on the branch name alone (the critic
   verified all three; the first draft had `Origin` and "per file, not whole").

## Critic's objections and responses

1. Accepted with change. The Constraints describe the clone form; the remote-name rule is
   withdrawn with B, which now lists what a client rule must cover and why that is too much to
   ship to every sibling for one web view. E has no inbound push to guard.
2. Accepted with change. E fixes the base: the branch is cut from a named `hatchery/main` sha, the
   send check lists the range before the push, a wrong base publishes nothing new (the push goes
   to the sibling's own `origin`), and the landing refuses a range that is not the branch's own.
   The check lives in the skill and the landing, not a guard rule: under E a mistake is not permanent.
3. Accepted. The durable record is the landing sha in the row and a landing body carrying the
   substance; B's exit cost names `refs/pull/*` and Support; the rewrite finding is a constraint;
   a recreation is listed under what would change (it would not).
4. Accepted with change. E is added at the same depth and chosen. One difference: the issue is
   deferred, because the owner's own `Waits:` on the issues idea is not an agent's to take; the
   queue is the departure rows read by a triage step, the issue a Deferred item on that trigger.
5. Accepted. The triage step and the handoff's Next are the prompt; the landing deletes the ideas
   it completes, two of them for PR 1.
6. Accepted. `-M --binary`, Bash, the per-path fallback and the fail-whole disposition are in the
   landing; not `git merge --squash`, since the coder who resolves the tree may not run `merge`.
7. Accepted with change. The guard rule, pusher exception and policy row go with B, so no deny
   text is needed; the lifts are named; PR 1 is grandfathered, its delete the owner's (rule 2).
8. Accepted. The reviewer reads `git diff --cached` after the apply, plus the fetched messages.
9. Accepted. Two predictions in ADR 0012's form, the critic's, go in the ADR.
10. Accepted. Claim 3 and option D cite `Forwarded` and `Applied-Upstream`.
11. Accepted. B is renamed to the same-account form; the fork form is noted there and lands by E.
12. Accepted. The pusher closes PR 1 and, later, any issue, after `--text` on the comment file.
