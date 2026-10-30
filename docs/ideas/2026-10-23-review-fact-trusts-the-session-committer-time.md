---
date: 2026-10-23
kind: bug
related: []
---

# The review fact trusts a committer time the session writes itself

ADR 0025 rule 7 makes a land onto the default branch wait for a fact: a `reviewer` `Agent` call
whose result's audit row is timestamped after the committer time of the session's last commit.
As built in P19 (`internal/session/workflow.go`, `ReviewFact`; `internal/git/create.go`,
`SessionTip`), that committer time is read from the commit itself, at `refs/sessions/<id>` as of
the last bundle. The session writes that commit, so it chooses the time: `GIT_COMMITTER_DATE`, or
`git commit --amend --date` with `-c` overrides, set before a reviewer ran makes any earlier
reviewer result look newer than the commit. The audit row's time is the bot's own clock and cannot
be moved by the session; the commit's time can. The P19 review raised it; the rule was not changed
in that pass.

Anchor: `ReviewFact`'s `time > <committer time>` comparison and rule 7's wording in
`docs/design/decisions/0025-session-workflow-composition.md`.

## Open

- Measure against a time the bot observes instead: when the bundle that first fetched the commit
  ran (`worktree check` or the handoff's step 1, audited or kept beside the ref), which the session
  cannot backdate; or compare tree ids, so a reviewer counts only for the tree it saw.
- Whether the attribution check should refuse a committer time earlier than the session's previous
  bundle or later than the bot's clock, which would also catch the backdating.
- An ADR amendment either way, since rule 7's sentence names the committer time; for the architect.

2026-10-30 (architect): decided, ADR 0025 Amendments 2026-10-30, and `05-debug-interface-session.md`'s
`review:` line. A bot-observed sha, not a time: when `session` sees the `reviewer` `Agent` call
launched in the stream, it bundles and fetches `refs/sessions/<id>` as `worktree check` does and
writes the sha into the call's `subagent-result` row; the newest such `ok` row names the reviewed
commit, and rule 7 holds only when that sha is the ref's current sha (a reset past it to an earlier
reviewed sha is unreviewed), reported `review: <tool use id> at <commit>`. No clock enters it. The
committer-time bounds of the second Open item are not taken (they narrow a window the sha closes);
a tree-id compare is not (the same bundle, nothing more).

Plan: coder, pass P31 of `../design/08-implementation-plan-canonical.md`, which is the brief:
`git.Refresh` in `internal/git/handoff.go` (steps 1 and 2 of `Collect`, shared with it), the bundle
at the reviewer's launch in `internal/session/turn.go` with `commit` and `commitError` in the
`subagent-result` row, `ReviewFact` comparing shas in `internal/session/workflow.go`, `SessionTip`
without its time in `internal/git/create.go`, the `review:` line in `internal/cli/session.go`,
`internal/session/review_test.go` with the backdated-commit case, `tests/repro/full-session.sh`'s
wording. Proof: the pass's; the live half on the owner's word. Delete this file in the pass's commit.
