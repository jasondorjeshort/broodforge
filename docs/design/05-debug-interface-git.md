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
`error: handoff-refused-attribution: <n> of <m> commits fail the attribution check`, exit 1. With
the commits passing, the push scan (ADR 0025 rule 7) prints one `privacy: <sha>  <class>
<location>` line per hit (`--json`: `{"privacy":<sha>,"class","location"}`), then `error:
handoff-refused-privacy: <n> of <m> commits carry personal data`, exit 1, the tree kept; with
none, a missing hook ends it with `error: pre-push-missing: ...`, exit 1 (`handoff`
is not refused for it: the tree's configuration is the session's).
`worktree bundle <id>` prints `<bundle path>  <commits>`, or `none  0`. `worktree handoff <id>
[--dry-run] [--expect <value>]` runs the five steps and prints `<outcome>  <remote>  <refspec>
<commits>  <url>  <request>`, outcome `landed` (a `main` project: the land below, the role unchecked
as a host's), `pushed`, `pull-request`, `none` (no commits: the tree is
removed) or `dry-run`, which stops after the check; for `pull-request` the request column is
`{"url":<endpoint>,"credential":<id>,"body":{"title","head","base","body"}}`, the credential by
id only. Refusals, exit 1, tree kept: `handoff-refused-bundle` (the bundle step itself),
`handoff-refused-bundle-verify`, `handoff-refused-fetch`, `handoff-refused-attribution` (the
failing commit lines stream on stdout first), `handoff-refused-privacy` (the `privacy:` lines
first; the session's venue gets the ADR 0025 companion's push-scan text), `handoff-failed-push` (git's first telling stderr
line), `handoff-failed-pull-request` (`HTTP <status>: <message>`). A session id that is not
`s` and four or more base-36 digits is `bad-args`; an unknown one `unknown-session`; one whose
`refs/sessions/<id>` is gone (handed off already) `already-handed-off`, with no audit row.
`worktree list [--kept]` (a later pass) prints `<session id>  <path>  active|pushed|kept|removed
<unpushed commits>  <missing since or ->`.
`session end <id>` (05-debug-interface-session.md) runs the same handoff with one step before
the bundle: a tree with uncommitted changes (`git status --porcelain` lists anything) is committed
through the backend's worktree-wip (`git add --all`, then `git commit --no-verify` as the driver
with the subject `WIP: uncommitted changes committed at end` and the trailers `Required` gives for
now, `core.hooksPath` pinned to `<bf>/share/hooks`; a HEAD off the session branch refused), so its
`--json` line carries `wip: <sha>` and the `session-ended` audit row a `wip` key; a failure is
`handoff-refused-wip`, the session kept open. `worktree handoff` makes no WIP commit. `session end
<id> --discard` ends without a handoff: the WIP commit, then the bundle, verify, fetch and check
(which decides nothing here), `refs/aborted/<id>` set to `refs/sessions/<id>` when there are
commits, the tree and `refs/sessions/<id>` removed, nothing pushed; it prints `discarded  <remote>
refs/sessions/<id>:refs/aborted/<id>  <commits>  -  -` (the refspec `-` with no commits; `--json`
adds `ref` and `wip`), audits `session-discarded` (`commits`, `ref`, `wip`) and `session-ended`
with outcome `discarded`. A refusal is the handoff's code, or `handoff-refused-aborted-ref` when the ref cannot be written, the tree, `refs/sessions/<id>` and the session kept. The WIP status and add leave out every other worktree inside the tree (`git worktree list`; a subagent's is never a gitlink), and the status takes no optional lock. The Discord
route is the header's `bf:<id>:discard`, which posts the confirm, then its `bf:<id>:discard-confirm`
(`04-ux-session-end.md`).

