# Debug interface: git handoff

Companion to `05-debug-interface.md`, which points here from its Git handoff section. The steps
are the ADR 0022 companion's (Clone at session start; Handoff at session end) and the rules are
`03-architecture.md`'s (Git); this file is the CLI side. Conventions are `05-cli-conventions.md`'s.

Surface. `worktree check|bundle|handoff <id>` are write commands the bot serves, because each
fetches into the canonical; until `serve` (P10) they are proven through the harness
(`cli.WorktreeHandlers`). `worktree list [--kept]` and `worktree forget|adopt` (ADR 0021) come
with later passes. Three hidden standalone commands serve git itself: `broodforge git-askpass
<prompt>`, which `share/askpass.sh` (Linux) and `share/askpass.cmd` (Windows) run as git's
`GIT_ASKPASS`, and `broodforge git-commit-msg <file>` and `broodforge git-pre-push`, which
`share/hooks/commit-msg` and `share/hooks/pre-push` run in a session's clone. The scripts find the
binary as `<bf>/bin/broodforge` beside `<bf>/share`.

Drive. `worktree check <id>` first prints `ok  pre-push` when the tree's `core.hooksPath` is
exactly the bot's `<bf>/share/hooks` and its `pre-push` is there, else `missing  pre-push`
(`--json`: `{"hook":"pre-push","status":"ok|missing"}`; no row when the tree cannot be read), then
runs the bundle, `git bundle verify` and the forced fetch into `refs/sessions/<id>`, then the
attribution check alone: one line per commit since the base,
oldest first, `<hash>  ok|<problems>`, problems joined by `; ` from `author:<name <email>>`,
`committer:<name <email>>` and `missing:<trailer line>`; then `ok  <commits>`, or
`error: handoff-refused-attribution: <n> of <m> commits fail the attribution check`, exit 1; with
the commits passing, a missing hook ends it with `error: pre-push-missing: ...`, exit 1 (`handoff`
is not refused for it: the tree's configuration is the session's).
`worktree bundle <id>` prints `<bundle path>  <commits>`, or `none  0`. `worktree handoff <id>
[--dry-run] [--expect <value>]` runs the five steps and prints `<outcome>  <remote>  <refspec>
<commits>  <url>  <request>`, outcome `pushed`, `pull-request`, `none` (no commits: the tree is
removed) or `dry-run`, which stops after the check; for `pull-request` the request column is
`{"url":<endpoint>,"credential":<id>,"body":{"title","head","base","body"}}`, the credential by
id only. Refusals, exit 1, tree kept: `handoff-refused-bundle` (the bundle step itself),
`handoff-refused-bundle-verify`, `handoff-refused-fetch`, `handoff-refused-attribution` (the
failing commit lines stream on stdout first), `handoff-failed-push` (git's first telling stderr
line), `handoff-failed-pull-request` (`HTTP <status>: <message>`). A session id that is not
`s` and four or more base-36 digits is `bad-args`; an unknown one `unknown-session`; one whose
`refs/sessions/<id>` is gone (handed off already) `already-handed-off`, with no audit row.
`worktree list [--kept]` (a later pass) prints `<session id>  <path>  active|pushed|kept|removed
<unpushed commits>  <missing since or ->`.

Rules the code fixes where the design was silent. The base is the merge base of
`refs/sessions/<id>` and the default branch, so no column stores it and a retry after an amend
still finds it; the fetch is forced for the same reason, and the check takes the base again
after the fetch, so default-branch commits the session merged are not checked as its own. Required trailers at a commit's
committer time: each other participant from `participant.joined_at` (R-13: a contributor joins
on their first accepted message), in join order, then each model with non-zero usage from the
`started_at` of its first turn (03, Attribution), by `policy`'s display name; the key matches without case, the
value exactly. The push runs `git -c credential.helper= push <remote> refs/sessions/<id>:refs/heads/<branch>`
with `GIT_ASKPASS` and, when the project names `repo.gitCredentialId`, the value in
`BROODFORGE_GIT_CREDENTIAL` on that one process. The helper answers a `Username` prompt with
`x-access-token` and a `Password` prompt with that value; any other prompt, or none set, prints
`error: no-git-credential: ...` and exits 1, so git reads no answer and the push fails. The hook
appends each missing line of `sessions/<id>/home/trailers` through `git interpret-trailers`
and never blocks a commit: with no binary beside `share/` it prints one `commit-msg: no ...`
line and exits 0. The `pre-push` hook is the opposite: every push from the tree, whatever remote
or path it names, prints `Broodforge: session <id> does not push. ...` (`cli.PrePushRefusal`,
`<id>` from `BROODFORGE_SESSION`, `this tree` when unset; the script prints the same line first
itself when the binary is absent, fails to start or prints something else) and exits 1, git's own `error: failed to push some refs` after it. `git push
--no-verify` and `git -c core.hooksPath= push` skip it: a refusal, not the boundary (ADR 0025,
rule 4). A done handoff sets the worktree row `removed` and the session's
end outcome to the branch or the request URL; a failed one sets the row `kept`.

Observe. `worktree list` (later pass); `audit --session <id> --kind push,pull-request-opened,handoff-refused`;
`git -C <root>/projects/scratch for-each-ref refs/sessions/`; `git -C <root>/remote/scratch.git
branch` for the pushed branch in scratch.

Reproduce. In a scratch root, commit in the session tree with a wrong author (`git -C
<root>/sessions/s0001/tree commit --author "X <x@example.com>" -m t` after a change) and run
`worktree handoff s0001 --expect handoff-refused-attribution`. The whole P8 Proof:
`go test -v -run 'TestProofWorktree|TestWorktreeRefusals|TestAskpass' ./internal/cli/`. The
`pre-push` refusal: `git -C <root>/sessions/s0001/tree push origin HEAD` from a terminal inside
the tree, then `worktree check s0001`; the P15 Proof is `go test -v -run
'TestProofPrePush|TestGitPrePush|TestPrePushHook' ./internal/cli/`. A pull
request cannot complete offline: `--dry-run` is the reproduction; `go test ./internal/git/` posts
to a local test server.

Isolation. The scratch remote is a local bare repository with `handoff` `push`; the bot's git
never opens the session tree (ADR 0022) and runs with the inherited `GIT_*` repository
variables stripped; `worktree check` reads `refs/sessions/<id>` after its own fetch. Tests make
their repositories under the test's temporary directory and commit through `os/exec` with the
four identity variables (`internal/git/gittest`).
