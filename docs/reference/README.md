# Reference

Durable facts the project depends on: how a tool behaves, what an API accepts, what an environment
provides. One topic per file, named by topic (`git-credentials.md`, `android-sdk.md`), never by date.
Each file starts with frontmatter `verified: YYYY-MM-DD` and `source: <url or "observed">`, then
facts, commands that work, and gotchas. Cap 300 lines; split by topic when near it.

A fact gets here by being promoted from a finding (`finding` skill) or written directly when it is
already certain. When a fact turns out to be wrong, fix it and bump `verified:`; do not append history.

| File | Topic | Verified |
|---|---|---|
| `go-on-windows.md` | The Go toolchain on the owner's PC: gcc for `-race`, the `CGO_ENABLED` lines, junctions, the scanner. Claude Code's own Windows behaviour is `../hatchery/reference/claude-code-on-windows.md`. | 2026-10-16 |
| `go-dependencies.md` | Every module in `go.mod`: version, licence, which component or tool needs it; commands to check one. | 2026-10-16 |
| `claude-code-flags.md` | The engine flags the adapter passes (`--model`, `--effort`, `--resume`): what each takes, how to check them against a new engine version. | 2026-10-23 |
