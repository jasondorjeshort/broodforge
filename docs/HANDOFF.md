# Handoff

Coordinator state. Subagents follow their brief, not this file.
Updated: 2026-10-13, usage-gated session after the three-repo merge, by main session (Hatchery coordinator)

## Now

- `main` is pushed and level with `origin/main` (the merge session's run: `git-guard` with policy
  in `.claude/hooks/policy.json`, ADR 0006; queues and triage in `docs/ideas/` and `docs/operator/`;
  `reviewer` agent and ADR 0007; `backlog.md` retired into ideas).
- Owner answered the push question: the main session pushes on an explicit ask, as `CLAUDE.md`
  and `policy.json` already say; the idea is deleted. The owner's note that a Haiku pusher may be
  cheaper is on `docs/ideas/2026-10-13-haiku-runner-escalates-to-sonnet.md`.
- Weekly meter was at 93% (deny line 80, no override above it), so no agent ran; it resets
  2026-10-16 00:00 EDT.

## Next

1. Dystopo runs `update-from-hatchery` in patch mode from `0ce6670` and resolves `policy.json` in
   its own favour (in the Dystopo repo).
2. Triage `docs/ideas/` (`triage` skill): fourteen files have no block; one `Question:` is open
   (verify agent beside the reviewer, decided by proposal 0003).
3. Trim `CLAUDE.md` (149 lines, cap 150): `architect`, one commit; move hook mechanics (git-guard
   rule list, usage-cache details) to `docs/reference/` with pointers, point Major decisions at the
   `proposal` skill for the steps, collapse the Delegation example list to one sentence. Owner has
   seen the assessment and not yet said go.
4. Critic review of the hook rewrite and ADR 0006 as a whole; the reviewer's remaining advisories
   are the two stale paths in `docs/reference/claude-code-on-windows.md`.
5. Proposal 0003 (wave workflow) critic pass (`.claude/skills/proposal/brief-critic.md`); decides
   the `wave` skill and the verify-agent question.
6. ReGenesis adoption.

## Waiting on user

- Go-ahead on the `CLAUDE.md` trim (item 3), and which sections stay verbatim, if any.
- The `Answer:` on the verify-agent idea, or leave it to proposal 0003.

## Open findings

- `docs/findings/2026-10-11-bash-heredoc-apostrophes.md`
