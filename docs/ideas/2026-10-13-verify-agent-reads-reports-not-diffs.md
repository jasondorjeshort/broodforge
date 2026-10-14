---
date: 2026-10-13
kind: question
---

# Decide whether a verify agent runs beside the reviewer

A downstream build-phase project runs a separate `verify` agent over each change, which runs the thing rather than reading
the diff, and the coordinator reads that agent's report rather than the diff. Hatchery now has a
`reviewer` agent (`.claude/agents/reviewer.md`) and a one-review rule in CLAUDE.md, so the reviewer
reads the diff and no step runs the change.

## Open

- Whether a separate verify agent, which runs the change and does not read the diff, is still
  wanted beside the reviewer.

Recorded 2026-10-11 from a downstream build-phase project; a candidate once proposal 0003 (`docs/design/proposals/0003-wave-workflow.md`) is decided.

Waits: the architect's revision of proposal 0003 (owner, 2026-10-14). That revision decides
whether a verifier runs the change in the build phase and how it splits work with the reviewer
(the 0003 critic review proposes: verifier is the one review per PR; the Opus reviewer is added
for a PR that implements a major choice or carries an "Interacts with:" line), amends ADR 0007 in
one sentence, and deletes this file in the same commit.
