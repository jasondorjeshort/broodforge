# Implementation plan: a subagent's prompt in prompt mode (ADR 0029)

This file continues `08-implementation-plan.md` under the same conventions: the pass format, the size words, the
standing rules and the `[no-check: ...]` rule. It holds P40, the build pass of ADR 0029 as amended 2026-11-07
(`decisions/0029-subagent-prompt-in-prompt-mode.md`): the hook reads `agent_id`, a prompt-mode process runs the
stdio prompt tool, and a subagent's `ask`-cell call is answered in place from the venue, bounded. The spec is
`03-hook-contract-prompt-tool.md` (the process, the request, the answers, the turn's end, Stop, the sources, the
fixtures); the hook's side is `03-hook-contract.md`. Each brief reads the ADR (Decision and Amendments), the
companion, `05-debug-interface-session.md` (Prompt-mode defer and resume) and the spike finding
`../findings/2026-11-07-spike-subagent-prompt-stdio-claims-for-adr-0029.md` before its Implements line; where a
verdict is leaned on, the run is cited, not the finding's table (under review). Every text a person or a model
reads is a placeholder until the owner words it. P40 is five briefs, P40a to P40e, in this order; the steps the
ADR's amendment lists map to them as follows.

| Step | Brief |
|---|---|
| 1. The hook reads `agent_id` and `agent_type`; the `ask` decision; a subagent's `reviewer` deny (the `land` deny rides with P35) | P40a |
| 2. The spike's recordings imported; `hook-stdin-<n>.json`; the parser's `AgentID`, `DecisionReasonType` and `ReasonType`; the fixture check's assertions | P40b |
| 3. The stdio tool on prompt-mode processes; the stdin writer and its lifetime; the kill at `result`; `TurnHandle.Answer`; the fake's wait and `answers`; `--requests` | P40c |
| 4. `subagentPromptTimeoutMinutes` and `config check`'s refusal of 0 and negatives | P40d (first) |
| 5. The pending map by `request_id`; the venue prompt with the agent line; the answers; the bound and the timeout refusal | P40d |
| 6. The drop at `result` and the first-result rule; Stop, end and shutdown; the record-first decider; rule 8's detector; `session show`, `tail`, `answer` | P40d |
| 7. The `defer` fixtures re-recorded under the flag; one real prompt-mode session with a subagent's call answered | P40e (the owner's word) |

## P40a. The hook: `agent_id`, the `ask` decision, a subagent's `reviewer` deny (small)

Implements: ADR 0029 rule 1; `03-hook-contract.md` (How the engine runs it: the stdin fields; Policy table, the
`reviewer` row; Decision step 4); `05-debug-interface.md` (The PreToolUse hook: `hook try`, `hook cases`). Files
[no-check: the cases are added by this pass]: `internal/policy/policy.go` (`Request.Subagent bool`; a new `Ask`
decision; `Decide`: under `ModePrompt` with `Subagent`, `CellAsk` and `CellWeb` return `Ask` with the ADR's
placeholder reason; `CellAsk` returns `Ask` unless the tool is session-allowed, then `Allow`; `CellWeb` returns `Ask`
always (a session allow does not cover `web`); `CellReviewer` with `Subagent` returns `Deny`
with the ADR's reviewer text in every mode, before any pin; no other branch changes: a policy or auto process and a
call without `agent_id` decide as today); `internal/hook/hook.go` (`agent_id` and `agent_type` read from stdin, both
strings or absent, any other type a hook error; `Subagent` set when `agent_id` is present and non-empty; the `ask`
output `{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"ask","permissionDecisionReason":<reason>}}`,
exit 0; the `hook.log` line carries `agent_id` and `agent_type`; the chain runs on an `ask` as on an allow, a child
deny a deny); `internal/hook/cases.go` (`agentId`, `agentType`; `expect` `ask`); `internal/cli/hook.go` (`hook try
--agent-id <id> --agent-type <type>`, `--expect ask`); `tests/hook/cases.jsonl` (every `ask` and `web` row of the
table repeated with `agentId` under `prompt` expecting `ask`, with `sessionAllow` naming the tool expecting
`allow`, under `policy` and `auto` unchanged; the `reviewer` row with `agentId` in each mode expecting `deny` with
the reviewer text; a reach deny with `agentId` still `deny`; `agent_id` of a non-string type expecting `error`).
Debug surface: `hook try --tool Bash --input '{"command":"echo hi"}' --mode prompt --agent-id a1 --agent-type probe
--transcript-path <path> --expect ask`; the same `--mode policy --transcript-path <path> --expect allow`; `hook try --tool Agent --input '{"subagent_type":"reviewer"}'
--agent-id a1 --mode policy --transcript-path <path> --expect deny` printing the reviewer text; `hook cases tests/hook/cases.jsonl`.
Proof: the build, check and test lines; `hook cases` all `ok`; the three `hook try` lines; a test that the `ask`
output is byte-for-byte the shape the spike's stub printed (the finding, What was tried). Does not: touch the
engine, `session` or the spawn; add the `land` deny (P35 builds the `land` cell and takes the subagent row from
this pass's cases with the companion's `land-not-allowed (subagent)` text); word anything. Depends on: P33, P39.
Fixtures: none (the cases file).

