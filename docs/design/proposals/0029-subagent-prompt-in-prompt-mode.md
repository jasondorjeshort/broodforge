# 0029. A subagent's deferred call in prompt mode: how a participant's answer reaches it and the subagent continues

Date: 2026-11-07
Status: promoted to ADR 0029 (revised 2026-11-07 against the critic: 17 objections, 13 accepted, 4 accepted with change,
none rejected; the recommendation moved from B to D with a bounded hold)

## Question

In prompt mode the hook answers `defer` to any `ask`-cell call, the engine exits `tool_deferred`, the bot posts the
prompt, and the click resumes the process with `BROODFORGE_ANSWER` (ADR 0020, C27). That holds for the coordinator's
own calls only. For a subagent's call the engine (2.1.319, three real turns on 2026-11-07, a bare `claude -p` the same)
ends the subagent at the `defer`, returns `(Subagent completed but returned no output.)` to the coordinator with
`canContinueAgent` true, raises no top-level `deferred_tool_use`, and the turn ends `end_turn`: nobody is asked, the
work is lost, and a relaunch costs the whole run again (`../../findings/2026-10-23-subagents-in-a-session-end-before-writing.md`,
Third look). So a prompt-mode session cannot follow the agreement's delegation rule; the owner delegates in policy
mode meanwhile. This proposal chooses how a subagent's `ask`-cell call gets a participant's answer and then runs or
is refused in place, with the subagent alive. The requirement (objection 7 rewrote this line): a subagent's call gets
a person's answer and the subagent continues; the loss above is impossible by construction, not by instruction, in
every cell; sessions keep surviving a bot restart (R-18) and prompts keep reaching a person (01). How many answer
paths that takes, and whether a prompt restarts a process, is for the options to earn.

What the engine offers (the finding): hook stdin for a subagent's call carries `agent_id` and `agent_type` (the
coordinator's own carries neither); a hook `ask` is survived (a `system/permission_denied` line with `agent_id`, an
error `tool_result` the subagent reads, `permission_denials` listing it); the subagent's `tool_use` line carries
`parent_tool_use_id`, so `session` already places it (`call.parent`, `call.depth`); `SendMessage` to the ended agent
is `ClassOther`, denied in every reach; `--permission-prompt-tool stdio` was untried when this was written and is now
seen once (the critic's run, C41a). Ruled out: `auto` mode (the C34 scrub forces it to `default`, the 2026-10-31
finding, row 3); the bot's handling of the subagent's id (the control run); the end of the coordinator's turn (the
finding). Considered and not a path (objection 12): the engine's `PermissionRequest` hook event runs in `-p` and for
background subagents, but its input carries no `tool_use_id`, so nothing could correlate its answer to a call.

## Constraints that matter here

- Both hosts (`../06-platforms.md`); GPL-compatible (no new dependency expected); every option drivable from a
  terminal (`session answer`, `hook try`, a fixture); agents write the code.
- The hook decides from the environment and has no channel to the bot (ADR 0020 Decision; ADR 0022: nothing under
  `data/` is bound into the sandbox). Inside the sandbox the hook and the session's Bash are one principal with
  `home/` read-write, so a file the hook would trust as the bot's answer is a file the session can forge.
- One process per turn, resumed by engine session id (ADR 0020); the Windows interrupt is a tree kill; a bot crash
  must not leave an engine running unseen (job objects on the PC, `KillMode=mixed` on the droplet, ADR 0023). A wait
  that holds a process is a wait a bot restart kills (ADR 0026: the session is then held), so the design says which
  waits hold a process and bounds those (objection 1).
- `auto` never widens reach; the two bot-answered cells (`reviewer`, `land`) resume with `BROODFORGE_ANSWER` and
  `updatedInput` (ADR 0028) for the coordinator's own calls; a subagent's call on either cell is the finding's loss in
  every mode today, and this proposal must close it (objection 2).
- The prompt timeout (R-53, default five minutes, up to a year) and the idle timeout keep their meaning for a
  top-level prompt; the venue prompt (`../04-ux.md`, Permission prompts) keeps its three buttons; wording is the
  owner's.
