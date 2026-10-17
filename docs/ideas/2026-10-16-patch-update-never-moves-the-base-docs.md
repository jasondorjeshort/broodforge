---
date: 2026-10-16
kind: bug
related: [2026-10-16-adopt-step-7-dangling-pointers.md]
---
# Patch-mode update never moves the base-owned docs that adopt copied

`.claude/skills/update-from-hatchery/SKILL.md` step 4 diffs only `.claude CLAUDE.md LICENSES`.
`adopt-hatchery` step 7 copies base-owned docs into the project: ADRs 0001 and 0002, the hook
specs, the design, proposals and ideas READMEs (the ideas README is the base's by ADR 0007). Those
files never move forward under a patch update. Observed 2026-10-16 in an adopted project at base
d64ead2: after a simulated update its new `usage-gate.js` and `statusline.js` headers name
`0002-hook-specs-usage.md`, which the project lacks, while its copy of `0002-hook-specs.md` still
describes the old cache.

Fix: step 4 adds the paths adopt step 7 lists as base-owned (the two ADRs, the hook specs, the
READMEs it empties) to the patch pathspec, with the index READMEs handled as the project's rows
plus the base's prose; or adopt step 7 records the list in the adoption ADR and step 4 reads it.
