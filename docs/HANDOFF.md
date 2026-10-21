# Handoff

Coordinator state. Subagents follow their brief, not this file.
Updated: 2026-10-21 by main session (written from the Dystopo coordinator session, in `../dystopo`)

## Now

- Pushed through `4da83b7`: ADR 0020 (attribution trailers and GitHub's web committer; PR 2, `51d29f9`)
  and ADR 0021 "guardrails bound damage, not who may act" (proposal, critic review, decision; PR 3,
  `e62c7f0`). Both came out of Dystopo's 2026-10-21 identity rewrite; Dystopo already runs the 0020
  scan and has adopted the base at `f7f41af` (its ADR 0038). None of 0021's tasks T-a to T-g is started.
- Two ideas filed today by a regenesis session (`2026-10-21-git-guard-uses-the-policy-of-the-repo-it-runs-in.md`):
  the guard applies the session project's policy to a `git -C <sibling>` command (this is why every
  Hatchery commit and push from the Dystopo session was refused and the owner typed them); the push
  rule ignores the remote (a downstream clone could fast-forward this public `main` to a private
  history); the pusher may not `pull --rebase --autostash`. Untriaged; T-b is the natural home.
- Leftovers: worktrees `../hatchery-wt/T120` and `T121` (merged; their `task/*` branches local),
  `../hatchery-wt/usage-gate` (prunable), `.claude/worktrees/agent-*` (untracked, some session's
  isolated agent); a cleanup chore from a Hatchery-rooted session.

## Next

1. The ADR 0021 wave, from a Hatchery-rooted session, in the ADR's order T-a to T-g (the proposal's
   Recommendation lists each task's files); fold today's three guard gaps and
   `2026-10-19-guard-gaps-where-git-runs-shell-text.md` into T-b. Agents may commit and push the
   wave's branches under the new rule; `main` stays the coordinator's until T-a's rulesets hold.
2. Triage `docs/ideas/` (`triage` skill): the two untriaged guard ideas; the ADR 0012 scan.
3. T-g's migration row for Dystopo (`usage.mode: "block"`, `carry-back/*` in the push branches, the
   rulesets step, the deny lines) runs in Dystopo's session as its second base update, after T-a to T-f.
4. Each sibling in its own session, regenesis and droplet-management too: ADR 0018 back
   (`0018-privacy-scan.md` 9); its 0016 migration; then 0020 and 0021.
5. Architect answers the critic reviews of proposals 0003 and 0005; critic review of ADR 0006 as a whole.

## Waiting on user

- Decided 2026-10-21: every pushed commit carries the owner's real identity, listed in each
  repository's `policy.json`; trailers allowed (ADR 0020). ADR 0013 stays `proposed`, not re-asked.
- `2026-10-16-scheduled-architect-design-pass.md`: yes or no; the HOLD deny's pool sentences
  (ADR 0011) unseen; delete `~/.claude/usage-state.json` once all projects run the new scripts.

## Open findings

- `docs/findings/2026-10-11-bash-heredoc-apostrophes.md` (list: `queue-list.js findings --status open`)
