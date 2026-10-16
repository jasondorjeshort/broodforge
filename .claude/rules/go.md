---
paths:
  - "**/*.go"
  - "go.mod"
  - "go.sum"
---

# Go conventions

- First line of every `.go` file: `// SPDX-License-Identifier: GPL-3.0-or-later`.
- Format with `gofmt`. Clean under the check line, run from the repository root:
  `go vet ./... && go tool staticcheck ./... && go tool exhaustive ./...`.
- Build before test: `go build ./...`, the check line, then `CGO_ENABLED=1 go test -race ./...` on
  Linux and on the PC (there with `PATH="/c/msys64/ucrt64/bin:$PATH"` for gcc).
- The binary is `CGO_ENABLED=0 go build -o bin/ ./cmd/broodforge`: the trailing slash makes Go
  add `.exe` on Windows, where `-o bin/broodforge` writes a file PowerShell will not run.
  `CGO_ENABLED` is set on the command line, never in `go.mod`. Nothing goes on `PATH`; never
  `go install`. Exit codes are proven against `bin/`, never `go run`, which always exits 1 on a
  non-zero child.
- No `interface{}` or `any` outside `internal/engine`'s raw stream layer.
- Sum types are a struct with a `Kind` enum field and a pointer payload per kind, switched on
  `Kind` so `exhaustive` checks every switch. Never a type switch for them.
- Tests use `testing` only, beside the code as `_test.go`; fixtures and scripts live under
  `tests/` and `fixtures/`.
- `json.Decoder.DisallowUnknownFields` on configuration files only, never on the engine stream:
  an unknown field or kind there is read, not refused.
- No host branching (`runtime.GOOS`, `_windows.go`, `//go:build windows`) outside `spawn`, the
  worktree removal it owns, and `control`.
- One package per component of `docs/design/03-architecture.md`'s table, under `internal/`; a
  package imports only the components its row names. A library is imported only by the component
  ADR 0023 names; `docs/reference/go-dependencies.md` changes with `go.mod`.
- Import exception: `internal/cli` also imports `engine`, `config` and `hook`, for the standalone
  subcommands the plan places there (engine run, record, replay; config check; hook pre-tool-use,
  try, cases), and `store`, `audit`, `backup` and `credentials`, for the `(file)` commands and the
  read commands' fallback to the store file; nothing else beyond what 03's `cli` row names.
