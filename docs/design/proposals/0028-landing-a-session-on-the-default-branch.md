# 0028. Landing a session's work on the default branch: `/session land`, `handoff: main`, and the review gate

Date: 2026-10-31 (revised the same day against `0028-critic-review.md`)
Status: promoted to ADR 0028

## Question

A Discord session's work reaches the remote only when the session ends (`internal/git/handoff.go`,
`Run`), and only as a session branch or a pull request. Under "check in rarely; lose nothing"
(`../00-vision.md`, 2026-10-23) a session lives for days, so finished work waits on an end nobody
wants yet, and the owner's own projects want no branch to merge by hand. Decided here as one design:
a land mid-session (`/session land`), `"handoff": "main"` at session end as a fast-forward only with
a fallback and a role gate (both ideas of 2026-10-23, the owner agreed to that shape), and the review
gate: how the bot knows an independent review happened before it lets commits onto the default
branch. The owner, 2026-10-31: "I want discord to be able to push
directly though, or (setting) send pull requests. Which indeed means another review phase and sub
agent and the push word working mid-session." And 2026-10-24: "a push by the session's own runner,
after the session's own independent review, should be an acceptable handoff, so the session's
agents can land work too, not only the driver." In ten years this should read as one verb, land,
whose gate is a fact the bot measured and whose every refusal names its lift.

## Constraints that matter here

- Both hosts (`../06-platforms.md`); GPL (ADR 0001); every feature drivable from a terminal; agents
  write all code; nothing per project in the bot's code. Already decided, not reopened: never a
  force push; the Go push scan (ADR 0025 rule 7) on every push; attribution trailers per model with
  usage; one independent review per change before it is pushed (`one-review.md`); the fallback to
  the session branch on a conflict or a rejected push; a role-group gate for the default branch.
- The bot never opens a session's tree and never rewrites history (ADR 0022 and its companion); the
  session never holds the git credential; on the droplet the canonical is bound read-only and
  `data/` is unbound, so nothing in a session can reach the bot or the remote (ADR 0022).
- The hook runs inside the sandbox with no channel to the bot (`../03-hook-contract.md`): it can
  reach neither the canonical's refs, the store nor the bot's locks; the bot reads the stream, and
  a `defer` ends the turn until the bot resumes it with `BROODFORGE_ANSWER` and
  `BROODFORGE_ANSWER_REASON`. A `defer` holds only for a call alone in its message (critic V2).
- The review fact as built by P31: the bot bundles the tree when it reads the `reviewer` `Agent`
  call's `tool-use` block and writes the sha into that call's `subagent-result` row; the fact holds
  when the newest `ok` row names the sha `refs/sessions/<id>` holds after the last bundle (ADR 0025,
  amendment of 2026-10-30; live on 2026-10-31, the milestone-1 finding).
- The PC has no boundary between a session and the bot (ADR 0025 rule 8): there, every gate below
  is a refusal a session could lift, until ADR 0022's W1 or WSL2.
- Venue texts are the owner's; what spends the owner's money is the owner's (`major-decisions.md`).

## The land, common to every option

One procedure, run by the bot from the canonical, that the session end and the mid-session command
both call: run the project's handoff now. On a `push` project it pushes the session branch; on a
`pull-request` project it pushes the branch and opens the request on the first land (later lands
fast-forward the same branch, so the request follows); on a `main` project it pushes the default
branch. Only the last needs the review gate; the first two are the review venue (rule 7 as it stands).

1. Fetch the remote's default branch into the canonical and fast-forward the local branch, as the
   session start does (ADR 0022 companion, Clone step 0). A failed fetch is noted, not a refusal:
   the remote's answer to the push is the truth. A canonical diverged from the remote refuses
   `land-canonical-diverged` with the start's reconcile lift (`../05-debug-interface-git.md`: push
   its commits, or `branch -f` / `reset --keep` onto the remote's branch).
2. `Refresh` and `Collect` as today (bundle, verify, forced fetch into `refs/sessions/<id>`, the
   attribution check), then the push scan. A refusal keeps everything as it was.
