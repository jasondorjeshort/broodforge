# Handoff

Coordinator state. Subagents follow their brief, not this file.
Updated: 2026-10-16, at commit fc1157e, by main session

## Now

- Clean stopping point. P0 to P11 are committed and reviewed (`08-implementation-plan.md`; P10 at
  `33e965c` and `998851f`, P11 at `e2e6de6`). Nothing is in flight. `origin/main` is at `c28155a`;
  everything after it is unpushed and review-clean. Push only on the owner's words.
- The owner is closing sessions, then rebooting (operator entry of 2026-10-16); passes stopped at his request.
- Owner: Jason Dorje Short <jdorje@gmail.com>, repo-local identity; UX is his, technical calls mine.
- Fixtures are read by agents, not the owner (`9accf2d`); commands register from an `init()` in their
  file; `-race` needs the ucrt64 PATH; Linux has never run the tests (droplet first, `07-spike-droplet.md`).

## Next

1. Pusher pushes `main` on the owner's words, launched fresh with his request quoted.
2. Owner: P7's real turns (`internal/cli/owner_test.go` prints the credential and login line;
   then one `engine run --real`, one `engine record`); record a `closing` fixture while at it.
3. P12 (Discord live) needs the bot token and a test guild from the owner; its leftovers from P11
   are in `docs/ideas/2026-10-16-p11-remainder-for-the-live-gateway.md`.
4. Tester or coder: the P10 remainder idea's `Plan:` items (needs-login, interrupt kill, live
   reconcile); the instance-credentials add path idea (P8 and P12 need it).
5. Owner: droplet entries, the PC account entry; then droplet claims, P14 (held), `spike/` deleted;
   P13 after P12 and the owner's account.

## Waiting on user

- The push; the reboot; P7's real turns; the bot token and test guild for P12.
- Wording: the hook no-answer phrase (one phrase for both, recommended); first names vs full names
  in Discord; "acts freely" short or long; trace lines above Stop lines and under an answered prompt.

## Open findings

- `docs/findings/2026-10-15-spike-login-claims-for-adr-0020.md`,
  `docs/findings/2026-10-15-spike-stream-and-process-claims-for-adr-0020.md`,
  `docs/findings/2026-10-15-spike-hook-flow-claims-for-adr-0020.md` (PC rows hold, droplet open)
- `docs/findings/2026-10-15-spike-claims-for-adr-0022.md` (PC rows hold, droplet, C37, key half)
- `docs/findings/2026-10-16-prior-art-chat-front-ends-for-coding-agents.md` (relay decided; L1-L4 mine)
