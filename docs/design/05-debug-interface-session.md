# Debug interface: sessions

Companion to `05-debug-interface.md`, which points here from its Registration and participants,
Session lifecycle without Discord and Prompt-mode defer and resume sections. The service is
`03-architecture.md`'s `session` row and Data flow; a trace line's wording is `04-ux.md`'s, the
owner's; conventions are `05-cli-conventions.md`'s.

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
<seconds>] [--prompt-timeout-seconds <seconds>]` prints the session id (`s0001`); `--reach`, `--fixture`
and the timeouts are dry-run only (`live-instance`). Every timeout is at most a year: a larger
`--...-timeout-seconds` is capped, with `idle timeout capped at a year` (or `prompt`) on stderr, and
0 or absent takes the channel's value. In a timeout's words `<n>` is a count with its unit, `5 minutes`
for whole minutes and else seconds (`90 seconds`). Never is a timer of a year, worded "never".
`--credential` defaults to the driver's (`no-credential` without one; R-41: `credential-not-usable: <id> is <status>`); `--fixture` is a
name under `engine.fakeFixtures` or a path, default `basic` (`unknown-fixture`; a broken
`fixture.json` is `bad-fixture`). A start refused before its row writes `action-refused` and
leaves nothing; one that fails later (the workflow copy, the clone, the fake, the store) also
removes the copy, the tree and `refs/sessions/<id>` and keeps the row as `ended`, outcome
`refused: <code>`, so no id is reused; the refusal names the stage and keeps the git, workflow or
fixture code. The copy comes first (ADR 0025 rules 5 and 6): `workflow/<id>/` rendered from the
canonical's default branch as `workflow render` does, then each `hooks.json` entry probed as
`workflow probe` does; a missing runtime refuses before the clone with `error:
workflow-probe-failed: <command> needs <word>`, exit 1, `action-refused` with reason `workflow`;
through Discord (`discord inject click --custom-id bf:<channel>:start`) the starter alone gets
the ephemeral `Could not start a session: <command> needs <word>.` and the outbox holds nothing
for it (`session outbox - --channel <channel>` shows no post). `session send <id> --as <user id>
[--roles <ids>] [--wait] [--tail] "<text>"` (or `-` for stdin) calls `SessionService.submit` and
prints `turn  <n>  started|queued` (a message during a turn, or while it waits for an answer,
waits for the next turn, one `From: <name> (<role>)` block each, 03); `--wait` returns at the
turn's end or its wait for an answer, exit 0, printing `turn-ended  <n>
completed|error|interrupted|deferred` or `waiting-for-answer  <n>  <tool use id> [<tool>]
<target>` instead; `--tail` streams the turn's events meanwhile and, without `--wait`, prints the
start line after them. `session mode <id> prompt|policy` (the driver; from the next turn) prints
`mode-changed  <next turn>  <mode>`. `session start` also takes `[--model <alias|model id>]`, resolved
as `05-debug-interface-discord.md` (Model and effort) says; absent, the channel's own `model` key (`defaults`
has none; null is unset); the result is copied to the session row at start like `reach` (empty: the engine's
own), and the spawn reads the row, not the channel. `session model <id> <alias|model id>` and `session
effort <id> low|medium|high|max` (the driver; from the next turn) print `model-changed  <next turn>
<model id>` and `effort-changed  <next turn>  <level>`, write the audit kind of that name with `from`
and `to`, and refuse a bad value as `bad-args` naming the accepted ones. `session timeout <id>
never|<minutes>|<duration>` (the driver; effective now) sets the session's idle timeout (`never`,
`off` or `0` is never; a bare number is minutes; else a Go duration of at least a minute such as
`1h30m15s`, which reads `5415 seconds`; above a year is capped, and the summary adds
` Capped at a year.`) and
prints `timeout-changed  <next turn>  <never or 04's words>`; audit `timeout-changed` with `from`
and `to`; held in the bot's memory, not the row. `session interrupt <id>` (the `answer` list) stops the
running turn and prints `turn-ended  <n>  interrupted` once it ended; while a call waits for an
answer it denies the call with the reason `interrupted` and ends the turn without a resume (the
default until the owner decides); otherwise `no-running-turn`. `session end <id>
[--no-closing-turn]` (the `answer` list) runs 03's end of session: a running turn interrupted,
then the handoff, printing its line as `worktree handoff` does; a refused handoff
(`handoff-refused-*`, its failing commits streamed first) keeps the session open, waiting for
input (after an idle end the idle timer waits for the next activity: one attempt); anything else
ends it. Ending (end, idle, `stop`) while a call waits records it `dropped`, decider `instance`,
with the cause (`the session ended`, `the session ended: no activity`, `the bot stopped`), and
ends its turn (`turn-ended`, result `interrupted`); each queued message is dropped with a
`message-dropped` event and an `action-refused` row. The closing turn that
writes the pull request is not run yet (no `closing` fixture; the fallback title stands), so
`--no-closing-turn` changes nothing. `session invite <id> --user <user id>` (the driver) prints
the invited registration and writes `participant-invited`; the invitee becomes a participant on
their first message (R-13), so no co-author line is owed for an invitation alone. `session list
[--state <state>]` prints `<id>  <state>  <channel>  <driver>  <mode>  <reach>  <model or ->  <turns>
<started at>`. `session show <id>` prints `session: <id>  <state>  <channel>  <project>  <driver>  <mode>
<reach>  <harness>`, `credential: <id>`, `engine-session: <id or ->`, `model: <id or ->  effort: <level or ->`, `turns: <n>`, the
`participants:` lines, `worktree: <path>  <branch>  <state>` and `pending: -`, or while waiting
`pending: <tool use id>  <tool>  <target>  since <time>  until <deadline>`, then ADR 0025's
lines: `workflow: <commit> <date>` or `workflow: none (inject none)` or `none (<branch> at
<commit> has no CLAUDE.md or .claude)` (`-` for a session started without a copy), `hooks:
<names>` (`hooks.json`'s, `-` for none), and `review: <tool use id> after <commit>` or `review:
none since <commit>` (rule 7: `<commit>` is `refs/sessions/<id>`, the session's last commit as of
the last bundle, which `worktree check` refreshes; the id is the newest `Agent` (or `Task`) call
whose result's `subagent-result` audit row, written when the tool result comes and never at a
person's answer, has `agent` `reviewer`, `result` `ok`, and a time after that commit's committer
second: a reviewer allowed and then interrupted has no such row, an errored or denied one is
`result` `error`; `-` once handed off, no ref left). A fact that cannot be read prints `review:
error: <code>`, the rest of the record still printed and `--json`'s `review.error` holding the
message: `session-ref` when the canonical cannot be read (moved, not a repository),
`unknown-channel` when the bot's configuration no longer holds the session's channel, and without
a bot `bad-config` or `unknown-project`. Without a bot the store, the configuration and the
canonical give the same lines. `session show --header <id>` prints only the header the
session's prompt opens with (`instructions.md` in its copy up to the first blank line;
`no-header` without a copy). `session tail <id>
[--follow] [--since <seq>] [--limit <n>|--all]` prints `<seq>  <time>  <type>  <summary>`; a
subagent's `text`, `tool-call`, `permission-decision` and `tool-result` summaries are indented two
spaces per level under their `Agent` line, from the stream's `parent_tool_use_id` (`--json`:
`parentToolUseId`, `depth`).
Types: `session-started` (04's header line, `Session <id> on <project>, branch <branch>, driver
<name>.`), `participant-joined`, `participant-invited` (`<name> was invited.`), `message-queued`, `message-dropped` (`<cause>: <text>`),
`turn-started` (the input's first 80 characters), `text` (its first 80), `tool-call` (04's trace
line, `[<tool>] <target>`), `permission-decision` (`<decision> [<tool>] <target> (<decided
by>)`), `tool-result` (the trace line, with 04's outcome after ` -> ` when the call was denied,
blocked, bypassed or failed), `waiting-for-answer` (`<tool use id> [<tool>] <target>`),
`mode-changed` (04's `From the next turn: asks first.` or `acts freely.`), `model-changed` and
`effort-changed` (04's `From the next turn: Sonnet.`, `From the next turn: effort high.`),
`timeout-changed` (04's `Ends after <n> without activity.` or `Never ends for inactivity.`), `turn-ended` (the stop
reason, or `interrupted`), `handoff-refused`, `session-ended` (`Session <id> ended by <name>.`, or
04's `Session <id> ended: no activity for <n>.`), `session-interrupted`. A decision row's
decider is `policy` (the hook answered from the table; also the `deferred` row, and the `dropped`
row of every other call of that message, C28), `participant` or `prompt-timeout` (an answer),
`instance` (dropped as its session ended), `hook` (it failed: exit 2, `hook:`, audited `action-refused` with actor `hook`), `config` (the
engine's default: audited `hook-bypassed`), or `unknown` (no source in the stream); a tool's own
`<tool_use_error>` gets no row (03-hook-contract.md, What the bot sees). Each engine process's
usage is one record per model, written with the turn row before `turn-ended` (N-21): the result's
`modelUsage` minus the engine session's last result, or for a process without a result the
deduplicated assistant counts, no output, flagged interrupted and never a baseline (S2);
`harness_ref` is `<session id>/<engine session id>/<turn>.<process>/<model>`. The events are the bot's, in
memory: without the bot `tail` is `not-running`, and a session an earlier run held is
`session-closed`. `session reach-report <id>` prints one `<tools>  <cell under the tree or
canonical>  <cell elsewhere>` line per row of the policy table for the session's reach (`*` for
every other tool), then `binds: -`, the empty bind list until P14; `--json` both; `--expect` the
reach. P11 (the renderer, `05-debug-interface-discord.md`, which has the detail): `session outbox
<id> [--follow]` prints the dry-run renderer's posts: `<seq>  <time>  channel|create|edit|delete|attach
<message id>  <length>  <first line>`, `--json` with the full body; `session transcript <id> [--out
<file>]` prints 04's Markdown transcript as the summary would attach it; `broodforge panel post
<channel id>` posts or refreshes the project channel's panel (04) and keeps its message id in the
bot's memory (a store record is later), which `panel show <channel id>` prints back; on a dry-run
instance it lands in the outbox.
Observe. `session show`, `session tail --json` (every field: tool, target, decision, decided by,
reason, result), `audit --kind permission-decision --session <id>`, the fake's
`sessions/<id>/fake/received-<n>.json` (the exact `TurnRequest`, plus `args`, the flags it sets
on the real engine: `--append-system-prompt-file <root>/workflow/<id>/instructions.md` always,
`--agents <root>/workflow/<id>/agents.json` when the copy has agents, never
`--append-system-prompt`; `hookEnv` holds `BROODFORGE_WORKFLOW` when something was copied),
`usage --session <id> --by turn --json` (its `raw` holds the model's cumulative `modelUsage` and
the delta recorded). A subagent's calls on the fake: `--fixture subagent`.
Reproduce. A whole R-51 run in one line: `scratch run -- bash tests/repro/full-session.sh`:
the canonical given `fixtures/workflow/`'s files and both hook events (node on `PATH`), `serve
&`, `status --wait 10 --expect ok`, `register 1001 ...`, the refused start, `session start
--channel 100 --as 1001 --roles 10 --credential cred-scratch --mode policy --fixture basic`,
`session send s0001 --as 1001 --wait "..."`, then `session tail` and `audit` checked line by
line, the request's flags, `session show`'s three workflow lines and `--header`, `session
reach-report`, `usage`, `config reload`, `worktree handoff s0001 --expect none`, `stop`; P11
adds `session outbox`.
Isolation. All of it in the scratch root against the fake; the socket under the root.

## Prompt-mode defer and resume

Drive. On a prompt-mode session (channel `100` of a scratch root) with the fixture `defer` (turn 1
ends `stop_reason: tool_deferred`; its resume is the fixture's turn 2): `session send s0001 --as
1001 --wait "write hi.txt"` prints `waiting-for-answer  1  <tool use id> [Write] <target>`;
`session answer <id> allow|deny|allow-session [--tool-use-id <id>] [--reason <text>] [--as <user
id>] [--roles <ids>] [--wait] [--tail]` calls `SessionService.answer` (the channel's `answer`
list; a driver-only list refuses others with 04's `Only <name> (the driver) can answer this.`;
`--tool-use-id`, when given, must be the pending call, `not-pending`; nothing pending is
`no-pending`), prints `turn  <n>  resumed` or with `--wait` the outcome, and the resumed process
(the answer in `BROODFORGE_ANSWER`, no stdin line, C27) continues the same turn in `session
tail`. `allow-session` also puts the tool in `BROODFORGE_SESSION_ALLOW` from that resume on (04,
Owner 6); `--reason` on a deny is `BROODFORGE_ANSWER_REASON` (with an allow it is `bad-args`: the agent sees
no reason for an allow); a resume that cannot start ends the turn as an error and the answer is
`resume-failed`. The prompt timeout
(`--prompt-timeout-seconds <seconds>` at start, else the channel's minutes) denies the pending
call with the reason `the prompt timed out after <n>`, decider `prompt-timeout`, and the turn
finishes: the trace line `[<tool>] <target> -> denied: the prompt timed out after <n>.` (04).
With none set (never) the call still carries a real deadline, a year out: `session show` prints it
as `until`, and `--json` has `until` always. The idle timeout (`--idle-timeout-seconds <seconds>`
at start, else the channel's minutes; `session timeout` changes it) runs from the last turn's end,
or the start of a wait, and ends the session (`session-ended`, reason `idle` in `audit`); the two
timers are independent (R-53). Both run on the service's clock, which the tests replace
(`session.ManualClock`). With the fixture `parallel`, the message's two Writes leave one
`deferred` and one `dropped` row.
Observe. `session show` prints `pending: <tool use id>  Write  <target>  since <time>  until
<deadline>` while waiting; `received-2.json` carries `answer` and `engineSessionId`; `audit --kind
permission-decision` the deferred, dropped and answered calls; P11's `session outbox` the buttons
post, custom id `bf:s0001:<tool use id>:allow`.
Reproduce. `scratch run -- bash tests/repro/defer-resume.sh`; an answer refused for roles is
`session answer s0001 allow --as 1001 --roles 11 --expect refused`.
Isolation. Against the real engine this is spike C27 (`engine run --real --resume`), not scratch.
