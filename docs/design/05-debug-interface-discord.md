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
word for word is marked `not 04:`. Interim, the owner's review pending: `Ending: checking and
handing off the commits...` (count unknown), the summary's `Title: <title>`, its commit list
capped at ten then ` · and <n> more`, and a subject or title cut with `...`.

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
`Start session` button (`bf:<channel>:start`), a second post edits it, after a restart too; `panel
show <channel id>` prints `<channel>  <message id>  <posted at>` from the store's `panel` table,
loaded at start (`no-panel` before a post); a live create also records the message's Discord id
there, so a restarted live bot edits that message (one with no Discord id is posted anew), and on a
live instance `panel post --json` and `panel show --json` carry it as `discordMessageId`. An edit
Discord answers Unknown Message (the panel deleted by hand) clears that id, so the next post creates it anew.
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

## Golden sessions (P32)

Surface. A golden session is one directory `tests/discord/golden/<name>/` [no-check: created by P32]
holding `events.jsonl`, `session tail --json --all` lines in the `ReplayEvent` shape of `discord
render` (the `session` key as the fixtures carry it, `s0001` when absent; a first `#` line says what
it shows), and what the renderer made of them when the golden was blessed: `outbox.jsonl`, one post
per line as `session outbox --json` prints it less `seq` and `payload`, and one `transcript-<session>.md`
per session in the file, in `session transcript`'s layout over that session's events alone, the names
as the replay's views have them, not the registry's. The events are the input, the rest the blessed
output; nothing else is kept. Why the renderer alone: the bot is never in the loop, so a golden
replays in milliseconds with no root and no timing of its own, and the fake clock steps from the
recorded event times, so the same events render the same posts on every machine.
Drive. `broodforge discord render --golden <dir> [--update] [--expect <token>]` (standalone, no
root) is `discord render --events <dir>/events.jsonl --clock fake` followed by the comparison.
`--golden` beside `--events` or `--clock`, or a `render` with neither `--events` nor `--golden`, is a
usage error, exit 2 (`--clock`'s default is empty and means real, so `--golden` can set fake). The
comparison: each post reduced to the golden's fields against `outbox.jsonl` line by line, then each
session's transcript against its `transcript-<session>.md`. A file that differs prints its first
differing position as two records, `<file>  <line>  golden  <text or ->` and `<file>  <line>  now
<text or ->` (`-` where that side has no such line), then `<file>  more  <n>` when further lines
differ; the error line is `error: golden-differs: <dir>: <files>; --update, then git diff
tests/discord/golden/, shows the whole`, exit 1. Equal, the pacing line of `discord render` prints
and `--expect` judges its token as before, so `--expect 'edits<=1/s'` keeps the pacing check on
every golden. `--update` writes the output files (never `events.jsonl`; a transcript of a session
no longer in the events is removed), prints `golden  updated  <dir>`, exit 0, and compares nothing:
the diff a human reads is `git diff tests/discord/golden/`, the bless is the commit of it and the
revert is `git checkout` of the directory, so the command carries no diff engine and takes no
library for one. `--json` prints the posts and the result as before, then one `golden` object per
differing file with its first differing line and its count of further differences.
What counts as a difference: text is exact. Compared: `time`, `op`, `session`, `channel`,
`messageId`, `length`, `first`, `body`, `buttons` (label, custom id, style), `selects` (custom id,
placeholder, options with their defaults), `pinned`, `file`; `time` is compared because on the fake
clock it comes only from the event times, as stable as the body, so a coalescer that slows shows; a
message id is the replay's own `m<n>` in creation order, never a Discord id, so a post added before
another shifts the ids after it and the diff shows the shift. Not compared: `seq` (the line's
position); `payload`, disgo's request body, not kept at all (the library's shape: a swap to arikawa,
ADR 0023's exit, must not fail every golden). CRLF reads as LF on both sides; a final newline is
not a difference.
Capture. From a session the bot still holds (after `session end`, before `stop`: an ended session's
events stay in memory until the bot stops), on a scratch root (`scratch run -- bash <script>`, the
events written under the OS temp directory, never under the root) or on the dev root (the `run`
skill, steps 0 to 3, then `session end <id> --no-closing-turn`, so the golden is the one turn):
`broodforge session tail <id> --json --all > <dir>/events.jsonl` with the id `session start`
printed, the `#` line prepended, then `go run ./internal/tools/importfixture --events
<dir>/events.jsonl` (P1's token rewrite over one JSONL file in place: every PC home and root path
form to `/home/user`, the project slug, and every whole-token occurrence of the captured session id
to `s0001`, the summaries, `start.branch`, `end.branch`, the custom ids and the worktree paths in
tool targets included; a file with no PC home path gets no home rewrite, which is not an error),
then `discord render --golden <dir> --update`, then the read. The driver of every capture is the
placeholder registration `1001 Ada Example`, on the dev root too: a start names the credential, not
its user (`session start --channel 200 --as 1001 --roles 10 --credential cred-<dev>`), so a
real-engine golden carries no name of the owner's and the owner's registration is never a golden's
driver; a real turn runs on the owner's word only, as the skill's step 4 says, and its prompt names
a file this repository has (`go.mod`). Before the commit, P1's independent read: an agent that did
not capture it reads the golden's files with the nested JSON decoded, confirms no name, email, host
name, path outside the placeholder or repository content survives, and reports what a script cannot
classify; the owner decides on the report. The engine's words about this repository in a `text`
event are repository content: the read says so and the owner decides.
Observe. `discord render --golden <dir>`; `git diff tests/discord/golden/` after `--update`;
`go test ./internal/discord -run TestGolden`.
Reproduce. `bash tests/repro/golden-sessions.sh` [no-check: created by P32] runs `discord render
--golden <dir> --expect 'edits<=1/s'` on every directory under `tests/discord/golden/`, prints
`ok  <name>` or `differs  <name>` per golden with the records of each that differs, and exits 0
only when every golden matches; it needs no root and does not run under `scratch run`, since the
renderer is standalone. The same comparison is `TestGolden` of `internal/discord` (one `Errorf` per
file that differs; `-update` blesses, as the parser goldens of `internal/engine` do), so every
golden runs under every pass's test line and the pusher's check 8 on every push: that is the
nightly form until something schedules nights (a scheduled run of the script is the overnight
idea's to add, `../ideas/2026-10-29-broodforge-works-on-itself-overnight.md`, not P23's task, which
is the service instance's backup). The `run` skill is not the place: it drives one bot on the dev
root, and a golden needs none.
Isolation. No root, no socket, no Discord; `--update` writes only the output files under the named
directory. Not covered: the service's own wording in the events (a trace summary, `NoAnswerCause`)
is the input, so a change there passes every golden and shows in `tests/repro/*.sh` instead; the
live gateway; what a human sees, which stays `serve --discord live` and the owner.

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
