# Ideas

One file per idea: something that would be better than what is here now, with a concrete anchor.
Not a task — nothing here is scheduled or decided. Not a finding — an investigation with a result
goes to `../findings/`. A bug that nobody has looked into yet is an idea of `kind: bug`; once it
has a root cause it becomes a finding and this file is deleted in the same edit. An idea with no
anchor (a file, a route, a command, a screen) does not earn a file: "the UI could be faster" is a
mood, and a directory of moods is worth nothing in six months.

## Shape

Files are named `YYYY-MM-DD-<slug>.md`, the date being when the idea was recorded, so a plain
listing shows how long each has sat. The slug states the claim in a few words, as a findings slug
does. Frontmatter:

```
---
date: 2026-10-13
kind: bug | feature | question | chore
related: [<file>, ...]        # optional: overlapping ideas that stay separate
---
```

Body: a title, then what you saw or want and why, in a few lines, with the anchor named in path
form; steps to reproduce for a bug; gaps under `## Open`. Short — brainstorming that settles into
a real choice becomes a proposal, not a longer idea.

The file ends in at most one block, which is how a reader tells one that is ready from one that is
waiting or has not been read yet:

- `Plan:` — the work is settled and fits one pass; the block is the brief it will be built from.
- `Question:` with a blank `Answer:` — a choice only the owner can make (whether they want it, which
  of two shapes, whether it is big enough to become a design doc). The owner fills in `Answer:`
  whenever they get to it; nothing waits on them.
- `Waits:` — the owner's own deferral to a trigger it names (a release, a platform change). No pass
  takes it before then, and no agent writes one on its own judgement.

A file with no block is untriaged and nothing builds from it. The `triage` skill reads each
untriaged file against its anchor and writes one block. The owner may write a block at filing time.

## Before adding: look for overlap

Duplicates are expected, so a new idea starts with a search, not a file: `grep -ril` the keywords
across this directory, `../findings/` and `../HANDOFF.md`. Same idea as an existing file: add the
new detail to that file as a dated line. Overlapping but distinct: create the file and link both
ways with `related:`, saying in one line how they differ. A duplicate that slipped in is folded into
the older file and deleted; git has it.

**Record an idea:** is how the owner files one from conversation. Write the entry from their words
rather than transcribing them — find the anchor, say what it would take — and ask only when the
anchor is genuinely unclear.

## What drains it, and on what terms

A pass builds only a `Plan:` or an answered `Question:`, oldest first, and deletes the file in the
commit that lands the work. A file is deleted for one other reason only: the code already has what
it asks for, which is a fact about the tree rather than a verdict on the idea.

No pass deletes an idea because it judged the idea not worth building. This directory is where the
owner decides what the project does not do, and a loop that pruned by its own taste would make that
decision by attrition. "Not worth building" is a `Question:` with a recommendation attached, and the
owner's no is the deletion. An idea that sits here for months is at rest, not stalled. When the
listing outgrows a screen, the owner reads the whole of it in one sitting, answers what is asked and
deletes what they do not want; no agent runs that sitting.

## Index

One line per file; the post-edit hook warns about an unlisted one.

