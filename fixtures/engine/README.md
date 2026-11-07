# fixtures/engine/

Recorded Claude Code stream-json turns, replayed by the fake engine. Each directory holds
`turn-<n>.jsonl` (the engine's stdout, verbatim but for paths) and `fixture.json` (`claim`,
`engineVersion`, `recorded`, and `turns: [{n, exit, engineSessionId}]`). All but the hand-written `subagent*`, `reviewer-*` and `auto-*` fixtures were
recorded on the PC with engine 2.1.296, by the spike or by `engine record` (claim `recorded`); the
import rewrote the PC home to `/home/user`, the spike root to `/home/user/bf-spike`, and the
engine's project slugs to match, and changed nothing else. `subagent` and its copies `subagent-pinned` and `subagent-pinned-denied` (claim `hand-written`) are
written by hand in the stream's shape until a real subagent turn is recorded (ADR 0025, check C6);
`reviewer-defer` and `reviewer-empty` likewise until the engine is seen deferring an `Agent` call
(ADR 0028, V1); `auto-refused` and `auto-prompt` likewise until the first real `auto` turn shows
how the engine's own check stops a call in `-p` (claim C40 of ADR 0020, P39): that recording
replaces the one of the shape it shows, and the other is deleted.
A hand-written fixture carries no `engineVersion` and no `recorded`: no engine produced it. Its
shape is assumed, not observed: that a subagent's own lines carry `parent_tool_use_id`, and that
its results carry `tool_result_meta` with `source: hook` (the bot's hook deciding a subagent's
call), are unproven until the owner's C6 turn; replace it with that recording.
`events-<n>.jsonl` is the parser's golden for `turn-<n>.jsonl`, one `EngineEvent` per line;
`go test ./internal/engine/` compares them field by field, `-run TestGolden -update` rewrites them.

Import one recorded directory (refuses a non-empty destination unless `--force`, and any output
that still names the PC user):

    go run ./internal/tools/importfixture [--force] [--user <pc user>] <src> fixtures/engine/<name>

`<src>` is a spike fixture, `<spike root>/fx/<name>`, whose `meta-<n>.json` files build
`fixture.json` with the claim the tool knows for `<name>`; or a directory `engine record` wrote,
whose own `fixture.json` is copied through the same rewrite. The PC user is `--user`, else read
from the source path's `C:/Users/<user>` prefix. Read the result as `docs/design/05-debug-interface.md`
(`engine record`) says before committing it.

| Fixture | Claim | Turns | What it shows |
|---|---|---|---|
| `basic` | S1 | 3 | A Read turn, a resumed turn, and a turn asking for AskUserQuestion: the stream shape, the `modelUsage` delta over a resumed session, AskUserQuestion absent under `-p`. |
| `defer` | C27 | 4 | Prompt-mode `defer` round trip: turns 1 and 3 end deferred; turn 2 resumes with allow and the Write runs; turn 4 resumes with deny and the reason reaches the model wrapped. |
| `interrupted` | S2 | 3 | Turn 1 the engine blocks a standalone `sleep` before the hook (`<tool_use_error>`) and the kill lands after the `result`, exit 1; turn 2 a ping killed mid-call with no `result`, exit 1; turn 3 the resume reruns it to a `result`. |
| `parallel` | C28 | 2 | Two Writes in one message, both deferred, `result` naming the last; on resume the answered call runs and the other is re-issued and deferred again. |
| `hook-exit2` | C39 | 1 | A hook exiting 2 blocks Read and Bash; the reason reaches the model wrapped with `[<hook command>]: `. |
| `hook-exit1` | C39 | 1 | A hook exiting 1 falls through to the engine's own permission check: Read and Bash run, the Write is denied for want of a grant. |
| `hook-sleep` | C39 | 1 | A hook that outlives its timeout falls through the same way as exit 1. |
| `scrub` | C34 | 1 | Under `CLAUDE_CODE_SUBPROCESS_ENV_SCRUB=1`, no `ANTHROPIC_*` was set, and no `CLAUDE_CONFIG_DIR` reaches Bash. |
| `settings-a` | C26 | 1 | The hook given by `--settings <file>` runs under `--setting-sources user` with an empty user settings file (the Read's `tool_result_meta` source is `hook`). |
| `settings-b` | C26 | 1 | The same with `--setting-sources ""`. |
| `recorded-read` | recorded | 1 | `engine record` through the owner's credential: a Read of `hello.txt` the hook allows, answer `hello`. |
| `closing` | recorded | 1 | The closing turn, no tools: a pull-request title line, a blank line, a Markdown body. |
| `subagent` | hand-written | 1 | The driver launches the `reviewer` through `Agent`; the subagent's `Read` and text carry `parent_tool_use_id`, then the `Agent` result: `session tail`'s indentation and `session show`'s `review:` line (ADR 0025). |
| `subagent-pinned` | hand-written | 1 | `subagent` with the `Agent` input pinning `"model":"haiku"`, the lines carrying `parent_tool_use_id` naming `claude-haiku-5-5`, and the result's `modelUsage` split between Opus and Haiku: the `pin` row of `model_seen`, the `subagent-result` row's `model`, both trailer lines (ADR 0025 rule 10 as amended 2026-10-31, P38). |
| `subagent-pinned-denied` | hand-written | 1 | `subagent-pinned` whose `Agent` launch the hook denies (`Denied by Broodforge:`, in `permission_denials`), with no Haiku lines and Opus alone in `modelUsage`: a denied launch credits nothing, its `pin` row deleted and the Haiku line gone from the trailers file (P38). |
| `reviewer-defer` | hand-written | 2 | Turn 1 ends `tool_deferred` on an `Agent` call whose `subagent_type` is `reviewer` (the `reviewer` cell); turn 2 is the bot's resume with `allow`: the subagent's `Read` and text under `parent_tool_use_id`, the `Agent` result `No findings.`: the pin at the defer, `session show`'s `review: <id> at <commit>` (ADR 0028 Decision 3, P33). |
| `auto-refused` | hand-written | 2 | Shape B of the hook contract's Auto mode, assumed: turn 1 `init` with `permissionMode` `auto`, a Write of the relative `hello.go` (under any session's tree) the hook passed, refused with no prompt by a `system/permission_denied` line and an error `tool_result` whose `tool_result_meta` source is `config` (the form C40a saw under `default`; the classifier's own source is unobserved), the Write in `permission_denials`; turn 2 `init` with `permissionMode` `default`, text only, the turn after `session mode <id> prompt`. The `engine` decider, and the `engine-mode-mismatch` row under a policy session (P39). |
| `auto-prompt` | hand-written | 2 | Shape A, assumed: turn 1 `init` with `permissionMode` `auto`, a Bash call, then a `control_request` (`request.subtype` `can_use_tool`, `tool_name`, `input`, `tool_use_id`, `decision_reason`) and nothing more, the engine waiting until the bot interrupts it (exit 1); turn 2 the resume with the answer `allow`: the same `tool_use_id`'s result with source `hook`, then `init` and text. The call deferred by `engine` with the reason, `waiting-for-answer`, the answer resuming it by id (P39). |
| `reviewer-empty` | hand-written | 2 | `reviewer-defer` whose `Agent` result has no text: the row is written with `empty` and the fact stays `review: none since <commit>` (P33). |
