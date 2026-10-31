# 0028. Landing a session's work on the default branch: `/session land`, `handoff: main`, and the review gate

Date: 2026-10-31
Status: under review

## Question

A Discord session's work reaches the remote only when the session ends (`internal/git/handoff.go`,
`Run`), and only as a session branch or a pull request. Under "check in rarely; lose nothing"
(`../00-vision.md`, 2026-10-23) a session lives for days, so finished work waits on an end nobody
wants yet, and the owner's own projects want no branch to merge by hand. Three things are decided
here as one design: a land mid-session (`/session land`, idea of 2026-10-23), a `"handoff": "main"`
at session end with a fast-forward only, a fallback and a role gate (idea of 2026-10-23, the owner
agreed to that shape), and the review gate: how the bot knows an independent review of the commits
happened before it lets them onto the default branch. The owner, 2026-10-31: "I want discord to be
able to push directly though, or (setting) send pull requests. Which indeed means another review
phase and sub agent and the push word working mid-session." And 2026-10-24: "a push by the
session's own runner, after the session's own independent review, should be an acceptable handoff,
so the session's agents can land work too, not only the driver." In ten years this should read as
one verb, land, whose gate is a fact the bot measured and whose every refusal names its lift.

## Constraints that matter here

- Both hosts (`../06-platforms.md`); GPL (ADR 0001); every feature drivable from a terminal;
  agents write all code; nothing per project in the bot's code (a project is configuration).
- Already decided, not reopened: never a force push; the Go push scan (ADR 0025 rule 7) runs on
  every push; attribution trailers per model with usage (03, Attribution); one independent review
  per change before it is pushed (`one-review.md`); the fallback to the session branch on a conflict
  or a rejected push; a role-group gate for writing the default branch.
- The bot never opens a session's tree and never rewrites history (ADR 0022 and its companion); the
  session never holds the git credential; on the droplet the canonical is bound read-only and
  `data/` is unbound, so nothing in a session can reach the bot or the remote (ADR 0022).
- The hook has no channel to the bot (`../03-hook-contract.md`); the bot reads the stream, and a
  `defer` ends the turn until the bot resumes it with `BROODFORGE_ANSWER` and `BROODFORGE_ANSWER_REASON`.
- The review fact as built by P31: the bot bundles the tree when it reads the `reviewer` `Agent`
  call's `tool-use` block and writes the sha into that call's `subagent-result` row; the fact holds
  when the newest `ok` row names the sha `refs/sessions/<id>` holds after the last bundle (ADR 0025,
  amendment of 2026-10-30; live on 2026-10-31, `../../findings/2026-10-31-milestone-1-reached-a-channel-200-session-committed-was-reviewed-and-landed.md`).
- The PC has no boundary between a session and the bot (ADR 0025 rule 8): there, every gate below
  is a refusal a session could lift, until ADR 0022's W1 or WSL2.
- Venue texts are the owner's; what spends the owner's money is the owner's (`major-decisions.md`).

## The land, common to every option

One procedure, run by the bot from the canonical, that the session end and the mid-session command
both call. "Land" means: run the project's handoff now. On a `push` project it pushes the session
branch; on a `pull-request` project it pushes the branch and opens the request on the first land
(later lands fast-forward the same branch, so the request follows); on a `main` project it pushes
the default branch. Only the last needs the review gate; the first two are the review venue
themselves and need none (ADR 0025 rule 7 as it stands).

