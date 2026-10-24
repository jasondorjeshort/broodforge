# Handoff

Coordinator state. Subagents follow their brief, not this file.
Updated: 2026-10-23, C2/C6 verified, C5 partial, by main session (rooted in the repo, hooks loaded)

## Now

- ADR 0025: C2 and C6 held on the owner's real turn (chain, the live bot, session s0008). C5 only
  shows prompt-level compliance with `tools:`, not yet separated from enforcement (checks table).
  The operator entry that called for this is done and removed.
- s0008's commit (a subagent-ending-early finding, an `auto` mode idea) pushed to
  `broodforge/scratch-4a695f0a4486/s0008` on origin, not `main`: the handoff design held under a
  real push. Merging it onto `main` is a separate, undecided step.
- A reviewer read and the pusher's checks gate everything before `main`; P12 left: the relay, the
  post-restart notice.

## Next

1. Decide whether and how to land s0008's branch onto `main` (review it, or open a PR); its own
   open items: `SendMessage`/`Skill` in code reach, the handoff reaching a session, proposal
   0026 (`auto` mode, the next wave's first brief). Triage: the two 2026-10-23 ideas.
2. Triage: the `main` handoff and `/session land` ideas route through ADR 0025 rule 7.
3. Owner: continue the live session (milestone 1: edit Broodforge from `broodforge/discord`); set the
   live `instance.json` timeouts to 0; a real root outside Temp.
4. Then the last-good cold start (architect; the fake-engine header is the owner's), the net-diff
   finding, the P11 remainder `Plan:`, `audit --follow`, `/usage`; carry-backs; the surfaces ADR.

## Waiting on user

- The `pre-push` line (P15, one constant) and its no-id form "this tree does not push": confirm or reword.
- Wording: P12a and `/model` showing effort; "Never ends for inactivity."; a switch post naming who
  switched; a plain-message refusal; hook no-answer phrase; R-17; the session's `Waits:` blocks.
- Commit name and email (`privacy.identities`)? The `Question:` ideas of 2026-10-21.

## Open findings

- `docs/findings/2026-10-23-system-prompt-append-renders-once-per-turn-trailer-list-stale.md`,
  `...-first-live-session-commits-reached-main-unreviewed.md` (s0008's branch push is one data point; items 6-7 still the owner's)
- `docs/findings/2026-10-23-subagents-in-a-session-end-before-writing.md` [no-check: s0008's branch, unmerged]
- `docs/findings/2026-10-17-net-diff-privacy-scan-let-named-commits-go-public.md` (resolve); `2026-10-15-spike-*.md` (droplet open); `2026-10-16-prior-art-chat-front-ends-for-coding-agents.md` (L1-L4 mine)
