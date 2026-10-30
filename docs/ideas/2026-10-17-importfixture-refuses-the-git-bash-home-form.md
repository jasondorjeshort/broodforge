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

Plan: tester, one pass. `internal/tools/importfixture/main.go`: a third home form in
`matchPath`, `/<drive letter>/Users/<user>` (MSYS: `/` separators only, the leading `/` must not
follow a word byte, the same segment-ending and spike-root rules, `/home/user` or
`/home/user/bf-spike` out); the slug rewrite needs nothing new. `main_test.go`: the
`/c/Users/alice/x` cases in `TestImportRecordedRefusals` and `TestSurvivingUserNameRefused` flip
from refused to rewritten (`/home/user/x`), `/c/Users/bob/x` stays refused in
`TestOtherUsersHomeRefused`, and one new case puts `cd /c/Users/alice/bf-p7/tree` inside a JSON
string. Verify: `go test ./internal/tools/importfixture/`, then re-import one spike fixture into
a temp directory and `diff -r` it against its `fixtures/engine/<name>` (byte-identical). Out of
scope: URL-encoded and `\u`-escaped forms (the user-name scan refuses the first; the engine does
not emit the second for ASCII), and whether a deferred-Bash recording becomes a fixture.
