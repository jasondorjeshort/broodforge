---
date: 2026-10-23
kind: bug
---

# Carry back: a backslash `@` import escapes the tree in imports.js on Windows

Anchor: `targetOf` and `absOf` in `.claude/hooks/imports.js`, and the IM cases of
`.claude/hooks/test-hooks.js`.

`targetOf` treats a token as outside only when it starts with `~`, `/`, `\` or a drive, or when its
`path.posix` normal form climbs (`../`). A token holding a backslash elsewhere stays one posix name,
so `@.claude\..\..\outside.md` is `outside: false`; then `absOf` joins it with the host's `path`,
which on Windows reads the backslashes as separators, and the file above the repository root is
stat'ed, counted, and read for its own imports. The header promises a target outside the root is
"never stat'ed, read or followed".

Reproduce on the PC: a root holding `CLAUDE.md` with the line `@.claude\..\..\outside.md` and a file
`outside.md` beside the root; `require('./.claude/hooks/imports.js').resolve(root, 'CLAUDE.md')`
returns that import with `outside: false, exists: true` (observed 2026-10-23).

This project's Go port (`internal/workflow/imports.go`, ADR 0025's workflow copy) closes it by
treating any token holding a backslash as outside and any target `filepath.IsLocal` refuses as
outside. The same two rules, with an IM case for each, would close it in the base: in `targetOf`,
a token containing `\` is outside; after normalizing, a target with a `:` or a reserved device name
is outside on Windows.
