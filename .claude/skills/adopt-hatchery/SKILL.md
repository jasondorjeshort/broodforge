---
name: adopt-hatchery
description: Bring the Hatchery practice (working agreement, agents, skills, hooks, docs skeleton) into an existing project that was not decanted from Hatchery, by copying from hatchery/main without merging histories. Use when a project has its own history and no .claude/agents/, when asked to adopt Hatchery, or when update-from-hatchery stops for lack of a merge-base.
---

# Adopt Hatchery

A project that already exists keeps its history and takes the practice as files. The files come
from `hatchery/main`; what was copied and from which commit is recorded in the project's own ADR, so
`update-from-hatchery` can patch forward later (patch mode). Reasoning and the full table of what is
copied, merged, or added: Hatchery ADR 0004 (`git show hatchery/main:docs/design/decisions/0004-adopt-hatchery.md`).
Everything below runs in a private clone of the project, on its default branch, as the main session.

## Steps

1. Preconditions, stop at the first failure and report it: `node --version` and `gh --version` work
   (reasons: `decant` skill step 1); `git status --porcelain` is empty; `git rev-parse --abbrev-ref HEAD`
   is the project's working branch (`master`, `main`, whatever it is; not detached); this clone is
   private (not a working tree shared with other sessions); `.claude/agents/` does not exist (if it
   does, the project is already decanted or adopted: use `update-from-hatchery`).
2. Remote: `git remote add hatchery https://github.com/jasondorjeshort/hatchery.git` if missing;
   `git fetch hatchery`; `BASE=$(git rev-parse hatchery/main)`. Read this skill and ADR 0004 from
   `hatchery/main` if you have not already.
3. Verbatim files (base wins): `git checkout hatchery/main -- .claude/agents .claude/skills
   .claude/hooks .claude/statusline.js .claude/rules/README.md .claude/rules/docs.md`.
4. `.claude/settings.json`: if absent, take Hatchery's. If present, write the union: Hatchery's
   `statusLine`, `permissions` (allow and deny lists appended, no duplicates), and every Hatchery
   hook entry, plus every existing project hook entry under its own event. Nothing of the project's
   is dropped. Note each project hook whose purpose contradicts a Hatchery rule (for step 6).
5. `.gitignore`: append `.claude/settings.local.json` if absent. `README.md`: untouched.
6. `CLAUDE.md` fold. Read the project's file and Hatchery's (`git show hatchery/main:CLAUDE.md`).
   Classify every project rule as one of: every-turn and compatible (goes to a new `## Project rules`
   section, condensed); path-scoped (goes to `.claude/rules/<name>.md` with `paths:`); a procedure
   (stays in the project's docs, or becomes a skill later; one pointer line); a fact (`docs/reference/`);
   a contradiction of a Hatchery rule section (This agreement is part of the work, Where things
   live, Delegation, Debug interface, Major decisions, Usage gating, Model routing, Git, Licensing).
   Present every contradiction to the user with both texts quoted, one pair per item, and wait; do
   not write `CLAUDE.md` with a guess. Then write it with the Write tool: Hatchery's text in the rule
   sections (the user's choices applied), the project in the slots: title; provenance sentence
   "adopted from Hatchery at `<BASE>`, see ADR NNNN"; `## Phase` as the project's actual phase;
   thresholds; routing table; branch name in `## Git`; `## Licensing` per step 8; `## Project rules`.
   `wc -l CLAUDE.md` under 130; move more to `.claude/rules/` until it is.
7. Docs skeleton, only where the project lacks the file: `git checkout hatchery/main -- docs/HANDOFF.md
   docs/ideas/README.md docs/findings/README.md docs/reference/README.md docs/design/proposals/README.md
   docs/design/decisions/README.md docs/design/decisions/0001-licensing.md
   docs/design/decisions/0002-repository-layout.md docs/design/decisions/0002-hook-specs.md`, then
   reset `findings/` and `reference/` index tables to their empty row and leave `docs/ideas/` with only its README.
   `docs/design/README.md`: absent, write one that lists the existing design files one line each plus
   "Decisions so far"; present, add the list. `docs/README.md`: add one row per added path to the
   project's own map; move nothing. `docs/HANDOFF.md`: `handoff` skill; Now is "adopting Hatchery",
   Next links the project's current work wherever it already lives.
8. License. No `LICENSE`: ask the user (GPL-3.0-or-later as Hatchery, or none). Yes: `git checkout
   hatchery/main -- LICENSE LICENSES`, keep 0001, SPDX headers on new files, retrofit as an ideas
   file. No: copy nothing, `## Licensing` is one line stating the terms, the ADR of step 10
   supersedes 0001. `LICENSE` present and different: keep it, copy nothing, supersede 0001 the same way.
9. Caps. `git ls-files 'docs/*.md' | xargs wc -l | awk '$1>300 && $2!="total"'`: list every pre-existing file over
   300 lines. Per file the user chooses split or exempt; an exemption is a `case` skip added to
   pusher check 7 and `post-edit.js`, recorded in step 10 as the reason to differ.
10. The project's ADR: `docs/design/decisions/NNNN-adopt-hatchery.md` from `template-adr.md` beside
    this skill (NNNN: `adr` skill numbering). The line `Hatchery base: <BASE>` is required and living.
    Add it to "Decisions so far".
11. Verify: `git status --short` lists only the paths above; `grep -rn "^<<<<<<<\|^>>>>>>>" .` empty;
    `wc -l CLAUDE.md docs/HANDOFF.md` under 130 and 40; `node .claude/hooks/usage-gate.js < /dev/null`
    exits 0; `node -e "JSON.parse(require('fs').readFileSync('.claude/settings.json','utf8'))"`.
12. Commit from the main session (the runner and git-guard are not loaded until restart): stage the
    paths explicitly, subject `Adopt the Hatchery practice at <short BASE>`, body listing copied,
    merged, added, and decided items, `Co-Authored-By` trailer. Do not push; the user decides.
13. Tell the user to restart Claude Code so hooks, status line, agents, and skills load; the first
    session then runs step 11's node checks again and launches nothing until the status line shows usage.

## Report

Preconditions checked; `BASE`; files copied, merged (how), added; each contradiction and the user's
decision; the license decision; files over the cap and their fate; the ADR number; what to do after
restart. Hand-merge steps a model cannot decide alone go to the user with both versions quoted.

## Rules

- Never `git merge`, `rebase`, or `replace` to join histories; the project's log stays its own.
- Do not edit agents, skills, or hooks during adoption; a wrong base file is fixed in Hatchery.
  The only base edit allowed here is the cap exemption of step 9, and it is recorded.
- Use the Write tool for files containing apostrophes or backticks.