Rules the code fixes where the design was silent. The base is the merge base of
`refs/sessions/<id>` and the default branch, so no column stores it and a retry after an amend
still finds it; the fetch is forced for the same reason, and the check takes the base again
after the fetch, so default-branch commits the session merged are not checked as its own. Required trailers at a commit's
committer time: each other participant from `participant.joined_at` (R-13: a contributor joins
on their first accepted message), in join order, then each model with non-zero usage from the
moment the bot first recorded it (`model_seen`: the turn's start for the turn's own model, the
`Agent` launch for the model a call pins, else its first `assistant` line), and from the
`started_at` of its first turn only when the record has no row for it (03, Attribution; ADR 0025
rule 10 as amended 2026-10-31), by `policy`'s display name; the key matches without case, the
value exactly. The push scan (`internal/git/privacy.go`) ports the base's `privacy-scan.js` classes
over author, committer, message and added lines (not file names, tags, transcripts or key shapes):
`home-path`, `email`, `name` (an author or committer pair outside the identities, or as a whole
word, any case, the OS user name, the host name or a `privacy.never` literal, each of 3 or more
characters), `host` and `session-link`; `<location>` is
`<path>:<line>`, `message:<line>`, `author` or `committer`, one line per (commit, class, location).
Identities are the instance's `privacy.identities` (`Name <email>` each) plus the driver, the other
participants and the models with usage; an identity's address passes as author or committer, in a
trailer-block `Co-Authored-By` naming it, and in an added line of `.claude/hooks/policy.json`.
`privacy.allow` lifts a hit whose match equals an entry, any case (a home path's user segment, a
host's word); the bot's service account (`broodforge`, `broodforge-engine`) must be in the
instance's `privacy.allow`, since the OS user name is a never literal and the project's name is in
every import path, else each commit naming the project shows a `name` row;
`config check` refuses a `never` entry under 3 characters, an `allow` entry holding
`*` or `/`, and an identity not `Name <email>`. A scan whose `git log` fails is
`handoff-refused-fetch` with git's line, never a hit. The push runs `git -c credential.helper= push <remote> refs/sessions/<id>:refs/heads/<branch>`
with `GIT_ASKPASS` and, when the project names `repo.gitCredentialId`, the value in
`BROODFORGE_GIT_CREDENTIAL` on that one process. The helper answers a `Username` prompt with
`x-access-token` and a `Password` prompt with that value; any other prompt, or none set, prints
`error: no-git-credential: ...` and exits 1, so git reads no answer and the push fails. The hook
appends each missing line of `sessions/<id>/home/trailers` through `git interpret-trailers`
and never blocks a commit: with no binary beside `share/` it prints one `commit-msg: no ...`
line and exits 0. The bot rewrites that file at each first sighting of a model and at every
process start; the `pre-tool-use` hook, on an `Agent` call it answers `allow`, appends the call's
pinned model's line itself (`tool_input.model`, else the copy's `agents.json` `model` for the
type; an unpinned type adds nothing, and a failed write changes no decision, each one `hook.log`
line: the one error the hook logs instead of failing closed on), so a subagent's first commit
carries it. `hook try --tool Agent --input '{"subagent_type":"runner"}' --tree <tree> --workflow
<copy>` prints the line in its `pin` column (`try` reads no environment and writes no file). The `pre-push` hook is the opposite: every push from the tree, whatever remote
or path it names, prints `Broodforge: session <id> does not push. ...` (`cli.PrePushRefusal`,
`<id>` from `BROODFORGE_SESSION`, `this tree` when unset; the script prints the same line first
itself when the binary is absent, fails to start or prints something else) and exits 1, git's own `error: failed to push some refs` after it. `git push
--no-verify` and `git -c core.hooksPath= push` skip it: a refusal, not the boundary (ADR 0025,
rule 4). A done handoff sets the worktree row `removed` and the session's
end outcome to the branch or the request URL; a failed one sets the row `kept`.

Observe. `worktree list` (later pass); `audit --session <id> --kind push,pull-request-opened,handoff-refused,session-discarded`;
`git -C <root>/projects/scratch for-each-ref refs/sessions/ refs/aborted/`; `git -C <root>/remote/scratch.git
branch` for the pushed branch in scratch.

Reproduce. In a scratch root, commit in the session tree with a wrong author (`git -C
<root>/sessions/s0001/tree commit --author "X <x@example.com>" -m t` after a change) and run
`worktree handoff s0001 --expect handoff-refused-attribution`. The whole P8 Proof:
`go test -v -run 'TestProofWorktree|TestWorktreeRefusals|TestAskpass' ./internal/cli/`. The
`pre-push` refusal: `git -C <root>/sessions/s0001/tree push origin HEAD` from a terminal inside
the tree, then `worktree check s0001`; the P15 Proof is `go test -v -run
'TestProofPrePush|TestGitPrePush|TestPrePushHook' ./internal/cli/`. The push scan: commit a line
with a home path in the tree and run `worktree check s0001`; the P16 Proof is `go test -v -run
TestProofPushScan ./internal/cli/` and `go test -run 'Privacy|TrailerBlock|ParseLog'
./internal/git/`. A pull
request cannot complete offline: `--dry-run` is the reproduction; `go test ./internal/git/` posts
to a local test server.

Isolation. The scratch remote is a local bare repository with `handoff` `push`; the bot's git
never opens the session tree (ADR 0022) and runs with the inherited `GIT_*` repository
variables stripped; `worktree check` reads `refs/sessions/<id>` after its own fetch. Tests make
their repositories under the test's temporary directory and commit through `os/exec` with the
four identity variables (`internal/git/gittest`).

## The fetch at session start

ADR 0022 companion, Clone at session start, step 0; P30 of `08-implementation-plan-canonical.md`.
Drive. `session start` first fetches the project's `repo.remote`'s `repo.defaultBranch` into the
canonical's remote-tracking ref (`fetch --no-tags --no-write-fetch-head`, as the handoff's, a
thirty-second deadline), with the project's git credential as the push carries it, before the
workflow copy and the clone; nothing else is fetched and the canonical is never reset. Then, by
`merge-base --is-ancestor` both ways: the local branch behind is fast-forwarded (`merge --ff-only`
while it is checked out, `update-ref` otherwise); equal is nothing; the local branch ahead (it
contains the remote's commit) is left as it is and the session starts from it, the
`session-started` details carrying `canonical` `ahead <n>`, no venue line; diverged (neither
contains the other) is `error: canonical-diverged: <canonical> <branch> is at <sha>, <remote> at
<sha>; neither contains the other`, exit 1, audited `action-refused` with reason `canonical`, the
row ended `refused: canonical-diverged`, nothing cloned. The owner reconciles a diverged canonical
by pushing its local commits, or by `git -C <canonical> branch -f <branch> <remote>/<branch>`
(`reset --keep <remote>/<branch>` while it is checked out, as setup leaves it); a refused start from Discord posts the
`reset --keep` form to the notice channel (`04-ux.md`, Failure flows; `session outbox --channel <notice id>` in a dry
run; `TestDivergedNoticeFixRuns` runs the printed command on a checked-out canonical). A fetch that fails or runs past its
deadline (offline, a credential refused, a `repo.gitCredentialId` the store lacks, which skips the
fetch with the reason `credential-missing`, a fast-forward refused by a dirty checkout) starts the
session from the local branch: the details carry `canonical` `stale: <git's line or the reason>`
(`fetched` otherwise) and the venue's first post gains 04's `canonical-stale` line (wording pending
the owner).
Observe. `status --json`'s `projects` (`branch`, `commit`, `fetched`, `ahead`, `error` from the last
fetch, per project); `audit --kind session-started --json`; `git -C <root>/projects/scratch log
--oneline -1 main` beside the remote's.
Reproduce. `scratch run -- bash tests/repro/stale-canonical.sh` [no-check: written by P30]: a
commit pushed to `<root>/remote/scratch.git` from a throwaway clone, then a start on it; an empty
commit in the canonical for an `ahead 1` start; the remote renamed away for a stale start; a commit
pushed to the remote from the clone while the canonical keeps its own for a diverged refusal.
Isolation. The bot's git runs in the canonical with `-c credential.helper=` and the askpass helper
alone; a scratch remote is a local path and needs no credential.

