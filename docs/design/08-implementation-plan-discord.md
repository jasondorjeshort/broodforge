# Implementation plan: the Discord adapter

This file continues `08-implementation-plan.md` under the same conventions: the pass format, the size
words, the standing rules and the `[no-check: ...]` rule. It holds the Discord passes, P11 to P12b,
moved here by topic when 08 reached the size cap (P12's deleted file gained its `[no-check]` marker and
P12a's Implements line now names `04-ux-session-controls.md`); P13's and P14's dependencies on them are unchanged.

## P11. Discord dry-run: renderer, coalescer, outbox, transcript, panel (medium; owner sees)

Implements: `04-ux.md` (every screen and text); `03-architecture.md` (`discord` row; Data flow steps
1 and 6); ADR 0023 (Libraries: Discord); `05-debug-interface-discord.md` (The Discord adapter; `session
outbox`, `transcript`, `panel`). Files: `internal/discord/` (03's `discord` row without the gateway:
renderer, coalescer with the injectable clock and a global budget of 50 requests per second per bot
beneath the per-session one, mask, custom ids, command parsing, `inject`; disgo's types appear here
and nowhere else); `internal/cli/{discord,panel}.go`; `tests/discord/burst.jsonl`; the two repro
scripts of P10 gain their `session outbox` checks (the buttons post, the timeout trace line). Proof:
`discord render --events tests/discord/burst.jsonl --clock fake --expect 'edits<=1/s'`; `discord
inject message --channel 100 --user 1001 --roles 10 "sk-ant-..."` yields the refusal and an
`action-refused` event with no key text; a scratch session shows 04's working message, footer,
buttons post and summary in `session outbox`, which the owner reads. Depends on: P10.

## P12. Discord live: gateway, commands, buttons, venues, the relay (medium; owner sees; needs his guild)

Implements: `04-ux.md` (Registration through Starting; Credentials); `03-architecture.md` (Startup 9
and 10; Failure modes, `discord` row); `05-debug-interface-discord.md` (The Discord adapter: `check`, `serve
--discord live`; `credential login --relay`, off by default). Files: `internal/discord/gateway.go`
(the disgo client pinned in `go.mod`, slash commands, interactions, the private-channel venue, the
notice channel, the DM relay; resume and the rate limiter are disgo's; the REST calls used are
listed in one place in this file, the surface a swap to arikawa must cover); `discord check` in
`internal/cli/discord.go`; `internal/deps/deps.go` deleted [no-check: deleted by this pass]. Proof: `discord check --guild <id>`
lists every configured channel `present`, `perms:ok`; `serve --discord live` on a scratch-shaped
root with the fake: the owner registers, starts, sends, clicks Allow and ends from Discord while an
agent reads `session tail` and `audit`; with P7's real engine this is the vision's first usable
version. Depends on: P11; the bot token and a test guild.

## P12a. Model and effort from Discord (small; owner sees the header line and replies)

Implements: R-29; `04-ux-session-controls.md` (Model and effort); `05-debug-interface-discord.md` (Model and effort);
`05-debug-interface-session.md` (`session start --model`, `session model|effort`, the `model:` line).
Files: `internal/policy/roles.go` (fable; aliases); `internal/store/migrations/0002-session-model.sql`
[no-check: written in this pass] (`model`, `effort`); `internal/audit/audit.go`; `internal/session/`
(`StartRequest.Model`, `SetModel`, `SetEffort`, the spawn); `internal/discord/` and `internal/cli/session.go`
(commands, choice lists, `/model`, wording, the header edit); `internal/engine/claudecode.go` only if the
pinned engine has an effort flag (the coder checks first; without it, effort, commands and column, is out
of this pass). Proof: `session start --channel 100 --as 1001 --roles 10 --credential cred-scratch --model
sonnet`; `session show s0001` prints `model: claude-sonnet-5-5`; `discord inject command --channel s0001
--user 1001 --roles 10 "/model"` prints the `Model:` line; `"/model fable"` yields `model-changed` in
`session tail`, an `edit` of the pinned header in `session outbox` and `claude-fable-5-1` in the next
`received-<n>.json`; `"/model gpt"` is `bad-args` plus `action-refused`. Depends on: P12.

## P12b. The header's menus (small; owner sees the header)

Implements: `04-ux-session-controls.md` (The header's menus); `05-debug-interface-discord.md` (The header's
menus; `session outbox --json`'s `selects`; `inject click --value`; Model and effort's `mode-changed` header
edit); `05-debug-interface-session.md` (Session lifecycle: the `session-closed` refusal of a switch);
`03-architecture.md` (Data flow: the custom ids). Files: `internal/session/model.go` and
`internal/session/actions.go` (`setColumn` and `SetMode` refuse a session whose state is ending, ended or
interrupted, read under `l.mu` before the write, with `CodeSessionClosed` and `end`'s text `<id> is <state>`,
through `s.refuse` with the reason `closed` so `action-refused` holds the row; one function the three share;
the `04, Policy mode` comment now cites `04-ux-session-controls.md`); `internal/discord/outbox.go`
(`Select{CustomID, Placeholder string; Options []Option}` and `Option{Label, Value string; Default bool}`;
`Post.Selects`, `--json`'s `selects`; `messageCreate` and `messageUpdate` build one action row per select, in
the post's order, above the button row, through disgo's `NewStringSelectMenu` with `min_values` and
`max_values` left at one; the outbox text line unchanged); `internal/discord/coalesce.go` (the merge of an
edit into a queued create or edit copies `Selects` with `Body` and `Buttons`); `internal/discord/customid.go`
(`IDModel`, `IDEffort`, `IDMode`: the three-part forms `bf:<session id>:model|effort|mode`, parsed and printed
by the two functions there, the session id checked as `stop`'s is; its `Button custom ids` comment now cites
03's `Custom ids`); `internal/discord/render.go` (`headerSelects(headerState)`: the model options from
`policy.ModelAliases()` labelled by `session.ModelShort` of each alias's id, the default per 05, else the
`Model: <id>` placeholder, `Model: the engine's own` for none; the effort options from `policy.Efforts` labelled
by the level, no default and `Effort: the engine's own` until set; the mode options `asks first` and `acts
freely (within reach)` with values `asks-first` and `acts-freely`; the header's create and every edit carry
them; `EventModeChanged` edits the header as `switched` does for a model, its post still `modeChangedText`;
`sessionEnded` edits the header with no buttons and no selects; a View mutex held by `Handle` and by the new
`Resync()`, which re-posts the header from the View's own `start` and `hdr` and does nothing once the View
has handled `session-ended` or `session-interrupted`); `internal/discord/adapter.go` (a `views` map, written
in `SessionStarted` and deleted under the adapter's lock when the follow returns, before `Done`;
`Inbound.Values`; `click` on the three kinds: exactly one value, else `bad-args` audited with the action
`click`, then the function the slash branch calls, the model, effort and mode branches of `slash` moved into
three functions both routes share, each taking the route's audit action; after a service refusal other than
`session-closed`, `Resync()` on the session's View, nothing without one; `venueRefusal` answers
`session-closed` with 04's `This session has ended.`; the `Button custom ids` comment now cites 03's `Custom
ids`); `internal/discord/gateway.go` (`onInteraction`: a `StringSelectMenuInteractionData`'s `Values` into
`Inbound.Values`; the ack, follow-up and `restAPI` unchanged; the same comment fix);
`internal/discord/text.go` and `internal/policy/roles.go` (the comments citing 04's Policy mode, Model and
effort and Idle timeout now name `04-ux-session-controls.md`; no code change); `internal/cli/discord.go`
(`inject click --value <v>`); `tests/repro/full-session.sh` (the header's `selects`). Proof: on a scratch root,
whose channel `100` sets no `model`, after `register 1001 --name "Ada Example" --email ada@example.com` and
`session start --channel 100 --as 1001 --roles 10 --credential cred-scratch --mode policy --fixture basic`
(no `--model`), `session outbox s0001 --json` shows the pinned create with `selects` in the order model,
effort, mode, the first two with no `default` and 04's placeholders, the mode's default `acts-freely`, then
the `End session` button; `discord inject click --custom-id bf:s0001:effort --value high --user 1001 --roles
10` prints `none  effort from turn <n>: high`, `session tail` gains `effort-changed`, and the header's next
`edit` marks `high` default with the model menu unchanged; `register 1002 --name "Bob Example" --email
bob@example.com`, then the same click `--user 1002 --roles 10` exits 1 with `error: refused: effort needs the
driver, Ada Example`, `audit --kind action-refused` holds the row with the action `effort`, and the outbox
holds an `edit` of the header equal in body, selects and buttons to the one before it; `--value turbo` is
`bad-args` with 04's `Not an effort:` text and an `action-refused` row with the action `click`; a menu id
without `--value`, and `--value` with `bf:s0001:end`, are `bad-args`; `--custom-id bf:s0001:mode --value
asks-first` posts the mode switch's text and the header's `edit` marks `asks-first`; after `session end
s0001` the header's last `edit` has no `selects` and no `buttons`, the same mode click `--user 1001` exits 1
with `error: session-closed: s0001 is ended`, `audit --kind action-refused` holds it with the reason `closed`,
`audit --kind mode-changed` and the outbox gain nothing, and `session mode s0001 prompt --as 1001` and
`session model s0001 opus --as 1001` are refused the same; `go test ./internal/discord -run
'TestSelect|TestCoalesceSelects|TestLive'` and `go test ./internal/session -run TestSwitchClosed` pass: the
select test feeding a recorded string-select interaction to `onInteraction` and seeing the deferred update,
then an `UpdateMessage` of the header with one action row per menu and the button row, and for a non-driver
the follow-up then the unchanged header; a `Resync` and a `session-ended` on one View in either order leaving
the header's last edit without components; the coalescer test merging a switch's edit into the queued create
and the end's edit into a queued switch's edit, the later components winning both times; the session test the
three setters on an ended session, the row, the audit and the events unchanged. Owner step, under `serve
--discord live` with the fake, after the dry-run proof: a non-driver picks a model and says whether the menu
returns to the true value; the driver picks the option already shown and an agent reads `session tail` for a
`model-changed`; both results go in a finding, and a reset that fails turns the re-sync into two edits
(`05-debug-interface-discord.md`, The header's menus). Does not: touch the engine's permission mode; add an
audit kind or an event type; add a level to `policy.Efforts` (the owner's open question); add a label row, an
embed or a second message. Depends on: P12a.
