# 0016 critic review: base-owned and project-owned files

Date: 2026-10-17
Reviewer: critic (Claude Opus 5.5)
Proposal: `0016-base-owned-and-project-owned-files.md`, status under review.

The diagnosis is right: the conflicts come from both sides writing the same file, and splitting by
ownership is the cure. The agreement half of option A (rules in a base-owned file, imported by a
project-owned `CLAUDE.md`) works in the installed Claude Code, subagents included. But the merge-mode
mechanism fails as written, the ownership map is wrong for every adopted project, the queue indexes
cannot be "mixed by design", and the policy half solves a problem the map already solves. Thirteen
objections: two blocking, six serious, five minor.

## Claims checked

Experiments ran in `scratchpad/critic16/` (git 2.56.0.windows.2, Claude Code 2.1.296). The guard
denies `git commit` and `git merge` to this agent, so merge states were built with plumbing
(`git read-tree -m -u <base> <ours> <theirs>`, then `git merge-index -o git-merge-one-file -a`),
which leaves the same stage 1/2/3 index entries a merge leaves. Restores used `git checkout <ours tree> --`,
which takes the same tree-ish argument `HEAD` resolves to. Content merges used `git merge-file`, the
xdiff three-way merge that `ort` uses. Guard verdicts came from feeding tool-call JSON to
`.claude/hooks/git-guard.js` as the main session, `runner`, `coder` and `tester`.

| # | Claim | Result |
|---|---|---|
| 1 | Loader concatenates `./CLAUDE.md` and `./.claude/CLAUDE.md` | Confirmed. `claude -p` in a scratch repo with a sentinel word in each file reported both. `CLAUDE.local.md` was not tested. |
| 2 | `@.claude/agreement.md` resolves, needs no approval, reaches subagents | Confirmed in `-p` mode: the main session and one general-purpose subagent reported the imported sentinel and the nested `@docs/HANDOFF.md` sentinel. The docs say approval applies only to an import "that resolves outside your working directory" (code.claude.com/docs/en/memory). Interactive mode was not tested. After `--resume` the imports were re-read from disk (sentinels changed between runs). Re-expansion after `/compact` was **not verified** (objection 7). |
| 3 | No `.claude/*.md` other than `CLAUDE.md` and `rules/` loads by itself | The docs list only `CLAUDE.md`, `.claude/CLAUDE.md`, `.claude/rules/*.md` and `CLAUDE.local.md`. Consistent with this; not tested separately. |
| 4 | `policy.js` ENOENT gives `DEFAULTS` with `error: null`; the harness passes with no file | Confirmed: lines 132 to 133, and LIVE01 is labelled "loads (or is absent)". The base file equals `DEFAULTS` plus `usage.pool` (checked by script: true). |
| 5 | `post-edit.js` 196 to 202 and check 7 line 28 handle `CLAUDE.md` by exact path | Confirmed. `missingImports` resolves `@.claude/agreement.md` from `.` correctly. But check 7 does **not** enforce any `CLAUDE.md` line cap: only the post-edit warning and the skills' `wc -l` do (objection 8). |
| 6 | Restoring from `HEAD` inside the merge clears unmerged state | **Partly.** UU and UD paths present in ours: cleared, `ls-files -u` empty for them. As specified, though, it fails (objection 2). `:(exclude)` is accepted by `checkout`. |
| 7 | The guard allows `checkout HEAD -- <literal paths>`, `git rm`, `git config merge.ours.driver true` | Confirmed for every caller. But `git ls-files ... \| xargs git checkout HEAD --`, the proposal's own spelling, is **denied for every caller, the main session included**: rule 19, "unexpanded argument". The deny says "Run git by name with its arguments written out", so the message documents the lift. `xargs git rm` is allowed, since `rm` is not on the destructive list. `git merge --no-commit` is allowed for the main session and `runner`, and denied for `coder` and `tester`, as the skill already expects. |
| 8 | Appends under separate headings merge; adjacent appends conflict | Confirmed. One unchanged marker line between the two insertions is enough (exit 0). A blank line plus heading also merges. Two appends at end of file with no marker conflict (exit 1). But see objection 3 for the queue indexes. |
| 9 | `apply --3way` of a deletion onto a modified file fails; `:(exclude)` drops it | Confirmed. "patch does not apply", exit 1, nothing applied. On an unmodified file the deletion applies and the file is gone. With the exclude pathspec the diff stat omits the file. The exclude must stay for ever: right. |
| 10 | `merge=ours` under `git merge` | Not verified: the guard denies this agent `merge`. Relevant only to B. |
| 11 | A pool command that cannot run gives "no reading", not a crash | Confirmed by reading: `usage-gate.js` lines 95 and 205 wrap the pool call and append " No pool reading (...)". |
| 12 | Siblings: base lines, line counts, merge driver unset | Base lines, the missing line in `../broodforge`, and the unset driver: confirmed. **Counts misordered** in Constraints: `../broodforge` is 138 and `../regenesis` is 150 (`wc -l`). |
| 13 | 5 and 17 base commits | Confirmed at 48a43e4. |

