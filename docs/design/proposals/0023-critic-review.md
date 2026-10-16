# Critic review of proposal 0023

Date: 2026-10-16. Reviewer: critic (Fable 5.1). Sources: pkg.go.dev, go.dev release and toolchain
pages, the GitHub API and raw files for disgo, arikawa, gorilla/websocket, serenity, NetCord,
staticcheck, exhaustive and x/sys, crates.io, nuget.org, nodejs.org docs, docs.discord.com, the
rustup book; and measurements on the owner's PC (Go 1.27.0, Node 22.23.2, no cargo, no C compiler,
no .NET SDK). Scope per the brief: the engine mechanism, a raw-API primary and a hosted service are
settled; Python, the JVM and the BEAM stay below the top three unless a verified fact moved one.
None did.

## Verdict in one paragraph

The recommendation survives verification, and no claim it rests on is refuted; but two of its
three scoring arguments are weaker than the table says, one bundle claim is false on the PC today,
and one hidden coupling closes a path ADR 0020 left open. Hook startup is a real but small Go
advantage (measured: 9 ms against Node's 48 ms, both three orders below the 30 s timeout), so
criterion 7 cannot carry weight 3; the sessions axis is decided by the engine's own memory, not
the bot's language, so criterion 6 separates nothing. `go test -race` needs cgo and a mingw C
compiler on Windows, which the PC does not have, so the "narrowed by the race detector" sentence
is not true in the loop the agents run here. Go still leads TypeScript on every reweighting I
tried, because the owner's own longevity words (small dependency tree, stable toolchain) are
verified facts that favour it; Go against Rust is four points either way and turns on the one
weighting the owner has not given. The library risk the proposal under-states is that the Go
Discord ecosystem is one person deep.

## Claims verified

| Claim | Finding | Source |
|---|---|---|
| C40 | PC, idle, fifty spawns, stdin JSON in, decision JSON out, `BROODFORGE_MODE=prompt`: Go static binary (4.1 MB, `CGO_ENABLED=0`) median 8.5 to 9.0 ms, p90 10.5 ms, working set 9.4 MB; Node 22 plain script median 48.3 ms, p90 51.5 ms, working set 29 MB; an empty Node script is the same 48 ms, so bundling buys nothing, the cost is the runtime boot. Rust and Node 24 unmeasurable here; under engine load unmeasured. The proposal's "low single-digit ms" for Go is Linux-shaped; Windows process creation sets the floor near 8 ms. | scratchpad driver, `Get-Process` |
| C41 | disgo: Apache-2.0; latest release v0.19.6 (2026-06-07), v0.19.2 to .6 all by one author; commits 1387 by topi314, next 117, then 77; `go 1.24.0`; README lists gateway, sharding, "Full Rest API coverage", rate limiting, components and modals, threads; overwrites and uncached member fetch not shown on the page (REST coverage claim covers them, unverified individually). arikawa: ISC, 600 stars, pushed 2026-05-18, not archived. | github.com/disgoorg/disgo, api.github.com |
| C42 | modernc.org/sqlite: BSD-3-Clause, v1.60.1 (2026-09-29, not latest), SQLite 3.53.4, no cgo, twenty GOOS/GOARCH pairs including windows/amd64 and linux/amd64; `_journal_mode=WAL`, `_busy_timeout` and `_pragma` on the DSN; CPU-bound queries 1.3x to 2.0x slower than C SQLite; PRAGMAs persist per pooled connection, `SetMaxOpenConns` advised, `StrictPragmas(true)` advised; no authorizer visible. zombiezen.com/go/sqlite: ISC, over modernc, `SetAuthorizer`, `OpenWAL`, `SetBusyTimeout`. ncruces/go-sqlite3: MIT, wasm, `Conn.SetAuthorizer`, windows/amd64 tested, pre-v1. | pkg.go.dev |
| C43 | x/sys/windows v0.49.0 (BSD-3-Clause) exports `CreateJobObject`, `SetInformationJobObject`, `AssignProcessToJobObject`, `JOBOBJECT_EXTENDED_LIMIT_INFORMATION`, `JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE` (0x2000), `JobObjectExtendedLimitInformation` (9). `os/exec` does not expose the thread handle (golang/go#32404, declined), so suspend-then-assign needs a direct `CreateProcess`; the window after `Start` is microseconds. | pkg.go.dev, raw types_windows.go, go.dev/issue/32404 |
| C44 | Refuted as bundled. "The race detector requires cgo to be enabled, and on non-Darwin systems requires an installed C compiler"; Windows needs mingw-w64 runtime 8 or later. On the PC: `go: -race requires cgo; enable cgo by setting CGO_ENABLED=1`; no gcc anywhere. | go.dev/doc/articles/race_detector, local run |
| C45 | Go 1.27.2 current (2026-10-08); 1.27.0 2026-08-19; PC has 1.27.0, `GOTOOLCHAIN=auto`. The `toolchain` line wins over `go`; a newer pin is downloaded as module `golang.org/toolchain` through `GOPROXY`, verified by the sumdb, into the user module cache; fails under `GOSUMDB=off`; `GOTOOLCHAIN=local` refuses instead. `Decoder.DisallowUnknownFields` since Go 1.10. Go 1.27 backs `encoding/json` with v2 (rejects invalid UTF-8 and duplicate names; `GOEXPERIMENT=nojsonv2` opt-out "expected to be removed"), adds generic methods, and `go test` runs the `stdversion` vet check under `go 1.27`. | go.dev/doc/toolchain, go.dev/doc/devel/release, go.dev/doc/go1.27, pkg.go.dev |
| C46 | serenity 0.12.5 (2025-12-20), ISC, MSRV 1.74 on `current`; `next` unreleased (page says only that it tracks latest stable). twilight-gateway 0.17.1 (2025-12-13), ISC, MSRV 1.89, repository pushed 2026-04-04. rusqlite 0.40.2 (2026-08-08), MIT, `bundled` feature. Windows MSVC target needs Build Tools with the C++ workload and Windows SDK; the gnu target needs mingw-w64 (the same install C44 needs). | github, crates.io, rustup book |
| C47 | NetCord: MIT, 1.0.0-beta.28 (2026-10-01), "requires .NET 10 or higher"; "Native AOT Support" is a README bullet with no end-to-end evidence found. System.CommandLine 2.0.13 stable (2026-10-13), 3.0 rc in flight. .NET 10 LTS 2025-11-11 to 2028-11-14. Discord.Net under AOT not checked (D is not promoted, objection 11). | github, netcord.dev, nuget.org, dotnet.microsoft.com |
| C48 | `node:sqlite` is Stability 1.2 release candidate (rc since 25.7.0; `setAuthorizer` since 24.10.0); SEA is Stability 1.1 active development, Windows on CI, native addons only by writing the asset to a temp file and `process.dlopen`. Oceanic v1.15.0 (2026-08-13); discord.js 14.27.0 (2026-10-14, 0020 review C17). Oceanic component and overwrite coverage not individually re-verified. | nodejs.org docs, github |
| C49 to C51 | JDA v6.7.0 (2026-09-20); Kord 0.18.1 (2026-03-14); Nostrum MIT, pushed 2026-04-05; discord.py and hikari as 0020 C19. Nothing moves E, F or G. Python hook startup unmeasured (3.10 on the PC, not 3.13). | api.github.com |
| C52 | disgo requires `github.com/gorilla/websocket v1.5.3`: BSD-2-Clause, last release 2024-06-14, last push 2025-03-19, 84 open issues, not archived (the 2022 archive was reverted; activity since is thin). Not pinned by disgo beyond the usual `go.mod` requirement. | raw go.mod, api.github.com |
| C53 | "All bots can make up to 50 requests per second"; per-route limits key on the top-level resource (two channel ids are limited independently); 10,000 invalid requests per 10 minutes blocks the IP; raised global limits by request to support. No per-channel edit limit on the page; the 5 per 5 s figure is community lore, not primary. | docs.discord.com/developers/topics/rate-limits |
| Licences | staticcheck MIT (pushed 2026-10-12); exhaustive BSD-2-Clause (GitHub shows NOASSERTION because the file carries a second BSD-3 block for Go Authors code); arikawa ISC; all named libraries GPL-compatible. | raw LICENSE files |

Not verified: npm's 2025 worm and account compromises (not re-checked; the PC `.npmrc` incident is
verified by its finding); Rust hook timings; Windows `windows-sys` job object bindings (standard,
low risk); whether modernc exposes an authorizer at all (its page is truncated; zombiezen does).

## Objections

1. **Serious. C44 refuted as bundled: `go test -race` is not available on the PC.** The bundle
   says `CGO_ENABLED=0` everywhere and no C compiler, and criteria 2 and 6 and the Recommendation
   lean on the race detector to narrow the gap to Rust. On windows/amd64 `-race` needs cgo and a
   mingw-w64 gcc; the PC has none; the exact refusal is quoted above. Agents run the loop on the
   PC. Resolves by one of: an operator entry installing mingw-w64 on the PC (MSYS2 UCRT64) and
   `gcc` on the droplet, the test command running `-race` with `CGO_ENABLED=1` for that run only;
   or `-race` on Linux only, stated; and criterion 4 reworded to "no C compiler for the build".

2. **Serious. Criterion 7 (hook startup, weight 3) is over-weighted and its load argument is
   unevidenced.** Measured: 9 ms against 48 ms, a 40 ms difference against a 30,000 ms timeout and
   a 3,000 ms engine start per turn. C39's bypass came from a deliberate `sleep`, not load; no run
   has shown a Node hook near the timeout under any load. At weight 1 the criterion gives B three
   points over A instead of nine; B still leads. Resolves by: weight 1, the Recommendation drops
   "closing the C39 bypass as a load phenomenon" or the spike measures the Node hook while N engine
   turns run on the PC and the sentence cites the number.

3. **Serious. Criterion 6 (hundreds of concurrent turns) cannot separate the options.** Hundreds
   of concurrent `claude` processes at a few hundred MB each is tens of GB on one host; the bot's
   footprint and scheduler are a rounding error, and Node and Go both hold hundreds of pipes. The
   proposal says this in Constraints and then awards B a four-point lead on it. Resolves by:
   weight 1 or A scored 4; the proposal states that the sessions axis decides nothing between A, B
   and C and the choice rests on longevity and engines.

4. **Serious. The Go Discord library is one person deep and sits on a dormant websocket
   library.** disgo: one author for every 2026 release, 1387 of about 1700 commits, pre-1.0,
   requiring gorilla/websocket (last release 2024, last push 2025-03). arikawa: one maintainer,
   last push May 2026. For a five-year horizon the likeliest failure is the library stopping; the
   table scores this 3 on criterion 11 but scores dependency surface 4 and toolchain stability 5
   as if the library were part of the standard library. The Rust side is no better (serenity
   0.12.5 and twilight 0.17.1, both December 2025, `next` unreleased); only TypeScript has a
   library with a decade of continuity. Resolves by: the ADR names the exit: Broodforge's Discord
   surface is narrow (gateway events, components, a dozen REST calls) behind 03's `discord`
   component, so the fallback is arikawa or an in-house gateway and REST client written against
   Discord's documented API; disgo pinned by version with its websocket dependency listed; the
   proposal says so under criterion 3.

5. **Serious. Hidden coupling: Go closes ADR 0020's deferred Agent SDK path.** The SDK ships for
   TypeScript and Python only (0020 review C1); ADR 0020 keeps it as a second implementation
   behind two triggers, and mid-turn message injection waits on it. In Go, Rust or C# it needs a
   Node sidecar, a second runtime on both hosts. A raw-API second engine is unaffected. Resolves
   by: the ADR amends 0020's "Deferred, with triggers" to say the SDK route is withdrawn or needs a
   sidecar, and the owner hears that the engines axis loses one shape.

6. **Serious. The cost of switching is not in the table or the Recommendation.** Thirty-three
   Node or TypeScript references across 03, 05, 07 and 08; `08-implementation-plan.md` (248 lines)
   rewritten with new file names and proofs; 03's `Engine` snippet, `paths.node` and the
   components table; the 0022 companion's `<node>` bind and `engine-entry`; ADR 0020, 0021 and
   0022 amendments; `spike/run.ts` (`parseLine`, `usageDelta`, `interruptedRow`, which P3 seeds
   from) re-done in Go against the same fixtures. Roughly two architect passes, one tester pass
   and one extra coder pass. Under the owner's principle this is secondary; it is still the one
   cost on which A scores zero and the owner has not seen the figure. Resolves by: a short "cost
   of change" paragraph naming the files and passes; the owner accepts it with the ADR.

7. **Minor. Unknown-field refusal must not touch the engine stream.** 03 reads unknown kinds as
   `unknown` and the spike found seven new kinds in one engine version; `DisallowUnknownFields` on
   stream structs would turn an engine pin move into a parse failure. Go 1.27's v2-backed
   `encoding/json` also rejects invalid UTF-8 and duplicate names, so a toolchain major can change
   what the parser accepts. Resolves by: refusal on configuration files only; fixtures replayed on
   every Go major as part of the test command.

8. **Minor. The SQLite binding needs one more decision.** modernc through `database/sql` brings
   pool semantics ADR 0021 did not plan for (PRAGMAs per connection, `SetMaxOpenConns`,
   `StrictPragmas`), a 1.3x to 2.0x CPU cost (irrelevant at this scale) and no visible
   authorizer; zombiezen over the same engine has `SetAuthorizer`, `OpenWAL` and `SetBusyTimeout`.
   Resolves by: the ADR names zombiezen as the binding (ISC, authorizer for ADR 0021's audit
   guard) or modernc with a single-connection writer, not "or".

9. **Minor. Job objects: accept the post-`Start` window and take the free win.** `os/exec` cannot
   suspend-then-assign (C43); the window is microseconds and the engine spawns nothing in its first
   hundreds of milliseconds. A job on the bot process itself with kill-on-close makes every engine
   die with a crashed bot, which the 0022 companion lists as a Windows gap. Resolves by: the ADR
   says both; `taskkill /T` stays the fallback.

10. **Minor. C45's toolchain download has a refusal with no documented lift.** A `toolchain`
    pin newer than the installed Go downloads through the proxy; offline, or with `GOSUMDB=off`,
    the build fails naming the version. Resolves by: the host reference says the lift (install that
    Go, or `GOTOOLCHAIN=local` with the pin lowered) and the droplet build has network.

11. **Minor. C47: the D-promotion trigger is not met.** NetCord is a beta with a README AOT
    claim and no end-to-end evidence; the PC has no .NET SDK. D stays third; no change.

12. **Minor. C46: the Rust path on the PC is one operator step heavier than stated.** Neither
    cargo nor any C compiler is installed; MSVC Build Tools or mingw-w64 is needed for `rusqlite`
    `bundled` and the linker. A clean serenity plus tokio build on a small droplet is minutes; the
    proposal should name cross-compilation from the PC (musl) as the droplet build if C is chosen.

13. **Minor. C48 holds; option A's "where stable" is not yet met.** `node:sqlite` is a release
    candidate with an authorizer since 24.10, so `better-sqlite3` stays for A today; SEA carries
    addons only by temp-file extraction. A's scores stand.

14. **Minor. Criterion 1 under-scores TypeScript against Go on the one type that matters.** A
    discriminated union with a `never` check is compiler-exhaustive; Go's type switch is
    lint-exhaustive; Go's nil and discarded errors are escapes as `any` and `as` are. Score A 4
    and B 3 on the event type, or both 3 with the reason stated; the order does not change.

15. **Minor. C53: the global budget follow-up has its number.** 50 requests per second per bot,
    per-route limits keyed on channel id, no documented per-channel edit limit. Hand to 03.

## Refusals, denials and fail-closed paths the bundle adds, and who lifts them

| Mechanism | Who lifts it, from where, while a session runs | Message says so? |
|---|---|---|
| `go test -race` without cgo | The operator installs a C compiler on the host; agents cannot | Yes: names `CGO_ENABLED=1`, not the compiler (objection 1) |
| `toolchain` download fails (offline, `GOSUMDB=off`) | The operator installs that Go or sets `GOTOOLCHAIN=local`; never mid-session | Names the version, not the lift (objection 10) |
| `exhaustive` or `staticcheck` failing the test command | The agent, in the same change (add the case, handle the error, or a documented `//exhaustive:ignore`) | Yes, names the switch or the call |
| `stdversion` vet check under `go 1.27` | The agent bumps `go.mod` | Yes, names symbol and version |
| `DisallowUnknownFields` on an engine line | A code change only; no runtime lift | Names the field; must not exist (objection 7) |
| Job object kill-on-close | A feature, not a refusal: a crashed bot ends its engines | n/a |

Every refusal above is build- or test-time; none fires inside a session, and the hook's own
contract (03) is unchanged by the language.

## Hidden coupling

- Go decides the Discord library (one maintainer) and its websocket dependency (objection 4),
  the SQLite binding and its pool semantics (8), and the loss of the in-process SDK (5).
- Go makes the Windows job object cheap, which pre-decides the job object ADR 0022 deferred; and
  `syscall.Exec` on Linux removes the `sh` half of `bin/broodforge`, so 05's command classes and
  the 0022 companion change in the same ADR.
- Cross-compilation tempts the plan to ship the droplet binary from the PC; the droplet should
  build its own from the pinned source, which needs network for the toolchain download (10).
- Installing mingw-w64 for `-race` (1) is the same install Rust's gnu target needs (12): one
  operator entry serves both, and the Rust door stays open at low cost.

## The strongest case for Rust, and for TypeScript in its best form

Rust. The owner's longevity axis is years of agent-written changes staying safe. Rust's compiler
rejects what Go's lint layer only reports: non-exhaustive matches, unused `Result`s, null, and
data races across session state through `Send` and `Sync`, which Go finds only when a `-race`
test exercises the path, and on the PC not at all today (1). A refactor that compiles mostly works,
which is worth most when the author is a Haiku agent in 2029 editing code it has never seen. The
process story is Go's equal (`CommandExt`, `tokio::process`, job objects through `windows-sys`),
the hook is smaller still, the binary is static on both hosts. The costs are real and ongoing:
each change pays borrow-checker and async-trait iterations in tokens and minutes, a clean build is
minutes on either host, the PC needs a linker install, and the Discord libraries are as thin as
Go's with a slower cadence. Under the weighting the proposal names (type strength 5, criteria 2
and 10 at 1) the table reads C 121 to B 117: a flip, but a four-point one.

