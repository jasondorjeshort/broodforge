# Implementation plan: coder passes for `src/`

Date: 2026-10-16. Status: plan, written after the PC spike closed every PC row (ADRs 0020 and
0022, Amendments of 2026-10-16) and `07-spike-plan.md`'s exit criterion was met for host-neutral
code. Each pass is one coder brief (`brief` skill): the coordinator copies it, adds attribution
and usage state, and launches. Done means the Proof runs on a clean checkout from the doc alone
(`debug-interface` skill) and the `05-debug-interface.md` section it builds matches the code;
the `tester` writes the `node:test` cases, the `reviewer` reads the diff, the `runner` commits.
Every pass reads `CLAUDE.md`, `docs/design/README.md`, `03-architecture.md` (Components) and
`05-cli-conventions.md` first, then its Implements line. Sizes: small is one sitting, medium one
to two, large needs a mid-pass commit. "Owner sees" marks a pass whose output is 04's.

Standing rules. No code outside the pass's Files (each Files line names files the pass creates
[no-check: the whole plan names files not yet written]); no `if windows` outside `spawn`,
`worktree-remove` and `control` (`06-platforms.md`, Sharing); the SPDX header on every source
file; no dependency beyond ADR 0020's four without the coordinator's word; the fake engine and
the scratch root are the test bed, and a real engine turn spends the owner's plan, so it runs
only where a pass says "real turn", on the owner's word; `spike/` stays until P7 deletes it. A
pass that finds the design silent makes the smallest choice and reports it; one that finds it
wrong stops and reports (`finding` skill). Order: P0 to P2 touch no store; P3 to P6 need no
engine; P7 is the first real-engine call, after the hook because a real turn needs the settings
file; P10 is the product without a front end; P11 adds the words, P12 the wire; P13 and P14 are
the per-host installs, the droplet's held.

## P0. Toolchain and repository skeleton (small)

