# 0023. Language, runtime and library bundle for the long term

Date: 2026-10-16
Status: under review

## Question

ADR 0020 picked TypeScript on Node 24, discord.js, `better-sqlite3`, npm and `node:test` for
everything Broodforge itself runs, weighting time to first version above every other cost
(proposal 0020, Question; the critic contested the mechanism, not the bundle). The owner's
principle of 2026-10-16 (`../00-vision.md`, Principles) reverses that weighting: scalability and
maintainability first, three growth axes (sessions and users per instance; codebase longevity;
engines and providers), not many instances per deployment, and a choice judged on mechanical
properties because agents know every language. This proposal re-decides the bundle under that
weighting over the full field the owner asked for, before `src/` exists. Settled and out of
scope: the engine mechanism (the Claude Code CLI per turn over stream-json with the `PreToolUse`
hook, ADR 0020 mechanism B, held by the PC spike), a raw-API primary engine, a hosted service.
Open: the language of the bot process, the debug CLI, the hook binary and the tests. The outcome
is an ADR amending ADR 0020 (Language bundle), ADR 0021 (the binding) and the 0022 companion (the
runtime bind and the entry wrapper), then a rewrite of `../08-implementation-plan.md`; the
spike's stream-json fixtures carry over, the `spike/` code is throwaway.

## Constraints that matter here

- Both hosts from one code base (N-01, N-04); every dependency GPL-compatible, licence named
  (N-24); every feature driven from one binary by a Haiku agent (R-50, N-05 to N-11).
- Spike facts the language must serve: the hook is a fresh process per tool call under a
  thirty-second `timeout`, and a hook that overruns is bypassed by the engine (C39,
  `../03-hook-contract.md`), so its startup is paid on every call and its latency under load is a
  reach property, not only a cost. The Windows interrupt is a process-tree kill (`taskkill /T`, or
  a job object ADR 0022 defers for lack of one); Linux is SIGINT through `sudo kill`. The droplet
  sandbox binds the runtime (`<node>`) and `engine-entry` is a `sh` script because Node cannot
  `exec` (0022 companion). `better-sqlite3` needs a native rebuild per Node major and the PC runs
  Node 22 against a pin of 24; a bare `npm install` destroyed the PC's global npm folder
  (`../../findings/2026-10-14-npmrc-location-global-makes-a-plain-npm-install-destructive.md`);
  discord.js pulls a large transitive tree.
- Scale under the per-turn model: an idle session owns no process, so hundreds of concurrent
  sessions means hundreds of concurrent turns, each a `claude` process of a few hundred MB that
  no bot language changes, each spawning the hook per tool call (tens of spawns a second). Discord
  is one gateway and one REST budget per bot, language-neutral; at that scale R-21's per-session
  coalescer needs a global budget (03's follow-up, not this proposal's).
- What the bundle must express: the `Engine` interface and `EngineEvent` sum type of
  `../03-architecture.md` with a second and third engine behind them (R-42, R-43); SQLite in WAL
  mode with the audit triggers (ADR 0021); AES-256-GCM (ADR 0022); one GitHub HTTP call; a unix
  socket or named pipe; the command tree of `../05-debug-interface.md`.

## Criteria and weights

Scored 1 to 5 per option, evidence in each option's bullets; weights (W) are this proposal's
reading of the owner's axes (3 where the owner named the property, 1 where it is second-order;
1 to 4, 10 and 11 are longevity, 5 to 8 sessions, 9 both, 12 engines, 13 all three). The critic
may contest them; the Recommendation says which shift flips the result. Columns A to G are the
options below; the last row is the weighted total out of 140.

