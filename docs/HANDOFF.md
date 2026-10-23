# Handoff

Coordinator state. Subagents follow their brief, not this file.
Updated: 2026-10-23, P16 pushed, by main session (rooted in the repo, hooks loaded)

## Now

- ADR 0025 (`docs/design/decisions/0025-session-workflow-composition.md` and its companion) settles
  how a session composes the bot's and the repository's workflows; the owner answered its four calls
  on 2026-10-23. On `main` with the `Agent` fix, the P15-P19 plan, P15 (the session never pushes)
  and P16 (the push scan), `a504ed9`. P17 (the copy, the `workflow` key, the probe) is with the coder.
- The live `privacy.allow` holds the two service accounts (owner's yes, 2026-10-23); the running bot
  is the pre-P16 binary, so rebuild `bin/` before the next `serve` or the strict loader refuses the key.
- Until P17-P19 land a session sees none of the repository's rules: a reviewer read and the
  pusher's checks before anything from it reaches `main`. P12 left: the relay, the post-restart notice.

## Next

1. P17 commit, review, push; then P18, P19 (`08-implementation-plan-session-workflow.md`); P19's
   real turns are the owner's.
2. Triage: the `main` handoff and `/session land` ideas route through ADR 0025 rule 7.
3. Owner: continue the live session (milestone 1: edit Broodforge from `broodforge/discord`); set the
   live `instance.json` timeouts to 0; a real root outside Temp.
4. Then the last-good cold start (architect; the fake-engine header is the owner's), the net-diff
   finding, the P11 remainder `Plan:`, `audit --follow`, `/usage`; carry-backs; the surfaces ADR.

## Waiting on user

- The `pre-push` line a session sees: "Broodforge: session <id> does not push. Commit in this tree;
  Broodforge pushes at the handoff when the session ends." (then git's own error). Confirm or reword.
- Wording: P12a and `/model` showing effort; "Never ends for inactivity."; a switch post naming who
  switched; a plain-message refusal; hook no-answer phrase; R-17; the session's two `Waits:` blocks.
- Commit name and email (`privacy.identities`)? The `Question:` ideas of 2026-10-21.

## Open findings

- `docs/findings/2026-10-23-system-prompt-append-renders-once-per-turn-trailer-list-stale.md`,
  `...-first-live-session-commits-reached-main-unreviewed.md` (both resolve in ADR 0025's passes)
- `docs/findings/2026-10-17-net-diff-privacy-scan-let-named-commits-go-public.md` (rewrite done; resolve);
  `2026-10-15-spike-*.md` (PC rows hold; droplet open); `2026-10-16-prior-art-chat-front-ends-for-coding-agents.md` (L1-L4 mine)
