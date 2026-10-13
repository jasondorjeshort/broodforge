---
date: 2026-10-13
kind: chore
related: [2026-10-13-verify-agent-reads-reports-not-diffs.md, 2026-10-13-usage-gate-phase-override.md]
---

# First sync with the build-phase sibling, both ways

The owner's direction (2026-10-13): the base and the build-phase project checked out beside it
(`../dystopo`) are modeled on each other and are kept in sync both ways. That project aligned by
hand at base commit `0ce6670` (its ADR 0020) before `adopt-hatchery` existed, so the
`update-from-hatchery` skill's preconditions fail there: no copy of the skill, no `Hatchery base:`
line. Found the same day: no `policy.js`, `policy.json` or `shell-parse.js`; no `agents/` (declined
in its ADR 0020: roles are implementer, verifier, chore, model set per launch); hooks forked with
HOLD hardcoded at 75 and the usage read from a different file under `~/.claude/`; its harness
passes 113 cases. Its `adr`, `brief`, `finding` and `proposal` skills were edited on 2026-10-12,
after the base's copies; `wave`, `release` and `sweep` exist only there.

The patch from `0ce6670` to the base's head on `.claude` and `CLAUDE.md` is 22 files.

Plan: a session in that repository, with agents (after the weekly reset or with the deny line
raised there). (1) Copy `update-from-hatchery` and add `Hatchery base: 0ce6670` to its ADR 0020 so
the skill's step 3 finds patch mode. (2) Run the skill. The four hooks conflict: take the base
scripts and write the project's exceptions into `policy.json`, starting from
`.claude/hooks/policy.examples/task-branches.json`, which encodes them; `CLAUDE.md` has none of the
base's section names, so merge the rule sections by hand under its own headings. (3) Its harness
must pass the base's cases plus its own. (4) Reverse direction: diff its four edited skills against
the base's and bring back what is not project-specific as an edit here; the three skills it alone
has wait on proposal 0003. Report both directions in its handoff and this file's deletion commit.
