# fixtures/engine/

Recorded Claude Code stream-json turns, replayed by the fake engine. Each directory holds
`turn-<n>.jsonl` (the engine's stdout, verbatim but for paths) and `fixture.json` (`claim`,
`engineVersion`, `recorded`, and `turns: [{n, exit, engineSessionId}]`). All but the hand-written `subagent`, `subagent-pinned`, `subagent-pinned-denied`, `reviewer-*`, `auto-*` and `prompt-ask-session-allow` fixtures were
recorded on the PC, by the spike or by `engine record` (claim `recorded`), with engine 2.1.296 except
`subagent-deferred`, `defer`, `parallel` and the `prompt-*` fixtures (2.1.319); the
import rewrote the PC home to `/home/user`, the spike root to `/home/user/bf-spike`, and the
engine's project slugs to match, and changed nothing else. The `prompt-*` fixtures are ADR 0029's
spike runs (`docs/findings/2026-11-07-spike-subagent-prompt-stdio-claims-for-adr-0029.md`) and
the 2026-10-23 subagent finding's `control-ask` run, under the stdio prompt tool: their streams
opened with the recording account's own `SessionStart` plugin hook (`system/hook_started`,
`system/hook_response`), not the engine's contract, which the import dropped and `fixture.json`'s
`dropped` names, and the import renamed that plugin, listed in each `init`, to `account-plugin`.
Each keeps the hook's stdin for the turn's subagent calls as `hook-stdin-<n>.json`, a JSON array in
call order, paths rewritten the same way; `TestPromptFixtures` asserts `agent_id` and `agent_type`
on each entry, each request's result meta, and ADR 0029 rule 8's invariant (03-hook-contract-prompt-tool.md,
Fixtures). `subagent` and its copies `subagent-pinned` and `subagent-pinned-denied` (claim `hand-written`) were
written by hand in the stream's shape before a real subagent turn was recorded (ADR 0025, check C6).
`subagent-deferred` is the first recorded one; whether it replaces them is the architect's call;
`reviewer-defer` and `reviewer-empty` likewise until the engine is seen deferring an `Agent` call
(ADR 0028, V1); `auto-refused` and `auto-prompt` likewise until the first real `auto` turn shows
how the engine's own check stops a call in `-p` (claim C40 of ADR 0020, P39): that recording
replaces the one of the shape it shows, and the other is deleted.
A hand-written fixture carries no `engineVersion` and no `recorded`: no engine produced it. Its
shape is assumed, not observed. `subagent-deferred` shows a subagent's own `user` and `assistant`
lines carrying `parent_tool_use_id`; that a subagent's tool results carry `tool_result_meta` with
`source: hook` (the bot's hook deciding a subagent's call) stays unproven, since its one call ended
at the defer with no result, until a recorded subagent call runs to a result.
At a `control_request` line the recording answered inside the turn (the call's result before the
turn's first `result`; the engine's own abort at a stdin close, source `config` and reason_type `other`, answers nothing) the fake waits for the handle's `Answer`, and `received-<n>.json` lists every
answer it got as `answers`; any other request streams on, and a recording is replayed whole, lines
after its first `result` included. `engine record --mode prompt` writes its `--requests` policy to
`fixture.json` as `requests`.
`events-<n>.jsonl` is the parser's golden for `turn-<n>.jsonl`, one `EngineEvent` per line;
`go test ./internal/engine/` compares them field by field, `-run TestGolden -update` rewrites them.

Import one recorded directory (refuses a non-empty destination unless `--force`, and any output
that still names the PC user):

    go run ./internal/tools/importfixture [--force] [--user <pc user>] [--turn2 <src2>] <src> fixtures/engine/<name>

`<src>` is a spike fixture, `<spike root>/fx/<name>`, whose `meta-<n>.json` files build
`fixture.json` with the claim the tool knows for `<name>`; a directory `engine record` wrote,
whose own `fixture.json` is copied through the same rewrite; or a spike run directory
(`stream.jsonl`, `hook-stdin.jsonl`, `driver.log`), turn 1, with `--turn2` naming turn 2's, whose
`fixture.json` is built from the streams (the claim the tool knows for the run's name, each
`init`'s version and session, the first timestamp's date) and `driver.log`'s exit (0 without one
when the stream holds a `result`). The PC user is `--user`, else read
from the source path's `<drive>:/Users/<user>` prefix. Read the result as `docs/design/05-debug-interface.md`
(`engine record`) says before committing it.

