# Handoff

Coordinator state. Subagents follow their brief, not this file.
Updated: 2026-10-16, at commit f8eff6e, by main session

## Now

- RUNS PAUSED on this PC: BullGuard Sentry quarantined `bin/broodforge.exe`, Git's `sh`/`sed`/
  `head`/`find`/`cp`/`seq` and a run's outputs on 2026-10-16 19:28 (quarantine 681). No binary,
  `go test` or scratch run until the owner restores and excludes (operator entry of 2026-10-16).
- `origin/main` is at `c28155a` (pushed 2026-10-16). Everything from `36d023a` to HEAD is
  unpushed and reviewed. Push on the owner's words. HOLD 60%/75%. `-race` needs the ucrt64 PATH.
- Owner: Jason Dorje Short <jdorje@gmail.com>, repo-local identity; UX is his, technical calls mine.
- Go settled 2026-10-16 (ADR 0023); P0 to P10 committed (P10 `33e965c`, `998851f`; all reviewed;
  Linux tests not yet run, the droplet runs them first). P10's remainder is an idea file. P7's
  real turns are the owner's and have not run (`internal/cli/owner_test.go` has the steps).
  P11 coder in flight (Discord dry-run renderer, outbox; the owner reads its outbox text).
- PC spike closed (PC rows hold; droplet rows, C37 open); the spike and this PC run on the
  owner's account since 2026-10-16, so the status-line meters are the spike's cost too.
- Prior-art: build. Relay off by default, API keys the droplet fallback (ADR 0020 last amendment).

## Next

1. P11 in flight: review, fold fixes, commit; the owner reads the outbox text and the string
   list (hook no-answer phrase, interrupt-while-pending, "acts freely") and his word applies.
2. Owner: droplet entries, the PC account entry; then droplet claims, P14 (held), `spike/` deleted.
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
