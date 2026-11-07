# Critic review of proposal 0029: a subagent's deferred call in prompt mode

Date: 2026-11-07
Reviewed: `0029-subagent-prompt-in-prompt-mode.md` (Status: under review)
Verdict: the mechanism B rests on is real and works under the bot's own scrub (one real turn, below), but B as
written breaks a constraint it lists, leaves the exact loss it claims to make impossible in two cells, and has no
detection for its own silent failure. Two blocking, seven serious, eight minor. Recommendation at the end: D with a
bounded hold, B as the later promotion, not the reverse.

## One real turn, run by the critic

The brief allows one short bare run on the owner's login. Engine 2.1.319, Haiku, cwd a scratch tree outside any
repository, every `CLAUDE*` name unset as the spawn strips them, `CLAUDE_CODE_SUBPROCESS_ENV_SCRUB=1`,
`--setting-sources user --settings <stub>` (the 2026-11-07 control stub, mode `ask`), the bot's flags plus
`--permission-prompt-tool stdio`, stdin held open by a node driver, no `initialize` handshake. Raw stream, hook stdin,
driver log and the driver itself: `<scratchpad>/bf-critic-0029/c41/`.

| Seen | Line |
|---|---|
| The hook's `ask` on a top-level `Write` became a `control_request` at 4.3 s and the engine waited | `{"type":"control_request","request_id":"<uuid>","request":{"subtype":"can_use_tool","tool_name":"Write","display_name":"Write","input":{...},"description":"hello.txt","decision_reason":"Broodforge: waiting for a person in this session to answer","decision_reason_type":"hook","decision_reason_hook_event":"PreToolUse","decision_reason_hook_source":"settings","tool_use_id":"toolu_..."}}` |
| A `control_response` allow 3 s later ran the call under the same id | `tool_result` for that id, `File created successfully`; `tool_result_meta` `{"decision":"accept","source":"user_temporary"}` |
| The turn ended; the engine did not exit on `result` with stdin open | `result` `end_turn`, `permission_denials` `[]`, `num_turns` 2; exit 0 came 0.6 s after the driver closed stdin, 5 s after `result` |
| The scrub did not force the tool off | `init` `permissionMode` `default`, stderr empty, no forced-mode line |

