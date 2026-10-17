---
date: 2026-10-15
kind: question
---

# Review fixes before a push: amend, fixup and autosquash, or a follow-up commit

Anchor: `CLAUDE.md`, sections Git ("Amend only to fold a review fix ... into the unpushed commit it
belongs to") and One independent review per change ("a fix to an unpushed commit is amended into
it"); `.claude/skills/commit/SKILL.md` step 5; `.claude/hooks/git-guard.js`, rebase handling.

The owner asked on 2026-10-15 whether minor review fixes made before a push should be amended into
the commit or committed as a follow-up, and what best practice is, and wants it researched. The
rule today amends. That is easy when the reviewed commit is HEAD. It is not when a commit sits on top.
On 2026-10-15 the carry-back af601b5 went to review with the status line commit landing above it,
so a fix there would need `git commit --fixup` and a non-interactive `git rebase --autosquash`
(non-interactive only from Git 2.44), or a follow-up commit. No change to the rule until the
research is in.

## Open

- What projects that push straight to a branch do (git.git and kernel patch rerolls, Gerrit's
  amend per change), against PR flows that squash at merge, and which of those fits a
  single-branch repo pushed by agents.
- What a follow-up commit costs here besides log noise: a commit in history with a known defect
  (bisect), the link between a reviewer's report and the commit it read, the fixer's
  Co-Authored-By line.
- Whether `--fixup` plus `--autosquash` works under git-guard and the installed git version, and
  whether the `commit` skill should describe it for the case where the commit is not HEAD.
- Output: a finding in `docs/findings/`, then a rule edit if one follows.
