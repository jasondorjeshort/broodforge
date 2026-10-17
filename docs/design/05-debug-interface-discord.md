# Debug interface: the Discord adapter

This is the Discord part of `05-debug-interface.md` and shares its Conventions.

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
client. Every kind runs the key-paste scan first (`sk-ant-` at a word start once Unicode format
characters are removed: `error: key-pasted: <04's refusal>`, exit 1, an `action-refused` event,
reason `key-pasted`, with no key text). A `message` is always text, even one starting with `/`:
in a venue (in a dry run its channel id is the session id, `--channel s0001`) it is submitted,
in a project channel a permitted registered writer gets 04's lobby hint, anything else is
`none`. A `command` is parsed (`/register name:<name> email:<email>`, `/session
start|stop|end|mode asks-first|acts-freely|invite user:<@id>`; `/credential` and `/usage` are
`not-built`). It prints `ephemeral  <text>` or `none  <what happened>`; a refusal is `error:
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
