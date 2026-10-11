# Hatchery

## Use it with Claude Code

Start a new project from Hatchery. Replace the two placeholders, then paste into Claude Code:

```
Create a new project from the Hatchery base. Clone https://github.com/jasondorjeshort/hatchery.git
into a directory named <project> and work inside it from now on. Rename the remote `origin` to
`hatchery`, then add `origin` pointing at <new project remote URL>, and confirm both with
`git remote -v`. Then load the `decant` skill and follow it to the end: check that node and gh
are installed, replace the README, retitle CLAUDE.md, reset docs/HANDOFF.md and docs/backlog.md,
clear docs/findings/ and docs/reference/, keep ADRs 0001 and 0002, and commit through the runner.
Do not push. Report the remotes, the files you changed, and the first design step.
```

Update an existing project from Hatchery. Paste into Claude Code inside that project:

```
Update this project from the Hatchery base. Load the `update-from-hatchery` skill and follow it:
confirm the working tree is clean, add the remote `hatchery`
(https://github.com/jasondorjeshort/hatchery.git) if it is missing, fetch it, and merge
`hatchery/main`. Resolve conflicts by the skill's rule: this project's own text wins in README.md,
the project-specific sections of CLAUDE.md, docs/HANDOFF.md, docs/backlog.md, and everything under
docs/design/, docs/findings/, and docs/reference/; the base wins under .claude/ unless an ADR here
records a reason to differ. Leave no conflict markers. Re-run the project's checks and tests.
Do not push. Report what changed in the base, how each conflict was resolved, and what I should act on.
```

Both need Node (hooks and status line) and the GitHub CLI `gh` (pushing) on `PATH`.

## What this is

A base repository to fork for agent-driven, design-first software projects. New projects are
decanted from it: they inherit the conditioning (the working agreement, the agents, the skills,
the hooks) and grow their own design and code. It contains no application code. What it does contain:

- `CLAUDE.md`: the working agreement Claude Code follows in the repo. Agents do all coding and
  debugging, every feature ships with an agent-usable debug interface, major choices go through a
  proposal, a critic, and an ADR, and work stops when plan usage gets high.
- `.claude/agents/`: six subagents with models matched to their job (architect and critic on
  Fable, coder on Opus, tester and pusher on Sonnet, runner on Haiku).
- `.claude/skills/`: the procedures (proposal, ADR, handoff, debug interface, finding, commit,
  brief, decant, update-from-hatchery), loaded on demand.
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
this repository as a template (Settings, "Template repository") if you prefer a clean history; then
add the `hatchery` remote yourself so updates still work.

## Pulling base improvements into a project

The second prompt above does this. By hand: `git fetch hatchery && git merge hatchery/main`, then
resolve conflicts by the rule in `.claude/skills/update-from-hatchery/SKILL.md`: the base wins under
`.claude/`, the project wins in its own README, design, handoff, backlog, findings, and reference.

## License

GPL-3.0-or-later. See [`LICENSE`](LICENSE). Some components may be LGPL-3.0-or-later; each such
component has its own `LICENSE` file, and the list is in
[`docs/design/decisions/0001-licensing.md`](docs/design/decisions/0001-licensing.md).
