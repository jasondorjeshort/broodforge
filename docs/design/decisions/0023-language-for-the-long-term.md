# 0023. Language for the long term: Go, disgo, zombiezen SQLite, one static binary per host

Date: 2026-10-16
Status: accepted

Record: proposal `../proposals/0023-language-for-the-long-term.md` and
`../proposals/0023-critic-review.md` (its verification table is authoritative on library facts).
Amends ADR 0020 (Language bundle; deferred SDK triggers), ADR 0021 (the binding) and
`0022-linux-spawn-and-handoff.md` (the runtime bind and the entry wrapper); their Amendments
sections point here. The engine mechanism of ADR 0020 is untouched.

## Context

The owner's principle of 2026-10-16 (`../00-vision.md`, Principles) puts scalability and
maintainability above time to first release, names three growth axes (sessions and users per
instance, codebase longevity, engines and providers) and says the language is judged on
mechanical properties. ADR 0020 had picked TypeScript on Node for the opposite weighting. The
proposal scored seven languages on thirteen criteria; the critic verified its library and toolchain claims on the PC.

Verified (sources in the review): Go 1.27.2 is current, the PC has 1.27.0, and a `toolchain` pin
newer than the install is downloaded through the proxy and checked by the sumdb; `go test -race`
needs cgo and a C compiler, which the PC lacks; a Go static hook starts in 9 ms on the PC against
48 ms for a Node script, both three orders below the hook's 30 s timeout; disgo (Apache-2.0) is
pre-1.0 with one author for every 2026 release and requires gorilla/websocket (BSD-2-Clause, last
release 2024); arikawa (ISC) is maintained by one person; `modernc.org/sqlite` (BSD-3-Clause) is
cgo-free on both hosts and `zombiezen.com/go/sqlite` (ISC) over it exposes `SetAuthorizer`,
`OpenWAL` and `SetBusyTimeout`; `golang.org/x/sys/windows` (BSD-3-Clause) exports the job object
calls and `os/exec` cannot suspend-then-assign; Go 1.27 backs `encoding/json` with v2 and runs
the `stdversion` vet check; staticcheck is MIT and exhaustive BSD-2-Clause; Discord allows 50
requests per second per bot. Hundreds of concurrent turns are decided by the engine's own memory,
not the bot's language, so the sessions axis separated nothing between the top three.

The coordinator's decision. Go over TypeScript on the owner's longevity words as verified facts:
a dozen modules against hundreds, a toolchain stable for a decade, seconds-long feedback, and one
static binary that simplifies the Linux sandbox and the Windows instance. Go over Rust with type
strength and per-change cost weighted equally, as the principle lists them together: three of the
four longevity sub-criteria (small dependency tree, stable toolchain, cheap refactors) favour Go,
type strength favours Rust, and agent-driven development, where every change is written and
verified in an agent loop, makes minutes-long builds and borrow-checker token cost a recurring
tax that a lint layer in Go does not carry. The owner was told the switching cost; his principle
says it does not decide.

Switching cost, recorded: `../03-architecture.md` (the process sentence, the `Engine` snippet,
`paths.node`), `../05-debug-interface.md` and `../05-cli-conventions.md` (command classes, the
`sh` half of the entry point), `../07-spike-plan.md` (its `spike/` section stays as the record),
`../08-implementation-plan.md` (every pass), the 0022 companion's bind list and entry wrapper,
and `spike/run.ts`'s parser re-done in Go against the same fixtures: about two architect passes,
one tester pass and one extra coder pass.

## Decision

Language and toolchain. Go, with `go 1.27` and `toolchain go1.27.2` in `go.mod` (the pin moves
by a commit; `GOTOOLCHAIN=auto` downloads a newer pin through `GOPROXY` and the sumdb, and a host
that is offline or has `GOSUMDB=off` fails naming the version: the lift is installing that Go or
`GOTOOLCHAIN=local` with the pin lowered, written in each host reference). One module, one binary
`cmd/broodforge`, every subcommand of `../05-debug-interface.md` inside it, the hook and
`engine-entry` included. `CGO_ENABLED=0` for the binary on both hosts; no C compiler is needed to
build. Lint tools pinned by `tool` directives in `go.mod`: `honnef.co/go/tools` (staticcheck, MIT)
and `github.com/nishanths/exhaustive` (BSD-2-Clause). Commands, identical on both hosts:

```
CGO_ENABLED=0 go build -o bin/ ./cmd/broodforge      # bin/broodforge.exe on Windows
go vet ./... && go tool staticcheck ./... && go tool exhaustive ./...   # check
go test ./...                                         # test; fixtures replayed
CGO_ENABLED=1 go test -race ./...                     # Linux always; PC after the mingw entry
go run ./cmd/broodforge <args>      # from source; same usage output, exit code is the binary's only
```

The droplet builds its own binary from the pinned source (network for the toolchain download);
cross-compiling from the PC (`GOOS=linux GOARCH=amd64`) is the fallback when it cannot.

