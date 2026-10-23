# Debug interface: the Discord adapter

This is the Discord part of `05-debug-interface.md` and shares its Conventions.

## The Discord adapter

Drive. `broodforge discord check [--guild <id>] [--json] (file)` connects once with the bot
credential from the store (`discord.botCredentialId`, a `bot-token`; the guild defaults to
`discord.guildId`), over REST only, no gateway session, prints `guild  <id>  <name>` then one
line per configured channel `<id>  present|missing  <reach>  perms:ok|missing:<names>`: each
project channel with its reach (view, send, embed, attach, read-history, manage-messages for
deleting a pasted key, pin for the panel), the notice channel as `notice` (view, send), each
`venue.category` as `category` (those, add-reactions, and manage-channels for a private-channel
venue; manage-threads later, the thread venue is not in the first version), then
`message-content  present|missing  intent` for the privileged intent; it computes the bot's
permissions from the guild's roles and the channel's overwrites, as Discord does. Any missing
is `error: discord-missing: <n> of <m> missing`, exit 1. No token, a value without a bot token's
shape (checked before anything connects), or a failed call is `error: start-discord: <why>`,
exit 1, as `serve --discord live` refuses; the credential is used for that connection and never
printed. On a dry-run instance the gateway is never opened and the renderer
writes to the outbox; `broodforge discord render --events <jsonl> [--clock fake]` feeds a file of
`SessionEvent`s into the renderer alone and prints the outbox lines with the virtual time of each
create or edit, so the one-per-second coalescer (R-21) is observable without waiting. Inbound
without Discord: `broodforge discord inject message --channel <id> --user <id> --roles <ids>
"<text>"` and `discord inject click --custom-id <id> --user <id> --roles <ids>` (any custom id
of 03: an answer, `stop`, `end` or `start`), and `discord inject command` for a slash command,
run the adapter's own handlers (command parsing, the
key-paste scan, role check, custom-id parsing) with the roles given in place of a guild lookup;
all are refused on a live instance.
Observe. `session outbox`; `discord check --json`; `audit --kind action-refused`.
Reproduce. `discord render --events tests/discord/burst.jsonl --clock fake --expect
'edits<=1/s'`; an inbound bug is one `discord inject` line.
Isolation. Dry-run opens no connection; `check` is the only command besides `serve` that
connects, and only to the guild it is told. Not exercisable without a guild the owner creates:
the gateway's reconnect and rate-limit paths (N-25), slash-command registration, real button
rendering, private-channel creation (thread creation: later, the thread venue is not in the first version), the direct-message relay (Discord error `50007`)
and what a human sees. There, `serve --discord live` on a scratch-shaped root is the test: the
owner clicks, the agent reads `session tail` and `audit`.

## The dry-run renderer, outbox and inbound handlers (P11)

`internal/discord` is the adapter without the gateway: `serve` gives the session service a hook
that starts one renderer per session, following its events from the first; every post goes
through the outbound mask, then one coalescer for the bot, then the outbox. The wording is
`04-ux.md`'s, built in one file (`internal/discord/text.go`; the no-answer trace phrase is
`session.NoAnswerCause`, shared with `session tail`), where a string 04 does not give
word for word is marked `not 04:`.