| File | Kind | One line |
|---|---|---|
| `2026-10-14-git-guard-could-scope-commits-by-repository-location.md` | chore | A policy key exempting repositories under the scratchpad or temp directory from the commit rule, so spikes, reviews and hook tests can make scratch commits; base work |
| `2026-10-14-sessions-run-remote-controlled-on-the-droplet.md` | feature | Replace the PC-local wrapper script with remote-controlled sessions on the droplet, cloning per session |
| `2026-10-14-ux-choices-that-could-become-options.md` | feature | Venue, footer, trace, allow-session button, mode names, model switch, dot aliases as user or channel options |
| `2026-10-16-acp-as-the-second-harness-boundary.md` | feature | ACP (OpenAB, `claude-agent-acp`) as a candidate boundary for the R-42 and R-43 second harness; the first-version `Engine` stays bespoke |
| `2026-10-16-git-guard-false-positives-beside-variable-expansions.md` | bug | Lines with a bare `git` or `.git/` word beside a `$B` expansion are refused though nothing runs git; expansion-adjacent words should read as data; scratch-repository commits push reviewers to os/exec |
| `2026-10-16-p10-remainder-closing-turn-needs-login-interrupt-kill-live-reconcile.md` | feature | P10 left four pieces: the closing turn (needs a fixture), needs-login on auth error, the interrupt kill, live `audit --follow` and `reconcile` |
| `2026-10-16-p11-remainder-for-the-live-gateway.md` | feature | P11 left three pieces after the 2026-10-30 pass: notice texts, `/usage`, a real-clock repro |
| `2026-10-17-git-bash-stayed-waiting-after-engine-record-exited.md` | bug | The first `engine record` left Git Bash without a prompt after the recorder, engine and `tail` were all gone; three bash forks and the ConPTY pair remained; not reproduced yet |
| `2026-10-21-carry-back-quoted-git-does-not-name-git.md` | bug | The update to the base's guard dropped this project's fix: a quoted string that merely contains git names git again, so rule 19 refuses expansion stages; carry the fix back |
| `2026-10-21-carry-back-main-session-push-runs-check-5-read.md` | chore | The base's commit skill lets a main-session push skip the pusher's human read of outgoing commits; this project keeps it as a `CLAUDE.md` line; carry it back |
| `2026-10-21-carry-back-adr-amendments-section.md` | chore | Departure row: the adr skill's dated `## Amendments` section; carry it back |
| `2026-10-21-carry-back-brief-boundary-escape-classes.md` | chore | Departure row: a reviewer brief for a path or name boundary lists the escape classes; carry it back |
| `2026-10-21-carry-back-windows-reference-go-and-junction-facts.md` | question | Whether the base's Windows reference wants the scanner-quarantine and junction facts this project's copy carried |
| `2026-10-22-discord-layer-trace-for-debugging-the-live-bot.md` | feature | `serve --trace`: a stderr line per REST call and gateway state change beside the inbound log, on any root (designed 2026-10-30); what a second bot cannot see; the coder half waits on P12d |
| `2026-10-23-session-land-pushes-to-main-mid-session.md` | feature | `/session land`, `session land` and the agent's own `git push` run the one land on the commits so far and keep the session open; decided by ADR 0028, built by P34 to P37 |
| `2026-10-21-coder-stop-hook-for-build-and-vet.md` | question | A SubagentStop hook in the coder frontmatter running build and vet; a carry-back; wait for the trigger or build now |
| `2026-10-23-engine-auto-permission-mode-as-a-third-session-mode.md` | feature | The engine's `auto` permission mode as a third session mode beside asks first and acts freely; ADR 0020's trigger has fired; first find out whether `auto` works under `-p` here |
| `2026-10-23-carry-back-imports-js-backslash-import-escapes-the-tree.md` | bug | On Windows an `@` import holding backslashes (`@.claude\..\..\x.md`) is not outside in the base's `imports.js`, so a file above the root is stat'ed and read; the Go port treats it as outside; carry the rule back |
| `2026-10-23-a-config-error-starts-the-bot-on-last-good-and-the-fake-engine.md` | bug | A project file whose id and name disagree makes `serve` start on the last-good config with the fake engine; the owner got a canned reply and only `status` and the audit said why |
| `2026-10-24-a-refused-button-click-speaks-internal-parser-text.md` | bug | A `bad-args` on a bot-posted button answers with the parser's text (`internal/discord/customid.go`); say the click did nothing, the session's state, and `/session end` |
| `2026-10-24-a-turn-ending-on-a-tool-call-could-prompt-the-engine-for-a-closing-message.md` | feature | A turn whose last step is a tool call gets one engine prompt for a closing message (`internal/session/turn.go`), so the venue has the agent's own summary, not only the footer's `ended on a tool call` |
| `2026-10-24-discord-attachments-reach-the-session.md` | feature | A message's attachments are dropped at the gateway (only `m.Content` is read); download them, scan them like pasted text, and name the saved paths to the agent |
| `2026-10-26-o1-lift-names-the-hatchery-remote.md` | chore | O1's FAIL lift says "classify it"; in a project the cause is a missing `hatchery` remote and the fix is `git remote add hatchery <url>`; carry back, and warn at decant or session start |
| `2026-10-29-the-game-files-its-own-tickets.md` | feature | Moonshot: the hosted game's error handler or a player's bug report posts a channel message with the repro attached and one button opens a session on it; needs attachments first |
| `2026-10-29-broodforge-works-on-itself-overnight.md` | feature | Moonshot: a scheduled session triages ideas, builds the `Plan:` ones on branches and posts accept, review, discard buttons; needs the R-46 cap first |
| `2026-10-29-a-session-remembers-across-weeks.md` | feature | Moonshot: the engine writes a per-session handoff file into the worktree each turn, so a session resumed after a long gap starts from it, not cold |
| `2026-10-29-cross-host-lanes-run-a-check-on-the-other-instance.md` | feature | Moonshot: a droplet session runs a Windows-only check on the PC instance and gets the result in its venue; a departure from the one-instance vision, a proposal |
| `2026-10-29-voice-sessions-from-a-discord-voice-channel.md` | feature | Moonshot: a voice channel feeds speech-to-text turns into a session and summaries come back as speech; text stays the record; a proposal |
| `2026-10-30-carry-back-a-reviewer-finding-is-a-failing-test.md` | chore | Departure (ADR 0024 row): the reviewer Rules and the brief skill ask for a finding as a failing test; carry it back after carry-back/brief-escape-classes lands |
| `2026-10-30-real-turns-on-the-dev-root-seed-fixtures.md` | feature | A real turn on the dev root recorded and rewritten by importfixture into a candidate fixture, listed for the independent read; depends on the dev credential |
| `2026-10-30-the-dev-instance-is-a-base-practice.md` | chore | An agent-drivable dev instance and a project run skill as a Hatchery contract every decanted project ships; a carry-back request |
| `2026-10-30-venue-posts-shaped-for-the-reader.md` | feature | Posts lead with the next action, numbered steps, state restated, lists capped, no preamble; the bot's own lines first |
| `2026-10-30-credential-pool-scheduling-by-headroom.md` | feature | Moonshot: the bot routes overnight work to the subscription with headroom, extending ADR 0011 from advice to decision; money is the owner's |
| `2026-10-30-the-base-tends-itself.md` | feature | Moonshot: Hatchery as a Broodforge project that lands carry-backs on a sibling push and opens the update in each sibling |
| `2026-10-30-the-bot-files-its-own-findings.md` | feature | Moonshot: a refusal or warning in the serve log opens a scratch session that reproduces it on the fake engine and posts a finding with a repro |
| `2026-10-30-owner-less-weeks.md` | feature | Moonshot: overnight, memory, pool, self-findings and golden sessions combined so the owner is asked only for wording and money |
| `2026-10-30-carry-back-privacy-scan-at-commit-time.md` | chore | A `privacy-scan.js --staged` mode (added lines of one cached diff, same policy and allow passes) behind a base `pre-commit`, so a refused string is fixed before a rebase; the first build was reverted (bf96a70); a carry-back |
| `2026-10-30-serve-snapshots-the-store-before-a-migration.md` | feature | `serve` takes a `backup snapshot` before applying a schema migration and logs one line naming it; the 3 to 4 migration ran silently |
| `2026-10-30-two-session-leftovers-a-sixth-store-site-and-the-agent-decision-order.md` | bug | an `Agent` call's decision prints after its subagent's block (`transcript.go:81`) |
| `2026-10-30-policy-time-machine-replays-recorded-sessions-against-a-new-policy.md` | feature | Moonshot: replay recorded sessions' tool calls through the policy layer and diff the decisions, at no engine cost |
| `2026-10-30-engine-upgrade-canary-replays-prompts-on-the-new-engine.md` | feature | Moonshot: after an engine update, replay recorded prompts on the dev instance and diff the event-stream shape; one owner-gated real turn per session |
| `2026-10-30-the-canonical-stale-line-quotes-git-and-a-raw-timestamp.md` | question | P30's venue line carries git's whole fatal line (a path or URL) and an ISO timestamp; the owner picks a short reason, a relative time, or keeps both; Question |
| `2026-10-30-the-review-bundle-should-run-in-the-pretooluse-decision.md` | feature | P31 bundles on the `tool-use` read, seconds after the reviewer starts; superseded by ADR 0028's `reviewer` cell (the hook defers the call, the bot bundles, then resumes it); built by P33 |
| `2026-10-31-one-session-reaches-several-repositories-and-one-bot-serves-several-servers.md` | feature | After milestone 1: a project spanning several repositories with a worktree of each per session, groups of these under one bot, and one bot serving several servers with per-server account settings; reverses R-11; a proposal; Waits |
| `2026-10-31-a-reviewer-subagents-model-must-be-credited-on-the-commit-it-read.md` | question | The attribution check wants a trailer for every model with usage, so a reviewer that only read the commit must be credited and the amend loses the review fact; credit it or exempt reviewer-only usage; Question |
| `2026-10-31-research-agents-reach-the-web-without-reaching-the-machine.md` | feature | Refiled from s0009: a research subagent needs web search and fetch while files and shell stay in the tree; reach governs files and commands, not the network; a new reach value, a flag, or an agent-type grant; not sanity checked |
| `2026-10-31-rewind-and-fork-a-session-at-a-turn.md` | feature | Moonshot, not sanity checked: a ref per turn, a rewind button that resets the worktree and forks the engine transcript at that turn; the vibe programmer's undo |
| `2026-10-31-broodforge-deploys-broodforge.md` | feature | Moonshot, not sanity checked: a landing that passes build and the goldens builds, swaps and restarts the service, sessions held by ADR 0026, the old binary as rollback |
| `2026-10-31-the-agent-asks-the-venue-with-buttons.md` | feature | Moonshot, not sanity checked: a small tool set over the control socket lets the agent ask with buttons, read who is present and the budget, and pause itself; reverses the `mcp__*` deny, a proposal |
| `2026-10-31-zero-token-buttons-from-the-repository-workflow.md` | feature | Moonshot, not sanity checked: a repository declares fixed actions (tests, build, lint) that run as venue buttons under the session's reach with no engine turn |
| `2026-10-31-preview-this-branch-in-the-game.md` | feature | Moonshot, not sanity checked: a button deploys the session branch to a per-session preview instance of the hosted game; the hosting half lives in the game's repository |
| `2026-10-31-role-upgrade-evidence-digest.md` | feature | Moonshot, not sanity checked: R-05 built from the audit; a nightly digest to a moderator channel with evidence per person and a recommendation the bot never applies |
| `2026-10-31-a-since-you-left-line-on-return.md` | feature | Not sanity checked: on a person's first message after a long gap the bot leads with what landed, what waits on them and what was spent, from the store at no engine cost |
| `2026-10-31-spectator-mirror-of-a-session.md` | feature | Not sanity checked: a driver opts a session into a read-only public mirror of its venue posts, buttons stripped, so people learn by watching |
| `2026-10-31-a-changelog-channel-for-landings.md` | feature | Not sanity checked: each landing on the default branch posts the closing message, driver, branch and commits to a changelog channel |
| `2026-10-31-the-simple-questions-channel.md` | feature | Not sanity checked: the vision's easiest feature, a read-only cheap-model session answering anyone's question about the code under its own cap |
| `2026-10-31-carry-back-one-review-the-land-review-is-the-last-step.md` | chore | Departure (ADR 0024 row, ADR 0028 Decision 6): where the bot measures the review as a sha, a fix after the review is reviewed again, so for a `main` land the review is the last step; the agreement untouched; carry it back as a request after the first land |
| `2026-10-31-session-start-reply-opens-the-new-channel-with-a-button.md` | feature | Not sanity checked: the ephemeral start reply carries an "Open session" link button to the new channel and the venue's first post mentions the driver; `Reply` needs a link-button path (a URL, no custom id) |
| `2026-11-01-rename-the-session-home-directory-so-the-privacy-scan-stops-misreading-it.md` | chore | Not sanity checked: the session `home` directory trips the push scan's home rule on every file named in it; rename it and retire the allow entries |
| `2026-11-01-serve-posts-a-notice-when-the-engine-version-differs-from-the-pin.md` | feature | Not sanity checked: `serve` compares `claude --version` with the pin at start and posts one notice line when they differ |
| `2026-11-01-a-reviewers-tests-ship-with-the-fix.md` | chore | Not sanity checked: the reviewer writes its failing tests into the tree and the fix brief is make-them-pass; a carry-back |
| `2026-11-01-owner-worded-strings-carry-provenance-and-a-check.md` | chore | Not sanity checked: a 04 string marked the owner's carries its answer date and a check greps the quoted line in an `Answer:` |
| `2026-11-01-carry-back-one-commit-per-bash-call-in-the-commit-skill.md` | chore | Not sanity checked: the runner makes one commit per Bash call, messages with apostrophes or several paragraphs go through a file; a carry-back |
| `2026-11-01-questions-answered-from-discord.md` | feature | Moonshot, not sanity checked: each `Question:` block is posted to an owner channel with Accept, Edit, Later buttons and the answer is committed back |
| `2026-11-01-self-serve-privacy-allow-from-a-refused-push.md` | feature | Moonshot, not sanity checked: a refused push posts the hit to the owner channel with an Allow button that edits the list and relaunches the push |
| `2026-11-01-wave-lanes-in-their-own-worktrees.md` | chore | Moonshot, not sanity checked: each wave lane gets its own worktree and the runner lands lanes in order; a Hatchery practice change |
| `2026-11-01-the-wip-commit-runs-the-trees-own-git-config-as-the-bot.md` | bug | The session-end WIP commit runs `git status` and `git add` as the bot, so the tree's `core.fsmonitor` and `filter.*.clean` run as the bot; harmless on the PC, so the bwrap `WorktreeWIP` waits for a fix |
| `2026-11-01-reach-report-prints-no-row-for-the-reviewer-cell.md` | bug | `session reach-report` (`ClassRows`) prints no row for the `reviewer` cell P33 added to the policy table; fix with P34, the architect decides whether `CellFor` grows a row kind |
| `2026-11-01-carry-back-heredoc-reason-in-three-base-files-is-stale.md` | chore | Three base files say apostrophes break heredocs; they do not (the real reasons are parse-time failure and halved `\\`); fix each as a departure row and carry it back |
| `2026-11-01-lazy-venue-removal-when-the-channel-cap-nears.md` | feature | An ended session's channel waits under Ended sessions until Discord's category or guild channel cap nears, then the oldest ended goes first, in place of the ten-minute timer |
| `2026-11-02-redeploy-cannot-replace-the-binary-under-the-running-bot.md` | bug | `scripts/dev-task.ps1` `redeploy` builds before `Stop-Bot`, but on the PC go cannot overwrite `bin\broodforge.exe` while the task's bot runs (observed building P34); reproduce through `redeploy`, then stage-and-rename or stop first |
| `2026-11-02-every-repro-script-honours-bf.md` | chore | Not sanity checked: `full-session.sh` hard-codes the binary under `bin/` while the newer repro scripts take `BF`; with the dev-live bot holding the binary open, every proof run copied the whole tree |
| `2026-11-02-worktree-handoff-lands-main-with-the-role-unchecked.md` | question | Not sanity checked: The amendment names `session end` on the host (falls back) but not `worktree handoff`, which lands `main` with nobody's role checked; trust the host at both doors or give it `--as --roles` |
| `2026-11-02-role-lists-read-live-or-fixed-at-session-start.md` | question | Not sanity checked: The land gate now reads `roles.land` live so `config reload` lifts it; `start`, `contribute` and `answer` read whatever the session holds; one rule for every door, and a reload test per door |
| `2026-11-02-a-land-runs-the-projects-checks-before-the-fast-forward.md` | feature | Moonshot, not sanity checked: a `main` land runs the project's named checks on the bundled sha in a scratch worktree between the review fact and the push, `land-checks-failed` otherwise; the deploy and changelog ideas get their green mark |
| `2026-11-02-an-unattended-session-lands-on-idle.md` | feature | Moonshot, not sanity checked: a starter with the land role grants land-on-idle at start, so the overnight session lands a reviewed tip when it idles out instead of always falling back |
| `2026-11-02-an-adr-with-a-companion-is-read-for-agreement-before-a-pass-builds-it.md` | chore | Not sanity checked: Hatchery practice: the `adr` skill and the critic brief check a companion against the decision; the coder reads both for agreement and stops on a contradiction before coding (P34 met one half-built) |
| `2026-11-07-the-ask-cell-path-resolution-is-kept-twice.md` | chore | Not sanity checked: `internal/session/askpath.go` is a hand-kept copy of the hook's path resolution because `session` may not import `hook`; the two drift; the alternative is one copy in `policy` |
| `2026-11-07-usage-gate-pool-advice-needs-a-five-hour-window.md` | bug | The base usage gate holds on a `seven_day`-only usage-state file, but its pool advice needs both windows, so an account with no five-hour window is never offered; a carry-back with a test case |
| `2026-11-07-the-post-edit-hook-is-skipped-by-bash-edits-and-hangs-when-run-bare.md` | chore | Auto mode routes edits through Bash, so the PostToolUse edit hook never runs; a bare `post-edit.js <file>` reads stdin and hangs or checks nothing; the stdin form works |
| `2026-11-07-session-tail-agent-column-and-the-agent-rows-order.md` | chore | `session send --wait` prints a blank field where the tool use id's neighbour should be, and the `allow [Agent]` row lands at the result, after the subagent's rows |
| `2026-11-07-a-prompt-mode-process-that-never-prints-result-has-no-bound.md` | feature | A prompt-mode process that never prints `result` and never exits is held for ever (stdin open, no stream end, `ExitTimeout` unarmed); an idle bound in the adapter or in `session`, and a companion clause |
| `2026-11-08-one-engine-process-per-session-not-per-turn.md` | feature | Moonshot, not sanity checked: one engine process held open per session, turns on its stdin, no resume; a turn starts in milliseconds, background subagents survive the turn (`c42-bg` shows the engine does a second turn); an ADR 0020 mechanism change |
| `2026-11-08-a-venue-thread-per-subagent.md` | feature | Moonshot, not sanity checked: each subagent launch opens a Discord thread carrying its prompts, rows, its own Stop and cost, keyed by `agent_id`; the coordinator's turn as a tree |
| `2026-11-08-a-concurrency-pass-over-the-bot.md` | chore | Not sanity checked: where the bot serialises what goroutines could run at once (the event pump under the session lock, the hook chain's children, the pusher's checks); measure, then the three cheapest wins with race tests |
| `2026-11-08-engine-conformance-as-a-command.md` | feature | Moonshot, not sanity checked: `engine conform --real` re-records the claim fixtures under the installed engine and diffs shapes; `setup engine` refuses an unconformed engine; the claim list as the engine contract a second vendor must pass |
| `2026-11-08-policy-proposals-from-the-audit-log.md` | feature | Moonshot, not sanity checked: the bot proposes policy rows from its own `permission-decision` rows (allowed 40 times, never denied), the owner accepts in the notice channel, the row lands by review; the table needs a data form first |