TypeScript. Zero switching cost: every design doc, the plan and the spike parser are written in
it (6). discord.js is the one Discord library in this field with a decade of continuity and a
release this week; `node:sqlite` is a release candidate with an authorizer, so the native addon
is on its way out; Node 24 is LTS for thirty months; a discriminated union is compiler-exhaustive
(14); the hook's 48 ms is nothing against a 30 s timeout (2); and the event loop holds hundreds
of pipes (3). What stands against it is verified, not taste: the npm registry's record and the
PC's `.npmrc` hazard, a native addon rebuilt per Node major with the pin already mismatched, no
job object or `exec` without an addon, the `<node>` bind and the `sh` wrapper on the droplet, and a
dependency tree no reviewer reads. Those are two of the owner's four longevity words, which is
why A loses on every reweighting I tried (82 as scored; at most 83 with 2 and 3 at weight 1 and
criterion 1 corrected), while B and C stay within five points of each other.

## Recommendation

Go stands over TypeScript, with conditions 1, 4, 5, 7 and 8 recorded in the ADR and the cost of
change (6) shown to the owner; confidence high on that pair, because the facts that decide it are
verified and survive the reweighting of 2 and 3. Go over Rust I hold at moderate confidence,
assuming the owner weights type strength and per-change cost equally, which is how the vision
lists them ("type strength ... cheap refactors" in one breath): this program is spawn, pipe, parse,
coalesce, post and record, and for that shape a fast loop with vet, staticcheck, exhaustive and a
working race detector catches what matters at a fraction of Rust's per-change cost. The question
the owner should be asked, in his own terms: when an agent changes the session code in three
years, do you want the compiler to reject more mistakes at the price of every change costing more
tokens and minutes (Rust), or a cheaper, faster loop with a lint layer catching most of them (Go)?
If the answer is the compiler, C with serenity, a workspace, the hook as its own crate, and
mingw-w64 on the PC for both `-race` and the gnu target. Either answer is defensible; TypeScript is
not, under the principle of 2026-10-16, unless the owner wants to count the switching cost as the
deciding number, which the principle says he does not.
