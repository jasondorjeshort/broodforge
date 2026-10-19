---
date: 2026-10-18
kind: feature
related: [2026-10-13-verify-agent-reads-reports-not-diffs.md]
---

# A critic pass on a planned batch of briefs before launch

Proposed by a downstream build-phase project in the description of the first carry-back (2026-10-18),
where it has run before every batch since 2026-10-16. Before launching several briefs at once,
the coordinator runs the written plan past a critic (the most capable model, read-only) whose
brief names the plan, the previous batch's outcomes and the records the plan draws on, and asks
for numbered objections with severity and evidence on: disjoint ownership over files that exist,
task size, every done-when a command or an observable, premises checked against the code, major
choices that need a proposal, follow-ups dropped silently, and sequencing between tasks. Every
objection is answered by an edit to the plan or a written reason in it; launch follows the edits;
a blocking objection sends the plan back. The sibling's latest pass found no blocking, seven
serious and six minor objections (two files with two owners, a void sequencing rule, a decision a
task would have made in passing), all taken into the plan.

The base has no batch skill, so the step would sit in the delegation agreement or the `brief`
skill. A protocol change is a proposal here, never landed from a branch (ADR 0019, item 1); the
cost is one Fable or Opus pass per batch, which is the owner's call under `usage-gating.md`.

Question: do you want a critic pass before each batch of briefs, at one Fable or Opus pass per batch? It is a protocol change and a recurring cost, both yours; the sibling's evidence (seven serious objections taken on its latest batch) is strong. I recommend yes, through a proposal (`proposal` skill) for an Opus critic on batches of three or more briefs, with the step in the `brief` skill, so single launches stay cheap.
Answer: 2026-10-19, yes, through a proposal: an Opus critic on batches of three or more briefs, the step in the brief skill.
