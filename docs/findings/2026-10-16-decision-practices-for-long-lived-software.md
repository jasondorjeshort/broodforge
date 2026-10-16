---
date: 2026-10-16
status: resolved
area: process / decision-making for long-lived software
promoted_to: docs/design/proposals/0012-decision-practices-for-long-lived-software.md
---

# Complexity in agent-built software accumulates in the small choices the loop does not review for design

## Question

The owner wants Hatchery to optimise for software that keeps doing its job for years with little
upkeep ("actual software that does things, not churning out code"), not for the first version out
the door. Hatchery's loop reviews the few major choices in depth (proposal, critic, ADR) and the
many small ones once, for defects (ADR 0007). What does the evidence on agent-written code say
about where long-term cost comes from, and what practices from projects that have lasted apply?

## What was tried

Web survey on 2026-10-16 by the coordinator; the architect recorded it. Sources read: four 2026
studies of agent-written code (CodeThread, Needle in the Repo, SWE-CI, an EASE 2026 study on the
AIDev dataset), the SQLite long-term support page, Ousterhout's A Philosophy of Software Design,
McKinley's Choose Boring Technology with Lefkowitz's Against Innovation Tokens and a 2025 post
revisiting the idea, and Russ Cox's Our Software Dependency Problem. Figures below are as the
coordinator noted them; the critic verifies them against the sources (proposal 0012, Claims to
verify). No local experiment: Hatchery has no application code to measure.

## Result

1. Agent-built code is harder to build on than human-built code, and the loss is in later work.
   CodeThread (arxiv 2606.21804) reports agents resolving later tasks up to 13.1 percent less
   often when the earlier tasks were done by agents than by humans. Needle in the Repo (arxiv
   2603.27745) defines the failure: code correct now but harder to extend or test, found by
   executable structural probes rather than by reading. SWE-CI reports regressions under
   long-term maintenance; the EASE 2026 AIDev study is the dataset-scale view (figure not noted).
   The common thread: nothing in these failures is a defect at the time of the change, so a
   defect review does not catch it. The signal is structural and must be checked by a program.
2. Projects that lasted name what they promise and keep everything else disposable. SQLite's LTS
   page plans support to 2050; the promised surfaces are the file format and the C API, and
   everything behind them may change; the page warns against following fashion.
3. Ousterhout: complexity is dependencies plus obscurity, and shows as change amplification,
   cognitive load and unknown unknowns. Deep modules (small interface, much behind it) contain
   it; tactical programming (each change the quickest that works) accumulates it; the remedy is
   a standing 10 to 20 percent of effort on design, not a design phase.
4. Boring technology: the sound claim is known failure modes, not a budget of innovation tokens.
   In an agent loop the test becomes whether the reviewing model can verify what was generated
   in that technology; a technology it cannot check line by line is not boring here.
5. Dependencies: Cox treats each one as a liability to evaluate (its code, issues, maintainers,
   transitive closure, and how it would be replaced); the Go proverb says a little copying is
   better than a little dependency.

What this means for Hatchery: the major-choice loop already covers point 2 for the few choices
it sees. Points 1, 3 and 5 are about the many small choices inside briefs, which the base
reviews for defects only, and about structural properties only a program can check. The
practices that follow, and where each lives, are proposal 0012.

Sources: emergentmind.com/papers/2606.21804 (summary of CodeThread); arxiv.org/pdf/2603.27745
(Needle in the Repo); sqlite.org/lts.html; Ousterhout, A Philosophy of Software Design (2nd
ed.); boringtechnology.club (McKinley); blog.glyph.im (Lefkowitz, Against Innovation Tokens);
research.swtch.com/deps (Cox); go-proverbs.github.io.
