# fixtures/engine/

Recorded Claude Code stream-json turns, replayed by the fake engine. Each directory holds
`turn-<n>.jsonl` (the engine's stdout, verbatim but for paths) and `fixture.json` (`claim`,
`engineVersion`, `recorded`, and `turns: [{n, exit, engineSessionId}]`). The spike recorded them
on the PC with engine 2.1.296; the import rewrote the PC home to `/home/user`, the spike root to
`/home/user/bf-spike`, and the engine's project slug to match, and changed nothing else.
`events-<n>.jsonl` is the parser's golden for `turn-<n>.jsonl`, one `EngineEvent` per line;
`go test ./internal/engine/` compares them field by field, `-run TestGolden -update` rewrites them.

Import one spike fixture (refuses a non-empty destination unless `--force`):

    go run ./internal/tools/importfixture [--force] <spike root>/fx/<name> fixtures/engine/<name>

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
