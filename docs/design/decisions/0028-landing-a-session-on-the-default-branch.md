# 0028. Landing a session's work on the default branch: `/session land`, `handoff: main`, the review gate

Date: 2026-10-31
Status: accepted (the mechanisms; the texts and the items under Waits on the owner are the owner's)

Record: proposal `../proposals/0028-landing-a-session-on-the-default-branch.md` and
`../proposals/0028-critic-review.md` (its verification table is authoritative; every objection resolved).
Companion: `0028-land-refusals-and-push-forms.md` (the push forms the land accepts and refuses, the
`land-*` refusals with their lifts, the `config check` warnings, the venue lines).
Amends ADR 0025 rules 4, 7 and 8, the ADR 0022 companion's Handoff steps, ADR 0024's departure table.

## Context

A session's work reaches the remote only when it ends, and only as a session branch or a pull request
(`internal/git/handoff.go`, `Run`; ADR 0025 rule 4). Under "check in rarely; lose nothing" a session
lives for days, so finished work waits on an end nobody wants, and the owner's own projects want no
branch to merge by hand. The owner asked (2026-10-24, 2026-10-31) for a direct push or a pull request
per project, the push word working mid-session, and a review phase by a subagent before the push.
The two ideas of 2026-10-23 (a `main` handoff; `/session land`) carried the shape the owner agreed to:
fast-forward only, a fallback, a role gate. Verified by the critic on the pinned engine's hooks
documentation, git's and GitHub's pages and this repository: the hook runs inside the sandbox and can
pin nothing the bot trusts (V1); a `defer` ends the turn and the resume re-fires the hook for the same
id, which may return `allow` with `updatedInput` replacing the command (P2, seen on 2.1.309); a `deny`
reaches the model as a tool error (V3, C27); `receive.denyCurrentBranch` defaults to `refuse`, so a
non-bare path remote with the default branch checked out rejects every push to it (P1); a `defer`
from inside a subagent is unobserved (V2, objection 5). Already decided and not reopened: never a
force push; the Go push scan on every push; trailers per model; one independent review per change.

## Decision

1. One land. `Land` in `internal/git` (sharing `Collect`, the scan and the push with `Handoff.Run`)
   runs the project's handoff now, from the canonical, never opening the tree: (1) fetch the remote's
   default branch and fast-forward the local one as session start does; a failed fetch is noted, a
   diverged canonical refuses `land-canonical-diverged`; (2) `Refresh`, `Collect`, the attribution
   check, the push scan; (3) for a `main` target the gate (the land role; the review fact naming the
   tip exactly), then `merge-base --is-ancestor <remote>/<default> refs/sessions/<id>`, false being
   `land-not-fast-forward` with the count: the bot never rebases (a clean replay onto a moved base is
   a different program from the one reviewed, with no CI behind a `main` push; a tree changed under a
   resumed engine fails its pending edits); (4) `git -c credential.helper= push <remote>
   refs/sessions/<id>:refs/heads/<default>` with the askpass helper, never `--force`, audited
   `land` `attempted`, `done` or `failed`; a rejection is `land-rejected` with git's one line;
   (5) after a done push, fast-forward the canonical's default branch and its tracking ref to the
   landed sha (`merge --ff-only` while checked out, else `update-ref`); when the fast-forward is
   refused the tracking ref is still updated, `Base` is taken against `<remote>/<default>` where it
   is ahead, and the outcome line says the canonical lags; nothing public is re-collected. The tree
   is untouched and the engine's next `--resume` sees the files it left. `refs/sessions/<id>` stays;
   the next land hands off only what came after. One land at a time per session: the land,
   `worktree check` and the reviewer pin share a lock (all three write `<sess>/out.bundle`). A
   running turn does not block a land. On a `push` project a land pushes the session branch; on a
   `pull-request` project the first land opens the request and later lands fast-forward its branch.
   The session end on a `main` project is a land followed by the removal; `land-not-fast-forward`
   and `land-rejected` at the end fall back to `handoffFallback` and the venue says why; mid-session
   there is no fallback, the session goes on. Four doors: the end; `/session land` in the venue;
   `session land <id> [--dry-run] [--expect <value>] [--as <id> --roles <ids>]` on the host, printing
   `worktree handoff`'s line with outcome `landed`, `pushed`, `pull-request`, `none` or `dry-run`;
   the push word from inside the session (rule 4).
2. Configuration. `"handoff": "main"` as a third value; `"handoffFallback": "push" | "pull-request"`
   (default `push`, meaningful only with `main`); `roles.land` per channel beside `start`,
   `contribute` and `answer` (group names or raw ids; absent means nobody). `config check` warns on
   a `main` project whose channel has no `land` list, and on a path remote that is a non-bare
   repository with the default branch checked out (companion).
3. The review gate (R-a). P31's fact, pinned by a new policy cell: an `Agent` call whose
   `subagent_type` is `reviewer` is answered `defer` in both modes (the `reviewer` cell,
   `../03-hook-contract.md`; the child chain still judges the call); the bot, on `deferred_tool_use`,
   runs `Refresh` under the lock, records the sha and the tree's cleanliness on the call, and resumes
   with `BROODFORGE_ANSWER` `allow`, so the reviewer starts strictly after the pin on both hosts. The
   fact holds when the newest `subagent-result` row of a `reviewer` call with result `ok`, a
   non-empty result text and no `commitError` names the sha `refs/sessions/<id>` holds after the
   land's own refresh. A modified or deleted tracked file at the pin (`status --porcelain`) is
   `commitError: dirty tree at the reviewer's launch` and never counts; untracked files do not dirty
   it. A reviewer call the engine did not defer (several calls in one message) gets no pin and never
   counts; the header says to launch the reviewer as a call on its own, after committing. What the
   fact proves: a `reviewer` agent launched in this session read this exact committed tree and
   returned. Not proved: that it approved, or that the brief did not say approve; the bot interprets
   no project text (ADR 0025 rule 2), so the venue line says "a reviewer ran on <sha>", never
   "reviewed". `land-unreviewed` carries `review: none since <sha>` and the newest row's
   `commitError` when there is one. No clock enters: a commit or amend after the review moves the ref.
4. The push word (W-ii; rule 4 of ADR 0025 amended). A `Bash` command beginning `git push` or
   `broodforge session land` gets the `land` cell: `defer` in both modes, returned before the child
   chain (the land is the bot's; the project's say is `git.push.subagents`, read as data). The bot
   parses the deferred call's whole `tool_input.command` against the companion's table: the landing
   forms land, `--dry-run` is the land's dry run, `-u`, `-q`, `-v` are ignored, everything else is
   `land-bad-push` naming the form that works. A done request (landed, pushed, pull-request, none,
   dry-run) is resumed `allow` with the outcome line in `BROODFORGE_ANSWER_REASON`; the hook, seeing
   the `land` cell answered `allow`, returns `allow` with `updatedInput` replacing the command by a
   shell-quoted `printf` of that line, so the model reads `Landed 3 commits on main at <sha>` as a
   Bash call that succeeded. A refused land is resumed `deny` with `Denied by Broodforge: not
   landed: <refusal>; <lift>`, which 03's classifier knows. The tree's `pre-push` stays a refusal
   whose text names the two forms (W-i rejected: the credential and the fact never enter the tree).
5. Who may land (L-3 with L-2). The channel's `roles.land` is the gate; every land is judged by the
   driver's roles, the engine landing as the driver, never as itself, the audit row naming both. The
   in-session request is accepted from the driver process (no `agent_type`); a subagent the copy's
   `.claude/hooks/policy.json` `git.push.subagents` names is accepted only after the V12 probe
   passes (Deferred); until then the header says the push word is the driver process's. From the
   host `--as` and `--roles` make the gate reproducible, but the audit actor is always `host`, the
   `--as` value recorded as claimed, the venue saying `landed from the host` (rule 8 as amended).
6. Departure from `.claude/agreement/one-review.md`. For a `main` land the review is the last step: a fix after it
   amends the commit, voids the fact, and is reviewed again on the amended sha, because the bot
   cannot tell a fix from new work without reading content. A row in ADR 0024's departure table and
   `../../ideas/2026-10-31-carry-back-one-review-the-land-review-is-the-last-step.md`; the agreement
   file is untouched.

Out of scope: the venue beyond the companion's texts (04, the owner); the PC boundary (ADR 0022).
Deferred, each with its trigger: R-d, the bot running the review itself in a bot-owned read-only
worktree with a fixed verdict shape, as the per-project `"review": "land"` setting, when the first
channel's `land` group is wider than the owner's `trusted` group (measurable from configuration; "a
fact found gamed" has no detector and is dropped); a bot-side replay onto a moved base (`git replay`
with `patch-id --stable` equality as the fact) when the `land-not-fast-forward` rate hurts the rhythm;
L-2's subagent list, when the V12 probe (a `pusher` subagent issuing `git push` under the bot's hook,
kept as a fixture) shows the subagent's `defer` ends the process, re-fires on resume and the subagent
survives; `main` as a `config check` refusal, when a remote requires a request for the branch by policy.

