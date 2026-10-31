# Implementation plan: landing a session on the default branch

This file continues `08-implementation-plan.md` under the same conventions: the pass format, the size
words, the standing rules and the `[no-check: ...]` rule. It holds the passes that build ADR 0028
(`decisions/0028-landing-a-session-on-the-default-branch.md` and its companion
`decisions/0028-land-refusals-and-push-forms.md`), in the order the proposal fixed: the `reviewer`
cell and the pin (P33), `Land` with `session land` and the `config check` warnings (P34), the `land`
cell with the command parse and the resume (P35), the venue command (P36), the subagent probe (P37).
Each pass reads the ADR's Decision, the ADR 0022 companion (Handoff at session end, Amendments
2026-10-31), `03-architecture-git.md`, `03-hook-contract.md` and `05-debug-interface-git.md` before
its Implements line. The bot's git runs in the canonical only and never opens a session's tree (ADR
0022); the tree never pushes (ADR 0025 rule 4 as amended); every text a person sees is the owner's and
ships as the companion's placeholder until the owner words it. Three claims close by fixture, each on
the owner's word for the one real turn it needs: V1 (P33), V11 (P35), V12 (P37). P38 (added 2026-10-31,
ADR 0025 rule 10 as amended that day) is outside ADR 0028 and lands first: it fixes a refusal the live
instance meets on every `runner` commit today, touches no landing code, and P34's land runs the same
attribution check.

## P38. Attribution owed from the bot's record; the model pin at an `Agent` launch (small; one real turn on the owner's word)

