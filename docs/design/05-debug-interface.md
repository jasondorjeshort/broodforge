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
(`scratch new|run|delete|prune`, channels `100` and `200`, credential `cred-scratch`), and the
shared isolation rules (no command prints a secret; `error: live-instance` for a debug flag
that widens a live session), the hidden probe `control echo`. Sections below give only what differs.

## Bot lifecycle

Drive. `broodforge serve [--discord live|dry-run]` runs the bot in the foreground through 03's
startup steps; a failed step is `error: start-<step name>: <reason>`, exit 1; the flag overrides
the config key for this run. `broodforge status [--wait <seconds>]` waits for the socket to
answer and prints, in order: `config: ok` or `config: last-good from <time>`, `discord:
live|dry-run connected|disconnected`, `sessions: <open> open, <waiting> waiting-for-answer,
<interrupted> interrupted`, `worktrees: <kept> kept with unpushed commits`, `backup: last good
<age> ago|none`, `engine: <path> <version>`; `--json` nests the same under those keys.
`broodforge stop [--timeout 30]` is 03's shutdown, exit 1 after the timeout with the open
sessions listed. `broodforge reconcile [--report-only]` runs ADR 0021's reconciliation on
demand; `--report-only` works on the file without the bot, writes nothing and prints `<kind>
<subject>  <session>  <detail>` per finding (`interrupt`, `missing`, `present`, `unrecorded`).
Observe. `status`; `audit --kind instance-started --limit 1` and the events after it (the report).
Reproduce. `broodforge scratch run -- broodforge serve --discord dry-run` with a second shell
running `status --wait 10 --expect config=ok`; a start refusal keeps the scratch root.
Isolation. `serve` refuses a root whose socket already answers (`error: start-control: another
bot owns <socket>`); scratch uses the same-account backend on both hosts.

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
`broodforge config reload` asks the running bot; on failure the old config stays and the errors
are the stderr lines, exit 1.
Observe. `status` line one; `audit --kind configuration-changed --limit 1 --json` shows the file
hash and changed keys; `config show --effective --channel 100 --json` returns the channel object
of 03 with `defaults` merged in (`{"id":"100","project":"scratch","reach":"code",...}`).
Reproduce. Last-good: in a scratch root start the bot, `config reload` once, write invalid JSON
over `config/instance.json`, `stop`, `serve`, `status --expect config=last-good`. A validation
bug is a tree beside `tests/config/valid/` and `tests/config/dup-key/`, or `config check --root
<kept scratch root>`.
Isolation. `config` reads the credential store for id existence only (03).

## Registration and participants

