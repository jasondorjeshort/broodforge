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
| 13 | 5 and 17 base commits | Confirmed at 2f98979. |

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

## Second pass: recommendation E

Date: 2026-10-17
Reviewer: critic (Claude Opus 5.5)
Scope: the design of E and its parts as revised. Objections 1 to 13 and the three owner decisions
(merging ends for decanted projects; a 40-line `CLAUDE.md` with `edit.capExempt` as the lift; the
pool command per project) are not re-argued.

The patch-only mechanism holds up under experiment better than the proposal claims. `git apply
--3way` handled an already-applied change, a diverged one, CRLF checkouts and a rename of a
project-edited file. What fails is around it. The helper has no way to tell a decanted tree from an
adopted one once both carry the base line. Hatchery's ADR range runs out in days. Replacement
files and mid-line imports get past check 7. Three helper details lose base files without an error.
Fifteen objections: one blocking, ten serious, four minor.

### Claims checked (second pass)

Experiments ran in `scratchpad/critic16b/` (git 2.56.0.windows.2): `x1.sh`, `x2.sh`, `x3.sh` and
`guard.js`. Each fixture is a Hatchery repository plus a clone acting as the decanted project,
with commits made by `commit-tree`. Each case applies `git diff $FROM hatchery/main -- <paths> |
git apply --3way` on a freshly built fixture, with `FROM` set to `git merge-base HEAD hatchery/main`.

| # | Claim or question | Result |
|---|---|---|
| C1 | First update with shared history: `merge-base` is the right `FROM` | Yes in the fixture. In `../broodforge` (read only) the merge-base is 2f98979, its fetched `hatchery/main`, so its first patch starts exactly where its last merge ended. |
| C2 | Project file already matches the base change | Clean, exit 0, the file is unchanged (case a). |
| C3 | Project diverged on the same line / another line | Same line: `UU`, exit 1, markers (b). Another line: merged, both edits kept (c). |
| C4 | Base adds a file the project also added | Identical content: clean (g). Different content: `AA`, exit 1 (h). |
| C5 | Base deletes a file the project already deleted | **Whole apply fails**: `does not exist in index`, nothing written, although both sides agree (d). Not in the proposal's list of whole failures. |
| C6 | Base edits a file the project deleted; dirty base file | Both fail whole: `does not exist in index` (e), `does not match index` (dirty). A conflict and a hard failure both exit 1; only the output text, or an empty `git status --porcelain`, tells them apart. |
| C7 | Excluding the failing paths | The rest applies, conflicts included (noDE). |
| C8 | CRLF: `core.autocrlf=true`, no `.gitattributes` (the setup of `../droplet-management`, the one sibling without one) | Same results as LF, CRLF kept in the working tree. A clone checked out CRLF whose `autocrlf` was then turned off gave whole-file conflicts on untouched files. That is a broken setup, not E's problem. |
| C9 | Open claim: rename of a project-edited base file | Default `git diff` (renames on): one `rename from/to` hunk, applied cleanly, both edits kept (X7a). `--no-renames`: fails whole (X7b). **The per-path fallback strands it**: `--name-only` lists only the new name, which is applied as a new file with exit 0, and the edited old file stays (X15). |
| C10 | Open claim: `$(node .claude/hooks/ownership.js --paths)` stays allowed | Allowed for the main session, `coder` and `tester`, as are the `FROM=$(grep ...)` form, a `for` loop per path, and `git checkout <ref> -- <path>`. **But unquoted substitution is globbed by the shell**: a printed `docs/design/decisions/00[01]*` expanded to the project's existing 0001 and 0015, and Hatchery's new 0016 was not in the diff, with no error (X8). Quoted, or with a `:(glob)` prefix, it arrived. |
| C11 | Which map computes the pathspec | The project's map (the old one) omitted a path the same base window added along with its map entry. The incoming map included it. After the line moves, no later diff carries it (X9). |
| C12 | `git show hatchery/main:<path>` (the migration step, line 294) | **Fails in Git Bash on this machine**: MSYS rewrites the argument to `hatchery\main;.claude\...` and git reports `ambiguous argument`. Quoting does not help. `MSYS_NO_PATHCONV=1` or `<sha>:<path>` works. Same result in `../broodforge`. |
| C13 | `update.exclude` under a `policy.js` that predates it | A policy holding `{"update": {"exclude": [...]}}` makes the guard deny every git command for the main session, `coder` and `tester`: "policy file is malformed (... unknown key update) ... Every git command is denied until it is fixed; ask the user." |
| C14 | Imports mid-line and in code | The memory docs: "See @README for project overview and @package.json ..." imports both, and "Import parsing skips Markdown code spans and fenced code blocks". `post-edit.js` matches only `^@` and skips fences; check 7 greps `^@` and skips nothing. |
| C15 | `merge=ours` under `git merge` (B only) | Not tested; B is not the recommendation. |

