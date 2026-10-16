---
date: 2026-10-16
kind: bug
---
# The update's conflict rule treats mixed files as base-wins and misses three project slots

`.claude/skills/update-from-hatchery/SKILL.md` steps 5 and 6. Three gaps, each seen 2026-10-16 in
a scan of a sibling that has run or simulated an update:

- Under `.claude/` the rule is "base wins" except for `policy.json`, `settings.json` and the
  harness group. An adopted project hand-patched `rules/docs.md` (a 130-line `CLAUDE.md` cap, cap
  exemptions for its own big docs), five `SKILL.md` files and the proposal templates, and the base
  has changed every one since. Read literally the rule wipes the project's text. The skill needs a
  mixed-file rule: take the base, then re-apply the project lines its adoption ADR names.
- The Usage gating table's threshold cells in `CLAUDE.md` are listed as project slots by adopt
  step 6 but not by update step 5, which calls the whole section base-wins; a simulated update's
  only `CLAUDE.md` conflict was exactly those rows, and the literal resolution writes "base: 50%"
  against a policy of 80/90. The `Hatchery base:` provenance line near the top of `CLAUDE.md` is
  not rewritten either; only the ADR line is.
- Step 6's `wc -l CLAUDE.md` at most 150 fails for a project at the cap (154 after a simulated
  update). ADR 0012 Consequences states the rule for one project; the step should state it for
  all: the project trims its own lines before the update, and the step says how many.
