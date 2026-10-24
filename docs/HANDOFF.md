# Handoff

Coordinator state. Subagents follow their brief, not this file.
Updated: 2026-10-24, s000a post-mortem, by main session (rooted in the repo, hooks loaded)

## Now

- s000a's P12b commits are on `main` (through c8154a3), cherry-picked from its tree: its End click was
  refused (Discord's id check took digits only; s000a is the first base-36 id with a letter), so its handoff
  never ran. The fix is on main; the live bot needs a stop, `bin/` rebuild and restart to get it. s000a is
  still open to the bot (owner's call to end it; its handoff then pushes a duplicate session branch).
- Owner approved 2026-10-24: `xhigh`, the driver-only text, "This session has ended." for every
  closed-session refusal (settles the plain-message refusal), the header menus (judge live).

## Next

1. The s000a post-mortem's Next (R-35 for open sessions, the credential carve-out, foreground
   subagents, the pusher's `gcc` PATH, 86 hook-test failures); rewrite the closing-message idea.
2. Milestone 1 blocker, coder probe: `engine run --real`, prompt mode, a one-call subagent whose
   call the hook defers; capture the raw stream. Then the architect: prompt mode and subagents.
3. s0008's other open items: `SendMessage` and `Skill` in `code` reach, the handoff reaching a
   session. Triage: the 2026-10-23 and 2026-10-24 ideas without a block.
4. Owner: in policy mode until item 2 lands; set the live `instance.json` timeouts to 0; a real root
   outside Temp (s000a's commits sat only there); delete the s0008 branch on origin when unwanted.
5. Then the last-good cold start, the net-diff finding, the P11 remainder `Plan:`, `audit --follow`,
   `/usage`; carry-backs; the surfaces ADR; the session-land idea (owner's ADR; agent lands added).

## Waiting on user

- The `pre-push` line (P15) and its no-id form "this tree does not push" (s000a lost 19 min to it).
- Wording: P12a and `/model` showing effort; "Never ends for inactivity."; a switch post naming who
  switched; hook no-answer phrase; R-17; the session's `Waits:` blocks.
- Commit name and email (`privacy.identities`)? The `Question:` ideas of 2026-10-21.

## Open findings

- `docs/findings/2026-10-24-s000a-post-mortem-*.md` (Next 1), `2026-10-24-background-subagents-*.md`, `2026-10-24-session-friction-*.md`; `2026-10-23-subagents-in-a-session-end-before-writing.md` (Next 2), `...-first-live-*.md` (6-7 owner's)
- `docs/findings/2026-10-17-net-diff-privacy-scan-let-named-commits-go-public.md` (resolve); `2026-10-15-spike-*.md` (droplet open); `2026-10-16-prior-art-chat-front-ends-for-coding-agents.md` (L1-L4 mine)
