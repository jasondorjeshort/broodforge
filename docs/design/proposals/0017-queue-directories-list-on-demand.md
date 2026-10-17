# 0017. Queue directories list on demand: no hand-kept index in their READMEs

Date: 2026-10-17
Status: promoted to ADR 0017

Revised 2026-10-17 after the critic review (`0017-critic-review.md`): scope narrowed to the three
directories the owner named, the switch moved from a policy key to a line in the directory's own
README, an edit-time check kept, the listing promised, the dystopo misreadings and the factual
errors fixed. The responses are the last section; the body below is the corrected record.

## Question

Three directories under `docs/` hold one file per item and are queues in the sense of
`.claude/rules/docs.md`: `ideas/`, `operator/` and `findings/`. Each README ends in a hand-kept
table with one row per file, and the rule file, the post-edit hook and four skills read or write
those rows. The owner's words: in these three "we most likely should not put an index of the files
into the readme. The entire point of using files for this is to not have conflicts on that one
file constantly"; `reference/` and `design/` "serve as a documentation file" and may differ.
Decide whether the three keep a per-file index, and if not, how a reader, a skill and a sibling
project find items and their status. The first draft also took `design/proposals/` as a fourth
queue on its row-only count; objection 2 showed its rows are curated summaries a listing cannot
print, and it stays a written index (see Options, last subsection). The evidence is
`docs/findings/2026-10-17-queue-directory-indexes.md` (cited below as "the finding"): the
practice of nine one-file-per-item tools, a merge experiment and the counts for Hatchery and two
siblings. In ten years the chosen shape should let any number of sessions add, resolve and delete
items in parallel with no file they all edit, and let an agent answer "what is open" with one
command whose output it can parse.

## Constraints that matter here

- Agents write everything; a listing must run from the Bash and PowerShell tools on Windows. Git's
  `grep.exe` is on the PowerShell tool's path on this machine (the critic checked), so the case
  for a Node script is not grep's absence: it is `--json` and one tested reading of the block
  grammar instead of a grep each skill restates. Node is already required by every hook, and ADR
  0013's fitness check `F` holds `.claude/hooks/*.js` to Node built-ins.
- The debug interface rule: whatever replaces the rows must be drivable from a terminal and
  parseable (`--json`), and tested in `test-hooks.js` like the hooks.
- ADR 0016 (accepted 2026-10-17): `docs/` is project-owned except the base-owned subtree
  `docs/hatchery/`; updates are one patch from the base-owned allowlist, for decanted and adopted
  projects alike, and the queue READMEs never move with an update (item 9 hands this question to
  0017). So the three READMEs are each project's own file, base-update conflicts on them are over
  with or without an index, and `.claude/` (hook, script, skills) reaches every sibling.
- ADR 0013: an addition is free only when its default preserves old behaviour; "a file or line
  written against an older base loads under a newer one with the same meaning". The policy keys
  are promised; the findings and operator frontmatter are "not promised, trigger: the first
  program that reads them". The idea file grammar is promised; the ideas README's index is not
  part of it. A surface written into project-owned files that the base can never patch must be
  named in 0013's table (objection 5).
- `.claude/rules/docs.md`: an index is the one list a doc may carry; prose never inventories files.
  The status of a finding (`status:`) and the kind and block of an idea live in the item; the row
  repeats them. In the other direction the item can be wrong where the row is right (proposals
  0003 and 0005 say `Status: draft` after a critic review, objection 3), so "the row is always a
  second copy" was too strong: either copy can drift, and one copy with a check at the edit beats
  two with none.
- Already ruled out (the finding, section 4): `merge=union` and marker lines for the queues (union
  resurrects deleted rows and imports the base's rows; GitHub and Bitbucket do not honour it), and
  a generated index that is still committed (the same shared file, resolved by regenerating).

### What the histories show

Hatchery, 101 commits (the finding, section 5): commits to the ideas README changed rows only in
16 of 17, findings 13 of 14; `docs/design/README.md` changed prose in 7 of 19 and
`reference/README.md` in 2 of 5. Files added and deleted: ideas 39 and 7, findings 13 and 0,
reference 2 and 0, operator none.

`../regenesis`, read 2026-10-17 (`git -C ../regenesis log -- docs/findings/README.md docs/ideas/README.md`,
recounted by the critic): 40 commits to the findings README, 37 row-only; 20 to the ideas README,
17 row-only. The only merges that touched either are its two base updates (e4d61c7e created both
tables from the base's rows; 8745c5d1 created the ideas and operator READMEs); its 45 work merges
never conflicted on them (the finding's replay), because its sessions have run one at a time on
these files. Two later commits (af2b8d27, the handoff split, and 62f15f9c, the update to d343b69)
each drop a stale ideas index row naming a file the repository did not have: the hook checks
file-to-row, never row-to-file, so a row outlives its file unnoticed. One finding there
(`2026-10-13-suite-duration.md`) says `status: resolved (ADR 0011, 2026-10-14)` while its row says
`open`; the resolving commit d2a28872 left the README alone. The owner adds that findings and
ideas are underused there (18 ideas, 28 findings), so the counts understate the churn a used
queue would show. The conflicts seen so far are base-against-project, which 0016 ends; the
parallel-session conflict is the shape the finding's experiment shows (two appends at the end of a
dated table conflict by default, and so does a delete against an append) and has not yet been
paid because no sibling has run two sessions on a queue at once. ADR 0015 chose the split handoff
for exactly that day.

## The listing script (shared by A and B)

`.claude/hooks/queue-list.js`, Node built-ins only (`fs`, `path`), under fitness check `F` like
the other files there, with cases in `test-hooks.js` (the next free group letter) run by the test
command and pusher check 8. It is the agent-usable surface for the three directories, and it
exports its per-file reading (`read(mode, file)`) for the post-edit hook, so the two never
disagree (an internal seam between two base-owned files, not promised).

```
node .claude/hooks/queue-list.js <ideas|operator|findings> [--status <v>] [--untriaged] [--json] [--markdown] [--check]
```

One line per file except `README.md`, sorted by name (the date prefix makes that age order),
tab-separated, the title last (the H1 text; the first non-blank line when there is no H1; `?` for
an empty file):

- `findings`: `<file>  <status>  <title>`; status is the first word of the `status:` frontmatter
  value, one of `open | resolved | superseded` (regenesis's `resolved (ADR 0011, 2026-10-14)` reads
  as `resolved`), else `?`. `--status <v>` keeps the files whose status column is `<v>` or whose
  whole `status:` value is `<v>`.
- `ideas`: `<file>  <kind>  <block>  <title>`; kind is the `kind:` value when it is one of the
  rule file's four, else `?`; block is `plan`, `question`, `answered` (a `Question:` whose
  `Answer:` is not blank), `waits` or `untriaged`, by the reading the `triage` skill makes today
  (step 2), and `?` when the frontmatter carries a key outside the promised grammar (`date`,
  `kind`, `related`): the file speaks a grammar the script does not read, so it is not confidently
  anything. `--untriaged` keeps the files whose block is `untriaged`, never a `?`.
- `operator`: `<file>  <title>`.
- `--json` prints an array of objects with those field names (`file`, `title`, `status`, `kind`,
  `block`; a `?` is the literal string `"?"`); `--markdown` prints the rows in today's table format
  (the exit path, below). For every `?` the script prints one stderr line,
  `queue-list: <file>: <field> is "<value>"; allowed: <set>` (or `<field> missing`). The exit code
  is 0 when the listing printed, `?` rows included; 1 only under `--check` when any row has a `?`;
  2 for a usage error (unknown mode or flag, directory absent). So a tree with files in its own
  grammar (dystopo below) lists on every call, and a skill that wants the verdict asks for it.

Cost: about a hundred lines and a dozen harness cases; a Node start per call (`node -e 0` 0.065 s
here, `grep` 0.034 s). Who calls it: the `handoff` skill (open findings), `triage` (the untriaged
list), `decant` and `adopt-hatchery` (the resting-state check and the split map's Open findings
line), the post-edit hook (one file, through the export), the coordinator and any reader who wants
status without opening files, and the `docs/README.md` map in its Loaded column. It works in
every tree whether or not that tree has dropped its tables: it reads files, never rows.

Promised (ADR 0013, a new row): the path, the three mode names, `--status`, `--untriaged`,
`--json` with those field names and the `?` literal, `--check`, the exit meanings, the first word
of a finding's `status:` as its status, and the README marker line of option A below. Not promised:
the tab-separated text, `--markdown`, the stderr wording, harness ids. Agents parse `--json`.

## Options

### A. No index in the three READMEs; the script lists; the README says so and the hook reads it

What it is. Each of the three READMEs keeps its format rules, loses its `## Index` table and gains
one line, the marker: `Listed on demand: node .claude/hooks/queue-list.js <mode>`, where `<mode>`
is the word after the script path. Status, kind and block are read from the item. Changes:

- Hook (`post-edit.js` lines 92 to 110): before the walk-up, read the edited file's own
  `README.md`; if a line matches `^Listed on demand: node .claude/hooks/queue-list.js (ideas|operator|findings)\b`,
  skip the index check for that file and instead call `read(mode, file)`; when it would print `?`,
  warn `post-edit: <path> would list as ? (<field> is "<value>"; allowed: <set>). See <dir>/README.md.`
  A marker whose mode word is anything else warns on the README edit
  (`post-edit: <readme> names an unknown listing mode <m>`) and the directory counts as indexed. A
  README without the marker: the index check exactly as today, and no parser warning. The
  findings frontmatter key check (lines 84 to 87) stays. No policy key, no schema change: the
  switch lives in the project-owned file it is about (0016's ownership rule), and a sibling's hook
  behaves as before until that sibling edits its README. This is the critic's first choice under
  objection 1; the second (`edit.unindexed` defaulting to `[]`, set by Hatchery, `decant` and
  `adopt-hatchery`) is ruled out below as the more moving parts for the same effect.
- Rule file (`.claude/rules/docs.md`, Indexes): a directory is either indexed (its README names
  every file: `docs/design/`, `docs/design/proposals/`, `docs/reference/`, `docs/hatchery/`) or
  listed on demand (its README carries the marker line and the format, never a file name; the
  hook reads the marker). The line "`docs/findings/README.md` shows status per file" goes; the
  one-list rule stays. Frontmatter: a finding's status is read by its first word.
- `finding` skill: step 3 of Record becomes "if `docs/findings/README.md` carries an index table,
  add its row; otherwise the post-edit warning is the check (the file must list without `?`)";
  step 3 of Resolve and the Supersede sentence are conditional the same way, as `proposal` step 3
  already is. The hook's warning, not the skill text, tells a tree which case it is in.
- `triage` skill: step 2 becomes the `--untriaged` call, and the report names the stderr `?`
  files as not triaged; the Bounds line about index rows becomes "and their rows where the README
  carries a table".
- `decant` skill: steps 7 and 8 delete files only, no table reset (the READMEs keep the marker);
  the split template line (54) and `handoff/split.md` line 15 say
  `Open findings: node .claude/hooks/queue-list.js findings --status open`. `adopt-hatchery` line
  49 copies the base's READMEs only where the project lacks them, so an adopted project's own
  tables and warnings stay; line 53 loses the table reset.
- `handoff` skill step 1: "the Open column of `docs/findings/README.md`" (a column that does not
  exist) becomes the `--status open` call.
- `proposal` skill step 3 and the `adr` skill: no edit (step 3 is already conditional; the `adr`
  skill never edited the proposals row, which closes the first draft's open claim).
  `update-from-hatchery` lines 55 and 65 lose "their `README.md`s are the base's" and "drop their
  index rows" (0016 rewrites that skill anyway).
- `docs/README.md`: each of the three queue rows' Loaded cell names the command beside the README.
- ADRs amended: `0002-repository-layout.md` lines 52 ("a `README.md` index, one line per file"),
  92 (the `PostToolUse` row) and 140 ("costs one index line"); `0002-hook-specs.md` line 193 (the
  index-check row gains the marker skip and a row for the `?` warning); ADR 0006 (the sentence
  "only exemptions are policy": this exemption lives in the README; its key table gains nothing,
  since no key is added); ADR 0007 line 71 ("every idea costs a file and an index line"); ADR 0013
  gains the row above, since the script is the first program that reads the findings frontmatter
  values. ADR 0008 is superseded by 0016 and is not amended.
- Hatchery's own three READMEs lose their tables and gain the marker.

Constraints: no shared file, so parallel sessions conflict on nothing by construction (the
finding's first practice row: towncrier, scriv, reno, changesets, adr-tools, kepctl, git-issue);
one copy of status, checked at the edit by the same parser that lists, so the sibling's drift and
the stale rows cannot recur; the command is the debug interface. What a reader loses: a forge page
shows file names, not status; the date prefix and the H1 slug carry what the one-line column did;
the single-file handoff links the open findings and the split map names the command (objection 8).
Exit cost: `--markdown` prints the rows with the H1 as the one-line text; a tree that re-adds a
table pastes them under `## Index`, rewrites the one-line text it wants, removes the marker, and
the hook's index check returns for that directory; each tree is its own commit under 0016; the
skill lines are already conditional. Moving parts: one script, one marker line the hook greps,
one harness group, the command line in three READMEs and the map, one ADR 0013 row.
Promised surfaces (ADR 0013): the script and the marker, by a new row; the policy file is
untouched; the idea grammar, the base line, the cache, the grant files, the hook seams and the
handoff paths are untouched. Behaviour-preserving for every sibling: a README without the marker
is checked as today.
Sequencing: implemented after 0016's update half lands (both edit `post-edit.js`, `decant`,
`adopt-hatchery`, `update-from-hatchery`, `split.md` and the rule file); no policy key, so 0016's
key-order rule (update procedure, step 4) has nothing to order here.
Sibling migration, each in its own session after its 0016 update brings the hook and script:
- `../regenesis` (patch mode; 18 ideas, 28 findings, 1 operator file; every `status:` and `kind:`
  reads cleanly): drop the three tables, add the three marker lines, one commit; the
  `suite-duration` drift and the two stale rows go with the tables.
- `../broodforge` (14 ideas, 10 findings, 5 operator files; 5 open, 5 resolved): the same three
  READMEs, one commit.
- `../droplet-management` (4 ideas, 3 findings, no operator file; 3 open): the same, smallest.
- `../dystopo` (19 ideas in its own grammar: `kind:` present on all 19, plus `status: new | done`
  and `pipeline:`, which the rule file does not have; 18 findings, 1 open, 17 resolved; no
  `docs/operator/`): the script lists its findings cleanly today. Its ideas print their kind and
  `?` in the block column with a stderr line naming `status` or `pipeline`; `--untriaged` prints
  none of them; exit 0, so `handoff` and `triage` run. Its two critic reviews with
  `Status: review of an accepted ADR ...` are not read by anything here (no proposals mode). Its
  migration: keep the ideas table and its Triage section (no marker, so its hook is unchanged)
  until it chooses, in its own session, between moving to the promised grammar (delete the 12
  `done` files as the queue rule says, `Plan:` blocks for the planned ones, drop the two keys) and
  then adding the marker, or keeping its grammar and table for good and reading the `?` lines as
  "mine". The findings README it may switch at once.

### B. As A, but the findings README keeps its table because of the Status column

What it is. Ideas and operator as in A; `docs/findings/README.md` keeps `File | Status | Area |
One line`, so a reader sees open findings on a forge page without a terminal.
Changes: no marker in the findings README; the `finding` skill keeps its three row steps;
`handoff` step 1 and the split line read the column (the column name fixed to `Status`); `decant`
and `adopt-hatchery` keep the findings table reset. To stop the drift the sibling shows, the hook
gains a check that a finding's `status:` first word equals its row's Status cell, and the script's
`findings` mode compares the two and prints `drift` on a mismatch.
Constraints: the table that stays is the most-churned one (regenesis: 40 commits, 37 row-only,
against the ideas README's 20); its rows are resolved and superseded as well as added, which is
the delete-against-append shape the experiment shows conflicting. Findings are never deleted (13
added, 0 deleted in Hatchery; 28 in regenesis), so the table only grows and the README reaches the
300-line cap in time, which no rule here resolves. The forge reader's gain is real but small:
the single-file handoff links the open findings already, and the H1 slug states the result.
Exit cost: to A, drop the table and the sync check and add the marker (one commit); to C,
nothing. Moving parts: A's plus one hook check and its cases. Promised surfaces: as A. Sibling
migration: as A for two directories; the findings table is kept as it is.

### C. Status quo, with the two defects the finding exposed fixed

What it is. All three tables stay. The `handoff` skill's "Open column" becomes "Status column"; the
hook gains the status-sync check of B for findings and a row-to-file check for every indexed
directory (a row naming an absent file warns, so regenesis's two stale rows would have been
caught). The `triage` skill keeps deriving idea state from files and ignores the rows, as today.
Constraints: every add, resolve and delete edits one shared file per directory (in Hatchery, 59
file adds and deletes across the three in 101 commits); until 0016 lands, each base update in
merge mode conflicts on the tables (3 of 5 in regenesis, 3 of 3 in broodforge, replayed); once a
project runs two sessions, dated rows meet at the end of each table and conflict (the experiment,
rows 1 and 4); status stays in two places with a check instead of a single copy. The owner's
stated purpose for one-file-per-item, no shared file, is not met.
Exit cost: none; this is where the tree is. Moving parts: two new hook checks and their cases.
Promised surfaces: none. Sibling migration: none.

### `docs/reference/`, `docs/design/` and `docs/design/proposals/`: the owner's distinction holds

The numbers separate `reference/` and `design/` from the three. `reference/README.md` changed
prose in 2 of 5 commits and gained 2 files in 101 commits; `docs/design/README.md` changed prose
in 7 of 19 and its ADR list is append-only (16 added, 0 deleted) with a sentence per ADR that the
ADR title does not carry: it is the map of the decisions, read before the files, and stays curated
under every option. Under 0016 Hatchery's ADRs and a project's live in different directories, so
the two writers of that list separate (the finding's last practice row).

`design/proposals/README.md` looked like a queue index by count (11 of 13 commits row-only, 14
files added and 0 deleted) and the first draft took it as the fourth directory. The critic's
objection 2 stands: its rows are curated summaries in every repository that has them (Hatchery's
0016 row is a paragraph of the decision; broodforge's and regenesis's carry verdicts and who is
waited on), a listing's H1 carries none of that, proposals are never deleted and are added about
once per major decision, and their `NNNN` number is a shared counter that two parallel sessions
collide on before any row. It is a README that serves as documentation, in the owner's words, and
stays a written index like `docs/design/README.md`; its `Status:` lines are still the item's own
record, and 0003's and 0005's are wrong today (`draft` after a critic review) and are fixed in the
first step. Should the owner later want proposals listed, a `proposals` mode is an addition under
the promised row, and the companion-review `Status:` lines (dystopo's `review of ...`) are
classified by file name first. `NNNN` collisions between parallel sessions (log4brains's reason for
dated names) are a separate choice and out of scope; 0016 already removes the base-against-project
collision.

## Recommendation

A, for `docs/ideas/`, `docs/operator/` and `docs/findings/`. The owner's reason is the
structural one: a shared file that every add, resolve and delete edits is the thing
one-file-per-item was chosen to avoid, and the three READMEs are that file. The measured history
says the rows are all the READMEs change (16 of 17 and 13 of 14 commits in Hatchery, 37 of 40 and
17 of 20 in regenesis), that a second copy drifts in both directions (the `suite-duration` row, two
stale rows, and 0003 and 0005 the other way), and that the conflicts so far came from the base's
rows meeting the project's, which 0016 ends; what A removes beyond that is the parallel-session
conflict ADR 0015 expects, the second copy and one edit per change, while the marker keeps the
edit-time check and every sibling's behaviour until it chooses. Every tool the finding surveyed
that keeps one file per item lists on demand; the ones that committed an index did so for a site
build this repository does not have. B keeps the most-churned table for a reader who has the
handoff and the slug already, and adds a check to police a copy A does not have. C fixes the two
defects and keeps the structure that produced them. Nothing here is human-facing: the READMEs are
read by agents and the owner, the owner asked for this shape, and the decision is technical under
`CLAUDE.md`; it waits on nothing.

First step, after 0016's update half has landed: one coder brief for the script, its export, its
harness group, and the hook's marker skip and `?` warning (the existing index fixtures keep their
expected output); then one tester brief for Hatchery's three READMEs (tables out, marker in), the
rule file, the skill lines named under A, `docs/README.md`, and the `Status:` lines of proposals
0003 and 0005 set to `under review`; then the ADR (0017), amending `0002-repository-layout.md`,
`0002-hook-specs.md`, 0006, 0007 and 0013 as listed under A. Siblings at their next update, each
by the paragraph above.

Predictions (the trigger scan reads these):
- Prediction, check by 2027-04-17: `node .claude/hooks/queue-list.js <mode> --check` exits 0 for
  all three modes in Hatchery, and no work merge in any sibling that added the marker has
  conflicted on `docs/ideas/README.md`, `docs/operator/README.md` or `docs/findings/README.md`.
- Prediction, check by 2027-04-17: no tree that added the marker has re-added a per-file table
  under it.

## What would change this recommendation

- The owner wants status visible on a forge page without a terminal: B for findings.
- Node is found missing from a tool shell where the hooks still run: the script becomes a one-line
  `grep` per skill (the finding's section 3 form) and loses `--json` and the block reading.
- Either prediction above fails: the ADR is reopened with B's forge-page argument heard again.
- The pusher needs the listing (an unparsable status blocking a push): check 8 runs `--check`;
  nothing in the shape changes.
- A project wants one queue indexed and two not: the marker is per README already, so that is a
  README edit, not a change.
- The critic shows the first-word reading misclassifies a real finding `status:` line: the script
  reads the whole value against the rule file's set instead.

## Claims to verify

- `post-edit.js` 34 and 92 to 110: `INDEX_EXEMPT` is README and HANDOFF only; the check walks up to
  `docs/<dir>/README.md`, tests `includes(base)` (a substring), and warns without blocking. Lines
  84 to 87 test the findings frontmatter for key presence, not values. (Critic: holds.)
- `fitness.json` `deps.paths` is `.claude/hooks/*.js`, so the script falls under group `F`.
  (Critic: holds.)
- `finding/SKILL.md` 19, 29, 34; `triage/SKILL.md` 22 to 23 and 49; `decant/SKILL.md` 54, 78 to 80,
  85; `adopt-hatchery/SKILL.md` 49, 53; `handoff/SKILL.md` 17 to 18 and `split.md` 15;
  `update-from-hatchery/SKILL.md` 55, 65: the lines named under A are the only ones that read or
  write a queue row; `proposal/SKILL.md` 22 is already conditional; the `adr` skill never edited
  the proposals row. (Critic: holds, with those two corrections, taken.)
- `0002-repository-layout.md` lines 52, 92, 140; `0002-hook-specs.md` line 193; ADR 0006's "only
  exemptions are policy" sentence; ADR 0007 line 71; ADR 0013 lines 49 to 52 (the frontmatter
  trigger) are the ADR text A amends; ADR 0008 is superseded by 0016.
- The regenesis counts above (recounted by the critic: hold); the stale-row bodies are af2b8d27 and
  62f15f9c; the status values in use (Hatchery 3 open, 10 resolved, 1 superseded; regenesis 18, 8
  plus one free-text, 1; broodforge 5, 5; dystopo 1, 17; droplet-management 3 open) all read by
  first word; dystopo's 19 ideas carry `kind:` and the two extra keys (read 2026-10-17 at c781ab5).
- The finding's experiment rows 1 and 4 (append against append and delete against append conflict
  by default) and its section 5 replay numbers.
- Node starts in about the time `grep` takes here (0.065 s against 0.034 s); Git's `grep.exe` is on
  the PowerShell tool's path on this machine.

## Critic's objections and responses

1. The `edit.unindexed` default changes every sibling's behaviour; the proposal called it free.
   Response: correct, and the two sentences contradicted each other. The switch is now the marker
   line in the directory's own README, which the hook reads: a README with a table keeps its
   warning, no key is added, no schema moves, no older-`policy.js` hazard, and the choice lives in
   the project-owned file it is about. The second choice (`[]` default set by Hatchery, `decant`
   and `adopt-hatchery`) was weighed and loses: it needs a schema change, the 0016 key order, a
   policy edit in every sibling that drops its tables, and a decant step, for the same effect the
   marker gets from the README edit the sibling makes anyway. Skills that stop writing rows: their
   row steps are conditional on the README carrying a table, and the hook's unchanged warning
   tells the agent which case it is in, so a kept table is still kept current. The migration
   names all four siblings. Resolution: accepted with change (the README marker).
2. `docs/design/proposals/` is not a queue and its rows are not a second copy. Response: correct
   on every point; the owner named three directories and the fourth was added on a count. The
   proposals README stays a curated written index; the `proposals` mode is dropped from the script
   (named as a possible later addition); the exit-cost sentence no longer claims to regenerate it.
   Resolution: accepted.
3. "The row is always a second copy" is false in the direction that matters, and A removed the
   only edit-time check. Response: accepted on both halves. The Constraints now say either copy
   drifts; 0003 and 0005 get `Status: under review` in the first step. The hook, for a file in a
   listed-on-demand directory, replaces the index warning with the script's own reading of that
   file through a shared export, naming the field and the allowed set, so an idea filed from
   conversation is checked at the edit as today. The review column is moot with proposals kept
   indexed; the record says review state is the README's row. Resolution: accepted with change.
4. The reading misclassifies dystopo's files. Response: proposals mode is gone, so the
   `Status: review ...` lines are read by nothing. Dystopo's 19 ideas all carry `kind:` (checked
   2026-10-17), so kind reads; the two keys outside the grammar make the block column `?` with a
   stderr line, `--untriaged` never selects a `?`, and the exit code is 0 unless `--check` is
   asked for, so no call in that tree fails. Dystopo's migration is named: keep its table and
   grammar (hook unchanged) or move to the promised grammar and add the marker.
   Resolution: accepted with change (the `?` block, `--check`, exit 0 by default).
5. The command line becomes text in project-owned files the base can never patch, and is not
   promised. Response: accepted. The ADR 0013 row now names the path, the three modes, `--status`,
   `--untriaged`, `--json` with its field names and the `?` literal, `--check`, the exit meanings,
   the finding's first-word status and the marker line; the text output, `--markdown` and stderr
   wording are not promised; agents parse `--json`. Resolution: accepted.
6. The amendment list is incomplete and partly moot. Response: corrected. `0002-hook-specs.md` 193
   and ADR 0006 join; ADR 0008 drops out; the 52, 92, 140 lines are in `0002-repository-layout.md`
   (the critic placed them in the hook specs, where those lines are the guard's parser text). ADR
   0006's key table gains nothing because no key is added; its "only exemptions are policy"
   sentence is what changes. Resolution: accepted with change.
7. Sequencing against 0016 not stated; the malformed-policy hazard. Response: 0017 is implemented
   after 0016's update half lands, said under A and in the first step; "either mode" is gone
   (patch only). With no key the hazard does not exist, so 0016's step 4 sentence needs no
   extension from this record. Resolution: accepted with change (the hazard removed rather than
   documented).
8. Factual corrections. Response: the stale-row bodies are af2b8d27 and 62f15f9c; the merge-mode
   case is a content conflict (and is now history under 0016, so the sentence is gone); Git's
   `grep.exe` is on the PowerShell tool's path here and the Node case rests on `--json` and one
   tested parser; the split map's Open findings line names the command, and the record says a
   split-shape reader runs it or opens the files. Resolution: accepted.
9. Boilerplate exit cost, an unfalsifiable reversal, no predictions. Response: the exit cost now
   says what `--markdown` restores (rows with the H1) and what a tree rewrites by hand, per tree;
   moving parts list the marker, the README command lines and the promise; the year-of-two-sessions
   item is replaced by "either prediction fails"; two dated predictions are added, with
   `--check` as the first one's instrument. Resolution: accepted with change.
10. Script details. Response: the stderr line is specified; `--status <v>` matches the column or
    the whole value; `proposal` step 3 and the `adr` skill need no edit, and the first draft's open
    claim about the `adr` skill is closed as the critic found it. Resolution: accepted.

Nothing serious remains unresolved; the recommendation is A as revised, for the three directories
the owner named.
