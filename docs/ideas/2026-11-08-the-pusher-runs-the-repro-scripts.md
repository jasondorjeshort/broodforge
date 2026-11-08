---
date: 2026-11-08
kind: chore
---

# The pusher runs the repro scripts

`tests/repro/*.sh` (`defer-resume.sh`, `full-session.sh`, `auto-mode.sh`, `subagent-prompt.sh`) are
the end-to-end proofs the passes name, and none is in `go test`. On 2026-11-07 P40c's held-open stdin
made `auto-mode.sh` hang at its real-engine step; the full `-race` suite stayed green, the pusher's
eight checks passed, and two pushes went out before the P40d coder met the hang by running the script.
The pusher's brief that day named the scripts by hand; the next pusher will not unless told.

Two shapes, pick one [no-check: the test file does not exist yet]: a Go test (`tests/repro/repro_test.go`, build tag or `-short` skip) that runs
each script through `scratch run` with a timeout and fails on a non-zero exit, so `go test ./...`
covers them and the pusher's check 8 does without a word; or a line in the pusher's check list (a
Hatchery carry-back: `.claude/agents/pusher.md` is base-owned) that runs every `tests/repro/*.sh` the
project lists in `tests/README.md`. The Go test is the one that also runs on a coder's own check line.

Anchors: `tests/README.md` (the scripts' rows), `.claude/agents/pusher.md`, `internal/cli/scratch.go`
(`scratch run`), `tests/repro/auto-mode.sh` (step 6's `--expect result`, added the same day).

## Open

- The scripts take 10 to 40 s each and need `MSYS_NO_PATHCONV=1` and `CLAUDE_CODE_GIT_BASH_PATH` on the
  PC; a Go wrapper sets both.
- `auto-mode.sh` spends no real turn (the stand-in), so the suite stays free; keep it that way.
