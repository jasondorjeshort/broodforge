# 0026. Interrupted sessions are held after a restart; every refusal and every inbound event leaves a trace

Date: 2026-10-24
Status: accepted

Record: no proposal. The owner approved the four parts on 2026-10-24 from
`../../findings/2026-10-24-interrupted-session-cannot-be-ended-after-a-restart.md`, and the texts and
the header choice below the same day after the design review. Not a major choice in the agreement's
sense: no protocol, storage, API surface or dependency changes; the audit log gains reason values only,
the control wire nothing, the session row nothing; the spawn backend gains one read-only method.

## Context

After the live bot's restart the owner could not end the interrupted session s000a: the End
session button, `/session end` and the CLI's `session end` were all refused `session-closed`,
Discord showed the false `This session has ended.`, and neither an audit row nor a log line
recorded any of it. The finding traced it: `Service.live` is filled only by `Start`, so after a
restart no interrupted session is in memory and `acting` falls through to a store read and a
bare refusal (`internal/session/actions.go`, `turn.go`); `end` refused `interrupted` even when
held; the service returned a dozen refusals without `s.refuse`; the gateway logged failed REST
calls only and returned without acknowledging several interaction shapes. The design was split:
`03-architecture.md` (Shutdown) and `04-ux.md` (the interrupted notice) promised `end` on an
interrupted session, while the P12b texts in `04-ux-session-controls.md`,
`05-debug-interface-session.md` and `05-debug-interface-discord.md` described the code as it
was, with a carve-out from 04's rule that every refusal is audited (N-17).

Constraints. The event log is in memory: the transcript before a restart is gone (R-18, resume,
is later). The pinned header's Discord message id is in memory too, so a restarted bot cannot
edit it. The sweep binds an interrupted session's venue, but only inside the gateway's open
(`gateway.go`), after the coalescer has started sending (`serve.go`): a post offered before it
would find no venue. The bot never opens a session's tree itself (ADR 0022): a commit count goes
through the spawn backend. The bot's own posts come back as message events, about one a second
per session. Human actions are few, so a line per inbound event costs nothing. The audit log is
for security-relevant actions (N-17), not for traffic or reads.

## Decision