## Objections

1. **Blocking. The map's ADR rows are wrong for every patch-mode project.** The map says, "in both
   modes", that `docs/design/decisions/0001` to `0019` are base-owned and `0020` up project-owned.
   ADR 0009 says the opposite: "An adopted project may hold numbers in `0001` to `0019` that are its own."
   Observed: `../regenesis` holds its own 0003 to 0018, including `0003-hatchery-adoption.md` with the
   living `Hatchery base:` line, and `0010-concurrency-redesign.md` beside the base's 0010 name.
   `../droplet-management`'s adoption ADR is 0003. Under the map as written, the file carrying the
   project's base line is one "the project never edits". The proposal also contradicts itself:
   "Patch mode patches exactly this set" (a set that includes ADRs and `LICENSE`), while its own
   patch command covers only `.claude LICENSES`. Promised under ADR 0013, the error would be expensive
   to undo. Resolution: give the map a per-mode row for `decisions/`. In patch mode it is
   project-owned, except the files `adopt-hatchery` step 7 copies by name. Make "patches exactly" match
   the command, or extend the command and say how it meets same-numbered project ADRs.

2. **Blocking. The merge-mode restore fails on its first run, as specified.** In the fixture
   (Hatchery adds an idea, edits an idea and a finding that decant had deleted, deletes `policy.json`,
   edits `CLAUDE.md` and the handoff), four things went wrong:
   (a) The `xargs` spelling is denied by the guard for everyone (claim 7).
   (b) During a merge, `git ls-files -- <paths>` lists paths absent from `HEAD`: base additions and
   modify/delete paths, with a duplicate for each stage. `git checkout HEAD --` with any of them exits 1
   ("pathspec ... did not match any file(s) known to git") and restores **nothing**, so even
   `policy.json` and the handoff stay conflicted.
   (c) Hatchery editing a file that decant deleted is routine: a triage block on an old idea, a
   finding's status flip. That leaves a DU entry that is not in `--diff-filter=A HEAD...MERGE_HEAD`.
   It survives into the "only a mixed index remains" step.
   (d) `git rm` of a staged base addition refuses: "has changes staged in the index", exit 1. git-rm(1)
   says the file must be "identical to the tip of the branch". It needs `-f`.
   A directory-level spelling (`git checkout HEAD -- docs/ideas docs/findings ...`) fails whole as soon
   as one directory holds only its excluded README, which is a queue's resting state, or does not exist
   (`docs/handoff/` in the single-file shape). `--no-overlay` refuses unmerged paths.
   What worked: restore exactly `git ls-tree -r --name-only HEAD -- <project-owned paths>`, written
   out; then `git rm` the unmerged paths absent from `HEAD`; then `git rm -f` the clean additions.
   After that only the index README stayed unmerged. Resolution: put these three steps in the skill.
   Since agents must write the lists out, consider a base-owned helper that prints the literal
   commands (it reads, the agent runs). Add a harness or scripted scratch case reproducing (b) to (d).
   The "two commands" claim and its moving-parts line change accordingly.

3. **Serious. The queue, findings and proposals indexes cannot be "mixed by design".** Their table
   rows are Hatchery's own state: its ideas, findings and proposals. If those rows sit in Hatchery's
   project section, every Hatchery add or delete conflicts with the project's rows. Experiment case 5:
   Hatchery deletes one idea row and adds another, the project has its own row, exit 1, a conflict
   block. That happens on nearly every update, because ideas are deleted when done. If the rows sit
   in the base section instead, they merge cleanly into every project as rows naming files that step
   (2) just removed. The marker mechanism works only where the base's rows are about the base: ADR
   lists, `docs/README.md`. The current skill's "drop their index rows" step has no counterpart in A.
   Resolution: make these READMEs project-owned (restored like the handoff). Move the parts that must
   stay identical (the idea grammar, ADR 0007; the findings frontmatter) to a base-owned file, which
   `.claude/rules/docs.md` already half is. Keep markers for the ADR lists and the map only.

4. **Serious. Deleting the base `policy.json` and adding `~/.claude/usage-pool.json` is not needed,
   and it costs a promised surface.** Under the map, `policy.json` is project-owned. Hatchery is a
   project, so its file is Hatchery's own state, like its `docs/HANDOFF.md`: restored in merge mode,
   excluded in patch mode, replaced by decant step 5 and adopt step 3 from `policy.examples/base.json`.
   Nothing of it reaches a project. The deletion instead brings:
   - a changed meaning for `usage.pool: null` (from "no pool" to "read the user file"), an incompatible
     change under ADR 0013 ("a changed type or meaning"), against "no promised key changes";
   - a new user-scope surface with no stated behaviour when the file is malformed;
   - a three-step ordering (gate first, owner writes the file, then delete);
   - an operator step, and amendments to ADR 0006.
   Whether the switcher command belongs to the user is a fair question, but it is a separate one.
   Resolution: keep Hatchery's `policy.json` as Hatchery's project file. If the user-scope pool is
   wanted, give it its own proposal, with the null-meaning migration stated.

5. **Serious. The ten-year prediction cannot fail, because its antecedent is false on day one.**
   "A project that never edits a base-owned file" matches no adopted sibling. Compared with their
   recorded base (`diff --strip-trailing-cr`, `policy.json` excluded), `../regenesis` has edited 13
   base-owned files (`pusher.md`, `post-edit.js`, `test-hooks.js`, `rules/docs.md`, `settings.json`,
   eight skills). `../dystopo` has edited 11. Some of these led the base: regenesis's split-handoff
   edits to `post-edit.js` and `docs.md` predate ADR 0014. The agreement's own first rule tells project
   sessions to "fix the protocol in the same sitting" by editing agents, skills and hooks, all
   base-owned. Under A, each such fix is a departure needing an ADR and a hand-merge every update until
   it is carried back. The proposal does not describe that carry-back flow. Resolution: a measurable
   prediction, for example "each sibling's next three updates conflict only in recorded departures,
   checked by 2027-04-17". A lightweight departure record (path, reason, carry-back idea) instead of an
   ADR per fix. A stated rule that once Hatchery has the change, base wins.

6. **Serious. A departure written as an adjacent sentence gives two rules where there was one, and
   check 7 forces the import with no opt-out.** The memory docs: "If two files give different guidance
   for the same behavior, Claude may pick one arbitrarily". Today a project edits the rule in place
   and has one rule. `../dystopo`'s 89-line `CLAUDE.md` carries none of the base's rule sections. It
   sets routing per launch, waves and worktrees (its ADR 0020). Under A it must load the full base
   agreement, or fail check 7. The only lift is editing `pusher.md`, itself a departure. No refusal
   text is specified, and no opt-out is documented. Resolution: per-section imports
   (`.claude/agreement/<section>.md`, one `@` line each). A project departs by swapping one import
   line for a project-owned file. Check 7 then verifies that each base section is imported or replaced
   by a file that exists. This keeps the conflict-free property and makes the override real, not
   adjacent text.

7. **Serious. Compaction and runtime observability are unverified for the one file that matters
   most.** The docs say "Project-root CLAUDE.md survives compaction: ... re-reads it from disk". They
   say nothing about imports. Today the agreement is inline. Under A it rides entirely on import
   re-expansion, and this review confirmed that only for `--resume`. The proposal's only detector is
   check 7 at push, a static check. `CLAUDE.md` makes the debug interface part of every feature, and
   the loader offers one: the `InstructionsLoaded` hook fires with load reasons including `include` and
   `compact` (code.claude.com/docs/en/hooks, matcher table). Resolution: before the ADR, run an
   interactive `/compact` with an `InstructionsLoaded` logging hook, or a `/context` check, and record
   the result. Ship a non-blocking logging hook or harness instruction as the debug interface. Also have
   `post-edit.js` warn at the edit when `CLAUDE.md` loses the import line. It already sees every
   Write and Edit of the file, so "not at the edit" is a choice, not a limit.

8. **Serious. The 40-line project cap has no lift and nowhere for the overflow to go.** What decant
   writes today fits in 40 lines: title, provenance, import, Phase, `Branch:`, project rules,
   handoff import, about 15 fixed lines. So do `../broodforge` (one non-base line) and
   `../droplet-management` (23 non-verbatim lines, some of them stale base text). `../regenesis` does
   not: about 76 lines of project-only sections (Two kinds of machine, its own Git, its hooks, live
   data, Phase). Neither does `../dystopo` (89). The proposal sends the overflow to `.claude/rules/`,
   but `.claude/rules/README.md` forbids unscoped rules, and the map files all of `.claude/**` as
   base-owned. A project's `go.md` or `kotlin.md` rule is in the class the project "never edits". Docs
   caps have `edit.capExempt`; this cap has nothing. It is enforced only as a post-edit warning and an
   update-step failure. Resolution: classify project-added files under `.claude/` (rules, agents and
   skills the base does not ship) as project-owned. Name the overflow path (facts to `docs/reference/`,
   path-scoped rules). Either give the cap a lift or say it is advisory.

9. **Minor. The map is not total, and it is not "readable from the tree".** Unclassified paths:
   `.gitignore` (adopt step 5 appends to it), `.editorconfig`, and `.claude/hooks/fitness.json`, whose
   `skip` entries are the coordinator's lift under ADR 0013, project state in a base-owned file. The
   question's own success line says ownership should be readable "from the tree, not from a skill's
   prose", yet the map is carried by the update skill's prose. Resolution: one data file holding the
   map, which the skills and a harness case read, plus a default class for unlisted paths. A fitness
   line could check that every tracked base path is classified.

10. **Minor. Promising the map makes every fix above a major choice.** "Moving a path between classes
    is a major choice from then on": objections 3, 8 and 9 each need such a move. Resolution: promise
    only the direction that hurts projects (project-owned to base-owned). Keep additions and the other
    direction free.

11. **Minor. An unlisted surface and an unannounced change.** The base-owned agreement will read
    values out of the project file by convention: the `Branch:` line, `## Phase`, `## Project rules`.
    Update step 1 does too ("the branch named in the project's `CLAUDE.md` Git section"). That shape is
    a surface, absent from "Surfaces changed". "Routing is base text" changes decant step 4 and adopt
    step 6, which both list routing as a project slot, without saying so.

12. **Minor. Moving parts and exit cost are understated.** Left out: the decant template, the markers
    in each index, `base.json`, the ADR 0006 and 0013 amendments, and the written-out path lists of
    objection 2. The exit cost leaves out that once the map is promised, leaving it is itself a
    proposal. For `../broodforge` the first update keeps its 138-line inline agreement (restored from
    `HEAD`), while the base file lands unimported. Say that check 7 fails until the migration edit.