### Objections

14. **Blocking. Once every project carries a base line, the helper cannot tell decanted from
    adopted, so the decanted-only rows stop applying after one update.** `ownership.js --paths`
    prints "the base-owned pathspecs for the tree's mode (the base line, else `merge-base`, as
    update step 3)" (line 62). E writes a base line into every decanted project, at its first
    update or at decant (lines 75, 199 to 200), and under ADR 0010 the line means adopted. So from
    the second update on, `../broodforge` and every new decant get the adopted set. Hatchery's ADRs,
    `decisions/README.md`, the Windows reference and the `docs/design/README.md` marker all stop
    arriving. Nothing fails, because an allowlist fails closed and says nothing. "Hatchery's ADRs
    keep arriving as additions" (line 216) is then false. `merge-base` cannot be the test either: a
    joined adopted project passes it (ADR 0010, Context). The numbering amendment ("next free at or
    above `0020` for every project") is a symptom. It exists only because the line stops meaning
    adopted. For the four siblings it changes nothing harmful (`../broodforge` 0024 either way,
    `../dystopo` 0036, `../regenesis` 0019 to 0020, `../droplet-management` 0004 to 0020), and it
    is pointless for adopted projects, which receive no ADRs under E. Resolution: record the mode
    as one explicit fact that `ownership.js`, update step 3 and the `adr` skill all read. For
    example, `Hatchery base: <sha> decanted`: the anchored regex and `cut -f3` still parse it, so
    it is an addition under ADR 0013. The provenance file name would also work. Or remove the
    decanted-only rows altogether (objection 15, which removes the need).

15. **Serious. Hatchery has three ADR numbers left, and E builds `0020` into every project.**
    Hatchery has used 0001, 0002, 0004 and 0006 to 0015. 0003 and 0005 are held by proposals, and
    0016 is this one. Thirteen ADR files are dated 2026-10-11 to 2026-10-16, so 0017 to 0019 will
    last days. ADR 0008, Consequences: "When Hatchery's next number would be `0020`, supersede this
    record with a wider range for the base and a higher start for projects". `../broodforge`
    already holds 0020 to 0023, and E names `0020-decanted-from-hatchery.md` in every future decant
    (line 75). While Hatchery's ADRs are patched into decanted projects, widening the range in
    place gives an `AA` conflict, or two records with one number, in each of them. The proposal
    amends ADRs 0008 and 0009 on numbering and does not decide this. Resolution: decide it in ADR
    0016. Strongest option: stop shipping Hatchery's records under `docs/design/decisions/`. Either
    put them in a base-owned subtree that no project numbers in, the same path in every project (for
    example `docs/hatchery/decisions/`), or ship them not at all and cite them by
    `git show <sha>:<path>` as ADR 0009 already does for adopted projects. Then Hatchery numbers
    freely, the range retires, and 14, 16 and the marker go with it. Weaker option: a disjoint
    range for Hatchery, named now, with the collisions listed.

16. **Serious. `decisions/README.md` and the Windows reference are mixed in practice, not
    base-owned.** `git diff 2f98979 HEAD` in `../broodforge` (read only) shows `decisions/README.md`
    grown by 21 lines: an index table of Hatchery's ADRs and its own 0020 to 0023. It also shows
    `claude-code-on-windows.md` +28 lines, `0002-hook-specs.md` 6 lines and `docs/design/README.md`
    46 lines. `.claude/rules/docs.md` requires every `docs/` README to list every file, so a decanted
    project must edit a base-owned file for each ADR it writes. That is a departure row whose
    carry-back can never happen ("once Hatchery has the change" cannot apply to the project's own
    rows), so the table becomes a list of permanent mixed files. The open warning that Hatchery's
    `decisions/README.md` lists no ADRs (HANDOFF Next 6) decides this file's class whichever way it
    is fixed. Factual: the proposal says `../broodforge` made "four edits to base files". Seven
    base-owned files under `.claude/` differ from its merge-base (`pusher.md`, `git-guard.js`,
    `shell-parse.js`, `test-hooks.js`, `rules/README.md`, the `adr` and `brief` skills), plus the
    three `docs/` files above. Resolution: class `decisions/README.md` and the Windows reference as
    mixed with a marker, or as project-owned. Under 15's strongest option the question goes away.
    Correct the count, since it sizes the first departure table.

17. **Serious. Replacement files are invisible to the update, and base changes to sections bring
    back the two-rule case.** "A replacement file is read by the loader every turn and by nothing
    else" (lines 272 to 273). The departure table covers edited base paths, not replaced sections.
    So when the base changes `git.md`, a project that replaced it (`../regenesis` will) is never
    told. Rename (X7a): `model-routing.md` to `routing.md` arrives cleanly. Check 7 then fails with
    "section routing neither imported nor replaced" while `@.claude/project/model-routing.md`
    stays. The cheapest edit that passes is adding `@.claude/agreement/routing.md`, which loads two
    rules, the case objection 6 removed. A split section does the same. A removed section leaves its
    replacement loading, and nothing checks it. Resolution:
    - each replacement is a departure row (path, reason, carry-back);
    - the update report lists base changes to every replaced section and every new section;
    - check 7 fails on `@.claude/project/<s>.md` with no `.claude/agreement/<s>.md`;
    - renaming or splitting a section is an incompatible change under the 0013 row, and its
      migration names the replacement files.

18. **Serious. Check 7's exactly-one count sees line-start imports only; the loader also imports
    mid-line and inside imported files.** C14. Each replacement file "states what it replaces"
    (line 108). The natural sentence "Replaces @.claude/agreement/git.md." loads the base section
    too. Check 7 never reads replacement files, and the post-edit hook does not match mid-line. In
    the other direction, a fenced example `@.claude/project/git.md` in `CLAUDE.md` satisfies check
    7's grep while the loader skips it, so the section loads from neither file. Resolution: one Node
    parser that follows the loader's rules (any `@token` outside code spans and fences, followed to
    four hops). Check 7 and `post-edit.js` both call it, and the count covers every file
    `CLAUDE.md` loads. The replacement template names the replaced file in backticks.

