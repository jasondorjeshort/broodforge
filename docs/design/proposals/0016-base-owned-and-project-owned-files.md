# 0016. Base-owned and project-owned files: an update touches nothing the project wrote

Date: 2026-10-17
Status: under review

## Question

Every `update-from-hatchery` conflicts on `.claude/hooks/policy.json` and usually on `CLAUDE.md`
(the owner, 2026-10-17). The evidence is `docs/findings/2026-10-17-base-repo-defaults-and-upstream-merges.md`
(the finding): both files mix base-owned and project-owned text, the base changed them in 5 and 17
commits, and in every surveyed practice that reports fewer conflicts upstream and downstream
stopped writing the same file; frequent merges and `git rerere` shrink a conflict without removing
it. The owner suggested "a default/ directory or something" and accepted the coordinator's reading:
the cause is ownership, not location; a copy the project edits at a path the base ships still
conflicts. What is decided: which paths the base owns, which the project owns, which are mixed by
design, and the mechanism by which an update leaves every project-owned file untouched. Measurable
prediction (replacing the first draft's, whose antecedent no sibling meets; objection 5): each
sibling's next three updates conflict only on paths in its departure table, checked by 2027-04-17.

## Constraints that matter here

- The update must come out conflict-free for every path the project has not edited, in a decanted
  project (shared history, ADR 0004) and an adopted one (patch from a base line, ADRs 0004, 0010).
  A merge brings the base's whole tree, including the base's own state (its `CLAUDE.md` phase line,
  handoff, ideas, findings, proposals, index rows, policy); a design that keeps a merge handles them.
- Promised surfaces (ADR 0013): the policy file's keys, `@docs/HANDOFF.md` in `CLAUDE.md`, the
  base line, the hook seams. An incompatible change to any names how each sibling migrates.
- Claude Code 2.1.296 loads `./CLAUDE.md`, `./.claude/CLAUDE.md` and unscoped `.claude/rules/*.md`
  concatenated with no override; `@` imports resolve from the importing file, four hops, a missing
  target loads nothing and says nothing (ADR 0014). Verified by the critic in `-p` mode: an
  `@.claude/...` import loads for the main session and a subagent, needs no approval inside the
  repository, and is re-read on `--resume`; re-expansion after `/compact` is unverified.
- `policy.js` treats an absent file as `DEFAULTS` and a missing key as the base value (lines 125 to
  134); the base's own `policy.json` is `DEFAULTS` plus `usage.pool`, the owner's machine-level
  account-switcher command (ADR 0011).
- `CLAUDE.md` is capped at 150 lines, a constant in `post-edit.js`, which also resolves its `@`
  lines (lines 196 to 202); pusher check 7 resolves `@` lines in `CLAUDE.md` and the handoff files
  and caps the handoff, not `CLAUDE.md`; the skills' `wc -l` is the only other check of the 150.
- Git 2.56.0, verified by the critic and this revision: `git apply --3way` merges an append above
  an unchanged marker line with an append below it cleanly, and conflicts on two appends at one
  spot (so ADR 0008's "the merge takes both sides' lines" is false without a marker); a base
  addition of a file the project also added gives an ordinary `AA` marker conflict; a deletion
  hunk onto a modified file fails whole, and `:(exclude)` drops it from the diff. `merge=ours`
  needs `git config merge.ours.driver true` per clone and falls back to a plain conflict silently.
  The guard allows `git diff ... | git apply --3way` for every caller, denies `git merge` to
  `coder` and `tester`, and denies any `xargs git checkout` to everyone (rule 19).
- The siblings, 2026-10-17: `../broodforge` has no base line and a `hatchery` remote (merge mode);
  `../regenesis`, `../droplet-management` and `../dystopo` carry a `Hatchery base:` line (patch
  mode). `CLAUDE.md` is 138, 150, 150 and 89 lines there; project-only lines about 1, 76, 23 and
  89. Each has added its own files under `.claude/` (`rules/go.md`, `rules/kotlin.md`, a `wave`
  skill); `../regenesis` has edited 13 base-owned files against its recorded base, `../dystopo` 11,
  some ahead of the base (the split handoff before ADR 0014). ADR 0009: an adopted project's
  `0001` to `0019` may be its own (`../regenesis` holds 0003 to 0018).
- Ruled out before this proposal: frequent merges and `git rerere` alone; a `default/` directory
  of copies the project then edits in place (the same both-sides edit at a new path).

## The ownership map (shared by the options)

Each option decides the mechanism; all need the same map. The map is data, not prose:
`.claude/hooks/ownership.json` (base-owned), four lists of pathspecs (`baseOwned`,
`baseOwnedDecanted`, `mixed`, `projectOwned`). `node .claude/hooks/ownership.js --paths` prints
the base-owned pathspecs for the tree's mode (the base line, else `merge-base`, as update step 3),
`--command` the update command written out, `--check` the tracked base paths no list matches;
harness case F7 fails on any such path, so a new base path is classified when it is added. The
default class for an unlisted path is project-owned: nothing unclassified arrives.

| Class | Paths | Rule |
|---|---|---|
| Base-owned, both modes | `.claude/**` as the base ships it, except `hooks/policy.json`, `settings.local.json` and the mixed files below; `LICENSE`, `LICENSES/` | Patched. A project's edit is a departure: kept where the base did not touch the lines (three-way); on a conflict the base wins unless the path is in the project's departure table (below). |
| Base-owned, decanted only | Hatchery's ADRs `docs/design/decisions/0001` to `0019` and `decisions/README.md`; `docs/reference/claude-code-on-windows.md` | Patched; the project's living lines in a base ADR (`Status: superseded by 0020`, a living-table row) are hand-merged on conflict (ADR 0008). An adopted project owns all of `decisions/`, its own numbers below `0020` included (ADR 0009); nothing under `docs/` is patched there (ADR 0004: the skeleton is copied once). |
| Mixed by design | `.claude/settings.json` (base entries, the project's entries); `.claude/hooks/fitness.json` (`deps` the base's, `skip` the project's); decanted only: `docs/design/README.md` (Hatchery's rows above the `## Project decisions` marker, the project's below) | Patched three-way; the one place a hand-merge is expected. Each side appends on its own side of the marker. |
| Project-owned, the default | `CLAUDE.md`, `.claude/hooks/policy.json`, every path under `.claude/` the base does not ship (`rules/<lang>.md`, project skills and agents, `.claude/project/`), `README.md`, `.gitignore`, `.gitattributes`, `.editorconfig`, everything under `docs/` not named above: the handoff, the queues and their READMEs, findings, proposals, reference, the design docs, `docs/README.md`, the project's ADRs | Never patched. The base's copies are the base's own state and never enter a project. |

The departure table: a living table in the project's provenance ADR (the adoption ADR today; a
decanted project gets `0020-decanted-from-hatchery.md` at decant, or at its first update under
this decision), one row per base-owned path the project edits: path, reason, carry-back (an idea
file, or the Hatchery commit that took it). While a row exists the project wins on that path; once
Hatchery has the change, base wins and the row goes. A path the project never wants the base's
version of (its own `LICENSE`, a skill it removed) goes in a new policy key `update.exclude`
(default `[]`, an addition under ADR 0013), which the helper turns into `:(exclude)` entries.
Promised under ADR 0013 (one row: the map, carried by `ownership.json`; readers: the update,
`decant` and `adopt-hatchery` skills, the harness): moving a path a project may have written from
project-owned to base-owned is a major choice; adding a newly shipped base path, or a move toward
project-owned, is free. A new base file colliding with a project-added file of the same name is an
`AA` conflict named in the report; the project renames its file or takes the base's.

## Options

### A. The agreement in base-owned files, a short project-owned `CLAUDE.md`; merge mode repaired

Agreement. The rule sections of today's `CLAUDE.md` move verbatim to base-owned files under
`.claude/agreement/`, one per section (`this-agreement.md`, `where-things-live.md`,
`delegation.md`, `debug-interface.md`, `major-decisions.md`, `usage-gating.md`,
`model-routing.md`, `git.md`, `licensing.md`; one file in the first draft, changed by objection
6), each starting with its `##` heading, written so no sentence carries a project value:
thresholds are "the lines in `.claude/hooks/policy.json`" (`node .claude/hooks/policy.js` prints
them), the branch is "the `Branch:` line of `CLAUDE.md`", the licence is "`LICENSE`" with
GPL-3.0-or-later as the base default. Routing is base text since the models are pinned in
base-owned agent files. `this-agreement.md` says the files are edited in Hatchery only, and that in
a project a fix to a base-owned file is a departure: a table row and a carry-back idea, same sitting.

Project file. `CLAUDE.md` becomes the project's: title, a provenance line, `@docs/HANDOFF.md`,
`## Phase`, `Branch:`, nine lines `@.claude/agreement/<section>.md`, `## Project rules`; at most 40
lines (owner question, below), the agreement files 150 together; `policy.js --cap-check` learns
both so `post-edit.js`, check 7 and update step 6 read one rule, and `edit.capExempt` lifts the 40
as it lifts the 300. A project departs from a section by replacing its import line with
`@.claude/project/<section>.md`, a project-owned file (the base never ships that directory;
nothing under `.claude/` loads on its own except `rules/`) that states what it replaces and names
the project's ADR; one line is enough. The model then reads one rule, not two. Check 7 gains: for
each `.claude/agreement/<s>.md`, `CLAUDE.md` has exactly one of `@.claude/agreement/<s>.md` or
`@.claude/project/<s>.md`, else `FAIL CLAUDE.md: section <s> neither imported nor replaced` (or
`imported twice`); the resolve check covers the target. `post-edit.js` warns at the edit when
`CLAUDE.md` loses an agreement line. Overflow from the 40: facts to `docs/reference/`, path-scoped
rules to `.claude/rules/<name>.md` with `paths:`, procedures to a project skill, every-turn rules
to `## Project rules` or a replacement section.

Placement. `.claude/CLAUDE.md` was weighed: load order is the loader's, the project file does not
show the agreement exists, `/init` writes `./CLAUDE.md` unaware of it, neither the hook nor check 7
sees it. An unscoped rules file is the same mechanism and `.claude/rules/README.md` forbids it. The
`@` import is documented, relative, one hop, ordered by the project file, resolved by the checks
that exist, and is the finding's one-line hook (GitLab EE's prepend, Brave's `chromium_src`).

Debug interface. `docs/reference/agreement-loading.md` (verified, observed): the critic's sentinel
procedure (`claude -p` with a sentinel word per agreement file, main session and a subagent, then
`--resume`) and the compaction check (interactive `/compact` with an `InstructionsLoaded` hook
logging the `include` and `compact` reasons; a one-time instruction, not a shipped hook). The
compaction check runs before the ADR; a fail moves the agreement to `.claude/CLAUDE.md` with the
hook and check 7 pointed at it.

Update, step 5 in merge mode, corrected after the critic's fixture (objection 2): `git merge
--no-ff --no-commit hatchery/main` (runner only, a commit under policy); restore exactly `git
ls-tree -r --name-only HEAD -- <project-owned paths>`, written out (`xargs` is denied; `ls-files`
lists paths absent from `HEAD` and one such path fails the whole checkout; a directory spelling
fails on a directory holding only its README); `git rm` the unmerged paths absent from `HEAD` (a
base edit to a file decant deleted leaves a `DU` entry the `--diff-filter=A` list misses); `git rm
-f` the base's clean additions under project-owned directories; then `git diff --name-only
--diff-filter=U` is a mixed file or a departure. The helper prints the three commands. Patch mode:
`git diff $FROM hatchery/main -- <base-owned paths> ':(exclude).claude/hooks/policy.json' | git
apply --3way`; the exclude stays for ever since the base edits its own policy file.

Policy. The first draft deleted the base's `policy.json` and read `~/.claude/usage-pool.json` when
`usage.pool` is `null`. Withdrawn (objection 4): under the map the base's file is Hatchery's
project-owned state like its handoff, so it reaches no project in either mode, and `usage.pool:
null` keeps its meaning. `decant` step 5 sets `usage.pool` to `null` unless the machine has the
switcher. The pool's home is an owner question, below.

Meets the constraints: no conflict on either file; the base's own state is removed after the fact,
by a denylist. Risks: the denylist fails open (a base path outside the map lands in every decanted
project); three restore steps with written-out lists; a runner-only merge; two code paths, each
exercised by one project. Who else: Django `global_settings` plus a project file that "doesn't
have to define any settings"; GitLab's one-line prepend (GitLab then merged its repositories, which
is precedent for E as much as for A); Debian conffiles. Exit cost: paste the agreement back into
each `CLAUDE.md`; one update per sibling; the map stays useful. Moving parts: nine agreement files,
nine import lines per project, the project template in `decant`, two caps and a path rule in
`policy.js`, check 7's section line, the reference entry, `ownership.json` and its helper, the
departure table, `update.exclude`, three restore steps, one exclude, the marker in
`docs/design/README.md`, amendments to ADRs 0007, 0008, 0013.

### B. Keep both files; automate the project-wins rule

`.gitattributes` gains `.claude/hooks/policy.json merge=ours` and `CLAUDE.md merge=ours`; the
update and decant skills run `git config merge.ours.driver true`; patch mode adds `--exclude` for
both. Meets the constraints for `policy.json` exactly. For `CLAUDE.md` it meets the letter and
breaks the point: the base's edits to the rule sections never reach any project again (the
finding: "the project wants the base's lines"), so the agreement forks silently the day this
lands. The driver is per clone: a fresh clone gets the old conflict, silently; `.gitattributes` is
a file every sibling extended. Nothing is done about the base's own copies of project-owned files.
Who else: Pro Git's `database.xml merge=ours`, cruft `skip`, copier `_skip_if_exists`, all for
files the downstream never wants upstream's version of, which `CLAUDE.md` is not. Exit cost: two
attribute lines and a config key; then hand-merge every base rule edit skipped while B ran. Moving
parts: two attribute lines, a per-clone setting the skills keep setting, two flags, a guard allowance.

### C. A layered policy: `policy.json` base-owned, `policy.local.json` project-owned

`policy.js` reads `policy.json` (the base's) then `policy.local.json`, each local key replacing
the base's; a second seam names the local file; `CLAUDE.md` is not addressed. Meets the policy
constraint, and ships the base's `usage.pool` to every project on every machine: without the
switcher the gate reports no reading at each HOLD deny, noise for a value that is the owner's. The
defaults exist twice and the harness must keep them equal. The promised policy row changes
incompatibly: every sibling renames its file. Who else: dpkg fragments, systemd drop-ins, Compose
override, with the documented cost that an overlay goes stale silently after a structural change.
Exit cost: fold each local file into one, revert the loader and seam, one update per sibling.
Moving parts: a second file per project, a second seam, a layer step with harness cases, a
`DEFAULTS`-equals-file check.

### D. Status quo

Step 5 as written: hand-resolve by rule at every update. Cost per update: the policy conflict, the
`CLAUDE.md` conflict, in merge mode the handoff conflict, the index conflicts and the clean-adds
cleanup. Who else: GitLab CE into EE before the repositories merged, "days of engineering time" on
"busy work". Exit cost: none. Moving parts: none new; the prose rule keeps growing.

### E. One update mode: patch everywhere from an allowlist (the critic's alternative)

Mechanism. `update-from-hatchery` has one step 4 for every project: `git diff $FROM hatchery/main
-- <base-owned paths> ':(exclude).claude/hooks/policy.json' <update.exclude entries> | git apply
--3way`, the line `ownership.js --command` prints, run by the agent running the skill (allowed for
every caller; no runner-only step, no merge commit). `FROM` is the base line (ADR 0010); a decanted
project without one uses `git merge-base HEAD hatchery/main` once, and the same update writes its
provenance ADR with the line and an empty departure table, so the next update is a patch like every
other. ADR 0004's patch mode becomes the only mode; its path list grows from `.claude CLAUDE.md
LICENSES` to the map's base-owned set and loses `CLAUDE.md`. Conflicts: `git apply --3way` leaves
markers (`U`, or `AA` for a colliding addition), resolved by the map's rule; a hunk it cannot merge
(a deletion onto a modified file, a file the project never took) fails the whole apply with nothing
written, and the skill then applies per path (`git diff $FROM hatchery/main -- <path> | git apply
--3way`, one printed line per changed path) and reports the paths that failed; the project adds
each to `update.exclude` or takes the base's version. `git diff --name-only $FROM hatchery/main`
minus the base-owned set is reported as "changed in the base outside the base-owned set": a new
base path is seen, not leaked. The agreement half of A combines unchanged.

Decanted projects. Shared history stays in the DAG up to the switch, and `git merge-base` stays
there; after it, `git log -- .claude/hooks/git-guard.js` in the project shows `Update from
Hatchery <a>..<b>` commits whose bodies list the incoming base commits (step 3's list, as today),
and `git blame` points at them; the base's history is `git log hatchery/main -- <path>`, one hop,
since the remote is fetched for every update. Three of four siblings live this way already.
Hatchery's ADRs keep arriving as additions (ADR 0008); `docs/design/README.md` takes them under the
marker. Numbering: a project with the line and Hatchery's ADRs in its tree must not number in
`0016` to `0019` (an incoming base ADR would collide), so the rule becomes "every project numbers
from the next free number at or above `0020`", one rule readable from the tree, amending ADR 0009
(its tie-break goes: the line decides mode and `FROM`, not numbering) and ADR 0008 for the adopted
case (a project below `0020` jumps to it; nothing is renumbered).

Meets the constraints by construction: Hatchery's `CLAUDE.md`, policy, handoff, queues, findings,
proposals and index rows never enter any project, so there is nothing to restore, remove or
row-drop, and a path outside the list cannot leak. One code path, exercised by every project on
every update. Costs: shared history for base files ends at the switch (above); the whole-apply
failure, mitigated per path; a renamed and project-edited base file merges worse under `apply`
than under `merge` (a conflict or a failed path; no file under `.claude/` has been renamed since
decant existed); the base line is bumped by hand; the owner's 2026-10-12 sentence "merging stays
the path for projects that are based on Hatchery" (ADR 0004) is superseded, which the coordinator
reports. Who else: Debian quilt and gbp (the finding: upstream untouched, the delta apart, cost
"the lack of history"), cruft and copier (template updates by `git apply -3`), the three adopted
siblings since 2026-10-12. Exit cost: low; histories are still joined, so `git merge hatchery/main`
works again at any time, and a first merge after a patch period is clean where both sides hold the
same content. Moving parts: the agreement parts of A; `ownership.json` and its helper; the
departure table and `update.exclude`; one exclude; the provenance ADR template (the adoption
template generalised); the marker; amendments to ADRs 0004, 0007, 0008, 0009, 0010, 0013.

## Recommendation

E for the update, with the agreement half of A (per-section imports, the 40-line project file,
check 7's section line, the reference entry) and without A's policy half. E over A's merge-mode
repair because E removes every leak the critic found where A handles each after the fact: A
restores and deletes the base's state with three written-out lists that fail open on any new path,
keeps a runner-only merge, and keeps two code paths each tested by one project; E has one command,
allowed for every caller, that cannot carry a path not on the list. The cost E adds is shared
history for base files after the switch, which today one sibling has and three do not; the update
commit body and the fetched remote keep that history one hop away. What would reverse it: the
owner wanting `git blame` on base files inside a decanted project to reach Hatchery's commits (then
A's merge mode as corrected, the three restore steps printed by the helper); the loader dropping
or requiring approval for an in-tree `@` import (then `.claude/CLAUDE.md`).

Owner questions, open in the handoff; the recommendation does not decide them:
- The project file's cap: 40 lines, lifted by `edit.capExempt`, or another number, or advisory.
  Forty fits a decant, `../broodforge` and `../droplet-management`; `../regenesis` (about 76
  project-only lines) and `../dystopo` (89) would trim or move text as described under A.
- The switcher command's home: per project in `policy.json` (today, unchanged here) or per user
  in `~/.claude/usage-pool.json` (its own proposal, with the `null`-meaning migration stated).

First step: the compaction check (A, Debug interface). Then the ADR (0016), amending 0004, 0007,
0008, 0009, 0010, and adding the map, the agreement imports and the `Branch:` line to 0013. Then
one coder brief: `ownership.json`, `ownership.js`, F7, the caps and path rule in `policy.js`, the
`post-edit.js` warning, `update.exclude`; one tester brief: the skills (`update-from-hatchery`
around the one command, `decant` 4 and 5 with the `CLAUDE.md` template and the provenance ADR,
`adopt-hatchery` 3 and 6 without the routing slot, the provenance template, check 7's line) and
Hatchery's own migration (`.claude/agreement/`, the short `CLAUDE.md`, the marker, the reference
entry). Then each sibling at its next update, run from the skill at `hatchery/main` (`git show
hatchery/main:.claude/skills/update-from-hatchery/SKILL.md`) since its own copy describes the old
step: rewrite `CLAUDE.md` to the project shape in the same commit (check 7's section line fails
until it is), keep `policy.json`, write the departure table from its base-owned edits;
`../broodforge` also writes its provenance ADR with the merge-base.

## What would change this recommendation

- An in-tree `@` import needs approval, does not resolve, or does not re-expand after `/compact`:
  take `.claude/CLAUDE.md` with the hook and check 7 pointed at it.
- The owner wants Hatchery's commits in a decanted project's `blame`: A's merge mode as corrected.
- A project needs to override one sentence of a section, not the section: the agreement gains a
  parameter (a policy key) for that rule; the replacement file is the stopgap.
- `git apply --3way` fails whole on most updates in practice: per-path application becomes the
  default, or renames get a `--find-renames` diff plus a rename step in the skill.

## Claims to verify

Verified by the critic (its table, claims 1 to 5, 7 to 9, 11 to 13) and this revision: the loader
concatenates both project files; the import loads for the main session and a subagent, needs no
approval, is re-read on `--resume`; `policy.js` ENOENT gives `DEFAULTS`; `post-edit.js` and check
7 handle `CLAUDE.md` by exact path and check 7 does not cap it; the guard verdicts above; the
marker merge under `git merge-file` and `git apply --3way`; the deletion-hunk failure and
`:(exclude)`; the `AA` conflict on a colliding addition; the pool code path; the sibling facts.
Open: import re-expansion after `/compact` (the check before the ADR); `git apply --3way` on a
rename of a project-edited base file; `merge=ours` under `git merge` (B only); that `ownership.js
--paths` fed to `git diff` by `$(...)` stays allowed (it did in this revision's guard test;
writing the paths out is the fallback).

## Critic's objections and responses

1. The map's ADR rows are wrong for every patch-mode project, and "patches exactly this set" does
   not match the command. Accepted with change: a decanted-only row for Hatchery's ADRs and the
   decisions README; an adopted project owns all of `decisions/`, its adoption ADR included;
   nothing under `docs/` is patched there. The command is now the map (the helper prints it).
2. The merge-mode restore fails on its first run: `xargs` denied, `ls-files` lists paths not in
   `HEAD`, `DU` entries survive, `git rm` needs `-f`. Accepted; A's text is corrected to the three
   steps that worked, printed by the helper. Under E there is no restore, so the failure cannot
   recur; F7 replaces the harness case for (b) to (d).
3. The queue, findings and proposals indexes cannot be mixed by design. Accepted: project-owned in
   both modes; the grammar they share lives in base-owned `.claude/rules/docs.md` and ADR 0007, and
   a grammar change is an incompatible surface change whose migration note names the READMEs. The
   marker stays only for `docs/design/README.md` in decanted projects. Amends ADR 0007.
4. Deleting the base `policy.json` and adding a user-scope pool file is not needed and costs a
   promised meaning. Accepted: the coordinator's reading is the same; the base's file is Hatchery's
   project-owned state, off the list in both modes; `usage.pool: null` keeps its meaning; the
   deletion, the user-scope file, the ordering and the ADR 0006 amendment are withdrawn. The
   switcher's home stays an owner question, its own proposal if wanted.
5. The ten-year prediction is false on day one; siblings edit base-owned files routinely; no
   carry-back flow. Accepted with change: a dated, measurable prediction (Question); the departure
   table instead of an ADR per fix; the carry-back rule (base wins once Hatchery has the change);
   `this-agreement.md` says what "fix the protocol in the same sitting" means in a project.
6. A departure as an adjacent sentence gives two rules; check 7 forces the import with no opt-out.
   Accepted with change: per-section files, a replacement file per departed section in
   project-owned `.claude/project/`, check 7's exactly-one rule with its `FAIL` text. Nine lines in
   the project file and nine base files are the price of a real override.
7. Compaction and runtime observability unverified. Accepted with change: the compaction check is
   the first step before the ADR; the sentinel procedure and the `InstructionsLoaded` snippet are a
   reference entry (the debug interface), not a shipped hook; `post-edit.js` warns at the edit.
8. The 40-line cap has no lift and nowhere for the overflow to go; project-added `.claude/` files
   are misclassified. Accepted with change: project-added paths under `.claude/` are project-owned
   (the diff never carries them; a collision is an `AA` conflict); the overflow path is named;
   `edit.capExempt` is the lift. The number is the owner's question, kept open.
9. The map is not total and not readable from the tree. Accepted: `ownership.json`, a default
   class, the helper, F7; `.gitignore`, `.gitattributes`, `.editorconfig` project-owned;
   `fitness.json` mixed (`deps` the base's, `skip` the project's).
10. Promising the map makes every fix a major choice. Accepted: only the move of a path a project
    may have written into the base-owned set is promised.
11. The `Branch:` line and the slot shape are an unlisted surface; routing silently stops being a
    slot. Accepted: both are ADR 0013 rows in the first-step list; `decant` step 4 and `adopt` step
    6 lose the routing slot (a project adds agents in `## Project rules` or replaces
    `model-routing.md`).
12. Moving parts and exit cost understated; `../broodforge` runs unimported until it migrates.
    Accepted: both lists restated under A and E; the migration says check 7 fails until the
    `CLAUDE.md` rewrite, made in the same commit as the update.
13. Factual: the sibling counts were misordered; GitLab merged its repositories. Accepted: counts
    fixed (138, 150, 150, 89); the GitLab sentence under A now says so.
