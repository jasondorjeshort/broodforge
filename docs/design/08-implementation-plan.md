# Implementation plan: coder passes for `src/`

Date: 2026-10-16. Status: plan, held until proposal 0023 settles the language (the pass order stands; P0 and file names follow the language); written after the PC spike closed every PC row (ADRs 0020 and
0022, Amendments of 2026-10-16) and `07-spike-plan.md`'s exit criterion was met for host-neutral
code. Each pass is one coder brief (`brief` skill); done is the `debug-interface` skill's
checklist, its Proof run on a clean checkout from the doc alone, and the `05-debug-interface.md`
section it builds matching the code. Every pass reads `CLAUDE.md`, `docs/design/README.md`,
`03-architecture.md` (Components) and `05-cli-conventions.md` first, then its Implements line.
Sizes: small is one sitting, medium one to two, large needs a mid-pass commit. "Owner sees"
means the output is 04's.

Standing rules. A pass's Files line is its boundary (every name there is created by the pass
[no-check: the whole plan names files not yet written]); "must not touch" is stated only where it
lies beyond the obvious. No `if windows` outside `spawn`, `worktree-remove` and `control`
(`03-architecture.md`, Components, "Host differences live in"). A real engine turn spends the
owner's plan and runs only where a pass says "real turn", on his word. `spike/` stays until the droplet rows
land, because the droplet run copies it (P14 deletes it). Every Proof's `broodforge` is
`bin/broodforge` (Git Bash) or `bin\broodforge.cmd` (PowerShell) from the repository root, each
running `node <repo>/dist/cli/main.js`; nothing goes on `PATH`, `npm link` is forbidden (global;
`docs/reference/claude-code-on-windows.md`, "npm on this PC"). Command classes
(`05-cli-conventions.md`, Surface): standalone, no root and no socket (`hook *`, `engine run`,
`engine record`, `engine-entry`, `scratch *`); `(file)` and read commands, which take `--root`
and open the store file read-only; write commands, socket clients that exit `not-running` until
P10's `serve`, so before P10 a write command is proven through P2's harness.

## P0. Toolchain and repository skeleton (small)

Implements: ADR 0020 (Language bundle); `03-architecture.md` (Components); `.claude/rules/README.md`;
`../operator/README.md` (Filing an entry). Files: `package.json` (`type` module, `engines.node`
`>=24`, this scripts block):
```
"build": "tsc -p tsconfig.json",   "check": "tsc -p tsconfig.json --noEmit",
"test": "node --test 'dist/**/*.test.js'",   "start": "node dist/cli/main.js"
```
All four ADR 0020 dependencies installed here (`discord.js` 14, `better-sqlite3`; dev `typescript`,
`@types/node`); `.nvmrc` `24`; `tsconfig.json` (strict, `nodenext`, `rootDir` `src`, `outDir`
`dist`: every emitted file, tools included, lives under `src/`); `src/cli/main.ts` (usage, exit
2); an empty `index.ts` per component of 03's table; `bin/broodforge` (the ADR 0022 companion's
`sh` dispatcher: `engine-entry` to shell, the rest to Node) and `bin/broodforge.cmd`;
`tests/README.md`; `.gitignore` (`dist/`, `node_modules/`, `fixtures/**/received-*.json`);
`.claude/rules/typescript.md` (SPDX first line; strict `tsc` clean; `node:test` only; never `npm link`).
Node: ADR 0020 pins 24, the PC runs 22. The coder installs nothing machine-wide: it files
`docs/operator/2026-10-16-pc-node-24.md` [no-check: written by the coder] with its README line,
asking the owner to install 24 beside 22; `engine-strict` stays off; `@types/node` 24 cannot flag
a 24-only API, so the Node 22 run of `npm test` is the check, and every pass report states the
version run. A bare `npm install` is destructive on this PC (the reference above): the only form,
here and for a later `npm rebuild better-sqlite3`, is `npm_config_userconfig=<empty file> npm --prefix <dir> install`.
Proof: `npm run check`; `npm run build`; `npm test` (one placeholder test); `node dist/cli/main.js`,
`bin/broodforge` and `bin/broodforge.cmd` each exit 2 with usage on stderr. Depends on: nothing.

## P1. Fixture import (small; a person reads every file before the commit)

