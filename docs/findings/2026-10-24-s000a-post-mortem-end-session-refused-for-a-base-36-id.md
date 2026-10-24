---
date: 2026-10-24
status: open
area: sessions / discord / engine
---

# s000a post-mortem: End session was refused because the id has a letter; two coders were dropped by the engine's 600 s wait

## Question

Live session s000a (2026-10-24, project `scratch`, reach `code`, mode `policy` throughout, engine
2.1.310, bot built from `main` before P12b) built P12b from Discord. Its three commits (`174397b`,
`b3b57f1`, `02d891d`) never left the tree, and the owner thinks the session is over. What went wrong,
in order, and why? The session's own finding
(`2026-10-24-background-subagents-die-when-the-turn-ends.md`) is checked here, not repeated.

## What was tried

Read only, under `<root>` (the live root): `broodforge --root <root> session show|tail --all|list`,
`audit --session s000a --all`, `audit --kind action-refused,session-ended,push`, `status`; the
engine transcripts under `<root>/harness/<credential>/projects/<encoded tree>/` (the session's and its
14 subagents', with their `.meta.json`); `sessions/s000a/home/hook.log`; `workflow/s000a/instructions.md`;
the bot process's command line and start time; `internal/discord/customid.go` at `caabb40` (what the
running bot was built from) and on `main`. Repro on a scratch dry-run bot, not the live one:

```
export B=<repo>/bin/broodforge.exe
$B scratch run -- bash -c '"$B" serve --discord dry-run & "$B" status --wait 60;
  for id in bf:s000a:end bf:s0009:end bf:s000a:stop bf:s000a:model; do
  "$B" discord inject click --custom-id $id --user 1001 --roles 10; done; "$B" stop'
```

## Result

Times are UTC. "Turn" is the bot's turn number.

1. **End session refused; no handoff (Broodforge bug, the headline).** At 18:23:35 the driver clicked
   End session. Audit row `action-refused`, reason `bad-args`: `a custom id of 12 bytes starting "bf"
   is not a Broodforge custom id` (shown to the driver as an ephemeral follow-up). The custom id is
   `bf:s000a:end`. Session ids are `s` plus base-36 (`03-architecture.md:206`, `internal/git/git.go:48`
   and `internal/spawn/worktree.go:49` at `caabb40` accept `[0-9a-z]`). The Discord package alone checks
   `^s[0-9]{4,}$` (`internal/discord/customid.go:48` at `caabb40`), and s000a is the first id
   with a letter. The repro gives that exact refusal for `end`, `stop` and `model`; `bf:s0009:end`
   parses and fails on `unknown-session`. Effects for any such id: End session, Stop, permission
   answers (only policy mode spared s000a those) and the P12b menus all fail to parse. The same regex
   gates the startup sweep (`internal/discord/gateway.go:368` at `caabb40`). Fixed by the commit
   "Discord: accept base-36 session ids in custom ids and the sweep": Discord now calls
   `session.ValidSessionID`; the tests are `TestCustomID` and `TestSweepTakesBase36IDs`. The live bot
   gets it only after a `bin/` rebuild and restart. Reading the code, not tried live: after
   a bot restart the venue is not bound again, so its messages and `/session end` would not reach the
   session, and once it ends its channel would never be deleted.
   State now: `waiting-for-input`, `endedAt` null. No `session-ended`, no `push`, no handoff row. The
   project's `idleTimeoutMinutes` is 0, so it never ends by itself. `status` prints `1 open` and
   `worktrees: 0 kept with unpushed commits`, because R-35's count covers kept trees only, so an open
   session's unpushed commits appear nowhere. The untried way out is the typed `/session end` (no
   custom id) or `broodforge session end s000a`. Meanwhile the three changes reached the local `main`
   by hand (`11b19c0`, `9f34e96`, `c8154a3`; the first two match by patch-id, the third was edited). A
   later handoff would push the same changes on the session branch under different hashes.

2. **Typed `/effort high` reached the model (turn 1).** This is item 4 of the friction finding. The
   driver then set the model and effort with slash commands (audit 15:35:36 to 15:36:11).

3. **The engine waits up to 600 s for background agents after its last `result` (engine behaviour;
   corrects the session's finding).** That finding says the engine does not wait under `-p`, and that
   the early agents' success is unexplained. The transcript shows otherwise. In turn 3 (15:39:37 to
   16:05:13) the coordinator ended its reply at 15:40:15. The process stayed up, and each background
   agent's `task-notification` arrived later as a new prompt in the same process (15:48:29, 15:56:30,
   16:04:39, 16:05:09), with another `end_turn` after each. The process exited only once no task was
   left. Every one of those gaps was under 600 s. Turn 4: the coder was launched with the defaults
   (`launchedWithDefaults: true`, `requestShape: background` in its meta), the reply ended at
   16:05:34.521, and the coder's last transcript line is 16:15:34.660. Turn 5: the reply ended at
   16:38:14.436 and the second coder's last line is 16:48:14.567. Both gaps are 600.1 s. Both turns
   ended with `exit 0` in `turn-ended`, and Broodforge has no turn timer (`internal/session/turn.go`
   reads to the exit; `internal/spawn/process.go` kills only on interrupt). So the engine exited by
   itself and left the agent behind. The next prompt carried the engine's `stopped` notice ("didn't
   finish before the previous session ended"). Whether 600 s is a fixed engine constant: two
   observations, unverified.

