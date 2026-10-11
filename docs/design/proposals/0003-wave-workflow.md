# 0003. Wave workflow for the build phase: task board, worktrees, verifier, sweeps, release

Date: 2026-10-11
Status: draft

Source: the Dystopo project (https://github.com/jasondorjeshort/dystopo), which ran twelve waves,
83 tasks and 78 pull requests this way between 2026-10-01 and 2026-10-11 with one coordinator
session. Its `CLAUDE.md` section "Coordinating a wave" and `docs/TASKS.md` are the as-built record.

## Question

Hatchery's working agreement covers the design phase and single-branch work: the runner commits,
the pusher pushes, agents leave a clean tree. Once a project has code and several features in
flight at once, how is the work organised so that five to eight implementers run in parallel
without touching each other's files, every change is verified by a second model before it lands,
and the coordinator can resume after an abrupt stop? This proposal records the model Dystopo
converged on so a decanted project can adopt it when it reaches the build phase.

## Constraints that matter here

- Agents write all code and debug it; the coordinator plans, briefs, reviews and merges.
- Usage gating: the plan must survive a stop at the gate mid-wave and resume next session.
- One owner, one coordinator session; verification must not depend on the owner reading diffs.
- The debug interface rule: every feature is agent-drivable, so verification is a script, not a look.
- Privacy and licensing rules are enforced by hooks and scanners that must see PR text as well as commits.

## Options

### A. Single branch, runner commits (Hatchery today)

Agents leave a clean tree; the runner commits; the pusher pushes. Works for one or two agents in
sequence. Breaks with parallel implementers: two agents editing the same tree collide, nothing
verifies a change before it lands, and a half-finished change blocks the branch for everyone.

### B. Waves of owned tasks in worktrees with a verifier per PR (Dystopo)

- A wave is five to ten tasks written on a board (`docs/TASKS.md`) before launch. Each task has a
  Goal, an Owns line (the paths it may edit, made disjoint from every other task in the wave before
  launch; two tasks that must touch one file go in different waves or one records a follow-up), and
  a Done-when line with measurable checks.
- Each implementer (coder model) runs in its own git worktree on branch `task/Tnn`, debugs end to end
  before pushing, updates the docs its change touches in the same PR, and opens a pull request whose
  body states the key runs with numbers, follow-ups, and every edit outside its Owns line.
- A verifier (a cheaper model) per PR: rebases onto main if it moved, runs the checks, repeats the
  PR's key runs, views the images, scans the PR text, reviews the code, and returns MERGE, MERGE
  WITH NOTES or FIX with specifics. Small fixes go back to the implementer; the verifier never edits.
- The coordinator merges (squash), writes a dated log entry (what was built, the numbers, the
  follow-ups) and a Status line on the board through one helper script, then deletes the branch and
  the worktree. Post-wave: one command runs every sweep (tests, tours, device, release smoke) and its
  findings become the next wave's sweep task; then a release is tagged and published.
- Resuming: the first wave on the board without Status lines is the open wave; `gh pr list` shows
  what is in flight; the log tail says what happened last.

### C. GitHub Issues and feature branches without a board

Each task an issue, each PR linked to it. Rejected for the base: ADR 0002 keeps in-flight work in
files for one owner; issues add a network dependency and a tool call per lookup, and the Owns line
(the thing that prevents collisions) has no natural home in an issue.

## Recommendation

B, as a second phase of the working agreement that a decanted project switches on when it has code:
a `wave` skill (write the board, make Owns lines disjoint, launch, verify, merge with the helper,
sweep, release) and a `verify` agent definition (Sonnet-class) beside the existing six. The
single-branch rules stay for the design phase and for docs-only sessions. Concretely:

1. Add `docs/TASKS.md` to the layout (board; exempt from the 300-line cap like proposals, or split
   per wave) and a `merge-log` helper that writes the log entry, the Status line and the changelog
   line together, so the record is never half-updated.
2. Agents in worktrees commit and push their own task branch; rules 3 and 4 of `git-guard` (only the
   pusher pushes, only the runner commits) therefore apply to the single-branch phase only, keyed off
   a project setting or the branch name (`task/*` allowed for the implementer that owns it).
3. The verifier is the only reviewer the coordinator trusts before merging; the coordinator reads
   the report, not the diff, and spot-checks the delta after a fix round.
4. Post-wave sweep as one command with a JSON summary and a contact sheet; a Haiku-class runner
   executes it and a Sonnet pass is escalated only on failure or when an image must be judged.
5. Publish after every green sweep (tag-driven draft, verify the artifact downloaded from the draft
   itself, hand-written notes, then publish); version strings derived from the tag, never typed.

First step once promoted: write the `wave` skill from Dystopo's "Coordinating a wave" section and
the `verify` agent from the verifier briefs there; port `tools/merge-log.py`.

## What would change this recommendation

- A second human contributor or public bug reports: issues become intake (ADR 0002 trigger) and the
  board becomes a view over them.
- A hosted CI that runs the device sweeps: the local one-command sweep becomes the CI job and the
  coordinator stops running it by hand.
- Evidence that verifier reports miss defects the owner later finds: add a second verifier model or
  an architect review for large PRs.

## Claims to verify

- Dystopo's numbers above: `docs/PLANNING.md` and `docs/TASKS.md` in that repository.
- `git worktree` on Windows: long paths block `git worktree remove`; see finding
  `2026-10-11-windows-worktrees-long-paths.md`.
- Hooks fire inside subagents (ADR 0002 relies on it), so `git-guard` rule 3 would block an
  implementer pushing its own task branch unless keyed as in item 2.
- The verifier model: Dystopo used Sonnet 5.5 for every PR and found real defects (Android 8 crash,
  redirect hole in an updater, a leak) that the implementer's own runs missed.

## Critic's objections and responses

(none yet)