Implements: `07-spike-plan.md` (The code: fixtures kept); `03-architecture.md` (The engine adapter);
`../ideas/2026-10-16-fixtures-need-a-path-rewrite-before-they-move-into-the-repo.md`, deleted in
this pass's commit (the queue rule). Files: `src/tools/import-fixture.ts`;
`fixtures/engine/<name>/turn-<n>.jsonl` and `fixture.json` for `basic`, `defer`, `interrupted`,
`parallel`, `hook-exit2`, `hook-exit1`, `hook-sleep`, `scrub`, `settings-a`, `settings-b` from
`C:/Users/<pc user>/AppData/Local/Temp/bf-spike/fx/<name>/`; `fixtures/engine/README.md` (one
line per fixture: claim, turns, what it shows). The rewrite works on whole path tokens: a token
starting with any form of the PC home or spike root (`C:\Users\<user>`, `C:/Users/<user>`, the
JSON-escaped `C:\\Users\\<user>`) becomes `/home/user` or `/home/user/bf-spike` with the rest of
its separators normalised to `/`; the engine's project slug `C--Users-<user>-AppData-Local-Temp-bf-spike-tree`
(`memory_paths.auto` and the like) becomes `-home-user-bf-spike-tree`. Only `turn-<n>.jsonl` is
imported; `meta-<n>.json` yields `fixture.json` (engine version, date, claim, the exit code per
turn: `interrupted` turns 1 and 2 exit 1). No test asserts on an original path. Proof: one `node dist/tools/import-fixture.js <src> fixtures/engine/<name>`
per name; `grep -rlE "<pc user>|[A-Za-z]:\\\\" fixtures/` empty. Depends on: P0. Must not touch: `spike/`.

## P2. CLI conventions, the control socket, the in-process harness (medium)

Implements: `05-cli-conventions.md` (all); `03-architecture.md` (Debug surface attachment; the
`cli` and `control` rows); `05-debug-interface.md` (Conventions). Files: `src/cli/` (dependency-free
argument parsing, `--root`, `--json`, `--expect`, the one formatter, exit codes, the error line,
the command classes above); `src/control/` (unix socket and named pipe, the wire format,
streaming, the `cmd` registry, a server serving whatever services are registered);
`tests/helpers/serve-inprocess.ts` (starts that server on a temp socket with the services built
so far, for the CLI to talk to); a hidden `control echo`. Proof: a `node:test` that starts
the harness and runs `broodforge --root <tmp> control echo --expect ok`, with `--json`, and a
streaming command; no root exits 2; `--expect nope` exits 1 with `error: expect: wanted nope,
got ok`; no server, `error: not-running: no bot at <socket>`. Depends on: P0.

## P3. Engine events and the fake engine (medium)