Implements: ADR 0020 (Language bundle); `03-architecture.md` (Components); `.claude/rules/README.md`.
Files: `package.json` (`type` module, `engines.node` `>=24`, scripts `build` `tsc -p tsconfig.json`,
`check` `tsc --noEmit`, `test` `node --test` over `dist/`, `start` `node dist/cli/main.js`),
`.nvmrc` `24`, `tsconfig.json` (strict, `nodenext`, `src` to `dist`; the spike's as seed),
`src/cli/main.ts` (usage, exit 2), an empty `index.ts` per component of 03's table, `bin/broodforge`
(the ADR 0022 companion's `sh` dispatcher: `engine-entry` to shell, the rest to Node) and
`bin/broodforge.cmd`, `tests/README.md`, `.gitignore`, `.claude/rules/typescript.md` (paths `src/**`,
`tests/**`, `bin/**`: SPDX first line; `tsc --noEmit` clean with `noUnusedLocals`; no `any` outside
the parser's raw layer; `node:test` only; CLI output through P3's one formatter). No formatter, no
eslint: strict `tsc` is the lint until one is missed.
Node: ADR 0020 pins 24, the PC runs 22. The coder installs nothing machine-wide: it files
`docs/operator/2026-10-16-pc-node-24.md` [no-check: written by the coder] asking the owner to
install 24 beside 22, keeps `engine-strict` off so 22 runs meanwhile, avoids APIs 22 lacks, and
every pass report states the version run. On this PC a bare `npm install` is destructive
(`docs/reference/claude-code-on-windows.md`, "npm on this PC"); the only install form is
`npm_config_userconfig=<empty file> npm --prefix <dir> install`, once, `package-lock.json` committed.
Proof: `npm run check`; `npm test` (one placeholder test); `node dist/cli/main.js`, `bin/broodforge`
and `bin/broodforge.cmd` each exit 2 with usage on stderr. Depends on: nothing. Must not touch: `spike/`.

## P1. Fixture import (small; a person reads every file before the commit)

Implements: `07-spike-plan.md` (The code: fixtures kept); `03-architecture.md` (The engine
adapter); `../ideas/2026-10-16-fixtures-need-a-path-rewrite-before-they-move-into-the-repo.md`.
Files: `tools/import-fixture.ts` (standalone, outliving the spike), `fixtures/engine/<name>/turn-<n>.jsonl`
and `fixture.json` for `basic`, `defer`, `interrupted`, `parallel`, `hook-exit2`, `hook-exit1`,
`hook-sleep`, `scrub`, `settings-a`, `settings-b` from `C:/Users/<pc user>/AppData/Local/Temp/bf-spike/fx/<name>/`,
and `fixtures/engine/README.md` (one line per fixture: claim, turns, what it shows).
The rewrite: every form of the PC home directory and spike root (`C:\Users\<user>`, `C:/Users/<user>`,
the JSON-escaped `C:\\Users\\<user>`) becomes `/home/user` and `/home/user/bf-spike`; the username
sits in `file_path`, `cwd`, `command`, `permission_denials`, Bash stdout and `tool_result` text.
Only `turn-<n>.jsonl` is imported; `meta-<n>.json` yields `fixture.json`: `{"engine":"2.1.296",
"recorded":"2026-10-16","claim":"<id>","turns":[{"exit":0},...]}` (`interrupted` turns 1 and 2
exit 1). No test asserts on an original path. A person reads every file before the runner commits.
Proof: one `node dist/tools/import-fixture.js <src> fixtures/engine/<name>` per name; `grep -rl
"<pc user>" fixtures/` empty. Depends on: P0. Must not touch: `spike/`, the source fixtures.

## P2. Engine events and the fake engine (medium)

Implements: `03-architecture.md` (The engine adapter, all of it); `05-debug-interface.md` (A
turn against the fake engine: `engine run --fake`); ADR 0020 Amendments 2026-10-16 (S1, S2, C28, C39).
Files: `src/engine/{events,parse,fake,index}.ts` (`parse.ts` seeded from `spike/run.ts`
`parseLine`, `usageDelta`, `interruptedRow`: copied, not imported), `src/cli/engine.ts` (`run
--fake` only), tests beside them.
What the docs do not say: `is_error` is absent, not `false`, on a successful `tool_result`; one
`assistant` line per content block sharing `message.id` and identical `message.usage`, so
deduplicate before counting; `tool_result_meta` is undocumented, so `permission_decision.source`
becomes `decidedBy` only when present; kinds to pass as `unknown` quietly: `system/commands_changed`,
`rate_limit_event`, `system/thinking_tokens`, `command_lifecycle`, `attachment`, `system/permission_denied`,
`system/background_tasks_changed`, `system/task_started`, `system/task_notification`, `tool_progress`;
a resumed turn's `tool_result` may precede its `system/init`; the usage baseline is the last
`result` row, never an interrupted row (S2). The fake: one `turn-<n>.jsonl` per `startTurn`,
`received-<n>.json` per request, the per-line delay, `interrupt()` ending the file early with `exit` `interrupted: true`.
Proof: `broodforge engine run --fake fixtures/engine/basic --turn 1 --prompt x | grep -c
'"type":"unknown"'` is 3; `parallel --turn 1` yields `result.deferred` naming the second id and
two `tool-use` blocks under one `messageId`; a malformed line in a new directory fails `--expect
result` with exit 1 naming the line. Depends on: P0, P1. Must not touch: `src/spawn`.

## P3. CLI conventions and the control socket (medium)

Implements: `05-cli-conventions.md` (all); `03-architecture.md` (Debug surface attachment; the
`cli` and `control` rows); `05-debug-interface.md` (Conventions).
Files: `src/cli/` (dependency-free argument parsing, `--root`, `--json`, `--expect`, the one text
and JSONL formatter, exit codes, the error line), `src/control/` (unix socket and named pipe, the
wire format, streaming, the `cmd` registry). No `serve` yet: a hidden `control echo` command.
Proof: `broodforge --root <tmp> control echo --expect ok`, the same with `--json`; no root exits
2; `--expect nope` exits 1 with `error: expect: wanted nope, got ok`; a streaming test.
Depends on: P0. Must not touch: `src/session`, `src/discord`.

## P4. Configuration (small)

Implements: `03-architecture.md` (Configuration); ADR 0021 (Configuration); `05-debug-interface.md`
(Configuration; `reload` is P10's). Files: `src/config/` (parse with `//` keys ignored,
validation, effective merge, last-good read and write), `src/cli/config.ts` (`check|show`),
`tests/config/` (a valid pair, a channel id in two project files, an unknown key); credential-id
existence is a stub until P6. Proof: `config check --root tests/config/valid` exit 0; `dup-channel`
exit 1 naming both files; `config show --effective --channel 100 --json` prints 03's merged
object. Depends on: P3. Must not touch: `src/credentials`.

## P5. Store, audit and backup (medium)

Implements: ADR 0021 (Decision) and `decisions/0021-backup-and-restore.md`; `02-domain.md` (Data
model); `03-architecture.md` (`store`, `audit`, `backup` rows); `05-debug-interface.md` (Usage and
audit queries; Backup snapshot and restore rehearsal; `reconcile --report-only`).
Files: `src/store/` (`better-sqlite3`, WAL, `migrations/0001-initial.sql`: one table per record
plus ADR 0021's extra columns, the two audit triggers, the schema-newer refusal, read-only open
for the CLI), `src/audit/` (sink interface, table and file sinks), `src/backup/`, `src/cli/{usage,audit,backup,reconcile}.ts`.
Proof: `backup snapshot`, `backup verify --expect ok`, `backup rehearse --expect ok` on a temp
root; `verify` on a truncated copy exits 1; a test that `UPDATE audit_event` raises the trigger
text; `audit --limit 1` on an empty store prints nothing, exit 0; `usage` prints the total line.
Depends on: P3, P4. Must not touch: `src/session`.

## P6. Policy, the hook, and credentials (medium)

Implements: `03-hook-contract.md` (all); `03-architecture.md` (`policy`, `hook`, `credentials`
rows); ADR 0022 (Both hosts: the store; Amendments from the spike); `05-debug-interface.md`
(The PreToolUse hook; Credentials).
Files: `src/policy/` (the table as one object, the decision function, the deny text, the role
check, the model display-name map), `src/hook/` (`pre-tool-use|try|cases`; one handler exiting 2
with `hook: <what>`), `tests/hook/cases.jsonl` (every cell in both modes, a symlink out of the
tree, `.credentials.json`, the config directory through `transcript_path`), `src/credentials/`
(AES-256-GCM store, key file, `init|add|list|remove|show|login`; `add --kind subscription` calls
the backend's `harness-create`, stubbed until P7), `src/cli/{hook,credential}.ts`.
Only the amendments say: the config directory is the parent of the nearest `projects` ancestor
of `transcript_path`, exit 2 when absent; `timeout` 30 s; `CLAUDE_CONFIG_DIR` never reaches the hook.
Proof: 05's `printf ... | broodforge hook pre-tool-use` line prints the `defer` object, exit 0;
stdin `{}` exits 2 with `hook: stdin has no tool_name`; `hook cases tests/hook/cases.jsonl` all
`ok`; `printf scratch | broodforge --root <tmp> credential add --kind api-key --user 1001 --label
t` prints `cred-...` and `credential show <id> --json` holds no value. Depends on: P3, P5. Must
not touch: `src/session`, `src/spawn` beyond the interface.

## P7. Spawn (same-account), the real engine, recording (medium; real turns: the owner's)

Implements: `03-architecture.md` (`spawn` row; the `claude-code` implementation; interrupt); ADR
0020 (Mechanism, Interrupt, Settings sources; Amendments 2026-10-16: no `--permission-prompts
none`); ADR 0022 (Windows; Both hosts: delivery); `05-debug-interface.md` (`engine run --real`,
`engine record`; Spawn preflight, same-account half); `07-spike-plan.md` (`spike/` deleted here).
Files: `src/spawn/` (`SpawnBackend`; `same-account`: `child_process` with the explicit
environment, the four git variables, `CLAUDE_CODE_SUBPROCESS_ENV_SCRUB=1`, `DISABLE_AUTOUPDATER=1`,
the binary from config, the settings template written to the engine's settings path before each
process (`<cfg>/settings.json` for a subscription, `<sess>/home/.claude/settings.json` for a key;
C26's `--settings <file>` is the fallback if that proves wrong); interrupt as the spike did it
(Windows `Get-CimInstance` walk then `taskkill /T /F`; Linux SIGINT, SIGKILL after the timeout);
`preflight`; `harness-create|remove` as directory operations; `linux-sudo-bwrap` as a stub refusing
`error: preflight-backend: held`), `src/engine/{claude-code,record}.ts`, `src/cli/{engine,spawn}.ts`; `git rm -r spike/`.
Only the findings say: a `defer` resume takes no stdin line; the engine refuses a standalone
`sleep` in Bash before the hook (test a kill with `ping -n 31 127.0.0.1`); the path is the `.exe`.
Proof: `spawn preflight` against a config naming the installed engine; `engine run --fake`
unchanged; then, by the owner against his spike `$CFG`, one real turn each: `engine run --real
--credential <id> --cwd <tmp tree> --prompt "Read hello.txt and reply with its one word."`
streams a `result`; `engine record --out <tmp>/rec --prompt ...` writes a directory that `engine
run --fake <tmp>/rec` replays. Depends on: P2, P6. Must not touch: `src/session`; the owner's `~/.claude`.

## P8. Git: canonical, clone, bundle, attribution, handoff (medium)

Implements: `03-architecture.md` (Git: trees, branches, attribution, handoff); ADR 0022
(Working trees) and its companion (Clone at session start; Handoff at session end; the same
steps on Windows); `05-debug-interface.md` (Git handoff).
Files: `src/git/` (refs, clone through the backend's `worktree-create`, `core.hooksPath` to
`share/hooks/commit-msg`, bundle, verify, fetch, the attribution check, push with
`bin/broodforge-askpass`, the GitHub pull-request call behind an interface with `--dry-run`),
`src/spawn/same-account.ts` gains `worktree-create|bundle|remove`, `src/cli/worktree.ts`.
Proof: in a temp root with a bare remote: canonical, `worktree-create`, a commit in the tree
under the four variables (through `child_process`, never the Bash tool: `07-spike-plan.md`,
Fixture commits), `worktree check s0001` all `ok`, `worktree handoff s0001` pushes and `git -C
<remote> branch` lists `broodforge/<instance>/s0001`; a commit with `--author "X <x@example.com>"`
yields `handoff-refused-attribution`. Depends on: P5, P7. Must not touch: `src/session`.

## P9. The scratch root (small)

Implements: `05-cli-conventions.md` (Scratch); ADR 0021 (Scratch).
Files: `src/cli/scratch.ts`, `src/config/scratch-template.ts` (dry-run, `fake`, `fixtures/engine/`,
same-account, the key under the root, channels `100` and `200`, `cred-scratch`),
`src/store/scratch.ts` (creation; deletion through `worktree-remove`).
Proof: `scratch new --json` prints a root whose `config check` passes and whose `credential list`
shows `cred-scratch`; `scratch run -- broodforge config check` deletes the root on exit 0 and
keeps it on exit 1 printing the path; `scratch delete` of a non-scratch path is `error: not-scratch`.
Depends on: P4, P5, P6, P8.

## P10. Sessions, turns, modes, timeouts, usage rows (large; owner sees the trace wording)

Implements: `03-architecture.md` (`session` row; Registration and participants; Credential
status; Data flow for one turn; Prompt mode; Startup 1 to 8; Shutdown; Failure modes); ADR 0020
(Permission relay and modes; Usage and attribution; Amendments 2026-10-16); ADR 0021
(Reconciliation); `05-debug-interface.md` (Bot lifecycle; Registration; Session lifecycle
without Discord; Prompt-mode defer and resume; `config reload`).
Files: `src/session/` (`SessionService`, the state machine, the turn queue, the `From:` blocks,
the system prompt append with the trailer lines and the one-call-at-a-time ask, event mapping,
the `permission decision` rows with `dropped` and `decidedBy` `config` (the `hook bypassed`
audit event), the classifier that strips the engine's wrapper, usage rows, idle and prompt
timeouts with an injectable clock, the end-of-session sequence through P8), `src/cli/{serve,status,stop,register,session}.ts`
(`serve` steps 1 to 8, dry-run only), `tests/repro/full-session.sh`, `tests/repro/defer-resume.sh`.
Mid-pass commit after (a) lifecycle, registration, a policy-mode turn against `basic`; then (b)
prompt mode against `defer` and `parallel`, timeouts, interrupt against `interrupted`, usage.
Proof: `scratch run -- bash tests/repro/full-session.sh` exit 0; `defer-resume.sh` exit 0 with
`session show` printing `pending:` while waiting; a `parallel` turn leaves one `dropped` row in
`audit --kind permission-decision`; `usage --session s0001 --by turn --json` equals the fixture's
deltas (`basic` turn 2: input 2, output 4); `stop` marks the open session interrupted. Depends on: P2 to P9.

## P11. Discord dry-run: renderer, coalescer, outbox, transcript, panel (medium; owner sees)

Implements: `04-ux.md` (every screen and text); `03-architecture.md` (`discord` row; Data flow
steps 1 and 6); `05-debug-interface.md` (The Discord adapter: `render`, `inject`; `session outbox`, `transcript`, `panel`).
Files: `src/discord/` (renderer to the outbox sink, the coalescer with the injectable clock, the
mask, custom ids, command parsing, the role check through `policy`, `inject`; discord.js types
only, no gateway), `src/cli/{discord,panel}.ts`, `tests/discord/burst.jsonl`.
Proof: `discord render --events tests/discord/burst.jsonl --clock fake --expect 'edits<=1/s'`;
`discord inject message --channel 100 --user 1001 --roles 10 "sk-ant-..."` yields the refusal
and an `action-refused` event with no key text; a scratch session shows 04's working message,
footer, buttons post and summary in `session outbox`. The owner reads one run's outbox. Depends on: P10.

## P12. Discord live: gateway, commands, buttons, venues (medium; owner sees; needs his guild)

Implements: `04-ux.md` (Registration; The project channel and the panel; Venue; Starting);
`03-architecture.md` (Startup 9 and 10; Failure modes, `discord` row); `05-debug-interface.md`
(The Discord adapter: `check`, `serve --discord live`, the untestable list).
Files: `src/discord/gateway.ts` (discord.js client, slash-command registration, interactions,
the private-channel venue, the notice channel), `discord check` in `src/cli/discord.ts`.
Proof: `discord check --guild <id>` lists every configured channel `present`, `perms:ok`; `serve
--discord live` on a scratch-shaped root with the fake: the owner registers, starts, sends,
clicks Allow and ends from Discord while an agent reads `session tail` and `audit`. The vision's
first usable version is this plus P7's real engine. Depends on: P11; the bot token and a test guild.

## P13. The Windows instance (medium; owner steps)

Implements: ADR 0022 (Windows); `03-architecture.md` (Process model, Windows column); the two
operator entries `../operator/2026-10-14-pc-broodforge-account-and-task.md` and
`../operator/2026-10-14-credential-store-key-per-host.md`; C37 (`07-spike-plan.md`).
Files: `docs/reference/pc-host-setup.md` [no-check: written in this pass] (the Task Scheduler
action, the key file path, the engine install for the `broodforge` account); code only where
`spawn preflight` must report the account. The C37 run is this pass's finding.
Proof: C37's three actions from the scheduled task; `status` over the named pipe from the
owner's account. Depends on: P10, P12; the owner creating the account. Must not touch: the owner's profile.

## P14. The Linux backend `linux-sudo-bwrap` and the droplet (large; HELD)

Held until the droplet rows of both findings land (`07-spike-droplet.md`: C27, C29 to C33,
C38, C39's exit-2 repeat, the droplet halves of C34 to C36). Implements: ADR 0022 (Linux, both
paragraphs) and its companion in full; `03-architecture.md` (Process model, Linux column;
Signals and leftovers). Files: `src/spawn/linux-sudo-bwrap.ts`, the `engine-entry` shell half
of `bin/broodforge`, `share/empty/`, the `namei` and sudoers checks in `spawn preflight`, the
systemd unit, `docs/reference/droplet-host-setup.md` [no-check: written when the droplet
operator entries are done]. Proof: C30 and C31 by hand from a code session, `reach-report`
matching the bind list, the Interrupt steps yielding a `result`. Depends on: P7 to P10.
