---
date: 2026-10-24
status: open
area: engine / sessions / subagents
---

# Background subagents die when the coordinator's turn ends: the engine exits at `result` and nothing reads its task lines

## Question

In a live Broodforge session the coordinator (a Claude Code main session driven one engine process
per turn, ADR 0020) launched a `coder` subagent with `run_in_background: true` and ended its turn
with a status message. The next thing it saw was the driver asking where the work had gone, with
the engine's own note that the agent "didn't finish before the previous session ended", no
completion record, and partial uncommitted edits in the tree. It happened a second time with the
relaunched coder, the same way. The agents' transcripts were saved; nothing told the driver or the
coordinator at the time. Every agent launched afterwards with `run_in_background: false` completed
(five in a row, up to fifteen minutes each), because the coordinator's turn stayed open until they
reported; a turn with a foreground coder ran about fifteen minutes with no engine output to the
channel beyond the working message. Earlier in the same session a background `architect` and
`reviewer` did complete; why is not known (see Result).

What ends the engine process, does the stream say anything about running background work, and what
should Broodforge do so a coordinator's background work survives, or so the coordinator is never
tempted to background it? The driver's own take: "coders shouldn't be killed when the turn ends.
Perhaps we're just supposed to be extending the turn waiting for them though."

## What was tried

- The events above, as the coordinator observed them in the session (treated as fact).
- Read: ADR 0020 (`docs/design/decisions/0020-harness-language-library-toolchain.md`),
  `internal/engine/claudecode.go`, `internal/spawn/process.go`, `internal/spawn/process_windows.go`,
  `internal/engine/parse.go`, `internal/engine/events.go`, `internal/session/turn.go`,
  `internal/session/prompt.go`, `docs/reference/claude-code-flags.md`, `docs/design/03-hook-contract.md`.
- Read `fixtures/engine/interrupted/turn-1.jsonl` and `turn-3.jsonl` (engine 2.1.296), the one
  recording with a background task in it.
- Web documentation (Claude Code headless mode, the Agent SDK, the `Agent` tool's
  `run_in_background`) could not be read: `WebSearch` and `WebFetch` sit behind `ToolSearch` in
  this session, and the channel's reach (`code`) denies `ToolSearch` with the reach text ("nobody
  in this session can allow it"). Claims below that rest on the author's own knowledge of Claude
  Code are marked unverified.
- One observation from inside the session: the `Agent` tool's description on the pinned engine, as
  this session's subagent reads it, says subagents run in the background by default and asks for
  `run_in_background: false` only when the caller's very next action depends on the result.

## Result

What ends the process. Broodforge never kills a process that ends normally. `internal/spawn/process.go:116-119`
writes the one user line to stdin and closes it at once (ADR 0020, Mechanism). The engine then
runs until the model's reply ends, prints `result` and exits by itself; `internal/engine/claudecode.go:189-193`
waits for that exit and pushes the exit event; `internal/session/turn.go:333-378` reads the events
to the close and `processEndedLocked` (`turn.go:395-410`) ends the turn. On Windows the engine's
kill-on-close job is closed when it exits (`internal/spawn/process_windows.go:86-95`), so a child
process the engine left (a background Bash) dies with it; a subagent is not a process but a task
inside the engine and dies when the engine exits. Nothing in Broodforge extends a turn, and nothing
could today: with stdin closed the engine has no way to receive more input.

Correction (`2026-10-24-s000a-post-mortem-end-session-refused-for-a-base-36-id.md`, item 3): the
claim in the next paragraph that the engine does not wait under `-p` is wrong. Per that transcript,
after the reply ends the engine keeps the process alive, takes each completion notice as a new
prompt, and exits about 600 s after the last activity, abandoning any agent still running (two
observations at 600.1 s; the constant is unconfirmed). Agents that finish inside that window
complete, which also explains the earlier `architect` and `reviewer`. The fixture reading below
(a 30-second sleep with no notice in that process) is left as recorded and is not reconciled with
this.

The engine does not wait for background tasks under `-p`. `fixtures/engine/interrupted/turn-1.jsonl`
lines 10 to 14: `system/background_tasks_changed` listing one task, `system/task_started` with
`is_backgrounded: true` (a 30-second sleep), the `tool_result` of the launch, an assistant text,
then `result` with `stop_reason` `end_turn` and no `task_notification` for the task in that
process. The foreground task in `turn-3.jsonl` lines 6 to 9 got `task_started`, `tool_progress`,
`task_notification` `completed`, then its `tool_result`, before the `result`. The note the
coordinator saw on its next turn ("didn't finish before the previous session ended") is the
engine's own wording for exactly this: a task record from a process that ended.

The stream does carry task events, and Broodforge drops them. The `system` subtypes seen on
2.1.296: `task_started` (`task_id`, `tool_use_id`, `description`, `is_backgrounded`, `task_type`),
`task_notification` (`status`, `summary`), `background_tasks_changed` (a `tasks` list). `ParseLine`
maps every `system` line other than `init` to `KindUnknown` (`internal/engine/parse.go:96-112`),
`events.go:87` calls such a line "logged at debug level and never fatal", and `runProcess` ignores
the kind (`internal/session/turn.go:369`). The only recorded `task_type` is `local_bash`; what a
background `Agent` launch emits (the same lines with another `task_type`, or nothing) is
unverified: no fixture holds an `Agent` call. The idea
`docs/ideas/2026-10-17-unknown-stream-events-are-dumped-verbatim.md` already asks for named kinds.

