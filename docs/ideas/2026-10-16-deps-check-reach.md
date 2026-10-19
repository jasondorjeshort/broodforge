---
date: 2026-10-16
kind: chore
---

# What the dependency check does not reach yet

Anchor: the group F section of `.claude/hooks/test-hooks.js` and `.claude/hooks/fitness.json`
(ADR 0013). The check passes today and still has gaps, each a way a dependency could enter
unseen:

- Files outside the `paths` globs. Patterns take `/`-separated segments with `*` and `?` only (no
  `**`), so a hook in a subdirectory, a `.cjs` or `.mjs` file, or a script outside the two named
  places is not scanned.
- Dynamic loading. Only a literal `require('<m>')` counts; `import()`, `import ... from`,
  `createRequire` and a `require` built from a variable pass unseen.
- Unknown keys under `deps` pass silently, so a misspelt `vendored` or `budget` limit is ignored
  rather than reported.
- No licence check on direct dependencies. While the budget is 0 there are none; the first one
  admitted needs its licence checked (ADR 0012's checklist), and the check only tests vendored
  files for an SPDX line.

Each is cheap to close when the first real need appears; none blocks anything while the budget is 0.

Question: close these gaps now, or when the first direct dependency is admitted? All four still stand in `.claude/hooks/fitness.json` and group F, and none matters while the budget is 0. I recommend one cheap pass now for unknown keys under `deps` alone (a misspelt limit silently ignored is the gap that hides a mistake), and `Waits: the deps budget rises above 0` for the rest.
Answer:
