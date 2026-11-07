# 0029. A subagent's prompt in prompt mode: `defer` at the top level, the stdio prompt tool for a subagent's call, the hold bounded

Date: 2026-11-07
Status: accepted (the mechanism; the texts and the subagent prompt timeout's default are the owner's)

Record: proposal `../proposals/0029-subagent-prompt-in-prompt-mode.md` and `../proposals/0029-critic-review.md`
(its verification table and its one real turn, C41a, are authoritative; every objection resolved, none rejected).
Amends ADR 0020 (the Deferred `--permission-prompt-tool` item has fired in its narrow form; the Rejected MCP
tool stands, `stdio` is not it; Mechanism's command line and stdin rule for a prompt-mode process), ADR 0028
(Decisions 3, 4 and 5 for a subagent's call, and the Deferred item, L-2's subagent list) and `../03-hook-contract.md`
(the stdin fields, How the engine runs it, What the bot sees, the policy table's subagent rows, the Decision order and
the `--permission-prompts` line; each dated 2026-11-07 and marked unbuilt until this ADR's build pass).

## Context

Under `-p` the engine ends a subagent at a hook `defer`, passes nothing up and lets the turn end `end_turn`
(`../../findings/2026-10-23-subagents-in-a-session-end-before-writing.md`, Third look, engine 2.1.319): a
prompt-mode session cannot delegate, and a subagent's push word or `reviewer` launch is lost silently in every mode,
since the hook reads neither `agent_id` nor `agent_type` from its stdin. The owner asked for the fix (2026-11-07) and
answered the proposal's three questions the same day (the agent line's wording is fine; `Allow <tool> this session`
from a subagent's prompt grants the session, one list; the spike's real turns are approved). Verified before this record, by
the critic's run (`../proposals/0029-critic-review.md`: C41a, C43b, C45 for `Write`): a hook `ask` under the flag is a `control_request`
the engine waits on, an allow on stdin runs the call under its id (`source` `user_temporary`), the engine exits when stdin closes after
`result`, the scrub leaves the tool on; the engine never times a pending ask out and recommends hook `defer` for a long wait; `stdio` is
the Agent SDK's internal value, undocumented for the CLI, its protocol broken for subagents twice this year; C42 was unobserved until
the spike (Amendments). Ruled out: `auto` mode (the scrub forces it off, ADR 0020 C40c); a file or socket the hook
trusts (ADRs 0020 and 0022: the hook has no channel to the bot); holding a process for every prompt (critic,
blocking 1: against one process per turn, ADR 0026's held sessions and 04's year-long default wait).

## Decision

1. The hook reads `agent_id` and `agent_type` from stdin (`agent_type` for `hook.log` only). With `agent_id`
   absent nothing changes: the `ask` and `web` cells `defer` in prompt mode, the `reviewer` and `land` cells
   `defer` in every mode, the resume as ADR 0028. With `agent_id` set: the `ask` and `web` cells answer `ask` in
   prompt mode (a session-allowed tool still `allow`); the `reviewer` cell and, in code reach, the `land` cell answer
   `deny` in every mode, before the child chain, the `land` text the companion's `land-not-allowed (subagent)` and the
   `reviewer` text placeholder `Denied by Broodforge: in this session the reviewer is launched by the driver process;
   report and let it launch one.` (owner wording). Reach and credential denies come first as today. The `ask`
   reason is written for the model on a fall-through, placeholder `Broodforge: asked a person in this session; if
   this reads as a refusal, no prompt reached them: say so and stop`. A `hook cases` row per new cell.
2. A prompt-mode process runs with `--permission-prompt-tool stdio`; policy and auto processes are unchanged, and
   `--permission-prompts` stays off the command line. The engine adapter owns the stdin writer (`spawn.Spec.Stdin`
   becomes a writer): it stays open until the `result` line or exit, is closed on `result`, and is never closed while a
   request is unanswered, since an early close is a deny by the engine's documented behaviour; that lifetime has its own
   test. The fake harness answers scripted `control_request` lines from the fixture; `engine record` holds stdin
   open the same way.
3. The adapter's `KindPrompt` event carries `RequestID`, `ToolUseID`, `Tool`, `Input`, `Reason`,
   `DecisionReasonType` and `AgentID` (when the request has one). `TurnHandle.Answer(requestID, allow bool, message
   string, interrupt bool)` writes one line: `{"type":"control_response","response":{"subtype":"success","request_id":
   <id>,"response":{"behavior":"allow","updatedInput":<input unchanged>}}}` or `{"behavior":"deny","message":<text>,
   "interrupt":<bool>}` in the same envelope.
4. `session`, in a prompt turn, on a `can_use_tool` request: records the call `deferred` with decider `policy`,
   adds `pending[request_id]` (tool, `tool_use_id`, agent type from the parent `Agent` call's `subagent_type` through
   the `tool_use` block's `parent_tool_use_id`, else the request's `AgentID`), publishes `waiting-for-answer`, posts
   04's prompt with one line naming the agent type after the turn number, and starts the request's timer. The
   process is not interrupted. Several requests may be outstanding; each has its prompt, timer and answer. In an
   auto turn the request is shape A as today; in a policy turn it is noise, as today.
5. Answers. `Allow`: `Answer(allow)`. `Deny`: deny, `interrupt` false, 04's message. `Allow <tool> this session`:
   allow, and the tool joins the session's one list (`BROODFORGE_SESSION_ALLOW` for the next process on); a later
   request for a session-allowed tool in the live process is answered allow at once, no prompt, decider `participant`
   with reason `session-allow`. `session answer <id> ... --tool-use-id` answers a live request as it answers a
   deferred call; `session show` lists pending requests with their agent type and timer. A call the bot answered by
   `request_id` is decider `participant` from the bot's own record whatever `source` the stream carries; 03 (What the
   bot sees, amended 2026-11-07) lists `user_temporary` as the observed value; a `user_*` source on a call the bot did
   not answer is `hook bypassed`,
   cause `engine answered`; the fixture check asserts the value on every recorded result.
