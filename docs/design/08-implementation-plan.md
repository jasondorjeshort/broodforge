# Implementation plan: coder passes for `cmd/` and `internal/`

Date: 2026-10-16. Status: plan for the Go bundle of ADR 0023, written after the PC spike closed
every PC row (ADRs 0020 and 0022, Amendments of 2026-10-16) and `07-spike-plan.md`'s exit criterion
was met for host-neutral code (the droplet rows are open and P14 is held). Each pass is one coder brief (`brief` skill); done is the `debug-interface` skill's
checklist, its Proof run on a clean checkout from the doc alone, and the `05-debug-interface.md`
section it builds matching the code. Every pass reads `CLAUDE.md`, `docs/design/README.md`,
`03-architecture.md` (Components), `05-cli-conventions.md` and ADR 0023 (Decision), then its
Implements line. Sizes: small is one sitting, medium one to two, large needs a mid-pass commit.
"Owner sees" means the output is 04's.

Standing rules. A pass's Files line is its boundary (every name there is created by the pass
[no-check: the whole plan names files not yet written]). One module, one binary `cmd/broodforge`;
one package per component of 03's table under `internal/<component>/`, tests beside the code as
`_test.go`, fixtures and scripts under `tests/` and `fixtures/`. No `if windows` (no `runtime.GOOS`,
no `_windows.go`) outside `spawn`, the worktree removal it owns and `control`. A real engine turn
spends the owner's plan and runs only where a pass says "real turn", on his word. `spike/` stays
until the droplet rows land (P14 deletes it). Every Proof's `broodforge` is `bin/broodforge` (`.exe`
on Windows) from `CGO_ENABLED=0 go build -o bin/ ./cmd/broodforge` at the repository root (the
directory form: `-o bin/broodforge` writes no `.exe` on Windows), or `go run ./cmd/broodforge`
(same usage output; its exit code is not the binary's, so exit codes are proven on the binary);
nothing goes on `PATH`, `go install` is never used. Build before test, always:
`CGO_ENABLED=0 go build ./...`, the check line, then `CGO_ENABLED=1 go test -race ./...` on Linux
and on the PC. Command classes are `05-cli-conventions.md`'s (Surface); write commands are
socket clients that exit `not-running` until P10's `serve`, so before P10 a write command is proven
through P2's harness.

## P0. Toolchain and repository skeleton (small)

Implements: ADR 0023 (Decision); `03-architecture.md` (Components); `.claude/rules/README.md`;
`../reference/README.md`. Files: `go.mod` (`module github.com/jasondorjeshort/broodforge`; `go
1.27`; `toolchain go1.27.2`; `tool` directives for `honnef.co/go/tools/cmd/staticcheck` and
`github.com/nishanths/exhaustive/cmd/exhaustive`; every ADR 0023 dependency required at a pinned
version, `github.com/disgoorg/disgo`, `zombiezen.com/go/sqlite`, `golang.org/x/sys/cpu` (the module has no root package), held by blank
imports in `internal/deps/deps.go` until the owning package imports each, so no later pass fetches
anything) and `go.sum`; `cmd/broodforge/main.go` (usage on stderr, exit 2; dispatch to
`internal/cli`); an empty `doc.go` per component of 03's table under `internal/` (`backup` and `config` are separate packages);
`.claude/rules/go.md` [no-check: written by this pass] (SPDX first line; `gofmt`; vet, staticcheck
and exhaustive clean; no `interface{}` or `any` outside `internal/engine`'s raw stream layer; sum
types as a tagged struct with a `Kind` enum and a pointer payload per kind, never a type switch;
`testing` only; `DisallowUnknownFields` on configuration only, never on the engine stream);
`docs/reference/go-dependencies.md` [no-check: written by this pass] (one line per module, disgo's
gorilla/websocket requirement included, version and licence, N-24) and its README line;
`tests/README.md`; `.gitignore` (`bin/`, `fixtures/**/received-*.json`). `CGO_ENABLED=0` is on the
build line, never in `go.mod`. The coder installs nothing: the PC has Go 1.27.0 and the `toolchain`
line fetches 1.27.2 on first use (ADR 0023 names the refusal). Proof: `CGO_ENABLED=0 go build -o
bin/ ./cmd/broodforge`; `go vet ./... && go tool staticcheck ./... && go tool exhaustive
./...`; `go test ./...` (one placeholder test in `internal/cli`); `bin/broodforge` with no arguments
exits 2 with usage on stderr; `go run ./cmd/broodforge` prints the same usage (its exit code is `go run`'s, not the binary's). Depends on: nothing.

## P1. Fixture import (small; an independent agent reads every file before the commit)

Implements: `07-spike-plan.md` (The code: fixtures kept); `03-architecture.md` (The engine adapter);
`../ideas/2026-10-16-fixtures-need-a-path-rewrite-before-they-move-into-the-repo.md`, deleted in
this pass's commit (the queue rule). Files: `internal/tools/importfixture/main.go` (a `go run` tool,
not a subcommand); `fixtures/engine/<name>/turn-<n>.jsonl` and `fixture.json` for `basic`, `defer`,
`interrupted`, `parallel`, `hook-exit2`, `hook-exit1`, `hook-sleep`, `scrub`, `settings-a`,
`settings-b` from `C:/Users/<pc user>/AppData/Local/Temp/bf-spike/fx/<name>/`;
`fixtures/engine/README.md` (one line per fixture: claim, turns, what it shows). The rewrite works
on whole path tokens: one starting with any form of the PC home or spike root (`C:\Users\<user>`,
`C:/Users/<user>`, JSON-escaped `C:\\Users\\<user>`) becomes `/home/user` or `/home/user/bf-spike`,
separators normalised to `/`; the engine's project slug
`C--Users-<user>-AppData-Local-Temp-bf-spike-tree` (in `memory_paths.auto` and the like) becomes
`-home-user-bf-spike-tree`. Only `turn-<n>.jsonl` is imported; `meta-<n>.json` yields `fixture.json`
(engine version, date, claim, exit code per turn: `interrupted` turns 1 and 2 exit 1). No test
asserts on an original path. Proof: one `go run ./internal/tools/importfixture <src>
fixtures/engine/<name>` per name; `grep -rliE "<pc user>|[A-Za-z]:(/|\\\\)+Users" fixtures/` empty. Before the commit, an agent that did not record the fixtures reads every file with nested JSON decoded, confirms no name, email, hostname, path outside the placeholder or repository content survives, and reports in a few lines what is account-level or personal and cannot be classified by a script; the owner decides on that report, never by reading the files. Depends on:
P0. Must not touch: `spike/`.

