---
date: 2026-10-17
kind: feature
related: []
---

# importfixture refuses the Git Bash home form instead of rewriting it

The import rewrite covers `C:/Users/<user>` in its Windows spellings and the engine's
`C--Users-<user>` slug, and its leak scan refuses anything else that still names the user. The Git
Bash spelling `/c/Users/<user>/...` is one of those: a recording whose Bash call names a path under
the home cannot be imported at all. It happened on 2026-10-17: the first closing recording
(`~/bf-p7/closing`, a `tool_deferred` turn whose pending Bash began with
`cd /c/Users/<user>/bf-p7/tree`) exits 1 at turn-1 line 5, which is why `fixtures/engine/closing`
comes from the second recording. The reviewer also noted URL-encoded (`C%3A%5CUsers`) and
lower-case slug forms are refused, and a JSON `\u` escape of the user name would pass both the
rewrite and the scan; the engine does not emit that for ASCII.

What it would take: a third path form in `rewritePaths` (`/c/Users/<user>` to `/home/user`, same
token-ending rules), a test pinning it, and the `basic`-style re-import check that the ten spike
fixtures stay byte-identical. Anchor: `internal/tools/importfixture/main.go` (`rewritePaths`,
`matchPath`, `scanLeak`), `main_test.go` (the test that pins the MSYS form as refused).

## Open

- Whether a deferred-Bash recording is wanted as a fixture at all; the `defer` fixture already covers the shape.
