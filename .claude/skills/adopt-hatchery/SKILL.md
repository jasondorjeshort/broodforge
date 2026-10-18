---
name: adopt-hatchery
description: Bring the Hatchery practice (working agreement, agents, skills, hooks, docs skeleton) into an existing project that was not decanted from Hatchery, by copying from hatchery/main without merging histories. Use when a project has its own history and no .claude/agents/, when asked to adopt Hatchery, or when update-from-hatchery stops because the project has no base line and no shared history with Hatchery.
---

# Adopt Hatchery

A project that already exists keeps its history and takes the practice as files. The files come
from Hatchery's main branch; what was copied and from which commit is recorded in the project's own ADR, so
`update-from-hatchery` can patch forward later (the one procedure, ADR 0016). Reasoning and the full table of what is
copied, merged, or added: Hatchery ADR 0004 (after `S=$(git rev-parse hatchery/main)`, `git show $S:docs/hatchery/decisions/0004-adopt-hatchery.md`).
Everything below runs in a private clone of the project, on its default branch, as the main session.

## Steps

1. Preconditions, stop at the first failure and report it: `node --version` and `gh --version` work
   (reasons: `decant` skill step 1); `git status --porcelain` is empty; `git rev-parse --abbrev-ref HEAD`
   is the project's working branch (`master`, `main`, whatever it is; not detached); this clone is
   private (not a working tree shared with other sessions); `.claude/agents/` does not exist (if it
   does, the project is already decanted or adopted: use `update-from-hatchery`).
2. Remote: `git remote add hatchery https://github.com/jasondorjeshort/hatchery.git` if missing;
   `git fetch --prune hatchery`; `BASE=$(git rev-parse hatchery/main)`; `S=$BASE`. Read this skill and ADR 0004
   with `git show $S:<path>` if you have not already.
3. Verbatim files (base wins): the base-owned set of `.claude/hooks/ownership.json` without
   `LICENSE` and `LICENSES` (step 8 decides those): `git checkout hatchery/main -- .claude/agents
   .claude/skills .claude/hooks .claude/agreement .claude/statusline.js .claude/package.json
   .claude/rules/README.md .claude/rules/docs.md docs/hatchery` (`.claude/package.json` keeps the scripts
   CommonJS when the project's root `package.json` says `"type": "module"`; without it every hook
   crashes). Then `.claude/hooks/policy.json` becomes the project's: who commits and pushes,
   deletable branches, usage thresholds, `usage.pool` (`null` unless this machine has the
   switcher), and `update.exclude` (step 8 writes it on a licence decline) (ADR 0006;
   `.claude/hooks/policy.examples/` has two worked ones). Ask the user
   where the project's existing rules do not say; `node .claude/hooks/policy.js` must print it.
   Identity (ADR 0018): ask the owner, in these words, "Which name and email go on this project's
   commits?" Recommend a `users.noreply.github.com` address; during a transition list both.
   Write `privacy.identities` (`[{ "name", "email" }]`, first entry is the repository identity),
   then set `git config user.name` and `user.email` repo-local from the first entry (the global
   identity is not read). Then look before writing any hooks path: `git config --get
   core.hooksPath` printing anything, or a file under `.git/hooks/` not ending `.sample`, means
   the project has hooks. Never overwrite a hook; never change an existing `core.hooksPath`.
   - No hooks (no `core.hooksPath`, only `.sample` files): set `git.hooksPath` to
     `.claude/hooks/git` in the policy, then `git config core.hooksPath .claude/hooks/git`.
   - The owner's hook can call ours (a chain): leave `core.hooksPath` as it is, set `git.hooksPath`
     in the policy to the project's existing value, and tell the owner the one line their hook
     adds to run `.claude/hooks/git/pre-push`.
   - The owner declines the pre-push hook: `git.hooksPath` stays `null`, no config line, both as
     they were; pusher check 5 is then the only layer.
   Ask the owner which of the last two applies. Step 10's ADR records the chain or decline outcome.
4. `.claude/settings.json`: if absent, take Hatchery's. If present, write the union: Hatchery's
   `statusLine`, `permissions` (allow and deny lists appended, no duplicates), and every Hatchery
   hook entry, plus every existing project hook entry under its own event. Nothing of the project's
   is dropped. Note each project hook whose purpose contradicts a Hatchery rule (for step 6).
5. `.gitignore`: append `.claude/settings.local.json` if absent. `README.md`: untouched.
6. `CLAUDE.md` rewrite (ADR 0016). Read the project's file and the ten files in
   `.claude/agreement/` (step 3 brought them). Classify every project rule as one of: every-turn
   and compatible (goes to `## Project rules`, condensed); path-scoped (`.claude/rules/<name>.md`
   with `paths:`); a procedure (stays in the project's docs, or becomes a skill later; one pointer
   line); a fact (`docs/reference/`); a contradiction of an agreement section (file name in
   `.claude/agreement/`). Present every contradiction to the user with both texts quoted, one pair
   per item, and wait; do not write `CLAUDE.md` with a guess. Each kept contradiction becomes
   `.claude/project/<section>.md`, one line: "Replaces `.claude/agreement/<section>.md` for this
   project: <the rule>. Decided in ADR <NNNN>." (the replaced path in backticks), plus one
   departure row in step 10's ADR. Then write `CLAUDE.md` with the Write tool in the shape of
   `decant` step 4's template, with "Adopted from Hatchery; provenance and departures:
   `docs/design/decisions/<NNNN>-adopt-hatchery.md`" as the provenance line (NNNN of step 10), no routing
   slot, each replaced section's `@.claude/agreement/<s>.md` line swapped for
   `@.claude/project/<s>.md`, and the project's actual phase and branch (`Branch: <name>`).
   `node .claude/hooks/policy.js --claude-check` prints no `FAIL`: at most 40 lines; move the
   overflow to `.claude/rules/`, `docs/reference/`, a replacement file under `.claude/project/` (with
   its row) or the owner's `edit.capExempt` lift for `CLAUDE.md`, never to `## Project rules`.
