# Handoff

Coordinator state. Subagents follow their brief, not this file.
Updated: 2026-10-16, at commit f8eff6e, by main session

## Now

- `origin/main` is at `c28155a` (pushed 2026-10-16). Everything from `36d023a` to HEAD is
  unpushed and reviewed (docs commits, P1 to P4). Push on the owner's words. HOLD 60%/75%.
  This session runs `-race` with `PATH="/c/msys64/ucrt64/bin:$PATH"`.
- Owner: Jason Dorje Short <jdorje@gmail.com>, repo-local identity; UX is his, technical calls mine.
- Go settled 2026-10-16 (ADR 0023); P0 to P10 committed (P10 `33e965c`, `998851f`; all reviewed;
  Linux tests not yet run, the droplet runs them first). P10's remainder is an idea file. P7's
  real turns are the owner's and have not run (`internal/cli/owner_test.go` has the steps).
  P11 coder in flight (Discord dry-run renderer, outbox; the owner reads its outbox text).
- PC spike closed: every PC row holds; droplet rows and C37 open. Since 2026-10-16 the spike
  directory and this PC's Claude Code are both the owner's account (his words, 2026-10-16),
  so the status-line meters are the spike's cost too.
- Prior-art: build. Relay off by default, API keys the droplet fallback (ADR 0020 last amendment).

## Next

1. P11 in flight: review, fold fixes, commit; the owner reads the outbox text and the string
   list (hook no-answer phrase, interrupt-while-pending, "acts freely") and his word applies.
2. Owner: the droplet entries (Go, gcc, Node for the spike); the PC account half of its entry.
3. Droplet claims (`07-spike-droplet.md`), then P14 (held) and the deletion of `spike/`.
4. Pusher pushes `main` on the owner's words, at session end; the whole stack is review-clean.

## Waiting on user

- Droplet entries; the PC account entry; the next push; P7's real turns (his plan, his word);
  the trace phrase when the hook gives no answer (recommended: one phrase for timeout and failure).

## Open findings

- `docs/findings/2026-10-15-spike-login-claims-for-adr-0020.md`,
  `docs/findings/2026-10-15-spike-stream-and-process-claims-for-adr-0020.md`,
  `docs/findings/2026-10-15-spike-hook-flow-claims-for-adr-0020.md` (PC rows hold, droplet open)
- `docs/findings/2026-10-15-spike-claims-for-adr-0022.md` (PC rows hold, droplet, C37, key half)
- `docs/findings/2026-10-16-prior-art-chat-front-ends-for-coding-agents.md` (relay decided; L1-L4 mine)
