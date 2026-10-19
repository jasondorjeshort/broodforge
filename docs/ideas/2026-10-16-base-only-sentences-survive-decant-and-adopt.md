---
date: 2026-10-16
kind: chore
related: [2026-10-16-adopt-step-7-dangling-pointers.md]
---
# Base-only sentences in CLAUDE.md and the agents survive decant and adopt

Seen 2026-10-16 in three siblings, one decanted and two adopted:

- `CLAUDE.md` Phase section ("Design first. No application code until...") and the line
  "Designing the base is itself the project" are unchanged in a decanted project that has Go code
  committed and a Go rule; a coder briefed for the next pass reads a ban on what it is asked to do.
- `CLAUDE.md`, `coder.md`, `tester.md`, `critic.md`, the `debug-interface` skill and the proposal
  template name the debug-interface and platforms design docs (`05-` and `06-` under
  `docs/design/`, not yet written here either); an adopted project has neither and its design
  README says so. [no-check: the two docs do not exist yet in the base]
- `CLAUDE.md` cites "ADR 0006", "ADR 0007", "ADR 0012" bare. An adopted project keeps its own
  numbering (its next ADR is 0004), so its own future 0006 takes over the citation.
- `adopt-hatchery` step 7 copies neither `docs/reference/claude-code-on-windows.md` nor its index
  row, yet `rules/docs.md`, the `brief` and `commit` skills cite it and the owner is on Windows.

Fix, in the `decant` and `adopt-hatchery` skills and the `CLAUDE.md` wording: the Phase section is
a slot both skills rewrite (decant to "build" with the project's plan named; adopt to the project's
phase); base-only lines are marked as such or dropped by the skills; design-doc references say
"where it exists"; base ADR citations in living docs read "Hatchery ADR NNNN" (the `adr` skill
says so for adopted projects); adopt step 7 copies the Windows reference and its row.

Plan: `tester`, the residue only; check each item against the tree first. Done since filing: the Phase section is a slot both skills rewrite (`decant` step 2, `adopt-hatchery` step 6). Left: (1) `.claude/agents/coder.md`, `tester.md`, `critic.md`, `.claude/agreement/debug-interface.md` and `major-decisions.md` name `docs/design/05-debug-interface.md` and `06-platforms.md` without "where it exists"; add it. (2) Bare base ADR citations in `.claude/agreement/*.md` become "Hatchery ADR NNNN" or carry the `docs/hatchery/decisions/` path, so an adopted project's own numbers never take them over. (3) `adopt-hatchery` step 7 copies `docs/reference/claude-code-on-windows.md` and its index row where the project lacks them. Verify: `node .claude/hooks/test-hooks.js` and `node .claude/hooks/policy.js --claude-check`. Out of scope: "Designing the base is itself the project" (already qualified for projects).
