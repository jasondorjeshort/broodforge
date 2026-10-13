# Handoff

Coordinator state. Subagents follow their brief, not this file.
Updated: 2026-10-13, usage-gated session after the three-repo merge, by main session (Hatchery coordinator)

## Now

- `CLAUDE.md` trimmed to rules with pointers (mechanism text now lives in ADR 0006 and the hook
  specs), and living docs no longer name the downstream projects: new content rule in
  `.claude/rules/docs.md`, policy examples renamed by the model they show, proposal 0005 renamed.
  Closed records keep their provenance. Done in the main session because the weekly meter (93%)
  was above the deny line, on the owner's word.
- Owner answered the push question: the main session pushes on an explicit ask, as `CLAUDE.md`
  and `policy.json` already say; the idea is deleted. The owner's note that a Haiku pusher may be
  cheaper is on `docs/ideas/2026-10-13-haiku-runner-escalates-to-sonnet.md`.
- Owner's direction: the base and its first build-phase sibling (`../dystopo`) are modeled on each
  other and are to be kept in sync both ways, by `update-from-hatchery` there and by lifting its
  improvements here. Weekly meter resets 2026-10-16 00:00 EDT.

## Next

1. Sync with `../dystopo`: it pulls the base in patch mode (its `Hatchery base:` line says from
   where) and resolves `policy.json` in its own favour; anything it has that the base lacks comes
   back as an idea or a proposal here.
2. Triage `docs/ideas/` (`triage` skill): fourteen files have no block; one `Question:` is open
   (verify agent beside the reviewer, decided by proposal 0003).
3. Critic review of the hook rewrite and ADR 0006 as a whole, and of this session's unreviewed
   main-session commits (the trim and the scrub), per the one-review rule.
4. Proposal 0003 (wave workflow) critic pass (`.claude/skills/proposal/brief-critic.md`); decides
   the `wave` skill and the verify-agent question.
5. Adoption of the owner's other existing project, after the above.

## Waiting on user

- The `Answer:` on the verify-agent idea, or leave it to proposal 0003.
- The push, when wanted.

## Open findings

- `docs/findings/2026-10-11-bash-heredoc-apostrophes.md`