6. The bound. A subagent's prompt waits `subagentPromptTimeoutMinutes` (per channel, inheriting `defaults`; absent
   from `defaults` is the default, 60, owner item). The key does not take its siblings' zero: `promptTimeoutMinutes` 0
   means never, a year (`../03-architecture.md`), and this wait holds the engine process, about 250 MB idle, so never
   is the hold the critic's objection 1 refused, and a third meaning for 0 in one file would be read as one of the other
   two. So `config check`, `config reload` and `serve` refuse 0 and a negative value as an error naming the key, and
   cap a value above 1440 (a day) there with a warning, as the siblings cap at a year: a hold longer than a day on a
   host that may sleep or reboot (06) is a held session, not a wait. On expiry: deny, `interrupt` false, reason
   placeholder `Broodforge: no answer in <n> minutes; stop here and report what you have` (owner wording), the message
   `No answer; denied`, the turn continuing. The top-level prompt timeout (R-53) and its year-long default are
   untouched: a top-level wait holds no process. Which waits hold a process: a subagent's prompt only; 03 (How the
   engine runs it, amended 2026-11-07) and ADR 0020's amendment of the same date say so.
7. `Interrupt`, `end` and the bot's shutdown: every pending request is answered deny with `interrupt` true; the bot
   waits 03's turn-interrupted timeout for `result`, then kills as today. A top-level `defer` (`deferred_tool_use`)
   while requests are outstanding, possible only with a background subagent: the process exits as today, each
   outstanding request is recorded `dropped` with a trace line, and the header tells a prompt-mode coordinator to
   run subagents in the foreground.
