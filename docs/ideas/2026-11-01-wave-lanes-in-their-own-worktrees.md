---
date: 2026-11-01
kind: chore
---

# Wave lanes in their own worktrees

Recorded from the owner's conversation of 2026-11-01 as a moonshot, not sanity checked: nobody has
read the anchor against the claim yet.

In the wave of 2026-10-31 two coders shared one working tree. Each saw the other's half-built
packages fail its `go test ./...`, every runner commit had to stage by exact name past the other's
files, and the texts fixes could not be amended into their commit because a revert had landed on
top. Each lane would get a worktree of its own (the Agent tool's worktree isolation, or `git
worktree add` under the scratchpad), run its checks against only its own edits, and the runner
would land each lane in order onto the main tree with one commit per lane. The bot's session model
already does this per session; the coordinator's own waves do not. A Hatchery practice change.

Anchor: `.claude/skills/brief/SKILL.md` (where a lane's isolation would be named); `.claude/skills/commit/SKILL.md`
(the runner lands a lane); `.claude/hooks/git-guard.js` (whether `git worktree add` under the
scratchpad is allowed to a coder, and a commit inside it to the runner only).

## Open

- The git-guard's commit rule matches the command, not the repository (`2026-10-31` finding in the
  run skill): a lane worktree's commits are the runner's, so the landing is a cherry-pick or a
  patch apply by the runner, not a merge by the coder.
- Review reads the lane's diff before the landing, so a landed lane is already reviewed.
