# Handoff

Coordinator state. Subagents follow their brief, not this file.
Updated: 2026-10-21, after the P12 commit, by main session (rooted in the repo, hooks loaded)

## Now

- P12 is committed on `main` (one commit, review fixes amended in), unpushed. Nothing has
  connected to Discord: `docs/operator/2026-10-21-discord-application-bot-and-test-server.md` is
  the owner's step before the first live run, which is P12's Proof second half.
- P12 left unbuilt: the relay (`not-built`), the post-restart notice in a venue (P11 remainder item 5).
- Subagents do not get the `LSP` tool (coder report of 2026-10-21); gopls helps the main session only.

## Next

1. Owner: the Discord operator entry, then the first live session (`serve --discord live` on the
   scratch-shaped root) while an agent reads `session tail` and `audit`; findings from it.
2. Pusher: push `main` when the owner asks.
3. Coder: the P11 remainder `Plan:` (items 1, 2, 6, 8), then `/usage`, the relay, the post-restart notice.
4. Carry-backs to Hatchery (`carry-back` skill), the `Plan:` ideas of 2026-10-21.
5. Tester: resolve the net-diff finding; `session end` runs the closing turn from the `closing`
   fixture (the P10 remainder idea's 2026-10-17 line); then its other `Plan:` items.
6. Architect, then coder: the earlier privacy layers against the base's ADR 0018 pieces. Triage
   the ideas of 2026-10-17 and the Questions of 2026-10-21. Architect: the surfaces ADR. Owner:
   the droplet and PC account entries; P14, `spike/`.

## Waiting on user

- P12 human-facing choice: a refusal to a plain message (a pasted key, the lobby hint) goes by
  direct message, falling back to a channel reply with a mention. Keep, or always in-channel?
- Which name and email go on this project's commits? (`privacy.identities`, as in Hatchery's policy.)
- The two `Question:` ideas of 2026-10-21 (coder Stop hook now or on trigger; Windows facts carry-back).
- Wording: the hook no-answer phrase; first vs full names in Discord; "acts freely" short or long.

## Open findings

- `docs/findings/2026-10-17-net-diff-privacy-scan-let-named-commits-go-public.md` (rewrite done; resolve)
- `docs/findings/2026-10-15-spike-login-claims-for-adr-0020.md`, `...-stream-and-process-...`,
  `...-hook-flow-...` (PC rows hold, droplet open); `2026-10-15-spike-claims-for-adr-0022.md` (droplet, C37, key half)
- `docs/findings/2026-10-16-prior-art-chat-front-ends-for-coding-agents.md` (relay decided; L1-L4 mine)
