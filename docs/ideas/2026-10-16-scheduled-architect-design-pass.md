---
date: 2026-10-16
kind: question
---

# A scheduled architect design pass, in addition to the trigger scan

Proposal 0012 first proposed a standing design pass: once per chunk of build work, before its
briefs, the architect (Fable) re-reads every ADR's deferred items and predictions, the last
fitness failures and the untriaged ideas, and writes at most one proposal or idea. The critic
(`docs/design/proposals/0012-critic-review.md`, objection 7) showed the cost was under-stated
(an artefact that is a proposal pulls in a critic, the answers and an ADR: up to four Fable
passes) and the cadence was set before the owner was asked. ADR 0012 replaced it with a scan in
the `triage` skill by a cheaper model, which calls the architect only when a trigger fired, a
prediction is due or a fitness check failed. Anchor: `.claude/skills/triage/SKILL.md`, step 1.

Question: do you also want a scheduled architect pass, run on a cadence you name (per session,
per week, per release), that reads the same inputs plus the untriaged ideas and may write one
proposal even when no trigger fired? Cost per run: one Fable pass bounded by one artefact, and
up to three more when the artefact is a proposal. Recommendation: no for now; the scan covers
every input a program or a cheaper model can read, and the one thing it cannot do (notice a
design problem nobody wrote a trigger for) is what your own reading of `docs/ideas/` in one
sitting does (`docs/ideas/README.md`, What drains it). Revisit if a project with code shows
decay the scan did not catch.
Answer:
