# Handoff

Coordinator state. Subagents follow their brief, not this file.
Updated: 2026-10-23, P19 pushed, by main session (rooted in the repo, hooks loaded)

## Now

- ADR 0025 (`docs/design/decisions/0025-session-workflow-composition.md` and its companion) settles
  how a session composes the bot's and the repository's workflows; the owner answered its four calls
  and the not-started flow (ephemeral) on 2026-10-23. P15-P19 are built, reviewed and on `main`
  (P19 the last, injection, the header, `session show`). Real turns owed, the owner's: P18 C2, P19
  C5 and C6 (`docs/operator/2026-10-23-real-turns-c2-c5-c6.md`), after a `bin/` rebuild.
- The live `privacy.allow` holds the two service accounts (owner's yes, 2026-10-23); the running bot
  is the pre-P16 binary, so rebuild `bin/` before the next `serve` or the strict loader refuses the key.
- Until P17-P19 land a session sees none of the repository's rules: a reviewer read and the
  pusher's checks before anything from it reaches `main`. P12 left: the relay, the post-restart notice.

## Next

1. Owner: rebuild `bin/`, serve, run the real turns (the operator entry); milestone 1 under the
   project's rules. Then the first-live-session finding resolves. Triage: the two 2026-10-23 ideas.
2. Triage: the `main` handoff and `/session land` ideas route through ADR 0025 rule 7.
3. Owner: continue the live session (milestone 1: edit Broodforge from `broodforge/discord`); set the
   live `instance.json` timeouts to 0; a real root outside Temp.
4. Then the last-good cold start (architect; the fake-engine header is the owner's), the net-diff
   finding, the P11 remainder `Plan:`, `audit --follow`, `/usage`; carry-backs; the surfaces ADR.

## Waiting on user

- The `pre-push` line (P15, one constant) and its no-id form "this tree does not push": confirm or
  reword. P18's C2 real turn on the owner's account: the coder, or the owner by the command it reports.
- Wording: P12a and `/model` showing effort; "Never ends for inactivity."; a switch post naming who
  switched; a plain-message refusal; hook no-answer phrase; R-17; the session's `Waits:` blocks.
- Commit name and email (`privacy.identities`)? The `Question:` ideas of 2026-10-21.

## Open findings

- `docs/findings/2026-10-23-system-prompt-append-renders-once-per-turn-trailer-list-stale.md`, `...-first-live-session-commits-reached-main-unreviewed.md` (both resolve in ADR 0025's passes);
  `...-subagents-in-a-session-end-before-writing.md` (s0008: a deferred call ends a subagent; triage)
- `docs/findings/2026-10-17-net-diff-privacy-scan-let-named-commits-go-public.md` (resolve); `2026-10-15-spike-*.md` (droplet open); `2026-10-16-prior-art-chat-front-ends-for-coding-agents.md` (L1-L4 mine)
