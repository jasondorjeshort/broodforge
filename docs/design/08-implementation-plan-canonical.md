# Implementation plan: the canonical at session start and at the review

This file continues `08-implementation-plan.md` under the same conventions: the pass format, the size
words, the standing rules and the `[no-check: ...]` rule. It holds the passes that change what the
bot reads from a project's canonical after P8 and P19: the fetch of the remote's default branch
before a session is cloned (P30) and the review fact of ADR 0025 rule 7 measured against a sha the
bot bundled itself (P31), both from the 2026-10-29 triage. Each pass reads the ADR 0022 companion
(`decisions/0022-linux-spawn-and-handoff.md`, Clone at session start and Handoff at session end) and
`05-debug-interface-git.md` before its Implements line; the bot's git runs in the canonical only
and never opens a session's tree (ADR 0022), which both passes keep.

## P30. The canonical fetched and fast-forwarded before the clone (small; owner sees the venue line)

Implements: ADR 0022 companion (Clone at session start, step 0; Amendments 2026-10-30);
`03-architecture.md` (Failure modes: the `git` row for the fetch); `05-debug-interface-git.md` (The
fetch at session start); `05-debug-interface.md` (Bot lifecycle: `status --json`'s `projects`);
`04-ux.md` (Failure flows: `canonical-stale` and `canonical-diverged`, wording pending the owner).
Files [no-check: `tests/repro/stale-canonical.sh` is created by this pass]: `internal/git/create.go`
(`Fetch(ctx, s Session, askpass string, token []byte) (FetchResult, error)`: `git -c credential.helper=
fetch --quiet --no-tags --no-write-fetch-head <remote> +refs/heads/<db>:refs/remotes/<remote>/<db>` in
the canonical with `PushEnv`, under a thirty-second context deadline, then `merge-base --is-ancestor`
both ways between `refs/heads/<db>` and `refs/remotes/<remote>/<db>`; behind is fast-forwarded by
`merge --ff-only --quiet refs/remotes/<remote>/<db>` when `symbolic-ref HEAD` is the branch
(`HeadBranch`), else by `update-ref refs/heads/<db> <new> <old>`; equal is a no-op; ahead (the local
branch contains the remote's commit) is left alone with `FetchResult{Ahead: n}` from `rev-list
--count <remote ref>..<branch>`; neither is `*Error{Code: "canonical-diverged"}` naming both commits;
a fetch or fast-forward that fails or times out is `FetchResult{Stale: true, Error: <git's telling
line, or the deadline>}` and no error, since the start goes on; `FetchResult{Commit, Fetched
time.Time}` otherwise); `internal/session/service.go` (`Start`: the fetch first, before the workflow
copy, through a new `Config.FetchCanonical` closure given the session's `git.Session` and the
project's `Repo.GitCredentialID`; an id the store lacks skips the fetch as `FetchResult{Stale: true,
Error: "credential-missing"}`; `canonical-diverged` refuses as `workflow-probe-failed` does, exit 1,
`action-refused` reason `canonical`, the row ended `refused: canonical-diverged`; the result goes into
the `session-started` details as `canonical` `fetched`, `ahead <n>` or `stale: <line>` and, stale only,
into the start facts the adapter renders; `Repo` gains `GitCredentialID`; the bot keeps the last
`FetchResult` per project in memory); `internal/cli/serve.go` (the `session.Config` block beside the
`CreateTree` closure: `FetchCanonical` calling `git.Fetch` with the backend's askpass path and the
credential value read by id from `b.creds`; `b.projectRepo` carrying `repo.gitCredentialId`);
`internal/cli/status.go` (`--json` gains `projects`: one
`{"id","canonical":{"branch","commit","fetched":<time|null>,"ahead":<n>,"error":<line|null>}}` per
configured project, the text form unchanged); `internal/discord/` (the venue's first post gains 04's
`canonical-stale` line after its first line when the start facts say stale, never for ahead; dry run
and live alike); `tests/repro/stale-canonical.sh` (under `scratch run`: a throwaway clone of
`<root>/remote/scratch.git` commits and pushes to `main`; `serve`, a start on channel `100`;
`session show`'s `worktree:` branch sits on the new commit and `git -C <root>/projects/scratch
rev-parse main` equals the remote's; an empty commit in the canonical itself (`git -C
<root>/projects/scratch commit --allow-empty`), a start whose `session-started` details read
`canonical` `ahead 1` and whose `session outbox` holds no stale line; `stop`; the bare remote renamed
away, `serve`, a start that succeeds with `canonical` `stale:` in `audit --kind session-started
--json` and the line in `session outbox`; the remote restored and a second commit pushed to it from
the clone, so the two have diverged: a start refused `canonical-diverged`, `audit --kind
action-refused` holding reason `canonical`) and its `tests/README.md` row; `internal/git/git_test.go`
(`Fetch` against a bare remote: behind fast-forwards and the checkout's `HEAD` and working tree
follow; equal is a no-op; ahead is left alone with its count; HEAD on another branch updates the ref
alone; diverged is the error; a missing remote is a stale result; a stalled remote hits the deadline).
Debug surface: `session start` on a scratch root, `status --json`, `audit --kind session-started
--json`, `git -C <root>/projects/scratch log --oneline -1 main`. Proof: `go test ./internal/git -run
TestFetch`; `scratch run -- bash tests/repro/stale-canonical.sh` exit 0; `tests/repro/full-session.sh`
unchanged and passing (it commits the workflow fixture into the canonical without pushing, so its
start is the ahead case, `canonical` `ahead 1`), `restart-end.sh` unchanged and passing (`fetched`). Does not:
fetch at the handoff or in `worktree check`; fetch any branch but `repo.defaultBranch`; change
`CloneCanonical`, `worktree-create` or the session tree; reset or clean the canonical's checkout;
decide the venue wording (the owner's: the two 04 rows stay pending until the handoff's Waiting on
user clears them). Depends on: P8, P10, P19.

## P31. The review fact against the sha the bot bundled at the reviewer's launch (small)

Implements: ADR 0025 Amendments 2026-10-30 (rule 7's fact restated); `05-debug-interface-session.md`
(`session show`'s `review:` line). Files: `internal/git/handoff.go` (`Refresh(ctx, b, s) (sha string,
err error)`: steps 1 and 2 of `Collect`, the bundle, `bundle verify` and the forced fetch into
`refs/sessions/<id>`, or the ref set back to the base with no commits, returning the ref's commit;
`Collect` calls it); `internal/git/create.go` (`SessionTip` returns the sha and whether the ref
exists; its committer time goes, nothing reads it); `internal/session/turn.go` (on the `tool-use`
block of a subagent call whose `subagent_type` is `reviewer`, before the call's `tool-call` event is
logged: `Refresh` through the spawn backend, the sha kept on the `call` as `commit`, a failure kept
as `commitError`, the turn's loop otherwise unchanged; `subagentResult` writes `commit` and
`commitError` into the `subagent-result` row's details beside `agent` and `result`; the bundle runs
through a new `Config.Refresh` closure); `internal/cli/serve.go` (the `session.Config` block beside
the `CreateTree` closure: `Refresh` calling `git.Refresh(ctx, b.backend, s)`);
`internal/session/workflow.go` (`ReviewFact`: the newest `subagent-result` row of the session whose
`tool` is a subagent tool, `agent` `reviewer` and `result` `ok` names the reviewed commit, and the
fact holds only when that `commit` equals `refs/sessions/<id>`'s sha (a reset past it to an earlier
reviewed sha is unreviewed, as the ADR says); no time in the query; `Review` unchanged);
`internal/cli/session.go` (`review: <tool use id> at <commit>`; `none since <commit>` as before);
`internal/session/review_test.go` (a reviewer row whose `commit` is the tip counts; a commit after the
review, refreshed by `worktree check`, voids it; a commit made after the review with
`GIT_COMMITTER_DATE` a year back, then `worktree check`, voids it too, the case the old rule missed; a
newer `ok` row for another commit voids an older row's match; a row with `commitError` never counts;
`Task` as the tool still counts); `internal/cli/session_workflow_test.go` (its `review:` expectation
reads `at`, and its wait for the committer second to pass goes, now dead). Debug surface: `session
show <id>`, `audit --kind subagent-result --json` (the `commit` field), `worktree check <id>`. Proof:
`go test ./internal/session -run 'TestReview'` and `go test ./internal/cli -run SessionWorkflow`;
`scratch run -- bash tests/repro/full-session.sh` unchanged and passing (it asserts `review: none
since` alone); on the fake with `--fixture subagent` a scratch session prints `review: <tool use id> at <base>`
(no commits: the ref is the base); then, on the owner's word in a live or dev-root session with the
real engine: after a commit the driver launches `reviewer`, `session show` prints `review: <id> at
<commit>`, and after `git commit --amend --no-edit` in the tree and `worktree check` it prints
`review: none since <new commit>`. Does not: refresh for any subagent but `reviewer`; bundle at the
result instead of the launch; compare tree ids; change `worktree check`'s output or the handoff.
Depends on: P19; P30 is independent of it.
