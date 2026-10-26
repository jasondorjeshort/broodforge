---
verified: 2026-10-26
source: observed (Go toolchain and mingw-w64 on the owner's Windows 11 PC, 2026-10-26)
---

# Go on the Windows PC

The project's Go facts for the owner's Windows PC. The general Windows behaviour of Claude Code
(processes, worktrees, shells, npm, shared state) is the base's
`docs/hatchery/reference/claude-code-on-windows.md`; this file held its "Go on this PC" section
until the base's records moved to `docs/hatchery/` (ADR 0024).

- Go is installed system-wide: `C:\Program Files\Go\bin\go.exe` (go1.27.0). A shell or Claude Code
  started before the install has a stale `PATH` and does not find `go`; open a new terminal or
  restart Claude Code. `winget install GoLang.Go` can fail with `0x80071130 : Fast Cache data not
  found` (winget's source index); the installer from go.dev/dl, run elevated, is the fallback.
- `go.mod` says `go 1.27` and `toolchain go1.27.2`. With the default `GOTOOLCHAIN=auto` the first
  `go` command in the repo downloads go1.27.2 into `%USERPROFILE%\go\pkg\mod\golang.org\toolchain@...`
  (network needed once) and switches to it, so `go env GOROOT` shows that path, not
  `C:\Program Files\Go`. `GOTOOLCHAIN=local` pins the installed 1.27.0, which works (`go list ./internal/setup`
  passes): `go 1.27` allows 1.27.0, and the `toolchain` line is a preference that auto mode follows by
  downloading. `CGO_ENABLED=0 go build ./...` and `go vet ./...` pass from the repo root.
- The C compiler for the race detector is MSYS2 MinGW64 gcc, `C:\msys64\mingw64\bin\gcc.exe`; put
  `C:\msys64\mingw64\bin` on `PATH` for the test line.
- With a compiler present cgo defaults on (`go env CGO_ENABLED` prints `1`), so the build line
  forces `CGO_ENABLED=0 go build ./...` and the test line sets `CGO_ENABLED=1 go test -race ./...`
  (ADR 0023).
- `CGO_ENABLED=1 go test -race ./...` is the check that the compiler still works.
- Without gcc on `PATH`, `go env CGO_ENABLED` prints `0` and `go test -race` fails with
  `cgo: C compiler "gcc" not found`. Put the directory first on `PATH` for the one command.
  PowerShell:
  `$env:Path = "C:\msys64\mingw64\bin;" + $env:Path; $env:CGO_ENABLED = "1"; go test -race ./...`.
  Git Bash: `PATH="/c/msys64/mingw64/bin:$PATH" CGO_ENABLED=1 go test -race ./...`.
- Reinstall: `winget install --id MSYS2.MSYS2 -e` (elevated PowerShell); in the "MSYS2 MINGW64"
  shell `pacman -S --needed --noconfirm mingw-w64-x86_64-gcc`. Check: `gcc --version`.
- `filepath.EvalSymlinks` does not follow directory junctions (Go 1.27) while `os.Readlink` reads
  both junctions and symlinks; resolve links by hand where a junction matters
  (`docs/findings/2026-10-16-go-evalsymlinks-does-not-follow-windows-junctions.md`).
- A behavioural scanner on this PC took a fresh unsigned build and the Git tools its process tree
  ran; BullGuard was removed 2026-10-16. If tools vanish at the instant a run ends, check the
  scanner's quarantine first
  (`docs/findings/2026-10-16-bullguard-sentry-quarantined-the-binary-and-its-process-tree.md`).
- The `broodforge` instance account never runs tests and needs none of this.
