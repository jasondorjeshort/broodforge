---
date: 2026-10-30
kind: feature
related: [2026-10-30-owner-less-weeks.md]  # that one is the weeks-away vision; this is the check that lets it land what passes
---

# Golden sessions replay recorded sessions against the new build

Nightly, or on demand, replay recorded sessions (their `session tail --json` events and outboxes)
against the current build in a scratch venue and diff the outboxes against the recorded ones, so a
UX or render regression shows as a diff. The replay tests exist in miniature: `tests/discord/*.jsonl`
and `discord render --events` feed one event stream through the renderer and compare. This takes
that from single cases to whole sessions.

Anchor: `internal/discord/render.go`; `tests/discord/`;
`docs/design/05-debug-interface-discord.md`.

## Open

- The diff's form: byte-exact, or normalised for ids and times.
- Recordings hold conversation text; they need the same rewrite as engine fixtures before they
  are kept.
- An intended UX change makes every golden differ; a re-bless command.

Plan: coder, P32 of `docs/design/08-implementation-plan-golden-sessions.md` (design: `docs/design/05-debug-interface-discord.md`, Golden sessions) [no-check: the pass creates the files named]. `discord render --golden <dir> [--update] [--expect <token>]` renders `<dir>/events.jsonl` on the fake clock and compares the reduced posts against the blessed `outbox.jsonl` and each session's transcript against its `transcript-<session>.md`, line by line (text exact, `time` included; `seq` not compared, `payload` not kept; ids are the replay's `m<n>`), printing per differing file the first differing position as `<file>  <line>  golden|now  <text>` records plus a `more` count, then `error: golden-differs`, exit 1; `--golden` beside `--events` or `--clock`, or neither flag, is a usage error, exit 2; `--update` re-blesses and `git diff tests/discord/golden/` is the diff a human reads. Files: `internal/discord/golden.go` and `golden_test.go` (`TestGolden`, one `Errorf` per file, `-update` as the engine's parser goldens; `TestGoldenTranscriptOneSession`), `internal/cli/discord.go` and `discord_test.go` (`TestGoldenFlagsExclusive`; the moved `burst` paths), `tests/discord/{burst,subagent-end,turn-end}.jsonl` moved to `tests/discord/golden/<name>/events.jsonl` and blessed, the `TestReplay*` paths, `importfixture --events` (the P1 rewrite over one events file, every whole-token occurrence of the captured session id to `s0001`; `main_test.go`'s `TestEventsRewritesEverySessionID`), `tests/repro/golden-sessions.sh`, `tests/README.md`, the moved paths in 05, P11's Files line and the P11-remainder idea. Proof: build, check and test lines; the named tests; `discord render --golden tests/discord/golden/burst --expect 'edits<=1/s'` exit 0, an edited golden line exit 1 with the two records then `git checkout`; `--update` on an unchanged golden leaves `git status` clean; the exclusive flags exit 2; `bash tests/repro/golden-sessions.sh` exit 0; the capture route end to end on a scratch root (no `--keep`) capturing `s0002` into a temp directory, the rewrite leaving no `s0002` token and P1's grep empty, nothing committed. `dev-read`, the real-engine golden, only after the owner's dev credential and the quoted go-ahead for one real turn: driver `1001 Ada Example` on `--credential cred-<dev>`, the id `session start` prints, a prompt naming `go.mod`, `session end <id> --no-closing-turn`, then the independent read and the owner's decision on it before the commit. This file is deleted in the pass's commit.