19. **Serious. The 40 caps a file, not what loads, and a replacement file is uncapped.** Line 273:
    "no cap". The memory docs: imports "don't reduce its context cost, because imported files also
    load at launch". The proposal's trigger (line 311) covers only an import that replaces no
    section. A `licensing.md` replacement with two lines of licence and sixty lines of project rules
    passes every check, which leaves the 40 the owner chose with nothing to hold. Resolution, not
    re-arguing the number: also cap the loaded total, `CLAUDE.md` plus every file it imports except
    the handoff chain, which has its own 120. Use 40 plus the agreement's 150, resolved by 18's
    parser, with `edit.capExempt` lifting it as it lifts the 40.

20. **Serious. Which map computes the pathspec is unspecified, and the wrong one loses files for
    ever.** X9. If the update reads the project's own `ownership.json`, a base window that adds a
    file and its map entry together never delivers the file. Once the line moves, no later diff
    contains it. On `../broodforge`'s first update the helper is not in the tree at all.
    Resolution: the skill reads the map and runs the helper from the incoming base sha (written to
    a temporary file, or via `git show <sha>:<path>`), and the ADR says so. A harness case adds a
    path and its map entry in one fixture commit.

21. **Serious. Unquoted `$(...)` globbing drops new base files without an error.** C10. The
    decanted-only ADR row needs a glob, and so may an `update.exclude` entry. Resolution: the skill
    runs what `--command` prints, verbatim, never `$(--paths)`. Every pattern the helper prints
    carries a pathspec magic prefix (`:(glob)`, `:(exclude,glob)`), which the shell leaves alone. A
    harness case checks that the output is shell-inert.