| # | Criterion | W | A | B | C | D | E | F | G |
|---|---|---|---|---|---|---|---|---|---|
| 1 | Type strength: what the compiler catches of an agent's mistakes | 3 | 3 | 3 | 5 | 4 | 4 | 1 | 2 |
| 2 | Compiler and test feedback speed and quality in an agent loop | 2 | 4 | 5 | 2 | 3 | 2 | 4 | 4 |
| 3 | Dependency surface and supply-chain exposure (Discord, SQLite, process, HTTP) | 3 | 2 | 4 | 3 | 4 | 3 | 3 | 3 |
| 4 | Toolchain stability over five years; pin friction on both hosts | 3 | 2 | 5 | 4 | 4 | 2 | 3 | 3 |
| 5 | Process control on both hosts (tree kill, job objects, signals, pipes, exec) | 2 | 2 | 5 | 5 | 4 | 3 | 3 | 1 |
| 6 | Concurrency model for hundreds of turns each owning a subprocess and a stream | 2 | 3 | 5 | 4 | 4 | 4 | 3 | 5 |
| 7 | Hook binary startup time and transient footprint | 3 | 2 | 5 | 5 | 4 | 1 | 2 | 1 |
| 8 | Memory and footprint of the bot on a small droplet | 1 | 3 | 4 | 5 | 3 | 2 | 3 | 3 |
| 9 | Single-binary deployment on both hosts | 2 | 2 | 5 | 5 | 4 | 2 | 1 | 2 |
| 10 | Token cost of reading and editing the code, compile-error loops included | 2 | 4 | 4 | 2 | 3 | 3 | 4 | 3 |
| 11 | Maturity and licence of the Discord and SQLite libraries | 2 | 5 | 3 | 4 | 3 | 4 | 4 | 2 |
| 12 | How cleanly an engine interface and a second engine plug in | 2 | 4 | 3 | 5 | 4 | 4 | 3 | 3 |
| 13 | The debug CLI as the same binary | 1 | 4 | 5 | 5 | 4 | 3 | 4 | 2 |
| | Weighted total | 28 | 82 | 120 | 115 | 105 | 79 | 78 | 72 |

## Options

Every option assumes mechanism B, one bot process per instance, the hook as a subcommand of the
one binary. Library facts are unverified unless the 0020 review verified them; timings are C40's.

### A. TypeScript on Node, in its best form

Node 24 LTS pinned; `node:sqlite` (built in) where stable, else `better-sqlite3` (MIT); Oceanic
(MIT) over discord.js 14 (Apache-2.0) to shrink the tree; the hook bundled with `esbuild`;
`util.parseArgs`; `node:test`.

- 1 Types, 3: structural and gradual; `strict` with `noUncheckedIndexedAccess` catches nulls and
  index errors; discriminated unions give the event sum type, exhaustive through a `never` check;
  `any`, `as`, untyped `JSON.parse` and a dropped `await` are escapes closed only by a lint dependency.
- 2 Feedback, 4: `tsc --noEmit` and `node:test` in seconds, no link step.
- 3 Deps, 2: discord.js is tens of packages and `better-sqlite3` a native addon with prebuilds per
  ABI; npm has the worst supply-chain record of the registries here (the 2025 worm and maintainer
  account compromises), and the PC incident shows the environment itself is fragile.
- 4 Toolchain, 2: a Node major every six months, LTS thirty months; native addons rebuild per
  major; TypeScript majors change checking; the pin mismatch is on the PC today.
- 5 Process, 2: `child_process` has no job object and no `exec`; the Windows tree kill is a WMI
  walk plus `taskkill`. 6 Concurrency, 3: one event loop; hundreds of pipes are fine; any
  synchronous work (the outbound mask over a long transcript) stalls every session.
- 7 Hook, 2: Node boots in tens of ms; a bundled hook is of the order of 50 to 120 ms and 40 MB,
  worse under memory pressure, and the failure mode is the C39 bypass. 8 Footprint, 3: 70 to
  100 MB idle. 9 Single binary, 2: SEA is experimental and carries no native addon; a Node
  install and the `<node>` bind stay on both hosts.
- 10 Tokens, 4: compact; the largest example corpus. 11 Libraries, 5: discord.js 14 and Oceanic
  verified in the 0020 review (C17, C18); `node:sqlite` on Node 24 to verify (a release
  candidate in the 0021 review). 12 Engine, 4: interface plus union. 13 CLI, 4: the same entry
  point through a Node shim.

### B. Go

Current stable Go, pinned by the `go` and `toolchain` directives in `go.mod` (the toolchain
fetches itself on both hosts); disgo (Apache-2.0), arikawa (ISC) the alternative, discordgo
rejected (stale, 0020 review C20); `modernc.org/sqlite` (BSD-3-Clause, cgo-free) through
`database/sql`, or `zombiezen.com/go/sqlite` (ISC) over it where the authorizer is wanted;
`os/exec` plus `golang.org/x/sys/windows` (BSD-3-Clause) for job objects; stdlib `flag`, `net/http`,
`encoding/json`, `crypto/aes` with GCM, `testing`; `go vet`, `staticcheck` (MIT) and `exhaustive`
(BSD-2-Clause) in the test command; `CGO_ENABLED=0` everywhere.

- 1 Types, 3: nominal; no sum types and no non-nil reference; errors are values an agent can
  discard (staticcheck flags it); the event type is an interface with a type switch, exhaustive
  by lint, not compiler; JSON decodes into tagged structs with unknown fields refused on request.
