# codebot

A base repository to fork for agent-driven, design-first software projects. It contains no
application code. What it does contain:

- `CLAUDE.md`: the working agreement Claude Code follows in the repo. Agents do all coding and
  debugging, every feature ships with an agent-usable debug interface, major choices go through a
  proposal, a critic, and an ADR, and work stops when plan usage gets high.
- `.claude/agents/`: six subagents with models matched to their job (architect and critic on
  Fable, coder on Opus, tester and pusher on Sonnet, runner on Haiku).
- `.claude/statusline.js` and `.claude/hooks/usage-gate.js`: a status line that shows the 5-hour
  and weekly plan meters, and a hook that refuses to launch new subagents once either reaches 50%.
- `docs/design/`: the design document skeleton, with a proposals folder and an ADR folder.
- `LICENSE` and `LICENSES/`: GPL-3.0-or-later by default, LGPL-3.0-or-later text for linkable components.

## Starting a new project from this base

1. Fork or clone, then point `origin` at the new project's remote. Keep this repo as a second
   remote named `template` if you want to merge later improvements to the base:
   `git remote add template <this repo's url>`.
2. Replace this README with the new project's README. Keep the license section.
3. In `CLAUDE.md`, set the title and adjust anything project-specific (the platform list, the
   usage thresholds, the model routing) if the defaults do not fit.
4. Revisit `docs/design/decisions/0001-licensing.md`. If the new project wants a different
   license, supersede that ADR and replace `LICENSE`.
5. Start the design with the `architect` agent: `docs/design/00-vision.md` first.

On GitHub, mark this repository as a template (Settings, "Template repository") so new projects
can be created from it with a clean history.

## Pulling base improvements into a fork

```
git fetch template
git merge template/main
```

Expect conflicts in `README.md` and `CLAUDE.md`; resolve in favor of the fork's project-specific text.

## License

GPL-3.0-or-later. See [`LICENSE`](LICENSE). Some components may be LGPL-3.0-or-later; each such
component has its own `LICENSE` file, and the list is in
[`docs/design/decisions/0001-licensing.md`](docs/design/decisions/0001-licensing.md).
