---
date: 2026-10-13
kind: question
---

# Decide whether the main session may push on the owner's instruction

The Pushing section of `.claude/skills/commit/SKILL.md` and CLAUDE.md's Git section both say only
the `pusher` or the user pushes. Neither lets the main session push when the owner asks for it.
`.claude/hooks/policy.json` does set `git.push.mainSession` to true, so the hook and the documents
disagree.

Question: should the main session be allowed to push on the owner's explicit instruction? If yes,
the commit skill's Pushing section must say so, and CLAUDE.md with it. If no, `mainSession` in the
policy file should be false.

Recommendation: no — keep pusher-only; the owner can run `git push` themselves.

Answer:
