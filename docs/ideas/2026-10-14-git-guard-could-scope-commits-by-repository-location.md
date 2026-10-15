---
date: 2026-10-14
kind: chore
---

# The git guard could scope the commit rule by repository location

`.claude/hooks/git-guard.js`, reading `git.commit.subagents` in `.claude/hooks/policy.json`,
denies `git commit` from every agent but the runner in any repository, a throwaway one under the
scratchpad or the temp directory included, and a commit handed to a shell inside the command line
counts. The spike plan (`docs/design/07-spike-plan.md`, Fixture commits) needs scratch commits (a
one-file repository, a clone to bundle) and routes them through a Node setup script so the Bash
tool never sees the command. Reviews and hook tests need scratch commits in the same way, and each
will invent its own detour.

A policy key that exempts repositories under the scratchpad or the temp directory from the commit
rule, the guard resolving the repository from `-C` or the cwd, would remove the friction without
weakening what the rule protects, which is commits to the project. Base (Hatchery) work: the hook
and the policy schema (ADR 0006) are the base's; a project takes the change by `update-from-hatchery`.