Why the earlier `architect` and `reviewer` completed: see the correction above (they finished
inside the engine's wait).

Relation to `2026-10-23-subagents-in-a-session-end-before-writing.md` (line ~34: foreground and
background ended alike). Both hold, with different causes. In s0008 the subagent ended at a
deferred call in prompt mode (the hook deferred, no prompt was posted), foreground or background.
In s000a the session ran in policy mode, so no call was deferred; only the background launches were
lost, to the engine's exit about 600 s after the last activity.

Unverified, from the author's knowledge of Claude Code: a background `Agent` launch returns at
once and its report arrives later as a task notification in the same conversation; the interactive
REPL outlives the reply, so the task survives; under `-p` the process ends at `result`, which is
what was observed. If so, the tool description (background by default) and the per-turn process
model pull in opposite directions, and an instruction alone will lose some of the time.

Options weighed.

- a. Instruct coordinators in the injected text to run subagents in the foreground and say so to
  the driver before each launch. The header (`internal/session/prompt.go:63-76`) already carries
  one operating rule of this kind ("Make one tool call at a time"), so a sentence such as "This
  session's engine process ends with your reply; a background task or subagent does not survive
  it" fits there as a fact about the runtime. Its sentences are the owner's verbatim under ADR 0025
  rule 1, so the line is an ADR 0025 amendment, small, and costs nothing in architecture. Alone it
  is advisory: the tool's own description says the opposite, and a model under pressure will
  background again.
- b. Keep the engine alive while it has running tasks. Broodforge would have to stop closing stdin
  at once, watch `background_tasks_changed` and `task_notification`, and close stdin only when the
  task list is empty or a timeout passes. That is a process waiting for more input, which is the
  core of the per-session model ADR 0020 rejected; interrupt, resume, a bot restart, the usage
  delta against the last `result`, and the per-process model and effort flags all assume one
  `result` per process. Evidence needed before it is even a proposal: a recording of a background
  `Agent` launch on the pinned engine with stdin held open after `result`, showing (1) whether the
  task lines name the agent, (2) whether its `task_notification` arrives in that process at all or
  only after another user message, (3) what the engine prints as a second `result`, if any, and
  (4) whether the engine exits on its own once the list is empty. ADR 0020 matter.
- c. One long-running engine process per session. Reverses ADR 0020 (its rejected list, the
  supersede trigger "the owner asks for a long-lived session", and the ADR 0023 amendment naming
  mechanism B). It alone makes background work survive across the driver's messages and also gives
  mid-turn message injection; its costs are the ADR's. ADR 0020 matter, out of proportion for this
  alone.
- d. Make backgrounding impossible at the one point that matters: the `PreToolUse` hook denies an
  `Agent` call whose input has `run_in_background: true` (or absent, if the default is background)
  with a reason that says why and what to do ("a background subagent does not survive this
  session's turn; launch it with run_in_background: false"); the deny reason reaches the model
  verbatim (C27, C39) and it re-issues the call, one round and about 3 s per launch. The same rule
  fits `Bash` with `run_in_background: true`, which the kill-on-close job ends at exit. A cleaner
  form rewrites the input to foreground through the hook's `updatedInput` (named in ADR 0020,
  C27, "untested by design"), costing no round; unverified on the pinned engine. This is a change
  to the policy table of `docs/design/03-hook-contract.md`, a design doc, so it goes through the
  proposal loop, but it is not an ADR 0020 matter: the process model stands.

Recommendation: d with a, now. The hook deny (or rewrite) is deterministic and is enforced where
the launch happens; the header line tells the model why so it does not fight the deny and tells the
driver what to expect ("I am running the coder in the foreground; this turn will take a while").
Neither needs the process model to change. b is not worth building before the recording above
exists; c is the owner's call under ADR 0020's supersede trigger, not a fix for this.

A human-facing question for the owner, decided nowhere here: how should the channel show a dead or
orphaned child? Today the trace shows the `Agent` line and its result (`turn.go:354-368`), and for
a background launch that result is the launch acknowledgement, so the trace reads as done. Nothing
shows the orphan. Candidates: a line in the turn's final message when the last
`background_tasks_changed` of the process still lists tasks ("Left running and lost at turn end:
coder, <description>"); the same in the footer; a `turn-ended` summary line in `session tail` and
an audit row. Whichever is chosen needs the task lines parsed as named kinds first.

## Next

- The owner: a or d or both now, and the header sentence's wording (ADR 0025 rule 1); whether a
  proposal for b's recording or for c is wanted at all (ADR 0020 supersede trigger). Not written
  here; this finding only names them.
- A spike recording (`engine record` on the pinned engine): one `Agent` launch with
  `run_in_background: true` and one with it false, each as a fixture; what task lines and
  `task_type` appear, and what the `result` carries. Resolves the unverified claims above and is the
  evidence b needs.
- Parse `system/task_started`, `task_notification` and `background_tasks_changed` as named kinds
  (anchor: `internal/engine/parse.go`, the 2026-10-17 idea), so the channel can show an orphan;
  then the owner's wording for that line, 04's turn loop and `08-implementation-plan.md`.
- If d is taken: the proposal for the `03-hook-contract.md` table row, and whether to deny or to
  rewrite (`updatedInput` test first).
