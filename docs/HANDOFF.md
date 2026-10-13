# Handoff

Coordinator state. Subagents follow their brief, not this file.
Updated: 2026-10-13, after the three-repo merge session, by main session (Hatchery coordinator)

## Now

- The base caught up with its two downstream projects in one unpushed run on `main`: Dystopo's
  `git-guard` plus the bypasses an independent review reproduced, hook policy moved to
  `.claude/hooks/policy.json` (ADR 0006), Dystopia's queues and triage blocks in `docs/ideas/`
  and `docs/operator/`, a `reviewer` agent and the one-review rule (ADR 0007). `backlog.md` is
  gone; its items are the first idea files.
- Dystopia adopted the hooks the same day (its `docs/reference/hatchery-adoption.txt` records
  what it takes and keeps); Dystopo pinned its `Hatchery base:` line but has not pulled yet.

## Next

1. Push `main` (user's call); then Dystopo runs `update-from-hatchery` in patch mode from
   `0ce6670` and resolves `policy.json` in its own favour.
2. Triage `docs/ideas/` (`triage` skill): every file there is untriaged except the `Waits:` ones;
   two carry a `Question:` for the owner.
3. Critic review of the hook rewrite and ADR 0006 as a whole; the reviewer's remaining advisories
   are the two stale paths in `docs/reference/claude-code-on-windows.md`.
4. Proposal 0003 (wave workflow) still decides the `wave` skill and a `verify` agent beside the
   reviewer; its critic brief is `.claude/skills/proposal/brief-critic.md`.
5. ReGenesis adoption, as before, once `main` is pushed.

## Waiting on user

- The push of all three repositories at the end of the session.
- `Answer:` lines on the two `Question:` ideas (main-session pushes; verify agent beside reviewer).

## Open findings

- `docs/findings/2026-10-11-bash-heredoc-apostrophes.md`
