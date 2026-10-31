# Critic review of proposal 0028

Date: 2026-10-31. Reviewer: critic (Fable 5.1). Sources: the pinned engine's hooks reference at
code.claude.com (the PreToolUse decision-control section, the `defer` section, the subagent
paragraph), git's `git-push` and `git-config` pages, Discord's application-commands reference,
GitHub's pull-request docs, and this repository (`internal/git`, `internal/hook`, `internal/policy`,
`internal/session/{turn,workflow}.go`, `internal/engine/claudecode.go`, `.claude/hooks/git-guard.js`
and `policy.json`, ADRs 0020, 0022 and 0025 with companions, the 0025 critic review, the three ideas
the proposal cites and the milestone-1 finding). Per the brief, nothing already ruled out is
re-argued: no force push, the Go scan on every push, trailers per model, one review before a push,
the fallback and the role gate, nothing per project in code. No push was run here (`git-guard`
denies it to this agent); git behaviour is cited from its documentation.

## Verdict in one paragraph

The shape is right: one `Land` in `internal/git` that the end and the mid-session command both
call, the review fact as a sha the bot measured, the tree never pushing, a channel role gate
composed with a per-project subagent list. Four things break the recommendation as written. The
first tightening of R-a (the bundle inside the hook's PreToolUse decision) cannot be built: on the
droplet the hook runs as the engine user inside the sandbox with the canonical bound read-only and
`data/` unbound, so it can neither fetch `refs/sessions/<id>` nor take the bot's lock, and on the PC
it is the same-account gap by another name; the happens-before it wants is available only by
deferring the reviewer call and letting the bot pin before it resumes, which is the path W-ii
already builds. The push word delivers a successful land to the model as a tool error (`is_error:
true`, `PreToolUse:Bash hook error: Landed ...`), when the engine documents the resume with `allow`
and `updatedInput` as the intended pattern. The `git push` prefix turns `--dry-run`, `--force` and a
push of another branch into a land of the session branch onto `main`. And the instance that
motivates the proposal, the owner's `dev-live` whose remote is a non-bare checkout with the default
branch checked out, rejects every push to that branch by git's default
(`receive.denyCurrentBranch` is `refuse`), so as configured every land there falls back to the
session branch, which is what happens today. The proposal's refusal of a bot-side rebase (point a)
is right, for a reason it does not give; its acceptance of a second review per land (point b) is a
departure from a base-owned agreement that the ADR must name.

## Claims verified

| Claim | Finding | How |
|---|---|---|
| V1 | Holds as worded, proves the wrong thing. The docs: a `command` hook's default timeout is 600 s, the bot's settings set 30 s, a hook that reaches it is cancelled and "doesn't block the tool call" unless `onFailure: block`; `Agent` is a matchable tool with `subagent_type` in its input, so the hook sees the reviewer launch. But `03-hook-contract.md` (How the engine runs it) and the 0022 companion (The engine spawn) put the hook inside the sandbox as `broodforge-engine`, `<canon>` read-only, `data/` absent, no `sudo`: the fetch into `refs/sessions/<id>` (`Refresh`, `internal/git/handoff.go`) cannot run there, and `worktree-bundle` goes through the backend's `sudo bwrap`, which the hook cannot call. Objection 1. | docs, 03, 0022 companion, code |
| V2 | Holds by code, with a documented constraint the proposal lacks. `claudecode.go` spawns every turn with `-p --permission-mode default` whatever `BROODFORGE_MODE` is, so the engine sees no mode. The docs: `defer` "requires non-interactive mode with the `-p` flag", "there is no timeout or retry limit" (the twelve-hour wait in the milestone finding agrees), and "only works when Claude makes a single tool call in the turn. If Claude makes several tool calls at once, `defer` is ignored with a warning and the tool proceeds through the normal permission flow" (C28 on 2.1.296 observed both deferred and one dropped instead: pin drift). Objection 5. | code, docs, ADR 0020 C28 |
| V3 | Holds, wrapped. `policy.go` line 284 returns `Answer.Reason` verbatim on a deny; C27 (`2026-10-15-spike-hook-flow-claims-for-adr-0020.md`): the model receives it as a tool result with `is_error: true`, text `PreToolUse:<tool> hook error: <reason>`, `permission_decision.decision: reject`. The docs: the reason is "shown to Claude" and a `-p` deny is read "in the tool result". Objection 2. | code, finding, docs |
| V4 | Holds, client side. `git-push`: "Usually, `git push` will refuse to update a branch that is not an ancestor of the commit being pushed"; output `rejected`: "Git did not try to send the ref at all, typically because it is not a fast-forward and you did not force the update". The check is the client's and needs the remote tip's object locally: after a failed fetch the same line says `fetch first`, still a rejection. `receive.denyNonFastForwards` is the server belt (set by `git init --shared`, otherwise unset). `merge-base --is-ancestor` on the tracking ref predicts it when the fetch succeeded. The same page's `remote rejected` names `receive.denyCurrentBranch`: objection 6. | git docs |
| V5 | Holds by code. `create.go` `Fetch` does exactly this (`merge --ff-only --quiet <tracking>` while HEAD is the branch, else `update-ref <ref> <new> <old>`) and treats a refused fast-forward (a dirty checkout) as `stale`, not an error. The land's step 5 inherits the dirty case and the proposal does not say what it does then: objection 12. | code |
| V6 | Holds for the ordinary resume. For β, the hazard is known engine behaviour rather than unobserved: the `Edit` tool refuses to write a file whose content changed on disk since the model's last `Read` ("has been modified since read"), so a tree rebased under a resumed session makes every pending edit fail until re-read, and the model's picture of the files is wrong until then. Not verified against a doc page; stated from the engine's behaviour. Supports (a). | engine behaviour |
| V7 | Holds. The docs: hooks "also run inside subagents" and "the input carries the `agent_id` and `agent_type` common input fields"; `agent_type` is "present when the session uses `--agent` or the hook fires inside a subagent". The spawn passes no `--agent`, so the driver process has none (0025 critic P5, V4). | docs, 0025 review |
| V8 | Holds. Discord: up to 25 nested options per command or group, one level of nesting; `/session` has eight subcommands (`05-debug-interface-discord-live.md`). Permissions (`default_member_permissions`, the permissions endpoint) take a top-level `command.id` only; nothing per subcommand, so the `land` role is the bot's check. The three-second acknowledgement is the live gateway's existing pattern (deferred reply, follow-up), which a land of some seconds fits. | Discord docs, 05 |
| V9 | Holds, with the nuance the lift text needs. `git-guard.js` rule 4: `rebase` is in `COMMIT_SUBS` and `INTEGRATE_SUBS`; `git.integrate` is unset in `policy.json`, so it is judged by `git.commit.subagents` (`runner`): denied to every other subagent, exempt for `agent === null` (the driver process), `--abort` and `--quit` exempt, `-i` denied to all (rule 10). So "rebase in the tree" is the driver's or the runner's. | code |
| V10 | Holds. GitHub: "After you open your pull request, you can continue changing files by adding new commits to your head branch." A fast-forward push of the same branch needs no API call. | GitHub docs |

Further facts the sources produced, not in the proposal's list:

| Id | Finding | How |
|---|---|---|
| P1 | `receive.denyCurrentBranch`: "git-receive-pack will deny a ref update to the currently checked out branch of a non-bare repository ... Defaults to `refuse`"; `updateInstead` "will refuse the push if the working tree or the index have any difference from the HEAD". The `dev-live` remote is the owner's own checkout, on `main`. | git docs, brief |
| P2 | The docs' resume pattern: "The same tool call fires `PreToolUse` again, and the hook can return `allow` with the answer in `updatedInput`"; `updatedInput` "replaces the entire input object ... Claude Code evaluates permission rules ... against the input your hook returns"; "Combine with `allow` to auto-approve"; "For `defer`, ignored". | docs |
| P3 | `hook.go` `runsChain`: the repository's PreToolUse children run whenever the bot's decision is not `deny`, so a `defer` runs the chain first. `git-guard` rule 3 denies `git push` to every subagent but `pusher` ("Leave the work committed and report"), rule 20 runs the base privacy scan over what the push would send. | code |
| P4 | `turn.go` line 361: the pin happens when the bot reads the `tool-use` block; `subagentType` reads `subagent_type` from the model's own tool input; `ReviewFact` counts the newest `subagent-result` row with `agent: reviewer`, `result: ok`, no `commitError`, `commit == refs/sessions/<id>`. Nothing records whether the tree was clean at the pin. | code |
| P5 | `05-debug-interface-session.md`: every session command "takes `--as <user id> [--roles <id,...>]`; `--roles` absent, the action is unchecked; `--as` absent, it is the owner's from the host (actor `host`)". On the PC a session's Bash reaches the pipe (03, Debug surface attachment). | 05, 03 |
| P6 | Precedence when hooks disagree: "`deny` > `defer` > `ask` > `allow`". Only the bot's hook is registered, so this matters for the chain alone (P3). | docs |

## Refusals and lifts

| Mechanism | Who lifts it, from where, while a session runs | Does the message say so? |
|---|---|---|
| `land-not-allowed` (L-3, `roles.land`) | The owner: `roles.land` in the channel's config, `config reload`; nobody in the session or venue. | Not stated; the owner words it. A text is needed: "the owner adds a land role to this channel". |
| `land-not-allowed` (L-2, the copy's `git.push.subagents`) | The project's authors on the default branch; the next session (the copy is per session). | Not stated. The base's `git-guard` text for the same case says "ask the user", wrong in a venue (0025 review, objection 10). |
| `land-unreviewed` | The session: launch the reviewer on the tip, land again. | Yes ("launch the reviewer, then push again"). When the cause is a failed pin (`commitError`), the lift loops: objection 11. |
| `land-not-fast-forward` | The driver or the runner: fetch, rebase in the tree, review again, land again (V9). | The proposal gives the count and says the lift is the session's; whether the line names rebase and the runner is unsaid. With a diverged canonical the stated lift ("fetch origin, now current") is false: objection 6. |
| `land-rejected` (race) | Anyone: land again. | "Push rejected by the remote: <one line>" fits. |
| `land-rejected` (branch protection, `denyCurrentBranch`) | The owner: `handoff` to `pull-request`, or the remote's configuration; a retry is useless. | git's one line names the cause; nothing names the owner's lift. |
| The `land` cell's deny carrying the outcome | Nothing: it is the outcome, not a refusal. | The wrapper says `hook error`: objection 2. |
| A project hook's deny of the push word (P3) | The project's authors, default branch, next session. | The companion's child-deny text; it does not know the call was a land request. |
| The new `pre-push` text | Inside the session: issue `git push` as a call on its own. | To be worded; the proposal says it names the two forms. `git -c core.hooksPath= push` meets neither hook; on the PC a push to a non-checked-out branch of the canonical succeeds (rule 8). |
| Attribution and privacy refusals at a land | The session amends; a false positive is the owner's `privacy.allow`. | Existing texts hold. |
| ADR 0025 rule 8 on the PC | The owner, by configuring `roles.land`. | The audit as the record: objection 9 says what it would record. |

## Objections

1. **Blocking. The first tightening of R-a cannot pin anything the bot trusts.** The proposal moves
   the bundle "from the `tool-use` read to the hook's `PreToolUse` decision for the reviewer call,
   so the sha is pinned before the reviewer can start". The hook is a process inside the sandbox
   (V1): it cannot fetch into the canonical (`--ro-bind <canon>`), cannot reach the store or the
   bot (`data/` unbound, no channel by design, 03), cannot run `worktree-bundle` through `sudo
   bwrap`, and cannot share the lock the proposal gives the land, `worktree check` and the
   reviewer bundle. What it can write is `<sess>/out.bundle` or a sha file in `<sess>/home`, both
   owned by the session, which then becomes the fact's input: exactly the text-the-session-controls
   weakness the proposal rejects R-b for. On the PC the hook can write the canonical, through the
   gap rule 8 names. V1 was listed as the risk and it is not the risk; "What would change this
   recommendation" item 1 argues for R-d on the wrong failure. Resolves by: the hook answers
   `defer` for an `Agent` call whose `subagent_type` is `reviewer` (a second `defer` cell beside
   `land`); the bot, on `deferred_tool_use`, runs `Refresh` under its lock, records the sha and
   the tree's cleanliness (objection 8) on the call, and resumes with `BROODFORGE_ANSWER`
   `allow`, so the reviewer starts strictly after the pin on both hosts. Cost: one process round
   trip per reviewer launch (seconds, as the land's), already accepted for the push word. The
   non-empty-result rule stays as proposed. Alternatively keep the `tool-use` pin and state the
   race as a bound with its fixture, which R-a then carries honestly.

2. **Serious. The push word delivers a success as an error.** A `deny` reaches the model as a tool
   result with `is_error: true` and the text `PreToolUse:Bash hook error: Landed 3 commits on main
   at <sha>` (V3, C27). The model's documented reaction to a hook error is to report failure or try
   another route: in C27 it wrote "I couldn't create bye.txt. A PreToolUse hook blocked the
   write"; in the milestone finding the engine, denied `Bash`, tried PowerShell. A `git push` that
   "fails" after landing invites a retry (harmless: the next land is `none`) or an escape (`git
   push --no-verify`, `git -c core.hooksPath= push`, PowerShell), each a denial or, on the PC, a
   push to a non-checked-out branch of the canonical. The engine documents the other transport
   (P2): on resume the hook returns `allow` with `updatedInput`, the command replaced by one that
   prints the outcome line (the bot writes it to `<sess>/home/land-result`, the command is `cat`
   of that file, or the hook passes `BROODFORGE_ANSWER_REASON` through), so a landed push is a
   Bash call that succeeded with `Landed 3 commits on main at <sha>` as its output; a refused land
   stays a `deny` whose reason starts `Denied by Broodforge:`, which 03's classifier already
   knows. P2 of the 0025 review observed `updatedInput` applied on 2.1.309; ADR 0020's C27 left it
   "untested by design". Resolves by: the resume with `allow` plus `updatedInput` for a done land,
   `deny` with the prefix for a refusal, one fixture holding each, and 03's classification given
   the row (today a `hook`-decided error result with neither prefix has no class).

3. **Serious. The prefix `git push` is too coarse to be a request.** Any Bash command beginning
   `git push` becomes a land of the session branch onto the default branch: `git push --dry-run`
   writes `main`; `git push --force` is laundered into a fast-forward rather than refused with the
   never-force text; `git push origin my-subagent-branch` lands the session branch, not the
   branch named; `git push --tags` lands commits. Resolves by: the request is read from the
   deferred call's full `tool_input.command` by the bot (not the hook: 03 keeps the hook to first
   words): `--dry-run` runs `Land` with `dryRun`; any `--force*`, `--delete`, `--mirror`,
   `--tags`, `--all` is refused `land-bad-push` with the form that works; a remote or refspec other
   than the project's remote and `HEAD`, the session branch or the default branch is refused the
   same way; `git push`, `git push <remote>`, `git push <remote> HEAD[:<default>]` and the
   session branch land. The texts are the owner's; the table goes in the companion.

4. **Serious. The chain judges the push word before the bot does (P3).** `runsChain` runs the
   repository's PreToolUse children for every non-`deny` decision, `defer` included. This
   repository's `git-guard` then denies `git push` to every subagent but `pusher` with "Leave the
   work committed and report" (a text written for an interactive checkout), and for the driver runs
   rule 20, the base privacy scan over the push's sources, against an upstream the session branch
   does not have ("The scan could not read what this push sends ... no push leaves without it",
   a deny). So on this repository the push word reaches the bot from the driver only if rule 20
   passes, and from no subagent at all, L-2's list notwithstanding; the land's own scan (rule 7)
   is the one that counts. Resolves by: the `land` cell returns before the chain, as `deny` does,
   because the land is the bot's and the project's say is `git.push.subagents` read as data (L-2);
   or the ADR states that a project's hook may refuse a land request and the companion gives that
   row a lift. The same holds for the reviewer `defer` of objection 1 (the chain's `usage-gate`
   and `git-guard` judge an `Agent` call today and must keep doing so; only the bot's answer
   changes).

5. **Serious. A `defer` from inside a subagent is undocumented and unobserved; L-2 rests on it.**
   The owner's line is "another review phase and sub agent and the push word working mid-session",
   and L-2 admits a subagent the copy's `git.push.subagents` names (`pusher`). The docs describe
   `defer` for the turn's single tool call and say nothing about a subagent's call; whether the
   subagent's deferred call ends the whole process with `tool_deferred`, whether `--resume`
   re-fires the hook inside the subagent, and whether the subagent survives the resume, nobody has
   seen (C27 deferred a driver's `Write`). The docs also contradict C28 on parallel calls (V2): on
   the pinned engine a push in a message with another call is either dropped and re-issued or
   "proceeds through the normal permission flow", which in `-p` denies `Bash` off the safe list
   with no land. Resolves by: a probe before L-2 is promised (a `pusher` subagent issuing `git
   push` under the bot's hook, recorded as a fixture); until it passes, L-2 is the driver process
   alone and the header says a push is a call on its own; the ADR lists the subagent case as the
   check that widens it.

6. **Serious. The motivating instance cannot land by push, and the proposal does not know.** The
   `dev-live` remote is the owner's own checkout with the default branch checked out (brief).
   `receive.denyCurrentBranch` defaults to `refuse` (P1), so every `main` push there is `remote
   rejected`, `land-rejected`, fallback to the session branch: today's behaviour, now with a venue
   line. `updateInstead` lifts it only while the owner's working tree is clean, which a checkout
   the owner works in is not. The second half of the same incident is also unhandled: the owner
   rebased that checkout and the canonical diverged. Session start refuses `canonical-diverged`;
   the land's step 1 says "a failed fetch is noted, not a refusal" and nothing about diverged, so
   the land goes on, `merge-base --is-ancestor` fails, and the refusal tells the session to
   "fetch `origin` (the canonical, now current after step 1)", which it is not. Under this
   proposal nothing lands on that instance without the owner's cherry-pick, exactly as before.
   Resolves by: `config check` (and the first land) inspects a path remote: `rev-parse
   --is-bare-repository` false and `symbolic-ref HEAD` equal to the default branch is a warning
   naming `receive.denyCurrentBranch updateInstead` and the clean-tree condition, or a bare
   intermediary the owner pulls from; the land's step 1 gains `land-canonical-diverged` with the
   `05-debug-interface-git.md` reconcile lift (`branch -f` or `reset --keep` onto the remote's
   branch); the `land-not-fast-forward` lift names `<remote>/<default>` and the runner. The owner
   should be told plainly which of the two their habit needs.

7. **Serious. The second review per land is a departure from `one-review.md`, unnamed.** The
   agreement: "a fix to an unpushed commit is amended into it" and "a second reading of the same
   diff by the same model finds what the first did"; the practice is review, fix, amend, push.
   R-a holds only for the final sha, so every review that found something costs a second Opus
   reviewer run before a land (point b), and the `pusher`'s route of pushing an amended commit
   after one review becomes impossible for a `main` target. That may be the right price (the fix
   is a new diff, and the bot cannot tell a fix from new work without reading content), but the
   agreement is base-owned: `this-agreement.md` makes it a departure row and a carry-back idea in
   the same sitting, and the proposal records neither. The trailer argument beside it is overstated:
   the `trailers` file is per turn (ADR 0025 rule 10), so under the first reading the reviewer
   model's trailer is missing from commits of the turn that first ran the reviewer and present on
   every later turn's commits; the amend-and-re-review happens once per session, not per land.
   Resolves by: the ADR names the departure ("for a `main` land the review is the last step; a fix
   after it is reviewed again") with the cost line, the departure row and the carry-back; the
   header carries the one sentence; the trailer question is still the owner's, worded with the
   per-turn fact.

8. **Serious. The fact pins a sha, not what the reviewer read.** The reviewer reads the working
   tree; the pin is `refs/sessions/<id>` after a bundle of commits. A dirty tree at launch makes
   them differ: edits not yet committed are reviewed and then committed as a new sha (fails safe),
   or a line committed in the sha is hidden by an uncommitted edit that is discarded after the
   review (fails open: the reviewed-looking sha is pushed with a line no reviewer saw). The honest
   case is a session that fixes, reviews, then commits; the hostile case needs a driver model that
   wants to pass the gate. Resolves by: the pin (objection 1) records `status --porcelain` from
   `worktree-bundle`; a non-empty status is `commitError: dirty tree at the reviewer's launch`,
   which never counts, and `land-unreviewed` carries it; the header says to commit before launching
   the reviewer. A reviewer definition that diffs `<base>..HEAD` is the project's to write; the
   cleanliness check is the bot's part.

9. **Serious. On the PC the audit would record an impersonation as the owner.** The proposal amends
   rule 8 so that configuring `roles.land` on a PC instance is acceptance "with the audit trail as
   the record". A session's Bash on the PC reaches the pipe (03) and `session land <id> --as
   <owner id> --roles <land role>` is a CLI form the prefix check does not catch (`bash -c`, a
   script, `cd . && broodforge ...`): the land runs, and the audit row names the owner. The record
   rule 8 relies on is then wrong in the one case it exists for. Resolves by: `session land` from
   the CLI takes no `--as`: it is actor `host`, audited and posted as `landed from the host`, so a
   session's use of it is at least visible as such; the rule 8 sentence says what the PC's audit
   can and cannot distinguish.

10. **Minor. R-d's second trigger cannot fire.** "A land whose fact was found gamed" has no
    detector: the fact measures that whatever the project's workflow calls `reviewer` returned
    non-empty text on this sha (P4), and the brief that launched it is the driver's; the proposal
    says so in R-a and then builds a trigger on finding it out. Resolves by: drop that trigger,
    keep the role-width one (measurable from config), and carry R-a's one-sentence "what it proves"
    into the ADR and the venue line's wording (the owner's: "a reviewer ran on <sha>" rather than
    "reviewed").

11. **Minor. `land-unreviewed` hides a failed pin.** A reviewer row with `commitError` never counts
    and the refusal reads `review: none since <sha>`; the lift says launch the reviewer, which
    fails the same way if the bundle keeps failing (the lock clash the 2026-10-30 idea describes,
    a hung `worktree-bundle`). Resolves by: the refusal carries the newest row's `commitError`.

12. **Minor. Step 5 after a done push can leave `Base` stale.** `merge --ff-only` fails on a dirty
    or ahead canonical (`Fetch` treats it as `stale`; the land does not say). `Base` is the merge
    base with the local default branch, so the next land re-collects and re-scans commits already
    public; a new `privacy.never` entry could then refuse commits that are on the remote, and the
    venue counts are wrong. Resolves by: on a failed fast-forward the tracking ref is still
    updated and `Base` is taken against `<remote>/<default>` when it is ahead of the local branch;
    the outcome line says the canonical lags.

13. **Minor. The rebase refusal is right for a reason the text does not give.** "That would write
    commits nobody reviewed and void the fact by construction" is not quite so: `git replay` (git
    2.44+) rebases in the canonical without a worktree, and `git patch-id --stable` equality between
    the reviewed and the replayed commits is a mechanical, content-free fact the bot could measure,
    so a reviewed-equivalent construction exists. The real arguments are that a textually clean
    replay onto a moved base is a different program (the review was against the old base, and a
    `main` push has no CI gate behind it), and that a tree rebased under a resumed engine breaks
    its pending edits (V6). Resolves by: the ADR gives those two reasons and records
    replay-plus-patch-id as the deferred alternative with a trigger (a `land-not-fast-forward`
    rate that hurts the rhythm).

14. **Minor. Unnamed lifts (the table above).** `land-not-allowed` in both forms and
    `land-rejected` under protection have no lift in the proposal; the project-hook deny of a land
    request does not know it was one. Resolves by: a companion row each, the owner wording them.

## Pre-mortem

Ten years on, the regret is that the review fact became the review. It began as an honest
sentence in R-a ("what it does not prove: that the reviewer approved, or that the driver's brief
did not say approve") and a venue line that read `reviewed`. Projects configured `main` because
the branch and request paths cost a human; the gate was satisfied by whatever each project's
workflow called `reviewer`, and under cost pressure those became short prompts on cheap models
briefed by the agent whose work they read; the changelog channel announced every landing as
reviewed, and the people in the wider channels believed the word. When a bad change landed, the
audit showed a reviewer call with a non-empty result on the right sha, every box ticked, and the
trigger that was meant to bring in R-d ("a land whose fact was found gamed") had never fired,
because nothing could detect it. The line that should have caught this is the recommendation's
"R-d is deferred with a trigger: ... or a land whose fact was found gamed": a trigger that cannot
fire is a decision never to revisit, and the sentence in R-a that named the weakness was never
carried into the ADR or the venue's wording.

## Recommendation

Adopt the shape (R-a, W-ii, L-3 composed with L-2, the five-step `Land`, no bot-side rebase) with
these changes before the ADR: the pin is a `defer` of the reviewer call answered by the bot after
its own `Refresh`, which also records tree cleanliness, in place of a bundle inside the hook; the
push word is read from the deferred call's whole command by the bot, lands only the forms that can
mean a land and refuses the rest by name, and reports a done land through the documented `allow`
plus `updatedInput` resume so the model sees success as success; the `land` and reviewer cells
return before the project's hook chain; L-2 waits on one recorded probe of a subagent's `defer`
and is the driver process until then; the ADR names the departure from `one-review.md` with its
cost and the departure row; the land's step 1 gains the diverged outcome and `config check` warns
about a path remote with the default branch checked out, with the owner told that their `dev-live`
remote as it stands rejects every `main` push by git's default. On (a) I agree with the proposal
and would record replay-plus-patch-id as the alternative with a trigger; on (b) the second review
is the right price for a `main` push but must be written down as a departure, not accepted in
passing. Confidence: high on objections 1, 2, 4 and 6 (each read in code or the primary docs);
moderate on 5 (unobserved either way, which is the point); the role gate and the one-verb design I
would not change.