22. **Serious. `update.exclude` denies every git command in a project whose `policy.js` predates
    it, and the first update is exactly that case.** C13. ADR 0013 promises that an old file
    loads under a newer base, not that a new key loads under an older `policy.js`. On the first
    update after the switch, the agent reads the new skill against an old tree. When the whole
    apply fails, nothing is written, and E then tells the agent to add paths to `update.exclude`,
    possibly before the new `policy.js` is in the tree. Lift: anyone who edits `policy.json` with
    the Edit tool, mid-session (git is not needed). The message says "ask the user", which stops an
    agent that could fix it itself. Resolution:
    - the skill writes the key only after the incoming `policy.js` is in the tree and
      `node .claude/hooks/policy.js` prints the policy;
    - `ownership.js` on a malformed policy or map prints no command and exits 1 with the fix
      named; it does not fall back to `DEFAULTS`, which would drop the excludes;
    - the guard's malformed-policy message names "a key from a newer base" as a likely cause.

23. **Serious. The per-path fallback silently strands a renamed file.** C9, X15. A whole
    failure caused by an unrelated path (C5, C6) sends every path to the fallback, a renamed
    one included. The rename then arrives as a second copy without the project's edit, exit 0. For
    a skill that means two skills with one name. For a section it means the case in 17.
    Resolution: the fallback iterates `git diff --name-status -M` and gives an `R` entry both of
    its paths in one pathspec. The helper passes `-M` explicitly, since a user's
    `diff.renames=false` would otherwise make every rename fail whole.

24. **Serious. A migration command fails on the target platform.** C12. Line 294 tells each
    sibling to read the skill with `git show hatchery/main:.claude/skills/update-from-hatchery/SKILL.md`.
    In the Bash tool on Windows that is `fatal: ambiguous argument`. An agent that does not know
    why may conclude the file is missing. The new skill's step 1 also says that reading it from
    `hatchery/main` means "stop", which contradicts the migration. The same spelling is in
    `adopt-hatchery` (twice), its ADR template, and ADRs 0002, 0004, 0006 and 0009. That part
    predates this proposal. Resolution: write `<sha>:<path>` (`git rev-parse hatchery/main`
    first) everywhere, and record the MSYS rewrite in `docs/reference/claude-code-on-windows.md`
    (a `finding` first). Step 1 gains the switch-over exception.

25. **Minor. Bumping the base line after a partial apply silently gives up the failed paths.**
    "The base line is bumped by hand" (line 229). A path that failed and was neither excluded nor
    taken drops out of every later diff. Resolution: the bump is the last step and happens only
    when the report gives every failed path a disposition: excluded, taken with
    `git checkout <sha> -- <path>`, or already done.

26. **Minor. The whole failures are not the two the proposal names.** C5 and C6: the
    base deleting a file the project also deleted, and a dirty base-owned file, both fail whole.
    Exit 1 also covers conflicts. Resolution: the skill says that exit 1 with an empty
    `git status --porcelain` is a whole failure, so go per path. A per-path `does not exist in
    index` on a deletion means already done, not an exclude.

27. **Minor. The outside-the-set report is mostly noise, and F7's reach is undefined.** Since ADR
    0008 (fa49e20), 60 paths outside `.claude/` and `LICENSE` changed in Hatchery: 17 ideas, 16
    decisions, 10 proposals, 9 findings and more. A path that is genuinely meant for projects
    drowns in that list. F7 classifies a new path only if `projectOwned` names Hatchery's state
    directories one by one; a `docs/**` entry makes it pass for any docs path. Where F7 runs is not
    said. In a project, "tracked base paths" has no meaning. Base changes to excluded paths are not
    reported, so a guard fix to an excluded hook goes unseen. Resolution:
    - the report lists paths that no list classifies, plus changed excluded paths;
    - `projectOwned` names directories;
    - F7 runs only where there is no `hatchery` remote;
    - `update.exclude` is for files the project removed, and a departure from a hook stays a
      table row.

28. **Minor. Lifts and messages left unstated.**
    - Widening `edit.capExempt` from docs to `CLAUDE.md` is compatible, but ADR 0006's key table
      ("Docs exempt from the 300-line cap") needs the amendment. A project's broad pattern could
      now exempt `CLAUDE.md` without anyone meaning it.
    - Neither check 7's `FAIL <path>: <n> lines, cap <cap>` nor the post-edit message names the
      lift.
    - "The agreement files 150 together" has no stated home, scope or lift. It is not said whether
      it runs in projects, where an in-place edit could fail a push on base-owned text.
    - `.claude/rules/docs.md` loads for `docs/**` and `CLAUDE.md` only. Add
      `.claude/agreement/**` and `.claude/project/**`.

    Resolution: one line each in the ADR.

### Mechanisms that refuse, deny or fail closed (second pass)

