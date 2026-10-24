# Debug interface: the live gateway

Sibling of `05-debug-interface-discord.md`, under its Conventions: the part of the Discord adapter that
connects to a guild (P12), the held sessions a restart leaves (P12c) and what the gateway writes and
answers for every inbound event (P12d), both ADR 0026. Moved here by topic when the Discord debug doc reached
the size cap; the dry-run renderer, `discord check|render|inject`, the switches and the header's menus
stay there.

## The live gateway (P12)

`internal/discord/gateway.go` puts disgo's gateway client (intents Guilds, GuildMessages and the
privileged MessageContent, no DMs) and its REST client behind the same handlers and sink. The
REST calls it makes are the `restAPI` interface at the top of that file, the surface a swap to
arikawa covers; resume, reconnects and the per-route limits are disgo's.
Drive. `serve --discord live` on a root whose `discord` keys are set: after the control socket it
checks the application behind the token against `discord.applicationId` and the
message-content intent, registers 04's slash commands in `discord.guildId` (`/register`,
top-level `/model`, `/session start|stop|end|mode|model|effort|timeout|invite`, `/credential add|list|remove|default|login`, `/usage`),
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
<id> (API key, ends in ...<last four>).`. Every interaction is acknowledged, the shapes the bot
does not handle included (Inbound log and held sessions, below). On a live instance the coalescer starts
only after the gateway has opened (ADR 0026): the held sessions' notices, offered at step 7, and anything
a socket client posts meanwhile wait in the queue until the sweep has bound the venues, so no create
is sent to a venue the gateway does not know yet; a dry run starts it with the socket. `status` shows
`discord: live connected` while the gateway session is ready.
Observe. `session outbox` (a live instance's too); `status`; `audit --kind action-refused`; on the
bot's stderr one `discord: inbound` line per event (below) and `warning: discord: <what>: <error>`,
one per failed call.
Reproduce. `go test ./internal/discord -run 'TestLive|TestSweep|TestCheck'`: a fake of
`restAPI` records every call with its disgo request body while a real session service runs
the fake engine; a guild bug is first a recorded interaction or message JSON fed to
`onInteraction` or `onMessage` there; `TestLiveHeld` loads an interrupted row before `prepare`, runs the
sweep against a channel carrying its mark and then the coalescer, and sees the notice's `CreateMessage` in
that channel and no `has no venue` warning.
Isolation. Only `serve --discord live` and `discord check` connect, both to the guild
configured or given; a value without a bot token's shape is refused before any connection.

## Inbound log and held sessions (P12c and P12d, ADR 0026)

Drive. Nothing new to type: the gateway writes one line to the bot's stderr (`GatewayConfig.Log`; the
`serve.log` a repro keeps) for every inbound message, message edit and interaction, when its handler has
finished and the outcome is known, always on. Not logged: the bot's own messages and edits echoed back by
the gateway (its posts run about one a second per session), and every other dispatch (guild, channel,
typing and presence events), which the handler passes over as before:

```
<time> discord: inbound <kind> user=<id> channel=<id> <route> -> handled|refused <code>|dropped <reason>
```

`<time>` is UTC to the second (RFC 3339). `<kind>` is `message`, `edit`, `command`, `click` (a button or a
menu pick), `modal` or `other`. `<route>` names what was asked without quoting anything a person typed or
the bot itself did not shape: a message's is `venue <session id>` or `lobby`, or `-`; a command's is its
path (`session end`, `register`), never an option; a click's is the parsed kind and session (`end s000a`,
`answer s000a allow`, `model s000a`), a menu pick without its value, an id that did not parse `?`; a
modal's is `api-key` or `?`; `other`'s is the interaction type's number. Never message text, an option, a
field, a key or a raw custom id (N-12). `user` and `channel` are Discord ids, `-` when the event has none.
Outcomes: `handled` when the action ran or the adapter replied (an ephemeral reply, a `none` result, a
queued message); `refused <code>` when a refusal was answered, the audit row's code; `dropped <reason>`
when nothing ran and nobody was answered, or only the acknowledgement went out: `bot` (another bot's
message), `webhook`,
`other-guild`, `unknown-interaction`, `unknown-command` (an application command that is not a slash
command), `unknown-component`, `unknown-modal`, `unconfigured-channel` (a message where no channel is
configured and no venue is bound), `lobby-not-permitted` (a lobby message from someone who may not
start). Drops are logged, never audited: the audit log is for actions (N-17). A failed REST call keeps its
`warning: discord:` line beside the inbound line.
Every interaction is answered within Discord's three seconds: the handled shapes as before (deferred, then
the ephemeral answer or the click's follow-up; the API-key modal first); every other shape, another guild's
included, gets an immediate ephemeral `CreateMessage`, text `Broodforge cannot handle this interaction.`
(not 04), and the `dropped` line. Autocomplete (type 4) cannot take a message; nothing registers one, so
one that arrives is logged `dropped unknown-interaction` and not answered.
Held sessions: the service's part is `05-debug-interface-session.md` (Held sessions). The adapter's `View`
takes a held session's first event, `session-interrupted` with a `start` payload: it sets the start facts,
lobby and venue (a dry run's venue is the session id; live, the sweep bound the channel), marks the View
closed (the pinned header's message id is gone with the old process, so no header edit follows, and the
header keeps its menus and `End session`: ADR 0026, point 6), and posts 04's `interrupted (R-17)` notice in
the venue, 03's startup step 10, from the event's `start.commits`: `Its tree is kept with <n> commits` (`1
commit` in the singular), `Its tree is kept.` when the count is null, and `Its tree is missing.` when the
service found the ref present but the tree gone (the worktree row `removed` or its path absent); a session
whose ref is gone too was handed off already and reads `Its tree is kept.` with no count. The summary of an
end by either escape carries 04's drafted line (Ending a session) from `EndInfo`'s outcome, branch and URL. The
transcript (`Transcript`) and the header state a bare `/model` reads (`headerOf`) take their start facts from
a `session-interrupted` event carrying `start` as they do from `session-started`, so a held session's
transcript opens with its heading and the restart line. On `session-ended` the summary, the lobby's copy with
the transcript attached and the venue's deletion follow as for any session; the transcript holds what came
after the restart.
Observe. The stderr lines; `audit --kind action-refused` (every service refusal, reason the code);
`session tail <id>` on a held session (`session-interrupted  restart` first); `session outbox <id>`
(the notice's `create`, then after `end` the summary and the lobby's `attach`).
Reproduce. `scratch run -- bash tests/repro/restart-end.sh` [no-check: written by P12c, extended by P12d]:
`serve`, a start, one turn, a commit in the tree with the Haiku co-author trailer, `stop`, `serve` again
(its stderr `held  s0001`); `session tail s0001` shows `session-interrupted  restart`, `session outbox s0001`
the notice with `1 commit`, `session transcript s0001` the heading and the restart line; `discord inject
click --custom-id bf:s0001:end --user 1001 --roles 10` ends it: `session-ended` in `audit` with
`"state":"interrupted"`, the summary and the lobby's `attach` in the outbox. P12d adds, before the click,
`discord inject command --channel s0001 ... "/model sonnet"` and `inject message` both refused
`session-closed` with the interrupted text and audited `closed`, and after it `session end s0001 --as 1001
--expect session-closed` (exit 0, the match) with its row. `go test ./internal/session -run TestHeld` (P12c):
a store with an interrupted row loaded into a new service, `end` closing it with `state` `interrupted`, a
refused handoff returning it to `interrupted`, a gone tree with its ref present ending it `tree-missing`, a
session `worktree handoff` finished earlier (tree and ref gone, `end_outcome` the branch) ending it `pushed`
with that branch and no handoff run, one whose `end_outcome` is null ending `none`, `Submit` and the switches
refused `session-closed` with the interrupted text. `go test ./internal/discord -run 'TestLive|TestInbound'` (P12d): the live harness feeds a
recorded message, a slash command, a button, a menu pick, the API-key modal, a bot's message, another
guild's interaction, a user-command interaction and a modal with another custom id, and asserts one
`discord: inbound` line each with the outcome, no text or custom id in the log, and an acknowledgement for
every interaction; `go test ./internal/session -run TestRefusalsAudited` (P12d): every bare refusal of the
list in `05-debug-interface-session.md` writes its row, a refused `tail` and the idle void write none.
Isolation. Dry run and the fake REST; the live format is the same code. The owner's step, once: after a
restart of the live bot, click `End session` on an interrupted session's header and read `serve.log` and
`audit` for the matching lines, then the lobby for the summary.
