---
date: 2026-10-16
kind: bug
related: [2026-10-14-git-guard-could-scope-commits-by-repository-location.md, 2026-10-21-carry-back-quoted-git-does-not-name-git.md]
---

# The git guard refuses script lines where a bare git word sits beside a variable expansion

Two false positives and one by-design refusal hit a reviewer and a coder on 2026-10-16. Anchor:
`namesGit` and `opaqueCommand` in `.claude/hooks/git-guard.js`.

- Two script lines held the word `git` or `.git/` unquoted, on a line whose command word was a `$B`
  expansion. Nothing ran git. `namesGit` counts any unquoted word with a git token (`LOOSE_GIT`),
  which makes the line "name git", and `opaqueCommand` then denies every command whose command word
  is an expansion, since what `$B` runs cannot be seen.
- A `git ... commit` in a throwaway repository under the scratchpad was refused. That is the
  commit rule working as designed (commits are the runner's), but reviewers who want a scratch
  repository now go through Go `os/exec` to avoid it. This half is the 2026-10-14 idea; the new
  evidence is that agents invent the detour.

## Want

A scratch-repository allowance for paths under the session scratchpad (the 2026-10-14 key, with the
repository resolved from `-C` or the cwd). And a bare word beside an expansion read as data unless
it is in command position: the word `git` or `.git/` as an argument of a `$B` command, or inside a
path, does not make the line name git; `opaqueCommand` fires only when git is the command word or
the expansion's value could be it.

## Open

- Whether `$B git ...` (the expansion as the runner, git as its first argument) stays denied; the
  tokenizer cannot tell a wrapper from a script path.
- 2026-10-21: the update to Hatchery `dc896c3` took the base's guard, which has no `namesGit`;
  "the line names git" is a substring test over the raw text again (the related carry-back idea).

Question: send the expansion-adjacent narrowing to Hatchery as a second carry-back on top of the
2026-10-21 one, or fold this evidence as a dated line into Hatchery's open rule-19 question
(`../hatchery/docs/ideas/2026-10-17-guard-false-positives-in-audits.md`, `Answer:` blank) and leave
the guard as it is? Recommendation: fold and leave. Hatchery built and withdrew a data-position
exemption because `for ...; do echo '<git cmd>'; done | sh` and `tee >(sh)` shapes bypassed it, and
`$B git ...` (the Open item) has the same hole: the tokenizer cannot tell a wrapper from a script
path, so the one safe narrowing is the quoted-text rule the 2026-10-21 Plan already carries, and the
workaround costs one extra call. The scratch-repository half rides on the 2026-10-14 idea's Plan.
Answer:
