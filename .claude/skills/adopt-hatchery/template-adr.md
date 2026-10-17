# NNNN. Adopt the Hatchery practice

Date: YYYY-MM-DD
Status: accepted

Hatchery base: <full sha of hatchery/main at adoption; living line, bumped by update-from-hatchery>
Mechanism: Hatchery ADR 0004 (`git show hatchery/main:docs/design/decisions/0004-adopt-hatchery.md`).

## Context

What the project looked like before (branch, size, own CLAUDE.md, own hooks, docs layout, license)
and why the practice was adopted rather than the project decanted anew.

## Decision

- Copied verbatim: `.claude/agents/`, `.claude/skills/`, `.claude/hooks/`, `.claude/statusline.js`,
  `.claude/rules/README.md`, `.claude/rules/docs.md`.
- Merged: `.claude/settings.json` (project hook entries kept: <list, with the purpose of each>),
  `CLAUDE.md` (fold; project rules moved to <.claude/rules/<name>.md and where else>), `.gitignore`.
- Added: <docs skeleton files that were absent>; `docs/README.md` rows added; nothing moved.
- Rule sections the project overrides, each with the user's decision and the project text that
  replaces Hatchery's lines (this is the "recorded reason to differ" the update rule honours):
  - <section>: <decision>.
- License: <adopted GPL-3.0-or-later and kept ADR 0001 | kept existing <license>, supersedes 0001 |
  none, supersedes 0001>. SPDX headers: <new files only; retrofit as an ideas file | all files>.
- Docs over the 300-line cap at adoption: <file: split | exempted in `edit.capExempt` of `.claude/hooks/policy.json`>.

## Consequences

Updates use `update-from-hatchery` patch mode: the patch from the base line above to `hatchery/main`
over `.claude` and `CLAUDE.md`, three-way; the overrides listed here conflict on purpose and are
resolved by keeping the project text. Costs and follow-ups specific to this project: <list>.