| Fixture | Claim | Turns | What it shows |
|---|---|---|---|
| `basic` | S1 | 3 | A Read turn, a resumed turn, and a turn asking for AskUserQuestion: the stream shape, the `modelUsage` delta over a resumed session, AskUserQuestion absent under `-p`. |
| `defer` | C27 | 4 | Prompt-mode `defer` round trip: turns 1 and 3 end deferred; turn 2 resumes with allow and the Write runs; turn 4 resumes with deny and the reason reaches the model wrapped. `engine record --mode prompt --requests deny` on the dev root, engine 2.1.319, 2026-11-07, the stdio prompt tool on: one engine session for all four turns (the 2.1.296 recording used two; the fake does not care), the resumes recorded with `--answer`, turn 4 with no `--reason` (the hook's own deny text); each allowed Write is followed by the PostToolUse hook's `attachment` line. |
| `interrupted` | S2 | 3 | Turn 1 the engine blocks a standalone `sleep` before the hook (`<tool_use_error>`) and the kill lands after the `result`, exit 1; turn 2 a ping killed mid-call with no `result`, exit 1; turn 3 the resume reruns it to a `result`. |
| `parallel` | C28 | 2 | Two Writes in one message, both deferred, `result` naming the last; on resume the answered call runs and the other is re-issued and deferred again. Recorded as `defer` (engine 2.1.319, 2026-11-07, turn 2 with `--answer` allow): C28 holds under the stdio prompt tool. |
| `hook-exit2` | C39 | 1 | A hook exiting 2 blocks Read and Bash; the reason reaches the model wrapped with `[<hook command>]: `. |
| `hook-exit1` | C39 | 1 | A hook exiting 1 falls through to the engine's own permission check: Read and Bash run, the Write is denied for want of a grant. |
| `hook-sleep` | C39 | 1 | A hook that outlives its timeout falls through the same way as exit 1. |
| `scrub` | C34 | 1 | Under `CLAUDE_CODE_SUBPROCESS_ENV_SCRUB=1`, no `ANTHROPIC_*` was set, and no `CLAUDE_CONFIG_DIR` reaches Bash. |
| `settings-a` | C26 | 1 | The hook given by `--settings <file>` runs under `--setting-sources user` with an empty user settings file (the Read's `tool_result_meta` source is `hook`). |
| `settings-b` | C26 | 1 | The same with `--setting-sources ""`. |
| `recorded-read` | recorded | 1 | `engine record` through the owner's credential: a Read of `hello.txt` the hook allows, answer `hello`. |
| `closing` | recorded | 1 | The closing turn, no tools: a pull-request title line, a blank line, a Markdown body. |
| `subagent-deferred` | recorded | 1 | `engine record --mode prompt` on the dev root, engine 2.1.319: the driver launches `general-purpose` on Haiku through `Agent`; the subagent's `Bash` (`echo hi`), under `parent_tool_use_id`, is deferred by the hook and the engine ends the subagent there, with no `tool_result` for the call and no `deferred_tool_use`; then the `Agent` result `(Subagent completed but returned no output.)` and an `end_turn` result (the 2026-10-23 subagent finding, Third look). Its `rate_limit_event` carries `seven_day` alone: an account with no five-hour window (ADR 0021, Amendments 2026-11-07). |
| `subagent` | hand-written | 1 | The driver launches the `reviewer` through `Agent`; the subagent's `Read` and text carry `parent_tool_use_id`, then the `Agent` result: `session tail`'s indentation and `session show`'s `review:` line (ADR 0025). |
| `subagent-pinned` | hand-written | 1 | `subagent` with the `Agent` input pinning `"model":"haiku"`, the lines carrying `parent_tool_use_id` naming `claude-haiku-5-5`, and the result's `modelUsage` split between Opus and Haiku: the `pin` row of `model_seen`, the `subagent-result` row's `model`, both trailer lines (ADR 0025 rule 10 as amended 2026-10-31, P38). |
| `subagent-pinned-denied` | hand-written | 1 | `subagent-pinned` whose `Agent` launch the hook denies (`Denied by Broodforge:`, in `permission_denials`), with no Haiku lines and Opus alone in `modelUsage`: a denied launch credits nothing, its `pin` row deleted and the Haiku line gone from the trailers file (P38). |
| `reviewer-defer` | hand-written | 2 | Turn 1 ends `tool_deferred` on an `Agent` call whose `subagent_type` is `reviewer` (the `reviewer` cell); turn 2 is the bot's resume with `allow`: the subagent's `Read` and text under `parent_tool_use_id`, the `Agent` result `No findings.`: the pin at the defer, `session show`'s `review: <id> at <commit>` (ADR 0028 Decision 3, P33). |
| `auto-refused` | hand-written | 2 | Shape B of the hook contract's Auto mode, assumed: turn 1 `init` with `permissionMode` `auto`, a Write of the relative `hello.go` (under any session's tree) the hook passed, refused with no prompt by a `system/permission_denied` line and an error `tool_result` whose `tool_result_meta` source is `config` (the form C40a saw under `default`; the classifier's own source is unobserved), the Write in `permission_denials`; turn 2 `init` with `permissionMode` `default`, text only, the turn after `session mode <id> prompt`. The `engine` decider, and the `engine-mode-mismatch` row under a policy session (P39). |
| `auto-prompt` | hand-written | 2 | Shape A, assumed: turn 1 `init` with `permissionMode` `auto`, a Bash call, then a `control_request` (`request.subtype` `can_use_tool`, `tool_name`, `input`, `tool_use_id`, `decision_reason`) and nothing more, the engine waiting until the bot interrupts it (exit 1); turn 2 the resume with the answer `allow`: the same `tool_use_id`'s result with source `hook`, then `init` and text. The call deferred by `engine` with the reason, `waiting-for-answer`, the answer resuming it by id (P39). |
| `reviewer-empty` | hand-written | 2 | `reviewer-defer` whose `Agent` result has no text: the row is written with `empty` and the fact stays `review: none since <commit>` (P33). |
| `prompt-ask-allow` | C42 | 1 | Run `c42-allow`: a subagent's `Bash` under a hook `ask`, the `control_request` with `agent_id` and `decision_reason_type` `hook`, answered allow; the result, source `user_temporary`, under `parent_tool_use_id`; the `Agent` result carrying the report (C42 foreground). |
| `prompt-ask-deny` | C42 | 1 | Run `c42-deny`: two requests in one subagent, the first denied with `interrupt` false (the message verbatim, source `user_reject`), the second allowed and run. |
| `prompt-ask-two` | C41b | 1 | Run `c41b`: two requests open at once from one message, answered out of order (the second allow, then the first deny), each result under its own id. |
| `prompt-defer` | C44, C51 | 2 | Runs `c44-c51` and `c51-resume`, stdin open: the coordinator's hook `ask` on a `Read` and a safe-listed `echo` raise requests with no `agent_id`, hook `allow` and `deny` raise none (C44), then a top-level `defer` ends the turn `tool_deferred` (C51); turn 2 the resume runs the call under the hook's `updatedInput`, no line written. |
| `prompt-ask-background` | C42 | 1 | Run `c42-bg`: a background subagent's request open at the coordinator's first `result`; the allow after it, the engine's second `init` and second `result`, all in turn 1: the first-result rule (ADR 0029, Amendments 2026-11-07). |
| `prompt-ask-abort` | C43a | 1 | Run `c43a`: stdin closed with a request open; the engine's own deny, source `config`, `reason_type` `other`; the subagent reports and the turn goes on to `end_turn`. |
| `prompt-ask-interrupt` | C49 | 2 | Runs `c49` and `c49-resume`: a deny with `interrupt` true; the subagent killed, its result the engine's stock text with no `tool_result_meta`, the `Agent` result `[Request interrupted by user for tool use]`, one more coordinator call, `end_turn`; turn 2 a resume relaunching the subagent, its request allowed. |
| `prompt-ask-kill` | C47 | 2 | Runs `c47-kill` and `c47-resume`: the tree kill with a request open, exit 1, no `result`; turn 2 the resume, nothing streamed for the dropped call. |
| `prompt-ask-no-tool` | recorded | 1 | The 2026-10-23 subagent finding's `control-ask` run (no stdio prompt tool): the hook's `ask` on a subagent's `Bash` becomes `system/permission_denied` and an error result, source `config`, with no `control_request`: ADR 0029 rule 8's detector. |
| `prompt-ask-session-allow` | hand-written | 1 | `prompt-ask-deny` with the first request answered allow (`Allow Bash this session`: its result `one`, source `user_temporary`), the reports quoting it, `permission_denials` empty; the second request answered allow at once, no prompt (ADR 0029 rule 5). Replaced when a recording exists. |