So C41 (top-level half), the third part of C43 and C45 (for a `Write`) hold on the pinned engine, and no
`initialize` control request is needed. C42 (a subagent's call), the rest of C43, C44, C46 and C47 stay for the spike.
Cost: one Haiku turn, under ten thousand tokens.

## Claims verified

| Claim | Verdict | Source |
|---|---|---|
| `--permission-prompt-tool stdio` is a documented value | No. The CLI reference documents the flag as "Specify an MCP tool to handle permission prompts in non-interactive mode. Pass the full name, such as `mcp__<server>__<tool>`"; `claude --help` on 2.1.319 has no line for the flag at all (it is named only inside `--permission-prompts`). `stdio` is the Agent SDK's internal value: the shipped TypeScript bundle 0.3.319 pushes `--permission-prompt-tool=stdio` when `canUseTool` is set and refuses it together with `permissionPromptToolName`. | code.claude.com/docs/en/cli-reference; local `claude --help`; `sdk.mjs` in the Agent SDK's TypeScript npm package (`claude-agent-sdk`, scope `anthropic-ai`), version 0.3.319 |
| The control protocol shape (C41) | Holds, observed above; the request carries `tool_use_id`, `decision_reason` (the hook's reason verbatim) and `decision_reason_type` `hook`. The TypeScript reference documents `requestId` as "The `control_request` envelope's `request_id`" that a response "must echo", the `CanUseTool` options `toolUseID: string; agentID?: string`, and the Python SDK reads `agent_id` from the request, so a subagent's request should name its agent (C42, unobserved). Response shape per the Python SDK: `{"type":"control_response","response":{"subtype":"success","request_id":...,"response":{"behavior":"allow","updatedInput":...}}}`, deny `{"behavior":"deny","message":...,"interrupt":...}`. | agent-sdk/typescript; claude-agent-sdk-python `query.py` |
| The engine never times a pending ask out (B's "the bot is the only clock") | Holds as documented: "permission prompts don't time out"; "Returning `null` ... leaves the tool call blocked indefinitely"; "The callback can stay pending indefinitely." | agent-sdk/typescript, agent-sdk/user-input |
| Closing stdin during a wait (C43) | Documented, not yet observed: "If your program closes the session through the Agent SDK, the SDK ends Claude Code's input before sending any signal, and Claude Code cancels the prompt as soon as the input ends." So an early close of the writer is a deny. | headless, "Stop a run with SIGTERM" |
| A subagent's ask reaches the prompt tool (C42) | Plausible, unobserved: 2.1.285 "Fixed `claude -p --permission-prompt-tool`: a background subagent's permission request now goes to the prompt tool instead of being auto-denied"; docs: foreground subagents' "Permission prompts are passed through to you as they come up". Against: issue #27203 (canUseTool not invoked for background subagent calls, closed stale 2026-05), #76204 (`can_use_tool` stream permanently closed after a Task subagent runs, CLI 2.1.201, closed NOT_PLANNED 2026-09), #34046 (no `can_use_tool` request at all on 2.1.73 to 2.1.78 with `stdio`, closed stale). The protocol has broken for subagents at least twice in this year's pins. | CHANGELOG 2.1.285; GitHub issues as numbered |
| Hook `ask` precedence (C44) | Partly documented: "Deny and ask rules are still evaluated regardless of what the hook returns"; hooks are step 1, allow rules step 5, the callback step 6; a hook `allow` "doesn't skip the deny and ask rules". Whether an engine allow rule or the read-only command set overrides a hook `ask` is not stated; the auto-mode paragraph says a hook `ask` "normally sends the tool call to the permission prompt". Keep C44. | agent-sdk/permissions; hooks, PreToolUse decision control |
| The scrub and the prompt tool (C45) | Holds for a top-level Write (above). Context the proposal lacks: a maintainer wrote on issue #79813 (open, reproduced) that the scrub coupling "is intentional hardening — that setting is designed for workflows processing untrusted input, where permission prompts must fall back to deny rather than being auto-approved, so permissive modes are pinned off at startup"; the lift the warning names works only as a settings-file `allowedTools` key or a `--settings` JSON `env` override, not the CLI flag (#60985, #51258 closed NOT_PLANNED). The hardening's stated intent is to deny prompts; today it does not touch the stdio tool, and nothing documents that it will not. | GitHub #79813, #60985, #51258 |
| `updatedPermissions` stops later asks (C46) | Documented for the SDK: "Echo one back in `updatedPermissions` to apply it"; `permission_suggestions` arrive on the request. Unobserved on the CLI path; the `source` of such a call is unknown (`user_temporary` was the one-shot allow). | agent-sdk/user-input |
| The engine's own `defer` guidance | The engine's docs recommend the opposite of B for long waits: "The callback can stay pending indefinitely ... If a user might take longer to respond than your process can reasonably stay running, register a `PreToolUse` hook that returns the `defer` decision instead of waiting in the callback, so the process can exit and resume later from the persisted session." The documented `defer` round trip is written with a permission host present ("A `-p` run offers `AskUserQuestion` only when it has a permission host ... so start the run with one"), so `defer` and the stdio tool coexisting on one process is the documented pairing, which D relies on. | agent-sdk/user-input; hooks, "Defer a tool call for later" |
| A's hook timeout (C48) | Weaker than the proposal says: the documented default `timeout` for a `command` hook is 600 s, so a six-minute hold is inside the engine's own default; a per-session value above ten minutes is the claim. | hooks, hook fields |
| A documented hook event the proposal does not consider | `PermissionRequest` "Runs when Claude Code is about to ask you for permission to use a tool. In sessions that can't show a prompt, such as background subagents in non-interactive mode, Claude Code still runs these hooks, and if no hook returns a decision, it denies the tool call. For a call that reaches a `--permission-prompt-tool` ... the hooks run alongside your host, and whichever decides first applies." Its input has `tool_name` and `tool_input` "but without `tool_use_id`", so it cannot be correlated to a call; rightly not a path, but it should be on the record. | hooks, PermissionRequest |
| 2.1.319 and subagent prompts | The pinned engine's own release adds `permissions.denyInsteadOfPrompting`: "a tool call that would wait on a permission prompt is denied and the turn continues, in terminal sessions, subagents and teammates", and "lists finished subagents, so you can reopen one and message it". The engine is moving subagent prompts toward deny-and-continue and toward resumable finished agents; not documented in the settings reference yet. | CHANGELOG 2.1.319 |
| The engine's memory while waiting | "A spare holds roughly 230 to 260 MB of memory while it waits" (the SDK's prewarmed process; the same binary idle). | agent-sdk/typescript, `prewarm()` |
| The finding's stream claims | Hold line for line: the three raw streams show `task_updated` `completed`, `task_notification` `completed`, the `Agent` result `accept`/`hook` with `canContinueAgent`, `end_turn`, no `deferred_tool_use`, `permission_denials` `[]` on both `defer` runs; the `ask` run shows `system/permission_denied` with `agent_id` and `decision_reason_type` `hook`, the error result `reject`/`source` `config`, and the subagent reporting the refusal. Hook stdin carries `agent_id` and `agent_type` on the subagent's call only. | `<scratchpad>/bf-subdefer/` `control/`, `control-ask/`, `rec/` |

## Refusals, and who can lift each

| Mechanism in B | Who lifts it, from where, while the session runs | Does the message say so |
|---|---|---|
| Prompt timeout deny (the bot writes `deny` with 04's reason) | Nobody needs to: a person clicks earlier; the model may re-issue | Yes (04's `the prompt timed out after N minutes`) |
| Reach deny, credential deny | Unchanged: nobody; the text says so | Yes |
| The engine refusing a hook `ask` because the tool is off or a pin drops the request | Nobody in the venue; only the owner at the spawn. The model reads the hook's reason `waiting for a person in this session to answer` (observed in the `ask` control run), which is false; the venue shows 04's hook-bypassed line naming a timeout or a hook failure, both wrong | No: objection 4 |
| `Interrupt` or `end` while a request waits (a kill) | The driver chose it; the subagent's run is lost, the output count unknown on Windows | 04's Stop text does not mention a lost subagent | 
| A subagent's `land` or `reviewer` call (`defer` kept) | Nobody: the subagent ends silently in every mode; the coordinator sees `(Subagent completed but returned no output.)` | No: objection 2 |
| The bot stopping while a subagent waits (job object, `KillMode=mixed`) | Nobody; the process and the wait are gone; after a restart the session is held (ADR 0026) | Not stated: objection 1 |

## The strongest case against B, and for the best alternative

B spends the project's one proven, documented, crash-safe prompt path (hook `defer`, exit, resume by engine session
id: C27, four fixtures, zero cost per hour of waiting) to buy a path that holds a 230 to 260 MB process per open prompt
on a host that "may sleep or reboot" (06), under a venue default that waits "however long (internally a year)" (04,
Owner 16), on a wire protocol the engine documents only as an SDK internal and has broken for subagents at least twice
this year. The engine's own documentation names B's case and tells integrators to use `defer` for it. B's one
decisive advantage over D is the coordinator's context at a top-level prompt, and that advantage is overstated: after
a `--resume` the transcript is intact; what the coordinator loses is the user turn's wording, not its work.

The best alternative is D, which the proposal dismisses in three lines for "two mechanisms for one thing" while B
itself keeps two (the `reviewer` and `land` cells stay on `defer`). D confines the undocumented protocol to the one
case where a held process is unavoidable (a subagent mid-run has nowhere else to wait), keeps every top-level prompt
at zero cost and restart-safe, changes no proven fixture, and needs one hook branch on `agent_id`, which the hook
already reads nothing of and must start reading for objection 2 anyway. D's held process must be bounded (objection
1); B's would have to be bounded for every prompt. A second alternative the proposal does not consider, C with the
engine's resume of a finished agent (`SendMessage` to the `agentId`, documented: "the subagent resumes in the
background without a new `Agent` invocation ... keeps the tool set ... and can keep reading the prompt cache"),
removes most of C's "whole brief again" cost; it stays rejected for the same reason C is, a per-tool grant where one
call was asked for, but the cost line in C should be corrected.