- 2 Feedback, 5: build and test in seconds, vet instant, `go test -race` finds shared-state bugs.
- 3 Deps, 4: the standard library covers HTTP, JSON, crypto, process, sockets and tests; disgo
  plus modernc are of the order of fifteen modules; checksum database, proxy, no install scripts;
  disgo's websocket transport has a revived-project history (C52).
- 4 Toolchain, 5: the Go 1 compatibility promise since 2012; one tool, no build file, no C
  compiler; cross-compiles the droplet binary from the PC.
- 5 Process, 5: `SysProcAttr` for creation flags and process groups, `Pdeathsig`, `syscall.Exec`
  for the entry wrapper, job objects through x/sys (the assignment race is C43). 6 Concurrency,
  5: a goroutine per turn blocks on its pipe, channels into the session; GC pauses
  sub-millisecond; `-race` in tests.
- 7 Hook, 5: a static binary starting in low single-digit ms with a few MB resident. 8 Footprint,
  4: 15 to 30 MB. 9 Single binary, 5: static on both hosts; the `<node>` bind and the `sh`
  wrapper go away.
- 10 Tokens, 4: verbose error handling but shallow; a diff reads top to bottom; a refactor is
  mechanical and the compiler names every call site. 11 Libraries, 3: disgo is young (2021) with
  a small maintainer set and a cadence to verify (C41); modernc tracks SQLite releases (C42).
- 12 Engine, 3: implicit interfaces make a second engine a package with the methods; the event
  union leans on the lint. 13 CLI, 5: stdlib `flag`, one binary.

### C. Rust

Stable Rust pinned by `rust-toolchain.toml`; serenity (ISC), twilight (ISC) the lower-level
alternative; rusqlite (MIT, `bundled` compiles SQLite in, needs a C compiler at build); tokio
(MIT); clap and `aes-gcm` (MIT or Apache-2.0); `windows-sys` for job objects; a workspace with
the hook in its own crate so its build stays small.

- 1 Types, 5: enums with exhaustive `match`, no null, `Result` must be used, ownership forbids
  data races at compile time; a refactor that compiles mostly works.
- 2 Feedback, 2: `cargo check` incremental in seconds but a test run links; a clean build of tokio
  plus serenity takes minutes on either host.
- 3 Deps, 3: serenity's transitive tree is of the order of a hundred crates, though from the
  standard async stack; cargo-vet and cargo-deny exist; the crates.io record is better than npm's.
- 4 Toolchain, 4: editions and six-week stables with strong compatibility; MSRV churn in crates;
  Windows needs the MSVC linker (a multi-GB Build Tools install) or the gnu target.
- 5 Process, 5: `CommandExt` (creation flags, process group, `pre_exec`, `exec`), `tokio::process`,
  `windows-sys` job objects. 6 Concurrency, 4: tokio tasks; `Send` bounds and `Arc<Mutex>` state
  are where agents stumble.
- 7 Hook, 5: about 1 ms and 2 MB. 8 Footprint, 5: the smallest. 9 Single binary, 5: static with
  the musl target, or glibc on the droplet.
- 10 Tokens, 2: the most lines per feature; borrow-checker iterations; async trait plumbing.
  11 Libraries, 4: serenity since 2016 (its `next` branch long unreleased, C46), twilight
  lower-level, rusqlite mature.
- 12 Engine, 5: trait plus enum, the strongest contract; `dyn` dispatch of async traits needs
  care. 13 CLI, 5: clap, one binary.

### D. C# on .NET

.NET 10 LTS pinned by `global.json`; NetCord (MIT, built for Native AOT), Discord.Net (MIT,
Newtonsoft-based, not AOT) the alternative; `Microsoft.Data.Sqlite` (MIT) with SQLitePCLRaw
bundled natives (Apache-2.0); the BCL for HTTP, source-generated JSON, AES-GCM and `Process`;
`System.CommandLine`; Native AOT publish for both hosts.

- 1 Types, 4: nullable reference types promoted to errors, records, pattern matching with
  exhaustiveness warnings; exceptions untyped. 2 Feedback, 3: incremental builds in seconds with
  the build server, tests in ten seconds; AOT publish slow but release-only.
- 3 Deps, 4: the BCL covers most; NetCord small; first-party SQLite. 4 Toolchain, 4: annual
  majors, LTS three years, documented breaks; a large SDK install.
- 5 Process, 4: `Process.Kill(entireProcessTree: true)` on both hosts; no managed SIGINT send or
  job object (one P/Invoke each). 6 Concurrency, 4: async over a thread pool; mature.
- 7 Hook, 4: AOT about 10 ms; JIT about 100 ms; the whole bot must stay AOT-clean or the hook is
  a second project. 8 Footprint, 3: 30 to 50 MB AOT. 9 Single binary, 4: AOT single file, or
  self-contained at about 70 MB.