## P40b. The fixtures: the import, `hook-stdin-<n>.json`, the parser's new fields, the check (small; an independent read before the commit)

Implements: ADR 0029 rules 3 and 9 and Amendments 2026-11-07 (the fixture list); the companion (The request;
What the bot sees, the observed values; Fixtures); `fixtures/engine/README.md`; ADR 0018. Files [no-check: the
fixtures are created by this pass]: `internal/tools/importfixture/main.go` (drops every `system` line whose
`subtype` begins `hook_` and says so in `fixture.json` as `"dropped": ["system/hook_started",
"system/hook_response"]`; rewrites every home path with the user segment wherever it appears, `cwd`,
`transcript_path`, `file_path`, `output_file` and the plugin's output included, in the stream and in
`hook-stdin.jsonl` alike, refusing any output that still names the PC user as today; writes the source's
`hook-stdin.jsonl` entries that carry `agent_id` as `hook-stdin-<n>.json`, a JSON array in order; takes a second
source directory for a fixture's turn 2, `--turn2 <src>`); `internal/engine/events.go` (`PromptEvent.DecisionReasonType`
and `AgentID`, JSON `decisionReasonType` and `agentId`, empty when absent; `ToolResultEvent.ReasonType`, JSON
`reasonType`, the entry's `permission_decision.reason_type`, empty when absent, beside `DecidedBy`, which is its
`source`), `parse.go` (`rawMeta.PermissionDecision` gains `ReasonType`; the `prompt` line's two fields; the doc
comment loses "assumed, not observed": the shape is `c42-allow`'s); `fixtures/engine/prompt-ask-allow/`,
`prompt-ask-deny/`, `prompt-ask-two/`, `prompt-defer/`, `prompt-ask-background/`, `prompt-ask-abort/`,
`prompt-ask-interrupt/`, `prompt-ask-kill/` from the runs the companion's table names, `prompt-ask-no-tool/` from
the 2026-10-23 `control-ask` run when it is still on disk, else hand-written in that run's shape, and
`prompt-ask-session-allow/` hand-written from `prompt-ask-deny` (claim `hand-written`, replaced when a recording
exists), each with its README row and `events-<n>.jsonl` goldens; `internal/engine/golden_test.go` or a sibling
(over every `prompt-*` fixture: each `hook-stdin-<n>.json` entry carries `agent_id` and `agent_type`; each result
of a call whose request the fixture holds carries the companion's `source` for the answer the recording gave,
`user_temporary`, `user_reject`, `config` with `ReasonType` `other`, or no `tool_result_meta` on the interrupting
deny; every `tool_use_id` the hook answered `ask` in `hook-stdin-<n>.json` has a `control_request` on the stream
except in `prompt-ask-no-tool`, where the test asserts the opposite). Before the commit an agent that did not
record the runs reads every file with nested JSON decoded, as P1: no name, email, host name, home path or
repository content survives, and the plugin's lines are gone. Debug surface: `engine run --fake
fixtures/engine/prompt-ask-allow --turn 1 --prompt x | grep '"type":"prompt"'` printing one line with
`"agentId"` and `"decisionReasonType":"hook"`; the same on `prompt-ask-abort` with `grep '"reasonType":"other"'`
printing one `tool-result` line; `go test ./internal/engine`. Proof: the two `grep` lines; the goldens
regenerated and the diff read (every changed line a `prompt` event gaining the two fields or a `tool-result`
gaining `reasonType`, nothing else); the check's five assertions passing on the ten fixtures; `grep -rliE "<pc
user>|[A-Za-z]:[/\\]+Users" fixtures/` empty; the independent read's report. Does not: change the fake's
behaviour (P40c; this pass's `engine run --fake` on a prompt fixture still streams without waiting); replace
`defer` or `parallel` (P40e); touch `session`. Depends on: P40a (the hook-stdin shape it asserts). Fixtures: the
ten above.

## P40c. The adapter: the flag, the stdin writer and its lifetime, the kill at `result`, `Answer`, the fake, `--requests` (medium)

Implements: ADR 0029 rules 2 and 3 and Amendments 2026-11-07 decision 1 (the adapter's half: the denies, the
close, the kill, `ExitTimeout`); the companion (The process; The answers; The turn's end, the adapter's part;
Fixtures, the fake; Debug surface); `05-debug-interface.md` (A turn against the fake engine: `engine run --real`
and `engine record` under `--mode prompt`). Files [no-check: the tests are this pass's]: `internal/spawn/spawn.go`
(`Spec.Stdin []byte` stays; `Spec.StdinOpen bool` beside it: when set, `Process.Stdin io.WriteCloser` is the
pipe's writer, left open after the bytes are written, the caller's to close; otherwise the write-then-close as
today); `process.go` (`start` takes the flag: it writes the bytes, then closes the writer or hands it to the
`Process`); `sameaccount.go` (the one caller of `start`, passing `spec.StdinOpen`; `bwrap.go`'s `Start` is held
and returns before any `start`, unchanged); `internal/engine/events.go` (`TurnHandle.Answer(requestID string,
allow bool, message string, interrupt bool) error`, the envelopes of the companion, `updatedInput` the request's
input unchanged, never `updatedPermissions`; no `Close`: the lifetime is the adapter's); `claudecode.go` (`Args`
gains `--permission-prompt-tool stdio` when `TurnRequest.Mode` is `prompt`, a new field set from the session's
mode beside `PermissionMode`, and never `--permission-prompts` in any mode; `ClaudeCode.ExitTimeout
time.Duration`; the open-request set from every `prompt` event read and every `Answer` written; at the `result`
line: answer each open request deny with `interrupt` true and the placeholder, close stdin once, then with any
request open call the backend's interrupt at once, else start a timer of `ExitTimeout` whose expiry calls it; at
exit the writer is closed if still open; events after `result` are parsed and pushed as today, the exit last);
`internal/cli/engine.go` (`ExitTimeout` set from `spawn.interruptTimeoutSeconds` where `spawn.Config.InterruptTimeout`
is, line 455 today; `--requests allow|deny` on `engine run --fake`, `engine run --real` and `engine record` under
`--mode prompt`, default `deny`, the stderr line, a value under another mode `bad-args`); `claudecode_test.go`
against `internal/spawn/spawntest` (the lifetime test: stdin stays open through an open request, closes once
after the deny-interrupt line at `result`, never twice, the interrupt called at once when a request was open and
not when none, the interrupt called at `ExitTimeout` when no exit follows a clean close, closes at exit without
`result`, and a policy process still gets the write-then-close; the args test: `--permission-prompt-tool stdio`
under `prompt` only, `--permission-prompts` under no mode); `fake.go` (a `prompt` line from the fixture is
emitted, then the fake waits for an `Answer` call when the recording's result for that `tool_use_id` precedes the
turn's first `result` line, else streams on; the recording is replayed whole, lines after its first `result`
included, the interrupt stopping it before the next line as today; `received-<n>.json` gains `answers`, the
lines received in order); `fake_test.go` (`prompt-ask-allow` waits, answered after a delay, then streams;
`prompt-ask-background` streams both results without an answer); `record.go` (the raw line kept after `result`
as today; `fixture.json` carries `requests` as the policy used). Debug surface: `engine run --fake
fixtures/engine/prompt-ask-background --turn 1 --prompt x` printing the `prompt` event, both results and the
exit; `engine run --fake fixtures/engine/prompt-ask-allow --turn 1 --prompt x --expect result` printing one
stderr `request ... deny` line (the default policy answers the fake's wait at once, so the command never
hangs); `go test ./internal/engine -run 'Lifetime|Args|Answer|Fake'`. Proof: the build, check and test lines; the
lifetime and args tests; the two `engine run --fake` lines; `received-1.json` of a `prompt-ask-allow` replay
holding one deny line under `answers`. Does not: post a prompt, keep a pending map or a timer (P40d); read the
first `result` only in `session` (P40d); run a real turn. Depends on: P40b.

## P40d. `session`: the key, the pending map, the prompt, the bound, the drop at `result`, Stop, the deciders, the detector (large; mid-pass commit)

Implements: ADR 0029 rules 4 to 8 and Amendments 2026-11-07 decisions 1 to 3 (`session`'s half); the companion
(What the bot does with a request; The turn's end, the first-result rule; Stop, end, shutdown; What the bot
sees); `05-debug-interface.md` (Configuration); `05-debug-interface-session.md` (`session answer`, `show`,
`tail`, `interrupt`, `end`). This pass adds, in the same commits: to `03-architecture.md` (Configuration) the
key `subagentPromptTimeoutMinutes` beside `promptTimeoutMinutes` in the example and the inheritance sentence; to
`04-ux.md` (Permission prompts) the agent line after the turn number, the wording the owner approved in the
proposal's Owner items of 2026-11-07 (the agent type after the turn number). First, its own commit:
`internal/config/types.go`, `effective.go`, `validate.go`, `load.go`, `setuptemplate.go`, `scratchtemplate.go`
(`subagentPromptTimeoutMinutes` on `defaults` and each channel, inheriting as its siblings; absent from both is 60;
0 and a negative value are `bad-config: <file>: <key>: <value>, want a positive number of minutes (0 is not never
here)` from `config check`, `config reload` and `serve` alike; above 1440 capped at 1440 with a `warning:` line;
`config show --effective` prints it); `tests/config/` (a 0, a negative, a 2000); `internal/cli/config_test.go`.
Then [no-check: the script and tests are this pass's]: `internal/session/turn.go` (`TurnRequest.Mode`; on a
`prompt` event in a prompt turn: the `deferred` row, `pending[request_id]` with tool, `tool_use_id`, target, agent
type through the parent call's `subagent_type` else `AgentID`, the timer from the channel's key, the
`waiting-for-answer` event indented under the `Agent` line, the state `waiting-for-answer` while any request is
open; the at-once allow for a session-allowed tool (never a `web` tool: a session allow does not cover `web`); at `result`: every pending request `dropped`, decider
`instance`, cause `the turn ended`, the prompt's buttons removed, the timer cancelled, the turn ended on that
`result`; the first-result rule in every mode: `lastResult` becomes the first `result` of the process and
`engine.UsageOf` reads the first, every event after it logged and never classified, never a decision row, a
prompt or a resume, the exit event last); `actions.go` (`Answer` on a live request by `tool_use_id`,
`--tool-use-id` required with two or more pending, `allow-session` joining the one list and answering at once;
`Interrupt` while a live request is open: each pending request answered deny with `interrupt` true, recorded
`deny` with the actor as today's Stop on a deferred call, `participant` or `host`, reason `interrupted`, then the
backend's interrupt at once through the running handle and the wait for `turn-ended` as the running-turn path
does today; `end`, the idle end and `Shutdown` the same with `dropped`, decider `instance`, and 05's cause; any
request, `result` or `deferred_tool_use` after the denies logged, not acted on; the outcome `interrupted`, the
usage S2's); `timeout.go` (one timer per request on the service's clock; the expiry's deny, `interrupt` false,
the ADR's reason and `No answer; denied`, decider `prompt-timeout`); `classify.go` (the record-first rule: a
result for a call in the bot's request record takes the record's decider and reason whatever `source` it carries
or omits; with no record, `user_*` is `hook bypassed` cause `engine answered`, `config` with `ReasonType` `other`
is `hook bypassed` cause `engine aborted`, the cause in the row's `reason` as today, no store change); the
detector (a hook-answered `ask`, the hook log's `ask` for the id or `decision_reason_type` `hook` on a
`system/permission_denied` line, reaching a result with no `control_request` seen: `internal-error` with the
ADR's summary, the audit row `engine-prompt-tool-missing`, the trace line naming the lift); `query.go` (`session
show`'s `pending:` lines with `agent <type>`); `prompt.go` (the header's foreground sentence, ADR 0029 rule 7,
placeholder); `internal/discord` (the prompt's agent line after the turn number; a click on a live request
resolved by `tool_use_id` as today; the dropped line); `classify_test.go`, `prompt_mode_test.go`,
`timeout_test.go` (each fixture of the companion's table driven through the service: allow, deny, two out of
order, session-allow at once, the expiry with `ManualClock`, the drop at `result` and the first-result rule on
`prompt-ask-background`: one `turn-ended` with the first `result`'s usage and no row for the second turn's call;
Stop on `prompt-ask-interrupt`: the deny line under `answers`, `turn-ended  1  interrupted`, the row `deny
(participant)` with reason `interrupted`, the exit before the settle; the kill fixture; the detector on
`prompt-ask-no-tool`; the record-first decider on `prompt-ask-abort`); `tests/repro/subagent-prompt.sh` (a
scratch root with `subagentPromptTimeoutMinutes` 1 on channel `100`: a session on `prompt-ask-allow`, `session
send --wait` printing `waiting-for-answer`, `session show`'s `pending:` line with `agent probe`, `session answer
s0001 allow --wait` printing `turn-ended  1  completed`, `session tail` showing the indented rows with
`(participant)`; a session on `prompt-ask-two` answered deny then allow by `--tool-use-id`; a session on
`prompt-ask-background` whose `session tail` shows `dropped ... (instance)` and exactly one `turn-ended`; a
session on `prompt-ask-interrupt` with `session interrupt` printing `turn-ended  1  interrupted` and a tail row
`deny ... (participant)`; a session on `prompt-ask-no-tool` whose `audit --kind engine-prompt-tool-missing` holds
one row; `config check` on the three `tests/config/` trees) and its `tests/README.md` row. Debug surface: the
companion's. Proof: the build, check and test lines; `scratch run -- bash tests/repro/subagent-prompt.sh` exit 0;
`scratch run -- bash tests/repro/defer-resume.sh` and `full-session.sh` unchanged; `config check --root
tests/config/<zero>` exit 1 naming the key; the two doc additions in the diff. Does not: change a top-level
prompt (its year-long default, R-53, and the `defer` round trip stand); carry `updatedPermissions`; read a second
`result` as a turn; change the store's schema; wait for a `result` on Stop; word anything. Depends on: P40c,
P12b (the prompt's buttons), P35 when built (the subagent `land` row is P35's).

## P40e. The real turns: the `defer` fixtures under the flag; one prompt-mode session with a subagent's call (small; the owner's word)

Implements: ADR 0029 rule 10's last clause (the existing defer fixtures re-recorded under the flag, on the owner's
word) and its predictions; `fixtures/engine/README.md`. On the owner's word, through the `run` skill on the dev
root: `engine record --mode prompt --requests deny` re-recording `defer` (four turns: the deferred Write, the
allow resume, the second defer, the deny resume) and `parallel` (two turns) on the pinned engine with the flag
on, the import's rewrite and read as P40b, each replacing its 2.1.296 directory and README row (`fixture.json`
keeps the claim); then one live prompt-mode session on the dev root in which the coordinator launches a
`general-purpose` subagent whose `Bash` is answered allow from `session answer`, its `session tail` pasted, which
closes the ADR's second prediction's first half for this pin and is `prompt-ask-allow`'s live twin. A claim the
build raised that only a real turn can settle is added here, not to a new spike. Proof: `go test
./internal/engine` with the goldens regenerated and the diff read (every changed line from the new recordings);
`scratch run -- bash tests/repro/defer-resume.sh` exit 0 on the re-recorded `defer`; the pasted tail. Does not:
run without the owner's word (the pass is complete without this brief; the handoff lists it); touch `~/.claude`.
Depends on: P40d.