## Pre-mortem

Ten years on, prompt mode is the thing nobody dares restart. Every open permission prompt is a live engine process,
so the PC instance cannot sleep, the droplet unit cannot be upgraded, and the bot cannot be redeployed without turning
every waiting session into a held one; a "re-ask after restart" layer was built on top, then rebuilt twice when the
`control_request` shape changed, and the venue grew a second prompt type for the re-asks; the per-session hook settings
file carries a timeout nobody remembers the reason for. Owners set the prompt timeout to minutes to keep memory down,
which is the opposite of what Owner 16 asked for. The line that should have caught it is in B's own Risks: "a waiting
process survives a bot crash only as long as the job object or systemd kills it; a long wait holds the engine
session's memory", read beside the constraint the proposal lists and then breaks without a word, "One process per turn,
resumed by engine session id (ADR 0020)", and beside the engine documentation's sentence that recommends `defer` for
exactly this. The "What would change this recommendation" list has no entry for "a wait longer than a process should
live", so the prediction could not fail.

## Objections

1. Blocking. B makes every prompt a held process, against a listed constraint and the venue's default.
   Evidence: Constraints, "One process per turn, resumed by engine session id (ADR 0020)"; 04, "the prompt waits for
   its answer, however long (internally a year)"; 06, the PC "may sleep or reboot"; the engine docs' `defer`
   guidance quoted above; ADR 0026, a session whose process dies with the bot is held and refuses every action but
   `end`, and resuming a waiting session after a restart (R-18) is possible by construction only when the wait holds no
   process. B's Risks admit the hold and the recommendation does not weigh it. Resolves: adopt D for the top level;
   for the subagent hold (D or B alike), a bound the design states: a per-channel subagent prompt timeout (owner item,
   default proposed, not "a year"), on expiry or on bot shutdown a `deny` with `interrupt` false and a reason telling
   the subagent to report what it has, so the agent ends with a report instead of a kill; and a line in 03 and ADR 0020
   saying which waits hold a process and which do not.

2. Blocking. The loss the proposal says it makes "impossible by construction" stays in two cells, in every mode.
   Evidence: B keeps `reviewer` and `land` on `defer`; 03's table makes `land` `defer` "in every mode"; a subagent's
   `defer` ends it silently (the finding, all three runs); ADR 0028 rule 5 and the V12 probe expect a `pusher` subagent
   to issue the push word, and the companion carries the `land-not-allowed (subagent)` text that today can never be
   delivered because the hook reads neither `agent_id` nor `agent_type`. Policy mode has the same hole today.
   Resolves: in this pass, whichever option wins, the hook answers a subagent's call (stdin `agent_id` set) that
   matches the `land` cell with `deny` and the companion's `land-not-allowed (subagent)` text, and a subagent's
   `reviewer`-typed `Agent` call with `deny` and a text; a `hook cases` row each; ADR 0028's Deferred item amended to
   say the subagent push is refused with a message until the V12 probe passes, not lost.

3. Serious. B assumes one outstanding ask; the engine can have several.
   Evidence: 2.1.285 sends a background subagent's request to the prompt tool; background subagents run concurrently;
   C41 itself asks whether "both outstanding" happens; B's `turn.go` sketch has one `pending`, one `requestID`, one
   `waiting-for-answer`, and `Interrupt` answers one. Resolves: `pending` keyed by `request_id`, one venue prompt per
   request (each with its agent line), the prompt timer per request, `Interrupt`, `end` and shutdown denying all, a
   fixture with two outstanding requests, and the header's one-call-at-a-time sentence rewritten for subagents.

4. Serious. B's silent failure has no detection, no lift, and lies to the model.
   Evidence: with the tool off (the scrub's stated intent, #79813) or a pin that drops the request (#34046), a hook
   `ask` becomes `system/permission_denied` and an error result `reject`/`config` whose text is the hook's reason
   `waiting for a person in this session to answer` (the `ask` control run); `classify.go` would audit it as
   `hook bypassed` with cause "timed out" or "failed", both wrong; nobody in the venue can lift it. Resolves: the hook's
   `ask` reason rewritten for both readers (the venue prompt shows it verbatim as `decision_reason`; the model reads it
   on a fall-through); `session` raises `internal-error engine prompt tool: wanted stdio, got none` and an audit row
   when a hook-answered `ask` call (`decision_reason_type` `hook` on the denial line) reaches a result with no
   `control_request` seen, the trace line naming the lift (switch the session to policy mode; the owner checks the
   spawn); a fixture for it.

5. Serious. The stream's `source` values for a tool-answered call are not the contract's.
   Evidence: the run above gives `permission_decision.source` `user_temporary` for a stdio allow; 03 lists `hook`,
   `config` and absent; `classify.go` falls to `ClassAllowed`/`DecidedByUnknown` for any other value, so a session
   grant applied by the engine (`updatedPermissions`, C46, presumably `user_permanent`) would be audited as decider
   unknown. B's "never classified (`l.answers`)" covers only calls the bot answered by id. Resolves: 03 lists the
   observed values; a `user_*` source on an ask-cell call in prompt mode is the person's answer (by request or by
   grant), decider `participant`; any other new value stays `hook bypassed`; the fixture check asserts the value.

6. Serious. C42 is the claim the whole proposal turns on, and the evidence is one changelog line against three
   closed-unfixed issues. Evidence: the verification row above. Resolves: the spike runs C42 first, foreground and
   background, and records the `agent_id` on the request; the pin guard is the `prompt-ask` fixture plus a replay
   assertion that a subagent's `ask`-cell call never reaches a result without a `control_request` (the same check as
   objection 4), so a future pin that regresses #76204 fails the fixture check, not a session.

