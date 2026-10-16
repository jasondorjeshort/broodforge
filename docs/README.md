# docs/ — the map

Read the index of a directory before its files. Nothing here is loaded automatically except
`HANDOFF.md` and the files it imports. Layout and reasoning: `design/decisions/0002-repository-layout.md`.

| Path | What lives there | Loaded |
|---|---|---|
| `HANDOFF.md` | Now, Next, Waiting on user, Open findings; in the split shape, the map of `@handoff/<stream>.md` lines (`handoff` skill, `split.md`). Coordinator state, 40-line cap. | every session and subagent, via `CLAUDE.md` |
| `ideas/` | One dated file per improvement nobody is working on, each anchored to code; triaged into `Plan:`, `Question:` or `Waits:`; deleted when built. | on demand; start at `ideas/README.md` |
| `operator/` | One dated file per step only the owner can take (accounts, keys, elevation); deleted when done. | on demand; start at `operator/README.md` |
| `design/` | What we build and why: `00`-`06` design docs, `proposals/`, `decisions/` (ADRs). | on demand; start at `design/README.md` |
| `reference/` | Durable facts about tools, APIs, platforms, environments. One topic per file, `verified:` date. | on demand; start at `reference/README.md` |
| `findings/` | Dated, status-tagged results of investigations: root causes, benchmarks, failed approaches. | on demand; start at `findings/README.md` |

Where does a new piece of text go? Ask in this order:

1. A rule Claude must obey every turn: `CLAUDE.md`. Only when touching certain paths:
   `.claude/rules/<name>.md` with `paths:` frontmatter.
2. A procedure (steps to do X): a skill under `.claude/skills/`. Never `docs/`, never `CLAUDE.md`.
3. A decision about what we build or why: `design/`; an ADR if the choice is non-obvious (`adr` skill).
4. A fact about a tool, API, platform, or environment that will stay true: `reference/`.
5. The result of an investigation: `findings/`, via the `finding` skill. When resolved, promote the
   durable part to `reference/` or `design/` and set `promoted_to`.
6. What is happening now or next: `HANDOFF.md` (`handoff` skill). Later than next, or not yet
   decided: an `ideas/` file (`triage` skill writes its block). A step only the owner can take:
   an `operator/` file, in the commit that needs it.

Conventions (caps, frontmatter, indexes) are enforced by `.claude/rules/docs.md`, which loads
automatically when a file under `docs/` is read or edited, and by the `pusher` before every push.