- ADR 0020 rejected the MCP permission-prompt tool for its undocumented schema and deferred a
  `--permission-prompt-tool` route to "C27 fails"; C27 holds for a top-level call and fails for a subagent's, so the
  trigger has fired in its narrow form. `stdio` is not a documented value: the CLI reference documents the flag for an
  MCP tool name only, and `stdio` is what the Agent SDK's bundle (0.3.319) passes when `canUseTool` is set (the critic's
  verification table). The engine's own documentation tells an integrator whose user may be away longer than a process
  should live to use the hook's `defer`, not a pending callback.

## Options

### A. The hook holds the subagent's call open while the bot asks

What. The hook, on a `Defer` result whose stdin carries `agent_id`, does not print `defer`: it connects to an answer
socket the bot listens on (`BROODFORGE_ANSWER_SOCKET`, a unix socket at `<session home>/answer.sock`, bound into the
sandbox with `home/`; a named pipe on the PC, the control package's transport), sends `{session, tool_use_id,
agent_id, agent_type}`, blocks until the bot replies `allow` or `deny <reason>`, and prints that. The bot, on the
hook's connect (not on the stream: the child chain or a path check may still deny), records `deferred` by policy,
publishes `waiting-for-answer`, posts the prompt, and replies on the click or the prompt timeout. The coordinator's
own calls keep today's `defer` and resume. The socket answers one question and nothing else: a session's Bash that
connects learns an answer it can read in the venue anyway and can set nothing, which is why a socket and not a file.
Venue: today's prompt plus one line naming the asking agent (`coder`, from `agent_type`; placeholder wording), the
trace line indented under the `Agent` call as `session tail` already indents it. Cost per prompt: none in tokens
and no process restart; the engine idles with the hook blocked. Timeout: the bot's prompt timer fires first and
replies `deny` with 04's reason; the hook's own `timeout` in the settings file is set per session to the channel's
prompt timeout plus a minute (the file is written per session, ADR 0022), so the engine's block (`onFailure`
`block`) is the backstop, reaching the subagent as a hook error it survives and the bot as `action refused` (03,
What the bot sees). The engine's documented default `timeout` for a command hook is 600 seconds (objection 12), so a
hold inside ten minutes is inside the engine's own default and only a longer per-session value is a claim (C48). A
coordinator `defer` while a background subagent's hook holds ends the process and the hold with it; only a
foreground subagent is covered, as the header's one-call-at-a-time already assumes.
`03-hook-contract.md`: a new environment row; "it has no channel to the bot" replaced by the socket's one-way
contract; the Decision gains step (2b) between the answer and the cell; the timeout paragraph. `turn.go`: a listener
goroutine per live session feeding `deferLocked` with a `held` flag (no result line, no usage write, the process still
`running`); `answerLocked` replies on the socket instead of `startProcessLocked` when `pending.held`; `Interrupt`
while held replies `deny` and kills as for a running turn. `hook try` gains `--answer-socket`.
Risks: the hook's I/O grows from "stdin, env, log" to a blocking network call, so the one unanswerable failure (the
hook killed at its timeout) now holds a venue prompt open; the engine runs hooks "once per call, in order", so a second
subagent's call waits behind the first hold (serial prompts, acceptable); on the droplet the socket is a new object in
a bind and a new line for the sandbox finding; the hold is a held process as much as B's, with the hook blocked instead
of the engine's own wait.
Exit cost: delete the listener, the env row and the hook branch; no data migrates. Moving parts: one socket per live
session, one env row, a per-session hook timeout in the settings file.

### B. `--permission-prompt-tool stdio` for every prompt

What. Every prompt-mode process runs with `--permission-prompt-tool stdio` and stdin held open until its `result`
line. The hook answers `ask` for the `ask` and `web` cells in prompt mode (today `defer`), for the coordinator's and a
subagent's call alike; the engine then emits a `control_request` (`request.subtype` `can_use_tool`, `tool_name`,
`input`, `tool_use_id`; the adapter's `KindPrompt`, built by P39 on the assumed form, which C41a confirmed) and waits.
`session` records `deferred` by policy, publishes `waiting-for-answer` and posts the prompt without interrupting; the
click or the timer writes one `control_response` line to stdin (`behavior` `allow`, or `deny` with 04's `message` and
`interrupt` false) and the call runs or is refused in the same process; the subagent, if any, reads the result and
goes on. `Allow <tool> this session`: the bot answers every later ask for that tool in the live process itself, and
`BROODFORGE_SESSION_ALLOW` still lets the hook answer `allow` from the next process on. The `reviewer` and `land`
cells keep `defer` and the resume (a call on its own, proven, `updatedInput` on the resume); nothing else uses
`BROODFORGE_ANSWER` in prompt mode. Venue: as A, the agent line included, one message shape for both levels. Cost
per prompt: no process restart (about three seconds and the resume's prompt re-render saved; the coordinator's
re-read of its plan after `Continue from where you left off.` is gone, since the turn never ends at a prompt; the
transcript itself is intact across a resume, objection 10; C28's dropped parallel calls become one ask each if the
engine asks per call, claim C41b). Timeout: the prompt timer writes the deny; the engine has no timeout of its own
(documented: a pending ask stays pending indefinitely), so the bot is the only clock, and `end` and `interrupt` kill
the waiting process as they kill a running one.
`03-hook-contract.md`: the policy table's `ask` cell reads prompt mode `ask`; `web` likewise; How the engine runs it
gains the flag and the stdin rule; What the bot sees: a call the bot answered is never classified (`l.answers`,
already so), whatever `source` the stream puts on it (C41a records `user_temporary`); Parallel calls rewritten after
C41b; Auto mode's shape A paragraph loses the interrupt (the same path answers a classifier ask, if C40 ever shows
one). `turn.go`: `KindPrompt` no longer interrupts in a prompt turn; `deferLocked` takes a `*PromptEvent` with no
result and no usage (the process lives); `answerLocked` calls `h.Answer(requestID, decision, reason)` when
`pending.requestID` is set, else resumes as today; `TurnHandle` gains `Answer`; `engine` keeps the stdin writer open,
closes it on `result` or exit; `spawn.Spec.Stdin` becomes a writer the engine owns; the fake harness answers a
scripted `control_request` from the fixture; `Interrupt` while waiting kills instead of ending without a resume.
Risks: every open prompt is a held engine process (230 to 260 MB idle, the SDK's own figure for the same binary) on a
host that may sleep or reboot, under a venue default that waits however long (04, Owner 16), so a bot restart turns
every waiting session into a held one (ADR 0026) and R-18's resume after a restart becomes harder, against the
one-process-per-turn constraint above and the engine's own guidance (objection 1); the control protocol is what the
Agent SDK's `canUseTool` rides on and is undocumented for the CLI (ADR 0020's reason for rejecting its MCP cousin),
and has broken for subagents at least twice in this year's pins (the critic's table: issues on 2.1.73 to 2.1.78 and
2.1.201); an engine pin that changes a field breaks every prompt, not only a subagent's; the hook's `ask` must reach
the tool and not the engine's own rules or safe list first (C44); the scrub's stated intent is to deny prompts
(C45); a waiting process survives a bot crash only as long as the job object or systemd kills it; a Windows kill
mid-wait loses the turn's output count (S2, as today's interrupt).
Exit cost (objection 8): the flag, the held stdin writer and its lifetime test, the `Answer` path, the per-request
pending map, the no-request detection, the in-process session-allow answering, the venue's agent line and the fake
harness's scripted requests go; prompt mode returns to `defer` by one cell change; every `prompt-ask` fixture is
re-recorded or dropped. Moving parts: one engine flag; a stdin writer whose lifetime is a safety property (the spawn
today writes stdin and closes it in a goroutine, `internal/spawn/process.go`, and an early close is a deny by the
engine's documented behaviour); a per-request pending map; a scripted-answer fake; no new object on disk.

### C. `ask` with a relay, and a relaunch after the answer

What. No engine or hook mechanism: the hook answers `ask` for a subagent's `ask`-cell call (stdin `agent_id` set),
which today's engine turns into a refusal the subagent survives. `session` reads the `system/permission_denied`
line (tool, `tool_use_id`, `agent_id`, reason) and posts an after-the-fact prompt: "`coder` wanted `[Bash] go build
./...` and was refused; allow Bash for this session?" with `Deny` and `Allow <tool> this session` only (an `Allow`
once has nothing to apply to). A click session-allows the tool; the subagent has already reported the refusal, so
the coordinator relaunches it on the next turn, the hook now answering `allow`. The coordinator's calls are
unchanged. Venue: a second message shape, a prompt about something already over. Cost per prompt (corrected,
objection 11): with the engine's own resume of a finished agent (`SendMessage` to the `agentId`, documented: the
subagent resumes without a new `Agent` invocation, keeps its tool set and its prompt cache), most of the brief's
re-reading is saved, but `SendMessage` is `ClassOther` and a policy-table change of its own; without it, one wasted
subagent run (tens of thousands of tokens on Opus) and the driver's next message; either way a grant per tool for the
session instead of per call, which is what Owner 6 wanted to keep rare. Timeout: none to add; an unanswered relay
just lapses. `03-hook-contract.md`: the `ask` cell gains "for a subagent's call, `ask`"; What the bot sees gains the
relayed refusal. `turn.go`: a `permission_denied` handler and a new event; no state change. Risks: it does not meet
the question (the call never continues); it teaches coordinators to pre-allow tools; a subagent that asks twice
loses twice. Exit cost: nil. Moving parts: nil.

### D. Hybrid: `defer` at the top level, the stdio tool for a subagent's call only, the hold bounded

What. B's mechanism, switched on by `agent_id`: the hook keeps `defer` for the coordinator's own calls (C27, proven,
zero cost per hour of waiting, restart-safe) and answers `ask` when stdin carries `agent_id`; a prompt-mode process
runs with the flag always, stdin held open until `result`, and the bot's `KindPrompt` path answers the subagent's ask
in place while a top-level `defer` still ends the process and resumes. The hold is bounded (objection 1): a
subagent's prompt has its own per-channel timeout (`subagentPromptTimeoutMinutes`, default proposed 60, never a year,
an owner item), and on expiry the bot writes `deny` with `interrupt` false and a reason telling the subagent to report
what it has, so the agent ends with a report instead of a kill. Several requests may be outstanding at once
(objection 3): `pending` is keyed by `request_id`, one venue prompt and one timer each. `Interrupt`, `end` and the
bot's shutdown answer every waiting request `deny` with `interrupt` true (the documented deny form that stops the
turn), wait 03's turn-interrupted timeout for `result`, then kill as today (objection 9). A hook `ask` that reaches a
result with no `control_request` seen (the tool off, or a pin that drops the request) is detected and audited, with
the lift named (objection 4). The subagent `land` and `reviewer` cells are closed by a hook `deny` with a message in
every mode (objection 2). Venue, cost and the rest as B for a subagent's call, as today for the coordinator's.
`03-hook-contract.md` and `turn.go`: B's changes plus the `agent_id` branch, with both answer paths kept, the
pending map and the detection; Auto mode is untouched, since the flag is on prompt-mode processes only. Risks: two
mechanisms for one thing (B has two as well, the bot-answered cells), two fixture families, two timeouts (the
engine's resume and the live answer), and the coordinator's re-read of its plan at every top-level prompt stays
(objection 10: one resume, about three seconds, the transcript intact); a top-level `defer` while a background
subagent waits on an ask ends the process and that wait with it, which the bot records as a dropped request and the
header forbids by telling prompt-mode coordinators to run subagents in the foreground; the undocumented protocol is
confined to the one case where a held process is unavoidable, a subagent mid-run with nowhere else to wait.
Exit cost: B's, less nothing, plus the branch; the top-level path is untouched by the exit, so no fixture of today's
is re-recorded. Moving parts: B's plus the `agent_id` branch in the hook and the second timeout.

## Recommendation

D (changed from B on 2026-11-07, objections 1 and 2), with A as the fallback if the spike refuses the stdio tool for
a subagent's call, and C taken by nobody: it does not answer the question and the owner already has the policy-mode
workaround at the same grant width. D keeps the project's one proven, documented, crash-safe prompt path (hook
`defer`, exit, resume by engine session id: C27, four fixtures) for every top-level call, which is the engine's own
documented advice for a wait longer than a process should live, and spends the undocumented control protocol only
where a held process cannot be avoided: a subagent that is mid-run. It bounds that hold, closes the two cells where a
subagent's call is lost silently today, and detects its own failure. B's one decisive advantage, no restart at a
top-level prompt, is a later proposal of its own, to be made once the `prompt-ask` fixtures have survived several
engine pins and the owner has chosen a bounded wait for every prompt. A buys the same for subagents at the price of
giving the hook a channel and a long-lived blocking call, which ADRs 0020 and 0022 built the hook to avoid, and holds
a process just the same. D takes ADR 0020's deferred `--permission-prompt-tool` route on its own trigger; the ADR's
Deferred item and its Rejected line are amended, not reversed. ADR 0028's Deferred item (L-2's subagent list) is
amended: a subagent's push word is refused with the companion's text until the stdio path carries the `land` cell.

Is this a major choice (ADR 0012)? Yes, under two of the agreement's triggers: D makes prompt mode lean on the
engine's undocumented control protocol for subagents (an API surface), and it changes cells of
`03-hook-contract.md`'s policy table, which that doc calls a design change. The critic loop followed. First
implementation step: the spike below on the real engine, owner-approved real turns, before any `internal/` change;
then the fixtures `fixtures/engine/prompt-ask-*` recorded from it (a subagent's ask answered allow; a subagent's ask
answered deny; two outstanding requests; the tool absent, a stand-in for a pin regression; a session-allow answered
in process), then the engine and session passes.

Owner items, answered 2026-11-07: the agent line in the prompt (the agent type after the turn number) is fine as
worded; `Allow <tool> this session` from a subagent's prompt grants the session, one list as today, not the agent
type; the spike's real-turn cost is approved ("Okay to spend the spike"). Owner items still open (wording or a
threshold a person meets, objections 1, 2, 4 and 15): the subagent prompt timeout's default (60 minutes proposed) and
its expiry reason to the subagent; the `reviewer`-cell deny text for a subagent; the no-request internal-error line
and its trace; the `land-not-allowed (subagent)` text is already the companion's and is reused. Placeholders ship.

## What would change this recommendation

- C42 fails: the engine never emits a `control_request` for a subagent's hook `ask`, foreground or background: D is
  dead for its one purpose, A is the design.
- C44 fails (the engine's own rules decide before the tool, or a hook `allow` still asks): D's cost in policy mode
  or its safety is not what the table says; A, which touches policy mode not at all.
- C45 fails for `Bash` (the scrub disables the tool, as its stated intent is to deny prompts): opting out of the
  scrub on prompt-mode processes is not acceptable (C34 hides the credential from Bash), so A.
- A later engine pin denies a hook `ask` under the scrub (the hardening's stated direction, issue #79813), or routes
  subagent prompts to `permissions.denyInsteadOfPrompting` (2.1.319's new setting) by default: the `prompt-ask`
  fixture check fails first (objection 13); A, or the engine's own resumable finished agents if they then carry a
  deferred call.
- The engine documents the hook's `defer` for subagents or passes a subagent's `deferred_tool_use` up: the status
  quo plus a resume of the subagent, and this proposal is withdrawn.
- The owner wants prompts answered by the person who launched the agent, not the channel's `answer` list: a role
  change in 04, no change here.
- The owner wants no restart at a top-level prompt and accepts a bounded wait for every prompt: B, as its own
  proposal, once the fixtures above have survived several pins.

## Claims to verify

Numbered after ADR 0020's table (C40 is its last); each is a run of the bot's own command line with
`CLAUDE_CODE_SUBPROCESS_ENV_SCRUB=1`, `--setting-sources user` and a stub hook, in a scratch tree, recorded as a
fixture; the finding's `promoted_to` is ADR 0020's table. Split per objection 14; the critic's one run is recorded
in `0029-critic-review.md`.

- C41a. Holds (2026-11-07, the critic, engine 2.1.319): `-p --input-format stream-json --output-format stream-json
  --permission-mode default --permission-prompt-tool stdio`, the hook answering `ask` on a top-level `Write`: the
  engine emits a `control_request` whose `request` carries `subtype` `can_use_tool`, `tool_name`, `input`,
  `tool_use_id`, the hook's reason verbatim as `decision_reason` and `decision_reason_type` `hook`, and waits with no
  `result`; a `control_response` `{"behavior":"allow"}` with `updatedInput` echoing the input runs the call under the
  same id; `tool_result_meta`'s `source` is `user_temporary`. Still open under this number: a `deny` with `message`
  (the wrapping the model reads, `permission_denials`); `updatedInput` with a changed value honoured (C50).
- C41b. Two asks in one message: one request each, serial, or both outstanding, and whether out-of-order answers
  hold.
- C42. The same for a subagent's `Bash` (a foreground `Agent` on Haiku, as the 2026-11-07 runs; then a background
  one): the request carries the subagent's `tool_use_id` (and whether `agent_id`); an allow runs it and the `Agent`
  result carries the subagent's report; a deny with `interrupt` false leaves the subagent running and reporting the
  refusal. Runs first in the spike.
- C43a. Closing stdin during a wait: deny, exit, or hang (documented as a cancel; unobserved).
- C43b. Holds (the critic's run): after `result` with stdin open the engine waits for input rather than exiting,
  and exits on EOF, so the adapter closes stdin on `result`. The wait itself: no result and no exit for at least six
  minutes with stdin open.
- C44. Precedence: a hook `allow` never asks the tool (policy mode's cost unchanged); a hook `deny` never asks; a
  hook `ask` asks the tool even for a `Bash` on the engine's safe list and for a `Read` (the hook answers `allow`
  for reads, so only the order matters); `--permission-prompts` left off.
- C45. Holds for a `Write` (the critic's run: `init` `permissionMode` `default`, stderr empty, the request emitted).
  Still open: a subagent's `Bash` under the scrub; if a pin forces the tool off as it forces `auto` to `default`,
  the stderr line and `init` field that say so.
- C46. `control_response` `updatedPermissions` (the SDK's form) stops later asks for that tool in the process, and
  the `source` such a call then carries; if not honoured the bot answers them itself (D's text assumes the latter;
  this only makes it cheaper).
- C47. Interrupt mid-wait (the Windows tree kill, SIGINT on Linux) and `--resume` after it: the unanswered call's
  fate in the resumed transcript (dropped, re-issued with a new id, or re-asked), so `session` knows what to record.
- C48 (A only, if D falls). A hook `timeout` above the documented 600-second default is honoured and a hook
  returning `allow` after it has its decision applied; two subagents' hooks run serially or in parallel.
- C49. A `deny` with `interrupt` true on a waiting request ends the turn with a `result` (its `stop_reason`, what
  the `Agent` result and `permission_denials` show) within 03's turn-interrupted timeout, so `Interrupt`, `end` and
  shutdown need no kill on the happy path.
- C50. `updatedInput` on a `control_response` allow with a changed `command` is honoured (the trigger for carrying
  the `land` cell for a subagent through the tool, ADR 0028's L-2 list).

## Critic's objections and responses

1. Blocking: B makes every prompt a held process, against a listed constraint and the venue's default. Accepted with
   change. The recommendation moves to D: the top level keeps hook `defer` and the resume (no process held, a
   restart harmless, as the engine's own guidance says); the subagent hold, which no option avoids, is bounded by a
   per-channel `subagentPromptTimeoutMinutes` (default 60 proposed, owner item; never the top-level prompt's year)
   with, on expiry, a `deny` with `interrupt` false and a reason telling the subagent to report what it has, and on
   `Interrupt`, `end` and shutdown a `deny` with `interrupt` true then 03's turn-interrupted wait before any kill.
   03 and ADR 0020 gain a sentence saying which waits hold a process (a subagent's prompt) and which do not (every
   top-level prompt). The constraint paragraph above now says so too.
2. Blocking: the loss stays in the `reviewer` and `land` cells in every mode. Accepted with change. The hook reads
   `agent_id` from stdin (it must for D anyway); a subagent's call that matches the `land` cell is answered `deny`
   with the companion's `land-not-allowed (subagent)` text, a subagent's `reviewer`-typed `Agent` call `deny` with a
   text of its own (owner wording; placeholder: `Denied by Broodforge: in this session the reviewer is launched by the
   driver process; report and let it launch one.`), in every mode, before the child chain; a `hook cases` row each;
   03's policy table amended today, dated; ADR 0028's Deferred item amended: the subagent's push word is refused with
   a message, not lost, until C50 holds and a later pass carries the `land` cell through the tool (then the bot's own
   answer with `updatedInput` on the request is the natural form, no resume).
3. Serious: B assumes one outstanding ask. Accepted. `pending` keyed by `request_id`; one venue prompt per request,
   each with its agent line; a timer per request; `Interrupt`, `end` and shutdown deny all; a fixture with two
   outstanding requests; the header's one-call-at-a-time sentence rewritten for subagents (foreground only in
   prompt mode; a background subagent's wait dies with a top-level `defer`, recorded as a dropped request).
4. Serious: B's silent failure has no detection, no lift, and lies to the model. Accepted. The hook's `ask` reason is
   rewritten for both readers (the model on a fall-through; it is not shown in the venue, the bot composes the prompt):
   placeholder `Broodforge: asked a person in this session; if this reads as a refusal, no prompt reached them: say so
   and stop`. `session` raises `internal-error` (summary `engine prompt tool: wanted stdio, got none`) and an audit row
   `engine-prompt-tool-missing` when a hook-answered `ask` call (`decision_reason_type` `hook` on the denial line, or
   the hook log's `ask` for the id) reaches a result with no `control_request` seen for it; the trace line names the
   lift (switch the session to policy mode; the owner checks the spawn). A fixture (`prompt-ask-no-tool`) for it.
5. Serious: the stream's `source` values for a tool-answered call are not the contract's. Accepted. 03 lists the
   observed `user_temporary`; a call the bot answered by `request_id` is decider `participant` from the bot's own
   record (`l.answers`), whatever the source; a `user_*` source on a call the bot did not answer is `hook bypassed`
   with cause `engine answered`; the fixture check asserts the value on every recorded result.
6. Serious: C42 is the claim the proposal turns on, on thin evidence. Accepted. The spike runs C42 first, foreground
   and background, and records `agent_id` on the request; the pin guard is the `prompt-ask` fixtures plus the replay
   assertion that a subagent's `ask`-cell call in prompt mode never reaches a result without a `control_request` (the
   same check as objection 4), so a regression of the 2.1.201 kind fails the fixture check, not a session.
7. Serious: the question is written to select B. Accepted. The ten-year line is rewritten as the requirement; B's
   "one answer path" is no longer a stated goal, and B is credited only with what it earns.
8. Serious: B's exit cost and moving parts are understated. Accepted. Both lines rewritten in B and D; the stdin
   writer's lifetime (the spawn's close-in-a-goroutine; an early close is a deny) named as a safety property with
   its own test in the engine package (the writer closes on `result` or exit and never before an answer).
9. Serious: `Interrupt` while a request waits is a kill where today it is free. Accepted with change. For a waiting
   request the bot writes `deny` with `interrupt` true and waits 03's turn-interrupted timeout for `result`, then
   kills; that the deny ends the turn with a `result` is claim C49, and C47 covers the kill path only.
10. Minor: the coordinator-context cost is overstated. Accepted. B's and D's text now say what is lost at a
    top-level resume: the user turn's wording and one re-read of the plan, about three seconds, the transcript intact;
    it is the price of D's top level and the reason B stays a possible later proposal.
11. Minor: C's cost line ignores the engine's resume of a finished agent. Accepted. Corrected in C; the rejection
    stands on the grant width and on `SendMessage` being a policy-table change of its own.
12. Minor: A's timeout risk misdescribed; `PermissionRequest` missing. Accepted. A names the 600-second default and
    C48 is narrowed to a value above it; the Question records `PermissionRequest` as considered and why it fails.
13. Minor: two more "what would change" entries. Accepted. The scrub's stated direction and
    `denyInsteadOfPrompting` added, the fixture check as the detector.
14. Minor: split C41 and C45. Accepted. C41a (holds), C41b, C43a, C43b (holds), C45 (holds for `Write`; `Bash`
    open); C49 and C50 added for objections 9 and 2.
15. Minor: the deny texts and the agent line are the owner's. Accepted. Listed under Owner items; the agent line
    was answered on 2026-11-07.
16. Minor: 03's Auto mode section and the `--permission-prompts` line. Accepted with change. Under D the flag is on
    prompt-mode processes only, so Auto mode's shape A paragraph is unchanged; the `--permission-prompts` line gains
    that on a prompt-mode process the stdio tool is the host the default `host` routes a hook `ask` to. "It has no
    channel to the bot" stands.
17. Minor: the fixture plan needs three more recordings. Accepted. Listed in the Recommendation: two outstanding,
    the tool absent (the pin-regression stand-in), and a session-allow answered in process, beside allow and deny.
