---
date: 2026-10-17
status: resolved (ADR 0018)
promoted_to: docs/design/decisions/0018-personal-data-never-leaves-the-machine.md
area: git / privacy, protocol surfaces
---

# The broodforge leak used steps Hatchery still has; its history carried the owner's second account

## Question

The owner: "In ../broodforge you can perhaps see we had a leak of personal data. I don't know if
the protocols that lead to that came from here or if we can tighten them up some." Broodforge's
agents, skills and hooks started as copies of these. Asked: what leaked, by class (never the value),
and through which step; what broodforge changed and whether it holds; whether Hatchery's protocols
have the same gap; whether its tree and history carry the same classes.

## What was tried

- Broodforge, read-only (`git -C`, file reads; `git status --short` empty before and after): the
  seven commits of the brief (message, stat, protocol diffs), pusher check 5a, git-guard rule 20,
  ADR 0002 row 20, the ADR 0006 row, its finding, operator entry and carry-back idea, its handoff.
- Rule 20's `PRIVACY` expression, loaded into a scratch script, against placeholder strings (user
  `alice`): three home-path spellings, the project slug, a bare name, two host forms.
- Hatchery: the files in Result (3); a scratch Node scanner over all refs (121 commits): added lines
  of `git log --all -p -U0 --cc`, messages, added file names, tracked files at HEAD. Classes: the
  machine user name in any spelling (slug and bare name included), any non-placeholder home segment,
  emails other than the owner's, noreply and RFC 2606, the second account's name, host markers,
  grant content, transcript markers, session links, credential prefixes. Output: counts and names
  of commits or files only. On broodforge it finds the three name-adding commits.
- Identity: author and committer emails over all refs against the owner's `user.email`; the root
  commit's ancestry per sibling; `gh repo view --json visibility` per remote.

## Result

### (1) What leaked in broodforge, by class, and the step each passed

- Machine user name in a home path. Added by 14304c6 (2026-10-16, a git-guard test case built from a
  real failing command line; whether a brief carried it cannot be told), 3af9178 (2026-10-17, an
  idea quoting the path a tool refused) and 4904287 (the coordinator wrote the owner's recording
  root into the handoff); removed by commits on top, 5288955 and 9cbfa46. No check at write or
  commit time; at push, check 5a read the net diff `git diff @{u}..HEAD`, which a fix on top makes
  clean, so all five commits reached origin.
- The owner's second account (name and email) as author and committer: Hatchery's 28 commits of
  2026-10-11 and 10-12, root included, made under the machine's global git identity until
  2026-10-15. Steps: no protocol sets or checks the identity; the pusher reads a message for a
  subject and a trailer only; decant clones the full history; broodforge's 2026-10-15 rewrite
  (re-author, roles for names, before its first push) stopped at the decant line.
- Real people's names in content: rewritten to roles in unpushed history on 2026-10-15; none found
  on its origin since. No backup or `refs/original` ref remains, so the names removed are unknown.
- Recorded conversations: the owner's recorded engine turns as fixtures (04e21b3, 4904287), paths
  rewritten by `importfixture` (which refuses a surviving user name or slug), read decoded by two
  independent agents (rule since f9cdb1f), account-level lines kept on the owner's decision; spike
  session ids remain. Accepted, not a leak.
- Session links: a `Claude-Session:` trailer with a claude.ai session URL in 39 messages, 25 of them
  Hatchery's (4f37331 and 4f4fb74 among broodforge's own). Neither grep matches it, and check 5a's
  read, which lists account identifiers, passed it.
- Credentials: none real; the `sk-ant-` and `ghp_` strings in its masking tests are synthetic.
### (2) What broodforge added, and whether it holds

- Check 5a (4f37331, revised in e09224b): a grep for emails, home paths with a user name and host
  markers; a read for people, numbers, addresses and ids; now per commit over `<branch> --not
  --remotes=origin`, messages and author and committer emails included. Holds for what it names.
  Gaps: it exempts `git config user.email`, the very identity that made the 28 commits on this
  machine; check 5 beside it still scans secrets over the net diff (its pusher line 17); the read is
  "of the diff" with no range, so a name added and then removed can still pass the read.
- Rule 20 (e09224b, not yet pushed): the grep inside git-guard, on every push from every caller,
  main session included, per commit in `rev-list <refs> --not --remotes=<r>`, tags too; a range it
  cannot compute denies. Sound core. Gaps: the same `user.email` exemption (`privacyScan`); the
  expression misses the project slug spelling (`C--Users-<name>-...`, as in a scratchpad sub-path or
  under `~/.claude/projects/`), a bare user name and a host name without the desktop marker (probe:
  miss on those three, hit on the three slash spellings and the desktop marker); it sees only pushes
  Claude's shell tools run, not the owner's terminal, a script that pushes, or `gh` (pull request
  titles and bodies, `gh repo create --push`).
