---
date: 2026-11-08
kind: chore
related: [2026-11-08-a-concurrency-pass-over-the-bot.md]
---

# `general-purpose` subagents on a cheaper model in sessions

`/usage` on 2026-11-07 put 64% of the last day's subagent cost on `general-purpose` ("Give
general-purpose subagents a cheaper model"), against 9% on `coder`. The repository's own agents pin
their models (`.claude/agents/*.md`, the routing table); `general-purpose` is the engine's built-in,
pinned to nothing, so it runs on the coordinator's model. The bot's sessions reach for it because the
workflow copy's coordinator has no named agent for "run this and report" and the live spike runs used it
by name (the P40e session's subagent was `general-purpose`).

Two levers, both small: the workflow copy passes `--agents` with a `general-purpose` entry that pins a
model (Sonnet or Haiku) and a tool list, so a session's unnamed launches stop costing Opus; and the
session header steers the coordinator to the named agents for anything that is not a one-shot read
(the `runner` for mechanical work, the `tester` for scoped edits). The pin is per project and per
channel, so a project that wants Opus everywhere keeps it.

Anchors: `internal/session/turn.go` and the spawn's `--agents` (ADR 0025, the agents copied from
`.claude/agents/`), the session header (`04-ux-session-start.md`), `.claude/agreement/model-routing.md`
(base-owned: a sentence there is a carry-back).

## Open

- Whether the engine honours a `model` on a `general-purpose` override at all, or only on new names; one
  cheap real turn settles it.
- The usage breakdown counts this machine's sessions only; the droplet's share is unknown.
