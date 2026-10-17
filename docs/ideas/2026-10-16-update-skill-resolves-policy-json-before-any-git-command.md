---
date: 2026-10-16
kind: chore
related: [.claude/skills/update-from-hatchery/SKILL.md, .claude/hooks/git-guard.js]
---

# The update skill resolves policy.json by hand before any other git command

Seen on 2026-10-16, merging `hatchery/main` (ea20b96..48a43e4) into this project. The base had
changed the `usage` block of `.claude/hooks/policy.json`, the project had its own thresholds there,
and the merge left conflict markers in the file. `git-guard` reads the policy before every git
command and denies all of them while the file is malformed (ADR 0006, fail modes), so the runner's
follow-up listings (`git status --porcelain`, `git diff --name-only --diff-filter=U`) were refused,
and so would `git checkout --ours` and `git merge --abort` have been. The only way out was to
rewrite the file without git (a script or an editor), then `git add` it.

The fail mode is right: a guard that cannot read its policy must fail closed. What is missing is a
sentence in the skill. Step 5 says the project wins in `policy.json` but lists the file among the
others, after `git diff --name-only --diff-filter=U`, which is itself denied at that point. The
runner's brief in step 4 cannot ask for a listing after the merge either.

Plan: in the base, `.claude/skills/update-from-hatchery/SKILL.md`: step 4 says the merge or apply
command is the last git command that runs until `policy.json` carries no markers; step 5 starts
with "`.claude/hooks/policy.json` first, by hand (not `git checkout --ours`): take the project
version, then `node .claude/hooks/policy.js` must print the policy; only then `git add` it and
list the rest with `git diff --name-only --diff-filter=U`". A `tester` makes the edit from this
file; then this file is deleted in the commit that merges that base change here.
