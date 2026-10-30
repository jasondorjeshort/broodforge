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

Plan: coder, `carry-back` skill Send (steps 1 to 3 and 5; the runner 4 and 7, the pusher 6), in a
worktree cut from `hatchery/main` (`../hatchery` sits at that sha; the base's `imports.js` is byte-identical
to ours, so no departure row exists yet). In `targetOf`: a token containing `\` is outside; a normalized
target with a `:` in any segment, or a segment that is a reserved device name (CON, PRN, AUX, NUL, COM1-9,
LPT1-9, with or without an extension, any case), is outside on every host; the header comment (lines 15-17)
says so. Harness, beside IM13: IM14 (`@.claude\..\..\outside.md` with `outside.md` beside the scratch root
resolves `outside: true, exists: null`) and IM15 (`@a:b.md`, `@nul.md`). Spec: the "(`../x`, `~/x`, an
absolute path)" sentence of `docs/hatchery/decisions/0016-update-procedure.md` gains both rules. Verify:
`node .claude/hooks/test-hooks.js` green in the worktree on the PC; one commit `Carry-back: imports.js: a
backslash or device-name import is outside the tree`; the row's Path is `.claude/hooks/imports.js`. Out of
scope: this project's copy (it returns by `update-from-hatchery`) and the Go port, which has both rules.
