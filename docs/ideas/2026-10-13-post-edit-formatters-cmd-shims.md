---
date: 2026-10-13
kind: bug
---

# post-edit formatters cannot run .cmd shims on Windows

`runFormatter` in `.claude/hooks/post-edit.js` calls `spawnSync` without a shell, which cannot run
`.cmd` shims such as `prettier` on Windows. Reproduce: put a `.cmd` shim in the FORMATTERS table
and edit a file on Windows; the formatter fails to spawn. A decanted project that fills the table
needs the `.exe` or `node <script>` form instead.

Plan: `coder`. In `.claude/hooks/post-edit.js` `runFormatter` (line 216), on win32 when `argv[0]` has no extension or ends in `.cmd` or `.bat`, run it through `cmd.exe /d /s /c` with each argument quoted, so a `prettier` shim spawns; `.exe` and `node <script>` forms stay as they are, and the failure text is unchanged. Update the FORMATTERS comment (lines 48-51). Add a harness case in `.claude/hooks/test-hooks.js` that puts a `.cmd` formatter in the test seam, runs on Windows and prints a skip line elsewhere. Verify: `node .claude/hooks/test-hooks.js`. Out of scope: any formatter in the base table (it stays `{}`).