8. Detection. A hook-answered `ask` call (the hook log's `ask` for the id, or `decision_reason_type` `hook` on a
   `system/permission_denied` line) that reaches a result with no `control_request` seen for it raises
   `internal-error` (summary placeholder `engine prompt tool: wanted stdio, got none`), an audit row
   `engine-prompt-tool-missing`, and a trace line naming the lift (switch the session to policy mode; the owner checks
   the spawn); the summary and the trace line are owner wording (critic's objection 15). The replay test
   asserts the same invariant over every prompt-mode fixture, so a pin that drops the request fails the fixture
   check, not a session.
9. Fixtures, recorded from the spike: `fixtures/engine/prompt-ask-allow/` (a subagent's `Bash` answered allow, the
   `Agent` result carrying its report), `prompt-ask-deny/`, `prompt-ask-two/` (two outstanding), `prompt-ask-no-tool/`
   (the request absent; the detection fires), `prompt-ask-session-allow/`. Each keeps the hook's stdin for the
   subagent's call as the spike recorded it (`hook-stdin-<n>.json` beside the fake's `received-<n>.json`, which is the
   `TurnRequest`, not hook stdin), and the fixture check asserts `agent_id` and `agent_type` on it, so a pin that drops
   the field fails the check, not rule 1's branch; the build pass owes that assertion. The debug surface is `hook try
   --agent-id <id> --agent-type <type>`, `hook cases`, `session answer`, `session show` and `session tail` (the trace
   line indented under the `Agent` call), `engine record --mode prompt`.
10. The spike before any `internal/` change, run 2026-11-07 (`07-spike-plan.md` conventions; the bot's command line, the scrub, a stub
    hook; owner-approved turns): its claims, the must-hold set (C42 foreground then background, C44, C45 for a subagent's `Bash`, C51),
    the must-be-known set (C41b, C43a, C47, C49), the cheaper-if-held pair (C46, C50) and the option A fallback are the proposal's
    "Claims to verify" and this file's history; the verdicts are the Amendments below. The build pass's last brief re-records the defer fixtures under the flag, on the owner's word (real turns).

Out of scope: the coordinator's own prompts (unchanged); `auto` mode; the engine's `PermissionRequest` hook (no
`tool_use_id`); `SendMessage` to a finished agent (a policy-table question of its own).

Deferred, each with its trigger: the stdio tool for every prompt (proposal B), as its own proposal, when the
`prompt-ask` fixtures have survived three engine pins and the owner chooses a bounded wait for every prompt; the
`land` cell for a subagent carried through the tool (the bot's answer with `updatedInput` on the request, no resume),
when C50 holds, which also revives ADR 0028's L-2 list; `updatedPermissions` on the allow instead of the bot
answering later asks, when C46 holds; a background subagent's wait surviving a top-level `defer`, when the engine
passes a subagent's `deferred_tool_use` up.

## Consequences

- Prompt mode can delegate: a subagent's `Bash`, `Write` or `Edit` in reach gets a person's answer and the subagent
  continues; its push word and `reviewer` launch are refused with a message instead of ending it, in every mode.
- Costs: a held engine process per open subagent prompt, up to the bound; two answer paths in `session`
  (`BROODFORGE_ANSWER` on a resume, `Answer` on a live request) and two fixture families; the top-level resume's
  three seconds and plan re-read stay; the undocumented control protocol enters the build, guarded by the fixture
  check and rule 8's detector; a bot restart during a subagent's prompt still loses that subagent's run (the session
  is held, ADR 0026); the stdin writer's lifetime is a safety property to keep tested; one more per-channel key.
- Supersede this when the engine passes a subagent's deferred call up (then `defer` alone and this ADR's rules 2
  to 8 go), or when B is adopted (rules 1 and 4 to 7 generalise to every prompt). Revisit it, option A first, when a
  later pin denies a hook `ask` under the scrub (the hardening's stated direction, the critic's C45 row) or routes a
  subagent's prompt to `permissions.denyInsteadOfPrompting` by default (2.1.319's new setting): the `prompt-ask`
  fixture check is the detector for both (critic's objection 13).
- Prediction, check when the spike runs (by 2026-11-21): C42 holds on 2.1.319 for a foreground subagent, and the
  request carries the subagent's `tool_use_id`. Held, 2026-11-07 (`c42-allow`).
- Prediction, check by 2027-02-07: no prompt-mode session on the pinned engine has raised `engine-prompt-tool-missing`,
  and no subagent in prompt mode has ended on an `ask`-cell call with `(Subagent completed but returned no output.)`.
- Prediction, check when the engine pin is first raised after the build: the `prompt-ask` fixture check, not a live
  session, is what reports any change in the control protocol.

## Amendments

2026-11-07 (the spike, `../../findings/2026-11-07-spike-subagent-prompt-stdio-claims-for-adr-0029.md`, engine 2.1.319, twelve starts; a run is named where a verdict is leaned on, the finding's table being under review; the spec of what follows is `../03-hook-contract-prompt-tool.md`, the build P40 in `../08-implementation-plan-subagent-prompt.md`). Verdicts: C42 foreground and background, C44, C45, C50 and C51 hold; C46 fails (`c46-c50`); C41b: two requests open at once, answered out of order, each by its id; C43a: a stdin close is the engine's own deny, `source` `config`, `reason_type` `other`, the turn going on (`c43a`); C47: a kill drops the call, the resume patching the transcript only; C49: an interrupting deny kills the subagent, not the turn: the engine's stock text, no `tool_result_meta`, one more model call, then `end_turn` (`c49`). Option A stays shelved; item 10 is done.

(1) Rule 2's "closed on `result`" stands and a process has one turn: at `result` with a request open the adapter answers each open request deny with `interrupt` true, closes stdin and calls the backend's interrupt at once (the kill as today: `TerminateJobObject` on Windows, SIGINT then SIGKILL at `spawn.interruptTimeoutSeconds` on Linux, the backend's own bound; nothing new reads the key); `session` records each `dropped`, decider `instance`, cause `the turn ended`, and ends the turn on that `result` with its usage. Why the kill, not the close alone (revised 2026-11-07 on the reviewer's reading): a close after `result` with a request open was never run; an answer written after `result` starts a second turn in the process within 3 s (`c42-bg`) and the engine's notification for the subagent the deny kills plausibly does the same, its allowed calls changing the tree with no `session` row; a kill with a request open is observed (`c47-kill`: exit 1, no further line, the resume patching the transcript only); the first `result`'s usage is in hand. The denies go first so an engine that reads them before the kill lands has the rejection in its transcript. First-result rule, every mode: the process's result is its first `result` line; a later line, a second `init`, `result`, call or `deferred_tool_use` included, is logged and never classified, prompted or resumed; the fake replays such a recording whole so the rule has a fixture test. With no request open the close stands and the engine exits (C51); an exit not seen within `engine.ClaudeCode.ExitTimeout` of the close (an internal value set from `spawn.interruptTimeoutSeconds` beside `spawn.Config.InterruptTimeout`, no new key) gets the same interrupt. Reading the second turn is the 2026-10-24 finding's option b, an ADR 0020 proposal of its own, its recording now in hand; till then the header's foreground rule stands. (2) Rule 7 without its wait: Stop, `end`, the idle end and shutdown answer each open request deny with `interrupt` true, then call the backend's interrupt at once, as a running turn's Stop does today; the turn ends on the exit event, never before it (no queued turn starts under a live process), outcome `interrupted`, usage as S2 (unknown on Windows; on Linux the engine's interrupt `result` when it comes before the SIGKILL), so `04-ux.md`'s Stop line stands; a `result`, a request or a `deferred_tool_use` between the denies and the exit is logged, not acted on. Stop keeps its actor as today's Stop on a deferred call: decision `deny`, decider `participant` or `host`, reason `interrupted`; the other three record `dropped`, decider `instance`, with 05's cause. Waiting for a `result` was refused: in that window the coordinator takes one more model call (`c49`) and a hook-allowed call of its runs during the Stop. (3) A call the bot holds a request record for takes its decider from the record (`participant`, `prompt-timeout`, `instance`) whatever `source` the stream carries or omits (`user_temporary` on an allow, `user_reject` on a deny, `config`/`other` on the abort, none on an interrupting deny; the fixture check asserts each); with no record, a `user_*` or `config`/`other` source is `hook bypassed`, cause `engine answered` or `engine aborted`.

C46's consequence: rule 5 stands, the session allow recorded by the bot and applied by the hook, and an allow never carries `updatedPermissions` (the sent shape passes the engine's validator; a hook `ask` outranking a session rule is the likely, unconfirmed cause). C50 fires the Deferred `land` item's trigger: its own pass after P40. New surface: `engine run --real` and `engine record` take `--requests allow|deny` under `--mode prompt`; `session show`'s `pending:` line gains `agent <type>`; `hook try --agent-id <id> --agent-type <type>` and `hook cases` `agentId`, `agentType`; `config check` refuses `subagentPromptTimeoutMinutes` 0 and below; the fixtures are rule 9's five and the companion's table, imported with every home path rewritten and the recording account's plugin lines dropped, or re-recorded with `engine record --mode prompt`. `--permission-prompts` stays off every command line, asserted by the adapter's args test (P40c).
