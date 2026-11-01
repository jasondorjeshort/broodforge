---
date: 2026-11-01
kind: chore
related: [2026-11-01-carry-back-one-commit-per-bash-call-in-the-commit-skill.md]
---

# Three base files give a stale reason for avoiding heredocs

Three base-owned files say to avoid heredocs because "apostrophes break them": `.claude/agents/architect.md`
(line 17), `.claude/agents/critic.md` (line 17) and `.claude/skills/proposal/brief-critic.md` (line 22).
Observed 2026-11-01 on bash 5.3.15: a heredoc passes apostrophes and backticks intact. The real reasons
are that a chained line fails at parse time and that the Bash tool halves `\\`; the reference states them
(`docs/hatchery/reference/claude-code-on-windows.md`, Shells, fixed in 1531c05).

The advice (write such files with the Write or Edit tool) stays; only the reason changes. Each edit is a
departure row in ADR 0024 and a carry-back, because the files are base-owned. It overlaps the carry-back
idea for the commit skill, which names the same reference; this one is only the three stale sentences.

Anchor: the three files above; `docs/design/decisions/0024-decanted-from-hatchery.md` (the departure table).

## Open

- Edit the three together in one chore commit, with one departure row each, and send one carry-back.
