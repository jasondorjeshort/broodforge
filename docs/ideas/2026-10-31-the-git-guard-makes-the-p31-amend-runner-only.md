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

Plan: tester. Decision on the Open item: the skill line, not the guard. `.claude/skills/run/SKILL.md` has no
amend text today (its step 4 is the real engine), so add a short paragraph at the end of step 3, the round
trip: the P31 check (`docs/design/08-implementation-plan-canonical.md`) amends the session's commit with
`git -C <tree> commit --amend --no-edit` and then runs `worktree check`; the amend is the runner's
(guard rule 4), so the drive splits: the tester runs the session and the check, the runner the amend,
each half named. No departure (the run skill is this project's); `git-guard.js` and `policy.json` stay.
- Verify: the skill reads correctly and `node .claude/hooks/test-hooks.js` is untouched.
