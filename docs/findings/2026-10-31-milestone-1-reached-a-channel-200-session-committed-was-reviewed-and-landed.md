---
date: 2026-10-31
status: resolved
area: dev instance / session / git
---

# Milestone 1 reached: a channel-200 session committed, was reviewed, and landed its branch from a terminal

## Question

Whether milestone 1 of `docs/design/08-implementation-plan.md` (the dev instance on the PC: an
agent drives the bot, the real engine and this repository's workflow from a terminal) holds end
to end, and whether P31's live step (`session show` prints `review: <id> at <commit>`, then an
amend and `worktree check` flip it to `review: none since <new>`) behaves on the real engine.
The owner granted four real turns ("Go ahead with 4 real turns on the dev").

## What was tried

The `run` skill on the dev root, 2026-10-31, from the main checkout, by a `tester` for the drive
and a `runner` for the git steps; engine 2.1.314 native on `PATH`; schema 4.

- Step 0 found a bot already answering: the detached one left on 2026-10-30 by the earlier P31
  attempt, with session `s0005` (channel 200, the owner's registration, the dev credential)
  `waiting-for-answer` since 04:35Z on a deferred `Bash` `git add && git commit` (the engine had
  first tried PowerShell, denied `outside this channel's reach`).
- Fake smoke, channel 100, session `s0006`: `turn-ended  end_turn`; outbox `create` reads
  `Session s0006 on broodforge, branch broodforge/dev/s0006, driver Ada Example.`; `session end`
  printed `none  origin  -  0  -  -`.
- `session answer s0005 allow --tool-use-id <id> --as <owner id> --roles 10` at 16:46:56Z printed
  `turn  1  resumed`; `turn-ended  end_turn` at 16:47:25Z. The turn committed `p31.txt`, then
  launched the `reviewer` by `[Agent]` (allowed by policy); the reviewer returned no text, a later
  `ToolSearch` was denied, and the turn ended.
- `session show s0005`: `review: toolu_… at f8b277e…`.
- `git commit --amend --no-edit` in the tree: denied by `git-guard` for the tester (`commit not
  permitted for this caller … the subagents that make commits … are: runner`); done by the runner.
- `worktree check s0005` after the amend: `missing:Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`,
  `error: handoff-refused-attribution: 1 of 1 commits fail the attribution check`. The commit named
  `Claude Fable 5.1` (the engine's reported model, `claude-fable-5-1`); `usage --session s0005`
  shows a second model, `claude-opus-5-5` (the reviewer subagent, 4 requests), and the rule in
  `05-debug-interface-git.md` credits every model with usage from its first turn.
- Re-amended with both trailers: `worktree check` `ok  pre-push`, `<sha>  ok`, `ok  1`; `session
  show` `review: none since 7f03a31…`; `session end` `pushed  origin
  refs/sessions/s0005:refs/heads/broodforge/dev/s0005  1  -  -`; `git branch --list
  'broodforge/dev/*'` in the checkout lists `broodforge/dev/s0005`; `stop` `stopped  0`; `status`
  exit 1 `not-running`.

## Result

Milestone 1 holds: the whole loop ran from a terminal with no Discord, and the session's branch
landed in the main checkout. Real turns spent: 0 of 4 (the resumed turn was already open).

- P31's live step holds on the real engine: the review fact names the reviewer call and the sha
  bundled at its launch, and an amend flips it to `none since <new sha>` after `worktree check`.
- A deferred permission survives for hours: the engine process waited twelve hours on the hook and
  resumed in 29 s when answered. The session row showed `waiting-for-answer` with `until` a year
  out, as designed.
- The attribution rule requires a trailer for every model with usage in the session, so a reviewer
  that only read the commit must be credited on it, and an amend made after the review has to add
  that trailer by hand. Whether that is wanted:
  `docs/ideas/2026-10-31-a-reviewer-subagents-model-must-be-credited-on-the-commit-it-read.md`.
- The `git-guard` denies a tester's `git commit --amend` in a session worktree under the root, not
  only in this repository, so the P31 check step is runner work:
  `docs/ideas/2026-10-31-the-git-guard-makes-the-p31-amend-runner-only.md`.
- The reviewer subagent inside the session returned no text and the session's `ToolSearch` was
  denied; evidence for the open `2026-10-23-subagents-in-a-session-end-before-writing.md`, not
  investigated here.