## P2. CLI conventions, the control socket, the in-process harness (medium)

Implements: `05-cli-conventions.md` (all); `03-architecture.md` (Debug surface attachment; the `cli`
and `control` rows); `05-debug-interface.md` (Conventions). Files: `internal/cli/` (`flag`-based
parsing, `--root`, `--json`, `--expect`, the one formatter, exit codes, the error line, the command
classes); `internal/control/` (unix socket; named pipe through `x/sys/windows` or a small MIT
package named in the dependency record; the wire format, streaming, the `cmd` registry, a server
serving whatever services are registered); `internal/control/controltest/` (that server on a temp
socket with the services built so far); `internal/cli/clitest/` (builds the binary once per test
package into `t.TempDir()` and runs it); a hidden `control echo`. Proof: a test that starts the
harness and runs `broodforge --root <tmp> control echo --expect ok`, with `--json`, and a streaming
command; no root exits 2; `--expect nope` exits 1 with `error: expect: wanted nope, got ok`; no
server, `error: not-running: no bot at <socket>`. Depends on: P0.

## P3. Engine events and the fake engine (medium)

Implements: `03-architecture.md` (The engine adapter); `05-debug-interface.md` (A turn against the
fake engine: `engine run --fake`); ADR 0020 Amendments 2026-10-16 (S1, S2, C28, C39); ADR 0023
(Types and JSON). Files: `internal/engine/{events,parse,fake}.go` (`events.go` the tagged
`EngineEvent` struct; `parse.go` a port of `spike/run.ts`'s `parseLine`, `usageDelta` and
`interruptedRow` into Go against the same fixtures, the raw line held as `json.RawMessage` or
`map[string]any` in this file only, no `DisallowUnknownFields`); `internal/cli/engine.go` (`run
--fake` only, standalone); tests beside them: every recorded tool result carries `tool_result_meta`
except a `<tool_use_error>` (`interrupted` turn 1); every directory under `fixtures/engine/`
replays, so a toolchain move that changes the parser fails here. What the docs do not say (ADR
0020's Amendments hold the rest): `decidedBy` only when `permission_decision.source` is present; a
resumed turn's `tool_result` may precede its `system/init`; one `assistant` line per content block
shares `message.id` and `message.usage`, so deduplicate before counting; `is_error` is absent, not
`false`, on success; the usage baseline is the last `result` row, never an interrupted one (S2); the
finding's new unknown kinds (Next, plus S1's three) pass as `unknown` quietly. The fake is 03's; its
`received-<n>.json` is ignored under `fixtures/`. Proof: `broodforge engine run --fake
fixtures/engine/basic --turn 1 --prompt x` piped to `grep -c '"type":"unknown"'` gives 3; `parallel
--turn 1` yields `result.deferred` naming the second id and two `tool-use` blocks under one
`messageId`. Depends on: P0, P1, P2.

