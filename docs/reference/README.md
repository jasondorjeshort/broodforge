# Reference

Durable facts the project depends on: how a tool behaves, what an API accepts, what an environment
provides. One topic per file, named by topic (`git-credentials.md`, `android-sdk.md`), never by date.
Each file starts with frontmatter `verified: YYYY-MM-DD` and `source: <url or "observed">`, then
facts, commands that work, and gotchas. Cap 300 lines; split by topic when near it.

A fact gets here by being promoted from a finding (`finding` skill) or written directly when it is
already certain. When a fact turns out to be wrong, fix it and bump `verified:`; do not append history.

| File | Topic | Verified |
|---|---|---|
| `claude-swap.md` | The `cswap` account switcher: where its state and Claude Code's config live, `switch`, `run` and `CLAUDE_CONFIG_DIR`, `auto` and its exit codes, the `list --json` contract, the usage endpoint budget. | 2026-10-16 |
