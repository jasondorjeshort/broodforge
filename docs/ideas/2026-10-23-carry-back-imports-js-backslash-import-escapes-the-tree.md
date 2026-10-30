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

Plan: coder, `carry-back` skill Send (steps 1 to 3 and 5), in a worktree from `hatchery/main` (step
1 fetches it). In `targetOf`: a token with `\` is outside; so is a target with a `:` in any segment
or a device-name segment (CON, PRN, AUX, NUL, COM1-9, LPT1-9, any extension or case), on every host;
header comment (15-17) says so. Harness, beside IM13: IM14 (`@.claude\..\..\outside.md`,
`outside.md` beside the scratch root: `outside: true, exists: null`), IM15 (`@a:b.md`, `@nul.md`).
`docs/hatchery/decisions/0016-update-procedure.md`: the "(`../x`, `~/x`, an absolute path)" sentence
gains both rules. Verify: `node .claude/hooks/test-hooks.js` green; one commit `Carry-back:
imports.js: a backslash or device-name import is outside the tree`. Not: this project's copy
(returns by `update-from-hatchery`); the Go port (`internal/workflow/imports.go` line 24,
`filepath.IsLocal`) refuses both on this host only; the base rule matches it on every host.
