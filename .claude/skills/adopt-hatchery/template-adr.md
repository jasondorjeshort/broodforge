# NNNN. <Adopted from|Decanted from> Hatchery

Date: YYYY-MM-DD
Status: accepted

Hatchery base: <full sha of hatchery/main at adoption or decant; living line, bumped by update-from-hatchery>
Mechanism: Hatchery ADR 0016 (update by patch over the base-owned set) and, for an adoption, ADR 0004.

## Context

What the project looked like before (branch, size, own CLAUDE.md, own hooks, docs layout, license)
and why the practice was adopted rather than the project decanted anew. A decant: one line saying so.

## Decision

- Base-owned files taken from the base: the set in `.claude/hooks/ownership.json`
  (`node .claude/hooks/ownership.js --check` lists tracked paths the map leaves unclassified); the policy file and `settings.local.json` stay the project's.
- Merged: `.claude/settings.json` (project hook entries kept: <list, with the purpose of each>),
  `.gitignore`. `CLAUDE.md` written from the template (title, provenance line, `@docs/HANDOFF.md`,
  `## Phase`, `Branch:`, ten agreement imports, `## Project rules`); project rules moved to
  <`## Project rules`, `.claude/rules/<name>.md`, and where else>.
- Added: <docs skeleton files that were absent>; `docs/README.md` rows added; nothing moved.
- Replaced sections: one `.claude/project/<section>.md` per contradiction, each with the user's
  decision and the project text; each is also a row of the departure table.
  - <section>: <decision>, or none.
- License: <adopted GPL-3.0-or-later and kept ADR 0001 | kept existing <license>, supersedes 0001 |
  none, supersedes 0001>. SPDX headers: <new files only; retrofit as an ideas file | all files>.
- Docs over the 300-line cap: <file: split | exempted in `edit.capExempt` of `.claude/hooks/policy.json`>.

## Departures

A living table: one row per base-owned path the project edits and per agreement section it
replaces. The project wins on a rowed path until Hatchery has the change; then base wins and the
row goes. Carry-back is an idea file, the Hatchery commit that took the change, or `none` and why.

| Path | Reason | Carry-back |
|---|---|---|

## Consequences

Updates use `update-from-hatchery`: the patch from the base line above to `hatchery/main` over the
base-owned set, three-way; a conflict on a rowed path keeps the project text, any other is base
wins. Costs and follow-ups specific to this project: <list>.