Libraries, each the only importer being the 03 component named. Discord: `disgo` pinned by
version in `go.mod` with its `gorilla/websocket` requirement listed in the dependency record;
imported only by `discord`, whose surface is gateway events, components and the REST calls 03
and 04 name, so a swap touches one package. Exit: `arikawa` (ISC); last resort: an in-house
gateway and REST client against Discord's documented API. SQLite: `zombiezen.com/go/sqlite` (ISC)
over `modernc.org/sqlite` (BSD-3-Clause), imported only by `store`: one writer connection owned
by the bot (ADR 0021, one writer) and a small read pool for the CLI's read commands; every
connection sets `journal_mode=WAL`, `synchronous=FULL`, `foreign_keys=ON` and the busy timeout at
open, since PRAGMAs are per connection; `SetAuthorizer` denies `UPDATE`, `DELETE` and `DROP
TRIGGER` on `audit_event` on the bot's connection (ADR 0021's optional hardening, now taken);
no `database/sql`. Process: `os/exec` with `SysProcAttr`, `golang.org/x/sys/windows` for job
objects, `syscall.Exec` on Linux for `engine-entry`. Standard library for `flag`, `net/http`,
`encoding/json`, `crypto/aes` with `cipher.NewGCM`, `net` (unix socket; the named pipe through
`x/sys/windows` or a small MIT package, the coder's call, imported by `control` only), `testing`.

Windows process control. At start the bot creates a job object with kill-on-close and assigns
itself, so every engine it spawns inherits the job and a crashed bot ends every engine (closing
the Windows gap of the 0022 companion). Each engine process is also assigned to its own
kill-on-close job right after `Start`; the window before assignment is microseconds and the
engine spawns nothing in its first hundreds of milliseconds, so it is accepted. Interrupt on
Windows closes that job; `taskkill /T` is the fallback when the assignment failed. Linux is
unchanged: SIGINT through the 0022 companion's signal path.

Types and JSON. Sum types (`EngineEvent`, `SessionEvent`) are a struct with a `Kind` enum field
and a pointer payload per kind, switched on `Kind`, so `exhaustive` checks every switch; type
switches on interfaces are not used for them. `encoding/json` decodes the engine stream into
tagged structs that never refuse an unknown field or kind (03 reads unknown kinds as `unknown`);
`DisallowUnknownFields` is set on configuration files only. The test command replays every
recorded fixture, so a toolchain change that alters what the parser accepts fails there first.

Facts recorded, not reasons: hook startup 9 ms Go against 48 ms Node on the PC; the Go hook is
a 4 MB binary with a 9 MB working set. Closed by this ADR: ADR 0020's deferred Agent SDK
implementation (the SDK ships for TypeScript and Python only; a Node sidecar is a second runtime
on both hosts and is rejected) and its mid-turn injection trigger "the SDK is built", which
becomes "a long-lived engine process per session is adopted under mechanism B", a supersession of
0020's process model. A raw-API second engine (R-42) is unaffected and plugs into the `Engine`
interface as a package.

Out of scope: the engine mechanism, the command line and the hook contract (0020, 03); the debug
commands (05); the words users see (04). Deferred, with triggers: a second Discord library in the
tree (disgo stops releasing or loses Discord API coverage 04 needs); the in-house client (arikawa
fails the same way); `-race` on the PC (the mingw entry, below); a per-session job object tree
report (the owner asks what a crashed bot left behind).

Conditions carried from the review, each binding on the implementation:

1. `go test -race` runs on Linux always and on the PC only after
   `../../operator/2026-10-16-pc-mingw-for-go-race.md` is done; a pass report says which ran.
2. The Discord exit above; disgo and its websocket dependency pinned.
3. zombiezen over modernc, pool and PRAGMA rules above.
4. Job objects through x/sys with the assign-after-start window accepted.
5. Unknown fields on the engine stream are never refused.
6. ADR 0020's SDK route and injection trigger are closed (0020, Amendments).
7. The hook timing is a fact in Context, not a reason.
8. The switching cost is recorded in Context.

## Consequences

Easier: one static binary per host, so the droplet sandbox binds `<engine>` and `<bf>` only and
the entry wrapper is the binary itself; the PC instance needs no Node, npm or native rebuild (npm
remains only if the engine is installed that way; the native installer avoids it); builds and
tests in seconds; a dependency list a reviewer reads in one sitting; a job object on Windows; a
crashed bot takes its engines with it; the audit authorizer is available.

Harder or costly: the switching cost in Context before any `src/` pass; Go's compiler does not
enforce exhaustiveness, nil safety or error handling, so the lint set is part of the test command
and never optional; the Discord library is one maintainer deep and the exit must stay cheap, which
is why `discord` is the only importer; `-race` on the PC waits on an operator step and the
droplet needs `gcc` (added to the droplet operator entry when it is rewritten for Go); the agent
SDK shape is gone from the engines axis; every design doc and the plan carry a rewrite pass.

Must be maintained: the `toolchain` pin and the tool pins; the fixture replay on every toolchain
move; the dependency record naming each module and its licence (N-24); the Discord surface list
in 03 that bounds a swap; the host references' toolchain-download lift.

Supersede when: the owner reverses the principle of 2026-10-16; disgo and arikawa both stop and
the in-house client is judged too costly; a verified Go-side Agent SDK appears and the owner wants
the in-process shape back; the lint set is shown to miss a class of agent mistakes the compiler
of another language would have caught, more than once.
