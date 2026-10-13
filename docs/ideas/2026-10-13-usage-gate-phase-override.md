---
date: 2026-10-13
kind: question
---

# Decide whether the usage gate needs a phase or an override for in-flight work

A downstream build-phase project ran its gate at 75%, with the owner's clarification that only new
implementers stop there while in-flight verification and merges finish. Hatchery's gate thresholds
live in `.claude/hooks/policy.json`, and they are untested on a build-phase project. Decide with
proposal 0003 whether the gate needs a phase or an override for in-flight work; that project uses
an environment variable the coordinator sets for this.

## Open

- Whether a phase setting or an environment override fits better.
