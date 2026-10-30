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
