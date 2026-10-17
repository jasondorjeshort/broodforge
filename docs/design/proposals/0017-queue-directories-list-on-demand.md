# 0017. Queue directories list on demand: no hand-kept index in their READMEs

Date: 2026-10-17
Status: under review

## Question

Four directories under `docs/` hold one file per item: `ideas/`, `operator/`, `findings/` and
`design/proposals/`. Each README ends in a hand-kept table with one row per file, and the rule
file, the post-edit hook and four skills read or write those rows. The owner's words: in `ideas/`,
`operator/` and `findings/` "we most likely should not put an index of the files into the readme.
The entire point of using files for this is to not have conflicts on that one file constantly";
`reference/` and `design/` "serve as a documentation file" and may differ. Decide whether the four
keep a per-file index, and if not, how a reader, a skill and a sibling project find items and
their status. The evidence is `docs/findings/2026-10-17-queue-directory-indexes.md` (cited below
as "the finding"); it has the practice of nine one-file-per-item tools, a merge experiment and the
counts for Hatchery and two siblings. In ten years the chosen shape should let any number of
sessions add, resolve and delete items in parallel with no file they all edit, and let an agent
answer "what is open" with one command whose output it can parse.

## Constraints that matter here

- Agents write everything; a listing must run from the Bash and PowerShell tools on Windows, so it
  cannot rely on `grep -H -m1` being the shell's own (it is in Git Bash, not in PowerShell). Node is
  already required by every hook, and ADR 0013's fitness check `F` holds `.claude/hooks/*.js` to
  Node built-ins.
- The debug interface rule: whatever replaces the rows must be drivable from a terminal and
  parseable (`--json`), and tested in `test-hooks.js` like the hooks.
- Proposal 0016 (decided 2026-10-17, Recommendation): `docs/` is project-owned except the base-owned
  subtree `docs/hatchery/`, where Hatchery's records keep a curated ADR index; updates become one
  patch from a base-owned allowlist. So the four READMEs are each project's own file from then on,
  and base-update conflicts on them end with or without an index; `.claude/` (hook, skills, policy
  defaults) is base-owned and reaches every sibling.
- ADR 0013: the policy file's keys are promised (an added key with a preserving default is free);
  the findings, reference and operator frontmatter are "not promised, trigger: the first program
  that reads them". The idea file grammar is promised; the ideas README's index is not part of it.
- `.claude/rules/docs.md`: an index is the one list a doc may carry; prose never inventories files.
  Every status already lives in the item: findings `status:`, ideas `kind:` plus the closing block,
  proposals and ADRs the `Status:` line under the title. The row is always a second copy.
- Already ruled out (the finding, section 4): `merge=union` and marker lines for the queues (union
  resurrects deleted rows and imports the base's rows; GitHub and Bitbucket do not honour it), and
  a generated index that is still committed (the same shared file, resolved by regenerating).

### What the histories show

Hatchery, 101 commits (the finding, section 5): commits to the ideas README changed rows only in
16 of 17, findings 13 of 14, proposals 11 of 13; `docs/design/README.md` changed prose in 7 of 19
and `reference/README.md` in 2 of 5. Files added and deleted: ideas 39 and 7, findings 13 and 0,
proposals 14 and 0, decisions 16 and 0, reference 2 and 0, operator none.

`../regenesis`, read today (`git -C ../regenesis log -- docs/findings/README.md docs/ideas/README.md`):
40 commits to the findings README, 37 row-only; 20 to the ideas README, 17 row-only. The only
merges that touched either are its two base updates (e4d61c7e created both tables from the base's
rows; 8745c5d1 created the ideas and operator READMEs); its 45 work merges never conflicted on
them (the finding's replay), because its sessions have run one at a time on these files. Two
update commits (8745c5d1 and the 6866d8a update) each "drop a stale ideas index row" naming a
file the repository did not have: the hook checks file-to-row, never row-to-file, so a row
outlives its file unnoticed. One finding there (`2026-10-13-suite-duration.md`) says
`status: resolved (ADR 0011, 2026-10-14)` while its row says `open`; the resolving commit d2a28872
left the README alone. The owner adds that findings and ideas are underused there (18 ideas, 28
findings today), so the counts understate the churn a used queue would show. The conflicts seen
so far are base-against-project, which 0016 ends; the parallel-session conflict is the shape the
finding's experiment shows (two appends at the end of a dated table conflict by default, and so
does a delete against an append) and has not yet been paid because no sibling has run two
sessions on a queue at once. ADR 0015 chose the split handoff for exactly that day.

## The listing script (shared by A and B)

`.claude/hooks/queue-list.js`, Node built-ins only (`fs`, `path`), under fitness check `F` like
the other files there, with cases in `test-hooks.js` (the next free group letter) run by the test
command and pusher check 8. It is the agent-usable surface for the four directories.

```
node .claude/hooks/queue-list.js <ideas|operator|findings|proposals> [--status <v>] [--untriaged] [--json] [--markdown]
```

One line per file except `README.md`, sorted by name (the date prefix makes that age order),
tab-separated, the H1 text last (the first non-blank line for an operator entry, which has no H1):

- `findings`: `<file>  <status>  <title>`; status is the first word of the `status:` frontmatter
  value (`open | resolved | superseded`; the sibling's `resolved (ADR 0011, 2026-10-14)` reads as
  `resolved`). `--status open` filters.
- `ideas`: `<file>  <kind>  <block>  <title>`; block is `plan`, `question`, `answered` (a
  `Question:` whose `Answer:` is not blank), `waits` or `untriaged`, by the same reading the
  `triage` skill makes today (step 2). `--untriaged` filters.
- `operator`: `<file>  <title>`.
- `proposals`: `<file>  <status>  <title>`; status is the `Status:` line under the title, first
  word (`draft`, `under`, `promoted`, `withdrawn`); a companion review file without one prints `-`.
- `--json` prints an array of objects with those fields; `--markdown` prints the rows in today's
  table format (the exit path, below). A file whose status, kind or frontmatter is missing or not
  in the allowed set prints `?` in that column and the exit code is 1, so a skill that calls it
  sees the bad file; nothing blocks on it (an addition to the pusher later, if wanted).

Cost: about eighty lines and a dozen harness cases; a Node start per call, about the 0.05 s the
finding measured for `grep`. Who calls it: the `handoff` skill (open findings), `triage` (the
untriaged list), `decant` and `adopt-hatchery` (the resting-state check and the split map's Open
findings line), the coordinator and any reader who wants status without opening files, and the
`docs/README.md` map in its Loaded column. The finding (section 3) shows the `triage` skill already
derives idea state this way; the script makes that reading one tested place instead of a grep
each skill restates.

## Options

### A. No index in the four READMEs; the script lists; status lives only in the item

What it is. Each of the four READMEs keeps its format rules and gains one line naming the listing
command; the `## Index` table goes. Status, kind and block are read from the item. Changes:

- Hook (`post-edit.js` lines 101 to 110): the index check skips a file matching a new policy key
  `edit.unindexed`, a `patterns` list like `anchorExempt`, default in `policy.js`
  `["docs/ideas/*", "docs/operator/*", "docs/findings/*", "docs/design/proposals/*"]`. A project
  that keeps an index sets it to `[]`. The check for every other directory is unchanged. The
  row-to-file direction, which the hook never had, needs nothing: there are no rows.
- Rule file (`.claude/rules/docs.md`, Indexes): a directory is either indexed (its README names
  every file: `docs/design/` with every ADR, `docs/reference/`, `docs/hatchery/`) or listed on
  demand (the four; the README carries the format and the command, never a file name;
  `edit.unindexed` names them for the hook). The line "`docs/findings/README.md` shows status
  per file" goes; the one-list rule stays. Frontmatter: the status value is read by its first word.
- `finding` skill: step 3 of Record, step 3 of Resolve and the Supersede sentence lose their row
  edits; Record step 3 becomes "run the listing; the new file must print without `?`".
- `triage` skill: step 2 becomes the `--untriaged` call; the Bounds line about the scan's index
  rows goes.
- `decant` skill: steps 7, 8 and 9 delete files only, no table reset; the split template line (54)
  and `handoff/split.md` line 15 say "the listing's `--status open` lines". `adopt-hatchery` lines
  49 and 53 likewise.
- `handoff` skill step 1: "the Open column of `docs/findings/README.md`" (a column that does not
  exist) becomes the `--status open` call.
- `proposal` skill step 3 and the `adr` skill's promotion step: no proposals row; the `Status:`
  line is the record. `update-from-hatchery` lines 55 and 65 lose "their `README.md`s are the
  base's" and "drop their index rows" (0016 rewrites that skill anyway).
- `docs/README.md`: each queue row's Loaded cell names the command beside the README.
- ADRs: 0002 (line 49, "every directory ... one line per file"; the hook row, line 89) and 0007
  (Consequences, "every idea costs a file and an index line") amended; 0008's deletion list loses
  its "reset the table" words; ADR 0013 gains a row, since the script is the first program that
  reads the findings frontmatter values and the proposal `Status:` line: promised from then on
  are the key names and the first word of `status:` and `Status:`.
- Hatchery's own four READMEs lose their tables in the same change as the hook default.

Constraints: no shared file, so parallel sessions conflict on nothing by construction (the
finding's first practice row: towncrier, scriv, reno, changesets, adr-tools, kepctl, git-issue);
one copy of status, so the sibling's drift and the stale rows cannot recur; the command is the
debug interface. What a reader loses: a forge page shows file names, not status; the H1 slug and
the date prefix carry what the one-line column did, and `HANDOFF.md` still links the open findings.
Exit cost: `--markdown` prints the rows, paste them under `## Index`, set `edit.unindexed: []`,
restore the eight skill lines; one base commit, no data migrates. Moving parts: one script, one
policy key with a default, one harness group, one ADR 0013 row.
Promised surfaces (ADR 0013): the policy file, by an added key (free); the ADR 0013 trigger on
the findings frontmatter fires, adding a row. The idea grammar, the base line, the cache, the grant
files, the hook seams and the handoff paths are untouched. The default is not behaviour-preserving
for a project that keeps its index (it loses the unlisted-file warning until it sets the key), so
the ADR says so and names the one-line migration.
Sibling migration: the `.claude/` half arrives with the next update in either mode. The READMEs
are the project's own: drop the four tables in one commit, or keep them and set
`edit.unindexed: []`. In merge mode today the base deleting its rows meets the sibling's rows
once (a delete-against-modify conflict, resolved by keeping or dropping the sibling's table); 0016
ends merging, and in patch mode the READMEs never move (the open idea
`2026-10-16-patch-update-never-moves-the-base-docs.md`), so `../regenesis` edits them by hand.

### B. As A, but the findings README keeps its table because of the Status column

What it is. Ideas, operator and proposals as in A; `docs/findings/README.md` keeps `File | Status
| Area | One line`, so a reader sees open findings on a forge page without a terminal.
Changes: `edit.unindexed` default without the findings pattern; the `finding` skill keeps its
three row steps; `handoff` step 1 and the split line read the column (the column name fixed to
`Status`); `decant` and `adopt-hatchery` keep the findings table reset. To stop the drift the
sibling shows, the hook gains a check that a finding's `status:` first word equals its row's
Status cell, and the script's `findings` mode compares the two and prints `drift` on a mismatch.
Constraints: the table that stays is the most-churned one (regenesis: 40 commits, 37 row-only,
against the ideas README's 20); its rows are resolved and superseded as well as added, which is
the delete-against-append shape the experiment shows conflicting. Findings are never deleted (13
added, 0 deleted in Hatchery; 28 in regenesis), so the table only grows and the README reaches the
300-line cap in time, which no rule here resolves. The forge reader's gain is real but small:
`HANDOFF.md` links the open findings already, and the H1 slug states the result.
Exit cost: to A, drop the table and the sync check (one commit); to C, nothing. Moving parts: A's
plus one hook check and its cases. Promised surfaces: as A. Sibling migration: as A for three
directories; the findings table is kept as it is.

### C. Status quo, with the two defects the finding exposed fixed

What it is. All four tables stay. The `handoff` skill's "Open column" becomes "Status column"; the
hook gains the status-sync check of B for findings and a row-to-file check for every indexed
directory (a row naming an absent file warns, so regenesis's two stale rows would have been
caught). The `triage` skill keeps deriving idea state from files and ignores the rows, as today.
Constraints: every add, resolve and delete edits one shared file per directory (in Hatchery, 73
file adds and deletes across the four in 101 commits); until 0016 lands, each base update in
merge mode conflicts on the tables (3 of 5 in regenesis, 3 of 3 in broodforge, replayed); once a
project runs two sessions, dated rows meet at the end of each table and conflict (the experiment,
rows 1 and 4); status stays in two places with a check instead of a single copy. The owner's
stated purpose for one-file-per-item, no shared file, is not met.
Exit cost: none; this is where the tree is. Moving parts: two new hook checks and their cases.
Promised surfaces: none. Sibling migration: none.

### `docs/reference/` and `docs/design/`: the owner's distinction holds

The numbers separate them from the four. `reference/README.md` changed prose in 2 of 5 commits and
gained 2 files in 101 commits; `docs/design/README.md` changed prose in 7 of 19 and its ADR list
is append-only (16 added, 0 deleted) with a sentence per ADR that the ADR title does not carry: it
is the map of the decisions, read before the files, and stays curated under every option. Under
0016 Hatchery's ADRs and a project's live in different directories, so the two writers of that
list separate (the finding's last practice row; ADR 0008 line 61 assumed both sides append at the
same spot, which the experiment shows conflicts by default). `design/proposals/README.md` is the
exception in `design/`: 11 of 13 commits row-only, 14 files added and 0 deleted, rows that change
on every status move; it behaves like a queue index and is treated as one here. `NNNN` numbering
collisions between parallel sessions (log4brains's reason for dated names) are a separate choice
and out of scope; 0016 already removes the base-against-project collision.

## Recommendation

A. The owner's reason is the structural one: a shared file that every add, resolve and delete
edits is the thing one-file-per-item was chosen to avoid, and the four READMEs are that file. The
measured history says the rows are all the READMEs change (16 of 17, 13 of 14, 11 of 13 commits),
that the second copy of status drifts (1 of 56 in the siblings, two stale rows in regenesis), and
that the conflicts so far came from the base's rows meeting the project's, which 0016 ends; what A
removes beyond that is the parallel-session conflict ADR 0015 expects, the second copy and one
edit per change. Every tool the finding surveyed that keeps one file per item lists on demand;
the ones that committed an index did so for a site build this repository does not have. B keeps
the most-churned table for a reader who has `HANDOFF.md` and the slug already, and adds a check
to police a copy A does not have. C fixes the two defects and keeps the structure that produced
them. Nothing here is human-facing: the READMEs are read by agents and the owner, the owner asked
for this shape, and the decision is technical under `CLAUDE.md`; it waits on nothing.

First step: one coder brief for the script, its harness group and `edit.unindexed` in `policy.js`
and `post-edit.js`; then one tester brief for the rule file, the eight skill lines, `docs/README.md`
and Hatchery's four READMEs, in one commit so the hook default and the tables move together; then
the ADR (0017), amending 0002, 0007, 0008 and 0013 as listed under A. Siblings at their next update.

## What would change this recommendation

- The owner wants status visible on a forge page without a terminal: B for findings.
- Node is found missing from a tool shell where the hooks still run: the script becomes a one-line
  `grep` per skill (the finding's section 3 form) and loses `--json` and the block reading.
- A project keeps its tables after setting `edit.unindexed: []` and shows no conflict in a year of
  two-session use: the default flips to `[]` and A's text becomes an option a project takes.
- The pusher needs the listing (an unparsable status blocking a push): a check is added; nothing
  in the shape changes.
- `edit.unindexed` turns out to be the wrong seam (a project wants one queue indexed and three
  not): it is a patterns list already, so that is a value, not a change.
- The critic shows the `Status:` first-word reading misclassifies a real proposal or ADR line:
  the script reads the whole value against the directory README's list instead.

## Claims to verify

- `post-edit.js` 34 and 101 to 110: `INDEX_EXEMPT` is README and HANDOFF only; the check walks up to
  `docs/<dir>/README.md`, tests `includes(base)` (a substring), and warns without blocking. Lines
  84 to 87 already test the findings frontmatter for key presence, not values.
- `policy.js` 61 to 63 and 103: `capExempt` and `anchorExempt` are `patterns`; `matches` lets `*`
  cross `/`; an unknown key makes the whole file malformed (ADR 0013 Context), so `edit.unindexed`
  must be added to the schema before any sibling writes it.
- `fitness.json` `deps.paths` is `.claude/hooks/*.js`, so the script falls under group `F`.
- `finding/SKILL.md` 19, 29, 34; `triage/SKILL.md` 22 to 23 and 49; `decant/SKILL.md` 54, 78 to 80,
  85; `adopt-hatchery/SKILL.md` 49, 53; `handoff/SKILL.md` 17 to 18 and `split.md` 15;
  `proposal/SKILL.md` 22; `update-from-hatchery/SKILL.md` 55, 65: the lines named under A are the
  only ones that read or write a queue row. Whether the `adr` skill's promotion step also edits the
  proposals row (its line 33 names `docs/design/README.md` only).
- ADR 0002 lines 49, 89, 137; ADR 0007 Consequences; ADR 0008 lines 47 to 51 and 61; ADR 0013
  lines 49 to 52 (the frontmatter trigger) are the ADR text A amends; no other ADR names the rows.
- Proposal 0016 Recommendation: `docs/` project-owned, `docs/hatchery/` base-owned, patch-only
  updates; so the four READMEs stop arriving from the base once it lands.
- The regenesis counts above (`git -C ../regenesis log --format=%h -- <README>` and the per-commit
  diff, rows being lines starting `|`); the two stale-row commit bodies; the status values in use
  (Hatchery 3 open, 10 resolved, 1 superseded; regenesis 18, 8 plus one free-text, 1).
- The finding's experiment rows 1 and 4 (append against append and delete against append conflict
  by default) and its section 5 replay numbers.
- Node starts in about the time `grep` takes here; `grep -H -m1` is not available in the
  PowerShell tool's shell without Git Bash on the path.

## Critic's objections and responses

(none yet)
