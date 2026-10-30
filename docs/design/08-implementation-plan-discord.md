# Implementation plan: the Discord adapter

This file continues `08-implementation-plan.md` under the same conventions: the pass format, the size
words, the standing rules and the `[no-check: ...]` rule. It holds the Discord passes, P11 to P12d,
moved here by topic when 08 reached the size cap (P12's deleted file gained its `[no-check]` marker and
P12a's Implements line now names `04-ux-session-controls.md`); P13's and P14's dependencies on them are unchanged.

## P11. Discord dry-run: renderer, coalescer, outbox, transcript, panel (medium; owner sees)

Implements: `04-ux.md` (every screen and text); `03-architecture.md` (`discord` row; Data flow steps
1 and 6); ADR 0023 (Libraries: Discord); `05-debug-interface-discord.md` (The Discord adapter; `session
outbox`, `transcript`, `panel`). Files: `internal/discord/` (03's `discord` row without the gateway:
renderer, coalescer with the injectable clock and a global budget of 50 requests per second per bot
beneath the per-session one, mask, custom ids, command parsing, `inject`; disgo's types appear here
and nowhere else); `internal/cli/{discord,panel}.go`; `tests/discord/golden/burst/events.jsonl` (P32 moved
it from `tests/discord/`); the two repro
scripts of P10 gain their `session outbox` checks (the buttons post, the timeout trace line). Proof:
`discord render --events tests/discord/golden/burst/events.jsonl --clock fake --expect 'edits<=1/s'`; `discord
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
effort and Idle timeout now name `04-ux-session-controls.md`; `policy.Efforts` gains `xhigh` between `high`
and `max`, per the owner); `internal/cli/discord.go`
(`inject click --value <v>`); `tests/repro/full-session.sh` (the header's `selects`). Proof: on a scratch root,
whose channel `100` sets no `model`, after `register 1001 --name "Ada Example" --email ada@example.com` and
`session start --channel 100 --as 1001 --roles 10 --credential cred-scratch --mode policy --fixture basic`
(no `--model`), `session outbox s0001 --json` shows the pinned create with `selects` in the order model,
effort, mode, the first two with no `default` and 04's placeholders, the mode's default `acts-freely`, then
the `End session` button; `discord inject click --custom-id bf:s0001:effort --value high --user 1001 --roles
10` prints `none  effort from turn <n>: high`, `session tail` gains `effort-changed`, and the header's next
`edit` marks `high` default with the model menu unchanged; `register 1002 --name "Bob Example" --email
bob@example.com`, then the same click `--user 1002 --roles 10` exits 1 with `error: refused: Only the driver,
Ada Example, can change the effort.`, `audit --kind action-refused` holds the row with the action `effort` and
the service's own text `effort needs the driver, Ada Example`, and the outbox
holds an `edit` of the header equal in body, selects and buttons to the one before it; `--value turbo` is
`bad-args` with 04's `Not an effort:` text and an `action-refused` row with the action `click`; a menu id
without `--value`, and `--value` with `bf:s0001:end`, are `bad-args`; `--custom-id bf:s0001:mode --value
asks-first` posts the mode switch's text and the header's `edit` marks `asks-first`; after `session end
s0001` the header's last `edit` has no `selects` and no `buttons`, the same mode click `--user 1001` exits 1
with `error: session-closed: This session has ended.`, `audit --kind action-refused` holds it with the reason
`closed` and the service's text `s0001 is ended`, `audit --kind mode-changed` and the outbox gain nothing, and
the typed `session mode s0001 prompt --as 1001` and `session model s0001 opus --as 1001` are refused with the
service's text, `error: session-closed: s0001 is ended`; `go test ./internal/discord -run
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
audit kind or an event type; add a label row, an
embed or a second message. Depends on: P12a.

## P12c. Held interrupted sessions: `end` after a restart, the notice (medium; owner sees the notice and the summary)

Implements: ADR 0026 (points 1, 5's state read, 6); `03-architecture.md` (Startup 7 and 10; Shutdown's "`end`
still runs the handoff"); `04-ux.md` (Ending a session's restart paragraph; Failure flows `interrupted (R-17)`
with its approved variants, `handoff-refused, interrupted session`); `04-ux-session-controls.md` (The header's
menus, last paragraph); `05-debug-interface-session.md` (Session lifecycle: `end` on a held session and its two
escapes; Held sessions); `05-debug-interface.md` (Bot lifecycle: `held  <ids>`); `05-debug-interface-discord-live.md`
(The live gateway: the coalescer after the gateway opens, `TestLiveHeld`; Inbound log and held sessions: the
held opening, the notice, `Transcript` and `headerOf`). Resolves
`../findings/2026-10-24-interrupted-session-cannot-be-ended-after-a-restart.md` in part (P12d sets
`promoted_to`). Files: `internal/spawn/spawn.go` (`Backend.WorktreeCommits(w Worktree) (int, error)`: the
`rev-list --count` of `WorktreeBundle` without the bundle; read `docs/reference/go-on-windows.md` and
`docs/reference/go-dependencies.md` first, `.claude/rules/go.md`), `internal/spawn/worktree.go` (`SameAccount`'s,
sharing the count with `WorktreeBundle`), `internal/spawn/bwrap.go` (the `held()` stub, as `WorktreeBundle`'s);
`internal/git/` (`Commits(ctx, b, s)`: the backend's count for the session's tree over its base, the one
caller of the new method); `internal/session/service.go` (`Service.LoadInterrupted(ctx) ([]string, error)`
building 05's held entry, `l.git` included, the unconfigured channel's `answer` list `driver`;
`Config.Commits func(ctx, id) (*int, error)`, nil or an error leaving `start.commits` null, the tree found
gone flagged for the notice); `internal/session/events.go` (`StartInfo.Commits *int`, `json:"commits,omitempty"`,
and `TreeMissing bool`, set on a held start only; the `session-interrupted` summaries `shutdown`,
`reconciliation`, `restart` as named constants); `internal/session/actions.go` (`end` accepts `interrupted`
with `before` `interrupted`; a refused handoff returns it to `interrupted` with no timer and the
`handoff-refused` event's `Text` carrying 04's interrupted-session sentence; the two escapes in 05's order: the
ref gone first (`end_outcome` mapped to `pull-request` with the URL, `pushed` with the branch, or `none`,
no handoff run; `endInfo` fills `EndInfo.URL` from the stored value and `Branch` from `l.git`, no new field),
then the ref present and the tree gone (`tree-missing`, no handoff); `Invite`, `Answer` and `Interrupt` take the shared closed check first so nothing but
`end` acts on a held session; `SetIdleTimeout`'s closed check before its driver check, `internal/session/timeout.go`);
`internal/cli/serve.go` (`LoadInterrupted` after `session.Reconcile` and before `control.Listen`, the stderr
line `held  <ids>` or `held  -`; `Config.Commits` from the worktree service; on a live instance
`b.adapter.Run()` moves after `b.gateway.Open`, a dry run unchanged); `internal/cli/worktree.go` (the count for
`Config.Commits`); `internal/discord/text.go` (`interruptedNoticeText(commits *int, missing bool)`: 04's text,
`1 commit`, `Its tree is kept.`, `Its tree is missing.`; `handoffRefusedHeldText`; `interruptedRefusalText`, 04's
`interrupted refusal` row; `summaryText`'s two drafted lines for `tree-missing` and `none` after an earlier
handoff, held by 04 (Ending a session, owner-approved 2026-10-24)); `internal/discord/render.go`
(`EventSessionInterrupted` with `Start`: the held opening, closed, the notice; without `Start` as today;
`headerState.apply` takes `Start` from it too); `internal/discord/transcript.go` (the heading from a
`session-interrupted` carrying `start`); `internal/discord/adapter.go` (`venueRefusal` reads the state through
`svc.Show`: `interrupted` answers 04's `interrupted refusal` text, `ending` and `ended` the old one); the
comments at `internal/discord/gateway.go`'s head and `gateway_test.go`'s head now name
`05-debug-interface-discord-live.md`; `internal/discord/gateway_test.go` (`TestLiveHeld`);
`internal/session/service_test.go` or `ending_test.go` (`TestHeld`); `tests/repro/restart-end.sh` [no-check:
written by this pass] and its `tests/README.md` row. Proof: `CGO_ENABLED=0 go build ./...`, the check line,
`CGO_ENABLED=1 go test -race ./...`; `scratch run -- bash tests/repro/restart-end.sh` exit 0 (`MSYS_NO_PATHCONV=1`
under Git Bash): `serve`, `register 1001 ...`, a start on channel `100` against `basic`, one turn, a commit in
the tree with the Haiku co-author trailer, `stop`, `serve` again whose stderr has `held  s0001`; `session tail
s0001 --all` opens with `session-interrupted  restart`; `session outbox s0001 --all` holds a `create` beginning
`Broodforge restarted; this session was interrupted. Its tree is kept with 1 commit`; `session transcript
s0001` opens with `# Session s0001`; `discord inject message --channel s0001 --user 1001 --roles 10 "hi"` exits
1 with `error: session-closed: This session was interrupted by a restart. /session end hands off its work.`
(the audit row for it and the stderr line are P12d's); `discord inject click --custom-id bf:s0001:end --user 1001 --roles 10`
prints `none  s0001 ended`, `audit --kind session-ended --all` holds `"state":"interrupted"` and
`"reason":"end"`, the outbox gains the venue's summary, the lobby's copy and its `attach`; `session end s0001
--as 1001 --expect session-closed` exits 0 (the match). `go test ./internal/session -run TestHeld` and `go test
./internal/discord -run 'TestLive|TestLiveHeld'` pass as `05-debug-interface-discord-live.md`'s Reproduce
lists them (the gone tree with its ref, a session `worktree handoff` finished earlier then ended `pushed`, the
null `end_outcome`, the sweep after a held load). Owner step, under `serve
--discord live` after a restart: `End session` on an interrupted session's header, then `audit` for the
`session-ended` row and the lobby for the summary; the first run on the live root is the owner's call for
s000a (its branch would be pushed again). Does not: resume a session's conversation (R-18); edit the pinned
header on `session-interrupted` or search for it over REST; add an audit kind, an event type or a control
command; change the wire format; audit the refusals P12d routes. Depends on: P12b.

## P12d. Every refusal audited, the inbound log, every interaction acknowledged (medium; owner sees the refusal text)

Implements: ADR 0026 (points 2, 3, 4, 5); `04-ux.md` (Conventions; Failure flows `interrupted refusal`, `unknown
interaction`); `05-debug-interface-session.md` (Session lifecycle: the closed check first for every action,
every action's refusal audited where it is refused, reads and the idle void unaudited);
`05-debug-interface-discord-live.md` (Inbound log and held sessions: the line, the outcomes, the acknowledgement).
Sets the finding's `promoted_to` to ADR 0026. Files: `internal/session/actions.go` (`acting` audits
`closed` and `unknown-session` for the action it serves, taking the action name; `closedSession` stays bare for
`Follow`, `Events` and `Show`; `end`'s own closed branch through `s.refuse`; the idle timer's void branch
untouched and unaudited; `shutting-down`, `no-pending`, `not-pending`, `no-running-turn`, `Answer`'s two
`bad-args` and `SetMode`'s through `s.refuse`); `internal/session/turn.go` (`Submit`'s lookup miss, audited
`closed` or `unknown-session` like `acting`'s; its `shutting-down`, `bad-args` and `session-closed`, the latter
now behind the shared closed check); `internal/session/service.go` (`Start`'s `shutting-down` twice, its
`bad-args`, `unknown-fixture` from `resolveFixture` and the credential's `bad-kind`; `startModel`'s `bad-args`);
`internal/session/model.go` (`SetModel`'s and `SetEffort`'s `bad-args`); `internal/session/timeout.go`
(`SetIdleTimeout`'s `bad-args`); `internal/session/registration.go` (`Register`'s `bad-args`, action
`register`); the reason list of `05-debug-interface-session.md` gains `unknown-fixture` and `bad-kind`;
`internal/discord/text.go` (`unknownInteractionText`); `internal/discord/gateway.go` (one function every handler
returns into writes the `inbound` line to `GatewayConfig.Log`; `inboundMessage` tells the bot's own echo, not
logged, from another bot's or a webhook's, logged `dropped`; `onInteraction`'s other-guild, non-slash,
non-component, non-API-key-modal and other-type branches acknowledge with the immediate ephemeral answer and
log `dropped <reason>`; `onEvent`'s other dispatches pass as before, unlogged); `internal/discord/gateway_test.go`
(`TestInbound`; `TestOnEventRoutes` unchanged); `internal/session/*_test.go` (`TestRefusalsAudited`;
`prompt_mode_test.go`'s `Interrupt` on an idle open session still `no-running-turn`, now with a row);
`tests/repro/restart-end.sh` [no-check: P12c writes it] (the refusal and audit checks of the live doc's Reproduce). Proof: build, check
line, `-race` tests; `scratch run -- bash tests/repro/restart-end.sh` exit 0 with its new steps: after the
restart `discord inject command --channel s0001 --user 1001 --roles 10 "/model sonnet"` exits 1 with `error:
session-closed: This session was interrupted by a restart. /session end hands off its work.`, `inject message
--channel s0001 --user 1001 --roles 10 "hi"` the same, `audit --kind action-refused --all` holding both with
`"reason":"closed"` and the service's `s0001 is interrupted`; after the end, `session end s0001 --as 1001
--expect session-closed` adds a row with `"reason":"closed"` and `s0001 is ended`, `session end s9999 --expect
unknown-session` one with `"reason":"unknown-session"` and, in `audit --json`, `"session":"s9999"`; `session
tail s0001` on the ended session is `session-closed` with no new row. `go test ./internal/discord -run
'TestLive|TestInbound'`: one `discord: inbound` line per fed event with the outcome of the live doc's list,
the log free of the message's text and of every custom id, a `CreateInteractionResponse` for each of the nine
interactions fed, the unknown ones' with the ephemeral text; `go test ./internal/session -run
TestRefusalsAudited`. Owner step: after the pass, `serve.log` of the live bot shows one `inbound` line for a
click and for a message; no "This interaction failed" on a stray interaction. Does not: audit a drop or a
read; log message text, an option, a field or a raw custom id; log the bot's own echoes or non-message,
non-interaction dispatches; add an audit kind or an event type. Depends on: P12c.

