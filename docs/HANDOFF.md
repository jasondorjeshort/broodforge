# Handoff

Coordinator state. Subagents follow their brief, not this file.
Updated: 2026-10-23, after the idle-timeout commit, by Broodforge session s0005

## Now

- P12a (model and effort from Discord, R-29 now v1) is pushed on `main` (9eb3b73, reviewed, fixes
  folded in). The live bot must be rebuilt (`bin/`) and restarted to register `/model` and
  `/session model|effort` and to migrate the store to schema 2.
- Idle timeout (owner, 2026-10-23): `idleTimeoutMinutes` 0 or absent is never, the new default;
  `/session timeout` and `session timeout` change a running session's (04, Idle timeout). Held in
  the bot's memory, not the row. `promptTimeoutMinutes` likewise: 0 or absent is never, the
  default (a prompt waits for its answer; `/session stop` or `end` still deny it). The live
  `instance.json` still says 60 and 5: the owner sets both to 0 (and checks the channels').
  Rebuild and restart to register `/session timeout`.
- P12 left unbuilt: the relay (`not-built`), the venue's post-restart notice (P11 remainder item 5).
  Subagents do not get the `LSP` tool (coder report of 2026-10-21); gopls is the main session's.

## Next

1. Owner: rebuild, restart the bot, continue the live session with model switching (milestone 1 is
   editing Broodforge from `broodforge/discord`); review P12a's wording (04-ux.md, Model and effort).
2. Then the P11 remainder `Plan:`, `audit --follow`, `/usage`, the Discord-layer trace idea of
   2026-10-22; `/rename` and the live-status header when the owner says.
3. Carry-backs to Hatchery (`carry-back` skill), the `Plan:` ideas of 2026-10-21. Tester: resolve
   the net-diff finding; `session end` runs the closing turn from the `closing` fixture (the P10
   remainder idea's 2026-10-17 line); then its other `Plan:` items.
4. Architect, then coder: the earlier privacy layers against the base's ADR 0018 pieces. Triage the
   ideas of 2026-10-17 and the Questions of 2026-10-21. Architect: the surfaces ADR (P12a added a
   `session list` column with nothing to gate it). Owner: droplet and PC entries; P14.

## Waiting on user

- P12a: the wording above; `/model` alone also shows effort?; does a switch post name who switched?
  P12: a refusal to a plain message (a pasted key, the lobby hint) goes by direct message, falling
  back to a channel reply with a mention. Keep, or always in-channel?
- Which name and email go on this project's commits? (`privacy.identities`, as in Hatchery's policy.)
- The two `Question:` ideas of 2026-10-21; wording: hook no-answer phrase, first vs full names, "acts freely".

## Open findings

- `docs/findings/2026-10-17-net-diff-privacy-scan-let-named-commits-go-public.md` (rewrite done; resolve)
- `docs/findings/2026-10-15-spike-login-claims-for-adr-0020.md`, `...-stream-and-process-...`, `...-hook-flow-...`
  (PC rows hold, droplet open); `2026-10-15-spike-claims-for-adr-0022.md` (droplet, C37, key half);
  `2026-10-16-prior-art-chat-front-ends-for-coding-agents.md` (relay decided; L1-L4 mine)
