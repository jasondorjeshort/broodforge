---
name: decant
description: Turn a fresh clone of the Hatchery base repository into a new project. Use when starting a new project from Hatchery, when asked to decant, or when a repository still has the Hatchery README and no project of its own.
---

# Decant

Hatchery is the base; a project is decanted from it. The result keeps the working agreement, the
agents, the skills, the hooks, every Hatchery ADR (the record of the practice it inherits; its own
ADRs start at 0020, ADR 0008) and the Windows reference, and starts its own design, handoff, ideas
and findings. Everything below runs in the new project's clone.

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
   After the rename run `git branch --unset-upstream` (step 11 checks it): the rename carries the branch's upstream
   to `hatchery`, and a bare `git push` would then go to Hatchery. The first push sets `origin`
   (the pusher's `git push -u origin <branch>`).
3. `README.md`: replace with the project's README, keeping the License section. Keep nothing about
   decanting; that lives here.
4. `CLAUDE.md`: set the title line to the project name, keep the sentence saying it comes from
   Hatchery, and review the project-specific parts: platform list reference and model routing.
   Thresholds are not here; they are policy (step 5), and the usage table reads them from there.
   Change only what the project needs.
5. `.claude/hooks/policy.json`: the project's choices for the hooks, never edits to the scripts
   (ADR 0006): usage thresholds (`usage.hold`, `usage.deny`, per meter under `usage.fiveHour` and
   `usage.sevenDay`), who commits and pushes (`git.commit.subagents`, `git.push.subagents`,
   `git.push.mainSession`), and cap exemptions (`edit.capExempt`). Thresholds and overage are the
   owner's money: ask, do not guess. `.claude/hooks/policy.examples/` has worked ones; copy the
   closest and edit. `node .claude/hooks/policy.js` must print the effective policy, not an error.
6. `docs/HANDOFF.md`: rewrite with the `handoff` skill; Now is "decanting from Hatchery", Next is
   the design sequence starting at `docs/design/00-vision.md`.
7. `docs/ideas/` and `docs/operator/`: delete every file except `README.md`; reset each index
   table to its empty row (`docs/operator/` is normally already at rest).
8. `docs/findings/`: delete every file except `README.md`; reset its index table to its empty row.
   Hatchery's findings are about Hatchery. `docs/reference/`: keep `README.md` and
   `claude-code-on-windows.md` with its README row (it describes the tool and the machine, not
   Hatchery); delete every other reference file and its row.
9. `docs/design/`: keep `README.md`, every file under `decisions/`, and the `proposals/` README;
   delete every proposal and reset the proposals index table to its header. The "Decisions so
   far" list stays as it is: the project's own decisions are added below it from 0020 up. If the
   project wants a different license, supersede 0001 with the `adr` skill (as 0020) and replace
   `LICENSE`.
10. `.claude/settings.json`: keep as is. `.claude/rules/`: keep `docs.md`; add language rules
    later per `.claude/rules/README.md`.
11. Verify: `git status --short` shows only the intended edits; `wc -l CLAUDE.md docs/HANDOFF.md`
    are at most 150 and 40; `node .claude/hooks/policy.js` exits 0;
    `git config --get branch.<branch>.remote` prints nothing or `origin`, never `hatchery`.
12. Commit via the runner (`commit` skill): subject `Decant <project> from Hatchery`, body listing
    what was removed and kept. Do not push; the user decides when.
13. Report: remotes, files changed, the policy choices made, and the first design step for the
    architect.

## Rules

- Do not edit agent files, skills, or hooks during decanting. If something in them is wrong for
  the project, that is a change to make in Hatchery and pull back with `update-from-hatchery`.
  `policy.json` is the project's file and is the one thing under `.claude/hooks/` this skill edits.
- Use the Write tool for any file containing apostrophes or backticks.
