---
date: 2026-10-21
kind: chore
related: [../findings/2026-10-21-claude-code-best-practices-for-a-build-phase-go-project.md]
---

# The mechanical rules of `go.md` become one lint test

`.claude/rules/go.md` states rules a check could enforce: the SPDX first line, no host branching
(`runtime.GOOS`, `_windows.go`, `//go:build windows`) outside `internal/spawn` and
`internal/control`, no `interface{}` or `any` outside `internal/engine`'s raw layer, and a package
imports only the components its row in `docs/design/03-architecture.md` names. Each is prose the
coder must remember and the reviewer must grep for; `go vet` and `staticcheck` enforce none of
them. The "Import exception" paragraph restates the `cli` row of 03 with eight clauses and drifts
whenever a package is added, and it loads on every `.go` read.

2026-10-30 (architect): the first half is done. The "Import exception" paragraph is gone from
`.claude/rules/go.md`, whose package rule now says the `cli` row of `03-architecture.md` lists that
package's exceptions; the row carries them (`engine`, `config`, `hook`, `store`, `audit`, `backup`,
`spawn` added to what it named), one line per component with the commands that need it.

Plan: coder, one brief: one Go test under `tests/lint/` [no-check: created by the pass] runs `go list -json ./...` and a file scan
for the four rules of `.claude/rules/go.md`: the SPDX first line; no host branching outside `spawn`,
the worktree removal it owns, `control` and `setup`'s `host_windows.go` and `host_linux.go`; no
`interface{}` or `any` outside `internal/engine`'s raw layer (`parse.go`); a package imports only the
components its row in `03-architecture.md` names, the `cli` row's exceptions included. The allowed
import graph is data in the test, one entry per row, with a comment naming the row it mirrors; a
package or import the table does not know fails the test with the row to edit. `go test ./...` then
enforces all four and pusher check 8 needs no new hook; a `tests/README.md` row. Reviewer reads it.
