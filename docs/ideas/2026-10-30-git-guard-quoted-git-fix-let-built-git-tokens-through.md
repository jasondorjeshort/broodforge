---
date: 2026-10-30
kind: bug
related: [2026-10-21-carry-back-quoted-git-does-not-name-git.md]
---

# The quoted-git fix had a gap: a git token built in a value reached `git push -f`

Anchor: `namesGit` in `.claude/hooks/git-guard.js`, and this project's commit `5288955`.

The quoted-git fix (`5288955` here, dropped at the update to the base `dc896c3`, now carried back as
`carry-back/git-guard-quoted-git` at `fffd564`) had a gap the carry-back review found. A git token
built inside an assignment's quoted value, a setter command's argument (`read`, `printf -v`,
`Set-Variable`), a heredoc or here-string body, or a captured substitution (`g=$(...)`) reached
`git push -f` unseen. Forms:

- `read -r g <<< git; $g push -f`
- `g="${x}git"; $g push -f`
- `g="x git"; ${g#x } push -f`
- `g='git.cmd'; $g push -f`
- PowerShell: `$g = 'x git'.Substring(2); & $g push -f`

The branch's `namesGit(cmd, captured)`, with harness cases `GR24l` to `GR24z`, closes it.

The point of this idea: any re-graft of the project's own fix, and its return by
`update-from-hatchery` once Hatchery lands the branch, must take the extended version, never
`5288955`. Until then main runs the base guard with the false denies ADR 0024 records.

Differs from `2026-10-21-carry-back-quoted-git-does-not-name-git.md`: that idea sends the fix; this
one records the gap in the original fix and which version must come back.

Question: Keep this file as a `Waits:` on Hatchery landing `carry-back/git-guard-quoted-git` (only the
owner writes a `Waits:`), or move the "never `5288955`, take the extended `namesGit(cmd, captured)`"
rule into the `git-guard.js` departure row of `docs/design/decisions/0024-decanted-from-hatchery.md`
and delete this file? Recommendation: the row, then delete.
Answer:
