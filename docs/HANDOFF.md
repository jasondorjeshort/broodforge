# Handoff

Coordinator state. Subagents follow their brief, not this file.
Updated: 2026-10-12, at commit dc35250, by main session (Hatchery coordinator)

## Now

- Adoption path landed, not yet critic-reviewed: ADR 0004, the `adopt-hatchery` skill, patch mode
  in `update-from-hatchery`, and three self-contained README prompts with preconditions. Trigger:
  the old update prompt was pasted into ReGenesis (own history, never decanted) and stopped.
- The adoption prompt reads the skill from `hatchery/main` on GitHub, so Hatchery must be pushed
  before it can run anywhere.

## Next

1. User runs the "Adopt the Hatchery practice" prompt in the ReGenesis laptop clone. The run will
   ask about CLAUDE.md contradictions (push after every commit and the Stop hook vs pusher-only;
   amend to fold review fixes; `/code-review` before push), the license, and five docs over 300
   lines. Friction found there comes back here as protocol fixes.
2. Launch two critics in parallel: whole-base review including ADR 0004 and `adopt-hatchery`
   (write to `docs/findings/2026-10-12-critic-review-of-base.md`) and proposal 0003 review (brief
   from `.claude/skills/proposal/brief-critic.md`, write to `docs/design/proposals/0003-critic-review.md`).
3. Fix after the critics, in one batch: backlog item claiming CLAUDE.md lets the main session push
   (it does not; decide the rule on purpose); reference file's usage-meters section states Dystopo
   policy as fact (move to proposal 0003); backlog Ideas entries from Dystopo exceed one line each;
   ADR numbering collision between Hatchery and projects (backlog Ideas).
4. Architect revises 0003 to ADR 0003; this decides `wave` skill, `verify` agent, phase keying of
   git-guard rules 3 and 4, and any usage-gate override for in-flight work.
5. Decide whether Dystopo is also adopted or stays as it is; ReGenesis is the first adopted project.

## Waiting on user

- A push of Hatchery `main` (pusher needs the request in the user's words), so ReGenesis can fetch
  the adopt skill.

## Open findings

- `docs/findings/2026-10-11-bash-heredoc-apostrophes.md`
