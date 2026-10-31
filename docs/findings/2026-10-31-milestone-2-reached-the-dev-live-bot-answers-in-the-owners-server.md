---
date: 2026-10-31
status: resolved
area: dev-live instance / setup / session / git
---

# Milestone 2 reached: the dev-live bot answers in the owner's server and a session's commit landed

## Question

Whether milestone 2 of `docs/design/08-implementation-plan.md` (Milestones, second row: the live
form under the owner's account, a second root and instance `dev-live`, answers in the owner's test
guild) holds end to end: setup in few steps against the real Discord, a real session from Discord
on the owner's subscription, its commit handed off and landed.

## What was tried

The owner drove on 2026-10-31, the main session coordinating; a `tester` then read the root back
read-only. Lines marked (owner) are the owner's report, not re-observed.

- (owner) Root `%LOCALAPPDATA%\Broodforge\dev-live`, instance `dev-live`, made by `bin/broodforge.exe
  setup --root <root> --instance dev-live --project broodforge --repo <this checkout>` (the live
  form, no `--discord dry-run`), with the owner's existing Discord application and own server. The
  project's remote is the checkout itself, a local path with no forge, so `handoff` is `push`. The
  setup lines were not pasted; whether the Manage Roles removal `fix` appeared is unrecorded.
- (owner) Four texts misled the owner on the way, and a fifth (`canonical-diverged`) later:
  `docs/ideas/2026-10-31-the-first-live-run-texts-name-missing-or-wrong-commands.md`.
- (owner) The bot was started by hand with `serve`. Two steps no text named were needed before a
  start was accepted: `credential login <id> --done` and `register set-default <user> <id>`; without
  them the start was refused "you have no default credential".
- (owner) Sessions on the project channel, the owner as driver, one `subscription` credential.
  `s0002` (Fable, 3 turns) committed ten idea files on `broodforge/dev-live/s0002`; its own `git
  push` was refused by the tree's pre-push hook, as designed; `/session end` ran the handoff.
- (owner) The branch was reviewed by an Opus reviewer in the checkout (merge after fixes: a README
  index conflict and five wording fixes), cherry-picked onto `main` by the runner with the fixes
  folded in, then rebased when `origin/main` moved. The rebase rewrote the hashes the bot's canonical
  had fetched, so the next start was refused `canonical-diverged`; reconciled with `git -C
  <canonical> reset --keep origin/main`, the remedy `05-debug-interface-git.md` (The fetch at session
  start) prescribes.
- Read-only, in Git Bash with `BROODFORGE_ROOT` set to the dev-live root:
  - `status`: `config: ok`, `discord: live connected`, 1 session open, 0 waiting-for-answer.
  - `session list`: `s0001` to `s000a`; nine `ended`, `s000a` `waiting-for-input` with 0 turns. The
    channel and driver are the same on every row.
  - `session show s0002`: `ended`, `claude-fable-5-1`, 3 turns, mode `policy`, worktree `removed`,
    one `cred-…` credential.
  - `audit --session s0002 --kind push,pull-request-opened,handoff-refused`: `push attempted` then
    `push done`, `commits: 1`, refspec `refs/sessions/s0002:refs/heads/broodforge/dev-live/s0002`;
    no `pull-request-opened` and no `handoff-refused`.
  - `credential list`: two `bot-token` (one `removed`, one `usable`) and one `subscription`
    (`usable`).
  - Checkout: `git branch --list 'broodforge/dev-live/*'` lists `s0002`, `s0004`, `s0005`; `git log
    --oneline -1 87e90fa` is the ten-ideas commit on `main`.
- (owner) Seen again: the `@docs/HANDOFF.md` import is not loaded in the session copy (item 3 of
  `2026-10-24-session-friction-skills-memory-handoff-typed-commands.md`), and the session told the
  owner it had no handoff.

## Result

Milestone 2 holds: setup against the real Discord, a real session from Discord on the owner's
subscription, a commit made in the session tree, the handoff pushed, the branch reviewed and landed
on `main` as `87e90fa`. It held with friction, all of it in text and in steps nobody named.

Cost, from `usage` (turns by model, all sessions in the root): Fable 3, Opus 2, Sonnet 10, total
15 turns. The Fable turns are `s0002`, the one that committed; no turn was run by the tester. The
sessions after `s0005` were started from the rebased `main` and are not part of the landed commit.

Durable observations:

- The first live run's texts name commands that do not exist, or the wrong ones, in five places;
  each cost the owner a question. They are the owner's to word and are recorded in the idea above,
  not repeated here. The two steps that no text names (`credential login --done`,
  `register set-default`) belong with item 3 of that idea.
- A rebase of the checkout's `main` under a live bot rewrites hashes the canonical already fetched,
  and the next start is refused `canonical-diverged`. The remedy is the one the design already
  prescribes: `git -C <canonical> reset --keep origin/main`. The refusal line does not carry it.
  The end-only handoff (the branch lands only at `/session end`) is what the owner reacted to;
  that reaction is proposal `docs/design/proposals/0028-landing-a-session-on-the-default-branch.md`,
  under review.
- The same rebase had a second effect: s0004 and s0005, started from the pre-rebase `main` and
  committing nothing, were handed off as `push done` with `commits: 4` each (audit rows 84 to 88):
  the base is the merge base with the rewritten `main`, so `main`'s own four rewritten commits
  counted as the session's. Their branches in the checkout hold nothing of theirs; the owner deletes
  them.
- s0008 and s0009 each wrote an idea file, said so without committing, and were ended; the `none`
  outcome removed both trees and the files with them. Refiled from the session logs; the hazard is
  `docs/ideas/2026-10-31-ending-a-session-with-uncommitted-changes-loses-them.md`.
- The `@docs/HANDOFF.md` import not loading in the session copy repeats from 2026-10-24, and the
  session told the owner it had no handoff. It is the same gap, now seen from the live form.

Not verified here: the setup lines (including any Manage Roles `fix`), the Discord side of every
exchange, the review and the cherry-pick (the owner's and the runner's report), and the hash
rewrite itself (the reset was done before this read).
