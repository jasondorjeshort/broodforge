# Handoff

Coordinator state. Subagents follow their brief, not this file.
Updated: 2026-10-23, findings cleanup pass, by main session (rooted in the repo, hooks loaded)

## Now

- s0008's finding is on `main` (3aaf308, cherry-picked; its branch is still on origin). Its audit
  read and hook log: for a subagent's in-tree call in prompt mode the hook ran and returned defer,
  the engine surfaced no `deferred_tool_use`, so no prompt was posted and the subagent ended; the
  mode switch explains the later allow. A session that delegates runs in policy mode for now.
- ADR 0025 is closed out: two findings promoted to it, the idea it decided deleted. C5 still shows
  prompt-level compliance only. P12 left: the relay, the post-restart notice.

## Next

1. Milestone 1 blocker, coder probe: `engine run --real`, prompt mode, a one-call subagent whose
   call the hook defers; capture the raw stream: what the engine emits for the subagent's deferred
   call (a nested `deferred_tool_use`, a control request, nothing) and what ends the subagent. Then
   the architect: prompt mode and subagents (bears on `auto`, proposal 0026, the next wave's first brief).
2. s0008's other open items: `SendMessage` and `Skill` in `code` reach, the handoff reaching a
   session. Triage: the four 2026-10-23 ideas without a block.
3. Owner: continue the live session (milestone 1: edit Broodforge from `broodforge/discord`), in
   policy mode until item 1 lands; set the live `instance.json` timeouts to 0; a real root outside
   Temp; delete the s0008 branch on origin when it is no longer wanted.
4. Then the last-good cold start (architect; the fake-engine header is the owner's), the net-diff
   finding, the P11 remainder `Plan:`, `audit --follow`, `/usage`; carry-backs; the surfaces ADR.

## Waiting on user

- The `pre-push` line (P15, one constant) and its no-id form "this tree does not push": confirm or reword.
- Wording: P12a and `/model` showing effort; "Never ends for inactivity."; a switch post naming who
  switched; a plain-message refusal; hook no-answer phrase; R-17; the session's `Waits:` blocks.
- Commit name and email (`privacy.identities`)? The `Question:` ideas of 2026-10-21.

## Open findings

- `docs/findings/2026-10-23-subagents-in-a-session-end-before-writing.md` (Next 1 above),
  `...-first-live-session-commits-reached-main-unreviewed.md` (items 6-7 the owner's)
- `docs/findings/2026-10-17-net-diff-privacy-scan-let-named-commits-go-public.md` (resolve); `2026-10-15-spike-*.md` (droplet open); `2026-10-16-prior-art-chat-front-ends-for-coding-agents.md` (L1-L4 mine)
