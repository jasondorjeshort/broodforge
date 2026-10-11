---
name: update-from-hatchery
description: Pull improvements from the Hatchery base repository into a project that was decanted from it, resolving merge conflicts by a fixed rule. Use when asked to update from Hatchery, sync the base, or merge hatchery/main.
---

# Update from Hatchery

The base evolves (agents, skills, hooks, rules, working agreement). A decanted project merges those
changes in and keeps its own text where the two overlap.

## Steps

1. Preconditions: `git status --porcelain` is empty (commit or stash first; do not merge onto a
   dirty tree). `git rev-parse --abbrev-ref HEAD` is `main` or the branch the user named.
2. Remote: `git remote -v`. If `hatchery` is missing,
   `git remote add hatchery https://github.com/jasondorjeshort/hatchery.git`.
3. `git fetch hatchery`. Record `git log --oneline HEAD..hatchery/main` as the list of incoming
   base commits; if empty, report "already up to date" and stop.
4. `git merge --no-ff hatchery/main -m "Merge hatchery/main into <project>"`. If it completes
   cleanly, go to step 6.
5. Conflicts. `git diff --name-only --diff-filter=U`. Resolve each file by this rule, then
   `git add` it:
   - Base wins: anything under `.claude/` (agents, skills, hooks, rules, `settings.json`) and
     `LICENSES/`. Exception: the project has recorded a reason to differ (an ADR, or a comment in
     the file naming one); then keep the project version and note it in the report.
   - Project wins: `README.md`; the title and project-specific sections of `CLAUDE.md`
     (platforms, thresholds, routing, anything the project added); `docs/HANDOFF.md`;
     `docs/backlog.md`; everything under `docs/design/`, `docs/findings/`, `docs/reference/`.
   - `CLAUDE.md` rule sections (Delegation, Debug interface, Major decisions, Usage gating, Git,
     Licensing): base wins unless the project has an ADR for the difference.
   - Mixed files: take the base version, then re-apply the project's lines by hand. Never leave
     conflict markers; `grep -rn "^<<<<<<<\|^>>>>>>>" .` must be empty before continuing.
   Finish with `git commit` (the merge message is already prepared).
6. Post-merge checks, each must pass or be reported as a failure:
   - `node --version`; `node .claude/hooks/usage-gate.js < /dev/null` exits 0.
   - `wc -l CLAUDE.md docs/HANDOFF.md`: under 130 and 40.
   - Every `.claude/skills/*/SKILL.md` and `.claude/agents/*.md` starts with frontmatter.
   - If the project defines test and lint commands (`CLAUDE.md`, `README.md`, manifest), run them.
7. Do not push. The user decides; pushing goes through the `commit` skill's Pushing section.

## Report

- Incoming base commits (the list from step 3).
- Files merged cleanly, files resolved by rule (which rule), files kept as project version with
  the recorded reason.
- Post-merge check results, verbatim where they failed.
- Anything in the base that the project should act on: new skills to learn, changed hook
  behaviour, changed working-agreement rules. One line each.

## Rules

- Never `--force`, never rewrite history, never resolve by discarding the merge silently.
- If a conflict does not fit the rule above, stop, leave the merge in progress, and ask the user
  with both versions quoted.