## P4. Configuration (small)

Implements: `03-architecture.md` (Configuration); ADR 0021 (Configuration); `05-debug-interface.md`
(Configuration; `reload` is P10's). Files: `internal/config/` (parse with `//` keys ignored and
`DisallowUnknownFields` on, the one place it is set; validation, effective merge, last-good read and
write); `internal/cli/config.go` (`check|show`, both `(file)`); `tests/config/` (a valid pair, a
channel id in two project files, an unknown key); credential-id existence is a stub until P6. Proof:
`config check --root tests/config/valid` exit 0; `dup-channel` exit 1 naming both files;
`unknown-key` exit 1 naming the key; `config show --effective --channel 100 --json` prints 03's
merged object. Depends on: P2.

## P5. Store, audit and backup (medium)

Implements: ADR 0021 (Decision, Amendments) and `decisions/0021-backup-and-restore.md` (rehearsal is
P9's); ADR 0023 (Libraries: SQLite); `02-domain.md` (Data model); `05-debug-interface.md` (Usage and
audit queries; Backup snapshot; `reconcile --report-only`). Files: `internal/store/`
(`zombiezen.com/go/sqlite`, imported here only: one writer connection owned by the bot, a small read
pool for the CLI, every connection opened with WAL, `synchronous=FULL`, `foreign_keys=ON` and the
busy timeout; `SetAuthorizer` on the writer denying `UPDATE`, `DELETE` and `DROP TRIGGER` on
`audit_event`; `migrations/0001-initial.sql` through `//go:embed migrations/*.sql`: one table per
record plus ADR 0021's extra columns, the two audit triggers, the schema-newer refusal, read-only
open for the CLI); `internal/store/create.go` (creates and migrates a store under a root; the bot's
start and the tests call it, nothing else); `internal/audit/`; `internal/backup/` (snapshot,
verify); `internal/cli/{usage,audit,backup,reconcile}.go`. Proof: on a root `create.go` made:
`backup snapshot` then `backup verify --expect ok`; `verify` on a truncated copy exits 1; a test
that `UPDATE audit_event` raises the trigger text on a plain connection and the authorizer's refusal
on the writer; `audit --limit 1` prints nothing, exit 0; `usage` prints the total line (reads, no
socket). Depends on: P2, P4.

## P6. Policy, the hook, and credentials (medium)