- 10 Tokens, 3: attribute and project-file noise. 11 Libraries, 3: NetCord young (2023) and
  single-vendor; Discord.Net mature but AOT-hostile (C47). 12 Engine, 4: interfaces and records.
  13 CLI, 4: `System.CommandLine`, one binary.

### E. Kotlin on the JVM

JDK 21 or 25 LTS; Kotlin with coroutines; JDA (Apache-2.0) or Kord (MIT); `sqlite-jdbc`
(Apache-2.0, bundled natives); Gradle; clikt; GraalVM native-image for the hook if at all.

- 1 Types, 4: null-safe, sealed classes with exhaustive `when`. 2 Feedback, 2: Gradle daemon,
  slow cold builds. 3 Deps, 3: JDA about ten, Kord the ktor stack. 4 Toolchain, 2: JDK LTS is
  fine; Gradle and Kotlin version churn is the known friction; two pins plus a build tool.
- 5 Process, 3: `ProcessHandle` tree walk and `destroy` (SIGTERM, which leaves the engine's turn
  unfinished per C16); no SIGINT or job object without JNA. 6 Concurrency, 4: coroutines.
- 7 Hook, 1: JVM boot 100 to 300 ms and over 100 MB; native-image fixes it at a build and
  reflection-config cost. 8 Footprint, 2: 150 to 300 MB. 9 Single binary, 2: a jlink directory
  or native-image. 10 Tokens, 3. 11 Libraries, 4: JDA since 2015, Kord, `sqlite-jdbc` (C49).
  12 Engine, 4: sealed classes. 13 CLI, 3: the same jar at 100 ms or more per invocation.

### F. Python

Python 3.13; `uv`; discord.py (MIT) or hikari (MIT); stdlib `sqlite3`; `asyncio` subprocesses;
pyright strict in the test command; pytest.

- 1 Types, 1: optional; pyright strict catches much but nothing stops an untyped path at runtime.
  2 Feedback, 4: no compile, pyright and pytest in seconds. 3 Deps, 3: discord.py plus aiohttp
  about ten, stdlib SQLite, PyPI record poor. 4 Toolchain, 3: annual majors, five-year support,
  `uv` pins, no native build here.
- 5 Process, 3: `asyncio` subprocess, `CREATE_NEW_PROCESS_GROUP`, `ctypes` for job objects,
  `os.execv`. 6 Concurrency, 3: single-threaded `asyncio`, slower JSON. 7 Hook, 2: 25 to 50 ms
  bare, about 100 ms with imports, 20 to 30 MB. 8 Footprint, 3: about 50 MB. 9 Single binary,
  1: PyInstaller is a workaround. 10 Tokens, 4: the most compact. 11 Libraries, 4: discord.py
  and hikari verified in the 0020 review (C19). 12 Engine, 3: Protocols, no exhaustiveness.
  13 CLI, 4: argparse, same entry.

### G. Elixir on the BEAM

OTP 27 or 28, Elixir 1.18 or 1.19; Nostrum (MIT); exqlite (Apache-2.0) with ecto_sqlite3; a mix
release; the CLI as a release command.

- 1 Types, 2: dynamic; dialyzer; set-theoretic types arriving incrementally. 2 Feedback, 4: `mix
  compile` and ExUnit fast. 3 Deps, 3: Nostrum ten to fifteen packages, a NIF for SQLite (C50).
  4 Toolchain, 3: OTP yearly majors; Windows is its least exercised host; NIF rebuilds.
- 5 Process, 1: ports cannot signal or kill a tree; the OS pid comes from port info and the kill
  is a shell-out or erlexec; weaker still on Windows. 6 Concurrency, 5: supervised processes per
  session; crash isolation is the runtime's. 7 Hook, 1: BEAM boot 200 to 400 ms; disqualifying
  for a per-call hook unless the hook is another language, which fails criterion 13.
- 8 Footprint, 3: about 50 MB. 9 Single binary, 2: a release directory with ERTS. 10 Tokens, 3:
  a second idiom to keep consistent. 11 Libraries, 2: Nostrum has a small community. 12 Engine,
  3: behaviours, dynamic. 13 CLI, 2: a BEAM per invocation.

## Recommendation

