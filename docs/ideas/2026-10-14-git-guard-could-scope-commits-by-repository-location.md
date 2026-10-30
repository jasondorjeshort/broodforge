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

Plan: base work, no diff from here: a new `git.commit` key is a policy-schema change (ADR 0006), which
the `carry-back` skill routes to Hatchery's coordinator as a Hatchery idea, not a branch. Hatchery's
coordinator (a session in `../hatchery`) files it in Hatchery's `docs/ideas/` from the text above, `kind:
chore`, linked `related:` both ways to its `2026-10-17-guard-false-positives-in-audits.md` (its Question
keeps `update-ref -d` in a scratch repository denied) and `2026-10-21-git-guard-uses-the-policy-of-the-repo-it-runs-in.md`
(its Open asks what governs a repository with no policy file), naming the shape: a `git.commit.scratchRoots`
list, the repository resolved from `-C` or the cwd as `invocations()` already resolves `inv.dir`, rule 4's
sentence in `docs/hatchery/decisions/0002-hook-specs.md`, harness cases. Verify: the file is on `hatchery/main`
(`git fetch hatchery; git ls-tree --name-only hatchery/main docs/ideas/`); then the runner deletes this file.
Out of scope: this project's `git-guard.js` and `policy.json`; the change comes back by `update-from-hatchery`.
