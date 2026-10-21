---
date: 2026-10-21
kind: chore
related: [.claude/skills/adr/SKILL.md, docs/design/decisions/0024-decanted-from-hatchery.md]
---

# Carry back: the adr skill's dated Amendments section

Anchor: `.claude/skills/adr/SKILL.md`, "Amend without superseding" (departure row in ADR 0024).

The base allows editing an accepted ADR only in a table the ADR calls living. This project amends
ADRs whose Decision keeps its core but whose detail a finding or an owner decision corrected: a
dated entry under `## Amendments` at the end of the ADR, citing its source, naming the sentence it
replaces and stating the replacement, the original left as written; a reversal of the core stays a
supersede. ADRs 0020 and 0022 here carry such sections. Without the rule in the skill, a session
either edits Decision text in place or writes a superseding ADR for a one-line correction.

Plan: carry back the section as this project's skill has it (`carry-back` skill, Send), with the
example ADR numbers replaced by a generic description ("an ADR corrected by a spike finding"); then
the row in ADR 0024 records `carry-back/adr-amendments at <sha7>`.
