---
name: decant
description: Turn a fresh clone of the Hatchery base repository into a new project. Use when starting a new project from Hatchery, when asked to decant, or when a repository still has the Hatchery README and no project of its own.
---

# Decant

Hatchery is the base; a project is decanted from it. The result keeps the working agreement, the
agents, the skills, the hooks, and ADRs 0001 and 0002, and starts its own design, handoff, and
ideas. Everything below runs in the new project's clone.

## Steps

1. Prerequisites. Run `node --version` and `gh --version`. Without Node: the status line, the usage
   gate, and both hooks fail to start; with `onFailure: block` on git-guard that means every Bash
   call is blocked until Node is installed. Without `gh`: the pusher cannot authenticate and the
   credential-helper step in the `commit` skill does not apply. Stop and tell the user if either is
   missing; do not continue with hooks disabled.
2. Remotes. `git remote -v`. If `origin` still points at Hatchery, rename it:
   `git remote rename origin hatchery`, then `git remote add origin <new project remote>`. If
   `origin` is already the project, add `git remote add hatchery https://github.com/jasondorjeshort/hatchery.git`.
   The second remote is always named `hatchery`; the `update-from-hatchery` skill relies on it.
3. `README.md`: replace with the project's README, keeping the License section. Keep nothing about
   decanting; that lives here.
4. `CLAUDE.md`: set the title line to the project name, keep the sentence saying it comes from
   Hatchery, and review the project-specific parts: platform list reference, usage thresholds,
   model routing. Change only what the project needs.
5. `docs/HANDOFF.md`: rewrite with the `handoff` skill; Now is "decanting from Hatchery", Next is
   the design sequence starting at `docs/design/00-vision.md`.
6. `docs/ideas/`: delete every idea file except `README.md`; reset its index table to its empty row.
7. `docs/findings/` and `docs/reference/`: delete every file except `README.md`; reset each index
   table to its empty row. Hatchery's findings are about Hatchery.
8. `docs/design/`: keep `README.md`, `decisions/0001-licensing.md`, `decisions/0002-*.md`, and
   the `proposals/` and `decisions/` READMEs. Delete other proposals. Edit the "Decisions so far"
   list to match. If the project wants a different license, supersede 0001 with the `adr` skill
   and replace `LICENSE`.
9. `.claude/settings.json`: keep as is. `.claude/rules/`: keep `docs.md`; add language rules
   later per `.claude/rules/README.md`.
10. Verify: `git status --short` shows only the intended edits; `wc -l CLAUDE.md docs/HANDOFF.md`
    are at most 150 and 40.
11. Commit via the runner (`commit` skill): subject `Decant <project> from Hatchery`, body listing
    what was removed and kept. Do not push; the user decides when.
12. Report: remotes, files changed, and the first design step for the architect.

## Rules

- Do not edit agent files, skills, or hooks during decanting. If something in them is wrong for
  the project, that is a change to make in Hatchery and pull back with `update-from-hatchery`.
- Use the Write tool for any file containing apostrophes or backticks.
