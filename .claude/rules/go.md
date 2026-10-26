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
- Build before test: `CGO_ENABLED=0 go build ./...`, the check line, then `CGO_ENABLED=1 go test -race ./...` on
  Linux and on the PC (there with `PATH="/c/msys64/ucrt64/bin:$PATH"` for gcc).
- The binary is `CGO_ENABLED=0 go build -o bin/ ./cmd/broodforge`: the trailing slash makes Go
  add `.exe` on Windows, where `-o bin/broodforge` writes a file PowerShell will not run.
  `CGO_ENABLED` is set on the command line, never in `go.mod`. Nothing goes on `PATH`; never
  `go install`. Exit codes are proven against `bin/`, never `go run`, which always exits 1 on a
  non-zero child.
- Before editing `internal/control`, `internal/spawn` or `internal/hook`, read
  `docs/reference/go-on-windows.md` and `docs/reference/go-dependencies.md` (named pipes,
  junctions, gcc on `PATH`, antivirus quarantine of `bin/`).
- No `interface{}` or `any` outside `internal/engine`'s raw stream layer.
- Sum types are a struct with a `Kind` enum field and a pointer payload per kind, switched on
  `Kind` so `exhaustive` checks every switch. Never a type switch for them.
- Tests use `testing` only, beside the code as `_test.go`; fixtures and scripts live under
  `tests/` and `fixtures/`.
- Tests do not depend on `BROODFORGE_*` in the process environment (a developer inside a session has
  it set): build the child's environment with `clitest.Environ`, or `t.Setenv(name, "")`. Check with
  `BROODFORGE_SESSION=s0001 go test -count=1 ./...`: the test cache does not track a child's environment, so a cached
  pass proves nothing.
- `json.Decoder.DisallowUnknownFields` on configuration files only, never on the engine stream:
  an unknown field or kind there is read, not refused.
- No host branching (`runtime.GOOS`, `_windows.go`, `//go:build windows`) outside `spawn`, the
  worktree removal it owns, `control`, and `setup`'s `host_windows.go` and `host_linux.go` (the
  elevated part and hidden terminal input are host code by nature; no other file of `setup` branches).
- One package per component of `docs/design/03-architecture.md`'s table, under `internal/`; a
  package imports only the components its row names. A library is imported only by the component
  ADR 0023 names; `docs/reference/go-dependencies.md` changes with `go.mod`.
- Import exception: `internal/cli` also imports `engine`, `config` and `hook`, for the standalone
  subcommands the plan places there (engine run, record, replay; config check; hook pre-tool-use,
  try, cases), and `store`, `audit`, `backup` and `credentials`, for the `(file)` commands and the
  read commands' fallback to the store file, and `spawn`, for `spawn preflight` and the backend
  the standalone engine commands start, `session`, for `serve`, which builds the session service
  and serves it, `discord`, for `serve`'s adapter, the standalone `discord render`
  and `setup`'s Discord client and its scripted fake (`BROODFORGE_SETUP_FAKE_DISCORD`), and `git`
  and `policy`, for the worktree handlers and the hidden `git-askpass` and `git-commit-msg`, and
  `workflow`, for the `(file)` commands `workflow render` and `probe`, the handoff's path to a
  session's copy, and `serve`'s copy and probe at session start (the session service's
  `Config.Workflow`); nothing else beyond what 03's `cli` row names.
