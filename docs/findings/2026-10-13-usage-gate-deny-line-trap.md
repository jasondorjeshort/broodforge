---
date: 2026-10-13
status: resolved
area: process / usage gate
promoted_to: docs/design/decisions/0006-hook-policy-file.md
---

# The deny line was a ceiling the owner never asked for, and nothing could lift it mid-session

## Question

The owner said on 2026-10-10 (downstream) and again on 2026-10-13 (here) to run past the weekly
meter into overage credits. The gate kept refusing launches anyway, in both repositories, and each
workaround moved the wall rather than removing it. Why, and how was the trap built?

## What was tried

- Downstream: `DYSTOPO_GATE_OVERRIDE=1` in the environment. Lifts HOLD, but a running session
  cannot gain a variable, so the session had to be restarted.
- Base: `~/.claude/usage-gate-override.json`. Lifts HOLD only; the base denied at 80 on a weekly
  meter at 96, so the file did nothing.
- Setting `deny` to 100 (downstream T94, base commit d2f415c). The loader caps thresholds at 100 and
  a meter saturates at 100, so this trips exactly when overage starts, with no lift.

## Result

Four causes, in order:

1. ADR 0006 made `deny` "a ceiling no override lifts" as a technical decision. Whether overage is
   acceptable is a question about the owner's money, not a technical one, and it was never asked.
   `CLAUDE.md` now says decisions about what the owner pays need the user (commit 5632bef).
2. The escape hatch was scoped to HOLD by design, so it could not reach the line that bit.
3. "100 means off" was false: the validator caps at 100 and readings saturate there. `deny` now
   accepts `null`, the base default, meaning no deny line (ADR 0006, living key table).
4. The critic step was skipped for ADR 0006 (critic review of proposal 0005, objection 3), and the
   critic brief never asked who can lift a refusing mechanism. `brief-critic.md` now asks, for every
   mechanism that refuses or fails closed: who lifts it, from where, while a session runs, and
   whether the refusal says so.

The same question, asked of the rest of the gate and guard, found siblings. Each is an idea:
`docs/ideas/2026-10-13-usage-gate-stale-deny-not-liftable.md`,
`docs/ideas/2026-10-13-override-file-is-machine-wide.md`,
`docs/ideas/2026-10-13-malformed-policy-falls-back-silently.md`. The `--no-verify` deny with no
owner override is already named as an accepted residual risk in the 0005 critic review.
