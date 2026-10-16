# Handoff

Coordinator state. Subagents follow their brief, not this file.
Updated: 2026-10-16, at commit 36d023a, by main session

## Now

- `origin/main` is at `c28155a` (pushed 2026-10-16). Unpushed: `36d023a`, the relay decision,
  docs only, unreviewed. Push only on the owner's words. HOLD 60%/75%. Sessions older than the
  mingw install need a restart for `-race` (Windows reference, Go section).
- Owner: Jason Dorje Short <jdorje@gmail.com>, repo-local identity; UX is his, technical calls mine.
- Go settled 2026-10-16 (ADR 0023); design docs and the plan are in Go form; P0 is committed.
- PC spike closed: every PC row holds; droplet rows and C37 open. Since 2026-10-16 the spike
  directory and this PC's Claude Code are both the owner's account (ids compared, not his words),
  so the status-line meters are the spike's cost too.
- Prior-art verdict: build. Relay decided 2026-10-16: off by default, API keys the droplet
  fallback, Anthropic asked once the droplet has users (ADR 0020 last amendment). L1-L4 are mine.

## Next

1. Coder: P1 fixture import per `08-implementation-plan.md` (path rewrite incl. the project slug;
   a person reads every fixture before the commit; the idea file is deleted in that commit).
2. Coder: P2 CLI conventions and control socket, then P3 engine events and the fake engine.
3. Tester: split the ADR 0020 finding by topic (login; stream and process; hook flow) first.
4. Owner: the droplet entries (Go, gcc, Node for the spike); the PC account half of its entry.
   Mingw closed 2026-10-16 (`-race` runs on the PC); the two pruned npm links dropped (a Discord
   bot's dev links on this PC, nothing runs them).
5. Droplet claims (`07-spike-droplet.md`), then P14 (held) and the deletion of `spike/`.
6. Pusher pushes `main` on the owner's words, at session end (the module path carries his
   GitHub handle, already public in the repository URL).

## Waiting on user

- Droplet entries; a word that this PC's session is his account; the next push.

## Open findings

- `docs/findings/2026-10-15-spike-claims-for-adr-0020.md` (PC rows hold, droplet rows open)
- `docs/findings/2026-10-15-spike-claims-for-adr-0022.md` (PC rows hold, droplet, C37, key half)
- `docs/findings/2026-10-16-prior-art-chat-front-ends-for-coding-agents.md` (relay decided; L1-L4 mine)