Implements: `03-hook-contract.md` (all); `03-architecture.md` (`policy`, `hook`, `credentials`
rows); ADR 0022 (Both hosts: the store; Amendments from the spike); `05-debug-interface.md` (The
PreToolUse hook; Credentials, without `--relay`, which is P12's). Files: `internal/policy/` (03's
`policy` row); `internal/hook/` (`broodforge hook pre-tool-use|try|cases`, subcommands of the one
binary, standalone; one handler exiting 2 with `hook: <what>`; no import of `store`, `control` or
`discord`, so startup stays the 9 ms ADR 0023 measured); `tests/hook/cases.jsonl` (every cell in
both modes, a symlink out of the tree, `.credentials.json`, a read inside the directory the
`transcript` field implies); `internal/credentials/` (AES-256-GCM through `crypto/aes` and
`cipher.NewGCM`, key file, `init|add|list|remove|show|login`; `add --kind subscription` calls the
backend's `harness-create`, stubbed until P7); `internal/cli/{hook,credential}.go`. Proof: 05's
`printf '{"tool_name":"Write",...,"transcript_path":"/tmp/cfg/projects/x/s.jsonl"}' | ... broodforge
hook pre-tool-use` line prints the `defer` object, exit 0, and without `transcript_path` exits 2
with `hook: no transcript_path`; `hook try --tool Read --input
'{"file_path":"/tmp/cfg/.credentials.json"}' --transcript-path /tmp/cfg/projects/x/s.jsonl --expect
deny`; `hook cases tests/hook/cases.jsonl` all `ok`; through the harness, `printf scratch` piped
into `broodforge --root <tmp> credential add --kind api-key --user 1001 --label t` prints
`cred-...` and `credential show <id> --json` holds no value. Depends on: P2, P5.

## P7. Spawn (same-account), the real engine, recording (medium; real turns: the owner's)

Implements: `03-architecture.md` (`spawn` row; the `claude-code` implementation; interrupt); ADR
0020 (Mechanism, Interrupt, Settings sources; Amendments 2026-10-16: no `--permission-prompts
none`); ADR 0022 (Windows; Both hosts: delivery) and its companion's `--mcp-config` amendment; ADR
0023 (Windows process control); `05-debug-interface.md` (`engine run --real`, `engine record`; Spawn
preflight, same-account half). Files: `internal/spawn/` (`SpawnBackend`; `same-account`: `os/exec`
with the explicit environment of ADRs 0020 and 0022, the binary from config, the settings template
written to the engine's settings path per credential kind before each process (C26's `--settings
<file>` is the fallback); job objects through `golang.org/x/sys/windows` in `_windows.go` files
only: the bot joins a kill-on-close job at start, each engine gets its own right after `Start`,
interrupt closes it, the spike's `Get-CimInstance` walk plus `taskkill /T /F` is the fallback when
assignment failed; Linux SIGINT, then SIGKILL; `preflight` `(file)`; `harness-create|remove` as
directory operations; `linux-sudo-bwrap` as a stub refusing `error: preflight-backend: held`);
`share/empty/mcp.json` (an empty map, named by every engine process), `share/empty/dir/`,
`share/empty/CLAUDE.md`; `internal/engine/{claudecode,record}.go`; `internal/cli/{engine,spawn}.go`
(`engine run --real` and `engine record` are standalone and read the credential store with the key
file, as 05 describes). Only the findings say: a `defer` resume takes no stdin line; the engine
refuses a standalone `sleep` in Bash before the hook (test a kill with `ping -n 31 127.0.0.1`); the
path is the `.exe`. Proof: `spawn preflight` against a config naming the installed engine; `engine
run --fake` unchanged; a test that a grandchild dies when the engine's job is closed; then, by the
owner: through the harness, `credential add --kind subscription --user 1001 --label owner` prints
`cred-...` and `credential login <id>` prints the login command he runs (no turn); then one real
turn each: `engine run --real --credential <id> --cwd <tmp tree> --prompt "Read hello.txt and reply
with its one word."` streams a `result`; `engine record --credential <id> --cwd <tmp tree> --out
<tmp>/rec --prompt ...` writes a directory that `engine run --fake <tmp>/rec` replays. Depends on:
P3, P6. Must not touch: `spike/`; `~/.claude`.

## P8. Git: canonical, clone, bundle, attribution, handoff (medium)