Option B, Go, as the bundle above. Confidence moderate: B and C are five points apart on a scale
the critic may re-weight, and the recommendation turns on a judgement stated here so it can be
argued: this program is process and stream plumbing (spawn, pipe, parse, coalesce, post, record),
not an algorithmic core, so the criteria that compound on every one of years of agent-written
changes (feedback speed, token cost per change, a toolchain that never moves under the code)
outweigh Rust's stronger compiler, and Go's gap on criterion 1 is narrowed inside the same loop by
vet, staticcheck, exhaustive and the race detector. What Go buys on the owner's axes: a hook that
starts in milliseconds under any load, closing the C39 bypass as a load phenomenon; a Windows job
object from the standard extension library; one static binary per host, so the droplet sandbox
loses the runtime bind and the `sh` entry wrapper (the binary can `exec` the engine itself) and
the PC loses Node, npm and the native rebuild (the engine can use its native installer); a
dependency list short enough to read in a review. Against TypeScript the owner's four longevity
words (type strength, small dependency tree, stable toolchain, cheap refactors) go two to Go, one
narrowly to TypeScript, one tie; the incumbent's remaining advantage, the example corpus, is
familiarity, which the owner excluded.

First implementation step, after the ADR: the coder writes `go.mod` with the pinned toolchain,
`cmd/broodforge` with the subcommand dispatcher and the `hook pre-tool-use` subcommand against
`../03-hook-contract.md` and the shipped case file, and measures C40 on both hosts; then the
engine adapter and the fake from the spike's fixtures, in the order of
`../08-implementation-plan.md` once it is rewritten for Go. The ADR amends ADR 0020 (Language
bundle), ADR 0021 (binding), the 0022 companion (`<node>` bind, `engine-entry` as the binary with
`exec`, `PATH`) and `../03-architecture.md` (the `Engine` snippet, `paths.node`).

## What would change this recommendation

- The owner weights type strength above per-change cost (criterion 1 to 5, criteria 2 and 10 to 1):
  the table flips to C, Rust, with serenity, a workspace and the hook as its own crate.
- The critic finds disgo short on cadence, maintainers, components, thread creation, channel
  creation with permission overwrites, or a fresh member fetch (C41): arikawa; failing that, C.
- `modernc.org/sqlite` fails on windows/amd64 or in WAL under concurrent turns (C42): zombiezen or
  ncruces over the same engine; failing those, C with rusqlite.
- NetCord is verified mature and AOT end to end (C47): D becomes the second alternative; no change.
- The owner keeps TypeScript for a stated reason: A in its best form, with C40 measured anyway.
- Hundreds of concurrent turns prove unreachable for a reason outside the bot (engine memory, the
  REST budget): the choice rests on longevity alone, which still favours B over A and C as above.

## Claims to verify

No web tool was available to the architect; every library claim above is unverified unless the
0020 review verified it. For the critic, and the spike where a measurement is the evidence:

- C40. Hook startup on both hosts, median of fifty spawns with peak resident size: a bundled Node
  hook on Node 22 and 24, a Go static binary, a Rust static binary; then with engine turns running.
- C41. disgo: licence, latest release and date, maintainers, gateway resume, REST rate limiter,
  components, thread creation, channel creation with overwrites, uncached member fetch; arikawa too.
- C42. `modernc.org/sqlite`: windows/amd64 and linux/amd64 without cgo, WAL and `busy_timeout`
  through the DSN, an authorizer reachable (modernc, zombiezen, ncruces), the SQLite version tracked.
- C43. `golang.org/x/sys/windows`: `CreateJobObject`, `SetInformationJobObject` with
  kill-on-close, `AssignProcessToJobObject`; whether `os/exec` lets the child be assigned before
  it can spawn, else the window after `Start` and the `taskkill` walk as the fallback.
- C44. `go test -race` on windows/amd64: whether a C toolchain is needed. C45. The current stable
  Go version and the `toolchain` directive's download behaviour behind a pinned `go.mod` on both
  hosts; `encoding/json` unknown-field refusal.
- C46. serenity and twilight status and licences; rusqlite `bundled` build needs on Windows.
  C47. NetCord: licence, cadence, Native AOT end to end; Discord.Net under AOT;
  `System.CommandLine` stability; .NET 10 LTS dates.
- C48. Node SEA and native addons; `node:sqlite` on Node 24; Oceanic's coverage of components,
  threads and overwrites. C49. JDA, Kord, `sqlite-jdbc` status; GraalVM Community licence.
  C50. Nostrum status; exqlite precompiled NIFs on Windows; port signal limits. C51. discord.py
  and hikari status; Python hook startup.
- C52. disgo's websocket dependency: which library, its maintenance history (the gorilla archive
  and revival), whether disgo pins it. C53. Discord's global REST limit per bot and the
  per-channel edit limit, for 03's global budget follow-up (language-neutral, in Constraints).

## Critic's objections and responses

(none yet)