Drive. `broodforge discord render --events <jsonl> [--clock fake|real] [--expect <token>]`
(standalone, no root) reads `session tail --json` lines, each with an optional `"session"` key
(default `s0001`), sorts them by time and replays them; `--clock fake` steps a virtual clock from
one due post to the next, `real` (the default) waits on the wall clock. It prints each post as
`session outbox` does, then `<token>  <requests>  <most creates and edits of one session in a
second>  <most posts of the bot in a second>`; the token is `edits<=1/s`, else `edits>1/s` or
`requests>50/s`. `discord inject message|click|command` (above; `command --channel <id> --user
<id> --roles <ids> "/<command> ..."` stands in for a slash-command interaction) is a socket
client. Every kind runs the key-paste scan first (once Unicode format characters and nonspacing marks are removed, `sk-ant-` with eight or more key
characters after it anywhere, or with any after no letter or digit: `error: key-pasted: <04's refusal>`, exit 1, an `action-refused` event,
reason `key-pasted`, with no key text). A `message` is always text, even one starting with `/`:
in a venue (in a dry run its channel id is the session id, `--channel s0001`) it is submitted,
in a project channel a permitted registered writer gets 04's lobby hint, anything else is
`none`. A `command` is parsed (`/register name:<name> email:<email>`, `/session
start|stop|end|mode asks-first|acts-freely|invite user:<@id>`, `/credential add kind:subscription|api-key
label:<text>|list|remove id:<id>|default id:<id>|login id:<id>`, the caller's own credentials only
(`not-yours`), `login` answering the host route while the relay is not built; `/usage` is
`not-built`). `/credential add kind:api-key` prints `modal  bfm:api-key:<label>`, the form's
custom id, since the key comes back only through the live modal. It prints `ephemeral  <text>` or `none  <what happened>`; a refusal is `error:
<code>: <04's text>`, and the adapter's own (`bad-args`, `not-a-venue`, `not-built`,
`live-instance`, a bad custom id, shown only by its length and prefix) are audited
`action-refused` like the service's. `--roles` stands in for the guild lookup and is always
checked (none given is no roles). `broodforge panel post <channel id>` posts the panel with its
`Start session` button (`bf:<channel>:start`), a second post edits it; `panel show <channel id>`
prints `<channel>  <message id>  <posted at>` from the bot's memory (`no-panel` before a post).
Observe. `session outbox <id> [--channel <id>] [--follow] [--since <seq>] [--limit <n>|--all]`
(the bot's memory, `not-running` without it) prints `<seq>  <time>  channel|create|edit|delete|attach
<message id>  <length>  <first line>`: `channel` creates the venue (`#s0001-<first name>`),
`delete` is 04's delete-and-replace (the final content is created first, then the working
messages deleted), `attach` puts `<id>.md` on the project channel's summary. Times are UTC to
the millisecond. `--json` adds the body, the buttons with their custom ids, `pinned` and
`payload`, the request body disgo builds for it (`MessageCreate` or `MessageUpdate`); the control
wire escapes `<` and `>` in `--json` output as JSON's unicode escapes (backslash, `u003c`),
so match a mention or an arrow in text mode. `--channel <id>` shows every post to one channel,
the panel included, with the id `-` for any session. `--follow` ends when the session's last
post is out. `session transcript <id> [--out <file>]` prints 04's Markdown
transcript from the session's events.
Pacing. One create, edit or attach per second per session and 50 posts per second per bot; an
edit merges into a queued create or edit of its message (the latest content wins), a delete
drops that message's queued edits, nothing else is merged or dropped, and sessions take turns:
each pass starts after the session served last.
A turn shorter than a second still shows its working message: its create goes out with the
latest content merged in, then the final message, then the delete.
Reproduce. Both repro scripts of `05-debug-interface-session.md` read the outbox:
`full-session.sh` the venue, header, working message and the final message's footer;
`defer-resume.sh` the prompt with its three custom ids, `Allowed by`, `No answer; denied` and
turn 2's timeout trace line.
Isolation. All in memory in the bot; a dry run sends nothing anywhere.

## The live gateway (P12)