Implements: `03-architecture.md` (The engine adapter); `05-debug-interface.md` (A turn against
the fake engine: `engine run --fake`); ADR 0020 Amendments 2026-10-16 (S1, S2, C28, C39).
Files: `src/engine/{events,parse,fake,index}.ts` (`parse.ts` seeded from `spike/run.ts`
`parseLine`, `usageDelta`, `interruptedRow`: copied, not imported); `src/cli/engine.ts` (`run
--fake` only, standalone); tests beside them, one asserting that every recorded tool result carries `tool_result_meta` except a `<tool_use_error>` (`interrupted` turn 1).
What the docs do not say: `is_error` is absent, not `false`, on a successful `tool_result`; one
`assistant` line per content block sharing `message.id` and `message.usage`, so deduplicate
before counting; `permission_decision.source` becomes `decidedBy` only when present; the kinds
the ADR 0020 finding lists as new unknowns (its Next section, plus S1's three) pass as `unknown`
quietly; a resumed turn's `tool_result` may precede its `system/init`; the usage baseline is the
last `result` row, never an interrupted row (S2). The fake is 03's; its `received-<n>.json` is
ignored under `fixtures/`. Proof: `broodforge engine run --fake fixtures/engine/basic --turn 1 --prompt x | grep
-c '"type":"unknown"'` is 3; `parallel --turn 1` yields `result.deferred` naming the second id
and two `tool-use` blocks under one `messageId`. Depends on: P0, P1, P2.

## P4. Configuration (small)

Implements: `03-architecture.md` (Configuration); ADR 0021 (Configuration); `05-debug-interface.md`
(Configuration; `reload` is P10's). Files: `src/config/` (parse with `//` keys ignored,
validation, effective merge, last-good read and write); `src/cli/config.ts` (`check|show`, both
`(file)`); `tests/config/` (a valid pair, a channel id in two project files, an unknown key);
credential-id existence is a stub until P6. Proof: `config check --root tests/config/valid` exit
0; `dup-channel` exit 1 naming both files; `config show --effective --channel 100 --json` prints 03's merged object. Depends on: P2.

## P5. Store, audit and backup (medium)

Implements: ADR 0021 (Decision) and `decisions/0021-backup-and-restore.md` (rehearsal is P9's);
`02-domain.md` (Data model); `05-debug-interface.md` (Usage and audit queries; Backup snapshot;
`reconcile --report-only`). Files: `src/store/` (`better-sqlite3`, WAL,
`migrations/0001-initial.ts` exporting the SQL as a string, since `tsc` copies no `.sql`: one
table per record plus ADR 0021's extra columns, the two audit triggers, the schema-newer
refusal, read-only open for the CLI); `src/store/create.ts` (creates and migrates a store under
a root; the bot's start and the tests call it, nothing else); `src/audit/`; `src/backup/`
(snapshot, verify); `src/cli/{usage,audit,backup,reconcile}.ts`.
Proof: on a root `create.ts` made: `backup snapshot` then `backup verify --expect ok`; `verify`
on a truncated copy exits 1; a test that `UPDATE audit_event` raises the trigger text; `audit --limit 1` prints nothing, exit 0, and `usage` prints the total line (reads, no socket). Depends on: P2, P4.

## P6. Policy, the hook, and credentials (medium)

Implements: `03-hook-contract.md` (all); `03-architecture.md` (`policy`, `hook`, `credentials`
rows); ADR 0022 (Both hosts: the store; Amendments from the spike); `05-debug-interface.md` (The
PreToolUse hook; Credentials, without `--relay`, which is P12's). Files: `src/policy/` (03's
`policy` row); `src/hook/` (`pre-tool-use|try|cases`, standalone; one handler exiting 2 with
`hook: <what>`); `tests/hook/cases.jsonl` (every cell in both modes, a symlink out of the tree,
`.credentials.json`, a read inside the directory the `transcript` field implies);
`src/credentials/` (AES-256-GCM store, key file, `init|add|list|remove|show|login`; `add --kind
subscription` calls the backend's `harness-create`, stubbed until P7); `src/cli/{hook,credential}.ts`.
Proof: 05's `printf '{"tool_name":"Write",...,"transcript_path":"/tmp/cfg/projects/x/s.jsonl"}'
| ... broodforge hook pre-tool-use` line prints the `defer` object, exit 0, and without
`transcript_path` exits 2 with `hook: no transcript_path`; `hook try --tool Read --input
'{"file_path":"/tmp/cfg/.credentials.json"}' --transcript-path /tmp/cfg/projects/x/s.jsonl
--expect deny`; `hook cases tests/hook/cases.jsonl` all `ok`; through the harness, `printf scratch
| broodforge --root <tmp> credential add --kind api-key --user 1001 --label t` prints `cred-...` and `credential show <id> --json` holds no value. Depends on: P2, P5.

## P7. Spawn (same-account), the real engine, recording (medium; real turns: the owner's)

Implements: `03-architecture.md` (`spawn` row; the `claude-code` implementation; interrupt); ADR
0020 (Mechanism, Interrupt, Settings sources; Amendments 2026-10-16: no `--permission-prompts
none`); ADR 0022 (Windows; Both hosts: delivery) and its companion's `--mcp-config` amendment;
`05-debug-interface.md` (`engine run --real`, `engine record`; Spawn preflight, same-account half).
Files: `src/spawn/` (`SpawnBackend`; `same-account`: `child_process` with the explicit
environment of ADRs 0020 and 0022, the binary from config, the settings template written to the
engine's own settings path per credential kind before each process (C26's `--settings <file>`
is the fallback), interrupt as the spike did it (Windows `Get-CimInstance` walk then `taskkill
/T /F`; Linux SIGINT, then SIGKILL), `preflight` `(file)`, `harness-create|remove` as directory
operations; `linux-sudo-bwrap` as a stub refusing `error: preflight-backend: held`); `share/empty/mcp.json`
(an empty map, named by every engine process), `share/empty/dir/`, `share/empty/CLAUDE.md`;
`src/engine/{claude-code,record}.ts`; `src/cli/{engine,spawn}.ts` (`engine run --real` and
`engine record` are standalone: they read the credential store with the key file, as 05
describes them; a 05 clarification to report). Only the findings say: a `defer` resume takes no
stdin line; the engine refuses a standalone `sleep` in Bash before the hook (test a kill with
`ping -n 31 127.0.0.1`); the path is the `.exe`. Proof: `spawn preflight` against a config naming
the installed engine; `engine run --fake` unchanged; then, by the owner: through the harness,
`credential add --kind subscription --user 1001 --label owner` prints `cred-...` and `credential
login <id>` prints the login command he runs (no turn); then one real turn each: `engine run
--real --credential <id> --cwd <tmp tree> --prompt "Read hello.txt and reply with its one word."`
streams a `result`; `engine record --credential <id> --cwd <tmp tree> --out <tmp>/rec --prompt
...` writes a directory that `engine run --fake <tmp>/rec` replays. Depends on: P3, P6. Must not touch: `spike/`; the owner's `~/.claude`.

## P8. Git: canonical, clone, bundle, attribution, handoff (medium)

Implements: `03-architecture.md` (Git); ADR 0022 (Working trees) and its companion (Clone at
session start; Handoff at session end); `05-debug-interface.md` (Git handoff). Files: `src/git/` (refs, clone through the backend's `worktree-create`,
`core.hooksPath` to `share/hooks/commit-msg`, bundle, verify, fetch, the attribution check, push
with `bin/broodforge-askpass`, the GitHub pull-request call behind an interface with `--dry-run`);
`src/spawn/same-account.ts` gains `worktree-create|bundle|remove`; `src/cli/worktree.ts`.
Proof: a `node:test` that seeds a registration, a session, its driver and one usage row through
the store API (the rows P10 will write), makes a canonical and a bare remote in a temp root, runs
`worktree-create`, commits in the tree under the four variables (through `child_process`, never
the Bash tool: `07-spike-plan.md`, Fixture commits), then through the harness: `worktree check
s0001` all `ok`, `worktree handoff s0001` pushes `broodforge/<instance>/s0001` to the remote; a
commit with `--author "X <x@example.com>"` yields `handoff-refused-attribution`. Depends on: P5, P7.

## P9. The scratch root and the backup rehearsal (small)

Implements: `05-cli-conventions.md` (Scratch); ADR 0021 (Scratch); `decisions/0021-backup-and-restore.md`
(rehearsal); `05-debug-interface.md` (Backup snapshot and restore rehearsal). Files: `src/cli/scratch.ts`; `src/config/scratch-template.ts` (dry-run, `fake`, `fixtures/engine/`,
same-account, the key under the root, channels `100` and `200`, `cred-scratch`);
`src/store/scratch.ts` (creation through P5's `create.ts`; deletion through `worktree-remove`);
`backup rehearse` in `src/backup/` (a rehearsal is itself a scratch root). Proof: `scratch new
--json` prints a root whose `config check` passes; `scratch run -- broodforge config check`
deletes the root on exit 0 and keeps it on exit 1 printing the path; `scratch delete` of a
non-scratch path is `error: not-scratch`; `backup rehearse --expect ok` on a scratch root. Depends on: P4, P5, P6, P8.

## P10. Sessions, turns, modes, timeouts, usage rows (large; owner sees the trace wording)

Implements: `03-architecture.md` (`session` row; Registration and participants through Failure
modes, Startup 1 to 8); ADR 0020 (Permission relay and modes; Usage and attribution; Amendments);
ADR 0021 (Reconciliation); `05-debug-interface.md` (Bot lifecycle through Prompt-mode defer and
resume; `config reload`; `session reach-report`, an empty bind list until P14). Files: `src/session/` (`SessionService` and everything
03's session row and Data flow name: the `From:` blocks, the system prompt append, the
`permission decision` rows with `dropped` and `decidedBy` `config` (the `hook bypassed` audit
event), the classifier that strips the engine's wrapper, usage rows, idle and prompt timeouts
with an injectable clock); `src/cli/{serve,status,stop,register,session}.ts` (`serve` steps 1
to 8, dry-run only; the harness of P2 retires into it); `tests/repro/full-session.sh`
and `tests/repro/defer-resume.sh`, both reading `session tail` and `audit` only. Mid-pass commit
after (a) lifecycle, registration, a policy-mode turn against `basic`; then (b) prompt mode
against `defer` and `parallel`, timeouts, interrupt against `interrupted`, usage. Proof:
`scratch run -- bash tests/repro/full-session.sh` exit 0, its last steps `worktree handoff
s0001` from the CLI and `stop`; `defer-resume.sh` exit 0 with `session show` printing `pending:`
while waiting and the prompt timeout's deny in `audit`; a `parallel` turn leaves one `dropped`
row there; `usage --session s0001 --by turn --json` equals the fixture's deltas. Depends on: P2 to P9.

## P11. Discord dry-run: renderer, coalescer, outbox, transcript, panel (medium; owner sees)

Implements: `04-ux.md` (every screen and text); `03-architecture.md` (`discord` row; Data flow
steps 1 and 6); `05-debug-interface.md` (The Discord adapter; `session outbox`, `transcript`,
`panel`). Files: `src/discord/` (03's `discord` row without the gateway:
renderer, coalescer with the injectable clock, mask, custom ids, command parsing, `inject`);
`src/cli/{discord,panel}.ts`; `tests/discord/burst.jsonl`; the two repro scripts of P10 gain
their `session outbox` checks (the buttons post, the timeout trace line). Proof: `discord render
--events tests/discord/burst.jsonl --clock fake --expect 'edits<=1/s'`; `discord inject message
--channel 100 --user 1001 --roles 10 "sk-ant-..."` yields the refusal and an `action-refused`
event with no key text; a scratch session shows 04's working message, footer, buttons post and summary in `session outbox`. The owner reads one run's outbox. Depends on: P10.

## P12. Discord live: gateway, commands, buttons, venues, the relay (medium; owner sees; needs his guild)

Implements: `04-ux.md` (Registration through Starting; Credentials); `03-architecture.md`
(Startup 9 and 10; Failure modes, `discord` row); `05-debug-interface.md` (The Discord adapter:
`check`, `serve --discord live`; `credential login --relay`, off by default). Files: `src/discord/gateway.ts`
(discord.js client, slash commands, interactions, the private-channel venue, the notice channel,
the DM relay); `discord check` in `src/cli/discord.ts`. Proof: `discord check --guild <id>`
lists every configured channel `present`, `perms:ok`; `serve --discord live` on a scratch-shaped
root with the fake: the owner registers, starts, sends, clicks Allow and ends from Discord while
an agent reads `session tail` and `audit`; with P7's real engine this is the vision's first
usable version. Depends on: P11; the bot token and a test guild.

## P13. The Windows instance (medium; owner steps)

Implements: ADR 0022 (Windows); `03-architecture.md` (Process model, Windows column: the bot task
and the nightly backup task); `../operator/2026-10-14-pc-broodforge-account-and-task.md`, `../operator/2026-10-14-credential-store-key-per-host.md`; C37 (`07-spike-plan.md`). Files:
`docs/reference/pc-host-setup.md` [no-check: written in this pass] (the two Task Scheduler
actions, the key file path, the engine install for the `broodforge` account); `spawn preflight`
reports the account. Proof, recorded as the C37 finding: C37's three actions from the scheduled
task; `status` over the named pipe from the owner's account.
Depends on: P10, P12; the owner creating the account. Must not touch: the owner's profile.

## P14. The Linux backend `linux-sudo-bwrap` and the droplet (large; HELD)

Held until the droplet rows of both findings land (`07-spike-droplet.md`); its commit deletes
`spike/`. Implements: ADR 0022 (Linux) and its companion in full; `03-architecture.md` (Process
model, Linux column, the systemd timer; Signals and leftovers). Files: `src/spawn/linux-sudo-bwrap.ts`,
the `engine-entry` shell half of `bin/broodforge`, the `namei` and sudoers checks in `spawn
preflight`, the systemd unit and timer, `docs/reference/droplet-host-setup.md` [no-check:
written when the droplet operator entries are done]. Proof: C30 and C31 by hand from a code
session, `reach-report` matching the bind list, the Interrupt steps yielding a `result`. Depends on: P7 to P10.