13. **Minor. Factual.** In Constraints the sibling line counts are misordered (claim 12). The
    "Who else" list cites GitLab's prepend as precedent for the import. That fits, but GitLab still
    merged its repositories (finding, section 4), which is precedent for option E below as much as
    for A.

## Mechanisms that refuse, deny or fail closed

- Check 7's import `FAIL`: lifted by the coordinator editing `CLAUDE.md`, mid-session. There is no
  opt-out for a project that replaces the agreement, and the message text is unspecified (objection 6).
- The 40-line cap: a non-blocking post-edit warning plus an update-step failure. Lifted only by
  trimming, with no policy key (objection 8).
- `git-guard` on the restore: the `xargs` form is denied for every caller. The message names the lift
  (write the arguments out), which is available mid-session. `git merge` is denied for `coder` and
  `tester`, and the message names the agents that may commit.
- The patch-mode exclude and `apply --3way`: a fail-whole, not a refusal. The exclude prevents it
  (claim 9).
- The usage gate: unchanged. Behaviour for a malformed user-scope pool file is unspecified (objection 4).

## The strongest case against, and the best alternative

The proposal keeps two update modes and makes merge mode undo the merge for most of the tree: restore
the project-owned set, remove the base's state, hand-merge what is left. That is a denylist applied
after the fact, and it fails open. Any new base path outside the map lands in every decanted project,
and objections 2 and 3 are both base state leaking through it. The option the proposal names only as
a future trigger is the stronger one.

**E. One update mode: patch everywhere, with the base-owned set as an allowlist:**
`git diff $FROM hatchery/main -- <base-owned set> | git apply --3way`, and a `Hatchery base:` line for
decanted projects at their next update (their merge-base; ADR 0010 already says the line wins).
Hatchery's own `CLAUDE.md`, policy, handoff, ideas, findings and index rows never enter any project, so
there is nothing to restore, remove or row-drop. There is no merge commit and no runner-only merge step.
There is one code path to test instead of two with separate bugs. A new base path outside the list
does not arrive, and a `--stat` outside the list can report it. Its costs: lost shared history in
`git log` and `blame` for base files; patch mode's known whole-apply failure on files a project never
took (fixable with per-file `--exclude`); and the decanted projects' ADR 0001 to 0019 rows, which need
the same per-mode answer as objection 1.

The agreement half of A combines with E unchanged, preferably with per-section imports (objection 6).

## Pre-mortem

It is 2036 and this is the decision the project regrets. The agreement became a 150-line file no
project could change, so projects piled "departure" sentences into 40-line files beside the rules
they contradicted, and the model followed whichever it read last. The siblings that improved the
practice (as regenesis and dystopo did with the split handoff) had to file an ADR for each local fix
and hand-merge it at every update, so they stopped updating. The queue indexes conflicted at every
merge anyway. The restore list in the skill drifted from the map, and new base directories leaked into
projects through merge mode. The line that should have caught it is the question's own promise: "a
project that never edits a base-owned file takes every update with no hand resolution". The sibling
data showed the antecedent false before the ADR was written. Next to it, "concatenation has no
override, so a project never contradicts a base rule silently, it names the departure" assumed the
model resolves named contradictions reliably, which the loader docs disclaim.

## Recommendation

Take the ownership idea and the agreement half of A, with per-section imports. Verify the compaction
behaviour with an `InstructionsLoaded` log before the ADR, and ship that log as the debug interface.
Drop the policy deletion and the user-scope pool file: Hatchery's `policy.json` is just Hatchery's
project file under the map. Fix the map per mode for `decisions/`, make the queue indexes
project-owned, and classify project-added `.claude/` files. Then choose between A's merge-mode
restore, rewritten as in objection 2 and tested in a scratch merge by a runner, and E. I lean to E:
it removes, rather than handles, every leak this review found, and the cost it adds (lost shared
history for base files) is one the owner can judge. Confidence: high that the agreement-by-import
mechanism works and that objections 1 to 3 are real (each reproduced or read in the source); medium
that E beats a corrected A, because the history cost has not been weighed by anyone yet.
