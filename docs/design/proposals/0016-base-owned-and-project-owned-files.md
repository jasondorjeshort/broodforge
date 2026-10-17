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
predictions (replacing the first draft's, whose antecedent no sibling meets; objection 5), both
checked by 2027-04-17: each sibling's next three updates conflict only on paths in its departure
table; and after each of those updates `git diff --name-only <FROM> <sha> -- <the base-owned set>`
equals the paths the update took plus the paths its report gave a disposition (excluded, departed,
already done), so a silent loss fails the prediction even though it is not a conflict (the critic's
second pass).

## Constraints that matter here

- The update must come out conflict-free for every path the project has not edited, in a decanted
  project (shared history, ADR 0004) and an adopted one (patch from a base line, ADRs 0004, 0010).
  A merge brings the base's whole tree, including the base's own state (its `CLAUDE.md` phase line,
  handoff, ideas, findings, proposals, index rows, policy); a design that keeps a merge handles them.
- Promised surfaces (ADR 0013): the policy file's keys, `@docs/HANDOFF.md` in `CLAUDE.md`, the
  base line, the hook seams. An incompatible change to any names how each sibling migrates. The
  paths of Hatchery's records, skill prose and hook comments are not promised.
- Claude Code 2.1.296 loads `./CLAUDE.md`, `./.claude/CLAUDE.md` and unscoped `.claude/rules/*.md`
  concatenated with no override; `@` imports resolve from the importing file, four hops, a missing
  target loads nothing and says nothing (ADR 0014). Verified by the critic in `-p` mode: an
  `@.claude/...` import loads for the main session and a subagent, needs no approval inside the
  repository, and is re-read on `--resume`; re-expansion after `/compact` is unverified. The loader
  also imports mid-line ("See @README for project overview and @package.json", the memory docs) and
  skips code spans and fenced blocks; `post-edit.js` matches `^@` outside fences and check 7 greps
  `^@` and skips nothing (critic C14).
- `policy.js` treats an absent file as `DEFAULTS`, a missing key as the base value, and an unknown
  key as a malformed file (lines 114, 125 to 150); on a malformed file the guard denies every git
  command for every caller, the main session included, and says "ask the user" (`git-guard.js`
  line 563; critic C13). The base's own `policy.json` is `DEFAULTS` plus `usage.pool`, the owner's
  machine-level account-switcher command (ADR 0011).
- `CLAUDE.md` is capped at 150 lines, a constant in `post-edit.js`, which also resolves its `@`
  lines (lines 196 to 202); pusher check 7 resolves `@` lines in `CLAUDE.md` and the handoff files
  and caps the handoff, not `CLAUDE.md`; the skills' `wc -l` is the only other check of the 150.
- Git 2.56.0, verified by the critic in two passes: `git apply --3way` merges an append above an
  unchanged marker line with an append below it cleanly, and conflicts on two appends at one spot
  (so ADR 0008's "the merge takes both sides' lines" is false without a marker); a base addition
  of a file the project also added is clean when identical and an `AA` conflict otherwise; an
  already-applied change is a no-op; a divergence on another line merges; a rename of a
  project-edited file applies cleanly when the diff carries the rename hunk (`-M`) and fails
  whole under `--no-renames`. It fails whole, writing nothing, when any hunk cannot be placed: a
  deletion onto a modified file, a deletion or an edit of a file the project deleted (`does not
  exist in index`), a base-owned file with uncommitted changes (`does not match index`); exit 1
  is also what a conflict returns, so only an empty `git status --porcelain` tells a whole
  failure from a conflict. `:(exclude)` drops a path from the diff. An unquoted `$(...)` that
  carries a glob is expanded by the shell against the project's tree and silently drops the base's
  new files; a `:(glob)` prefix is shell-inert. In Git Bash on Windows `git show hatchery/main:<path>`
  is rewritten by MSYS to `hatchery\main;<path>` and fails with `ambiguous argument`;
  `git show <sha>:<path>` works. `merge=ours` needs `git config merge.ours.driver true` per clone
  and falls back to a plain conflict silently. The guard allows `git diff ... | git apply --3way`,
  `git checkout <ref> -- <path>` and a `for` loop over paths for every caller, denies `git merge`
  to `coder` and `tester`, and denies any `xargs git checkout` to everyone (rule 19).
- The siblings, 2026-10-17: `../broodforge` has no base line and a `hatchery` remote (merge mode);
  `../regenesis`, `../droplet-management` and `../dystopo` carry a `Hatchery base:` line (patch
  mode). `CLAUDE.md` is 138, 150, 150 and 89 lines there; project-only lines about 1, 76, 23 and
  89. Each has added its own files under `.claude/` (`rules/go.md`, `rules/kotlin.md`, a `wave`
  skill). `../regenesis` has edited 13 base-owned files against its recorded base, `../dystopo` 11,
  some ahead of the base (the split handoff before ADR 0014); `../broodforge` differs from its
  merge-base (48a43e4) in seven base-owned files under `.claude/` (`pusher.md`, `git-guard.js`,
  `shell-parse.js`, `test-hooks.js`, `rules/README.md`, the `adr` and `brief` skills) and four
  under `docs/` (`decisions/README.md` +21, `claude-code-on-windows.md` +28, `0002-hook-specs.md`
  +6, `docs/design/README.md` +46; critic objection 16). ADR numbers: Hatchery has used 0001,
  0002, 0004 and 0006 to 0015, holds 0003 and 0005 as proposals, and this is 0016, so three
  numbers remain below ADR 0008's `0020`; `../broodforge` holds 0020 to 0023, `../dystopo` 0020
  to 0035, `../regenesis` 0003 to 0018 of its own (ADR 0009), `../droplet-management` 0003.
- Hatchery's records are cited by path (`docs/design/decisions/NNNN-...`) in living files: about
  20 lines in `.claude/` (hook headers, `POLICY_DOC` in `usage-gate.js`, the `adopt-hatchery`
  skill and its template, the harness), 4 in `CLAUDE.md`, 6 in `README.md`, the "Decisions so
  far" list (one row per record) in `docs/design/README.md`, one row in `docs/README.md`, the
  promoted rows and the template line in `docs/design/proposals/README.md`, the Windows-reference
  pointer in `.claude/rules/docs.md`; closed records (ADRs, findings, ideas) cite them in about
  25 more lines. In the siblings' project-owned files, outside their copies of Hatchery's records:
  `../broodforge` 3 (`CLAUDE.md` 2, `README.md` 1), `../droplet-management` 2 (`CLAUDE.md`, its
  adoption ADR), `../dystopo` 1 (a closed record), `../regenesis` 0 (its `0005` and `0010` to
  `0013` citations name its own ADRs). Counted 2026-10-17 with
  `grep -rnE 'docs/design/decisions/(000[1-9]|001[0-5])-'`.
- Ruled out before this proposal: frequent merges and `git rerere` alone; a `default/` directory
  of copies the project then edits in place (the same both-sides edit at a new path).

## The ownership map (shared by the options)

Each option decides the mechanism; all need the same map. The map is data, not prose:
`.claude/hooks/ownership.json` (base-owned), four lists of pathspecs: `baseOwned`, `baseExcluded`
(base paths that are the project's: `.claude/hooks/policy.json`, `.claude/settings.local.json`),
`mixed`, and `projectOwned`, which names directories and files one by one, never `docs/**`
(`CLAUDE.md`, `README.md`, `.gitignore`, `.gitattributes`, `.editorconfig`, `docs/README.md`,
`docs/HANDOFF.md`, `docs/handoff`, `docs/ideas`, `docs/operator`, `docs/findings`,
`docs/reference`, `docs/design`). One set for every project: there is no mode and no
decanted-only row (objections 14 to 16; Hatchery's records live in the base-owned subtree
`docs/hatchery/`, option E). The default class for an unlisted path is project-owned: nothing
unclassified arrives. The helper `node .claude/hooks/ownership.js` (Node built-ins only):

- `--command --ref <sha> [--exclude <pathspec>]...` prints the update command as one line with
  literal shas, `-M`, and every pathspec single-quoted and carrying pathspec magic (`:(glob)`,
  `:(exclude,glob)`); never a `$(...)`. The agent runs the line verbatim. The map is read from
  `<sha>` (`git show <sha>:.claude/hooks/ownership.json` through `child_process`, which MSYS does
  not rewrite), `FROM` from the base line in the working tree, else `git merge-base HEAD <sha>`,
  the excludes from `update.exclude` in the working tree's `policy.json` through the tree's
  `policy.js`, plus the `--exclude` arguments.
- `--per-path --ref <sha>` prints one apply line per entry of `git diff --name-status -M <FROM>
  <sha> -- <set>`; an `R` entry gets both of its paths in one pathspec.
- `--report --ref <sha>` prints: changed base paths that no list classifies; changed paths in
  `baseExcluded` and `update.exclude`; for each `@.claude/project/<s>.md` in `CLAUDE.md`, whether
  `.claude/agreement/<s>.md` changed in `FROM..sha`; agreement files with no import line in
  `CLAUDE.md`; `AA` collisions.
- `--check` lists every tracked path that matches no list; the harness runs it in Hatchery only.
- On a malformed map, or a policy its `policy.js` rejects, every flag prints no command, names
  the fix (`node .claude/hooks/policy.js` prints the key; a key from a newer base means bring
  `.claude/hooks/policy.js` in first) and exits 1; it never falls back to `DEFAULTS`.

Harness group `O` in `test-hooks.js` (`F7` is already a case): O1 `--check` is empty on the real
tree, `SKIP` where a `hatchery` remote exists; O2 the printed command is shell-inert (no `*`, `?`
or `[` outside a magic-prefixed, quoted pathspec); O3 a fixture that adds a path and its map
entry in one commit, where the command from the new sha carries the path; O4 a malformed map or
policy prints no command and exits 1. The lines follow the ADR 0013 contract (`PASS O1
ownership`, `FAIL O1 ownership: <path> matches no list in .claude/hooks/ownership.json; lift:
classify it`).

| Class | Paths | Rule |
|---|---|---|
| Base-owned | `.claude/**` as the base ships it, except `baseExcluded` and the mixed files below; `LICENSE`, `LICENSES/`; `docs/hatchery/**`, Hatchery's records as shipped (its ADRs with their index, `reference/claude-code-on-windows.md`, the subtree's README) | Patched three-way from the base line. A project's edit is a departure: kept where the base did not touch the lines; on a conflict the base wins (`git checkout <sha> -- <path>`) unless the path has a row in the project's departure table (then `git checkout HEAD -- <path>`). A project never writes in `docs/hatchery/`; it supersedes one of those records from its own `docs/design/decisions/` by citation. |
| Mixed by design | `.claude/settings.json` (base entries, the project's entries); `.claude/hooks/fitness.json` (`deps` the base's, `skip` the project's) | Patched three-way; the one place a hand-merge is expected. |
| Project-owned, the default | `CLAUDE.md`, `.claude/hooks/policy.json`, `.claude/settings.local.json`, every path under `.claude/` the base does not ship (`rules/<lang>.md`, project skills and agents, `.claude/project/`), `README.md`, `.gitignore`, `.gitattributes`, `.editorconfig`, everything under `docs/` outside `docs/hatchery/`: the handoff, the queues and their READMEs, findings, proposals, reference, the design docs, `docs/README.md`, the project's ADRs, among them the provenance ADR that carries the base line | Never patched. The base's copies are the base's own state and never enter a project. |

The departure table: a living table in the project's provenance ADR (the adoption ADR today; a
decanted project writes `NNNN-decanted-from-hatchery.md` at decant, or at its first update under
this decision, at the next free number in its `decisions/`), one row per base-owned path the
project edits and per agreement section it replaces (objection 17): path, reason, carry-back (an
idea file, or the Hatchery commit that took it). While a row exists the project wins on that path;
once Hatchery has the change, base wins and the row goes. A path the project never wants the base's
version of (its own `LICENSE`, a skill it removed) goes in a new policy key `update.exclude`
(default `[]`, an addition under ADR 0013), which the helper turns into `:(exclude,glob)` entries;
a departure from a hook or a skill is a row, never an exclude, so base changes to it keep arriving
as conflicts the project resolves (objection 27). Promised under ADR 0013 (one row: the map,
carried by `ownership.json`, with the subtree path; readers: the update, `decant` and
`adopt-hatchery` skills, the harness): moving a path a project may have written from
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
GPL-3.0-or-later as the base default, the decisions directory is "the tree's (`adr` skill, step
1)". Routing is base text since the models are pinned in base-owned agent files.
`this-agreement.md` says the files are edited in Hatchery only, and that in a project a fix to a
base-owned file is a departure: a table row and a carry-back idea, same sitting.

Project file. `CLAUDE.md` becomes the project's: title, a provenance line, `@docs/HANDOFF.md`,
`## Phase`, `Branch:`, nine lines `@.claude/agreement/<section>.md`, `## Project rules`; at most 40
lines (decided under Recommendation), and what it loads at most 190: `CLAUDE.md` plus every file
it imports at any hop, except `docs/HANDOFF.md` and its chain, which has its own 120 (objection
19; the agreement files are 150 of the 190, a Hatchery harness case `A1`, `SKIP` where a
`hatchery` remote exists). `edit.capExempt` naming `CLAUDE.md` by exact name (a pattern never
matches it) lifts the 40 and the 190 together, as it lifts the 300. A project departs from a
section by replacing its import line with `@.claude/project/<section>.md`, a project-owned file
(the base never ships that directory; nothing under `.claude/` loads on its own except `rules/`)
that states what it replaces, naming the replaced file in backticks so the loader does not import
it (objection 18), and names the project's ADR; one line is enough. The model then reads one rule,
not two. Every replacement is a departure row, so the update report says when the base changed the
section it replaces (objection 17). One parser, `.claude/hooks/imports.js`, follows the loader's
rule as documented: an `@` at line start or after whitespace, outside code spans and fenced
blocks, followed to four hops, each hop relative to its file. `post-edit.js` calls it at the edit
and `node .claude/hooks/policy.js --claude-check`, a new line in check 7, calls it at the push;
both print the same lines: `FAIL CLAUDE.md: section <s> neither imported nor replaced`, `imported
twice`, `FAIL .claude/project/<s>.md replaces no agreement section`, `FAIL CLAUDE.md: <n> lines,
cap 40` and `loads <n> lines, cap 190`, each ending `; lift: edit.capExempt naming CLAUDE.md`
where a cap applies, and the unresolved-import line check 7 prints today. Overflow from the 40:
facts to `docs/reference/`, path-scoped rules to `.claude/rules/<name>.md` with `paths:`,
procedures to a project skill, every-turn rules to `## Project rules` or a replacement section.
Renaming, splitting or removing an agreement section is an incompatible change under the ADR
0013 row, and its migration names the replacement files it orphans. `.claude/rules/docs.md`
loads for `.claude/agreement/**` and `.claude/project/**` too (objection 28).

Placement. `.claude/CLAUDE.md` was weighed: load order is the loader's, the project file does not
show the agreement exists, `/init` writes `./CLAUDE.md` unaware of it, neither the hook nor check 7
sees it. An unscoped rules file is the same mechanism and `.claude/rules/README.md` forbids it. The
`@` import is documented, relative, one hop, ordered by the project file, resolved by the checks
that exist, and is the finding's one-line hook (GitLab EE's prepend, Brave's `chromium_src`).

Debug interface. `docs/hatchery/reference/agreement-loading.md` (verified, observed): the critic's
sentinel procedure (`claude -p` with a sentinel word per agreement file, main session and a
subagent, then `--resume`) and the compaction check (interactive `/compact` with an
`InstructionsLoaded` hook logging the `include` and `compact` reasons; a one-time instruction, not
a shipped hook). The sentinel procedure is also the check that `imports.js` matches the loader: a
file the loader imports and the parser misses, or the reverse, is a parser fix. The compaction
check runs before the ADR; a fail moves the agreement to `.claude/CLAUDE.md` with the hook and
check 7 pointed at it.

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
switcher. The pool's home is decided under Recommendation.

Meets the constraints: no conflict on either file; the base's own state is removed after the fact,
by a denylist. Risks: the denylist fails open (a base path outside the map lands in every decanted
project); three restore steps with written-out lists; a runner-only merge; two code paths, each
exercised by one project. Who else: Django `global_settings` plus a project file that "doesn't
have to define any settings"; GitLab's one-line prepend (GitLab then merged its repositories, which
is precedent for E as much as for A); Debian conffiles. Exit cost: paste the agreement back into
each `CLAUDE.md`; one update per sibling; the map stays useful. Moving parts: nine agreement files,
nine import lines per project, the project template in `decant`, two caps and a path rule in
`policy.js`, `imports.js` and the `--claude-check` line, the reference entry, `ownership.json`
and its helper, the departure table, `update.exclude`, three restore steps, one exclude, the
marker in `docs/design/README.md`, amendments to ADRs 0007, 0008, 0013.

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

Mechanism. `update-from-hatchery` has one step 4 for every project: the line `ownership.js
--command --ref <sha>` prints, `git diff -M <FROM> <sha> -- '.claude' ':(exclude).claude/hooks/policy.json'
'LICENSE' 'LICENSES' 'docs/hatchery' <update.exclude entries> | git apply --3way`, run verbatim by
the agent running the skill (allowed for every caller; no runner-only step, no merge commit).
`<sha>` is `git rev-parse hatchery/main` after the fetch; the helper and the map are read from it,
not from the project's tree, so a base window that adds a path and its map entry together
delivers the path (objection 20; on a tree whose `.claude/hooks/` predates the helper, the skill
first writes `git show <sha>:.claude/hooks/ownership.js` to the scratch directory and runs it from
there). `FROM` is the base line (ADR 0010); a project decanted before this decision has none and
uses `git merge-base HEAD <sha>` once, and the same update writes its provenance ADR with the line
and an empty departure table, so every later update is a patch like every other; a `decant` after
this decision writes the line at decant. That is the whole mode test: a line, else a merge-base
once, else the tree is not a Hatchery project and the skill names the adoption prompt; the
pathspec is the same for every tree, so the line decides `FROM` and nothing else (objection 14).
ADR 0004's patch mode becomes the only mode; its path list grows from `.claude CLAUDE.md
LICENSES` to the map's base-owned set and loses `CLAUDE.md`.

Failures. `git apply --3way` exits 1 for a conflict (markers in the file, `U` or `AA` in `git
status`) and for a whole failure (nothing written, `git status --porcelain` empty): a deletion
onto a modified file, a deletion or an edit of a file the project deleted, a dirty base-owned file
(objection 26). A conflict is resolved by the map's rule (`git checkout <sha> -- <path>` or `git
checkout HEAD -- <path>` for a departure row, then `git add`); a whole failure sends the skill per
path: it runs the lines `--per-path --ref <sha>` prints, one per `git diff --name-status -M`
entry, an `R` entry with both its paths so a renamed file arrives as a rename and not as a second
copy without the project's edit (objection 23), and reports each line that failed with its
disposition: `does not exist in index` on a `D` entry is already done; on an `M` entry (a base
edit to a file the project deleted) the project adds the path to `update.exclude` or takes the
base's file with `git checkout <sha> -- <path>`; `does not match index` means step 1's clean-tree
precondition was skipped: stop, commit or stash, start over. The base line is bumped last, and
only when the report gives every failed path a disposition: excluded, taken, or already done
(objection 25); a path without one would drop out of every later diff. The report (`--report`)
lists base changes outside the set that no list classifies (a new base path is seen, not leaked),
changes to excluded paths (a guard fix to an excluded hook is seen), base changes to every
replaced agreement section, agreement files with no import line, and `AA` collisions (objection
27). `update.exclude` is written only after `node .claude/hooks/policy.js` prints the policy with
the incoming `policy.js` in the tree; until then the excludes a first update needs are `--exclude`
arguments to the helper (objection 22, the order under Recommendation). The agreement half of A
combines unchanged.

Hatchery's records. Hatchery's ADRs, their index and the Windows reference move to a base-owned
subtree at the same path in every tree, Hatchery included: `docs/hatchery/README.md` (what the
subtree is, one line per direct entry), `docs/hatchery/decisions/NNNN-*.md` with
`decisions/README.md` as their index (the "Decisions so far" list moves there; the
`decisions/README.md` warning in the handoff becomes true, since the project directory then lists
only project records), `docs/hatchery/reference/claude-code-on-windows.md` with its README. `docs/`
is otherwise wholly project-owned. Hatchery numbers its records in `docs/hatchery/decisions/` from
`0016` up with no ceiling; every project numbers its own in `docs/design/decisions/` from `0001`,
or the next free number in its tree, proposals included; the two never share a directory, so no
number collides and no range is needed (objection 15). The `adr` skill's step 1 keeps one test:
Hatchery itself (no `hatchery` remote) writes in `docs/hatchery/decisions/`; every other tree
writes in `docs/design/decisions/`; the `triage` and `handoff` scans read the same directory.
Hatchery's proposals, findings, ideas and its other reference files stay where they are as
Hatchery's own state and never ship (decant deletes them, as today); a proposal's Status line
cites its ADR by path. A project supersedes a Hatchery record from its own `decisions/` by citing
it (`supersedes docs/hatchery/decisions/0001-licensing.md for this project`); the record itself
is not edited, and ADR 0001's living components table becomes a table in the project's own ADR.
Every project, adopted ones included, then holds the records its `.claude/` cites, which closes
ADR 0009's "a base ADR number cited in `.claude/` can name a different file" and the open idea on
adopt step 7's dangling pointers. The move in Hatchery: `git mv` of the records (relative links
between them, `](0009-...)`, survive), the index rows, the citations in living files listed under
Constraints (about 30 lines, one pass; `policy.js` `anchorExempt` gains `docs/hatchery/*` as a
default, an addition); closed records keep their text (`anchorExempt`). Shared history for base
files ends at the switch: `git log -- .claude/hooks/git-guard.js` in the project shows `Update
from Hatchery <a>..<b>` commits whose bodies list the incoming base commits (step 3's list, as
today), `git blame` points at them, and the base's history is `git log hatchery/main -- <path>`,
one hop, since the remote is fetched for every update. Three of four siblings live this way already.

Meets the constraints by construction: Hatchery's `CLAUDE.md`, policy, handoff, queues, findings,
proposals and index rows never enter any project, so there is nothing to restore, remove or
row-drop, and a path outside the list cannot leak. One code path, exercised by every project on
every update. Costs: shared history for base files ends at the switch (above); the whole-apply
failure, mitigated per path; the citation rewrite and the duplicate removal at each sibling's
first update (Recommendation); the base line is bumped by hand; the owner's 2026-10-12 sentence
"merging stays the path for projects that are based on Hatchery" (ADR 0004) is superseded, which
the coordinator reports. Who else: Debian quilt and gbp (the finding: upstream untouched, the
delta apart, cost "the lack of history"), cruft and copier (template updates by `git apply -3`),
the three adopted siblings since 2026-10-12. Exit cost: low; histories are still joined, so `git
merge hatchery/main` works again at any time, and a first merge after a patch period is clean
where both sides hold the same content; the subtree is a `git mv` back. Moving parts: the
agreement parts of A; `ownership.json`, its helper with four flags and group `O`; `imports.js`
and `--claude-check`; the departure table and `update.exclude`; one standing exclude; the
provenance ADR template (the adoption template generalised); `docs/hatchery/` with three indexes
and the citation rewrite; the decisions-directory rule in the `adr`, `triage` and `handoff`
skills; amendments to ADRs 0002 (layout), 0004, 0006 (the `capExempt` row), 0007, 0008, 0009,
0010, 0013.

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

Revised after the second pass (objections 14 to 28): the records' home changes, the three
decisions below do not. The first revision shipped Hatchery's records to decanted projects only,
through rows that needed a mode the tree stops recording once every project carries a base line,
a number range with three numbers left, two files mixed in practice, a marker and two numbering
amendments. This revision gives every project one base-owned set with Hatchery's records in
`docs/hatchery/` (E, Hatchery's records), weighed against keeping two sets with the mode written
on the base line (`Hatchery base: <sha> decanted`, an addition the parsers tolerate): the word
fixes 14 and nothing else; the range still runs out, widening it collides with `../broodforge`'s
0020 to 0023 and every `0020-decanted-from-hatchery.md`, a disjoint range needs a ceiling on
projects or four-digit jumps, adopted projects still lack the records their `.claude/` cites, and
`decisions/README.md` stays mixed. The subtree's cost is the citation rewrite under Constraints,
about 30 living-doc lines in Hatchery in one pass and 1 to 3 per sibling, most in a `CLAUDE.md`
the migration rewrites anyway, plus the removal of each sibling's copies of the records. Not
taken: shipping nothing and citing by `<sha>:<path>`, because the deny texts and `POLICY_DOC`
point at files an agent must open, and every living-doc citation would need `[no-check]`.

Decided 2026-10-17. The owner takes the coordinator's recommendation on each question where a design
agent's own research agrees; the architect checked the three against the siblings (read only) and
agreed with each:

- E is taken: merging ends for decanted projects, and the owner's 2026-10-12 sentence in ADR 0004
  ("merging stays the path for projects that are based on Hatchery") is superseded. Evidence: in
  `../broodforge`, the one merge-mode sibling, all three merges from `hatchery/main` were resolved
  by hand and each removed the base's own state (their commit bodies); its edits to base-owned
  files (seven under `.claude/`, four under `docs/`; Constraints) never reached Hatchery, and no
  Hatchery commit carries a cherry-pick from it, so the shared DAG has carried nothing in either
  direction; the carry-backs Hatchery did take came from patch-mode siblings by hand. What E costs
  it: after the switch, `blame` on a base file names an update commit, and Hatchery's own commit is
  one hop away in `git log hatchery/main -- <path>`.
- The project file's cap is 40 lines, lifted by `edit.capExempt` naming `CLAUDE.md`. The lift is
  new mechanism, not a reading of today's hooks: the 150 is a constant in `post-edit.js`, checked
  by exact path and consulting no policy; `docCap` in `policy.js` covers `docs/` paths only; check
  7's list is `docs/*.md`, so `CLAUDE.md` has no pusher cap at all today. The coder brief moves the
  cap into `policy.js` beside `docCap`, where `capExempt` already applies, and the same name lifts
  the loaded total of 190 (objection 19). Overflow goes where A says and never to a free `@`
  import: a departed section is a replacement file; facts go to `docs/reference/`, path-scoped
  rules to `.claude/rules/`, procedures to a skill. A replacement file is read by the loader every
  turn, counted in the 190, listed as a departure row, and named in the update report when the
  base changes the section it replaces. Migration as read on 2026-10-17: `../regenesis` replaces
  `git.md` and `licensing.md` (its Git and Licensing sections), moves its hooks paragraph to
  `docs/reference/`, keeps its two-machines and live-data rules as project rules (about 50 lines,
  then a trim or the lift); `../dystopo` carries none of the base's rule text, so most of its
  sections become replacement files and its Rules section stays; the lift is for a project that
  refuses the rewrite.
- The switcher command stays per project in `policy.json`, unchanged. The only reason to move it
  was the base policy deletion, withdrawn under objection 4. Cost today on this machine: one line,
  present in Hatchery and `../droplet-management` (where it arrived through the policy patch this
  proposal stops) and absent from the other three siblings, each a one-line edit to a file the
  update never touches. A per-user file is its own proposal, with the `null`-meaning migration
  stated, if that edit keeps recurring.

First step: the compaction check (A, Debug interface). Then the ADR (0016, the first record
written in `docs/hatchery/decisions/`), superseding 0008 (Numbers; What a decant keeps) and 0009
and 0010 in part (the line decides `FROM`; nothing is left to tie-break), amending 0002 (the
layout gains `docs/hatchery/`), 0004 (one mode; the skeleton copies no record by name, the
subtree arrives with the set), 0006 (the `capExempt` row), 0007, and adding the map, the agreement
imports and the `Branch:` line to 0013; no promised surface changes incompatibly (the base line's
format, the handoff paths and the policy keys are untouched; `update.exclude` is an addition).
Then one coder brief: `ownership.json`, `ownership.js`, group `O`, `imports.js`, the caps and
path rule in `policy.js` with `--claude-check`, the `post-edit.js` lines, `update.exclude`, the
guard's malformed-policy message naming a newer base's key as a cause and the Edit-tool lift
(remove the key, bring `policy.js` in, put it back; text, not promised), `anchorExempt`'s new
default, case `A1`; one tester brief: the skills (`update-from-hatchery` around the one command
and the order below, with step 1's precondition being the relation and not where the skill was
read from; `decant` 4, 5 and 9 with the `CLAUDE.md` template, the provenance ADR and the empty
project `decisions/`; `adopt-hatchery` 3, 6, 7 and 10 without the routing slot, with
`docs/hatchery` in the checkout and no record copied by name; the provenance template; the
decisions-directory rule in `adr`, `triage` and `handoff`; check 7's line; every `git show
hatchery/main:<path>` in a skill, a template or an ADR's living line rewritten `<sha>:<path>`
after a `finding` records the MSYS rewrite), and Hatchery's own migration (`.claude/agreement/`,
the short `CLAUDE.md`, the `git mv` of the records with the indexes and the citation pass, the
reference entry, `docs/README.md`'s `hatchery/` row). Then each sibling at its next update, in
this order, since every sibling runs an older `policy.js` and an older skill: (1) `S=$(git
rev-parse hatchery/main)`; read the skill with `git show $S:.claude/skills/update-from-hatchery/SKILL.md`
and the helper with `git show $S:.claude/hooks/ownership.js > <scratch>/ownership.js`; (2) the
printed command, with `--exclude` for any path the project knows it removed; per path on a whole
failure; conflicts by the map's rule; (3) `node .claude/hooks/policy.js` prints the policy with
the new `policy.js` now in the tree; (4) the migration edits in the same commit: `CLAUDE.md` to
the project shape (check 7's section line fails until it is), the departure table from its
base-owned edits and replaced sections, the copies of Hatchery's records removed from
`docs/design/decisions/` and `docs/reference/` (a project-edited copy's `git diff <FROM> HEAD --
<path>` goes to a carry-back idea; `../broodforge` has four such), their rows dropped from
`docs/design/README.md` and `decisions/README.md`, a `hatchery/` row in `docs/README.md`, the
project-owned citations rewritten (`grep -rnE 'docs/design/decisions/(000[1-9]|001[0-9])-'
--exclude-dir=.claude`, read against the project's own numbers); (5) `update.exclude` written
now, never before (3); (6) the base line bumped, with every failed path dispositioned;
`../broodforge` writes its provenance ADR (`0024-decanted-from-hatchery.md`) with the merge-base;
(7) check 7's lines, commit.

## What would change this recommendation

- An in-tree `@` import needs approval, does not resolve, or does not re-expand after `/compact`:
  take `.claude/CLAUDE.md` with the hook and check 7 pointed at it.
- The owner wants Hatchery's commits in a decanted project's `blame`, or a decanted sibling carries
  a base-file fix back through the DAG (a cherry-pick Hatchery takes, which none has done so far):
  A's merge mode as corrected.
- A project needs to override one sentence of a section, not the section: the agreement gains a
  parameter (a policy key) for that rule; the replacement file is the stopgap.
- `git apply --3way` fails whole on most updates in practice: per-path application becomes the
  default.
- A project routes its overflow through an `@` import that replaces no section: the 190 already
  counts it; if the per-file 40 then stops mattering, drop it and keep the loaded total.
- Projects edit `docs/hatchery/` routinely (departure rows on records): the records stop shipping
  and `.claude/` cites them by `<sha>:<path>`, the critic's weaker option.
- The pool key is being added by hand at every decant and adoption on one machine, or the switcher
  ships a hook surface of its own (ADR 0011's supersede trigger): the per-user file gets its
  proposal; a `decant` step 5 default that reads the machine is the cheaper fix to try first.

## Claims to verify

Verified by the critic (its tables, first pass claims 1 to 5, 7 to 9, 11 to 13; second pass C1 to
C14) and this revision: the loader concatenates both project files; the import loads for the main
session and a subagent, needs no approval, is re-read on `--resume`; `policy.js` ENOENT gives
`DEFAULTS` and an unknown key makes the guard deny every git command; `post-edit.js` and check 7
handle `CLAUDE.md` by exact path and check 7 does not cap it; the guard verdicts above, including
E's commands for every caller; the marker merge under `git merge-file` and `git apply --3way`;
the whole failures (deletion onto a modified file, both sides deleting, a base edit to a deleted
file, a dirty file) and `:(exclude)`; the `AA` conflict and the clean identical addition; the
rename under `-M` and the stranded copy under the per-path loop without it; unquoted globbing;
the old map omitting a path the new one carries; the MSYS rewrite of `ref:path`; mid-line
imports in the loader docs; the pool code path; the sibling facts and counts. Open: import
re-expansion after `/compact` (the check before the ADR); the loader's exact token rule for a
mid-line `@` (the sentinel procedure settles it; the parser follows); `git checkout HEAD --
<path>` and `git checkout <sha> -- <path>` clearing an unmerged entry left by `git apply --3way`
(the first pass saw `checkout <tree> --` clear a merge's unmerged entries; O-group fixture to
confirm for `apply`); `merge=ours` under `git merge` (B only).

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
   switcher's home was the owner's question; decided under Recommendation (per project, unchanged).
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
   `edit.capExempt` is the lift. The number was the owner's question; decided under Recommendation (40).
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

Second pass (recommendation E). The responses to 1 to 13 stand as written; where 15 and 16
retire something they recorded (the decanted-only row of 1, the marker of 3, the harness case
named F7 in 2 and 9, now group `O`), the body above is the current design.

14. Once every project carries a base line, the helper cannot tell decanted from adopted, so the
    decanted-only rows stop applying after one update. Accepted: there is no mode. One base-owned
    set for every tree (15), so the helper has no mode branch and nothing stops arriving; the base
    line decides `FROM` only, `git merge-base` is used once by a project decanted before this
    decision, and that update writes the line. The `decanted` word on the line was weighed and not
    added: it would fix this objection and nothing else (Recommendation). The numbering amendment
    is withdrawn with the cause (15).
15. Hatchery has three ADR numbers left, and E builds `0020` into every project. Accepted with the
    critic's strongest option: Hatchery's records in the base-owned subtree `docs/hatchery/`
    (`README.md`, `decisions/` with its index, `reference/claude-code-on-windows.md` with its
    README), the same path in every tree, Hatchery included, so Hatchery numbers from `0016` up
    with no ceiling and a project numbers from `0001` or the next free number in its own
    `decisions/`; the range, the marker, the `0020` start and the provenance file's fixed number
    retire. What moves, every citation and its cost, and ADR 0008's new rule are under E
    (Hatchery's records), Constraints and Recommendation; ADR 0008 is superseded (Numbers; What a
    decant keeps: the subtree is in the tree, `docs/design/decisions/` is empty but its README,
    every other Hatchery record is deleted as today), ADRs 0009 and 0010 in part (the line's roles
    are `FROM` and the departure table; nothing is left to tie-break). The disjoint range was
    weighed and rejected: it needs a ceiling on projects or four-digit jumps, still collides at
    the widening, and leaves adopted projects without the records their `.claude/` cites; not
    shipping was rejected under Recommendation.
16. `decisions/README.md` and the Windows reference are mixed in practice, not base-owned.
    Accepted: both move into the subtree, where a project does not write; the project's
    `decisions/README.md` and its `docs/design/README.md` list the project's records only, the
    subtree's indexes list Hatchery's; a project's additions to the Windows facts go to its own
    `docs/reference/` file or a carry-back idea; a project supersedes a record by citation from its
    own ADR, so no project line ever lands in a base record. The count is corrected (seven files
    under `.claude/`, four under `docs/`; Constraints and Recommendation).
17. Replacement files are invisible to the update, and base changes to sections bring back the
    two-rule case. Accepted, all four: a replacement is a departure row; `--report` lists base
    changes to every replaced section and every agreement file with no import line; `--claude-check`
    and the hook fail `.claude/project/<s>.md` with no `.claude/agreement/<s>.md`; renaming,
    splitting or removing a section is an incompatible change under the 0013 row whose migration
    names the replacement files it orphans (A, Project file).
18. Check 7's exactly-one count sees line-start imports only; the loader also imports mid-line and
    inside imported files. Accepted: `.claude/hooks/imports.js`, one parser following the loader's
    documented rule (an `@` at line start or after whitespace, outside code spans and fences, four
    hops, each relative to its file), called by `post-edit.js` and by `policy.js --claude-check`
    (check 7's new line); the count covers every file `CLAUDE.md` loads; the replacement template
    names the replaced file in backticks; the sentinel procedure is the check that the parser
    matches the loader, and a mismatch is a parser fix (A, Debug interface).
19. The 40 caps a file, not what loads, and a replacement file is uncapped. Accepted: the loaded
    total (`CLAUDE.md` plus every import at any hop, except the handoff chain with its own 120) is
    capped at 190, 40 plus the agreement's 150, resolved by the parser; `edit.capExempt` naming
    `CLAUDE.md` lifts the 40 and the 190 together; the agreement's own 150 is Hatchery harness case
    `A1`, skipped where a `hatchery` remote exists (28).
20. Which map computes the pathspec is unspecified, and the wrong one loses files for ever.
    Accepted: `--ref <sha>`; the helper reads the map from the sha, and on a tree without the
    helper the skill extracts it from the sha into the scratch directory first; O3 is the harness
    case (E, Mechanism).
21. Unquoted `$(...)` globbing drops new base files without an error. Accepted: `--command` prints
    the line with literal shas and quoted, magic-prefixed pathspecs, the agent runs it verbatim;
    `--paths` is dropped so there is nothing to substitute; O2 checks the output is shell-inert.
22. `update.exclude` denies every git command in a project whose `policy.js` predates it, and the
    first update is exactly that case. Accepted, with the migration order: (1) the apply, whole or
    per path, brings `.claude/hooks/policy.js` with the rest of `.claude/`; the excludes the first
    apply needs are `--exclude` arguments to the helper, not a policy key; (2) `node
    .claude/hooks/policy.js` prints the policy; (3) only then does the skill write `update.exclude`;
    (4) the helper refuses on a malformed policy or map, exit 1 with the fix named, never
    `DEFAULTS`; (5) the guard's message gains "a key from a newer base is one cause" and the lift
    that works while git is denied: remove the key with the Edit tool, `git checkout <sha> --
    .claude/hooks/policy.js`, put the key back. The siblings all run the older `policy.js`, so the
    skill's first-update steps are this order (Recommendation).
23. The per-path fallback silently strands a renamed file. Accepted: `-M` explicit in the command
    and in `--per-path`, which iterates `git diff --name-status -M` and gives an `R` entry both of
    its paths in one pathspec.
24. A migration command fails on the target platform. Accepted: `<sha>:<path>` after `git
    rev-parse` everywhere this proposal writes (the skill, the migration, `adopt-hatchery` and its
    template, the living lines of ADRs 0002, 0004, 0006 and 0009); a `finding` first, then the
    reference entry (`docs/hatchery/reference/claude-code-on-windows.md`, Shells); step 1's
    precondition becomes the relation (a line, a merge-base, or the adoption prompt), not where
    the skill was read from.
25. Bumping the base line after a partial apply silently gives up the failed paths. Accepted: the
    bump is the last step and happens only when the report gives every failed path a disposition:
    excluded, taken with `git checkout <sha> -- <path>`, or already done (E, Failures).
26. The whole failures are not the two the proposal names. Accepted: the skill says exit 1 with an
    empty `git status --porcelain` is a whole failure, so go per path; a per-path `does not exist in
    index` on a `D` entry is already done, on an `M` entry the project excludes or takes; `does not
    match index` is a dirty tree, which step 1 forbids: stop, commit or stash (E, Failures).
27. The outside-the-set report is mostly noise, and F7's reach is undefined. Accepted: `--report`
    lists paths that no list classifies and changed excluded paths, not every base change;
    `projectOwned` names directories and files one by one, never `docs/**`; the classification
    check (O1, renamed from F7 since F7 exists) runs only where there is no `hatchery` remote;
    `update.exclude` is for files the project removed, and a departure from a hook stays a table row.
28. Lifts and messages left unstated. Accepted, one line each: ADR 0006's `edit.capExempt` row
    becomes "paths exempt from the docs caps; `CLAUDE.md` by exact name also exempts the project
    file's 40 and the loaded 190", so a broad pattern never lifts it unmeant; every `FAIL` line
    from `--cap-check`, `--claude-check` and the post-edit message names the lift
    (`edit.capExempt`); the agreement's 150 together is harness case `A1` in Hatchery, skipped
    where a `hatchery` remote exists, lifted by an ADR amendment; `.claude/rules/docs.md` `paths:`
    gains `.claude/agreement/**` and `.claude/project/**`.
