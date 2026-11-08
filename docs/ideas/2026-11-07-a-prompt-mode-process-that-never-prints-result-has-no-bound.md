---
date: 2026-11-07
kind: feature
related: [2026-11-07-the-post-edit-hook-is-skipped-by-bash-edits-and-hangs-when-run-bare.md]
---

# A prompt-mode engine process that never prints `result` has no bound

Under ADR 0029 rule 2 a prompt-mode process keeps stdin open until the adapter closes it at
`result` (or at exit). An engine that never prints `result` and never exits, with stdout still
open, is therefore held for ever: the stream never ends, so neither the close at the stream's end
nor `ExitTimeout` (armed from `result`) can fire. The P40c coder met the shape while fixing
`tests/repro/auto-mode.sh` (a stand-in that read no fixture and waited for the close: a mutual
wait) and could not write the test for it without a new rule. A real engine that hangs mid-turn
hangs the same way in policy mode today, so this is not a regression, but prompt mode adds the
held stdin to the picture and the companion names no idle bound.

Anchors: `internal/engine/claudecode.go` (`atResult`, `closeAtExit`, `ExitTimeout`);
`docs/design/03-hook-contract-prompt-tool.md` (The process; The turn's end); the finding
`docs/findings/2026-11-07-spike-subagent-prompt-stdio-claims-for-adr-0029.md` (C43a, C43b).

## Open

- Whether an idle bound belongs in the adapter (no stream line for N minutes with no request open:
  close stdin, then the interrupt) or in `session` (the turn's own idle rule), and what N is; an
  owner item if it stops a turn that would have finished.
- A companion clause either way; the stand-in test the P40c coder could not write.