Implements: `03-architecture.md` (Git); ADR 0022 (Working trees) and its companion (Clone at session
start; Handoff at session end); `05-debug-interface.md` (Git handoff). Files: `internal/git/` (refs,
clone through the backend's `worktree-create`, `core.hooksPath` to `share/hooks/commit-msg`, bundle,
verify, fetch, the attribution check, push with `GIT_ASKPASS` naming `share/askpass.sh` or
`share/askpass.cmd`, two-line scripts that run the hidden `broodforge git-askpass`, the GitHub
pull-request call behind an interface with `--dry-run`); `internal/spawn/` gains
`worktree-create|bundle|remove` for `same-account`; `internal/cli/worktree.go`. Proof: a test that
seeds a registration, a session, its driver and one usage row through the store API (the rows P10
will write), makes a canonical and a bare remote in a temp root, runs `worktree-create`, commits in
the tree under the four variables (through `os/exec`, never the Bash tool: `07-spike-plan.md`,
Fixture commits), then through the harness: `worktree check s0001` all `ok`, `worktree handoff
s0001` pushes `broodforge/<instance>/s0001` to the remote; a commit with `--author "X
<x@example.com>"` yields `handoff-refused-attribution`. Depends on: P5, P7.

## P9. The scratch root and the backup rehearsal (small)

Implements: `05-cli-conventions.md` (Scratch); ADR 0021 (Scratch);
`decisions/0021-backup-and-restore.md` (rehearsal); `05-debug-interface.md` (Backup snapshot and
restore rehearsal). Files: `internal/cli/scratch.go`; `internal/config/scratchtemplate.go` (dry-run,
`fake`, `fixtures/engine/`, same-account, the key under the root, channels `100` and `200`,
`cred-scratch`); `internal/store/scratch.go` (creation through P5's `create.go`; deletion through
`worktree-remove`); `backup rehearse` in `internal/backup/` (a rehearsal is itself a scratch root).
Proof: `scratch new --json` prints a root whose `config check` passes; `scratch run -- broodforge
config check` deletes the root on exit 0 and keeps it on exit 1 printing the path; `scratch delete`
of a non-scratch path is `error: not-scratch`; `backup rehearse --expect ok` on a scratch root.
Depends on: P4, P5, P6, P8.

## P10. Sessions, turns, modes, timeouts, usage rows (large; owner sees the trace wording)