Drive. `broodforge register <discord user id> --name "<git name>" --email <email>`; again
replaces and writes `registration-changed` (R-02). `register list` and `register show <user id>`
print `<user id>  <name>  <email>  <registered at>  <default credential id or ->`; `register
set-default <user id> <credential id>`. Participants are read per session: `session show <id>`
lists `participants: <user id>  driver|contributor  <joined at>`. Roles without Discord: every
session command that acts for a person takes `--as <user id> [--roles <id,...>]`; given, the
channel's `start`, `contribute` or `answer` list (or the driver, for `mode` and `invite`) is
checked against them with `policy`'s role check (03), and a miss is `error: refused: <action>
needs a role in <list>`, exit 1, plus the `action refused` audit event; `--as` absent, the
action is the owner's from the host (actor `host`), unchecked.
Observe. `register list --json`; `audit --kind registration-changed`.
Reproduce. `scratch run -- bash tests/repro/<name>.sh` [no-check: no code yet]: the script
registers, starts with `--roles 11 --expect refused`, then with `--roles 10`.
Isolation. A scratch root has no registrations; the ids above are not Discord's.

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
id; it writes no settings, since the hook's settings file is per session. `credential login
<id>` prints the exact command the user runs to complete the engine's own login into that
directory (spike C25's, `CLAUDE_CONFIG_DIR=<dir> <engine.path> auth login`, sh quoting); `credential login <id> --done` records `usable` once the owner
has run it; `--relay` runs ADR 0022's relay, refused unless `relay.loginByDirectMessage` is
true, and records `usable` itself on success. `credential list [--user <id>]` prints `<id>
<kind>  <user id>  <label>  usable|needs-login|removed  <last used or ->`; the status is the
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

Drive. `broodforge session start --channel <id> --as <user id> [--roles <ids>] [--credential
<id>] [--mode prompt|policy] [--reach read-only|code] [--fixture <dir>] [--idle-timeout-seconds
<n>] [--prompt-timeout-seconds <n>]` prints the session id (`s0001`); `--reach`, `--fixture`,
`--idle-timeout-seconds` and `--prompt-timeout-seconds` are scratch and dry-run only. `session send <id> --as <user id> [--roles <ids>] [--wait] [--tail]
"<text>"` (or `-` for stdin) calls `SessionService.submit` and prints `turn <n> queued|started`;
`--wait` returns at `turn-ended` or `waiting-for-answer`, exit 0 either way, printing which;
`--tail` streams the events meanwhile. `session mode <id> prompt|policy`, `session interrupt
<id>`, `session end <id>` (the end-of-session sequence; prints the outcome line; it runs the closing
turn against the fake engine when a fixture named `closing` exists, `--no-closing-turn` skips
it) and `session
invite <id> --user <user id>` take `--as` and `--roles` as above (`mode` and `invite` need the
driver, `interrupt` and `end` the `answer` list). `session list [--state <state>]` prints `<id>
<state>  <channel>  <driver>  <mode>  <reach>  <turns>  <started at>`; `session show <id>`
prints participants, worktree, credential id, engine session id, pending answer; `session tail
<id> [--follow] [--since <seq>] [--all]` prints `<seq>  <time>  <type>  <summary>` where the
summary is the text's first 80 characters, the tool name and target, or the stop reason.
`session outbox <id> [--follow]` prints the dry-run renderer's posts: `<seq>  <time>
create|edit|attach  <message id>  <length>  <first line>`, `--json` with the full body. `session
transcript <id> [--out <file>]` prints 04's Markdown transcript as the summary would attach it.
`broodforge panel post <channel id>` posts or refreshes the project channel's panel (04) and
records its message id in the store, which `panel show <channel id>` prints back; on a
dry-run instance it lands in the outbox. `session reach-report <id>` is below, Spawn and reach.
Observe. `session show`, `session tail --json`, `session outbox`, `usage --session <id>`.
Reproduce. A whole R-51 run in one line: `scratch run -- bash tests/repro/full-session.sh`
[no-check: no code yet]: `serve --discord dry-run &`, `status --wait 10`, `register 1001 ...`,
`session start --channel 100 --as 1001 --roles 10`, `session send s0001 --as 1001 --wait "..."`,
`session answer ...`, `session outbox s0001`, `session end s0001`, `stop`.
Isolation. All of it in the scratch root against the fake; the socket under the root.

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
./internal/tools/importfixture`, P1, generalised when `engine record` lands) replaces the recording host's home with `/home/user`,
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

Drive. On a prompt-mode session with the fixture `fixtures/engine/defer/` [no-check: no code
yet] (turn 1 ends `stop_reason: tool_deferred`, turn 2 is the resumed turn): `session send s0001
--as 1001 --wait "edit a.ts"` prints `waiting-for-answer toolu_01 Write a.ts`; `session answer
s0001 toolu_01 allow|deny [--as <user id>] [--roles <ids>]` calls `SessionService.answer` and the
resumed turn's events follow in `session tail`. The idle timeout is driven with
`--idle-timeout-seconds 2` at start and observed as `session-ended` with reason `idle`; `--prompt-timeout-seconds 2` with no answer
shows the timeout trace line (`denied: the prompt timed out`) in `session outbox`.
Observe. `session show` prints `pending: toolu_01 Write a.ts since <time> until <deadline>`; `received-2.json`
carries `answer` and `engineSessionId`; `session outbox` has the buttons post, custom id
`bf:s0001:toolu_01:allow`.
Reproduce. `scratch run -- bash tests/repro/defer-resume.sh` [no-check: no code yet]; an answer
refused for roles is `session answer ... --roles 11 --expect refused`.
Isolation. Against the real engine this is spike C27 (`engine run --real --resume`), not scratch.

## Spawn preflight and reach

