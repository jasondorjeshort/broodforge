# Hatchery

## Use it with Claude Code

Three prompts, each self-contained. Every one needs Node (hooks and status line) and the GitHub
CLI `gh` (pushing) on `PATH`, and stops at the first failed precondition instead of improvising.

**Start a new project from Hatchery.** Replace the two placeholders, then paste into Claude Code:

```
Create a new project from the Hatchery base. Preconditions, stop and tell me if one fails: `node
--version` and `gh --version` work; the directory <project> does not exist yet. Then clone
https://github.com/jasondorjeshort/hatchery.git into <project> and work inside it from now on.
Rename the remote `origin` to `hatchery`, add `origin` pointing at <new project remote URL>, and
confirm both with `git remote -v`. Load the `decant` skill (it is in the clone) and follow it to the
end: replace the README, retitle CLAUDE.md, reset docs/HANDOFF.md and docs/backlog.md, clear
docs/findings/ and docs/reference/, keep ADRs 0001 and 0002, and commit through the runner.
Do not push. Report the remotes, the files you changed, and the first design step.
```

**Update a project that came from Hatchery** (decanted, or adopted with the third prompt). Paste
into Claude Code inside that project:

```
Update this project from the Hatchery base. Preconditions, stop and tell me which failed if one
does: the file .claude/skills/update-from-hatchery/SKILL.md exists here (if not, this project
never came from Hatchery: stop and tell me to use the "Adopt the Hatchery practice" prompt from
the Hatchery README); `git status --porcelain` is empty; I am on this project's working branch.
Then load the `update-from-hatchery` skill and follow it: add the remote `hatchery`
(https://github.com/jasondorjeshort/hatchery.git) if missing, fetch it, pick the mode the skill
describes (merge when there is a merge-base, patch when the project's adopt-hatchery ADR records a
Hatchery base commit), and resolve conflicts by the skill's rule: this project's own text wins in
README.md, the project-specific sections of CLAUDE.md, docs/HANDOFF.md, docs/backlog.md, and
everything under docs/design/, docs/findings/, docs/reference/; the base wins under .claude/
unless an ADR here records a reason to differ. Leave no conflict markers. Re-run the project's
checks and tests. Do not push. Report the mode, what changed in the base, how each conflict was
resolved, and what I should act on.
```

**Adopt the Hatchery practice in an existing project** that was never decanted from it. The
project keeps its history; the practice arrives as files. Paste into Claude Code inside a private
clone of that project:

```
Adopt the Hatchery practice in this project. It has its own history and must keep it: no merge,
no rebase, no history rewrite. Preconditions, stop and tell me which failed if one does: `node
--version` and `gh --version` work; `git status --porcelain` is empty; I am on this project's
working branch (whatever its name); this clone is mine alone, not a shared working tree; there is
no .claude/agents/ directory here (if there is, use the update prompt instead). Then run
`git remote add hatchery https://github.com/jasondorjeshort/hatchery.git` (skip if present) and
`git fetch hatchery`, read the procedure with
`git show hatchery/main:.claude/skills/adopt-hatchery/SKILL.md`, and follow it to the end: copy
.claude/agents, skills, hooks, statusline, and rules from hatchery/main; merge Hatchery's hook
entries and permissions into my .claude/settings.json keeping every hook I already have; fold
Hatchery's working agreement into my CLAUDE.md keeping my own rules, and ask me, with both texts
quoted, about every rule of mine that contradicts one of Hatchery's; add the docs skeleton only
where I lack a file and add rows to my docs/README.md without moving anything; ask me before
adding a LICENSE; list my docs over 300 lines and ask split or exempt; write the adopt-hatchery
ADR with the Hatchery base commit; commit from this session. Do not push. Report what was copied,
merged, added, and decided, and tell me to restart Claude Code.
```

## What this is

A base repository for agent-driven, design-first software projects. New projects are decanted
from it: they inherit the conditioning (the working agreement, the agents, the skills, the hooks)
and grow their own design and code. Existing projects adopt the same practice as files. It contains
no application code. What it does contain:

- `CLAUDE.md`: the working agreement Claude Code follows in the repo. Agents do all coding and
  debugging, every feature ships with an agent-usable debug interface, major choices go through a
  proposal, a critic, and an ADR, and work stops when plan usage gets high.
- `.claude/agents/`: six subagents with models matched to their job (architect and critic on
  Fable, coder on Opus, tester and pusher on Sonnet, runner on Haiku).
- `.claude/skills/`: the procedures (proposal, ADR, handoff, debug interface, finding, commit,
  brief, decant, update-from-hatchery, adopt-hatchery), loaded on demand.
- `.claude/hooks/` and `.claude/statusline.js`: a status line showing the 5-hour and weekly plan
  meters, a gate that refuses new subagents at 50%, a guard against destructive git, and a
  post-edit check on documentation conventions.
- `.claude/rules/docs.md`: conventions that load only when files under `docs/` are touched.
- `docs/`: `HANDOFF.md` (where work stands, imported into every session), `backlog.md`, the
  design skeleton with proposals and ADRs, `reference/` for durable facts, `findings/` for
  investigation results. `docs/README.md` is the map.
- `LICENSE` and `LICENSES/`: GPL-3.0-or-later by default, LGPL-3.0-or-later text for linkable components.

Layout and reasoning: [`docs/design/decisions/0002-repository-layout.md`](docs/design/decisions/0002-repository-layout.md).

## Starting a new project by hand

The first prompt above does this for you. By hand: clone, rename `origin` to `hatchery`, add the
project's `origin`, then run `/decant` in Claude Code and follow the `decant` skill. On GitHub, mark
this repository as a template (Settings, "Template repository") if you prefer a clean history; a
template copy has no shared history, so treat it as an adoption (third prompt), not a decant.

## Pulling base improvements into a project

The second prompt above does this for either kind of project. By hand, decanted project:
`git fetch hatchery && git merge hatchery/main`. Adopted project: `git fetch hatchery`, then apply
`git diff <recorded Hatchery base> hatchery/main -- .claude CLAUDE.md` with `git apply --3way` and
bump the base line in your `adopt-hatchery` ADR. Either way, resolve conflicts by the rule in
`.claude/skills/update-from-hatchery/SKILL.md`: the base wins under `.claude/`, the project wins in
its own README, design, handoff, backlog, findings, and reference.

## Adopting the practice in an existing project

The third prompt does this. It copies the practice from `hatchery/main` and records the commit it
came from in an ADR in your project; your history, README, docs layout, and existing hooks stay.
Decisions a model must not make alone (a license, a rule of yours that contradicts one of
Hatchery's) are put to you with both texts. Reasoning and the exact list of what is copied, merged,
or added only where absent: [`docs/design/decisions/0004-adopt-hatchery.md`](docs/design/decisions/0004-adopt-hatchery.md).

## License

GPL-3.0-or-later. See [`LICENSE`](LICENSE). Some components may be LGPL-3.0-or-later; each such
component has its own `LICENSE` file, and the list is in
[`docs/design/decisions/0001-licensing.md`](docs/design/decisions/0001-licensing.md).
