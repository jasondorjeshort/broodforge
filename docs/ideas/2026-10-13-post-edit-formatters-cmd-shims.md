---
date: 2026-10-13
kind: bug
---

# post-edit formatters cannot run .cmd shims on Windows

`runFormatter` in `.claude/hooks/post-edit.js` calls `spawnSync` without a shell, which cannot run
`.cmd` shims such as `prettier` on Windows. Reproduce: put a `.cmd` shim in the FORMATTERS table
and edit a file on Windows; the formatter fails to spawn. A decanted project that fills the table
needs the `.exe` or `node <script>` form instead.
