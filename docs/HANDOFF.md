# Handoff

Coordinator state. Subagents follow their brief, not this file.
Updated: 2026-10-11, at commit cf4b22b, by main session (Dystopo coordinator, co-authoring)

## Now

- Nothing in flight. The Dystopo coordinator session added, from twelve waves of experience:
  proposal `docs/design/proposals/0003-wave-workflow.md` (draft, build-phase workflow), three
  resolved findings and `docs/reference/claude-code-on-windows.md`, and backlog items for the
  skills and hooks Dystopo has that the base lacks.
- Dystopo adopted from here: `docs/HANDOFF.md` imported by its `CLAUDE.md`, and the usage gate
  hook at its own threshold. Dystopo is the first live test of the base's ideas on a build-phase
  project; its divergences are recorded in its `docs/PLANNING.md` ("Process base").

## Next

1. Critic pass on proposal 0003 (brief from `.claude/skills/proposal/brief-critic.md`), then the
   architect revises and promotes it to ADR 0003; this decides the `wave` skill, the `verify` agent,
   the phase keying of git-guard rules 3 and 4, and the usage-gate override for in-flight work.
2. Write the `wave`, `release` and `sweep` skills from Dystopo's `CLAUDE.md` once 0003 is accepted.
3. Decide whether the first decanted app is Dystopo retrofitted (update-from-hatchery against it)
   or a new project; Dystopo's layout (`docs/TASKS.md`, `docs/PLANNING.md` as a dated log over 300
   lines) differs from ADR 0002 and would need recorded exceptions.

## Waiting on user

- Whether to run the critic pass now (plan usage is above the gate this week) or after the reset.

## Open findings

- `docs/findings/2026-10-11-bash-heredoc-apostrophes.md`