Drive. `broodforge spawn preflight [--expect ok] (file)` runs the backend's checks without a session: `ok  <check>  <detail>` per
passing check and `error: preflight-<check>: <detail>` per failing one on stderr, exit 1. Same-account: `backend`; `engine-path`
(absolute, a file, on Windows the `.exe`); `engine-version` (`--version` against an empty config directory, equal to
`engine.version`); `git-bash` (Windows; unset, the engine finds it); `share` (`<bf>/share/empty/mcp.json` a map, `CLAUDE.md`
empty, `dir/` empty, made when missing); `account`. `linux-sudo-bwrap` is `error: preflight-backend: held` until P14, which adds
the `namei` walk, the `engine-entry --check` spawn and the sudoers comparison of the ADR 0022 companion, printing the expected
sudoers block on a miss. `session reach-report <id>` prints the contract's table column for the session's reach beside the bind
list the spawn would use, computed from the same array, then `--json` for both. `session start --reach read-only` on channel `200`
of a scratch root makes a read-only session against the fake.
Observe. `reach-report`; on the droplet, C30's commands run by hand from a code session's Bash (`ls`, `env`, `cat` of the root,
another session and the canonical) and the hook's `deny` lines in `session tail` for an `Edit` under read-only reach.
Reproduce. `hook cases` for the policy half; for the bind half, C30 and C31 of ADR 0022 are the test, run once per host setup and
recorded as a finding; the fake exercises no bind. The Windows job: `go test ./internal/spawn -run 'Job|Fallback' -v` closes an
engine's job (and, with assignment refused, takes the walk and `taskkill /T /F`) and shows its `PING.EXE` grandchild gone.
Isolation. Preflight runs only the engine's `--version` (Linux: `engine-entry --check`); scratch has an empty bind list.

## Git handoff

Drive. `broodforge worktree list [--kept]` prints `<session id>  <path>  active|pushed|kept|
removed  <unpushed commits>  <missing since or ->`. `worktree check <id>` runs the attribution
check alone: one line per commit `<hash>  ok|author:<got>|missing:<trailer>`, exit 1 if any
fails. `worktree bundle <id>` runs the bundle step, prints the bundle path and commit count
(`none` for no commits). `worktree handoff <id> [--dry-run]` runs the companion's five steps;
`--dry-run` stops after the check and prints what `push` or `pull-request` would send (remote,
refspec, request body with the credential id in place of the value). Refusals are stable codes
on stderr, exit 1: `handoff-refused-bundle-verify`, `handoff-refused-fetch`,
`handoff-refused-attribution` (failing lines follow), `handoff-failed-push` (git stderr's first
line as the message), `handoff-failed-pull-request` (the HTTP status). `worktree forget|adopt`
are ADR 0021's.
Observe. `worktree list`; `audit --session <id> --kind push,pull-request-opened,handoff-refused`;
`git -C <root>/projects/scratch for-each-ref refs/sessions/`; `git -C <root>/remote/scratch.git
branch` for the pushed branch in scratch.
Reproduce. In a scratch root, commit in the session tree with a wrong author (`git -C
<root>/sessions/s0001/tree commit --author "X <x@example.com>" -m t` after a change) and run
`worktree handoff s0001 --expect handoff-refused-attribution`. A pull request cannot complete
offline: `--dry-run` is the reproduction, the forge call the one step left.
Isolation. The scratch remote is a local bare repository with `handoff` `push`; the bot's
git never opens the session tree (ADR 0022); `worktree check` reads `refs/sessions/<id>`.

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
rehearse [path] (file)` are the ADR 0021 companion's steps and exit codes. `snapshot` prints `ok
<snapshot path>  <audit events exported>` (no checkpoint from outside the bot; a second run at
once is `backup-busy`, the lock `backup/snapshot.lock`); `verify`, of
`latest` by default, prints `ok`, `migration  <id>`, then `<table>  <rows>` per table; a failed
check is `error: integrity-check: <result>`; `rehearse` prints the same from the restored copy plus
`reconcile: <n> missing` and, on failure, the scratch root it kept.
Observe. `status` line `backup:`; `ls <root>/backup/`; `backup verify --json`.
Reproduce. `scratch run -- bash tests/repro/backup-cycle.sh` [no-check: no code yet]: a session,
`backup snapshot`, `backup verify --expect ok`, `backup rehearse --expect ok`; a corruption case
is `backup verify --path <truncated copy>`, exit 1.
Isolation. A rehearsal is itself a scratch root; the nightly job is never started by the bot.

## The Discord adapter

Moved to `05-debug-interface-discord.md`: `discord check|render|inject`, the dry-run renderer and what only a live guild exercises.

## Command tree

```
broodforge [--root <path>] [--json]
  serve  status  stop  reconcile                     scratch new|run|delete|prune
  config check|show|reload                           register <id>|list|show|set-default
  credential init|add|login|list|remove|show         session start|send|answer|mode|interrupt|end|invite
  session list|show|tail|outbox|transcript|reach-report   engine run|record
  hook pre-tool-use|try|cases                        worktree list|check|bundle|handoff|forget|adopt
  usage  audit  panel post|show                        backup snapshot|verify|rehearse
  discord check|render|inject                        spawn preflight   engine-entry (internal)
```