7. Docs skeleton, only where the project lacks the file: `git checkout hatchery/main -- docs/HANDOFF.md
   docs/ideas/README.md docs/operator/README.md docs/findings/README.md docs/reference/README.md docs/design/proposals/README.md
   docs/design/decisions/README.md`, then
   reset the `reference/` index table to its empty row and leave `docs/ideas/` with only its README.
   The three queue READMEs carry no table (ADR 0017); an adopted project keeps its own tables and
   warnings where it already has the file.
   `docs/design/README.md`: absent, write one that lists the existing design files one line each plus
   "Decisions so far"; present, add the list. `docs/README.md`: add one row per added path to the
   project's own map, `hatchery/` included; move nothing. The handoff: ask the owner, in these words, "Will more than one
   session work this branch at the same time?" (ADR 0015). Yes: the split shape as the `decant`
   skill's handoff step writes it, the stream named for the project's current work.
   No: `docs/HANDOFF.md` with the `handoff` skill. Either way Now is "adopting Hatchery" and Next links the project's current
   work wherever it already lives.
8. License. No `LICENSE`: ask the user (GPL-3.0-or-later as Hatchery, or none). Yes: `git checkout
   hatchery/main -- LICENSE LICENSES`, keep 0001, SPDX headers on new files, retrofit as an ideas
   file. No: copy nothing; write `update.exclude: ["LICENSE","LICENSES"]` into `policy.json` (a project
   edit, no row) and `.claude/project/licensing.md` stating the terms (the replacement-file line of
   step 6), swap the licensing import in `CLAUDE.md` for `@.claude/project/licensing.md`, add one
   departure row, and the ADR of step 10 supersedes 0001; no `## Licensing` section in `CLAUDE.md`.
   `LICENSE` present and different: keep it, copy nothing, the same exclude, replacement file and
   row, and supersede 0001 the same way.
9. Caps. `git ls-files 'docs/*.md' | xargs wc -l | awk '$1>300 && $2!="total"'`: list every pre-existing file over
   300 lines. Per file the user chooses split or exempt; an exemption is the path (or a glob) added
   to `edit.capExempt` in `.claude/hooks/policy.json`, which the post-edit hook and pusher check 7
   both read, recorded in step 10's Decision (no departure row: `policy.json` is the project's). Never
   edit check 7 for it.
10. The project's provenance ADR: `docs/design/decisions/NNNN-adopt-hatchery.md` from
    `template-adr.md` beside this skill, title `Adopted from` (NNNN: `adr` skill numbering). The
    line `Hatchery base: <BASE>` is required and living; the departure table gets its header row and
    one row per replaced section and per base-owned file the project edits (not `policy.json`, which
    is the project's). The Decision records step 3's hooks outcome (chained or declined). Add it
    to "Decisions so far".
11. Verify: `git status --short` lists only the paths above; `grep -rn "^<<<<<<<\|^>>>>>>>" .` empty;
    `node .claude/hooks/policy.js --claude-check` prints no `FAIL`; stage the handoff first, since the commit step takes it anyway (`git add docs/HANDOFF.md` and,
    where it exists, `git add docs/handoff/`), so an untracked stream file is checked too; then the
    three command lines of pusher check 7 (`.claude/agents/pusher.md`)
    print no `FAIL` (both handoff shapes); `node .claude/hooks/usage-gate.js < /dev/null`
    exits 0; `node -e "JSON.parse(require('fs').readFileSync('.claude/settings.json','utf8'))"`.
12. Commit from the main session (the runner and git-guard are not loaded until restart): stage the
    paths explicitly, subject `Adopt the Hatchery practice at <short BASE>`, body listing copied,
    merged, added, and decided items, `Co-Authored-By` trailer. Do not push; the user decides.
13. Tell the user to restart Claude Code so hooks, status line, agents, and skills load; the first
    session then runs step 11's node checks again and launches nothing until the status line shows usage.

## Report

Preconditions checked; `BASE`; files copied, merged (how), added; each contradiction and the user's
decision; the license decision; files over the cap and their fate; the handoff answer and the shape it gave; the ADR number; what to do after
restart. Hand-merge steps a model cannot decide alone go to the user with both versions quoted.

## Rules

- Never `git merge`, `rebase`, or `replace` to join histories; the project's log stays its own.
- Do not edit agents, skills, or hooks during adoption; a wrong base file is fixed in Hatchery.
  The cap exemption of step 9 edits the project-owned `policy.json`, so it is no departure.
- Use the Write tool for files containing apostrophes or backticks.