Implements: `03-architecture.md` (`session` row; Registration and participants through Failure
modes, Startup 1 to 8); ADR 0020 (Permission relay and modes; Usage and attribution; Amendments);
ADR 0021 (Reconciliation); `05-debug-interface.md` (Bot lifecycle through Prompt-mode defer and
resume; `config reload`; `session reach-report`, an empty bind list until P14). Files:
`internal/session/` (`SessionService` and everything 03's session row and Data flow name: the
`From:` blocks, the system prompt append, the `permission decision` rows with `dropped` and
`decidedBy` `config` (the `hook bypassed` audit event), the classifier that strips the engine's
wrapper, usage rows, idle and prompt timeouts with an injectable clock; one goroutine per turn
reading its engine's events, nothing awaiting another session, `-race` clean on both hosts);
`internal/cli/{serve,status,stop,register,session}.go` (`serve` steps 1 to 8, dry-run only; the
harness of P2 retires into it); `tests/repro/full-session.sh` and `tests/repro/defer-resume.sh`,
both reading `session tail` and `audit` only. Mid-pass commit after (a) lifecycle, registration, a
policy-mode turn against `basic`; then (b) prompt mode against `defer` and `parallel`, timeouts,
interrupt against `interrupted`, usage. Proof: `scratch run -- bash tests/repro/full-session.sh`
exit 0, its last steps `worktree handoff s0001` and `stop`; `defer-resume.sh` exit 0 with `session
show` printing `pending:` while waiting and the prompt timeout's deny in `audit`; a `parallel` turn
leaves one `dropped` row there; `usage --session s0001 --by turn --json` equals the fixture's
deltas. Depends on: P2 to P9.

## P11. Discord dry-run: renderer, coalescer, outbox, transcript, panel (medium; owner sees)

Implements: `04-ux.md` (every screen and text); `03-architecture.md` (`discord` row; Data flow steps
1 and 6); ADR 0023 (Libraries: Discord); `05-debug-interface.md` (The Discord adapter; `session
outbox`, `transcript`, `panel`). Files: `internal/discord/` (03's `discord` row without the gateway:
renderer, coalescer with the injectable clock and a global budget of 50 requests per second per bot
beneath the per-session one, mask, custom ids, command parsing, `inject`; disgo's types appear here
and nowhere else); `internal/cli/{discord,panel}.go`; `tests/discord/burst.jsonl`; the two repro
scripts of P10 gain their `session outbox` checks (the buttons post, the timeout trace line). Proof:
`discord render --events tests/discord/burst.jsonl --clock fake --expect 'edits<=1/s'`; `discord
inject message --channel 100 --user 1001 --roles 10 "sk-ant-..."` yields the refusal and an
`action-refused` event with no key text; a scratch session shows 04's working message, footer,
buttons post and summary in `session outbox`, which the owner reads. Depends on: P10.

## P12. Discord live: gateway, commands, buttons, venues, the relay (medium; owner sees; needs his guild)

Implements: `04-ux.md` (Registration through Starting; Credentials); `03-architecture.md` (Startup 9
and 10; Failure modes, `discord` row); `05-debug-interface.md` (The Discord adapter: `check`, `serve
--discord live`; `credential login --relay`, off by default). Files: `internal/discord/gateway.go`
(the disgo client pinned in `go.mod`, slash commands, interactions, the private-channel venue, the
notice channel, the DM relay; resume and the rate limiter are disgo's; the REST calls used are
listed in one place in this file, the surface a swap to arikawa must cover); `discord check` in
`internal/cli/discord.go`; `internal/deps/deps.go` deleted. Proof: `discord check --guild <id>`
lists every configured channel `present`, `perms:ok`; `serve --discord live` on a scratch-shaped
root with the fake: the owner registers, starts, sends, clicks Allow and ends from Discord while an
agent reads `session tail` and `audit`; with P7's real engine this is the vision's first usable
version. Depends on: P11; the bot token and a test guild.

## P13. The Windows instance (medium; owner steps)

Implements: ADR 0022 (Windows); `03-architecture.md` (Process model, Windows column: the bot task
and the nightly backup task); `../operator/2026-10-14-pc-broodforge-account-and-task.md`,
`../operator/2026-10-14-credential-store-key-per-host.md`; C37 (`07-spike-plan.md`). Files:
`docs/reference/pc-host-setup.md` [no-check: written in this pass] (the two Task Scheduler actions
naming `bin\broodforge.exe` in full, built by `CGO_ENABLED=0 go build -o bin/ ./cmd/broodforge`, the key file path, the engine install for the `broodforge`
account by the native installer, the toolchain-download lift of ADR 0023); `spawn preflight` reports
the account. Proof, recorded as the C37 finding: C37's three actions from the scheduled task;
`status` over the named pipe from the owner's account; a bot stopped by the task leaves no engine
process (the job object). Depends on: P10, P12; the owner creating the account.

## P14. The Linux backend `linux-sudo-bwrap` and the droplet (large; HELD)

Held until the droplet rows of both findings land (`07-spike-droplet.md`); its commit deletes
`spike/`. Implements: ADR 0022 (Linux) and its companion in full, with the ADR 0023 amendment (no
runtime bind, the binary as entry wrapper); `03-architecture.md` (Process model, Linux column, the
systemd timer; Signals and leftovers). Files: `internal/spawn/linuxsudobwrap.go` and the
`engine-entry` subcommand (checks, then `syscall.Exec` into the engine; a Linux-only file), the
`namei` and sudoers checks in `spawn preflight`, the systemd unit and timer,
`docs/reference/droplet-host-setup.md` [no-check: written when the droplet operator entries are
done] (Go and `gcc` for `-race` included). Proof: the droplet builds `bin/broodforge` with `CGO_ENABLED=0 go
build -o bin/ ./cmd/broodforge` from the pinned source and `CGO_ENABLED=1 go test -race ./...` passes; C30 and C31 by hand from a code
session, `reach-report` matching the bind list, the Interrupt steps yielding a `result`. Depends on:
P7 to P10.
