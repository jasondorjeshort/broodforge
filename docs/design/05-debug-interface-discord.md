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
"<text>"` and `discord inject click --custom-id <id> [--value <v>] --user <id> --roles <ids>` (any custom id
of 03: an answer, `stop`, `end`, `start`, or a header menu's with its pick as `--value`), and `discord inject command` for a slash command,
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
the millisecond. `--json` adds the body, the buttons with their custom ids, the header's select menus
(`selects`, The header's menus below), `pinned` and
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

## The live gateway (P12), the held sessions (P12c) and the inbound log (P12d)

Moved by topic to `05-debug-interface-discord-live.md`: `serve --discord live` (the application check,
the slash commands, the venue sweep, the start-up report, each post as its REST call, the direct-message
reply, the deferred interaction answers, the API-key modal), the `restAPI` surface, the live tests; the
held interrupted sessions' notice, summary and end (P12c); and one stderr line per inbound event with its
outcome and the acknowledgement of every interaction (P12d), all ADR 0026.

## Model and effort (P12a)

Drive. Registered with 04's commands: `/session start model:<alias>` (optional) in a project channel;
in a venue `/model [<alias>]` (top-level; `not-a-venue` elsewhere), `/session model <alias>` and
`/session effort low|medium|high|xhigh|max`, `model` and `effort` as Discord choice lists; setting is the
driver's, `/model` alone anyone's there (`ephemeral  <04's Model: line>`). An alias is the lowercase
second word of a display name in `policy`'s model map (`opus`, `sonnet`, `haiku`, `fable`; the map
gains `claude-fable-5-1` = `Claude Fable 5.1`) and resolves to that entry's id, the value the engine's
`--model` gets; the typed form (`discord inject command --channel s0001 --user 1001 --roles 10
"/model sonnet"`; a venue message starting with `/` is text for the agent) also takes a full id
the map holds, bare or with a date suffix (`-20261001`), passed as typed. Anything else is `bad-args` with 04's `Not a model:` text (or `Not an effort: <value>.
Efforts: low, medium, high, xhigh, max.`) before any turn starts; nothing unknown reaches the engine. The
service side (`SetModel`, `SetEffort`, the events, the session row) is `05-debug-interface-session.md`'s.
The header's parenthesis is `this channel's default`, `chosen at start` or `from turn <n>`; `Model:
the engine's own.` when nothing set it; `Effort:` only once set; the renderer edits the pinned
header in place on a `model-changed`, `effort-changed` or, from P12b, `mode-changed` event.
Observe. `session show`, `session tail`, `session outbox` (an `edit` of the header's message id),
`audit --kind model-changed`, the fake's `received-<n>.json` (`model`, `effort` of the `TurnRequest`).
Reproduce. P12a's Proof (`08-implementation-plan-discord.md`): `"/session start model:sonnet"` on `--channel
100`, then `"/model fable"` and `"/model gpt"` (refused `bad-args`) on `--channel s0001`.
Isolation. Dry run; the fake records the flags and runs no model. Effort is the coder's check first
(`internal/engine/claudecode.go`, `Args`): without an engine flag for it, the effort commands and
column are left out of the pass and the docs' effort lines wait.

## The header's menus (P12b)

Drive. The header's three select menus (`04-ux-session-controls.md`, The header's menus) carry the custom
ids `bf:<session id>:model`, `bf:<session id>:effort` and `bf:<session id>:mode`, 03's forms parsed by the
one function; the value picked travels in the interaction's `values`, never in the id. Option values are what
the slash commands' choice lists take: a model alias, a level of `policy.Efforts`, `asks-first` or
`acts-freely`. The model menu's default is the alias of the map entry the session's model belongs to (a
dated id belongs to its prefix's entry, as `policy.ModelDisplayName` reads it); an id no entry matches marks
none and sets the `Model: <id>` placeholder. From a terminal: `discord inject click --custom-id
bf:s0001:model --value sonnet --user 1001 --roles 10`; `--value` is the one value picked, required with a
menu's id and `bad-args` with a button's. The handler is the slash command's branch (`SetModel`, `SetEffort`,
`SetMode`, shared with `slash`), so the reply is the same `none  model from turn <n>: <id>` (`effort from
turn`, `mode from turn` likewise); a value policy does not know, which inject can send and Discord cannot, is
`bad-args` with 04's `Not a model:` or `Not an effort:` text, a mode value outside the two `bad-args`; a picker
who is not the driver gets 04's `Only the driver, <name>, can change the model.` (`effort`, `mode` likewise), as a
typed `/model`, `/session model|effort|mode` does, while the service's own `model needs the driver, <name>` stays
the CLI's text and the audit row's `message`, audited `action-refused`.
The adapter's own refusals of a pick (`bad-args`, a bad custom id) are audited with the action `click`, the
route the input came by, as every component refusal is (`start` for the panel's button); the shared function
takes the route's action name, so the typed form keeps `command`. The service's refusals name the operation
(`model`, `effort`, `mode`) on either route, as `checkDriver` writes them. A pick on a session that is ending,
ended or interrupted is the service's `session-closed` refusal (`05-debug-interface-session.md`, Session
lifecycle), which the adapter answers in 04's words; nothing is re-sent. `venueRefusal` maps every
`session-closed` by the session's state, read from its row (`svc.Show`), never parsed from the refusal's text:
`ending` and `ended` to `This session has ended.`, `interrupted` to 04's `interrupted refusal` row (ADR 0026);
the reply covers what reaches that code: `End session` (on an ending or ended session only: on a held
interrupted one it ends the session), the switches, a venue message, an answer, `Stop` and `/session invite`,
since the service checks closed before every other check. A non-driver's pick on such a session gets this
text, not the driver-only refusal. `inject click` prints it as `error: session-closed: This
session has ended.` (or the interrupted text), while the typed `session mode|model|effort` keeps the
service's `s0001 is ended`.
After a refused pick on a live session the header is edited unchanged, so the picker's client drops the pick:
the adapter keeps a `views` map, session id to `View`, from `SessionStarted` until that session's follow
returns (the end or the interruption rendered; the entry deleted under the adapter's lock before `Done`), and
calls `View.Resync()`, which under the View's own lock, the one `Handle` holds, posts an `OpEdit` of the header
from the View's `start` and `hdr`, the menus from `headerSelects` and the `End session` button, through the
View's `post` (the mask, then the coalescer); a View that has handled `session-ended` or
`session-interrupted` does nothing. Why: `Handle` and `Resync` share one lock, so the re-sync either precedes
the end's edit in the coalescer's queue, where the end's components-free edit merges over it, or sees the
ended flag and is skipped; built outside the View from `svc.Events`, as `/model`'s query is, a late re-sync
could put the menus and the button back on an ended header. Live, a pick is a component
interaction whose data is a string select's `Values`: acknowledged with the deferred update a button gets,
the refusal as the ephemeral follow-up; a successful pick posts nothing of its own, since the event's `From
the next turn:` line and the header edit are the renderer's. Every header create and edit carries the three
menus from `headerState`, and a `mode-changed` event edits the header as a `model-changed` does; the
`session-ended` event edits the header once more with no components at all, and the coalescer's merge of an
edit into a queued create or edit copies `Selects` with `Body` and `Buttons`, so a switch inside the start's
second and an end inside a switch's second both show the later components. Two client behaviours are
assumed, not observed: that an edit whose components equal the shown ones resets a picker's displayed
choice, and that picking the option already marked default sends an interaction at all; both are the owner's
step under `serve --discord live` (Isolation).
Observe. `session outbox <id> --json`: the header's `create` and each `edit` carry `selects` (each with
`customId`, `placeholder` and `options` of `label`, `value`, `default`), and `payload` the action rows disgo
builds (one per menu, then the button row); after the end, the header's last `edit` has neither
`selects` nor `buttons`. `session tail` (`model-changed`, `effort-changed`, `mode-changed`); `audit --kind
model-changed|effort-changed|mode-changed`; `audit --kind action-refused` for a refused pick. `discord render
--events` replays a switch event into the header's edit as before, the menus visible in a `--json` outbox only.
Reproduce. `go test ./internal/discord -run 'TestSelect|TestCoalesceSelects|TestLive'`: a recorded
string-select interaction fed to `onInteraction` against the fake `restAPI`, asserting the deferred update,
then the header's `UpdateMessage` with the new default; for a non-driver, the ephemeral follow-up and an
`UpdateMessage` of the unchanged header; a `Resync` and a `session-ended` on one View in either order, the
header's last edit without components; the coalescer's two merges (a switch's edit into the queued create,
the end's edit into a queued switch's edit); `tests/repro/full-session.sh` reads the header's `selects` from
`session outbox --json`. An ended session's refusal: P12b's Proof (`08-implementation-plan-discord.md`).
Isolation. Dry run. A real menu renders only under `serve --discord live`, where the owner picks and an agent
reads `session outbox --json` and `session tail`. The owner's step there, once: a non-driver picks a model and
says whether the menu returns to the true value after the unchanged edit; the driver picks the option already
shown and the agent reads `session tail` for a `model-changed` (an interaction arrived) or nothing. A reset
that fails makes the re-sync two edits a second apart, one with no components and one with them back, which
re-creates them; a default pick that sends nothing changes no code, and 04's "a switch like any other" then
describes what never arrives. Either result is a finding, and the assumption lines above are then rewritten.

## Idle timeout

Drive. In a venue, `/session timeout <timeout>` (a free string, at most 20 characters; the
driver's), read by `session.ParseIdleTimeout`: a bad value is `bad-args` with 04's `Not a
timeout:` text before the service is called. The service side is `session timeout`
(`05-debug-interface-session.md`); the renderer posts the event's summary and edits the pinned
header on `timeout-changed`, whose sentence is `Ends after <n> without activity.` or `Never ends
for inactivity.`, with ` Capped at a year.` appended above a year
Observe. `session outbox` (the header's `edit`), `audit --kind timeout-changed`.
Reproduce. `discord inject command --channel s0001 --user 1001 --roles 10 "/session timeout 30"`,
then `"/session timeout soon"` (refused `bad-args`).
Isolation. Dry run.
