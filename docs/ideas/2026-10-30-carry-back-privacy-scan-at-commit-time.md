---
date: 2026-10-30
kind: chore
related: [2026-10-21-carry-back-main-session-push-runs-check-5-read.md]  # that one is the read at push; this moves the mechanical scan earlier
---

# Carry back: the privacy scan runs at commit time, not only at push

The push scan refused a fixture placeholder (`C:/Users/<user>` written with a real name) that a
runner had committed hours earlier, and the fix was a fixup rebase of an unpushed commit. A commit-time
run of `privacy-scan.js --text` over the staged files catches the string when it is committed, while
the fix is still an edit. Base-owned, so it goes back to Hatchery as a carry-back.

Anchor: `.claude/hooks/privacy-scan.js` and `.claude/hooks/git/` (a `pre-commit` beside `pre-push`),
or the runner's commit step in `.claude/skills/commit/SKILL.md`.

Plan: tester, then the carry-back Send (`.claude/skills/carry-back/SKILL.md`). Hooks are not copied:
`policy.json` `git.hooksPath` is `.claude/hooks/git` and each clone runs `git config core.hooksPath
.claude/hooks/git` once (the header of `.claude/hooks/git/pre-push`), so a `pre-commit` file there
reaches every clone with no install step; it must be executable in git.
- Add `.claude/hooks/git/pre-commit` (POSIX sh, SPDX header, like `pre-push`): for each staged file
  (`git diff --cached --name-only --diff-filter=ACM`) run `node "$top/.claude/hooks/privacy-scan.js" --text <file>`,
  stdout and stderr to stderr, exit 1 on any hit with the same `HIT` line format the push scan prints.
- `node .claude/hooks/test-hooks.js` gains a case beside PV22 (scratch repository, hooks copied in): a commit
  of a file holding a refused string is refused, a clean commit goes.
- Add a departure row to `docs/design/decisions/0024-decanted-from-hatchery.md` (base-owned `.claude/hooks/git/`),
  then the carry-back Send. Out of scope: the runner's commit step, `privacy-scan.js` itself, `pre-push`.
- Verify: the scratch-repository commit is refused; `test-hooks.js` passes.