7. Serious. The question is written to select B. Evidence: "In ten years: one answer path for every call ... no
   process restart per prompt" is B's distinguishing feature, not a requirement (01 asks for prompts that reach a
   person and sessions that survive; R-18 asks for resume after a restart, which B makes harder). B is not one path
   either. Resolves: state the ten-year line as the requirement (a subagent's call gets a person's answer and the
   subagent continues; the loss impossible by construction) and let the options earn the rest.

8. Serious. B's exit cost and moving parts are understated. Evidence: "prompt mode returns to `defer` by one cell
   change" omits the session-allow semantics (the bot answering later asks in-process), the venue's agent line, the
   fake harness's scripted control requests, the per-request pending map (objection 3) and the detection of
   objection 4; "a held stdin" omits that the spawn's `start` writes stdin and closes it in a goroutine
   (`internal/spawn/process.go`) and that closing it early is a deny by the engine's documented behaviour, so the
   writer's lifetime is a safety property that needs its own test. Resolves: rewrite both lines; add the test.

9. Serious. `Interrupt` while a request waits is a kill where today it is free. Evidence: `actions.go` denies the
   pending call and ends the turn with no process; under B or D it is the Windows tree kill, S2's unknown output
   count, the subagent's run lost, and C47's unanswered question about the call's fate on `--resume`. Resolves: for
   a waiting request, the bot answers `deny` with `interrupt` true (the SDK's documented deny form) and waits a bounded
   time for `result` before the kill, so the turn ends with a `result` and the count; C47 covers the kill path only.

10. Minor. The coordinator-context cost is overstated. Evidence: after `--resume` the transcript is intact; the
    finding records a bare `Continue from where you left off.` user turn, not lost work. Resolves: quantify (one re-read
    of the plan per prompt) and say it is the price of D's top level.

11. Minor. C's cost line ignores the engine's own resume of a finished agent. Evidence: the sub-agents doc and
    2.1.319's "reopen one and message it"; `SendMessage` is `ClassOther`, a policy-table change. Resolves: correct the
    cost, keep the rejection on the grant width.

12. Minor. A's timeout risk is misdescribed and a documented event is missing from the options. Evidence: the
    `command` hook default `timeout` is 600 s; the `PermissionRequest` hook runs in `-p`, for background subagents, and
    "alongside your host", but carries no `tool_use_id`. Resolves: C48 names the ten-minute default; A's text records
    `PermissionRequest` as considered and why it fails (no id).

13. Minor. "What would change this recommendation" needs two more entries. Evidence: the scrub's stated intent (a
    later pin denying prompts under the scrub) and 2.1.319's `denyInsteadOfPrompting` direction; neither is a listed
    trigger. Resolves: add both, with the fixture check as the detector.

14. Minor. C41 and C45 should be split now that half is settled. Evidence: this review's run. Resolves: C41a
    (top-level, holds 2026-11-07, this review), C41b (two asks in one message), C42, C43a (stdin close mid-wait), C43b
    (exit after `result`, holds), C45 (holds for `Write`; a `Bash` under the scrub still to see).

15. Minor. The subagent's deny text and the venue's agent line are the owner's; the proposal says so for the line but
    not for the deny reasons of objections 1, 2 and 4. Resolves: list them under Owner items.

16. Minor. The hook contract's "it has no channel to the bot" sentence survives B unchanged but the Auto mode section
    does not: "shape A ... interrupts the process" is replaced by the live answer, and the `--permission-prompts` line
    ("stays off the command line") now matters, since `host` is what routes the ask to the tool. Resolves: the 03
    edits list both.

17. Minor. The fixture plan names `fixtures/engine/prompt-ask/` with one subagent deny; objections 3, 4 and 6 need
    three more recordings (two outstanding; the tool absent; a pin regression stand-in). Resolves: list them.

## Recommendation

Take D, not B: keep hook `defer` and the resume for every top-level call (proven, zero-cost waits, restart-safe,
the engine's own documented advice), and use `--permission-prompt-tool stdio` with stdin held open for a subagent's
`ask`-cell call only, bounded by a stated subagent prompt timeout with a deny-and-report on expiry and on shutdown;
fix the `land` and `reviewer` cells for subagents in the same pass by a hook `deny` with a message; add the
no-request detection and the `source` values to the contract. Promote to B later, as its own proposal, only when the
`prompt-ask` fixtures have survived several engine pins and the owner has chosen a bounded wait for every prompt.
A stays the fallback if C42 fails. Confidence: high that B as written should not be adopted (the held-process and
the two silent cells are facts, not judgment); moderate, about seventy percent, that D beats B on balance, the open
part being how often a coordinator's top-level prompt interrupts a multi-step plan in practice, which the owner can
read from the finding's sessions better than the critic can.
