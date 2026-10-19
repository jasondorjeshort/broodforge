---
date: 2026-10-13
kind: chore
---

# Let the Haiku runner escalate to Sonnet only on failure

In a downstream build-phase project the Haiku runner executes sweeps and escalates to Sonnet only when a sweep fails, which
keeps the common case cheap. The runner definition (`.claude/agents/runner.md`) is where the model
is set, and the escalation rule would sit beside it.

Recorded 2026-10-11 from a downstream build-phase project; a candidate once proposal 0003 (`docs/design/proposals/0003-wave-workflow.md`) is decided.

2026-10-13, owner: the same shape may fit the `pusher` (`.claude/agents/pusher.md`, Sonnet today).
A push on the owner's explicit ask is mechanical; a Haiku pusher that escalates when a pre-push
check is ambiguous (secrets, unclear test output) could be cheaper than the main session doing it.

Question: hold this until proposal 0003 is decided, and should the pusher variant (a Haiku pusher escalating to Sonnet) be weighed in the same pass? It changes what each push and sweep costs, which is yours to call. I recommend `Waits: proposal 0003 decided`, with the pusher variant folded into that pass rather than tried alone, since a cheaper pusher misreading a secret is the expensive failure.
Answer:
