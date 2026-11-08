---
date: 2026-11-08
kind: feature
related: [2026-11-07-a-prompt-mode-process-that-never-prints-result-has-no-bound.md]
---

# One engine process per session, not per turn

Moonshot, not sanity checked. Today every turn spawns a fresh engine process (ADR 0020: one process per
turn, `--resume` by engine session id; `03-architecture.md` N-26) and a deferred call ends the process
until the answer resumes it. The P40e spike's `c42-bg` recording (`fixtures/engine/prompt-ask-background`)
shows the engine doing more when stdin stays open: an answer written after the first `result` started a
second coordinator turn in the same process, with a second `init` and `result`. ADR 0029's amendment
(decision 1) deliberately refuses that today (the first-result rule, the kill at `result`) and names a
second turn per process as a proposal of its own.

The idea is the full form: one engine process per session, held open across turns, each user turn
written to its stdin, no `--resume`. What it would buy: a turn starts in milliseconds instead of a
process spawn of about 250 MB and several seconds; the coordinator's context stays warm; a background
subagent survives the turn's end, which dissolves `docs/findings/2026-10-24-background-subagents-die-when-the-turn-ends.md`;
the defer round trip (C27) becomes an answer on the same process, as a subagent's already is. On the
bot's side each session is already one goroutine; the change is the engine's lifetime, the adapter's
reader and `session`'s turn boundary.

Anchors: `internal/engine/claudecode.go` (the lifetime, `atResult`); `internal/session/turn.go`
(`runProcess`, the first-result rule); `docs/design/decisions/0020-harness-language-library-toolchain.md`
(Mechanism, the one-process rule) and ADR 0029's amendment; `docs/design/03-architecture.md` N-26.

## Open

- A major choice (ADR 0012: a promised surface and ADR 0020's mechanism change): proposal, critic, ADR.
- What ends a held process: the idle bound (the related idea), end, Stop, and an engine that crashes
  mid-session (the resume path stays as the recovery).
- Memory: one idle engine per live session on the droplet, against today's zero between turns.
- Whether the engine documents a multi-turn stdin protocol at all, or `c42-bg` was an accident of the
  task-notification path; the spike claim that settles it.