Lands before P33: the `subagent-result` row already carries `commit` and `commitError` from the
tool-use read (P31); this pass adds `model` to it, and P33 then moves `commit` to the defer and adds
`dirty`. Implements: ADR 0025 Amendments 2026-10-31 (rule 10); `03-architecture-git.md`
(Attribution); `05-debug-interface-git.md` (Required trailers; the two hooks); `03-hook-contract.md`
(How the engine runs it: the pin write beside `hook.log`, the one error the hook logs instead of
failing closed on). Files [no-check: the migration, tests and the fixture's new copy are created by
this pass]: `internal/store/migrations/0005-model-seen.sql` (`model_seen`: `session_id`, `model`,
`first_seen_at`, `via` `turn` | `pin` | `stream`, `tool_use_id`, `agent`; one row per session and
model, the first sighting kept; schema 5); `internal/session/turn.go` (the record and the rewrite: at
`startTurnLocked` the turn's model when set; at an `Agent` or `Task` `tool-use` block the pin,
resolved as the hook resolves it from the copy's rendering the session holds (`l.wf`), kept on the
`call` as `model` and written on its `subagent-result` row, `none` when nothing pins; at an
`assistant` event whose `Model` has no row; each an insert-if-absent at the bot's clock, then
`git.WriteTrailers`; `trailers()` reads the record and the usage table together in first-seen order,
participants first as today); `internal/hook/hook.go` (after the decision and the chain, when the
answer `pre-tool-use` returns is `allow` and the tool is `Agent` or `Task`, a deferred call's
resumed run included: the pin from `tool_input.model`, else `$BROODFORGE_WORKFLOW/agents.json`'s
`model` for `subagent_type`, through `policy.ResolveModel` and `ModelDisplayName`; the line
appended to the session's `trailers` file (the `home` directory beside `BROODFORGE_TREE`, where
`CommitMsg` reads it) when missing, by a temp-file-and-rename copy of its own, since
`03-architecture.md`'s table has `hook` importing `policy` alone and `git` pulls in `spawn`,
`credentials` and `audit` (the row stands); no pin, or a failed write, is one `hook.log` line and the
decision stands, the one error the hook logs instead of failing closed on: a help never blocks);
`internal/hook/cases.go` (`Try.Run` resolves the same pin from its `Tree` and `Workflow` fields and
returns it; `try` writes nothing and logs nothing, as today); `internal/cli/hook.go` (`hook try`
gains a `pin` column, the line it would append or empty); `internal/cli/worktree.go` (`Since` = `model_seen.first_seen_at`, else the first turn's
`started_at`, else `recorded_at`); `internal/git/git_test.go` (`TestCheckOwedFromFirstSeen`: two
commits a minute apart, a model whose Since lies between them owed on the second only; the trailers
file rewritten between the commits by `WriteTrailers` and read by `CommitMsg` gives each commit its
own set); `internal/hook/hook_test.go` and `tests/hook/pin/` (a copy directory whose `agents.json`
pins `runner` to `haiku` and a `home/trailers` holding one Sonnet line: an `Agent` `runner` call
leaves both lines; `tool_input.model` `opus` outranks the type; `general-purpose` and a type without
`model` write nothing and log it; a `deny` writes nothing; no `BROODFORGE_WORKFLOW` with
`tool_input.model` set still writes); `tests/hook/cases.jsonl` (the `Agent` rows unchanged: the pin
changes no decision); `fixtures/engine/subagent-pinned/` (a copy of the hand-written `subagent`
stream, which is all `claude-opus-5-5` with no `model` on the `Agent` input and so yields no distinct
row: the `Agent` input gains `"model":"haiku"`, the lines carrying `parent_tool_use_id` say
`claude-haiku-5-5`, the result's `modelUsage` names both models) and its README row;
`internal/session/attribution_test.go` (a session started with model `opus` replaying
`subagent-pinned`: a `turn` row for Opus at the turn's start, a `pin` row for Haiku stamped at the
`Agent` line and the `subagent-result` row naming it, the trailers file holding both lines; a
commit stamped before the pin is not owed Haiku, one after is; and under `workflow_test.go`'s copy,
whose `agents.json` pins a type, an `Agent` call without `model` pins from the copy);
`internal/cli/worktree_test.go` (a `model_seen` row later than the turn's start lets an earlier
commit pass `worktree check`). Debug surface: `audit --kind subagent-result --json` (`model`);
`worktree check <id>` (the per-commit line); `hook try --tool Agent --input
'{"subagent_type":"runner"}' --tree <tree> --workflow <copy>` (`try` reads no environment: the
flags stand for `BROODFORGE_TREE` and `BROODFORGE_WORKFLOW`; its `pin` column shows the line; the
file itself is written only by `pre-tool-use`, checked in the hook test); `session show` unchanged. Proof: `go test ./internal/git ./internal/hook
./internal/session ./internal/cli -run 'Trailer|Pin|FirstSeen|Attribution'`; `hook cases
tests/hook/cases.jsonl` all `ok`; `tests/repro/full-session.sh` unchanged and passing on schema 5;
then, on the owner's word, one real turn on the live instance: the owner asks a session to commit,
the `runner` commits, `audit --kind subagent-result --json` shows `model` `Claude Haiku 5.5`, the
commit's trailer block carries the Haiku line, and `worktree check <id>` prints that commit `ok`;
recorded as a finding. Does not: change any hook decision or cell; defer any launch; drop a
reviewer-only model from later commits (the owner's remaining choice; `model_seen` keeps one row per
model, so that exemption needs one row per model and launching agent type, or per-call usage, when
answered); change the push scan; rewrite history. Depends on: P31 (the
`subagent-result` row); none of P33 to P37.

## P33. The `reviewer` cell and the pin at the defer (small; one real turn on the owner's word)

Implements: ADR 0028 Decision 3; ADR 0025 Amendments 2026-10-31 (rule 7); `03-hook-contract.md` (the
`reviewer` cell; Decision); `05-debug-interface-session.md` (`session show`'s `review:` line). Files
[no-check: fixtures and cases created by this pass]: `internal/policy/` (the `reviewer` cell: an
`Agent`, or `Task` on an older pin, whose `tool_input.subagent_type` is `reviewer` is `defer` in both
modes and both reaches where `Agent` is allowed; the cell is one object in the table, never
configuration); `internal/hook/` (the cell applied after the path checks and before the answer; the
child chain still runs for the call, as for any non-`deny`; a resume with `BROODFORGE_ANSWER`
`<id>:allow` returns `allow` as today); `internal/session/turn.go` (on `deferred_tool_use` whose call
the `reviewer` cell deferred: `Refresh` under the session's bundle lock, the sha kept on the call as
`commit`, the bundle's `status --porcelain` kept as `dirty` when a tracked file is modified or
deleted (untracked lines ignored), a failure as `commitError`, `dirty tree at the reviewer's launch`
as the `commitError` for a dirty pin; then the resume with `allow` at once, no venue prompt, no
`waiting-for-answer` event; the pin on the `tool-use` read goes, so a reviewer call that arrives with
its result and no defer gets no `commit`); `internal/spawn/worktree.go` (`WorktreeBundle` takes a
context and returns the porcelain status beside the bundle path, so the pin cannot hang the resume);
`internal/session/workflow.go` (`ReviewFact`: the newest `subagent-result` row with `agent`
`reviewer`, `result` `ok`, a non-empty result text, no `commitError`, whose `commit` equals the ref's
sha); `internal/cli/session.go` (`review: none since <commit>; last launch: <commitError>` when the
newest reviewer row carries one); `fixtures/engine/reviewer-defer/` (turn 1 ends `tool_deferred` on
an `Agent` call with `subagent_type` `reviewer`; turn 2 is the resume: the subagent's blocks, its
non-empty result; a sibling `reviewer-empty` whose result text is empty) and its README lines;
`tests/hook/cases.jsonl` (the cell in both modes; a `subagent_type` other than `reviewer` stays
`allow`); `internal/session/review_test.go` (the deferred pin counts; an empty result does not; a dirty
pin does not; a non-deferred reviewer call does not; the P31 cases unchanged). Debug surface:
`session show <id>`, `audit --kind subagent-result --json` (`commit`, `dirty`, `commitError`), `audit
--kind permission-decision` (the reviewer call's `deferred` row answered by `instance`), `hook try
--tool Agent --input '{"subagent_type":"reviewer"}' --expect defer`. Proof: `go test ./internal/policy
./internal/hook ./internal/session -run 'Review|Reviewer'`; `hook cases tests/hook/cases.jsonl` all
`ok`; on a scratch root with `--fixture reviewer-defer` a turn prints `review: <tool use id> at
<base>` and with `reviewer-empty` `review: none since <base>`; `tests/repro/full-session.sh` unchanged
and passing; then, on the owner's word, one real turn on the dev root (V1): the engine defers the
`Agent` call, the resume launches the subagent, `session show` prints `review: <id> at <commit>`;
recorded as a finding. If the real engine does not defer an `Agent` call, the pass stops at the
finding and the ADR's first reversal condition is the architect's. Does not: the land; the `land`
cell; change the chain for `Agent`; refresh for any subagent but `reviewer`. Depends on: P31.

## P34. `Land`, `handoff: main`, `roles.land`, `session land`, the `config check` warnings (medium)

Implements: ADR 0028 Decisions 1, 2 and 5 (the host door); ADR 0022 companion Amendments 2026-10-31;
`03-architecture.md` (Configuration; Failure modes, the two `git` rows); `03-architecture-git.md`
(Landing); `05-debug-interface-git.md` (Landing); the 0028 companion (Refusals and lifts, the codes;
`config check` warnings). Files [no-check: `tests/repro/land.sh` is created by this pass]:
`internal/config/types.go` (`handoff` accepts `main`; `handoffFallback` `push` | `pull-request`,
default `push`, an error with any other `handoff`; `roles.land` beside the three lists;
`config_test.go` keeps `"merge"` as the bad value); `internal/config/` check (the two warnings of the
companion: a `main` project's channel without `land`; a path remote that is a non-bare repository
with the default branch checked out and `receive.denyCurrentBranch` unset or `refuse`, read with
`git -C <remote> rev-parse --is-bare-repository`, `symbolic-ref -q HEAD`, `config
receive.denyCurrentBranch`); `internal/git/handoff.go` (`Land(ctx, b, s, LandOptions{DryRun, Gate})
(LandResult, error)`: the ADR's five steps, `Collect`, the scan and the push shared with `Run`; the
gate a closure the service supplies (role, fact) so `git` reads no store; `land-canonical-diverged`,
`land-not-fast-forward` with the count, `land-rejected` with git's telling line; step 5 through
`Fetch`'s fast-forward path, the refused case updating the tracking ref and `Base` read against
`<remote>/<default>` where ahead, `LandResult.CanonicalLags`; `Run` on a `main` project calls `Land`
and on `land-not-fast-forward` or `land-rejected` falls back to `handoffFallback`, the result
carrying both outcomes; a first land on a path remote repeats the `config check` inspection into the
outcome line); `internal/git/create.go` (`FastForward(ref, sha)` shared by `Fetch` and step 5);
`internal/policy/` (the role check over a channel's `land` list); `internal/session/service.go`
(`Land(sessionID, actor Actor, roles []string, dryRun bool)` on `SessionService`: the driver's roles
judged against `roles.land`, `land-not-allowed`; the fact from `ReviewFact`, `land-unreviewed` with
the newest row's `commitError`; one land at a time per session through the lock P33 and `worktree
check` share, a second request waiting; audit `land` with `attempted`, `done` or `failed`, `actor`
(`host` or the Discord user id), `claimed` (the `--as` value, when any), `outcome`, `commits`, `sha`;
the end on a `main` project runs `Land` then the removal, the fallback outcome in the summary);
`internal/cli/session.go` (`session land <id> [--dry-run] [--expect <value>] [--as <user id>] [--roles
<ids>]`: the line `<outcome>  <remote>  <refspec>  <commits>  <url>  <request>`, outcome `landed`,
`pushed`, `pull-request`, `none` or `dry-run`; every `land-*` code an `error:` line, exit 1, the
handoff's codes as today; the actor always `host`); `internal/cli/config.go` (the warnings as
`warning:` lines, exit 0); `tests/repro/land.sh` (a scratch root whose project is `handoff` `main`,
`handoffFallback` `push`, channel `100` with `roles.land` `10`: a start with `--fixture
reviewer-defer`, a commit in the tree through `os/exec` as `full-session.sh` commits, `session land
s0001 --expect land-unreviewed`, a turn (the reviewer pin), `session land s0001 --as 1001 --roles 11
--expect land-not-allowed`, `session land s0001 --as 1001 --roles 10` printing `landed` and `git -C
<root>/remote/scratch.git rev-parse main` equal to the tip, `audit --kind land --json` holding
`claimed` `1001` and `actor` `host`; a commit pushed to the remote's `main` from a throwaway clone,
a second tree commit and turn, `session land s0001 --expect land-not-fast-forward`; `session end
s0001` printing `pushed` with the fallback; `config check` on a copy of the root whose channel lost
`roles.land` printing the warning) and its `tests/README.md` row; `internal/git/git_test.go` (`Land`
against a bare remote: a fast-forward lands and the canonical follows; a moved `main` is the refusal
with its count; a non-bare remote with `main` checked out is `land-rejected` with git's
`denyCurrentBranch` line; a dirty canonical after a done push: tracking ref updated, `CanonicalLags`,
the next land's base the landed sha; `--dry-run` pushes nothing and prints the line). Debug surface:
`session land`, `audit --kind land --json`, `config check`, `session show` (`review:`), `git -C
<root>/remote/scratch.git log --oneline -1 main`. Proof: `go test ./internal/git -run TestLand`; `go
test ./internal/config ./internal/session -run 'Land|Handoff'`; `scratch run -- bash
tests/repro/land.sh` exit 0; `full-session.sh`, `stale-canonical.sh` and `restart-end.sh` unchanged
and passing. Does not: the `land` cell or the push word; the venue command or any venue text (P36);
a rebase or replay; a closing turn for a mid-session `pull-request` land (waits on the owner; the
fallback title); change `worktree handoff`'s output for `push` and `pull-request` projects. Depends
on: P30, P31, P33.

## P35. The `land` cell, the command parse and the resume (medium; one real turn on the owner's word)

Implements: ADR 0028 Decision 4 and Consequences (the `updatedInput` fallback); the 0028 companion
(The push forms; `pre-push`); ADR 0025 Amendments 2026-10-31 (rule 4); `03-hook-contract.md` (the
`land` cell, `updatedInput`, `BROODFORGE_ANSWER_REASON`, The child chain); `05-debug-interface-git.md`
(Landing: the push word); `05-debug-interface-session.md` (Prompt-mode defer and resume). Files
[no-check: fixtures, cases and the repro created by this pass]: `internal/policy/` (the `land` cell: a
`Bash` whose command's first words after the shell's word splitting are `git push` or `broodforge
session land` is `defer` in both modes in code reach, `deny` in read-only reach, judged before the
`ask` cell and returned before the child chain; `policy` keeps to first words); `internal/hook/` (the
`land` cell skips the chain on the defer and on the resume; a resume whose `BROODFORGE_ANSWER` is
`<id>:allow` for a call the `land` cell matches returns `allow` with `updatedInput` `{"command":
"printf '%s\n' '<line>'"}`, `<line>` the value of `BROODFORGE_ANSWER_REASON` single-quoted for `sh`
with every `'` as `'\''`, the other `tool_input` fields kept; `<id>:deny` returns `deny` with the
reason as today); `internal/git/landrequest.go` (`ParseLandRequest(command, remote, defaultBranch,
sessionBranch string) (LandRequest, *Error)`: the companion's table row by row, `LandRequest{DryRun}`
or `land-bad-push` with the form that works); `internal/session/turn.go` (on `deferred_tool_use` of a
`land` cell call: the request parsed; a call with an `agent_type` is `land-not-allowed` (subagent) until
P37; else `SessionService.Land` as the driver with the driver's roles; a done land resumes `allow` with
the outcome line in `BROODFORGE_ANSWER_REASON`, a refusal resumes `deny` with `Denied by Broodforge:
not landed: <text>; <lift>`; the `permission-decision` row decider `instance`, its `reason` the
outcome line or the refusal; no `waiting-for-answer`, no venue prompt; a request while a person's
answer is pending waits behind it); `internal/cli/hook.go` (`hook try` gains `--answer <id>:allow|deny`
and `--answer-reason <text>` and prints the `updatedInput` object when one is returned);
`share/hooks/pre-push` and `cli.PrePushRefusal` (the companion's new text as the placeholder);
`fixtures/engine/land-done/` and `land-refused/` (turn 1 ends `tool_deferred` on a `Bash` `git push`;
turn 2 the resume: `land-done`'s tool result is the printed line with no `is_error`, `land-refused`'s
the deny) and their README lines; `tests/hook/cases.jsonl` (one row per companion table row, both
modes, both reaches; `cd x && git push` is `ask`, so `pre-push` meets it); `tests/repro/land-push-word.sh`
(on `land.sh`'s root: a turn with `--fixture land-done` after the reviewer pin lands and `session tail`
shows the `tool-result` text `Landed 1 commit on main at <sha>` with no error, `audit --kind land`
`done`, the remote's `main` at the tip; `--fixture land-refused` before any pin: the decision row's
reason begins `Denied by Broodforge: not landed: review: none since`) and its `tests/README.md` row;
`internal/git/landrequest_test.go` (every table row). Debug surface: `hook try --tool Bash --input
'{"command":"git push --force"}' --expect defer`, then with `--answer <id>:deny --answer-reason ...`;
`session tail --json` (the resumed call's `tool-result`); `audit --kind land --json`. Proof: `go test
./internal/hook ./internal/git ./internal/session -run 'Land'`; `hook cases tests/hook/cases.jsonl` all
`ok`; `scratch run -- bash tests/repro/land-push-word.sh` exit 0; `defer-resume.sh` unchanged and
passing; then, on the owner's word, one real turn on the dev root (V11): the model is asked to commit
and push, the stream shows the `Bash` call deferred, the resume's tool result holding the outcome line
without `is_error`, and the model's text reporting the land as done; recorded as a finding and the
stream kept as a fixture. If the pinned engine ignores `updatedInput` on the resume, the pass ships the
ADR's fallback instead: a done land resumes `deny` with a reason starting `Landed`, and 03's
classifier gains that prefix as a done land, never an `action refused`. Does not: accept a subagent's
request (P37); the venue command; change the chain for any other `Bash` command; change the `ask`
cell. Depends on: P34.

## P36. The venue: `/session land`, the land lines, the fallback line (small; owner sees everything)

Implements: ADR 0028 Decision 1 (the venue door) and Waits on the owner (the command's placement);
the 0028 companion (Venue lines); `04-ux.md` (Landing); `05-debug-interface-discord.md` and
`05-debug-interface-discord-live.md` (the `/session` subcommands, the acknowledgement pattern). Files [no-check: the
inject step is added by this pass]: `internal/discord/` (the `land` subcommand under `/session`; the
caller's roles read at the moment of the command against the channel's `roles.land` (N-18), a miss
the companion's `land-not-allowed` text, ephemeral, `action refused`; the deferred reply, then the
follow-up with the outcome line or the refusal, every text the companion's placeholder until the owner
words it; the end's summary gains the `Landed` or fallback line; `discord inject message|click` feeds
the command in a dry run); `internal/cli/` (the dry-run outbox shows the posts, nothing new); the
registered command list (`setup`'s Discord step and the live gateway register `land` beside the other
subcommands; `05-debug-interface-discord-live.md` lists it); `tests/repro/land.sh` gains an inject of
`/session land` by a user with and without the role, the outbox checked for both lines. Debug
surface: `session outbox <id>`, `discord inject`, `audit --kind land --json` (`actor` the user id).
Proof: `scratch run -- bash tests/repro/land.sh` exit 0 with the inject steps; `discord render --golden`
unchanged where P32 exists; then the owner's live run on the development instance: `/session land` in
a session venue, the venue line, the remote's `main` moved. Does not: the header button (the owner's
placement call; the slash command alone until then); decide any wording; a closing turn for a
mid-session `pull-request` land. Depends on: P34, P12.

## P37. The subagent probe and `git.push.subagents` (small; one real turn on the owner's word)

Implements: ADR 0028 Decision 5 and Deferred (L-2's subagent list behind V12). Files [no-check: the
fixture and the finding are created by this pass]: `fixtures/engine/land-subagent/` (the recorded real
stream of a `pusher` subagent issuing `git push` under the bot's hook on the dev root, whatever it
shows) and its README line; `docs/findings/<date>-a-subagents-defer-under-the-bots-hook.md` (whether
the deferred call ends the whole process with `tool_deferred`, whether `--resume` re-fires the hook
inside the subagent for the same id, whether the subagent survives the resume and returns; status
`resolved` either way); then, only when all three held: `internal/workflow/` (the copy's
`.claude/hooks/policy.json` `git.push.subagents` read as data into the copy's rendering, as the
`privacy` section is); `internal/session/turn.go` (a `land` cell request whose `agent_type` is on
that list is accepted on the driver's behalf, judged by the driver's roles, the audit row naming
both; no policy file ever allows the driver process alone); the header sentence (the push word is a
call on its own; subagents on the list may issue it); `internal/session/review_test.go` or a
`land_test.go` case replaying the fixture. Debug surface: `engine run --fake fixtures/engine/land-subagent`,
`audit --kind land --json` (`agent_type` beside `actor`), `session show --header`. Proof: the fixture
replays through `engine run --fake`; the finding records the three answers; when they held, `go test
./internal/session -run LandSubagent` and `tests/repro/land-push-word.sh` extended with a subagent
fixture; when any failed, the pass ends at the finding, the header stands, and the ADR's Deferred item
keeps its trigger. Does not: widen the list beyond the copy's file; accept a subagent before the probe;
change the chain. Depends on: P35; a real turn on the owner's word.
