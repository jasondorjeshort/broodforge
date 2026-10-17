# fixtures/engine/

Recorded Claude Code stream-json turns, replayed by the fake engine. Each directory holds
`turn-<n>.jsonl` (the engine's stdout, verbatim but for paths) and `fixture.json` (`claim`,
`engineVersion`, `recorded`, and `turns: [{n, exit, engineSessionId}]`). All were recorded on the
PC with engine 2.1.296, by the spike or by `engine record` (claim `recorded`); the import rewrote
the PC home to `/home/user`, the spike root to `/home/user/bf-spike`, and the engine's project
slugs to match, and changed nothing else.
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
