# Reference

Durable facts the project depends on: how a tool behaves, what an API accepts, what an environment
provides. One topic per file, named by topic (`git-credentials.md`, `android-sdk.md`), never by date.
Each file starts with frontmatter `verified: YYYY-MM-DD` and `source: <url or "observed">`, then
facts, commands that work, and gotchas. Cap 300 lines; split by topic when near it.

A fact gets here by being promoted from a finding (`finding` skill) or written directly when it is
already certain. When a fact turns out to be wrong, fix it and bump `verified:`; do not append history.

| File | Topic | Verified |
|---|---|---|
| `agreement-loading.md` | How `CLAUDE.md` and its `@.claude/agreement/` imports load: the docs' rules, the sentinel procedure and what it showed, the `InstructionsLoaded` events. | 2026-10-17 |
| `claude-code-on-windows.md` | Processes, worktrees, shells, shared state and usage meters for parallel agents on Windows. Kept by decant (ADR 0008). | 2026-10-16 |
| `claude-swap.md` | The `cswap` account switcher: where its state and Claude Code's config live, `switch`, `run` and `CLAUDE_CONFIG_DIR`, `auto` and its exit codes, the `list --json` contract, the usage endpoint budget. | 2026-10-16 |
