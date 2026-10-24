---
date: 2026-10-24
status: open
area: sessions / discord / audit
---

# An interrupted session cannot be ended after a restart, and the refusal leaves no trace

## Question

After the live bot's restart (`stop`, then `serve --discord live`), the owner's End session button
and a typed `/session end` for the interrupted s000a produced no audit row and no `serve.log` line.
The CLI `session end s000a` then failed with `error: session-closed: s000a is interrupted`. Before
the restart the same button was audited `action-refused` (the base-36 id bug, fixed in d662814).
Where is the request dropped? Does the design say `end` works on an interrupted session? Is there
a supported way to hand off such a session today? And (the owner) which inbound paths leave no
trace at all?

## What was tried

Code read at 0aeaeaf: `internal/discord/gateway.go`, `adapter.go`, `internal/session/actions.go`,
`turn.go`, `service.go`; blame on the refusals. The live bot, its root and s000a were not touched.

1. Dry-run repro, binary built outside `bin/`, from `<repo>`:
   `B=<scratchpad>/broodforge.exe; $B scratch run -- bash <scratchpad>/restart-end2.sh`. The script
   (`MSYS_NO_PATHCONV=1`, or Git Bash rewrites `"/session end"` into a path) serves `--discord
   dry-run`, starts s0001 against `fixtures/engine/basic`, sends one turn, commits one file in the
   tree with the Haiku co-author trailer, `stop`s, serves again, then runs `discord inject click
   --custom-id bf:s0001:end`, `discord inject command --channel s0001 "/session end"`, `session end
   s0001`, `worktree handoff s0001`, with `audit --all` and `session list` between.
2. Live path: a throwaway test (not kept) on `gateway_test.go`'s `newLiveHarness`. It inserts
   s000a `interrupted` with a venue whose topic marks it, runs `sweep`, then `onInteraction` with the
   End button and with the `/session end` interaction, and dumps the fake REST calls, the gateway
   log and the count of `action-refused` rows. Control: a click with custom id `bf:zz`.

## Result

1. **Where it stops (root cause).** Both routes reach the session service, and the service refuses
   without auditing. `Service.live` gets an entry only in `Start` (`internal/session/service.go:501`).
   Nothing loads an interrupted session back at start-up. So after a restart `acting` finds no
   live session (`internal/session/actions.go:69-71`) and returns `closedSession`
   (`internal/session/turn.go:46-59`): `session-closed: <id> is interrupted`, with no audit row.
   That path dates from P10 (b), 4ec949f. P12b's `switchOpenLocked` (`actions.go:116-122`) gates only
   mode, model and effort and is not involved. Even with the session in memory, `end` refuses an
   interrupted state at `actions.go:379-381` (also 4ec949f). The CLI hits the same refusal: it is a
   control-socket client of the same `Service.End`.
2. **The Discord side is answered, not dropped.** The sweep binds an interrupted session's venue
   (`gateway.go:392-394`). In the live harness, `sessionOf(venue)` returns s000a, and both
   interactions are acknowledged and answered with an ephemeral reply. Click: `CreateInteractionResponse
   {"type":6}`, then `CreateFollowupMessage` `This session has ended.` (`gateway.go:805-811`). Slash:
   deferred `{"type":5,"data":{"flags":64}}`, then `UpdateInteractionResponse` `This session has
   ended.` (`gateway.go:793-795`). `venueRefusal` maps every `session-closed` to that text
   (`adapter.go:396-397`). The gateway log was empty. The `action-refused` count was 0, and 1 after the
   control click. So the owner most likely got an ephemeral "This session has ended." that is
   false (the session is interrupted, not ended). The absence of a row and a log line is by
   construction: the service does not audit this refusal, and `serve.log` gets only failed REST
   calls (`gateway.go:234-238`). Candidates ruled out: the sweep does bind the venue, the
   interaction is acknowledged, and the handler does not return before answering.
3. **Dry-run repro output** (classes):
   `inject click bf:s0001:end` gives `error: session-closed: This session has ended.`, exit 1, no row.
   `session end s0001`, with and without `--as`, gives `error: session-closed: s0001 is interrupted`,
   exit 1, no row. `inject command --channel s0001 "/session end"` gives `error: not-a-venue`, audited.
   That one is a dry-run artifact: dry run binds the venue only in `SessionStarted`
   (`adapter.go:227-229`) and has no sweep, so after a restart the venue is unbound. Live binds it (2).
   `serve.log` held only the two `serving` lines.