- Scrub procedures: the 2026-10-15 rewrite was right in timing, short in reach (own commits only).
  The 2026-10-17 operator entry (filter-repo, mailmap, text replacement, backups, owner's force
  push, re-clone, GitHub purge) is the right tool, but its premise that both repositories are
  private is wrong for Hatchery (see 4), and it names two while `../regenesis` carries the same 28
  commits in its history and on its private remote (14 branches). A Hatchery rewrite also changes
  each decanted project's recorded base-line sha (inferred, not tested). Its text replacement
  covers both slash spellings, enough for broodforge's content (no slug spelling).

### (3) Hatchery protocol files with the same gap

- `.claude/agents/pusher.md:16` (check 4): a subject and a trailer; no author or committer email.
- `.claude/agents/pusher.md:17-22` (check 5): secrets only, over the net diff; no personal-data
  class (broodforge's state before check 5a).
- `.claude/hooks/git-guard.js:313-483`: rules 1 to 19 read command shape; nothing reads what a push
  sends, from any caller.
- `.claude/agreement/git.md:8-11` and `.claude/skills/commit/SKILL.md:52-58`: a main-session push is
  allowed and names no check; line 67 proves the push landed, not what it carried.
- `.claude/skills/brief/SKILL.md:55-57`: briefs carry absolute drive paths; nothing says committed
  text writes `~`, `<repo>` or `../<dir>` instead, the step 14304c6, 3af9178 and 4904287 went
  through.
- `.claude/rules/docs.md:79-84`: bars sibling names from living docs, not home paths, user names,
  emails or other people's names; lines 82-83 send provenance to findings and commit messages, lines
  89-91 exempt closed records from the path check.
- `.claude/skills/finding/SKILL.md:17-18` and `.claude/skills/finding/template.md:15,19`: "commands,
  versions, environment" and "with the evidence" invite pasted output; line 40 limits size only.
- `docs/reference/claude-code-on-windows.md:82-89` and `:103-110`: no fact that every absolute path,
  scratchpad and project slug on this PC carries the machine user name, or that a grant's project
  scope is such a path.
- `.claude/skills/decant/SKILL.md:20-26`: keeps the full history, sets no repo-local identity.
- `.claude/agents/reviewer.md`: the one review before a push has no privacy item.

### (4) Hatchery's own tree and history (all refs, 121 commits)

- Machine user name, every spelling including the project slug, and the bare name: 0 in the tree, 0
  in history (lines, messages, file names). The scratchpad paths briefs quote were never committed.
- Author and committer: 28 commits (24 dated 2026-10-11, 4 dated 2026-10-12, root included) carry
  the owner's second account (name and email); all on origin/main; GitHub reports the repository
  PUBLIC on 2026-10-17 (0 forks, 0 stars). The same objects are in broodforge's and regenesis's
  history.
- Another person's name in content: 1 line, `docs/findings/2026-10-16-handoff-split-default.md`
  line 43 (2f98979, pushed): the first names of two other people with their commit counts, pasted
  from a sibling's author statistics. Step: a finding that pasted output; nothing reads for names.
- Session links: 25 messages carry a `Claude-Session:` trailer, 5 distinct links (24 dated
  2026-10-13, 1 on 2026-10-15), all pushed.
- 0 each: other emails (6 grep hits are `@` import strings ending `.md` in
  `.claude/hooks/test-hooks.js`), quoted grant files, recorded turns, host names, credentials (1
  hit is check 5's own pattern list).

### (5) Options to carry back or add (for the architect; not a design)

- Check 5a as broodforge has it at e09224b; check 5's secrets scan on the same range.
- Rule 20 with `git.push.privacyAllow` and its ADR 0002 and ADR 0006 rows, fixed for the slug
  spelling, with the email exemption taken from an owner identity in policy, not `git config`.
- An identity step: decant and adopt set the repo-local `user.name` and `user.email`; the scan flags
  any author or committer outside a policy list.
- A write-time rule in the docs rule and the brief skill: committed text writes `~`, `<repo>`,
  `../<dir>`, roles and placeholders; a finding reports classes and counts, never pasted identities.
- An earlier layer (broodforge's own Next item): a PreToolUse scan of Write and Edit, or at commit.
- A git `pre-push` hook (`core.hooksPath`) with the same scan, covering the owner's terminal.
- A reviewer item: report any personal-data class seen. A decision on `Claude-Session:` trailers.
- A reference fact: every absolute path and project slug on this PC carries the machine user name.

### (6) Outcome of the rewrite (added after the work)

The owner's 2026-10-17 history rewrite removed the second account's identity, the session links and
the names found in old blobs from the pushed history of the base and both siblings; HEAD trees are
identical (`2026-10-17-history-rewrite-of-three-repos.md`). Every hash above is the rewritten one.

## Next

- Each sibling, in its own session: re-record its base line and cited hashes from the commit maps
  (`2026-10-17-history-rewrite-of-three-repos.md`), and carry ADR 0018 back
  (`docs/design/decisions/0018-privacy-scan.md`, section 9). Also `../regenesis` and
  `../droplet-management`, not rewritten (mixed authorship is fine): base line from the base map.
- Not determined: which names the 2026-10-15 rewrite removed; whether a `Claude-Session:` link opens
  for anyone but its account; how long Hatchery was public and whether anyone cloned it.
