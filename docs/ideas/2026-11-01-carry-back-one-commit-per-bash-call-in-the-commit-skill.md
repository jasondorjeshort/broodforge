---
date: 2026-11-01
kind: chore
---

# Carry back: one commit per Bash call in the commit skill

Recorded from the owner's conversation of 2026-11-01, not sanity checked: nobody has read the
anchor against the claim yet.

On 2026-10-31 a runner chained 21 commits into one Bash command and a quoting error stopped it at
parse time after the first; the fix was a resumed runner told to make each commit its own call. The
rule belongs in the commit skill, not in each brief: a runner makes one commit per Bash call (the
adds, then the commit, nothing chained), and a message with an apostrophe, a backtick or more than
one paragraph goes through a file. The same brief also claimed the messages held no apostrophes
when several did; the skill should say that double-quoted `-m` arguments tolerate them and heredocs
do not. Base-owned, a Hatchery carry-back.

Anchor: `.claude/skills/commit/SKILL.md` (Commit via the runner, step 3); `.claude/agents/runner.md`.

## Open

- Whether the rule is a sentence in the skill or a line in the runner's own rules.
