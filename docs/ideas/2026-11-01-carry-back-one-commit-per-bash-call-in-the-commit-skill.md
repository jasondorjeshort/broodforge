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

Plan: `coder` (a settled docs edit), one pass, in this project first as a departure, then Sent with the `carry-back`
skill. Files: `.claude/skills/commit/SKILL.md` (Commit via the runner, step 3: the brief tells the runner to make each
commit its own Bash call, the adds then the commit, nothing chained; keep the rule that a message with an apostrophe,
backtick or several paragraphs goes through `git commit -F <file>`). Do not assert in the skill whether a heredoc
tolerates apostrophes: observe it again on the PC first (quoted, unquoted and command-substitution heredocs passed them
intact in the reviewer's test, against `docs/hatchery/reference/claude-code-on-windows.md`, Shells, which says the
opposite) and, if the reference is stale, fix it in place and bump `verified:` (a base-owned file, so it is a departure
and part of the Send). `.claude/agents/runner.md` is edited only for a one-line pointer to the skill; it is base-owned,
so that edit and each one above is a departure row either way, in `docs/design/decisions/0024-decanted-from-hatchery.md`.
The row's carry-back cell names this idea file, which stays until the Send lands; delete it then. Verify: the skill reads
clean; `node .claude/hooks/test-hooks.js` only if a hook file is touched (none expected). Out of scope: any hook change.
