# The stdio prompt tool on a prompt-mode process

Companion to `03-hook-contract.md`, which points here from How the engine runs it and What the bot sees. This
is the bot's side of the engine's control protocol under ADR 0029 as amended 2026-11-07: what a prompt-mode
process is started with, what a `control_request` carries, what the bot writes back, when stdin closes, and how
the results that follow are read. Unbuilt until P40 (`08-implementation-plan-subagent-prompt.md`). A fact marked
with a claim id or a run name was observed on engine 2.1.319 on the PC and is recorded in
`../findings/2026-11-07-spike-subagent-prompt-stdio-claims-for-adr-0029.md` (the runs under its scratch
directory); the hook's own side, the policy table and the Decision order, stays in 03. Every text a person or a
model reads here is a placeholder until the owner words it.

## The process

A prompt-mode process is started with `--permission-prompt-tool stdio` added to ADR 0020's command line; a
policy or auto process is unchanged, and `--permission-prompts` stays off every command line (the adapter's args
test asserts both). The user line is written to stdin as today and stdin is then held open: the engine does not
exit at `result` while stdin is open (C51: it exits 0.5 to 0.6 s after the close, a deferred turn and a resume
alike), and a resume with stdin open needs no line (C27 stands). The adapter (`engine`, the `claude-code`
implementation) owns the writer: `spawn.Spec.StdinOpen` leaves the pipe's writer open as `Process.Stdin` after the user line, and the adapter
closes it, never `spawn`. Which waits
hold a process: a subagent's prompt only, for at most `subagentPromptTimeoutMinutes` (ADR 0029 rule 6); a
top-level prompt holds none, the `defer` round trip (C27, C51) standing in every mode.