## Consequences

- Costs: one process round trip per reviewer launch and per land (seconds, no tokens); a second
  reviewer run after a fix before a `main` land; `updatedInput` enters the hook's output and the
  resume fixture guards an engine pin that ignores it (then a done land becomes a `deny` whose
  reason starts `Landed` and 03's classifier gains the prefix); the `land` and `reviewer` cells are
  design, held in `policy`'s table; the lock serialises the three bundle writers.
- The development instance's remote is GitHub (the coordinator's choice on the owner's "Decide for
  me", 2026-10-31; the bare repository beside the checkout and `updateInstead` were rejected for that
  instance: two sources of truth, and writes into a tree agents are editing). This is that instance's
  configuration, not a rule: the owner adds the git credential (`setup git-credential`) and re-points
  the canonical's `origin`; until then every `main` land there is `land-rejected` and every end falls back.
- Supersede when the engine lets a hook pin state the bot trusts (the `reviewer` cell goes), or when
  R-d becomes the default (the fact becomes the fast path).

Waits on the owner (recorded, not decided): R-d now or later, and on whose credential; whether a
mid-session land on a `pull-request` project spends a closing turn for the request's title
(recommended: no, the fallback title until the end's closing turn); the reviewer-model trailer
Question (`../../ideas/2026-10-31-a-reviewer-subagents-model-must-be-credited-on-the-commit-it-read.md`;
recommended: the second reading, which needs `parent_tool_use_id` on the usage row; it must be
answered before the first `main` land); `roles.land` on the PC instance under rule 8 as amended; every
venue and refusal text (companion, marked "wording: owner").

- Prediction, check by 2026-12-31: the first ten `main` lands on the development instance show no
  `land` audit row whose fact was pinned by the `tool-use` read (every pin is a `reviewer` defer).
- Prediction, check by 2027-01-31: fewer than one land in five on `main` projects is
  `land-not-fast-forward`; above that the replay alternative's trigger has fired.
- Prediction, check when the engine pin moves: the resume fixture for a done land still shows the
  model reading the outcome line as a succeeded Bash call, with no change to `internal/hook`.
- Fitness, run by the test command: the land resume fixtures (a done and a refused land) and every
  row of the companion's push-form table pass against the pinned engine's recorded stream.

## Amendments

- 2026-11-02, the P34 build (`../08-implementation-plan-landing.md`) found Decision 1 and the companion
  disagreeing. Sources: the owner's principle of 2026-10-23 (a refused land falls back to the session
  branch and says so; nothing is lost); the WIP commit of 2026-11-01 (`../04-ux-session-end.md`), which
  makes every dirty-tree end on a `main` project `land-unreviewed`; the 0022 companion's amendment of
  2026-10-31 (a refused gate at the end falls back). Decision 1's "`land-not-fast-forward` and
  `land-rejected` at the end fall back to `handoffFallback` and the venue says why" is replaced: at the
  end every land refusal falls back to `handoffFallback` except `land-canonical-diverged`, final as a
  `handoff-refused-*` is (tree and session kept; the owner reconciles and ends again), because it names
  a fault in the host's canonical that a fallback push would publish under the session's name. The
  fallback line (companion, Venue lines) carries the first refusal met in Decision 1's order: role,
  fact, fast-forward, push. At the end Decision 5's "the driver's roles" is the ender's roles: the
  member who pressed `[End session]` or ran `/session end`. An end with no roles behind it, the idle
  timeout and `session end <id>` on the host (no `--as`, no `--roles`), is `land-not-allowed` and so
  falls back; a host land is `session land <id> --as --roles` before the end. Mid-session nothing
  changes: every refusal keeps the session open. The companion is corrected in place (living tables).