`internal/discord/gateway.go` puts disgo's gateway client (intents Guilds, GuildMessages and the
privileged MessageContent, no DMs) and its REST client behind the same handlers and sink. The
REST calls it makes are the `restAPI` interface at the top of that file, the surface a swap to
arikawa covers; resume, reconnects and the per-route limits are disgo's.
Drive. `serve --discord live` on a root whose `discord` keys are set: after the control socket it
checks the application behind the token against `discord.applicationId` and the
message-content intent, registers 04's slash commands in `discord.guildId` (`/register`,
`/session start|stop|end|mode|invite`, `/credential add|list|remove|default|login`, `/usage`),
binds the venues of earlier runs under each `venue.category` (a text channel whose topic is
`<instanceName>:<session id>`, the mark every venue is created with: an open or interrupted
session's is bound again, an ended one's is deleted ten minutes after its end; a channel
without the mark or whose session the store does not know is left alone, and when two carry
one session's mark neither is bound or deleted and a warning names them), connects, and
posts 04's start-up report to the notice channel. Every post still passes the mask and the
coalescer and lands in the outbox first, then goes out as its REST call: `channel` creates the
private venue under the project's category with that topic (@everyone denied view; the bot,
Manage Channels included, the driver and the channel's `contribute` and `answer` roles allowed), a pinned create is pinned, `attach` adds the
transcript file to the summary; ten minutes after a session's last post the venue is deleted
on the adapter's clock. A message in a venue or lobby runs `discord inject message`'s handler:
a pasted key is deleted where the bot may (an edited message gets the key scan alone), a queued message gets an hourglass reaction, and a
reply or refusal goes to the writer by direct message (Discord has no ephemeral reply to a
message), in the channel with a mention when they take none. A slash command or a click is
deferred, then answered ephemerally (a click's answer is a follow-up); `/credential add
kind:api-key` answers with the modal, whose submission stores the key and answers `Stored
<id> (API key, ends in ...<last four>).`. `status` shows `discord: live connected` while the
gateway session is ready.
Observe. `session outbox` (a live instance's too); `status`; `audit --kind action-refused`;
warnings on the bot's stderr as `warning: discord: <what>: <error>`, one per failed call.
Reproduce. `go test ./internal/discord -run 'TestLive|TestSweep|TestCheck'`: a fake of
`restAPI` records every call with its disgo request body while a real session service runs
the fake engine; a guild bug is first a recorded interaction or message JSON fed to
`onInteraction` or `onMessage` there.
Isolation. Only `serve --discord live` and `discord check` connect, both to the guild
configured or given; a value without a bot token's shape is refused before any connection.

## Model and effort (P12a)

Drive. Registered with 04's commands: `/session start model:<alias>` (optional) in a project channel;
in a venue `/model [<alias>]` (top-level; `not-a-venue` elsewhere), `/session model <alias>` and
`/session effort low|medium|high|max`, `model` and `effort` as Discord choice lists; setting is the
driver's, `/model` alone anyone's there (`ephemeral  <04's Model: line>`). An alias is the lowercase
second word of a display name in `policy`'s model map (`opus`, `sonnet`, `haiku`, `fable`; the map
gains `claude-fable-5-1` = `Claude Fable 5.1`) and resolves to that entry's id, the value the engine's
`--model` gets; the typed form (`discord inject command --channel s0001 --user 1001 --roles 10
"/model sonnet"`; a venue message starting with `/` is text for the agent) also takes a full id
the map holds, bare or with a date suffix (`-20261001`), passed as typed. Anything else is `bad-args` with 04's `Not a model:` text (or `Not an effort: <value>.
Efforts: low, medium, high, max.`) before any turn starts; nothing unknown reaches the engine. The
service side (`SetModel`, `SetEffort`, the events, the session row) is `05-debug-interface-session.md`'s.
The header's parenthesis is `this channel's default`, `chosen at start` or `from turn <n>`; `Model:
the engine's own.` when nothing set it; `Effort:` only once set; the renderer edits the pinned
header in place on a `model-changed` or `effort-changed` event.
Observe. `session show`, `session tail`, `session outbox` (an `edit` of the header's message id),
`audit --kind model-changed`, the fake's `received-<n>.json` (`model`, `effort` of the `TurnRequest`).
Reproduce. P12a's Proof (`08-implementation-plan.md`): `"/session start model:sonnet"` on `--channel
100`, then `"/model fable"` and `"/model gpt"` (refused `bad-args`) on `--channel s0001`.
Isolation. Dry run; the fake records the flags and runs no model. Effort is the coder's check first
(`internal/engine/claudecode.go`, `Args`): without an engine flag for it, the effort commands and
column are left out of the pass and the docs' effort lines wait.

## Idle timeout

Drive. In a venue, `/session timeout <timeout>` (a free string, at most 20 characters; the
driver's), read by `session.ParseIdleTimeout`: a bad value is `bad-args` with 04's `Not a
timeout:` text before the service is called. The service side is `session timeout`
(`05-debug-interface-session.md`); the renderer posts the event's summary and edits the pinned
header on `timeout-changed`, whose sentence is `Ends after <n> without activity.` or `Never ends
for inactivity.`
Observe. `session outbox` (the header's `edit`), `audit --kind timeout-changed`.
Reproduce. `discord inject command --channel s0001 --user 1001 --roles 10 "/session timeout 30"`,
then `"/session timeout soon"` (refused `bad-args`).
Isolation. Dry run.