- **Check 7, section line.** The pusher refuses. Lift: edit `CLAUDE.md`, from any session,
  mid-session, then a fresh pusher. The message names the section. An orphan replacement and a
  mid-line import pass it (17, 18).
- **The 40, and the 150 together.** A post-edit warning, then a check 7 `FAIL`. Lift:
  `edit.capExempt` naming `CLAUDE.md`, written by the coordinator mid-session. Neither message
  names it (28). The 150's lift is unspecified.
- **F7 and `ownership.js --check`.** The harness fails in Hatchery. Lift: classify the path in
  `ownership.json`. Message unspecified.
- **Malformed `ownership.json`.** Behaviour unspecified. It should refuse to print a command.
  Lift: take the base's copy with `git checkout <sha> -- .claude/hooks/ownership.json`, allowed
  for every caller (22).
- **`git apply --3way` whole failure.** A failure, not a refusal. Lift: per path, then exclude or
  take the base's. The messages are git's own (26).
- **`AA` collision.** Lift: rename the project file, or take the base's. The report names it.
- **Guard on a policy with a key unknown to the tree's `policy.js`.** Every git command is denied
  for every caller, main session included. Lift: edit `policy.json` (Edit tool) or bring in the
  new `policy.js`, mid-session. The message says "ask the user" (22).
- **Guard on E's own commands.** Allowed for every caller tested, `coder` and `tester` included
  (C10). No runner-only step remains.

### Pre-mortem

It is 2036. Patching worked: three-way handled almost every update. What went wrong was around it.

- From their second update on, the decanted projects quietly stopped receiving Hatchery's
  records. Their `.claude/` cited ADRs they did not hold, and no report said so, because the
  allowlist fails closed. The line that should have caught it: "for the tree's mode (the base line,
  else `merge-base`)".
- A week after this ADR, Hatchery needed `0020`. The widened range collided with
  `../broodforge`'s 0020 to 0023 and every `0020-decanted-from-hatchery.md`, and was settled
  differently in each project. Line: ADR 0008, "The range is finite".
- Projects escaped both the 40 and the base's evolving rules through replacement files. A replaced
  `git.md` kept telling agents to do what a later guard denied, and nobody was told the base had
  changed it. Line: "read by the loader every turn and by nothing else".
- A per-path fallback once left two copies of a renamed skill. A helper run through `$(...)` never
  delivered a new reference file. The base line moved past paths that had failed. Each loss was
  silent.

The prediction "each sibling's next three updates conflict only on paths in its departure table"
would have passed through all of it, because none of these losses is a conflict. A second
prediction belongs beside it: after each update, the incoming base's map applied to `FROM..sha`
equals what the project took plus its excludes.

### The strongest case against E as designed, and the best alternative

Patching from an allowlist is sound for `.claude/` and the licence files: every git behaviour this
pass tested comes out right or is fixed with one line in the helper. The weak part is the
"decanted only" half. It needs a mode the tree no longer records (14), a number range about to run
out (15), files that are mixed in practice (16), a marker, and two numbering amendments. The
alternative the proposal did not consider: **one base-owned set for every project.** Hatchery's
records live in a base-owned subtree at the same path in every project, or are not shipped and are
cited by `<sha>:<path>`. `docs/` is otherwise wholly project-owned. There is no mode, no marker, no
range, and ADRs 0009 and 0010 have nothing left to decide. Its cost: moving Hatchery's 16 records
and the citations to them, which is prose ADR 0013 does not promise. In exchange, adopted projects
get the records too, which closes ADR 0009's "a base ADR number cited in `.claude/` can name a
different file".

### Recommendation (second pass)

Keep E's patch-only update. The experiments support it: three-way handled already-applied,
diverged, CRLF and renamed files, and the guard allows every step for every caller. Do not write
the ADR until 14 is fixed and 15 is decided. I recommend deciding 15 by giving every project one
base-owned set, which dissolves 14, 16, the marker and the numbering amendments together. Put 20,
21, 23 and 25 into the helper and skill spec (one line each), 17 to 19 into check 7's spec as one
loader-faithful parser, and 22 into the skill's order. Confidence: high that 14, 18, 20 to 24 are
real (each follows from the proposal's text plus an experiment or the loader docs). High that the
patch mechanics are right. Medium that a base-owned records subtree beats a disjoint range for 15,
because the citation rewrite has not been costed.
