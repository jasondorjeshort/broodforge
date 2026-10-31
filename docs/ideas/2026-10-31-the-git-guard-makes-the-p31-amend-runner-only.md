---
date: 2026-10-31
kind: chore
---

# The git-guard makes the P31 amend runner-only, in a session worktree too

The `git-guard` hook's commit rule (`.claude/hooks/policy.json`, `git commit` only from the
`runner`) matches the command, not the repository, so a `tester` running the `run` skill cannot do
the P31 check's `git -C <tree> commit --amend --no-edit` in a session worktree under the dev root
(2026-10-31, `s0005`: `commit not permitted for this caller`). The drive had to split: tester for
the session, runner for the amend and the rest. Evidence:
`docs/findings/2026-10-31-milestone-1-reached-a-channel-200-session-committed-was-reviewed-and-landed.md`.

Anchor: `.claude/hooks/git-guard.js` and `policy.json` (base-owned; a fix is a departure row and a
carry-back), or `.claude/skills/run/SKILL.md` step 4 saying the amend is the runner's.

## Open

- Scope the commit rule to this repository's tree (a commit whose `-C` or cwd is outside `<repo>`
  is not a commit to the project), or document the split in the skill. The skill line is the
  smaller change and needs no departure.