## Landing

ADR 0028 and its companion (`decisions/0028-land-refusals-and-push-forms.md`); `03-architecture-git.md`
(Landing); P33 to P37 of `08-implementation-plan-landing.md`.
Drive. `session land <id> [--dry-run] [--expect <value>] [--as <user id>] [--roles <ids>]` runs
`Land` from the canonical (the fetch and fast-forward of the default branch as session start runs
them, `Refresh`, `Collect`, the attribution check, the push scan, then for a `main` project the gate
and `merge-base --is-ancestor <remote>/<default> refs/sessions/<id>`, the push of
`refs/sessions/<id>:refs/heads/<default>`, the canonical fast-forwarded) and prints `worktree
handoff`'s line, `<outcome>  <remote>  <refspec>  <commits>  <url>  <request>`, outcome `landed`
(a `main` push done), `pushed` or `pull-request` (a `push` or `pull-request` project: the session
branch, the request opened on the first land and followed by later ones), `none` (no commits since
the last land) or `dry-run` (everything but the push), then `stale: <git's line>` when the fetch
before it failed and `canonical lags: <why>` when the canonical's default branch could not follow
(its tracking ref did, and the next land's base is the landed sha). The session stays open and its tree is
untouched; the next land hands off the commits after the landed sha. `--as` and `--roles` make the
gate reproducible (`--roles` absent, the role is unchecked; the fact is always checked), but the
audit row's actor is `host` and the `--as` value is recorded as `claimed`, never as the actor; the
venue says `landed from the host`. Refusals, exit 1, nothing pushed, tree and session as they were:
`land-not-allowed` (the driver, or `--as`, has no role in the channel's `roles.land`, or the channel
has none), `land-unreviewed` (`review: none since <sha>`, plus `last launch: <commitError>` when the
newest reviewer row has one: commit, launch the reviewer as a call on its own, land again),
`land-not-fast-forward` (`<default> moved by <n> commits`: fetch, rebase in the tree, review again,
land again; the bot never rebases), `land-canonical-diverged` (the reconcile lift of The fetch at
session start), `land-rejected` (git's one telling line: a race is landed again; branch protection
or a checkout remote is the owner's, below), and the handoff's own `handoff-refused-*` and
`handoff-failed-*` codes. At a `main` session end the land is judged by the ender's `--roles` (none,
as on the idle end or `session end <id>` alone, is `land-not-allowed`); every land refusal but
`land-canonical-diverged` falls back to `handoffFallback`, and `session end` prints the fallback's
line with the first refusal met before it, `fallback: <code>: Not landed on <default>: <text>. Handed
off as <branch> instead.` (`--json`: a `fallback` object); `land-canonical-diverged` keeps the
session open, as a `handoff-refused-*` does (ADR 0028 Amendments 2026-11-02). A channel with no
`roles.land`, read as configured at each land, is `land-not-allowed` whoever asks.
The push word: inside the session a `Bash` call `git push` (or `git push <remote> [HEAD|<branch>][:<default>|:<branch>]`,
`broodforge session land`, each with `--dry-run` for the dry run; `-u`, `-q`, `-v` ignored) is
deferred by the hook's `land` cell and answered by the bot with the same land; the model reads
`Landed 3 commits on main at <sha>` as the command's stdout, or the refusal as a `Denied by
Broodforge: not landed: ...` deny. Any other option, remote or refspec is `land-bad-push` naming the
form that works (the companion's table); a push that reaches git (`cd x && git push`, `--no-verify`)
meets the tree's `pre-push`, whose line names the two forms. `config check` warns on a `main`
project whose channel has no `land` list and on a path remote that is a checkout with the default
branch checked out (`rev-parse --is-bare-repository` false, `symbolic-ref HEAD` the branch,
`receive.denyCurrentBranch` unset or `refuse`), naming the three lifts (a bare repository beside the
checkout, `updateInstead`, GitHub as the remote); the first land on such a remote repeats it.
Observe. `audit --kind land --json` (`attempted`, `done`, `failed`; `actor`, `claimed`, `outcome`,
`commits`, `sha`); `audit --kind subagent-result --json` (`model` on every row; `commit`, `dirty`,
`commitError` on the reviewer's row; `empty` on a row whose result has no text); `session show` (`review:`); `git -C <root>/remote/scratch.git log --oneline -1 main`;
`hook try --tool Bash --input '{"command":"git push --force"}' --expect defer`, then with `--answer
<id>:allow --answer-reason 'Landed 1 commit on main at <sha>'` printing the `updatedInput`.
Reproduce. `scratch run -- bash tests/repro/land.sh` (a `main` project with
`roles.land`: unreviewed, not allowed, landed, moved base, the end's fallback, the `config check`
warning) and `tests/repro/land-push-word.sh` [no-check: written by P35] (the `land-done` and
`land-refused` fixtures through the hook's resume). A rejected push: a non-bare scratch remote with
`main` checked out, `session land s0001 --expect land-rejected`.
Isolation. The scratch remote is a local bare repository; the land's git runs in the canonical with
`-c credential.helper=` and the askpass helper, never in the tree; the fake's fixtures carry the
deferred `Agent` and `Bash` calls, so no real engine is needed below the three owner-word turns the
passes name (V1, V11, V12).
