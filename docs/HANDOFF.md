# Handoff

Coordinator state. Subagents follow their brief, not this file.
Updated: 2026-10-23, after the first live session's review, by main session (rooted in the repo, hooks loaded)

## Now

- The bot runs the real engine from a scratch root. P12a and the idle and prompt timeouts (session
  s0005, owner-pushed, `97ad23f`) are on `main`; the timeout commits were reviewed after the push:
  `docs/findings/2026-10-23-first-live-session-commits-reached-main-unreviewed.md` (open).
- A Discord session loads none of the project's `CLAUDE.md`, agents or hooks
  (`docs/findings/2026-10-23-session-settings-sources-drop-claude-md-and-agents.md`). Until the design
  question is settled, treat everything from a session as unreviewed: a reviewer read and the pusher's
  checks before it reaches `main`.
- P12 left unbuilt: the relay, the venue's post-restart notice. Subagents get no `LSP` tool.

## Next

1. Owner: continue the live session (milestone 1: edit Broodforge from `broodforge/discord`); set the
   live `instance.json` timeouts to 0; a real root outside Temp, eventually.
2. Architect: a proposal for the two workflows in a session (the ideas of 2026-10-23: sessions load
   none of the project's rules; engine `auto` permission mode; landing on `main`).
3. Tester: one pass on the review's open defects (minutes overflow, read after unlock, doc gaps, env
   test, a config error starting on last-good); then the net-diff finding and the other `Plan:` items.
4. Then the P11 remainder `Plan:`, `audit --follow`, `/usage`; carry-backs; the surfaces ADR; triage.

## Waiting on user

- Wording: P12a and `/model` showing effort; "Never ends for inactivity."; a switch post naming who
  switched; a plain-message refusal by direct message or in-channel; hook no-answer phrase.
- Vision says a session stays open until a person ends it, but restarts interrupt (R-17): reword? The
  two `Waits:` blocks the session wrote are the owner's to confirm.
- Commit name and email (`privacy.identities`)? The `Question:` ideas of 2026-10-21.

## Open findings

- `docs/findings/2026-10-17-net-diff-privacy-scan-let-named-commits-go-public.md` (rewrite done; resolve)
- `docs/findings/2026-10-15-spike-login-claims-for-adr-0020.md`, `...-stream-and-process-...`, `...-hook-flow-...`
  (PC rows hold, droplet open); `2026-10-15-spike-claims-for-adr-0022.md` (droplet, C37, key half);
  `2026-10-16-prior-art-chat-front-ends-for-coding-agents.md` (relay decided; L1-L4 mine)