4. **The design contradicts itself.** For `end` working on an interrupted session:
   `03-architecture.md` Shutdown, "`end` still runs the handoff, no new turns", and
   `04-ux.md:248`'s interrupted notice, "/session end hands off what exists". For the refusal: the
   P12b text of `04-ux-session-controls.md:87-89` ("after a restart ... every action is answered
   `This session has ended.`"), `05-debug-interface-session.md:72-74`, which says the refusal comes
   with no audit row, and `05-debug-interface-discord.md:185-186`. Those three describe the code as
   it is. The code is at odds with 03 and 04-ux, and the no-audit carve-out is at odds with 04's
   Conventions ("every refusal is ... an `action refused` audit event (N-17)").
5. **Supported path today: `worktree handoff <id>`.** It ignores the session state. In the repro it
   printed `pushed  origin  refs/sessions/s0001:refs/heads/broodforge/<instance>/s0001  1`, with audit
   `push attempted` and `push done`, and set the worktree `removed`. Without the trailer it was
   refused `handoff-refused-attribution` (audited `handoff-refused`). But the session stays
   `interrupted`: no `session-ended` row, no lobby summary or transcript, and live the venue is never
   deleted, because the sweep deletes only ended sessions' venues (`gateway.go:392-402`). For s000a,
   whose changes reached `main` by hand, this pushes the same changes again under other hashes on
   the session branch (post-mortem item 1). Not run on the live root.
6. **Side gaps.** 03 Startup step 10 (the interrupted notice in the venue) has no code: there is no
   04 notice text in `internal/`. On `session-interrupted` the View sets `closed` and posts no
   header edit (`render.go:190-192`), so the End session button stays on the pinned header after a
   restart and leads to the false reply in 2.

### Inbound paths that leave no audit row and no log line

Gateway (`internal/discord/gateway.go`):
- `onEvent` 609-622: other event types, and a failed type assertion, are dropped silently.
- `onMessage` 646-649 with `inboundMessage` 673-675: bots, webhooks and other guilds; and the
  `default` of 655-667, a `none` reply: a message in an unconfigured channel, or a lobby message from
  someone who may not start (`adapter.go:345-354`).
- `onInteraction` 759-762: another guild is answered, but not audited or logged. At 766-773, a
  non-slash command interaction (a user or message context command) or failed assertion; at
  797-800 a component that is not a `ComponentInteraction`; at 814-817 a modal that is not the
  API-key form; and any other interaction type: all return without acknowledging, so Discord shows
  "This interaction failed", and nothing is written.
- Every successful REST call is unlogged (`warn` 234-238 logs failures only).

Session service, refusals returned without `s.refuse`:
- `closedSession` (`turn.go:46-59`), through `acting` (`actions.go:69-71`; Answer, Interrupt, End,
  SetMode, Invite, `model.go:113`, `timeout.go:104`) and `Submit` (`turn.go:70-72`): `session-closed`
  and `unknown-session`.
- `shutting-down` (`actions.go:66-67`, `147-148`; `turn.go:67-68`; `service.go:368-369`).
- `no-pending` and `not-pending` (`actions.go:150-155`), `no-running-turn` (`actions.go:244-246`),
  and `end` on a closed live session (`actions.go:379-381`). `bad-args` in Answer (131, 136).

## Next

Fix shapes only; the architect decides, through the proposal loop where a design doc changes.

- **`end` on an interrupted session (per 03).** At start-up, load each interrupted session into
  `Service.live` as a dormant entry: state `interrupted`, driver and channel from the store, no
  engine. Let `end` accept `interrupted` (not the idle end). It sets `ending`, runs `cfg.Handoff`,
  then `closeSession` with `before = interrupted`. On a refused handoff it returns to `interrupted`,
  not `waiting-for-input`: no new turns. Every other action on it stays refused. Call the `Started`
  hook (or a sibling) for the dormant entry, so the adapter has a View: the summary, the transcript
  and the venue deletion then follow `session-ended` as they do today. Open choice: the transcript
  before the restart is not in memory (the event log is). Either the summary covers the post-restart
  part only, or the View is seeded from stored events. Alternative: `worktree handoff` also closes
  an interrupted session (`session-ended`), with no summary in Discord. Tests: `session end` and the
  End click on an interrupted session after `stop` and `serve`, plus a dry-run sweep or rebind so
  `inject command --channel <id>` reproduces the live path.
- **Wording (owner).** An interrupted session's refusals should not say `This session has ended.`
  The interrupted notice of step 10 needs building. Both texts are the owner's.
- **Logging (owner's ask).** (a) Audit every refusal, per N-17 and 04 Conventions: route the
  service's bare refusals above through `s.refuse` with reasons `closed`, `unknown-session`,
  `shutting-down`, `no-pending`, `no-running-turn`, and drop the no-audit carve-out from
  `05-debug-interface-session.md`. (b) One `serve.log` line per inbound event at the gateway,
  always on, since human actions are few: `inbound <message|command|click|modal> user=<id>
  channel=<id> <route> -> handled|refused <code>|dropped <reason>`. The route is the command path,
  or a click's parsed kind and session (never the raw custom id, never message text: N-12).
  Drops (bots, other guild, unknown type) go to the log only, not the audit, which is for
  security-relevant actions. (c) Acknowledge every interaction, unknown ones included, with an
  ephemeral `not 04` line, so none shows "This interaction failed". This shape belongs in
  `05-debug-interface-discord.md` (Observe) once decided.
- s000a: ending it now means `worktree handoff s000a` (state stays interrupted) or waiting for the
  fix. The owner's call, given the changes already on `main`.
