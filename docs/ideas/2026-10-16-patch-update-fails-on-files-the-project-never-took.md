---
date: 2026-10-16
kind: bug
related: [docs/ideas/2026-10-13-first-sync-with-the-build-phase-sibling.md]
---
# Patch-mode update fails outright when the diff touches a file the project never took

`.claude/skills/update-from-hatchery/SKILL.md`, the patch command in step 4: `git diff <base>
hatchery/main -- .claude CLAUDE.md | git apply --3way`. An adopted project takes only some of the
base's `.claude/` files (ADR 0004 lets it leave out the agents, `commit`, `triage`, the brief
template). When the base has since changed one of the files it left out, `git apply --3way`
prints `error: <path>: does not exist in index`, exits 1 and applies nothing, including the hunks
that would have applied cleanly. Reproduced 2026-10-16 in a scratch repository by the scan of an
adopted project (`../dystopo`, whose next update fails this way as written).

Fix: the skill builds exclude pathspecs from the adoption ADR's "not taken" list (or `git apply
--exclude=<path>` per file), and says so in the step; the adoption ADR template gains the list in
a fixed form the skill can read. The sibling-sync idea covers the first patch run in general; this
is the one failure that stops it before anything applies.