1. Fetch the remote's default branch into the canonical and fast-forward the local branch, as the
   session start does (ADR 0022 companion, Clone step 0; a failed fetch is noted, not a refusal:
   the remote's answer to the push is the truth).
2. `Refresh` and `Collect` as today (bundle, verify, forced fetch into `refs/sessions/<id>`, the
   attribution check), then the push scan. A refusal keeps everything as it was.
3. For a `main` target: the gate (the driver's land role; the review fact naming the tip exactly,
   by the option chosen below), then `merge-base --is-ancestor <remote>/<default> refs/sessions/<id>`.
   False means the default branch moved under the session: the land refuses `land-not-fast-forward`
   with the count, because pushing the exact reviewed sha is the whole point and the bot does not
   rebase (that would write commits nobody reviewed and void the fact by construction). The lift is
   the session's: fetch `origin` (the canonical, now current after step 1), rebase in the tree,
   review again, land again. At session end this refusal, and a rejected push, fall back to
   `handoffFallback` (`push` by default, `pull-request` where the project says so) and the venue
   says why; mid-session there is no fallback, the session simply goes on.
4. `git -c credential.helper= push <remote> refs/sessions/<id>:refs/heads/<default>` with the
   askpass helper, never `--force`; audit `land` `attempted`, `done` or `failed`. A rejection
   (someone pushed between steps 1 and 4, branch protection) is `land-rejected`.
5. After a done `main` push: fast-forward the canonical's local default branch to the landed sha
   (`merge --ff-only` while checked out, else `update-ref`, as `Fetch` does) and the tracking ref.
   Nothing in the tree changes, so the engine's next `--resume` sees the files it left.
   `refs/sessions/<id>` stays; `Base` (the merge base with the default branch) now equals the landed
   sha, so the next land and the end hand off only what came after, with no new column.

Mid-session the tree is kept and the session stays open; at the end step 5 is followed by the
removal (R-35) as today. One land at a time per session: the land, `worktree check` and the
reviewer-launch bundle share a lock, since all three write `<sess>/out.bundle`. A running turn does
not block a land from the venue or the CLI: only committed, reviewed commits move, and the tree is
untouched; the agent's later commits build on them.

## Options: the review evidence (for a `main` target)

### R-a. P31's review fact, covering the exact sha pushed

The newest `subagent-result` row of a `reviewer` call with result `ok` names the sha the bot
bundled at the call's launch, and that sha equals `refs/sessions/<id>` after the land's own refresh.
Two tightenings: the bundle moves from the `tool-use` read to the hook's `PreToolUse` decision for
the reviewer call, so the sha is pinned before the reviewer can start (the race the idea of
2026-10-30 found: the stream read lags the hook by seconds, and a commit between the reviewer's read
and the bundle counted as reviewed); and the row counts only with a non-empty result text (s0005's
reviewer returned nothing and counted). What it proves: a reviewer agent, launched in this session,
read this exact tree and returned. What it does not prove: that the reviewer approved, or that the
driver's brief did not say "approve"; the bot interprets no project text (ADR 0025 rule 2), so the
verdict stays the driver's to act on. Staleness: an amend or a commit after the review moves the
ref at the next bundle and the fact reads `none since <new sha>`; the land always bundles first, so
no clock enters. Cost: nothing new to spend; P31 is built. Moving parts: the bundle in the hook
decision (a context on `spawn.WorktreeBundle` so the decision cannot hang; the hook returns after
the bundle, inside its thirty seconds); the lock above. Failure modes: a hung bundle delays the
reviewer's launch (bounded; a failed bundle leaves `commitError` and never counts); a review
followed by a fix amends the commit and voids the fact, so a second review of the amended sha is
the price of a land, as the 2026-10-30 amendment already accepted. Exit cost: none; the row stays
useful under every other option.

### R-b. A trailer the reviewer leaves on the commit

`Reviewed-By: <model> <noreply@anthropic.com>` (or a `Broodforge-Reviewed: <tool use id>` line)
appended to the reviewed commit, checked by the land as the attribution check checks trailers. Moving
parts: the reviewer, or the `runner` after its report, amends the commit (the base `git-guard` lets
only the `runner` commit; the amend itself is the base's accepted use). Failure modes: the trailer is
text the session controls, so it proves nothing the bot did not see; adding it changes the sha, so
the reviewed sha is never the pushed sha; worst, an amend keeps the message, so the trailer survives
a change to the diff: it is the one form of evidence that does not go stale. As a public record in
`git log` it has value, and a project may require it in its own `commit-msg` rules; as the gate it is
weaker than R-a on every axis. Exit cost: a trailer in history is harmless to leave behind.

### R-c. An audit record of a reviewer launch the bot observed

The `subagent-result` row without the sha: a `reviewer` call that returned `ok` after the session's
last commit. This is the rule 7 text of 2026-10-23, replaced on 2026-10-30 because the committer
time is the session's to write (`GIT_COMMITTER_DATE`, `--amend --date`), so "after" proves nothing;
bounding the time by the previous bundle narrows the window but leaves it open inside the turn. With
the sha attached it is R-a. Staleness: undetectable within a turn. Moving parts: none beyond P31's
row. Kept as the record of why the fact carries a sha; not a candidate.

### R-d. The land runs the review itself

The bot pins the sha, checks out `refs/sessions/<id>` into a bot-owned read-only worktree in the
canonical, runs one engine process of its own with the `reviewer` definition from the session's
workflow copy (a default reviewer prompt the bot ships when the project has none; the project's file
overrides it, nothing per project in code), asks for a verdict in a fixed shape (`--output-format
json`, one `blocking: true|false` field and the findings), and refuses the land on a blocking
finding, posting the findings to the venue. What it proves: an agent that did not write the change
and was not briefed by its writer read exactly the sha pushed and found nothing blocking: the
strongest reading of `one-review.md`, with a happens-before that needs no hook change. Staleness
cannot arise: pin, review, push is one sequence, and a commit in the tree meanwhile is simply not in
the range. Moving parts: a second spawn kind (the bot's reviewer, not a session's; its own settings
file, read-only reach of that worktree, no `BROODFORGE_SESSION`), a credential to run it on (whose:
the driver's, or an owner key), a verdict format the bot parses, a timeout, a venue post. Failure
modes: it spends money on every land, the owner's call; a session that already ran its reviewer pays
twice unless R-d replaces the in-session review for `main` projects (one review, not two); the
verdict is model output and a lenient reviewer passes what a strict one would not; the worktree is
one more thing in the canonical to clean. Exit cost: the spawn kind and verdict parser are deleted;
no data migrates. Where it earns its cost: a channel whose land role is wider than the owner's
trusted circle, where R-a's "the driver briefed the reviewer" is the weak link.

## Options: the push word mid-session

### W-i. The tree's `pre-push` hook lets a push the bot marked reviewed through

The hook would need the fact (no channel to the bot; `data/` is unbound), the credential (the
session never has it, ADR 0022), and a writable remote (the canonical is read-only on the droplet).
A push from the tree cannot be scanned by rule 7's Go scan either, which runs in the bot. On the PC
it works only through the same-account gap. Moving parts: a mark file the bot writes into the tree's
`home` (as `trailers` is), the credential in the sandbox. Failure modes: the whole 0022 boundary.
Exit cost: none. Rejected as a mechanism on both hosts; `pre-push` stays a refusal, its text changed
to say how to land (owner's wording).

### W-ii. The tree never pushes; the request reaches the bot and the land runs from the canonical

Three doors, one land. From the venue, `/session land` (the driver, by the land role); from the
host, `session land <id> [--dry-run] [--expect <value>]`, printing `<outcome>  <remote>  <refspec>
<commits>  <url>  <request>` as `worktree handoff` does, outcome `landed`, `pushed`, `pull-request`,
`none` or `dry-run`, refusals `land-not-allowed`, `land-unreviewed` (with `review: none since
<sha>`), `land-not-fast-forward`, `land-rejected`, and the handoff's own codes; and from inside the
session, the push word: a `Bash` call whose command begins `git push` or `broodforge session land`
gets a new policy cell, `land`, which the hook answers `defer` in both modes. The bot, reading the
deferred call, runs the land and resumes the turn with `BROODFORGE_ANSWER` `deny` and the land's
outcome line as `BROODFORGE_ANSWER_REASON`, so the model reads `Landed 3 commits on main at <sha>`
or `Not landed: review: none since <sha>; launch the reviewer, then push again` as the call's
result. The control socket is not involved (unreachable from a droplet session by design); the hook
parses no git beyond the command's first words, so a `cd x && git push` meets `pre-push`'s refusal,
whose text names the two forms. Moving parts: the `land` cell and its two prefixes in `policy`; the
answer path reused with the bot as the answerer; the land in `internal/git`; the lock. Failure
modes: a defer ends the turn and the resume is a new process, so a land costs the prompt-mode round
trip (seconds, not tokens); a land request while a person's answer is pending queues behind it. Exit
cost: the cell is deleted and the prefixes fall to `ask`.

### W-iii. Both

W-i adds nothing W-ii lacks and reopens the boundary. Rejected with W-i.

## Options: who may land on the default branch

### L-1. The driver only

`/session land` and the end; the agent's `git push` is refused. Simplest; contradicts the owner's
2026-10-24 line. Failure mode: the driver becomes the bottleneck the vision's rhythm removes.

### L-2. The driver, or the session's engine where the project's workflow allows

The in-session request is accepted on the driver's behalf when the call came from the driver process
(no `agent_type`) or from a subagent the copy's `.claude/hooks/policy.json` `git.push.subagents`
names (read as data, as the privacy section is; the base names `pusher`); a project with no policy
file allows the driver process alone. Every land is still judged by the driver's roles: the engine
lands as the driver, never as itself, and the audit row names both.

### L-3. The role group

The channel's `roles` gains `land` beside `start`, `contribute` and `answer` (a group name from
`roleGroups` or raw ids; absent means nobody). It is the channel's, not the project's, so a project
with a trusted channel and a wider one differs per channel while `handoff` stays the project's. A
`main` project with a channel that has no `land` list gets a `config check` warning: every session
there would fall back. L-3 is the gate; L-2 says who inside an allowed session may pull it; they
compose, and the recommendation takes both.

## Recommendation

R-a with its two tightenings; W-ii; L-3 composed with L-2. The land is one procedure in
`internal/git` (`Land`, sharing `Collect`, the scan and the push with `Handoff.Run`); the session
end on a `main` project is a land followed by the removal, with the fallback. Configuration:
`"handoff": "main"` as a third value, `"handoffFallback": "push" | "pull-request"` (default `push`,
meaningful only with `main`), `roles.land` per channel. R-d is deferred with a trigger: the first
channel whose `land` group is wider than the owner's `trusted`, or a land whose fact was found gamed;
it slots in as the per-project `"review": "land"` setting without touching the other parts. ADR
0025 rule 8 is amended so that configuring `roles.land` on a PC instance is the owner's acceptance
that the gate there is a refusal, not a boundary, with the audit trail as the record. The
reviewer-model trailer question (`../../ideas/2026-10-31-a-reviewer-subagents-model-must-be-credited-on-the-commit-it-read.md`)
must be answered before the first `main` land: under the first reading every reviewed commit has to
be amended after its review to carry the reviewer's trailer, which voids the fact by construction,
so this proposal recommends the second reading (a model whose usage is a `reviewer` call's is not an
author), which needs `parent_tool_use_id` stored on the usage row. First implementation step: the
bundle in the hook decision and the non-empty-result rule (the fact's tightenings, provable on the
fake), then `Land` with `session land --dry-run`, then the `land` cell and the answer path, then the
venue command.

## What would change this recommendation

- The engine lets a hook block on a subprocess for seconds without the call falling through: if
  not, the bundle cannot move into the decision and R-a keeps its race, which argues for R-d.
- A `defer` in policy mode behaves differently from prompt mode on the pinned engine: then the push
  word needs a second transport (a `deny` whose reason tells the agent to ask the driver).
- The owner wants the review spent by the bot rather than the session on their own projects: R-d
  becomes the default and R-a the fast path.
- The owner answers the trailer question with the first reading: the land needs a re-review after
  the trailer amend, and the proposal should then make the `commit-msg` hook add subagent trailers
  at commit time so no amend is needed.
- The remote forbids fast-forward pushes to the default branch by policy (branch protection
  requiring a request): `main` is then `config check`'s refusal for that project, not a runtime fall
  back on every end.

## Claims to verify

- V1. The hook may run a subprocess (the bundle, under ten seconds) inside its `PreToolUse`
  decision for an `Agent` call and still return a decision the engine honours; the engine's hook
  timeout of thirty seconds applies (`../03-hook-contract.md`, How the engine runs it; engine docs).
- V2. The hook's `defer` output is honoured the same way whatever the bot's mode, since the mode is
  the bot's concept: `-p` ends the turn with `deferred_tool_use` and `--resume` with
  `BROODFORGE_ANSWER` set resumes it (`../../findings/2026-10-15-spike-hook-flow-claims-for-adr-0020.md`).
- V3. `BROODFORGE_ANSWER_REASON` returns verbatim to the model as the denied call's result text
  (`../03-hook-contract.md`, Environment; `internal/hook`).
- V4. `git push <remote> refs/sessions/<id>:refs/heads/<default>` without `--force` is refused by
  the remote when the branch is not an ancestor (git's `non-fast-forward`), on GitHub and on a bare
  local remote (`tests/repro`); `merge-base --is-ancestor` predicts it when the fetch succeeded.
- V5. `git merge --ff-only` in the canonical while the default branch is checked out, to a commit
  the canonical already holds (fetched from the bundle), leaves a coherent checkout (`internal/git/create.go`, `Fetch`).
- V6. The engine's `--resume` after a land with no tree change is the ordinary resume; and, for the
  rejected option β (a bot rebase), whether a resumed engine copes with files changed under it is
  unobserved (listed so the critic does not have to re-derive why β was dropped).
- V7. Subagent hook input carries `agent_type` and the driver process's does not
  (`../proposals/0025-critic-review.md`, P5), which L-2 relies on.
- V8. Discord: `/session` can gain a `land` subcommand within the slash command limits, and a
  subcommand can be restricted by role only on the bot's side, not Discord's
  (`../05-debug-interface-discord-live.md`).
- V9. `git-guard` as a project child hook denies `git rebase` in the tree, or does not; the lift
  text of `land-not-fast-forward` depends on it (`.claude/hooks/policy.json`, `git-guard.js`).
- V10. A second land to the same session branch on a `pull-request` project is a fast-forward of the
  remote branch and the open request follows its head without an API call (GitHub).

## ADR rules amended, and what the owner holds

ADR 0025 rule 4 ("never a push": the tree never pushes; a push word is a land request answered by
the bot), rule 7 (the fact pinned in the hook decision, non-empty result; the land as the procedure
above), rule 8 (the PC acceptance by configuration), the companion's `pre-push` text and refusal
table (new rows for `land-*`); ADR 0022 companion, Handoff step 4 (the `main` push and fallback) and
a new "Land mid-session" paragraph (steps 1 to 5 without removal); `../03-architecture.md`
Configuration (`handoff`, `handoffFallback`, `roles.land`), Git, the failure table (`land-refused`,
`land-failed`); `../03-hook-contract.md` (the `land` cell); `../04-ux.md` (a Landing section, the
end's `main` lines); `../05-debug-interface-git.md` and `-session.md`.

The owner words: the venue's land line, the fallback line, the refusals (`land-not-allowed`,
`land-unreviewed`, `land-not-fast-forward`, `land-rejected`), the new `pre-push` text, the
`/session land` command's placement (a slash command alone, or a header button beside `[End
session]`). The owner decides: whether R-d is wanted now (it spends a review per land) and on whose
credential; whether a mid-session land on a `pull-request` project spends a closing turn for the
request's title (the recommendation: no, the fallback title until the end's closing turn); the
trailer question above; whether `roles.land` on the PC instance is acceptable under rule 8.

## Critic's objections and responses

(none yet)
