---
verified: 2026-10-26
source: observed (go.mod, go mod graph, and the LICENSE file of each module in the module cache)
---

# Go dependencies

Every module in `go.mod`, its pinned version, and its licence as read from the module's own
licence file. All are GPL-3.0-or-later compatible (ADR 0001). The direct modules are ADR 0023's;
the tool modules are held by `tool` directives; everything else is indirect and pinned by minimum
version selection from what the direct modules require. A pass that adds, removes or moves a
module updates this table in the same commit.

Toolchain: `go 1.27`, `toolchain go1.27.2` (a host with an older Go downloads 1.27.2 on first use
through `GOPROXY` and the checksum database).

## Direct (ADR 0023)

| Module | Version | Licence | Owner |
|---|---|---|---|
| `github.com/disgoorg/disgo` | v0.19.6 | Apache-2.0 | `discord` |
| `github.com/disgoorg/snowflake/v2` | v2.0.3 | Apache-2.0 | `discord` (disgo's id type) |
| `zombiezen.com/go/sqlite` | v1.4.2 | ISC | `store` |
| `golang.org/x/sys` | v0.49.0 | BSD-3-Clause | `spawn`, `control`, `setup` |

`control`, `spawn` and `setup` import `golang.org/x/sys/windows` (Windows build only; `go mod tidy`
counts every platform, so the module stays pinned), and `setup` also `golang.org/x/sys/unix` (Linux
build only) for the terminal's echo during a hidden answer; `store` imports `zombiezen.com/go/sqlite`
and its `sqlitex`, and no other package does; `discord` imports disgo's `discord` (the message,
component, interaction and permission types), `gateway` and `rest` packages and the snowflake
id type, and no other package does. It does not import disgo's `bot` package or its root,
which pull in `voice`: so godave and `golang.org/x/crypto` are not in `go.mod`. No module is
held by a blank import any more; every one is imported by the package that owns it.

zombiezen facts (finding `docs/findings/2026-10-16-sqlite-facts-seen-while-building-the-store.md`):

- Run each `PRAGMA` as its own statement; inside `sqlitex.ExecuteScript` (a SAVEPOINT) `synchronous` fails.
- A read-only open of a WAL store creates `-wal` and `-shm` and leaves them after close; a truncated file opens and fails only at its first statement.

`x/sys/windows` pitfalls in the named-pipe transport (finding
`docs/findings/2026-10-16-go-named-pipes-on-windows-two-stdlib-pitfalls.md`):

- Do not wrap a pipe handle with `os.NewFile` when a reader and a writer run together: `-race`
  reports a data race on the shared offset in the standard library's poll.FD. Use overlapped I/O
  directly.
- Keep every `windows.Overlapped` and its I/O buffer on the heap until the operation completes:
  a goroutine stack moves, and the kernel completes into the old copy (`ERROR_IO_INCOMPLETE`).
- Pass `WaitForMultipleObjects` a handle array held on the heap (a struct field, as
  `pipe_windows.go` line 56 does), never a slice literal: the literal stays on the stack, and
  x/sys v0.49.0 calls another Go function between taking its address and the syscall, so a stack
  move hands the kernel freed memory (`ERROR_INVALID_HANDLE`; `TestStressUnderGC` catches it).

## Tools (`go tool <name>`)

| Module | Version | Licence | Tool |
|---|---|---|---|
| `honnef.co/go/tools` | v0.8.2 (tag 2026.2.2) | MIT | `staticcheck` |
| `github.com/nishanths/exhaustive` | v0.15.0 | BSD-2-Clause | `exhaustive` |

## Indirect

| Module | Version | Licence | Required by |
|---|---|---|---|
| `github.com/gorilla/websocket` | v1.5.3 | BSD-2-Clause | disgo (gateway) |
| `github.com/disgoorg/json/v2` | v2.0.0 | Apache-2.0 | disgo |
| `github.com/disgoorg/omit` | v1.0.0 | Apache-2.0 | disgo |
| `github.com/klauspost/compress` | v1.18.4 | BSD-3-Clause; `gzhttp/` Apache-2.0, two `s2/cmd` dirs MIT | disgo |
| `github.com/sasha-s/go-csync` | v0.0.0-20240107134140-fcbab37b09ad | Apache-2.0 | disgo |
| `golang.org/x/text` | v0.34.0 | BSD-3-Clause | zombiezen sqlite's tests (kept at the version x/crypto selected before it left) |
| `modernc.org/sqlite` | v1.37.1 | BSD-3-Clause | zombiezen sqlite |
| `modernc.org/libc` | v1.65.7 | BSD-3-Clause | modernc sqlite |
| `modernc.org/mathutil` | v1.7.1 | BSD-3-Clause | modernc |
| `modernc.org/memory` | v1.11.0 | BSD-3-Clause | modernc |
| `github.com/dustin/go-humanize` | v1.0.1 | MIT | modernc |
| `github.com/google/uuid` | v1.6.0 | BSD-3-Clause | modernc |
| `github.com/mattn/go-isatty` | v0.0.20 | MIT | modernc |
| `github.com/ncruces/go-strftime` | v0.1.9 | MIT | modernc |
| `github.com/remyoudompheng/bigfft` | v0.0.0-20230129092748-24d4a6f8daec | BSD-3-Clause | modernc |
| `golang.org/x/exp` | v0.0.0-20250408133849-7e4ce0ab07d0 | BSD-3-Clause | modernc, staticcheck |
| `github.com/BurntSushi/toml` | v1.4.1-0.20240526193622-a339e1f7089c | MIT | staticcheck |
| `golang.org/x/exp/typeparams` | v0.0.0-20231108232855-2478ac86f678 | BSD-3-Clause | staticcheck |
| `golang.org/x/mod` | v0.41.0 | BSD-3-Clause | staticcheck, exhaustive |
| `golang.org/x/sync` | v0.23.0 | BSD-3-Clause | staticcheck, exhaustive |
| `golang.org/x/tools` | v0.51.0 | BSD-3-Clause | staticcheck, exhaustive |

## Commands that work

- Latest tag of a module: `curl -s https://proxy.golang.org/<module>/@latest`.
- Who requires a module: `go mod graph | grep ' <module>@'`.
- Licence file of a module on Windows: `cygpath -u "$(go list -m -f '{{.Dir}}' <module>)"`, then
  read its `LICENSE` or `COPYING`.
- Every package the binary links: `go list -deps ./cmd/broodforge`.

## Gotchas

- `go list -m all` also lists modules that only the dependencies' own tests need (testify,
  go-cmp, pprof and others); they are in `go.sum` but not in `go.mod` and not in the binary.
- disgo's root package and its `bot` package pull in its voice package and with it godave and
  x/crypto; `discord` uses `gateway` and `rest` directly, without `bot`, and links neither.
- In Git Bash, `read` without `-r` eats the backslashes of a Windows module-cache path.
