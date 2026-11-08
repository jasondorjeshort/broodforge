---
date: 2026-11-08
kind: question
---

# A rule for the coordinator whose subagents die

In dev-live session s000g (2026-11-02) three `coder` launches ended at their first deferred `Bash`
(the engine, at a hook `defer`: the 2026-10-23 finding, resolved by ADR 0029). The coordinator then
wrote the Bot DMs change itself in the main process, on the owner's word, and a reviewer read it; the
commit (`b15362c`) sits on its session branch. The delegation rule (`.claude/agreement/delegation.md`)
says the coordinator does no coding and, when the usage gate blocks launches, writes the handoff and
stops. It says nothing about launches that start and die: the only two outcomes it knows are "the
agent did it" and "the gate refused it".

The question is which rule to write (base-owned, so a Hatchery carry-back): the coordinator stops after
the second loss, writes a finding with the loss's shape and the handoff, and reports, as it does at
HOLD; or it may code on the owner's explicit word for that session, as happened, with the review still
mandatory and the handoff saying so; or both, the second being the owner's lift of the first. What s000g
showed is that a coordinator left to decide will code, and will do it well enough to pass review, which
is the argument for making the lift explicit rather than forbidding it.

Anchors: `.claude/agreement/delegation.md` (the two outcomes), `docs/findings/2026-10-23-subagents-in-a-session-end-before-writing.md`
(the second occurrence), the session header (`04-ux-session-start.md`: where a session reads the rule).

## Open

- The owner's choice between the three shapes above.
- Whether the bot can see the loss (an `Agent` result of "returned no output" twice in one turn) and
  say so in the venue, so the person is asked rather than the coordinator deciding.