Lifetime, the adapter's one invariant, with its own test: stdin is closed exactly once, on the `result` line or
at exit, and never while a request is open. Closing stdin with a request open is a deny by the engine itself
(`c43a`: the tool result `Tool permission request failed: AbortError: Tool permission stream closed before
response received`, `source` `config`, `reason_type` `other`, the turn going on), so at `result` the adapter
first answers every request still open `deny` with `interrupt` true and the placeholder message `Denied by
Broodforge: the turn ended before anyone answered` (the engine's stock text replaces it for the subagent, The
answers), then closes stdin, then, when any request was open, calls the backend's interrupt at once (The turn's
end, below). With none open the close is enough: the engine exits (C51); an exit not seen within
`engine.ClaudeCode.ExitTimeout` of the close gets the backend's interrupt too. `ExitTimeout` is an internal value,
not a key: `internal/cli/engine.go` sets it from `spawn.interruptTimeoutSeconds` where it sets
`spawn.Config.InterruptTimeout`. The backend's interrupt is the kill as today, `TerminateJobObject` on Windows,
SIGINT then SIGKILL at `spawn.interruptTimeoutSeconds` on Linux (`03-architecture.md`, `spawn`); no new code
reads the key. Every line the engine prints after `result` is parsed and recorded (`engine record` keeps the raw
line), but the process's result is its first `result` line and `session` acts on nothing after it except the
exit event (The turn's end). A policy or auto process keeps today's write-then-close.

## The request

One line per hook `ask` on a prompt-mode process (C44: a hook `allow` or `deny` never asks; a hook `ask` asks even
for a `Bash` on the engine's own safe list), the same shape on every run:

```
{"type":"control_request","request_id":"<uuid>","request":{"subtype":"can_use_tool","tool_name":"Bash","display_name":"Bash",
"input":{...},"description":"...","decision_reason":"<the hook's reason>","decision_reason_type":"hook",
"decision_reason_hook_event":"PreToolUse","decision_reason_hook_source":"settings","tool_use_id":"<id>","agent_id":"<id>"}}
```

`agent_id` is the subagent's `task_started` `task_id` and the hook stdin's `agent_id`; the request carries no
`agent_type` (hook stdin does); a coordinator's own request has no `agent_id` key (`c44-c51`), and in a prompt
turn it never comes, since the hook defers at the top level. The adapter maps the line to the `prompt` event with
`RequestID`, `ToolUseID`, `Tool`, `Input`, `Reason`, `DecisionReasonType` and `AgentID` (ADR 0029 rule 3); a
`control_request` of another subtype, or without a tool name and a `tool_use_id`, stays `unknown`. Several
requests can be open at once (C41b: two from one message, answered out of order, each answer applied to its own
id); the adapter keeps the set of open request ids from the lines it read and the answers it wrote, and
`session` keeps `pending[request_id]` for the prompts and the timers.

## The answers

The adapter writes one line per answer, `TurnHandle.Answer(requestID, allow, message, interrupt)`:

- allow: `{"type":"control_response","response":{"subtype":"success","request_id":"<id>","response":{"behavior":"allow","updatedInput":<the request's input, unchanged>}}}`.
  The call runs under its `tool_use_id`; its result carries `source` `user_temporary` (`c42-allow`). An allow
  never carries `updatedPermissions` (C46 failed in `c46-c50`: a session rule sent that way did not stop the next
  ask; the shape passes the engine's own validator, so the driver was not at fault, and the likely, unconfirmed
  cause is a hook `ask` outranking a session rule; a pin that honoured it later would grant outside the hook's
  one list) and never changes `updatedInput` (C50 holds, and the `land` cell for a subagent through the tool is
  ADR 0029's deferred item, a pass of its own after P40).
- deny, `interrupt` false: `{"behavior":"deny","message":"<text>","interrupt":false}` in the same envelope. The
  subagent reads the message verbatim as an error result, no `PreToolUse:` wrapper, `source` `user_reject`, and
  goes on (`c42-deny`: its next call asked and ran).
- deny, `interrupt` true: the same with `true`. What the engine does when it reads the line (`c49`): the subagent
  is killed (`task_updated` `killed`, `task_notification` `stopped`); its result is the engine's stock rejection
  text, not the message, and carries no `tool_result_meta`; the `Agent` result is `[Request interrupted by user
  for tool use]` with `non_execution_kind` `interrupted` under the launch's own `hook` source; the coordinator
  takes one more model call, then `result` `end_turn` 2.2 s after the deny. The turn does not end by itself, so
  the bot never waits for it: the line is always followed by the backend's interrupt (The turn's end; Stop).

## What the bot does with a request

`session`, in a prompt turn, on a `prompt` event: ADR 0029 rule 4 (the `deferred` row with decider `policy`,
`pending[request_id]`, `waiting-for-answer`, 04's prompt with one line naming the agent type after the turn
number, the request's own timer; the process not interrupted) and rule 5 (the three answers; `Allow <tool> this
session` joins the one list, `BROODFORGE_SESSION_ALLOW` from the next process on and an at-once allow in the live
one, decider `participant`, reason `session-allow`; `session answer <id> allow|deny|allow-session --tool-use-id
<id>` answers a live request as it answers a deferred call, `--tool-use-id` required when more than one is
pending, `bad-args` naming them). The agent type is the parent `Agent` call's `subagent_type`, found through the
`tool_use` block's `parent_tool_use_id`, else the request's `agent_id`. The bound and its expiry are rule 6: deny,
`interrupt` false, the reason and the message there, decider `prompt-timeout`. `session show` prints one `pending:`
line per open request, `<tool use id>  <tool>  <target>  since <time>  until <deadline>  agent <type>`, the
`agent` column absent on a top-level deferred call; `session tail` prints the `waiting-for-answer` and the decision
rows indented under the `Agent` line as a subagent's calls are. In an auto turn a `control_request` is shape A as
today; in a policy turn it is noise, as today.

## The turn's end with a request open

The process has one turn (ADR 0020). A request still open when the `result` line comes, possible only with a
background subagent (`c42-bg`: the request came 0.2 s before the coordinator's first `result`), is dropped and
the process killed: the adapter answers it deny with `interrupt` true, closes stdin and calls the backend's
interrupt at once (The process); `session`, at the `result`, records every pending request `dropped`, decider
`instance`, cause `the turn ended`, with its trace line under the `Agent` call (placeholder `[<tool>] <target>
-> dropped: the turn ended before anyone answered`), removes its prompt's buttons, cancels its timer, and ends
the turn on that `result` with that `result`'s usage; the exit event closes the process as today. Why the kill
and not the close alone: a close after `result` with a request open was never run (the spike's close came
before, `c43a`); an answer written after `result` started a second turn in the same process within 3 s
(`c42-bg`: the allow ran the call, then a second `init`, a coordinator turn and a second `result`), and the
engine's own notification for the subagent the deny kills plausibly does the same, its allowed calls changing
the tree with no `session` row; a kill with a request open is observed (`c47-kill`: exit 1, no further line,
the resume writing the dropped call's interrupted result to the transcript only); the first `result`'s usage is
in hand before the kill. The deny lines go first so an engine that reads them before the kill lands (Linux,
where the interrupt is SIGINT) has the rejection in its transcript rather than a dangling request; on Windows
the kill lands first and `c47-kill`'s resume covers it.

First-result rule, every mode: the process's result is its first `result` line (`lastResult` and `UsageOf` read
the first, not the last); a line after it, a second `init`, `result`, tool call, tool result or
`deferred_tool_use` included, is logged as an event and never classified, never a decision row, never a prompt,
never a resume. The fake replays such a recording whole (Fixtures), so a session on `prompt-ask-background` sees
the second turn's lines and the rule has its test. Reading that second turn as a turn is the 2026-10-24
finding's option b (`../findings/2026-10-24-background-subagents-die-when-the-turn-ends.md`): an ADR 0020 matter,
its own proposal when the owner wants background subagents in prompt mode, with `c42-bg` as the recording that
finding asked for. Till then the header tells a prompt-mode coordinator to run subagents in the foreground (ADR
0029 rule 7). A top-level `defer` with a request open (`deferred_tool_use`, the same background case) is rule
7's: the same drop, the process exiting as today.

## Stop, end, shutdown

ADR 0029 rule 7 as amended: no wait. On `session interrupt` (Stop), `end`, the idle end and the bot's shutdown
while requests are open, `session` answers each deny with `interrupt` true and the placeholder message `Denied
by Broodforge: the session was interrupted`, then interrupts the process through the backend at once, as a
running turn's Stop does today (`TerminateJobObject` on Windows; SIGINT, then SIGKILL at
`spawn.interruptTimeoutSeconds`, on Linux). The records: Stop keeps its actor, decision `deny`, decider
`participant` (or `host` from the CLI with no `--as`), reason `interrupted`, as today's Stop on a deferred call, its result tracing `-> denied: the turn was stopped` (placeholder);
`end`, the idle end and shutdown record `dropped`, decider `instance`, with the cause
`05-debug-interface-session.md` names (`the session ended`, `the session ended: no activity`, `the bot stopped`).
The turn ends on the exit event as an interrupted turn does today, never before it, so a queued turn starts
under no live process; its outcome is `interrupted` whatever the exit carries, its usage S2's (unknown on
Windows; on Linux the engine's interrupt `result` when it comes before the SIGKILL): `04-ux.md`'s Stop line
stands. A `result`, a new `control_request` or a `deferred_tool_use` that lands between the denies and the exit
is logged, not acted on: no prompt, no row, no resume. Waiting for a `result` after the denies was refused: in
that window the coordinator takes one more model call (`c49`: the stock rejection, then `end_turn` 2.2 s later)
and a hook-allowed call of its runs during the Stop. A kill with a request open (C47) drops the call in the
engine as well: the resume writes an interrupted result for the `Agent` call to the transcript only, never to
the stream, so `session` owes the interrupted `Agent` call's record itself, as for S2. A turn with no request
open is interrupted as today.

## What the bot sees

The decider of a call the bot holds a request record for is the record's, whatever `source` the stream carries or
omits: `participant` (a click, `session answer` or a Stop; reason `session-allow` for the at-once allow,
`interrupted` for the Stop), `host` (a Stop from the CLI with no `--as`), `prompt-timeout` (rule 6's expiry),
`instance` (a drop above). The values observed under the tool, asserted by the fixture check on every recorded
result so that a pin which changes the protocol fails the check and not a session: `user_temporary` on an allow
(`c42-allow`), `user_reject` on a deny with `interrupt` false (`c42-deny`), `config` with `reason_type` `other`
on a request the engine aborted at a close (`c43a`), no `tool_result_meta` at all on a deny with `interrupt`
true (`c49`). With no record for the call: a `user_*` source is `hook bypassed`, cause `engine answered`;
`config` with `reason_type` `other` is `hook bypassed`, cause `engine aborted`; everything else is 03's (What
the bot sees). The cause rides in the decision row's `reason`, as `engine answered` does today: no schema
change. `result.permission_denials` lists the bot's denies beside the hook's. The detector of ADR 0029 rule 8 (a
hook-answered `ask` with no request seen) is unchanged; `prompt-ask-no-tool` is its fixture.

## Fixtures

Each is imported from the spike's run with the import tool's rewrite (every home path with the user segment:
`cwd`, `transcript_path`, `file_path`, `output_file`, the plugin's output; ADR 0018) and the recording account's
`system/hook_started` and `system/hook_response` lines dropped (a `SessionStart` plugin hook of that account, not
the engine's contract; `fixtures/engine/README.md` says so), or recorded afresh with `engine record --mode prompt
--requests <policy>` on the dev root; the independent read before the commit is P1's. Each prompt-mode fixture
keeps the hook's stdin for the turn's subagent calls as `hook-stdin-<n>.json` (a JSON array in call order, paths
rewritten the same way), and the fixture check asserts `agent_id` and `agent_type` on each entry, the `source`
value on each result as above, and rule 8's invariant.

The fake has no stdin and no lifetime: it replays the recording to its last line and then exits, as today, an
interrupt stopping it before the next line (the engine's close-of-stdin behaviour is not modelled: the engine
does not end its turn at a close, it aborts the open request and goes on to `result`, `c43a`, and the adapter
never closes with a request open). New: at a `control_request` line the fake waits for a `TurnHandle.Answer`
call when the recording answered that request inside the turn (the call's result precedes the turn's first
`result` line; the engine's own abort at a stdin close, a result with source `config` and reason_type `other`, answers nothing), whichever answer the bot gives (the recording's outcome is fixed; the opposite answer is another
fixture), and streams on without waiting otherwise; it records every answer it got in `received-<n>.json` as
`answers`, in order. A recording that runs past its first `result` (`prompt-ask-background`: the allow after
`result`, the second `init` and `result`) is replayed whole: the fake does not model the adapter's kill, so a
session on it sees the second turn's lines, which is the first-result rule's test.

| Fixture | Run | What it shows |
|---|---|---|
| `prompt-ask-allow` | `c42-allow` | a subagent's `Bash` under a hook `ask`: the request with `agent_id`, the bot's allow, the result `source` `user_temporary` under `parent_tool_use_id`, the `Agent` result carrying the report (C42 foreground) |
| `prompt-ask-deny` | `c42-deny` | two requests in one subagent: the first denied, `interrupt` false, the message verbatim, `source` `user_reject`; the second allowed and run |
| `prompt-ask-two` | `c41b` | two requests open at once from one message, answered out of order, each result under its own id (C41b) |
| `prompt-defer` | `c44-c51`, `c51-resume` | under the flag with stdin open: hook `allow` and `deny` with no request, hook `ask` requests for a `Read` and a safe-listed `echo` (C44), then a top-level `defer` ending the turn `tool_deferred` (C51); turn 2 the resume with `updatedInput`, no line written |
| `prompt-ask-background` | `c42-bg` | a background subagent's request open at the coordinator's first `result`; the recording goes on with an allow after it, the engine's second `init` and second `result`, replayed whole: the drop at `result` and the first-result rule |
| `prompt-ask-abort` | `c43a` | the engine's own deny at a stdin close: `source` `config`, `reason_type` `other`, the turn going on; the classifier's record-first rule and, with no record, the `engine aborted` cause |
| `prompt-ask-interrupt` | `c49`, `c49-resume` | a deny with `interrupt` true as the engine reads it (what an engine that reads the line before the kill lands does): the subagent killed, the stock text, no meta, the coordinator's last model call, `end_turn`; turn 2 a resume relaunching the subagent |
| `prompt-ask-kill` | `c47-kill`, `c47-resume` | the kill with a request open: exit 1, no `result`; turn 2 the resume with nothing streamed for the dropped call |
| `prompt-ask-no-tool` | the 2026-10-23 `control-ask` run (`<scratchpad>/bf-subdefer/control-ask`), else hand-written | a hook `ask` with no request on the stream: rule 8's detector fires |
| `prompt-ask-session-allow` | hand-written from `prompt-ask-deny` | `Allow Bash this session` on the first request; the second answered allow at once, no prompt, decider `participant`, reason `session-allow` |

## Debug surface

`hook try --agent-id <id> --agent-type <type>` and `hook cases` (`agentId`, `agentType`; `expect` `ask`), 05's
hook section; `engine run --fake --requests allow|deny` (`--fake` takes no `--mode`), `engine run --real --mode prompt --requests allow|deny` and `engine record --mode prompt
--requests allow|deny`, one rule for all three: every `control_request` the engine raises, or the fake waits on,
is answered that way at once (a deny with `interrupt` false and the placeholder `Denied by Broodforge: answered by
the recording's policy`), one stderr line `request  <request id>  <tool>  <agent id or ->  allow|deny` each, the
default `deny`, so no standalone run ever waits on a person; `engine run --fake fixtures/engine/prompt-ask-background`
prints the `prompt` event, both results and the exit; `session show` (`pending:` with `agent`), `session answer
--tool-use-id`, `session tail` (the indented rows), `audit --kind permission-decision`; `config check` on
`subagentPromptTimeoutMinutes` (rule 6: 0 and a negative value refused as `bad-config` naming the key, a value
above 1440 capped with a warning); `tests/repro/subagent-prompt.sh` (P40) [no-check: the script is written by P40].