3. For a `main` target: the gate (the driver's land role; the review fact naming the tip exactly,
   by the option chosen below), then `merge-base --is-ancestor <remote>/<default> refs/sessions/<id>`.
   False means the default branch moved under the session: `land-not-fast-forward` with the count.
   The bot does not rebase: a clean replay onto a moved base is a different program from the one
   reviewed (a `main` push has no CI gate behind it), and a tree changed under a resumed engine fails
   its pending edits until every file is re-read (critic V6). The lift is the session's: fetch,
   rebase in the tree onto `<remote>/<default>` (the driver or the `runner`, critic V9), review
   again, land again. At session end this refusal, and a rejected push, fall back to
   `handoffFallback` (`push` by default, `pull-request` where the project says so) and the venue
   says why; mid-session there is no fallback, the session simply goes on.
4. `git -c credential.helper= push <remote> refs/sessions/<id>:refs/heads/<default>` with the
   askpass helper, never `--force`; audit `land` `attempted`, `done` or `failed`. A rejection is
   `land-rejected` with git's one line: a race (someone pushed between steps 1 and 4; land again) or
   a remote that refuses the branch (protection, or a non-bare remote with the branch checked out;
   the lift is the owner's, below).
5. After a done `main` push: fast-forward the canonical's local default branch to the landed sha
   (`merge --ff-only` while checked out, else `update-ref`, as `Fetch` does) and the tracking ref.
   Nothing in the tree changes, so the engine's next `--resume` sees the files it left.
   `refs/sessions/<id>` stays; `Base` (the merge base with the default branch) now equals the landed
   sha, so the next land hands off only what came after. When the fast-forward is refused (a dirty
   or ahead canonical, `Fetch`'s `stale`) the tracking ref is still updated, `Base` is taken against
   `<remote>/<default>` where it is ahead, and the outcome line says the canonical lags; nothing
   public is re-collected or re-scanned.

Mid-session the tree is kept and the session stays open; at the end step 5 is followed by the
removal (R-35) as today. One land at a time per session: the land, `worktree check` and the
reviewer pin (below) share a lock, since all three write `<sess>/out.bundle`. A running turn does
not block a land: only committed, reviewed commits move, and the tree is untouched. A path remote
that is a non-bare repository with the default branch checked out refuses every push to that branch
by git's default (`receive.denyCurrentBranch` `refuse`, critic P1): every `main` land there would be
`land-rejected`, every end a fallback. `config check` and the first land inspect a path remote
(`rev-parse --is-bare-repository`, `symbolic-ref HEAD`, `receive.denyCurrentBranch`) and warn,
naming the three lifts: a bare repository beside the checkout that the owner pulls from;
`updateInstead` on the checkout, which still refuses while its working tree or index differs from
`HEAD`; or GitHub as the remote. The owner's development instance is this case today.

## Options: the review evidence (for a `main` target)

### R-a. P31's review fact, covering the exact sha pushed

The newest `subagent-result` row of a `reviewer` call with result `ok` names the sha the bot
pinned at the call's launch, and that sha equals `refs/sessions/<id>` after the land's own refresh.
Two tightenings. First, the pin moves from the `tool-use` read to a happens-before the bot owns: an
`Agent` call whose `subagent_type` is `reviewer` gets a policy cell, `reviewer`, answered `defer`
in both modes; the bot, on `deferred_tool_use`, runs `Refresh` under the lock, records the sha and
the tree's cleanliness on the call, and resumes with `BROODFORGE_ANSWER` `allow`, so the reviewer
starts strictly after the pin on both hosts (the race the idea of 2026-10-30 found: the stream read
lags the hook by seconds). The pin cannot live in the hook, which runs inside the sandbox. A
reviewer call the engine did not defer (several calls in one message) gets no pin and never counts;
the header says to launch the reviewer as a call on its own, after committing. Cleanliness: a
modified or deleted tracked file at the pin (`status --porcelain` from `worktree-bundle`) is
`commitError: dirty tree at the reviewer's launch`, which never counts, because an uncommitted edit
can hide a committed line from the reviewer and then be discarded; untracked files do not dirty the
pin (they hide nothing committed). Second, the row counts only with a non-empty result text
(s0005's reviewer returned nothing and counted). What it proves: a `reviewer` agent, launched in
this session, read this exact committed tree and returned. Not proved: that the reviewer approved,
or that the driver's brief did not say "approve"; the bot interprets no project text (ADR 0025 rule
2), so the verdict stays the driver's, and the venue line says "a reviewer ran on <sha>", never
"reviewed". Staleness: a commit or amend after the review moves the ref at the next bundle and the
fact reads `none since <new sha>`; the land always bundles first, so no clock enters. Cost: no
tokens; one process round trip per reviewer launch (the turn ends, the bot pins in seconds, the
resume re-fires the hook for the same id), the wait already accepted for the push word. Moving
parts: the `reviewer` cell; the answer path with the bot answering; the lock. Failure modes: a hung
bundle delays the launch (bounded; a failed bundle is `commitError`, never counted); a fix after the
review amends the commit and voids the fact, so a second review of the amended sha is the price of a
land: for a `main` land the review is the last step, a departure from `one-review.md`'s review, fix,
amend, push (a row in ADR 0024's table and a carry-back idea; the ADR carries the cost). Exit cost: none.

### R-b. A trailer the reviewer leaves on the commit

`Reviewed-By: <model> <noreply@anthropic.com>` (or a `Broodforge-Reviewed: <tool use id>` line)
appended to the reviewed commit, checked by the land as the attribution check checks trailers. Moving
parts: the `runner` amends the commit after the reviewer's report (the base `git-guard` lets only the
`runner` commit). Failure modes: the trailer is text the session controls, so it proves nothing the
bot did not see; adding it changes the sha, so the reviewed sha is never the pushed sha; worst, an
amend keeps the message, so the trailer survives a change to the diff: the one form of evidence that
does not go stale. As a public record in `git log` it has value, and a project may require it in its
own `commit-msg` rules; as the gate it is weaker than R-a on every axis. Exit cost: none.

### R-c. An audit record of a reviewer launch the bot observed

The `subagent-result` row without the sha: a `reviewer` call that returned `ok` after the session's
last commit. This is the rule 7 text of 2026-10-23, replaced on 2026-10-30 because the committer
time is the session's to write (`GIT_COMMITTER_DATE`, `--amend --date`), so "after" proves nothing;
bounding the time by the previous bundle leaves the window open inside the turn. With the sha
attached it is R-a. Kept as the record of why the fact carries a sha; not a candidate.

### R-d. The land runs the review itself

The bot pins the sha, checks out `refs/sessions/<id>` into a bot-owned read-only worktree in the
canonical, runs one engine process of its own with the `reviewer` definition from the session's
workflow copy (a default prompt the bot ships when the project has none), asks for a verdict in a
fixed shape (`--output-format json`, one `blocking: true|false` field and the findings), and
refuses the land on a blocking finding, posting the findings to the venue. What it proves: an agent
that did not write the change and was not briefed by its writer read exactly the sha pushed and
found nothing blocking: the strongest reading of `one-review.md`; staleness cannot arise. Moving
parts: a second spawn kind (its own settings file, read-only reach of that worktree, no
`BROODFORGE_SESSION`), a credential (the driver's, or an owner key), a verdict format, a timeout, a
venue post. Failure modes: it spends money on every land, the owner's call; a session that already
ran its reviewer pays twice unless R-d replaces the in-session review for `main` projects; a lenient
reviewer passes what a strict one would not. Exit cost: the spawn kind and parser are deleted. Where
it earns its cost: a channel whose land role is wider than the owner's trusted circle, where R-a's
"the driver briefed the reviewer" is the weak link.

## Options: the push word mid-session

### W-i. The tree's `pre-push` hook lets a push the bot marked reviewed through

The hook would need the fact (no channel to the bot; `data/` is unbound), the credential (the
session never has it, ADR 0022), and a writable remote (the canonical is read-only on the droplet);
rule 7's Go scan, which runs in the bot, would not see the push. On the PC it works only through the
same-account gap. Moving parts: a mark file the bot writes into the tree's `home` (as `trailers`
is), the credential in the sandbox. Failure modes: the whole 0022 boundary. Rejected as a mechanism
on both hosts; `pre-push` stays a refusal, its text changed to say how to land (owner's wording).

### W-ii. The tree never pushes; the request reaches the bot and the land runs from the canonical

Three doors, one land. From the venue, `/session land` (the driver, by the land role); from the
host, `session land <id> [--dry-run] [--expect <value>] [--as <id> --roles <ids>]`, printing
`<outcome>  <remote>  <refspec>  <commits>  <url>  <request>` as `worktree handoff` does, outcome
`landed`, `pushed`, `pull-request`, `none` or `dry-run`, refusals `land-not-allowed`,
`land-unreviewed` (`review: none since <sha>`, plus the newest row's `commitError` when there is
one), `land-not-fast-forward`, `land-canonical-diverged`, `land-rejected`, `land-bad-push`, and the
handoff's own codes; and from inside the session, the push word: a `Bash` call whose command begins
`git push` or `broodforge session land` gets a policy cell, `land`, which the hook answers `defer`
in both modes, returning before the project's child chain (the land is the bot's; the project's say
is `git.push.subagents`, L-2). The bot parses the deferred call's whole `tool_input.command` (the
hook keeps to first words, 03): `git push`, `git push <remote>`, `git push <remote> HEAD[:<default>]`
and the session branch in either form land; `--dry-run` is the land's dry run; `-u`, `-q`, `-v` are
ignored; `--force*`, `--delete`, `--mirror`, `--tags`, `--all`, `--no-verify`, any other option,
remote or refspec is refused `land-bad-push` naming the form that works; the `broodforge session
land` spelling takes `--dry-run` alone. A done request (landed, pushed, pull-request, dry-run, none)
is resumed with `BROODFORGE_ANSWER` `allow` and the outcome line in `BROODFORGE_ANSWER_REASON`; the
hook, seeing the `land` cell answered `allow`, returns `allow` with `updatedInput` replacing the
command by a `printf` of that line, shell-quoted (the engine's documented resume pattern, critic
P2), so the model reads `Landed 3 commits on main at <sha>` as a Bash call that succeeded. A refused
land is resumed `deny` with `Denied by Broodforge: not landed: review: none since <sha>; launch the
reviewer, then push again`, which 03's classifier knows; a success is never a hook error. A `cd x &&
git push` meets `pre-push`'s refusal, whose text names the two forms. Moving parts: the `land` cell
and its two prefixes in `policy`; `updatedInput` in the hook's output (03 gains the field; ADR 0025
rule 2 drops a child's, untouched); the answer path with the bot answering; the land in
`internal/git`; the lock. Failure modes: a land costs the prompt-mode round trip (seconds, no
tokens); a request while a person's answer is pending queues behind it; an engine pin that ignored
`updatedInput` on resume would run the original `git push`, which `pre-push` refuses after the land
happened (the next land is `none`): the resume fixture guards it. Exit cost: the cell is deleted and
the prefixes fall to `ask`.

### W-iii. Both

W-i adds nothing W-ii lacks and reopens the boundary. Rejected with W-i.

## Options: who may land on the default branch

### L-1. The driver only

`/session land` and the end; the agent's `git push` is refused. Simplest; contradicts the owner's
2026-10-24 line; the driver becomes the bottleneck the vision's rhythm removes.

### L-2. The driver, or the session's engine where the project's workflow allows

The in-session request is accepted on the driver's behalf when the call came from the driver process
(no `agent_type`) or from a subagent the copy's `.claude/hooks/policy.json` `git.push.subagents`
names (read as data, as the privacy section is; the base names `pusher`); no policy file allows the
driver process alone. Every land is still judged by the driver's roles: the engine lands as the
driver, never as itself, and the audit row names both. A `defer` from inside a subagent is
undocumented and unobserved (critic objection 5): the subagent list is enabled only after one
recorded probe (V12, a `pusher` subagent issuing `git push` under the bot's hook, kept as a
fixture); until it passes, the push word is the driver process's and the header says so.

### L-3. The role group

The channel's `roles` gains `land` beside `start`, `contribute` and `answer` (a group name from
`roleGroups` or raw ids; absent means nobody). It is the channel's, not the project's, so a project
with a trusted channel and a wider one differs per channel while `handoff` stays the project's. A
`main` project with a channel that has no `land` list gets a `config check` warning: every session
there would fall back. L-3 is the gate; L-2 says who inside an allowed session may pull it; they compose.

## Recommendation

R-a with its two tightenings (the pin by `defer`, the non-empty result); W-ii; L-3 composed with
L-2, L-2's subagent list behind its probe. The land is one procedure in `internal/git` (`Land`,
sharing `Collect`, the scan and the push with `Handoff.Run`); the session end on a `main` project
is a land followed by the removal, with the fallback. Configuration: `"handoff": "main"` as a third
value, `"handoffFallback": "push" | "pull-request"` (default `push`, meaningful only with `main`),
`roles.land` per channel. R-d is deferred with one trigger, the first channel whose `land` group is
wider than the owner's `trusted` (measurable from configuration; "a fact found gamed" has no
detector and is dropped); it slots in as the per-project `"review": "land"` setting. A bot-side
replay onto a moved base (`git replay` with `patch-id --stable` equality as the fact) is the
recorded alternative to `land-not-fast-forward`, trigger: a refusal rate that hurts the rhythm. ADR
0025 rule 8 is amended: configuring `roles.land` on a PC instance is the owner's acceptance that the
gate there is a refusal, not a boundary; the audit tells a venue land (actor the Discord user) from
a host land (actor `host`) and cannot tell the owner's terminal from a session's Bash. `session
land` from the CLI takes `--as` and `--roles` so the gate is reproducible from a terminal, but its
audit row and venue post are always `landed from the host`, the `--as` value recorded as claimed,
never as the actor. The reviewer-model trailer question
(`../../ideas/2026-10-31-a-reviewer-subagents-model-must-be-credited-on-the-commit-it-read.md`)
must be answered before the first `main` land: under the first reading the commits reviewed in the
turn that first ran the reviewer lack its trailer (the `trailers` file is per turn, rule 10) and
must be amended after the review, which voids the fact for them: one re-review per session, not per
land. Recommended: the second reading (a model whose usage is a `reviewer` call's is not an
author), which needs `parent_tool_use_id` on the usage row. Order of work: the `reviewer` cell and
the pin (provable on the fake); `Land` with `session land --dry-run` and the `config check` warning;
the `land` cell, the command parse and the resume with a fixture each for a done and a refused land;
the venue command; the subagent probe.

## What would change this recommendation

- A `defer` of an `Agent` call does not end the turn and resume as C27's `Write` did (V1): the pin
  returns to the `tool-use` read with the race stated as a bound and its fixture, which argues for R-d.
- `allow` with `updatedInput` on a resume is not applied by the pinned engine (V11; critic P2 saw it
  applied on 2.1.309 from a merged hook): a done land is then a `deny` whose reason starts `Landed`,
  and 03's classifier gains that prefix.
- The owner wants the review spent by the bot on their own projects: R-d becomes the default, R-a
  the fast path. The owner answers the trailer question with the first reading: the `commit-msg`
  hook should then add subagent trailers at commit time so no amend is needed.
- The remote requires a request for the default branch by policy: `main` is then `config check`'s
  refusal for that project, not a runtime fallback on every end.

## Claims to verify

The critic's verification table (`0028-critic-review.md`) is authoritative over this list. V1 as
first worded (a bundle inside the hook's decision) held and proved the wrong thing: the hook cannot
pin, and the mechanism changed. V2 to V10 hold, each with the nuance the body now carries. Three
remain, each closed by a fixture: V1 (replaced), a `defer` of an `Agent` call whose `subagent_type`
is `reviewer` ends the turn and the resume with `allow` launches the subagent (C27 observed a
`Write`; an `Agent` call is unobserved); V11, `allow` with `updatedInput` on a resume replaces the
Bash command and the model sees its stdout as a succeeded call (the 0025 review's P2 observation);
V12, a `defer` from inside a subagent ends the process, re-fires in the subagent on resume, and the
subagent survives (the probe L-2's subagent list waits on).

## ADR rules amended, and what the owner holds

ADR 0025 rule 4 (the tree never pushes; a push word is a land request answered by the bot), rule 7
(the fact pinned at the reviewer call's `defer`, clean tree, non-empty result; the land procedure),
rule 8 (the PC acceptance by configuration, what the audit can and cannot distinguish), its
companion's `pre-push` text and refusal table (a row and a lift for every `land-*` code and for a
project hook's deny of a land request); ADR 0022 companion, Handoff step 4 and a "Land mid-session"
paragraph; ADR 0024's departure table (the `one-review.md` row, with its carry-back idea);
`../03-architecture.md` (Configuration, Git, the failure table); `../03-hook-contract.md` (the
`land` and `reviewer` cells, `updatedInput`, the chain skipped for `land`); `../04-ux.md` (Landing);
`../05-debug-interface-git.md` and `-session.md`.

The owner words: the venue's land line ("a reviewer ran on <sha>", never "reviewed"), the fallback
line, every `land-*` refusal with its lift (`land-not-allowed`: "the owner adds a land role to this
channel"; `land-rejected`: the remote's owner), the `config check` warning for a checked-out path
remote, the new `pre-push` text, the `/session land` command's placement (a slash command alone, or
a header button beside `[End session]`). The owner decides: which lift the development instance's
remote takes (a bare repository beside the checkout, `updateInstead` on the checkout, or GitHub as
the remote; it lands nothing by push until one is chosen); whether R-d is wanted now (a review spent
per land) and on whose credential; whether a mid-session land on a `pull-request` project spends a
closing turn for the request's title (recommended: no, the fallback title until the end's closing
turn); the trailer question; whether `roles.land` on the PC instance is acceptable under rule 8.

## Critic's objections and responses

1. The hook cannot pin anything the bot trusts (blocking). Right: the hook runs in the sandbox with
   no reach to the canonical's refs, the store or the lock, and a sha it wrote would be the
   session's text (R-b's weakness). The pin is now a `defer` of the reviewer `Agent` call answered
   by the bot after its own `Refresh`, cleanliness recorded (R-a); cost one process round trip per
   reviewer launch, no tokens. Accepted.
2. A landed push delivered as a tool error (serious). The hook contract permits the engine's resume
   pattern: nothing in 03 or ADR 0025 forbids the bot's own hook emitting `updatedInput` (rule 2
   drops a child's). A done request resumes `allow` with the command replaced by a `printf` of the
   outcome line carried in `BROODFORGE_ANSWER_REASON`; a refusal stays a classified `deny`; V11.
   Accepted with change (the line travels in the variable, not a file).
3. The `git push` prefix is too coarse (serious). The bot parses the whole command; the landing
   forms, the dry run and `land-bad-push` are the critic's, plus `-u`, `-q`, `-v` ignored and the
   `broodforge session land` spelling taking `--dry-run` alone. Accepted with change.
4. The child chain judges the push word first (serious). The `land` cell returns before the chain
   on the defer and on the resume; the project's say is `git.push.subagents` as data. The
   `reviewer` cell keeps the chain (`usage-gate` and `git-guard` judge the `Agent` call as today). Accepted.
5. A subagent's `defer` is unobserved (serious). L-2's subagent list waits on the probe (V12); the
   driver process alone has the push word until it passes. The owner's words stand as the goal. Accepted.
6. The motivating instance cannot land by push (serious). Right, and the diverged case too: step 1
   gains `land-canonical-diverged`; `config check` and the first land inspect a path remote and
   warn with three lifts (the critic's two and GitHub as the remote); the `land-not-fast-forward`
   lift names `<remote>/<default>` and the runner. Accepted with change (a third lift; the choice
   put to the owner, not recommended here).
7. The second review per land is an unnamed departure (serious). Named in R-a and the amended-ADR
   list: a row in ADR 0024's departure table on `one-review.md` with the cost line and a carry-back
   idea, the agreement untouched; the per-turn trailer fact corrected in the body. Accepted.
8. The fact pins a sha, not what the reviewer read (serious). Cleanliness is recorded at the pin
   from `worktree-bundle`'s `status --porcelain`; a modified or deleted tracked file is a
   `commitError` that never counts; untracked files hide nothing committed. Accepted with change.
9. On the PC the audit would record an impersonation as the owner (serious). Rule 8 already accepts
   that a session there can do what the instance account can. `session land` keeps `--as`/`--roles`
   so the gate is reproducible from a terminal, but the audit actor is always `host`, the `--as`
   value recorded as claimed, the venue saying `landed from the host`. Accepted with change.
10. R-d's second trigger cannot fire (minor). Dropped; the role-width trigger stays; "what it
    proves" goes into the ADR and the venue line reads "a reviewer ran on <sha>". Accepted.
11. `land-unreviewed` hides a failed pin (minor). The refusal carries the newest row's
    `commitError`. Accepted.
12. Step 5 can leave `Base` stale (minor). On a refused fast-forward the tracking ref is updated,
    `Base` is taken against `<remote>/<default>` where ahead, the line says the canonical lags. Accepted.
13. The rebase refusal's reason (minor). Step 3 gives the two real reasons; replay plus `patch-id`
    is the recorded deferred alternative with its trigger. Accepted.
14. Unnamed lifts (minor). Each `land-*` code and the project-hook deny of a land request get a
    companion row with a lift; the texts are the owner's and listed. Accepted.
