---
date: 2026-10-23
status: resolved
area: engine / sessions
---

# Session settings sources drop the project's CLAUDE.md and agents

## Question

Do the flags Broodforge passes to a session (`--setting-sources user`, ADR 0020 Settings sources,
the `Args` in `internal/engine/claudecode.go`) still let the project's `CLAUDE.md`,
`.claude/agents/` and `.claude/rules/` reach the session?

## What was tried

A throwaway project outside the repository (`git init`) holding `CLAUDE.md` with a canary rule word,
`.claude/agents/canary-agent.md`, a path-scoped `.claude/rules/canary.md` and
`.claude/settings.json`. From that directory, on Claude Code 2.1.309, `claude -p --permission-mode
default --output-format text` was run twice, each asking for the canary word, the launchable
subagent types and the rule word:

- A: with `--setting-sources user`.
- B: without it.

## Result

- A loaded none of it: no canary word, `canary-agent` absent, only the built-in types (claude,
  Explore, general-purpose, Plan, statusline-setup).
- B returned the canary word and listed `canary-agent`.
- The path-scoped rule word was reported in neither run, but the prompt touched no matching file, so
  rules are unverified.
- B also printed that the `permissions.allow` entries of the project's `.claude/settings.json` were
  ignored because the workspace had not been trusted interactively; the dialog cannot be accepted
  by the bot.

Consequence: a Discord session sees none of the project's agreement. The first live session (s0005)
had no delegation, review, cap or role rules and used no subagent (see
`2026-10-23-first-live-session-commits-reached-main-unreviewed.md`).

## Next

Resolved as a fact; the design is open. See the idea
`docs/ideas/2026-10-23-sessions-load-none-of-the-projects-instructions-agents-or-hooks.md`.
