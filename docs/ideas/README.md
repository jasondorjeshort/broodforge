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
| `2026-10-17-unknown-stream-events-are-dumped-verbatim.md` | bug | `ParseLine` streams any unknown line whole; engine 2.1.296 sends two `commands_changed` events of several KB per turn, so `engine run` prints pages; designed 2026-10-30 as type, subtype, length and head, the recording verbatim; the coder's pass is P29 |
| `2026-10-17-git-bash-stayed-waiting-after-engine-record-exited.md` | bug | The first `engine record` left Git Bash without a prompt after the recorder, engine and `tail` were all gone; three bash forks and the ConPTY pair remained; not reproduced yet |
| `2026-10-21-carry-back-quoted-git-does-not-name-git.md` | bug | The update to the base's guard dropped this project's fix: a quoted string that merely contains git names git again, so rule 19 refuses expansion stages; carry the fix back |
| `2026-10-21-carry-back-main-session-push-runs-check-5-read.md` | chore | The base's commit skill lets a main-session push skip the pusher's human read of outgoing commits; this project keeps it as a `CLAUDE.md` line; carry it back |
| `2026-10-21-carry-back-adr-amendments-section.md` | chore | Departure row: the adr skill's dated `## Amendments` section; carry it back |
| `2026-10-21-carry-back-brief-boundary-escape-classes.md` | chore | Departure row: a reviewer brief for a path or name boundary lists the escape classes; carry it back |
| `2026-10-21-carry-back-windows-reference-go-and-junction-facts.md` | question | Whether the base's Windows reference wants the scanner-quarantine and junction facts this project's copy carried |
| `2026-10-22-discord-layer-trace-for-debugging-the-live-bot.md` | feature | `serve --trace`: a stderr line per REST call and gateway state change beside the inbound log, on any root (designed 2026-10-30); what a second bot cannot see; the coder half waits on P12d |
| `2026-10-23-handoff-main-lands-a-session-on-the-default-branch.md` | feature | A third `handoff` value, `main`: check, rebase, fast-forward push to the default branch, falling back to the session branch; gated on a role group; waits on the owner's ADR |
| `2026-10-23-session-land-pushes-to-main-mid-session.md` | feature | `/session land` runs the `main` handoff's steps on the commits so far and keeps the session open; waits on the owner's ADR |
| `2026-10-21-coder-stop-hook-for-build-and-vet.md` | question | A SubagentStop hook in the coder frontmatter running build and vet; a carry-back; wait for the trigger or build now |
| `2026-10-23-engine-auto-permission-mode-as-a-third-session-mode.md` | feature | The engine's `auto` permission mode as a third session mode beside asks first and acts freely; ADR 0020's trigger has fired; first find out whether `auto` works under `-p` here |
| `2026-10-23-carry-back-imports-js-backslash-import-escapes-the-tree.md` | bug | On Windows an `@` import holding backslashes (`@.claude\..\..\x.md`) is not outside in the base's `imports.js`, so a file above the root is stat'ed and read; the Go port treats it as outside; carry the rule back |
| `2026-10-23-a-config-error-starts-the-bot-on-last-good-and-the-fake-engine.md` | bug | A project file whose id and name disagree makes `serve` start on the last-good config with the fake engine; the owner got a canned reply and only `status` and the audit said why |
| `2026-10-24-a-refused-button-click-speaks-internal-parser-text.md` | bug | A `bad-args` on a bot-posted button answers with the parser's text (`internal/discord/customid.go`); say the click did nothing, the session's state, and `/session end` |
| `2026-10-24-a-session-starts-from-a-stale-canonical.md` | bug | `git.Create` branches from the canonical's local default branch, which nothing fetches; s000a started 4 commits behind the remote; designed 2026-10-30 as a fetch and fast-forward before the clone, the coder's pass is P30; the venue text is the owner's |
| `2026-10-24-a-turn-ending-on-a-tool-call-could-prompt-the-engine-for-a-closing-message.md` | feature | A turn whose last step is a tool call gets one engine prompt for a closing message (`internal/session/turn.go`), so the venue has the agent's own summary, not only the footer's `ended on a tool call` |
| `2026-10-24-discord-attachments-reach-the-session.md` | feature | A message's attachments are dropped at the gateway (only `m.Content` is read); download them, scan them like pasted text, and name the saved paths to the agent |
| `2026-10-26-o1-lift-names-the-hatchery-remote.md` | chore | O1's FAIL lift says "classify it"; in a project the cause is a missing `hatchery` remote and the fix is `git remote add hatchery <url>`; carry back, and warn at decant or session start |
| `2026-10-23-review-fact-trusts-the-session-committer-time.md` | bug | Rule 7's review fact compares the audit row with the commit's committer time, which the session sets and can backdate; amended 2026-10-30 to the sha the bot bundles at the reviewer's launch (ADR 0025 Amendments); the coder's pass is P31 |
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
| `2026-10-30-a-transcript-decision-line-names-neither-its-call-nor-its-depth.md` | bug | The transcript prints `  - allow (policy)` with no call or depth, and an Agent call's lands after its subagent's text; print `for <call>` at depth, re-bless the `subagent-end` golden |
| `2026-10-30-a-store-failure-inside-a-turn-is-logged-as-a-permission-decision.md` | bug | A store failure in `turn.go` is a permission-decision event with only a summary, rendered as `  -  ()`; whether internal errors get their own event type is the owner's |
| `2026-10-30-importfixture-rewrites-a-minted-credential-id-to-cred-dev.md` | chore | `importfixture --events` rewrites a minted `cred-` id to `cred-dev` like the session id, so a recapture of `dev-read` does not churn four places; re-blesses it once |
