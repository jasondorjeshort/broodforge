**Install a mingw-w64 C compiler on the PC so that `go test -race` can run there.** It waits on the owner because MSYS2 installs with elevation and the `PATH` change is on their account; agents cannot install programs on this PC. The reasoning is ADR 0023 (`docs/design/decisions/0023-language-for-the-long-term.md`, Decision, condition 1): the race detector needs cgo and a C compiler, the build itself does not, and until this is done `-race` runs on the droplet only.

Status: waiting by construction. Verified on 2026-10-16 by the critic: `go test -race` on the PC refuses with `go: -race requires cgo; enable cgo by setting CGO_ENABLED=1`, and no `gcc` is on the machine.

Runs as the owner. Elevated PowerShell for the first command; the MSYS2 UCRT64 shell for the second; the owner's own account for the `PATH` line (development and tests run under it; the `broodforge` instance account never runs tests and needs none of this). True first: nothing.

```
winget install --id MSYS2.MSYS2 -e
```

Then open "MSYS2 UCRT64" from the Start menu and run:

```
pacman -S --needed --noconfirm mingw-w64-ucrt-x86_64-gcc
```

Then, in a PowerShell as the owner, add the compiler to the user `PATH` (user scope, not machine; no other entry is touched):

```
[Environment]::SetEnvironmentVariable("Path", [Environment]::GetEnvironmentVariable("Path", "User") + ";C:\msys64\ucrt64\bin", "User")
```

Check it took, in a new PowerShell:

```
gcc --version
go env CGO_ENABLED
```

Expect a `gcc` version line (mingw-w64 runtime 8 or later is what the race detector needs; the current MSYS2 package is well past it) and `0` for the second (the build stays cgo-free; the test command sets `CGO_ENABLED=1` for the `-race` run only). Once `src/` exists, `CGO_ENABLED=1 go test -race ./...` in the repository is the real check. Allow 20 minutes; the MSYS2 download is a few hundred MB.

When done: delete this file; the next coder pass report states that `-race` ran on the PC, and the Windows host reference (when it exists) records the install under its Go section.
