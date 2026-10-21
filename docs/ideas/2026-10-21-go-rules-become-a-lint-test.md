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

Plan:
- Architect, one sitting: move the "Import exception" list into the `cli` row of
  `docs/design/03-architecture.md`; `go.md` keeps one line saying a package imports only the
  components its row names and the `cli` row lists its exceptions.
- Then one coder brief: one Go test under `tests/lint/` runs `go list -json ./...` and a file scan
  for the four rules, so `go test ./...` enforces them and pusher check 8 needs no new hook.
- Reviewer reads it.