1. **Held sessions.** At start-up, after reconciliation and before the control socket listens,
   the session service loads every `interrupted` row into memory as a held entry: the row's
   facts, its `git.Session`, the stored turn count and usage sums, the tree's commit count read
   through the spawn backend's new `WorktreeCommits` (null when unreadable), no engine, no
   timers, a fresh event log opening with `session-interrupted` (summary `restart`, the header's
   facts as its `start` payload), and the `Started` hook, so the adapter follows it as any
   session. A channel no longer configured keeps the row's ids with the `answer` list `driver`
   (an empty list permits nobody). Loading writes nothing; `serve` prints `held  <ids>`. On a
   live instance the coalescer starts only after the gateway has opened and swept, so the held
   notices reach bound venues. On a held session every action but `end` is refused
   `session-closed: <id> is interrupted`; the closed check comes before every other check, the
   pending, running and driver checks included (an answer and a Stop on a closed session are
   now `session-closed`, not `no-pending` and `no-running-turn`). `end` accepts `interrupted`,
   runs the handoff and closes the session as 03 says; a refused handoff puts it back to
   `interrupted`, never to waiting-for-input, with 04's interrupted-session sentence. Two escapes,
   in this order, keep every held session endable: `refs/sessions/<id>` gone (a `worktree handoff`
   finished earlier, which removes the tree and the ref) ends with the outcome that handoff
   recorded in `end_outcome` (a URL as `pull-request`, a branch as `pushed`, null as `none`),
   commits unknown; else the ref present but the tree gone (row `removed`, or path absent) ends
   with `tree-missing` and no handoff, the ref left for the owner. The
   venue's summary, transcript and deletion follow `session-ended` as for any session. The
   summary's counts come from the store; the transcript holds what the bot saw since the
   restart, with its heading taken from the held opening event. Rejected: closing an
   interrupted session from `worktree handoff` alone (no summary, no venue deletion, and the
   host CLI would end what Discord cannot); seeding the transcript from audit rows (they hold
   decisions, not the agent's text, and a half transcript reads as whole).
2. **Every refused action is audited.** Every refusal the service returns to an action goes
   through `s.refuse` where the action is refused (`acting`, `Submit`, `end`'s own closed
   branch), the reason the refusal's code: `closed`, `unknown-session`, `shutting-down`,
   `no-pending`, `not-pending`, `no-running-turn`, `bad-args`. The session column holds the id the
   request named (no foreign key), the actor is `--as` or the host. The lookup the reads share
   stays bare: a refused `tail`, `show`, `transcript` or bare `/model` is not an action and writes
   no row; nor does the idle timer finding its own end void. The P12b carve-out ("with no audit
   row") is gone.
3. **One log line per inbound Discord event**, always on, on the bot's stderr (the `serve.log` a
   repro keeps): time, kind, user id, channel id, the route, the outcome `handled`, `refused
   <code>` or `dropped <reason>`. Never message text, a command's options, a modal's field or a
   raw custom id (N-12): a command's route is its path, a click's the parsed kind and session.
   Drops (other bots, webhooks, another guild, unknown interaction shapes, a message nothing
   answers) are logged, not audited. Not logged: the bot's own echoed posts and edits, and the
   dispatches that are neither a message nor an interaction. Format:
   `05-debug-interface-discord-live.md`, Inbound log and held sessions.
4. **Every interaction is acknowledged.** An interaction the bot does not handle (another guild,
   a non-slash application command, an unknown component or modal, any other type) gets an
   immediate ephemeral `Broodforge cannot handle this interaction.` (owner-approved), so Discord
   never shows "This interaction failed"; the one shape that cannot take a message,
   autocomplete, is not registered and is only logged if it arrives.
5. **Wording** (the owner, 2026-10-24): a refusal on an interrupted session, a plain message or
   any action but `end`, says `This session was interrupted by a restart. /session end hands off
   its work.`; `This session has ended.` stays for ending and ended sessions. The adapter reads
   the state from the session's row, not from the refusal's text. The notice's `1 commit` and
   `Its tree is kept.` and `Its tree is missing.` (the tree gone) are approved. The summary lines
   for the outcomes `tree-missing` and `none` after an earlier handoff are in `04-ux.md`
   (Ending a session), owner-approved 2026-10-24. A handoff refused on a held session ends `The session was interrupted, so no
   agent can amend them; the owner can fix the tree and retry with broodforge worktree handoff
   <id>.` (approved; `04-ux.md`, Failure flows).
6. **The header after a restart is left as it is** (owner-approved). Its End session button now
   does what it says, and a pick from its menus gets the interrupted wording; no edit on
   `session-interrupted`, and no REST search for the pinned message.

Deferred, with triggers: resuming the conversation (R-18) when the engine's `--resume` across a
bot restart is tested; a time on the `warning: discord:` lines, with the first live debugging
session that needs to order them against the inbound lines; `worktree handoff`'s own unaudited
`already-handed-off` refusal, with the next git pass; a line per REST call and the trace over the
control pipe (`../../ideas/2026-10-22-discord-layer-trace-for-debugging-the-live-bot.md`).

## Consequences

- `end` works from Discord and the CLI on every interrupted session, and the lobby gets the
  record; s000a can be ended by `session end s000a` once P12c lands (its duplicate push is the
  owner's call, post-mortem item 1). A tree gone or a branch already pushed no longer leaves a
  session that can never be ended.
- `session tail` on a held session shows its post-restart events instead of `session-closed`;
  `session end` after a restart appears in `audit` whether it succeeded or not. Callers that
  expected `no-pending` or `no-running-turn` on a closed session get `session-closed`.
- `audit --kind action-refused` grows: `bad-args` from a typo now writes a row. Accepted: human
  actions are few and the owner asked for every refusal.
- The dry-run repro gains a restart: the held session's `Started` hook rebinds its venue, so
  `discord inject command --channel <id>` reaches the live path after `stop` and `serve` without
  a sweep.
- A live bot's first posts wait for the gateway to open (at most its 60 s timeout); a session
  started over the socket in that window gets its venue a little later.
- A held session costs one `live` entry and one goroutine (its follower) per interrupted row for
  the bot's lifetime; ended by `end`, never collected otherwise. Many stale interrupted rows
  would show in `status` first.
- The spawn backend's surface grows by one method; the Linux backend's stub is one more line
  P14 fills.
- Supersede this when sessions resume after a restart (R-18): a held session would then be a
  waiting one, and the interrupted wording would go.

Predictions (the trigger scan reads these).
- Prediction, check by 2026-11-30: no live session is refused `session-closed` without an
  `action-refused` row; `audit --kind action-refused` and the inbound log agree on every refusal
  the owner reports.
- Prediction, check by 2026-11-30: the owner has not seen "This interaction failed" from the live
  bot since P12d landed.
- Prediction, check when the first held session is ended from Discord: the summary's turn and
  token counts equal `session show` and `usage --session <id>`, the notice reached the venue
  (no `has no venue` warning in `serve.log`), and the venue is deleted ten minutes after.
