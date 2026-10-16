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
that would widen a live session). A section below writes out only what differs.

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
demand; `--report-only` works on the file without the bot.
Observe. `status`; `audit --kind instance-started --limit 1` and the events after it (the report).
Reproduce. `broodforge scratch run -- broodforge serve --discord dry-run` with a second shell
running `status --wait 10 --expect config=ok`; a start refusal keeps the scratch root.
Isolation. `serve` refuses a root whose socket already answers (`error: start-control: another
bot owns <socket>`); scratch uses the same-account backend on both hosts.

## Configuration

Drive. `broodforge config check (file)` validates `config/` without the bot, one error per line,
exit 1 on any (a channel id in two project files is one, R-07); warnings go to stdout prefixed
`warning:` with exit 0. `broodforge config show [--effective] [--channel <id>] [--project <id>]
[--last-good] (file)` prints the parsed files; `--effective` merges `defaults` into each channel
as the bot sees it; `--last-good` reads `data/config-last-good/`. `broodforge config reload`
asks the running bot; on failure the old config stays and the errors are the stderr lines, exit 1.
Observe. `status` line one; `audit --kind configuration-changed --limit 1 --json` shows the file
hash and changed keys; `config show --effective --channel 100 --json` returns the channel object
of 03 with `defaults` merged in (`{"id":"100","project":"scratch","reach":"code",...}`).
Reproduce. Last-good: in a scratch root start the bot, `config reload` once, write invalid JSON
over `config/instance.json`, `stop`, `serve`, `status --expect config=last-good`. A validation
bug is `config check --root <kept scratch root>`.
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
and prints the key once on stdout, at init, so the owner can put it in a password manager (a
restore from an empty host needs it; ADR 0022); never again: `credential show` and every other
command print the path only. An existing key file is `error: key-exists: <path>`, exit 1, and
nothing is written. `credential add --kind api-key --user <user id> --label <text>` reads the
value from stdin (never an argument) and prints the new id `cred-<base36>`. `credential add
--kind subscription --user <user id> --label <text>` runs `engine-entry harness-create` through
the spawn backend (ADR 0022 companion), records the credential as `needs-login` and prints the
id; it writes no settings, since the hook's settings file is per session. `credential login
<id>` prints the exact command the user runs to complete the engine's own login into that
directory (spike C25's text); `credential login <id> --done` records `usable` once the owner
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

Drive. Through a session: `session send` on a scratch root, where the channel's harness is `fake`
and `--fixture <dir>` picks the directory (default `fixtures/engine/basic/` [no-check: no code
yet]). Adapter alone, no store, no session: `broodforge engine run --fake <fixture dir> [--turn
<n>] [--cwd <dir>] --prompt "<text>"` streams `EngineEvent`s as JSONL to stdout exactly as the
adapter parsed them and exits with the fixture's `exit` code; `engine run --real --credential
<id> --cwd <dir> --prompt "<text>" [--resume <engine session id>]` does the same through the
configured spawn backend; it is the production form of the spike's throwaway runner, which comes first (`07-spike-plan.md`). `engine record --credential
<id> --cwd <dir> --out <fixture dir> --prompt "<text>"` runs the real engine and writes the raw
lines as `turn-<n>.jsonl` (`n` counting up in an existing directory, `--resume` taken from the
previous turn's `init`) plus a `fixture.json` without delays. The recorder redacts nothing: a
transcript carries whatever the agent read, so record with a credential made for recording and
before committing it, have an agent that did not record it read every file with nested JSON
decoded, confirm no name, email, hostname, path outside the placeholder or repository content
survives, and report in a few lines what is account-level or personal and cannot be classified by
a script (the owner decides on that report, never by reading the files); the recorder refuses an `--out` inside a session tree.
Observe. `session tail --json` for the mapped events; the fixture's `received-<n>.json` for the
exact `TurnRequest` the fake saw (delivery kind, hook variable names, `systemPromptAppend`,
`answer`); `engine run --fake <dir> --prompt x | grep '"type":"result"'` for the usage input.
Reproduce. A failing parse is a fixture: copy the offending `turn-<n>.jsonl` into a new directory
and run `engine run --fake <dir> --prompt x --expect result`; exit 1 names the rejected line.
Isolation. The fake reads the fixture directory and writes only `received-<n>.json` there
(`--received-dir <dir>` for a read-only tree); `engine record` needs a real credential, not scratch.

## The PreToolUse hook

Drive. Standalone, exactly as the engine runs it: the engine's JSON on stdin and every
`BROODFORGE_*` variable of the contract (`03-hook-contract.md`) in the environment:
`printf '{"tool_name":"Write","tool_input":{"file_path":"/tmp/x/a.ts","content":""},
"tool_use_id":"toolu_01","cwd":"/tmp/x","transcript_path":"/tmp/cfg/projects/x/s.jsonl"}' |
BROODFORGE_SESSION=s0001 BROODFORGE_MODE=prompt
BROODFORGE_REACH=code BROODFORGE_TREE=/tmp/x BROODFORGE_CANONICAL=/tmp/c
BROODFORGE_HOOK_LOG=/dev/null broodforge hook pre-tool-use` prints `{"hookSpecificOutput":
{"hookEventName":"PreToolUse","permissionDecision":"defer","permissionDecisionReason":
"<reason>"}}`, exit 0. Any error, a missing variable or malformed stdin included, is exit 2
with `hook: <what>` on stderr, the one exit the engine reads as a block. The same code path
with flags: `broodforge hook try --tool <name> --input '<json>' --mode prompt|policy --reach
read-only|code --tree <path> --canonical <path> --transcript-path <path> [--answer
<tool_use_id>:allow|deny] [--session-allow <names>] [--expect allow|deny|defer]` prints
`<decision>  <reason>`; `--transcript-path` is what the hook derives the config directory from
(`03-hook-contract.md`, Paths), so a case can read inside it.
Observe. The decision line; `BROODFORGE_HOOK_LOG` (`sessions/<id>/home/hook.log`), one line per
decision; in a session, `audit --kind permission-decision --session <id>`.
Reproduce. One `hook try` line with `--expect`, or a case file: `broodforge hook cases <file>`
runs a JSONL of `{"tool","input","mode","reach","tree","canonical","transcript","answer","expect"}` objects,
prints one `ok|FAIL` line each, exit 1 on any FAIL; the shipped set `tests/hook/cases.jsonl`
[no-check: no code yet] covers every cell of the contract's table in both modes, a symlink out
of the tree, and a read of `.credentials.json` and of the config directory.
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

Drive. `broodforge spawn preflight (file)` runs the backend's checks without a session: on
Linux the `namei` walk, the `engine-entry --check` spawn and the sudoers comparison of the ADR
0022 companion, printing the expected sudoers block on a miss; on Windows the engine path, Git
Bash path and account; `error: preflight-<check>`, exit 1. `session reach-report <id>` prints
the contract's table column for the session's reach beside the bind list the spawn would use,
computed from the same array, then `--json` for both. `session start --reach read-only` on
channel `200` of a scratch root makes a read-only session against the fake.
Observe. `reach-report`; on the droplet, C30's commands run by hand from a code session's Bash
(`ls`, `env`, `cat` of the root, another session and the canonical) and the hook's `deny` lines
in `session tail` for an `Edit` under read-only reach.
Reproduce. `hook cases` for the policy half; for the bind half, C30 and C31 of ADR 0022 are the
test, run once per host setup and recorded as a finding; the fake exercises no bind.
Isolation. Preflight spawns `engine-entry --check` and nothing else; a scratch root reports the
same-account backend's empty bind list.

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
<input>  <output>  <cache read>  <cache write>  <interrupted turns>`; no filter is the instance
total; `--user` is the requester total, `--credential` the payer total (02). `broodforge audit
[--since-seq <n>] [--kind <kind>[,<kind>]] [--session <id>] [--limit 20] [--follow]` prints
`<seq>  <time>  <kind>  <phase or ->  <actor>  <session or ->  <details, first 80 chars>`;
`--json` prints every column. Kinds are 03's audit event kinds in kebab-case.
Observe. `usage --by turn --json` carries each turn's raw `modelUsage` and `usage`.
Reproduce. A total that disagrees with a fixture: `usage --session s0001 --by turn --json` beside
`grep '"type":"result"' <fixture dir>/turn-<n>.jsonl`.
Isolation. Read-only; without the bot both read the store file (a direct reader needs write
access to `-shm`, ADR 0021; the CLI says `error: store-shm: <path>` when it lacks it).

## Backup snapshot and restore rehearsal

Drive. `broodforge backup snapshot (file)`, `backup verify [path] (file)`, `backup rehearse
[path] (file)` are the ADR 0021 companion's steps and exit codes; `verify` prints `migration
<id>` then `<table>  <rows>` per table; `rehearse` prints the same from the restored copy plus
`reconcile: <n> missing` and, on failure, the scratch root it kept.
Observe. `status` line `backup:`; `ls <root>/backup/`; `backup verify --json`.
Reproduce. `scratch run -- bash tests/repro/backup-cycle.sh` [no-check: no code yet]: a session,
`backup snapshot`, `backup verify --expect ok`, `backup rehearse --expect ok`; a corruption case
is `backup verify <path to a truncated copy>`, exit 1.
Isolation. A rehearsal is itself a scratch root; the nightly job is never started by the bot.

## The Discord adapter

Drive. `broodforge discord check [--guild <id>]` connects once with the bot credential from the
store, prints `guild <id> <name>` then one line per configured channel `<id>  present|missing
<reach>  perms:ok|missing:<names>` (view, send, embed, attach, add-reactions, manage-messages
for pins and deleting a pasted key, manage-channels for a private-channel venue, manage-threads
later, the thread venue is not in the first version), disconnects, exit 1 on any missing; the credential is used for that
connection and never printed. On a dry-run instance the gateway is never opened and the renderer
writes to the outbox; `broodforge discord render --events <jsonl> [--clock fake]` feeds a file of
`SessionEvent`s into the renderer alone and prints the outbox lines with the virtual time of each
create or edit, so the one-per-second coalescer (R-21) is observable without waiting. Inbound
without Discord: `broodforge discord inject message --channel <id> --user <id> --roles <ids>
"<text>"` and `discord inject click --custom-id <id> --user <id> --roles <ids>` (any custom id
of 03: an answer, `stop`, `end` or `start`) run the adapter's own handlers (command parsing, the
key-paste scan, role check, custom-id parsing) with the roles given in place of a guild lookup;
both are refused on a live instance.
Observe. `session outbox`; `discord check --json`; `audit --kind action-refused`.
Reproduce. `discord render --events tests/discord/burst.jsonl --clock fake --expect
'edits<=1/s'` [no-check: no code yet]; an inbound bug is one `discord inject` line.
Isolation. Dry-run opens no connection; `check` is the only command besides `serve` that
connects, and only to the guild it is told. Not exercisable without a guild the owner creates:
the gateway's reconnect and rate-limit paths (N-25), slash-command registration, real button
rendering, private-channel creation (thread creation: later, the thread venue is not in the first version), the direct-message relay (Discord error `50007`)
and what a human sees. There, `serve --discord live` on a scratch-shaped root is the test: the
owner clicks, the agent reads `session tail` and `audit`.

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
