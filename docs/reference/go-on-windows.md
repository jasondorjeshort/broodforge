---
verified: 2026-10-16
source: observed (Go toolchain and mingw-w64 on the owner's Windows 11 PC, 2026-10-16)
---

# Go on the Windows PC

The project's Go facts for the owner's Windows PC. The general Windows behaviour of Claude Code
(processes, worktrees, shells, npm, shared state) is the base's
`docs/hatchery/reference/claude-code-on-windows.md`; this file held its "Go on this PC" section
until the base's records moved to `docs/hatchery/` (ADR 0024).

- The C compiler for the race detector is MSYS2 UCRT64 gcc, `C:\msys64\ucrt64\bin\gcc.exe`, with
  `C:\msys64\ucrt64\bin` on the owner's user `PATH` (installed 2026-10-16).
- With a compiler present cgo defaults on (`go env CGO_ENABLED` prints `1`), so the build line
  forces `CGO_ENABLED=0 go build ./...` and the test line sets `CGO_ENABLED=1 go test -race ./...`
  (ADR 0023).
- `CGO_ENABLED=1 go test -race ./...` is the check that the compiler still works.
- A process started before the `PATH` change does not see gcc: `go env CGO_ENABLED` prints `0` and
  `go test -race` fails with `cgo: C compiler "gcc" not found`. Restart Claude Code, or put the
  directory first on `PATH` for the one command. PowerShell:
  `$env:Path = "C:\msys64\ucrt64\bin;" + $env:Path; $env:CGO_ENABLED = "1"; go test -race ./...`.
  Git Bash: `PATH="/c/msys64/ucrt64/bin:$PATH" CGO_ENABLED=1 go test -race ./...`.
- Reinstall: `winget install --id MSYS2.MSYS2 -e` (elevated PowerShell); in the "MSYS2 UCRT64" shell
  `pacman -S --needed --noconfirm mingw-w64-ucrt-x86_64-gcc`; then, in PowerShell as the owner, the
  user-scope append (no other entry is touched):
  `[Environment]::SetEnvironmentVariable("Path", [Environment]::GetEnvironmentVariable("Path", "User") + ";C:\msys64\ucrt64\bin", "User")`.
  Check in a new shell: `gcc --version`; `go env CGO_ENABLED` prints `1`.
- `filepath.EvalSymlinks` does not follow directory junctions (Go 1.27) while `os.Readlink` reads
  both junctions and symlinks; resolve links by hand where a junction matters
  (`docs/findings/2026-10-16-go-evalsymlinks-does-not-follow-windows-junctions.md`).
- A behavioural scanner on this PC took a fresh unsigned build and the Git tools its process tree
  ran; BullGuard was removed 2026-10-16. If tools vanish at the instant a run ends, check the
  scanner's quarantine first
  (`docs/findings/2026-10-16-bullguard-sentry-quarantined-the-binary-and-its-process-tree.md`).
- The `broodforge` instance account never runs tests and needs none of this.
