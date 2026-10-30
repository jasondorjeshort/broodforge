---
date: 2026-10-21
kind: bug
related: [2026-10-16-git-guard-false-positives-beside-variable-expansions.md]
---

# Carry back: a quoted string that merely contains git does not name git

Anchor: `decide` and `opaqueCommand` in `.claude/hooks/git-guard.js`, the word arrays of
`.claude/hooks/shell-parse.js`, and the rule 19 sentence of
`docs/hatchery/decisions/0002-hook-specs.md` ("in a line that names git").

The guard's "the line names git" is a substring match over the raw command text, so a regex such as
`'(node_modules|AppData|\.git)'` inside a `powershell -Command` string, or a grep for `'\.git'`,
makes the line name git, and a pipeline stage whose command word is an expansion (`$j.bin`) is then
denied under rule 19. This project fixed it on 2026-10-16 (commits `14304c6` and `5288955`): the
parser keeps `quoted[k]` and `bare[k]` per word, and a word names git when it is unquoted with a
git token, when its unquoted part has one, when it is quoted and is git or a path ending in git
(spaces allowed, as in the default Git for Windows path), or when it is an assignment whose quoted
value starts with one; a quoted argument such as `"git push"` to grep does not. Harness cases for
both the denials kept and the allowances went with it, and the spec sentence above.

The update to Hatchery `dc896c3` (ADR 0024) took the base's guard, parser and harness, which
rewrote rule 20 around `privacy-scan.js`; the fix did not survive the three-way apply and was
dropped rather than re-grafted by hand into a fail-closed hook. The false denies are back.

The harness cases are `GR24a` to `GR24k` in those commits. The pair that states the boundary: in
Bash, `grep -r '\.git' . | $filter` is allowed (the base guard at `dc896c3` denies it under rule 19);
in PowerShell, `$g='C:\Program Files\Git\cmd\git.exe'; & $g push -f` stays denied under rule 19.
Those commits' edits to `.claude/rules/README.md` and `.claude/rules/hooks.md` are not part of the
carry-back.

Plan: Send is done: the branch `carry-back/git-guard-quoted-git` was sent at `fffd564` with the
extended `namesGit(cmd, captured)` (not the `5288955` version), harness cases `GR24a` to `GR24z`,
and the 0002-hook-specs sentence. What remains is Land in Hatchery (`carry-back` skill, Land), then
`update-from-hatchery` here.
