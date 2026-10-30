---
date: 2026-10-23
kind: question
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

Plan: architect; deliverable, a dated entry under ADR 0025's `## Amendments` (`adr` skill) restating
rule 7's fact against something the session cannot move, with the matching edit to the `review:` line
of `05-debug-interface-session.md`, naming the coder's files for the pass after it (`ReviewFact` in
`internal/session/workflow.go`, `SessionTip` in `internal/git/create.go`, the subagent-result row's
details, `internal/session/review_test.go`). Steer: Open item 2 (bound the committer time between the
previous bundle and the bot's clock, in `internal/git/attribution.go`) narrows the window but leaves it
open inside the last turn; a bot-observed sha closes it: when the reviewer `Agent` call reaches the hook,
the bot bundles and fetches `refs/sessions/<id>` as `worktree check` does, writes that sha into the
subagent-result row, and rule 7 compares shas (`review: <tool use id> at <commit>`), no clock at all; a
tree-id compare needs the same bundle and buys nothing more. Out of scope: the code change itself.
