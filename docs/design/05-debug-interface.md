# Debug interface

Date: 2026-10-14. Status: first draft, written from `01-requirements.md` (R-50 to R-52, N-05 to
N-11), `03-architecture.md` (with `03-hook-contract.md`) and ADRs 0020 to 0022. It fixes the
command names, output shapes and control-socket wire format those documents left to it; a working
name in an ADR yields to the name here. Wording people see in Discord is `04-ux.md`'s. Each
feature has the skill's five parts (Surface, Drive, Observe, Reproduce, Isolation); shared
conventions first.

## Conventions shared by every command

Every command follows `05-cli-conventions.md`: one binary with `--root` and `--json`, the
control-socket wire format (one JSON object per line, `cmd` as `<group>.<command>`), the text
and JSONL output rules, the N-07 exit codes and what `--expect` compares, the scratch root
(`scratch new|run|delete|prune`, channels `100` and `200`, credential `cred-scratch`; a kept
root is named on stderr as `scratch root kept: <path>`), and the
shared isolation rules (no command prints a secret; `error: live-instance` for a debug flag
that widens a live session), the hidden probe `control echo`. Sections below give only what differs.

## Bot lifecycle

Drive. `broodforge serve [--discord live|dry-run]` runs the bot in the foreground through 03's
startup steps and prints `serving  <socket>` once the socket listens; a failed step is `error:
start-<step>: <reason>`, exit 1 (`root`, `config`, `discord`, `credentials`, `store`, `engine`,
`preflight`, `reconcile`, `control`); the flag overrides the config key for this run. Live needs
`discord.applicationId`, `guildId` and `botCredentialId` (a `bot-token` credential whose value
has a bot token's shape, checked before anything connects) and opens the gateway after the
socket (`05-debug-interface-discord.md`); any of it failing is `start-discord`. With only `fake`
channels, steps 5 and 6 skip the engine checks. `broodforge status [--wait <seconds>] [--expect ok|last-good]` waits for the
socket to answer and prints, in order: `config: ok` or `config: last-good from <time>`, `discord:
live|dry-run connected|disconnected` (dry-run is always connected), `sessions: <open> open,
<waiting> waiting-for-answer, <interrupted> interrupted`, `worktrees: <kept> kept with unpushed
commits`, `backup: last good <age> ago|none` (age `45s`, `12m`, `3h`, `2d`), `engine: <path>
<version>` (`-` when unset); `--json` nests the same under those keys; the outcome token is the
config value. `broodforge stop [--timeout 30]` is 03's shutdown: every running turn interrupted,
every open session marked interrupted, `stopped  <sessions marked>`, then the bot exits 0; past
the timeout `error: stop-timeout: open sessions: <ids>` and the bot exits 1. A session waiting for an
answer is open too, and its prompt and idle timers stop with the bot. `broodforge
reconcile [--report-only]` runs ADR 0021's reconciliation on demand; `--report-only` works on the
file without the bot, writes nothing and prints `<kind>  <subject>  <session>  <detail>` per
finding (`interrupt`, `missing`, `present`, `unrecorded`). A read the bot does not serve itself
(`audit`, `usage`, `session list`, `register list|show`) reads the store file while it runs.
Observe. `status`; `audit --kind instance-started --limit 1` and the events after it (the report).
Reproduce. `broodforge scratch run -- broodforge serve --discord dry-run` with a second shell
running `status --wait 10 --expect ok`; a start refusal keeps the scratch root.
Isolation. `serve` refuses a root whose socket already answers (`error: start-control: another
bot owns <socket>`; any failure but `not-running` refuses the same way), probed right after step 2,
before any write, so a second bot never touches the first's sessions or audit log; scratch uses the same-account backend on both hosts.

## Configuration

Drive. `broodforge config check (file)` validates `config/` without the bot: `ok  <projects>
<channels>`, exit 0, then warnings prefixed `warning:`; on any problem one `error: bad-config:
<file>: <what>` line each, exit 1 (a channel id in two project files is one, R-07; an unknown or
repeated key names its dotted path, `duplicate key channels.100`); paths print with `/` on both
hosts; no `config/instance.json` is `no-config`. `broodforge config show [--effective] [--channel
<id>] [--project <id>] [--last-good] (file)` prints the parsed files, one `<file>  <content>`
record each, comments removed; `--effective` prints each channel with
`defaults` merged in as the bot sees it; `--channel` and `--project` narrow either
(`unknown-channel`, `unknown-project`); `--last-good` reads `data/config-last-good/`.
`broodforge config reload [--expect reloaded]` asks the running bot to read `config/` again: valid,
it runs on it from then on (a running session keeps the reach and mode it started with), rewrites
the last-good copy, writes `configuration-changed` and prints `reloaded  <changed keys>`; nothing
changed prints `reloaded  -` and writes nothing. Refused, the old configuration stays and
`configuration-invalid` is written (`"using":"previous"`): invalid files are one `error:
bad-config: <file>: <what>; ...` line; another `instanceName` is `bad-config` too; a change to a
key the bot reads only at its start (`schemaVersion`, `credentials`, `discord`, `engine`,
`spawn`, `paths`, `audit`) is `reload-needs-restart: <keys>`; exit 1. Every socket command dials
`config/`'s `instanceName` and, with nothing there, the last-good copy's, so a bot is reached
after an edit of the name; `status` then adds `instance-name: <config's> (bot runs as <its>)`
after the config line.
Observe. `status` line one; `audit --kind configuration-changed --limit 1 --json` shows the file
hash and changed keys; `config show --effective --channel 100 --json` returns the channel object
of 03 with `defaults` merged in (`{"id":"100","project":"scratch","reach":"code",...}`).
Reproduce. Last-good: in a scratch root start the bot, `config reload` once, write invalid JSON
over `config/instance.json`, `stop`, `serve`, `status --expect last-good`. A validation
bug is a tree beside `tests/config/valid/` and `tests/config/dup-key/`, or `config check --root
<kept scratch root>`.
Isolation. `config` reads the credential store for id existence only (03).

## Registration and participants

Moved to `05-debug-interface-session.md`: `register <user id>|list|show|set-default`, participants, and the `--as` and `--roles` every session command takes.

## Credentials

Drive. `broodforge credential init (file)` writes the store key file the instance config names
and prints `<key hex>  <key file>` once, at init, so the owner can put it in a password manager (a
restore from an empty host needs it; ADR 0022); never again: `credential show` and every other
command print the path only. An existing key file is `error: key-exists: <path>`, exit 1, and
nothing is written. `credential add --kind api-key --user <user id> --label <text>` reads the
value from stdin (never an argument) and prints the new id `cred-<base36>`; the user must be
registered (`unknown-user`). `credential add
--kind subscription --user <user id> --label <text>` runs `harness-create` for `<root>/harness/<id>`
through the spawn backend (ADR 0022 companion; same-account: a directory operation), records the credential as `needs-login` and prints the
id; it writes no settings, since the hook's settings file is per session. `credential add
--instance --kind bot-token --label <text>` records one of the instance's own credentials, the
Discord bot token (provider `discord`, no user, `user_id` null): `--instance` is exclusive with
`--user` (a usage error), the value is read from stdin, and `bot-token` with `--user` is
`bad-kind`; it is the kind `discord.botCredentialId` names, and only the gateway and `discord
check` read its value. `credential login
<id>` prints the exact command the user runs to complete the engine's own login into that
directory (spike C25's, `CLAUDE_CONFIG_DIR=<dir> <engine.path> auth login`, sh quoting); `credential login <id> --done` records `usable` once the owner
has run it; a `bot-token` has no login (`bad-kind`); `--relay`, ADR 0022's relay, is not built
and is refused `not-built` (the plan: refused unless `relay.loginByDirectMessage` is true, and
`usable` recorded on success). `credential list [--user <id>]` prints `<id>
<kind>  <user id>  <label>  usable|needs-login|removed  <last used or ->`, the user `-` for an
instance credential; the status is the
store's record (03): the bot never reads the config directory to infer it. `credential remove
<id>` marks it removed; the next `session start` naming it is refused (R-41). `credential show
<id> --json` prints the record without any store value.
Observe. `credential list`; `audit --kind credential-added,credential-removed`.
Reproduce. `printf scratch | broodforge --root <r> credential add --kind api-key --user 1001
--label t`; a delivery bug is read from the fake's `received-<n>.json`, whose `delivery` field
carries the kind and the id (03), never the value.
Isolation. The scratch store key lives under the scratch root; the fake accepts only the fixed
string and refuses any other value with exit 1.

## Session lifecycle without Discord

Moved to `05-debug-interface-session.md`: `session start|send|mode|interrupt|end|invite|list|show|tail`, the event types and summaries `session tail` prints, who decided a call, `tests/repro/full-session.sh`, and P11's `session outbox|transcript` and `panel post|show`.

## A turn against the fake engine

Drive. Through a session: `session send` on a scratch root, where the channel's harness is `fake` and `--fixture <dir>` picks the
directory (default `fixtures/engine/basic/` [no-check: no code yet]). Adapter alone, no store, no session: `broodforge engine run
--fake <fixture dir> [--turn <n>] [--cwd <dir>] --prompt "<text>"` streams `EngineEvent`s as JSONL to stdout exactly as the
adapter parsed them and exits 0 when the fixture's `exit` is 0, else 1 with `error: engine-exit`. `engine run --real --credential
<id> --cwd <dir> --prompt "<text>" [--resume <engine session id>] [--model <m>]`, with `--root`, does the same through the configured spawn
backend: credential and git identity from the store, an API key through the key file; the hook sees session `engine-run`, code
reach, `defaults.mode`, log `<root>/sessions/engine-run/home/hook.log`; Ctrl-C interrupts the turn; no `--credential` is `error:
usage`, an unknown one `error: unknown-credential`, nothing spawned. `engine record --credential <id> --cwd <dir> --out <fixture
dir> --prompt "<text>" [--model <m>]` runs the real engine and writes the raw lines as `turn-<n>.jsonl` (`n` counting up in an existing
directory, `--resume` the previous turn's, from `fixture.json` or its `init`) plus a `fixture.json` without delays (claim
`recorded`, `engine.version`); an `--out` under `--root` or `--cwd` is `error: out-in-session`. The recorder redacts
nothing, so record with a credential made for recording; before a commit the import rewrite (`go run
./internal/tools/importfixture [--user <pc user>] <out> fixtures/engine/<name>`, P1) replaces the recording host's home with `/home/user`,
then an agent that did not record reads every file with nested JSON decoded, confirms no name, email, hostname, real host path or
repository content survives, and reports in a few lines what is account-level or personal and no script can classify (the owner
decides on that report, never by reading the files).
Observe. `session tail --json` for the mapped events; the fixture's `received-<n>.json` for the exact `TurnRequest` the fake saw
(delivery kind, hook variable names, `systemPromptAppend`, `answer`); `engine run --fake <dir> --prompt x | grep
'"type":"result"'` for the usage input.
Reproduce. A failing parse is a fixture: copy the offending `turn-<n>.jsonl` into a new directory and run `engine run --fake <dir>
--prompt x --expect result` (no `fixture.json`: every turn exits 0); a stream without a `result` exits 1, `error: expect: wanted
result, got no-result`.
Isolation. The fake reads the fixture directory and writes only `received-<n>.json` there (`--received-dir <dir>` for a read-only
tree); `engine record` needs a real credential, not scratch; the tests drive `--real` and `record` against
`internal/spawn/spawntest`, the test binary standing in for the engine.

## The PreToolUse hook

Drive. Standalone, exactly as the engine runs it: the engine's JSON on stdin and every
`BROODFORGE_*` variable of the contract (`03-hook-contract.md`) in the environment (Git Bash
on Windows: prefix `MSYS_NO_PATHCONV=1`, or it rewrites the variables' `/tmp` paths but not the JSON's):
`printf '{"tool_name":"Write","tool_input":{"file_path":"/tmp/x/a.ts","content":""},
"tool_use_id":"toolu_01","cwd":"/tmp/x","transcript_path":"/tmp/cfg/projects/x/s.jsonl"}' |
BROODFORGE_SESSION=s0001 BROODFORGE_MODE=prompt
BROODFORGE_REACH=code BROODFORGE_TREE=/tmp/x BROODFORGE_CANONICAL=/tmp/c
BROODFORGE_HOOK_LOG=/dev/null broodforge hook pre-tool-use` prints `{"hookSpecificOutput":
{"hookEventName":"PreToolUse","permissionDecision":"defer","permissionDecisionReason":
"<reason>"}}`, exit 0. Any error, a missing variable, malformed stdin or an unwritable log
included, is exit 2 with `hook: <what>` on stderr, the one exit the engine reads as a block.
The same code path with flags, nothing logged: `broodforge hook try --tool <name> [--input
'<json>'] [--tool-use-id <id>] [--cwd <path>] [--mode prompt|policy] [--reach read-only|code]
[--tree <path>] [--canonical <path>] --transcript-path <path> [--answer <id>:allow|deny]
[--answer-reason <text>] [--session-allow <names>] [--expect allow|deny|defer|hook]` prints
`<decision>  <reason>` (defaults: `{}`, `toolu_01`, the tree, `prompt`, `code`, `.`, the tree); a
hook error is `error: hook: <what>`, exit 1; `--transcript-path` is what the hook derives the
config directory from (`03-hook-contract.md`, Paths), so a case can read inside it.
Observe. The decision line; `BROODFORGE_HOOK_LOG` (`sessions/<id>/home/hook.log`), one line per
decision; in a session, `audit --kind permission-decision --session <id>`.
Reproduce. One `hook try` line with `--expect`, or a case file: `broodforge hook cases <file>`
runs a JSONL of `{"tool","input","toolUseId","cwd","mode","reach","tree","canonical","transcript",
"answer","answerReason","sessionAllow","expect","needs","note"}` objects (try's defaults; `expect`
also `error`; `needs` a path whose absence skips the case; relative paths but `input`'s from the
file's directory; `{dir}` is that directory, `{posixdir}` its `/c/...` form), prints `ok|FAIL|skip  <line>  <tool>  <mode>  <reach>  <got>  <want>  <note>`,
exit 1 `cases-failed` on a FAIL; `tests/hook/cases.jsonl` (over `tests/hook/linkworld/`) covers every cell, links, the refused spellings and 8.3 names.
Isolation. Pure: no socket, no store, no root; `--tree` is any directory; only the log is written.

## Prompt-mode defer and resume

Moved to `05-debug-interface-session.md`: `session answer`, deferred and dropped calls, the idle and prompt timeouts, the pending answer in `session show`, and `tests/repro/defer-resume.sh`.

## Spawn preflight and reach

Drive. `broodforge spawn preflight [--expect ok] (file)` runs the backend's checks without a session: `ok  <check>  <detail>` per
passing check and `error: preflight-<check>: <detail>` per failing one on stderr, exit 1. Same-account: `backend`; `engine-path`
(absolute, a file, on Windows the `.exe`); `engine-version` (`--version` against an empty config directory, equal to
`engine.version`); `git-bash` (Windows; unset, the engine finds it); `share` (`<bf>/share/empty/mcp.json` a map, `CLAUDE.md`
empty, `dir/` empty, made when missing); `account`. `linux-sudo-bwrap` is `error: preflight-backend: held` until P14, which adds
the `namei` walk, the `engine-entry --check` spawn and the sudoers comparison of the ADR 0022 companion, printing the expected
sudoers block on a miss. `session reach-report <id>` prints the contract's table column for the session's reach beside the bind
list the spawn would use (`binds: -` until P14), computed from the same array, then `--json` for both (its lines: `05-debug-interface-session.md`). `session start --reach read-only` on channel `200`
of a scratch root makes a read-only session against the fake.
Observe. `reach-report`; on the droplet, C30's commands run by hand from a code session's Bash (`ls`, `env`, `cat` of the root,
another session and the canonical) and the hook's `deny` lines in `session tail` for an `Edit` under read-only reach.
Reproduce. `hook cases` for the policy half; for the bind half, C30 and C31 of ADR 0022 are the test, run once per host setup and
recorded as a finding; the fake exercises no bind. The Windows job: `go test ./internal/spawn -run 'Job|Fallback' -v` closes an
engine's job (and, with assignment refused, takes the walk and `taskkill /T /F`) and shows its `PING.EXE` grandchild gone.
Isolation. Preflight runs only the engine's `--version` (Linux: `engine-entry --check`); scratch has an empty bind list.

## Git handoff

Moved to `05-debug-interface-git.md`: `worktree check|bundle|handoff`, the refusal codes, the hidden `git-askpass` and `git-commit-msg`, and the rules the code fixes.

## Usage and audit queries

Drive. `broodforge usage [--session <id> | --user <id> | --credential <id> | --channel <id> |
--project <id>] [--since <time>] [--by model|turn]` prints one line per model `<model>  <turns>
<input>  <output>  <cache read>  <cache write>  <interrupted turns>`, then the same columns as
`total`, printed even with no usage; no filter is the instance total; `--user` is the requester
total, `--credential` the payer total (02); `--by turn` prints `<session>  <turn>  <model>
<input>  <output>  <cache read>  <cache write>  <interrupted>  <time>`. `broodforge audit
[--since-seq <n>] [--kind <kind>[,<kind>]] [--session <id>] [--limit 20] [--follow]` prints
`<seq>  <time>  <kind>  <phase or ->  <actor>  <session or ->  <details, first 80 chars>` in seq
order: the newest `--limit` events, or with `--since-seq` the first after it; an empty log
prints nothing, exit 0; `--json` prints every column. Kinds are 03's audit event kinds in
kebab-case. `--follow` needs the bot (`not-running` without one).
Observe. `usage --by turn --json` carries each turn's raw `modelUsage` and `usage` as `raw`.
Reproduce. A total that disagrees with a fixture: `usage --session s0001 --by turn --json` beside
`grep '"type":"result"' <fixture dir>/turn-<n>.jsonl`.
Isolation. Read-only; without the bot both read the store file (a direct reader needs write
access to `-shm`, ADR 0021; the CLI says `error: store-shm: <path>` when it lacks it);
`no-store` without `data/store.sqlite`, `store-schema` when its migrations are not the binary's.

## Backup snapshot and restore rehearsal

Drive. `broodforge backup snapshot (file)`, `backup verify [--path <file>] (file)`, `backup
rehearse [--path <file>] [--keep] (file)` are the ADR 0021 companion's steps and exit codes. `snapshot` prints `ok
<snapshot path>  <audit events exported>` (no checkpoint from outside the bot; a second run at
once is `backup-busy`, the lock `backup/snapshot.lock`); `verify`, of
`latest` by default, prints `ok`, `migration  <id>`, then `<table>  <rows>` per table; a failed
check is `error: integrity-check: <result>`; `rehearse` restores into a new scratch root (config
copied with its `instanceName` and `keyFile` made the scratch root's), prints the same from the
restored copy plus `reconcile  <n> missing` (recorded trees checked at their place under the
scratch root, so each counts) and `root  <path>` with `--keep`, deletes the root on
success and, on failure, keeps it and names it on stderr.
Observe. `status` line `backup:`; `ls <root>/backup/`; `backup verify --json`.
Reproduce. `scratch run -- bash tests/repro/backup-cycle.sh` [no-check: no code yet]: a session,
`backup snapshot`, `backup verify --expect ok`, `backup rehearse --expect ok`; a corruption case
is `backup verify --path <truncated copy>`, exit 1.
Isolation. A rehearsal is itself a scratch root; the nightly job is never started by the bot.

## The Discord adapter

Moved to `05-debug-interface-discord.md`: `discord check|render|inject`, the dry-run renderer, its outbox and pacing, `panel post|show`, and what only a live guild exercises.

## Command tree

```
broodforge [--root <path>] [--json]
  serve  status  stop  reconcile                     scratch new|run|delete|prune
  config check|show|reload                           register <id>|list|show|set-default
  credential init|add|login|list|remove|show         session start|send|answer|mode|model|effort|interrupt|end|invite
  session list|show|tail|outbox|transcript|reach-report   engine run|record
  hook pre-tool-use|try|cases                        worktree list|check|bundle|handoff|forget|adopt
  usage  audit  panel post|show                        backup snapshot|verify|rehearse
  discord check|render|inject                        spawn preflight   engine-entry (internal)
```
