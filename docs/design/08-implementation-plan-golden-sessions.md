# Implementation plan: golden sessions

This file continues `08-implementation-plan.md` under the same conventions: the pass format, the size
words, the standing rules and the `[no-check: ...]` rule. It holds the pass that turns the replay
fixtures of P11 into golden sessions (P32): a recorded session's events replayed through the dry-run
renderer and its rendered outbox and transcripts compared line by line against the ones blessed when
the golden was made, so a render or wording regression is a diff an agent reads from the terminal and
every test line runs. The design is `05-debug-interface-discord.md` (Golden sessions); the idea is
`../ideas/2026-10-30-golden-sessions-replay-recorded-sessions-against-the-new-build.md`, deleted in
the pass's commit (the queue rule). The pass reads that section, the renderer section above it,
`internal/engine/golden_test.go` (the parser goldens' `-update` shape, which this pass mirrors) and
P1's rewrite and independent-read rules before its Implements line. The bot is never in the loop: a
golden replays through `discord.Replay` with the fake clock, standalone, and takes no root. A replay
file may hold several sessions (`burst` and `turn-end` do), and `discord.Transcript` reads one: the
golden keeps one transcript per session, each over that session's events alone.

## P32. Golden sessions: `discord render --golden`, the promoted fixtures, the dev-root capture (medium; one owner step for `dev-read`)

Implements: `05-debug-interface-discord.md` (Golden sessions); `05-debug-interface-session.md` (the golden's
input, one sentence under P11's paragraph); P1 of `08-implementation-plan.md` (the rewrite and the independent
read, applied to a captured events file); `05-cli-conventions.md` (Exit codes: a usage error is 2). Files
[no-check: every file under `tests/discord/golden/`, the script and the new Go files are created by this pass]:
`internal/discord/golden.go` (`GoldenPost`, a `Post` reduced to the compared fields, `time`, `op`, `session`,
`channel`, `messageId`, `length`, `first`, `body`, `buttons`, `selects`, `pinned`, `file`, marshalled without
HTML escaping as `encodeDiscord` does; `ReadGolden(dir)` reading `outbox.jsonl` and every `transcript-<session>.md`
with CRLF read as LF; `WriteGolden(dir, ...)` writing those files, removing a `transcript-<session>.md` whose
session the events no longer hold, and never `events.jsonl`; `CompareGolden(got, want) []GoldenDiff`, positional,
one `GoldenDiff{File string; Line int; Golden, Now string; More int}` per file that differs, its first differing
position with `-` for a missing side and `More` the count of further differing lines, a final newline not a
difference; `RenderGolden(events)` calling `Replay(events, true)` and, per session id in the events,
`Transcript(id, <that session's events>, who)` with the names the replay's views resolve); `internal/cli/discord.go`
(`discord render`: `events` leaves `required`, the handler refuses a call with neither `--events` nor `--golden`
as a usage error, exit 2, as it does `--clock slow`; `--golden <dir>` sets the events file to `<dir>/events.jsonl`
and the clock to fake, `--clock`'s default now empty meaning real, and `--golden` beside `--events` or `--clock`
is the same usage error; after the posts print, per differing file the two records as text columns `file`,
`line`, `side`, `text` and a `<file>  more  <n>` record when `More` is not zero, then `error: golden-differs:
<dir>: <files>; --update, then git diff tests/discord/golden/, shows the whole`, exit 1, or the pacing line and the
`--expect` judgement as before; `--update` writes through `WriteGolden`, prints `golden  updated  <dir>` and exits
0 without comparing; `--json` adds `golden`, one object per differing file with `file`, `line`, `golden`, `now`,
`more`); `internal/cli/discord_test.go` (its `burst` paths at the `render` tests, lines 18 and 23 today, moved;
`TestGoldenFlagsExclusive`: `--golden` with `--events`, `--golden` with `--clock fake`, and a `render` with neither
flag each exit 2 with the usage line, the reviewer's text); `internal/discord/golden_test.go` (`TestGolden`: every
directory under `../../tests/discord/golden/` rendered and compared, one `Errorf` per file that differs naming the
file, the line and the count of further differences, `-update` rewriting the output files as
`internal/engine/golden_test.go`'s flag does; `TestGoldenTranscriptOneSession`: every blessed `transcript-<session>.md`
holds exactly one `# Session ` heading, the reviewer's regression test); `tests/discord/golden/burst/`,
`subagent-end/`, `turn-end/` (`events.jsonl` by `git mv` from `tests/discord/<name>.jsonl`, then `outbox.jsonl` and
the transcripts blessed by `--update` and read by the coder line by line against the `TestReplay*` expectations
before the commit); `internal/discord/discord_test.go` and `ending_test.go` (`readReplay`'s paths moved; the
assertions unchanged); `08-implementation-plan-discord.md` (P11's Files line names `tests/discord/burst.jsonl`:
the moved path, as a pointer) and `../ideas/2026-10-16-p11-remainder-for-the-live-gateway.md` (its
`tests/discord/subagent-end.jsonl` path); `internal/tools/importfixture/main.go` (`--events <file>`: the same token
rewrite as a turn file's, in place over one JSONL file, plus every whole-token occurrence of the captured session
id, read from the `session-started` line, rewritten to `s0001`: the summaries, `start.branch`, `end.branch`, the
custom ids and the worktree paths in tool targets included; `--user` as before, read from the first PC home path
in the file when absent, and a file with no PC home path gets no home rewrite and is not an error; `<src> <dst>`
unchanged); `internal/tools/importfixture/main_test.go` (`TestEventsRewritesEverySessionID`: a file written by
the test as `s0002` with the id in a summary, `start.branch`, `end.branch`, a custom id and a worktree path in a
target holds no `s0002` token after the rewrite and keeps every other byte); `tests/repro/golden-sessions.sh`
(from the repository root, `bin/broodforge` or `BF`: one `discord render --golden <dir> --expect 'edits<=1/s'` per
directory under `tests/discord/golden/`, `ok  <name>` or `differs  <name>` with the records of each that differs,
exit 1 if any differs, no root); `tests/README.md` (the `discord/*.jsonl` rows replaced by one row per golden
directory and one for the script); `05-debug-interface-discord.md` (The Discord adapter, Reproduce: the moved
`burst` path); `tests/discord/golden/dev-read/` (the owner-gated tail below). Debug surface: `discord render
--golden <dir>`, `--update` then `git diff tests/discord/golden/`, `bash tests/repro/golden-sessions.sh`, `go test
./internal/discord -run TestGolden`.
Proof: the build, check and test lines; `go test ./internal/discord -run 'TestGolden|TestReplay'` (both golden
tests included) and `go test ./internal/cli -run TestGoldenFlagsExclusive`; `go test ./internal/tools/importfixture
-run TestEventsRewritesEverySessionID`; `bin/broodforge discord render --golden tests/discord/golden/burst --expect
'edits<=1/s'` exit 0 with the pacing line last; one character of a body changed in
`tests/discord/golden/burst/outbox.jsonl` makes the same line exit 1 with the two records naming `outbox.jsonl`
and the line and the error line naming the file, then `git checkout tests/discord/golden/burst` restores it;
`--update` on an unchanged golden leaves `git status` clean; `--golden ... --clock real`, `--golden ... --events x`
and a `render` with neither flag exit 2 with the usage line; `bash tests/repro/golden-sessions.sh` exit 0 with one
`ok` line per golden; the capture route end to end without a commit: `scratch run -- bash -c '...'` (no `--keep`)
runs `serve`, two policy-mode turns against `basic` on channel `100` as `1001` in two sessions each ended with
`session end`, so the second is `s0002`, then `session tail s0002 --json --all` into a directory under the OS
temp directory, `stop`; then `go run ./internal/tools/importfixture --events <that file>` leaves `grep -c s0002`
at zero and P1's `grep -rliE "<pc user>|[A-Za-z]:(/|\\\\)+Users"` over the file empty, `discord render --golden
<that directory> --update` writes the output files and `--golden` on it exits 0 (the directory is deleted after;
nothing of it is committed). `dev-read` only once `credential list` on the dev root shows the dev credential
`usable` and the brief quotes the owner's go-ahead for one real turn in so many words: the `run` skill's steps 0
to 2, `register 1001 ...`, `session start --channel 200 --as 1001 --roles 10 --credential cred-<dev>` printing
the id `<id>` the rest uses, one `session send <id> --as 1001 --wait "Read go.mod and reply with the module's
name."`, `session end <id> --no-closing-turn` (so the capture holds the one turn), the capture, the rewrite (the
id to `s0001`), the bless, `stop`; then the independent read by an agent that did not capture it, reported in a
few lines, and the commit after the owner's decision on that report, never before. Without the credential the
pass ends with the promoted goldens and the handoff names `dev-read` as waiting on the owner's step. Owner step:
the dev credential (`05-debug-interface-dev-instance.md`) and the go-ahead for the one real turn. Does not: replay
through `serve` or any bot, or round-trip Discord; add a diff library or any dependency; compare `seq` or keep
`payload`; change the renderer, the coalescer or any wording; add a `session` command (`session tail --json
--all` is the capture); record turns on the dev root by itself
(`../ideas/2026-10-30-real-turns-on-the-dev-root-seed-fixtures.md`); schedule anything; commit a golden before
its read. Depends on: P11; for `dev-read` only, P27 and the owner's credential step.