- 2026-11-07, ADR 0029 (`0029-subagent-prompt-in-prompt-mode.md`; source
  `../../findings/2026-10-23-subagents-in-a-session-end-before-writing.md`, Third look). Under `-p` the engine ends a
  subagent at a hook `defer` and passes nothing up, so a subagent's call on either bot-answered cell was lost silently
  in every mode (the `land-not-allowed (subagent)` text could never be delivered, since the hook read no `agent_id`).
  Three Decision sentences gain the same exception, a subagent's call being one whose hook stdin carries `agent_id`
  (ADR 0029 rule 1). Decision 3's "an `Agent` call whose `subagent_type` is `reviewer` is answered `defer` in both
  modes" holds for the driver process's call; a subagent's is answered `deny` in every mode with ADR 0029's reviewer
  text, no pin and no resume (the fact counts a reviewer the driver process launched). Decision 4's "gets the `land`
  cell: `defer` in both modes, returned before the child chain" holds for the driver process's call; a subagent's is
  `deny` in every mode, before the chain, with the companion's `land-not-allowed (subagent)` text. Decision 5's "a
  subagent the copy's `.claude/hooks/policy.json` `git.push.subagents` names is accepted only after the V12 probe
  passes (Deferred)" keeps its header sentence, but the V12 probe as written cannot pass, its premise refuted; the
  acceptance now waits on ADR 0029's C50, and `git.push.subagents` is still read as data and decides nothing till
  then. The Deferred item "L-2's subagent list, when the V12 probe ... shows the subagent's `defer` ends the process,
  re-fires on resume and the subagent survives" now reads: L-2's subagent list revives when C50 holds and the `land`
  cell for a subagent is carried through the stdio prompt tool, the bot answering the request itself with
  `updatedInput`, no resume.