4. **Nothing told anyone (Broodforge gap, known).** The `system/task_*` lines are dropped
   (`KindUnknown`). The Agent call's `subagent-result` row says `ok` at launch (16:05:31, 16:38:09),
   because for a background launch the result is the launch acknowledgement. The coordinator's
   closing message ("After the build comes a review, then the commit") came 10 minutes before the
   process ended. The trace then showed the subagent's own calls. So "turns 4 and 5 ended on a tool
   call with no closing text" (the 2026-10-24 closing-message idea) misreads the cause: the closing
   text was there, and the last entries were a background agent's calls until the engine dropped it. A
   prompt for a closing message would not have helped. The new footer label would name the wrong
   cause. The driver waited 22 minutes (16:15:34 to 16:37:42) before asking.

5. **The coordinator backgrounded again after the first loss (workflow).** In turn 5 the coordinator
   said the process had ended while the coder was working. It then relaunched with the defaults and
   promised to report back, and lost the second coder the same way. The driver sent the same question
   again at 16:52:10. From then on every launch was `run_in_background: false`, and all eight
   completed (the coder took 14 m 49 s). Each lost coder left uncommitted partial edits, and the next
   coder was briefed to pick them up. Turns 4 and 5 used 13.5 M input tokens (cache reads included)
   and produced no commit.

6. **Large tool output cannot be read back (Broodforge policy vs engine).** Four `Read` denies in
   `hook.log` (15:48:56 and 17:08:44 by reviewers, 17:43:36 and 17:45:21 by pushers) were all on the
   engine's own `<config dir>/projects/<cwd>/<engine session>/tool-results/` file. The engine writes
   that file when an output exceeds its limit ("Output too large ... Full output saved to") and tells
   the model to read it. The credential rule (`internal/hook/facts.go:143`) denies it. This is the
   same over-broad rule as the friction finding's memory item, on a subtree that finding does not
   name. The coordinator's memory `Write` was denied too (17:27:05, already known).

7. **Other reach denies: `Skill`, `ToolSearch`, `PowerShell`** (15:39:49 and 15:56:32, 17:31:58,
   16:37:53). These are known (friction item 1). One effect: the `Skill` deny blocked a question about
   which models accept `xhigh`, so `xhigh` shipped untested per model (the coordinator said so at
   16:05:34).

8. **"Commit and push it all" could not work (workflow; the design held).** Turn 8 (17:42:47 to
   18:01:42) ran two `pusher`s. The first: `git-guard.js` denied an unexpanded `$(...)` argument, then
   check 8 failed on `gcc` missing from the engine's `PATH` (cgo `-race`). It stopped at 17:44:41
   without pushing. The second added the `PATH` prefix from `docs/reference/go-on-windows.md` and ran
   the full checks, all passing. It also ran `node .claude/hooks/test-hooks.js` (86 of 772 FAIL;
   `git push` allow cases among them; cause not established). It then made one `git push -u origin`
   at 18:01:26. The tree's pre-push hook refused it with the P15 line: `session s000a does not push.
   Commit in this tree; Broodforge pushes at the handoff when the session ends.` The pusher did not
   work around it. The bot's own hook had allowed the call (`hook.log`, policy mode), so the stop was
   the tree's hook, as designed. The coordinator quoted the refusal to the driver and said the commits
   "go to the remote when you end the session". The driver did that 22 minutes later, and it failed
   (item 1). Cost: 19 minutes and two Sonnet runs on a push the tree refuses by design. The session
   header (`workflow/s000a/instructions.md`, lines 1 to 7) does not say that the tree does not push,
   while the injected agreement tells the coordinator to push through the `pusher`. The header line is
   waiting on the owner's wording (the handoff's P15 item).

9. **The 2026-10-23 prompt-mode defect: not exercised.** s000a ran in policy mode from its start
   (`session-started` `mode: policy`, no `mode-changed`). `hook.log` holds no `defer`. All 12
   foreground or completed-background subagents finished (`finished: true`). So this session neither
   confirms nor clears `2026-10-23-subagents-in-a-session-end-before-writing.md`.

Classification. Broodforge bugs: 1 (the digit-only id regex, plus the open-session gap in `status`),
4 (task lines dropped, `ok` at launch), 6 (credential rule too broad). Engine behaviour: 3
(background is the default; `-p` waits 600 s and then exits 0), and the spill file of 6. Workflow and
briefs: 5, the pusher brief without the `PATH` prefix, the push attempt in 8 with the header silent on
P15, and the session finding's unchecked claim in 3. Items 2 and 7 are known friction.

## Next

- Done: the Discord session-id check accepts base-36, with tests for every custom-id form and the
  sweep. Open: adding `discord inject click --custom-id bf:s000a:end` to `tests/repro/` (not done),
  and ending s000a through a path that works (the owner's call, given the hand-cherry-picked `main`).
- Architect: should `status` (R-35) also count an open session's unpushed commits? Should a session
  whose end is refused say so in the venue rather than only in an ephemeral reply?
- The session's background-subagent finding: its mechanism claim (no wait under `-p`) is contradicted
  by item 3. Option b there is partly what the engine already does; the 600 s cap is the gap. A
  recording of one background `Agent` launch lasting more than 600 s would settle the constant.
- The credential rule's carve-out for the engine's `tool-results/` and `memory/` (one proposal with
  friction item 2). The header sentence on P15 (the owner). The 86 hook-test failures inside a session
  tree (a tester, read only).
