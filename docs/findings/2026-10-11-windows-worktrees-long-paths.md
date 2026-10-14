---
date: 2026-10-11
status: superseded
area: tooling / git worktrees on Windows
promoted_to: docs/reference/claude-code-on-windows.md
---

# git worktree remove fails on Windows long paths; an empty robocopy mirror clears the tree

## Question

Dystopo ran one git worktree per agent under `.claude/worktrees/agent-<id>/`. After a task merged,
`git worktree remove --force` failed with "Filename too long" on Gradle and Android build output
nested below the 260-character limit, leaving dozens of gigabytes behind.

## What was tried

- `git worktree remove --force <dir>`: fails on the long paths; the worktree stays registered.
- `Remove-Item -Recurse -Force`: same failure on the same files.
- `robocopy <empty dir> <worktree dir> /MIR` then `Remove-Item -Recurse -Force` then
  `git worktree prune`: clears everything, because robocopy uses the long-path API. A directory
  held open by a lingering Gradle daemon survives one round and goes on the next.
- A worktree marked `locked` by the harness needs `git worktree unlock` first.

## Result

The robocopy mirror is the reliable removal on Windows. Agents that build inside their worktree
should stop their Gradle daemons before reporting, or the coordinator removes the directory on a
later pass. The harness also creates worktrees only when the shell's working directory is inside a
git repository: a `cd` that drifts out of the repo makes the next agent launch fail with "not in a
git repository", so the coordinator resets the directory before launching.
