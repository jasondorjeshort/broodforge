---
date: 2026-10-16
status: resolved
area: process / decision-making for long-lived software
promoted_to: docs/design/decisions/0012-decision-practices-for-long-lived-software.md
---

# The cost of agent-built software lands in the small changes the loop reviews for defects only

## Question

The owner wants Hatchery to optimise for software that keeps doing its job for years with little
upkeep ("actual software that does things, not churning out code"), not for the first version out
the door. Hatchery's loop reviews the few major choices in depth (proposal, critic, ADR) and the
many small ones once, for defects (ADR 0007). What does the evidence on agent-written code say
about where long-term cost comes from, and what practices from projects that have lasted apply?

## What was tried

Web survey on 2026-10-16 by the coordinator, recorded by the architect from summaries; the critic
then read the primary sources for proposal 0012 (`docs/design/proposals/0012-critic-review.md`,
Claims checked) and the Result below was corrected to what the papers say. No local experiment:
Hatchery has no application code to measure.

## Result

1. Agent-built code is somewhat harder for agents to build on, and the loss is in later work.
   CodeThread (arxiv 2606.21804): two-step chains on SWE-Bench variants with four agents; later
   tasks resolved 4 to 8 percentage points less often on average when the earlier change was an
   agent's, up to 13.1 points for one model on one benchmark in a subset table. Longer histories
   were not tested. The one code-level signal the authors found significant is behavioural:
   drift in input validation and error handling at the first change (gating removed, silent
   fallbacks, changed error paths) and code growth downstream; static complexity and size
   metrics "generally do not explain these differences". Needle in the Repo (arxiv 2603.27745)
   is the one structural study: 21 authored C++ cases, each with an executable oracle written
   for that task's intended solution; functional tests pass where the oracle fails. SWE-CI
   (arxiv 2603.03823) measures regressions over about 71 commits per repository and does not
   analyse causes. The EASE 2026 AIDev study (arxiv 2605.06464) cuts the other way: agent-created
   files received fewer fix commits and less maintenance over six months. Net: the cost lands in
   small, task-sized changes; the measured mechanism is behavioural drift and growth; structural
   decay is shown only where a probe was written to look for it.
2. Projects that lasted name what they promise and keep everything else disposable. SQLite's LTS
   page plans support through 2050; the promised surfaces are the file format and the C API, and
   everything behind them may change; the page warns against following fashion. Go's
   compatibility promise lets the API grow but never break existing code.
3. Ousterhout: complexity is dependencies plus obscurity, and shows as change amplification,
   cognitive load and unknown unknowns. Deep modules (small interface, much behind it) contain
   it; tactical programming accumulates it; the remedy is a continuous 10 to 20 percent of each
   developer's effort on design, not a design phase or a scheduled pass.
4. Boring technology (McKinley): the sound claim is known failure modes and the operational cost
   of each additional tool; Lefkowitz (2024) frames it as boundary tokens, spent when a choice
   crosses into another technology. Neither is a budget of innovation tokens.
5. Dependencies: Cox gives a checklist (design, code quality, tests, issues, maintenance, usage,
   security on untrusted input, licence, its own dependencies), then test, abstract, isolate,
   avoid, upgrade, watch; copying is for small needs. He calls reuse a positive transformation,
   not a liability. The Go proverb: a little copying is better than a little dependency.
6. Hatchery's own record: ADRs 0009, 0010 and 0011 are three compatibility incidents on formats
   nobody had named as promised (the base line, the update mode, the usage cache).

What this means for Hatchery: the major-choice loop covers point 2 for the few choices it sees.
Point 1 says the reviewer should look for drift in input checks and error handling, which it can
do on a diff today. Points 2 to 6 give the practices, which stand on the lasting projects and on
Hatchery's own needs rather than on the 2026 studies. The practices and where each lives are ADR
0012; Hatchery's own surfaces are ADR 0013.

Sources: arxiv.org/abs/2606.21804 (CodeThread); arxiv.org/abs/2603.27745 (Needle in the Repo);
arxiv.org/abs/2603.03823 (SWE-CI); arxiv.org/abs/2605.06464 (EASE 2026, AIDev); sqlite.org/lts.html;
go.dev/doc/go1compat; Ousterhout, A Philosophy of Software Design (2nd ed., via secondary
sources); boringtechnology.club (McKinley); blog.glyph.im/2024/07/against-innovation-tokens.html
(Lefkowitz); research.swtch.com/deps (Cox); go-proverbs.github.io.
