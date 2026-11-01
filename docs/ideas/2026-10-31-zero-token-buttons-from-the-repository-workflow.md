---
date: 2026-10-31
kind: feature
related: [2026-10-14-ux-choices-that-could-become-options.md, 2026-10-31-preview-this-branch-in-the-game.md]  # the first lists venue options; the second is one action this would carry
---

# Zero-token buttons from the repository workflow

Recorded from the owner's conversation of 2026-10-31 as a moonshot, not sanity checked: nobody has
read the anchor against the claim yet.

A project declares fixed actions in its own workflow files, which ADR 0025
(`docs/design/decisions/0025-session-workflow-composition.md`) already copies into
every session: run the tests, build, lint, preview. Each shows as a button in the venue, runs
under the session's reach in the session's worktree, and posts its output, with no engine turn
and no tokens. The thousand-player tier of `00-vision.md` checks their work for free, and a
driver who is away for a day can see the tests pass without waking the agent.

Anchor: `internal/workflow/copy.go` (the per-session copy of the repository's workflow, where an
actions file would be read); `internal/discord/customid.go` (one custom id per action);
`docs/design/decisions/0025-session-workflow-composition.md` (the workflow composition this extends).

## Open

- A repository's action runs a command the repository wrote; it runs with the session's reach, the
  same as the agent's Bash, and the hook's path checks must apply to it.
- Output goes to the venue or to the agent's next prompt or both.
- Whether an action may run while a turn is running, on the same worktree.

Question: Is this wanted, and as a proposal, since it runs repository-authored commands under the session's reach?
`internal/workflow/copy.go` renders the per-session copy of the workflow (`Render`, extracting listed files into the
session), so an actions file would be read there, but a command the repository wrote running with no engine turn
is a security surface. Recommend a proposal after the landing passes.

Answer:
