# Debug interface: sessions

Companion to `05-debug-interface.md`, which points here from its Registration and participants,
Session lifecycle without Discord and Prompt-mode defer and resume sections. The service is
`03-architecture.md`'s `session` row and Data flow; a trace line's wording is `04-ux.md`'s, the
owner's; conventions are `05-cli-conventions.md`'s. P10 builds it in two halves: what is marked
(b) below is the second's.

## Registration and participants

Drive. `broodforge register <discord user id> --name "<git name>" --email <email>` prints the
record, `<user id>  <name>  <email>  <registered at>  <default credential id or ->`; again
replaces the pair and writes `registration-changed` with the old one (R-02); a name empty or over
100 characters, or an email without `@` or with a space, is `error: bad-registration: <what>`
plus `action-refused` (04, Registration); a user id that is not decimal digits is `bad-args`.
`register list` and `register show <user id>` (`unregistered` for an id the store lacks) print
the same line; `register set-default <user id> <credential id>` records the credential a start
uses when it names none. Participants are read per session: `session show <id>` lists
`participants: <user id>  driver|contributor  <joined at>`; a writer joins as a contributor on
their first accepted message (R-13), with the event `<name> joined as a contributor.` (04).
Roles without Discord: every session command that acts for a person takes `--as <user id>
[--roles <id,...>]`; given, the channel's `start`, `contribute` or `answer` list (or the driver,
for `mode` and `invite`) is checked against them with `policy`'s role check (03), and a miss is
`error: refused: <action> needs a role in <list>`, exit 1, plus the `action refused` audit event;
the driver may always write in their session. `--roles` absent, the action is unchecked; `--as`
absent, it is the owner's from the host (actor `host`), and `start` and `send` need `--as`, since
a session's driver and a message's writer are registered users (`bad-args`).
Observe. `register list --json`; `audit --kind registration-changed`; `session show`.
Reproduce. `scratch run -- bash tests/repro/full-session.sh`: it registers, starts with `--roles
11 --expect refused`, then with `--roles 10`.
Isolation. A scratch root has no registrations; the ids above are not Discord's.

## Session lifecycle without Discord

Drive. `broodforge session start --channel <id> --as <user id> [--roles <ids>] [--credential
<id>] [--mode prompt|policy] [--reach read-only|code] [--fixture <dir>] [--idle-timeout-seconds
<n>] [--prompt-timeout-seconds <n>]` prints the session id (`s0001`); `--reach`, `--fixture` and
the timeouts are dry-run only (`live-instance`). `--credential` defaults to the driver's
(`no-credential` without one; R-41: `credential-not-usable: <id> is <status>`); `--fixture` is a
name under `engine.fakeFixtures` or a path, default `basic` (`unknown-fixture`; a broken `fixture.json` is `bad-fixture`). A start refused before its row
writes `action-refused` and leaves nothing; one that fails later (the clone, the fake, the store) also
removes the tree and `refs/sessions/<id>` and keeps the row as `ended`, outcome `refused: <code>`, so no
id is reused; the refusal names the stage and keeps the git or fixture code.
`session send <id> --as <user id> [--roles <ids>] [--wait] [--tail] "<text>"` (or `-`
for stdin) calls `SessionService.submit` and prints `turn  <n>  started|queued` (a message during
a turn waits for the next, one `From: <name> (<role>)` block each, 03); `--wait` returns at the
turn's end, exit 0, printing `turn-ended  <n>  completed|error|interrupted|deferred` instead
((b): `waiting-for-answer` and the call); `--tail` streams the turn's events meanwhile and, without
`--wait`, prints the start line after them. (b): `session mode <id> prompt|policy`, `session
interrupt <id>`, `session end <id>` (the end-of-session sequence; the closing turn against the
fake when a fixture named `closing` exists, `--no-closing-turn` skips it) and `session invite <id>
--user <user id>`, with `--as` and `--roles` as above (`mode` and `invite` need the driver,
`interrupt` and `end` the `answer` list). `session list [--state <state>]` prints `<id>  <state>
<channel>  <driver>  <mode>  <reach>  <turns>  <started at>`. `session show <id>` prints `session:
<id>  <state>  <channel>  <project>  <driver>  <mode>  <reach>  <harness>`, `credential: <id>`,
`engine-session: <id or ->`, `turns: <n>`, the `participants:` lines, `worktree: <path>
<branch>  <state>` and `pending: -` ((b): `pending: <tool use id>  <tool>  <target>  since <time>
until <deadline>`). `session tail <id> [--follow] [--since <seq>] [--limit <n>|--all]` prints
`<seq>  <time>  <type>  <summary>`. Types: `session-started` (04's header line, `Session <id> on
<project>, branch <branch>, driver <name>.`), `participant-joined`, `message-queued`,
`turn-started` (the input's first 80 characters), `text` (its first 80), `tool-call` (04's trace
line, `[<tool>] <target>`), `permission-decision` (`<decision> [<tool>] <target> (<decided by>)`),
`tool-result` (the trace line, with 04's outcome after ` -> ` when the call was denied, blocked,
bypassed or failed), `waiting-for-answer` (b), `turn-ended` (the stop reason, or `interrupted`),
`session-interrupted`. A decision row's decider is `policy` (the hook answered from the table),
`hook` (it failed: exit 2, `hook:`, audited `action-refused` with actor `hook`), `config` (the
engine's default: audited `hook-bypassed`), or `unknown` (no source in the stream); a tool's own
`<tool_use_error>` gets no row (03-hook-contract.md, What the bot sees). The events are the bot's,
in memory: without the bot `tail` is `not-running`, and a session an earlier run held is
`session-closed`. P11 (the renderer, `05-debug-interface-discord.md`): `session outbox <id>
[--follow]` prints the dry-run renderer's posts: `<seq>  <time>  create|edit|attach  <message id>
<length>  <first line>`, `--json` with the full body; `session transcript <id> [--out <file>]`
prints 04's Markdown transcript as the summary would attach it; `broodforge panel post <channel
id>` posts or refreshes the project channel's panel (04) and records its message id in the
store, which `panel show <channel id>` prints back; on a dry-run instance it lands in the outbox.
`session reach-report <id>` is (b), in 05 (Spawn preflight and reach).
Observe. `session show`, `session tail --json` (every field: tool, target, decision, decided by,
reason, result), `audit --kind permission-decision --session <id>`, the fake's
`sessions/<id>/fake/received-<n>.json` (the exact `TurnRequest`), `usage --session <id>` (b).
Reproduce. A whole R-51 run in one line: `scratch run -- bash tests/repro/full-session.sh`:
`serve &`, `status --wait 10 --expect ok`, `register 1001 ...`, the refused start, `session start
--channel 100 --as 1001 --roles 10 --credential cred-scratch --mode policy --fixture basic`,
`session send s0001 --as 1001 --wait "..."`, then `session tail` and `audit` checked line by
line, `worktree handoff s0001 --expect none`, `stop`; (b) and P11 add `session answer`, `session
outbox` and `session end`.
Isolation. All of it in the scratch root against the fake; the socket under the root.

## Prompt-mode defer and resume (b)

Drive. On a prompt-mode session with the fixture `fixtures/engine/defer/` (turn 1 ends
`stop_reason: tool_deferred`, turn 2 is the resumed turn): `session send s0001 --as 1001 --wait
"edit a.ts"` prints `waiting-for-answer toolu_01 Write a.ts`; `session answer s0001 toolu_01
allow|deny [--as <user id>] [--roles <ids>]` calls `SessionService.answer` and the resumed turn's
events follow in `session tail`. The idle timeout is driven with `--idle-timeout-seconds 2` at
start and observed as `session-ended` with reason `idle`; `--prompt-timeout-seconds 2` with no
answer shows the timeout trace line (`denied: the prompt timed out`) in `session outbox`.
Observe. `session show` prints `pending: toolu_01 Write a.ts since <time> until <deadline>`;
`received-2.json` carries `answer` and `engineSessionId`; `session outbox` has the buttons post,
custom id `bf:s0001:toolu_01:allow`.
Reproduce. `scratch run -- bash tests/repro/defer-resume.sh` [no-check: P10 half b writes it]; an
answer refused for roles is `session answer ... --roles 11 --expect refused`.
Isolation. Against the real engine this is spike C27 (`engine run --real --resume`), not scratch.
