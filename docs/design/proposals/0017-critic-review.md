# 0017. Critic review: queue directories list on demand

Date: 2026-10-17
Reviewer: critic (Opus 5.5)
Proposal: `0017-queue-directories-list-on-demand.md` (status `under review`)

Verdict in one line: the direction of A is right for `ideas/`, `operator/` and `findings/`, and the
owner asked for it; the mechanism (a policy key whose default changes every sibling's hook
behaviour) and the scope (adding `design/proposals/`, whose rows are not a second copy) are wrong,
and two of the proposal's load-bearing claims fail on the evidence below. No blocking objection.

## Claims checked

Experiments ran in `scratchpad/critic17/` (session scratchpad); sibling repositories were read with
`git -C <repo> show HEAD:<path>` and `git -C <repo> log` only.

| Claim | Result | Evidence |
|---|---|---|
| `post-edit.js` 34, 101 to 110: `INDEX_EXEMPT` is README and HANDOFF; walk-up to `docs/<dir>/README.md`; `includes(base)`; warns only | Holds | Read the file; the loop stops at `depth >= 2`, and `main` never exits non-zero. |
| 84 to 87 test findings frontmatter for key presence, not values | Holds | `hasFrontmatter` checks `startsWith(key + ':')`. No check of any kind exists for `docs/ideas/*.md` or `docs/operator/*.md` (grep of the hook: neither path appears). |
| An `edit.unindexed` pattern list can skip the four without breaking the walk-up for nested paths | Holds | `matches()` from `policy.js` on the proposed default: `docs/design/proposals/0017-x.md` true, `docs/design/proposals/sub/x.md` true, `docs/design/decisions/0001.md` false, `docs/ideasfoo/x.md` false, `docs/hatchery/decisions/0017.md` false. The skip is tested before the loop, so the loop is untouched for every other path; existing harness fixtures P03, PX03 and PX05 keep their expected output. |
| `policy.js` 61 to 63, 103: `patterns` type; `*` crosses `/`; an unknown key makes the file malformed | Holds | `merge()` throws `unknown key edit.unindexed` on an older `policy.js`; `git-guard.js` line 563 then denies every git command. |
| `fitness.json` `deps.paths` is `.claude/hooks/*.js` | Holds | Also `.claude/statusline.js`. A script in `.claude/hooks/` falls under group `F`. |
| Each named skill line reads or writes a queue row | Mostly holds | `finding` 19, 29, 34; `triage` 22 to 23 (derives from files, reads no row) and 49; `decant` 54, 78 to 80, 85; `adopt-hatchery` 49, 53; `handoff` 17 to 18; `split.md` 15; `update-from-hatchery` 55, 65. `proposal` 22 is already conditional ("if that index lists files") and needs no edit. The `adr` skill does not edit the proposals row: step 6 sets the proposal `Status:` line only, so nothing but hand edits keeps the proposals rows current today. |
| ADR text A amends; "no other ADR names the rows" | Fails | Missed: `0002-hook-specs.md` line 193 (the index-check row of the post-edit spec) and ADR 0006's living key table (lines 68 to 70, where every `edit.*` key is documented). ADR 0002's lines are 52, 92, 140 in the working tree (0016's amendment added three). ADR 0008 is `superseded by 0016`, so amending it is moot. |
| Regenesis counts: findings README 40 commits, 37 row-only; ideas 20, 17 | Holds | Recounted: every `+`/`-` line of each commit's diff to the file, rows being lines starting `|`. |
| The two stale-row commits are 8745c5d1 and the 6866d8a update | Half fails | The bodies are in af2b8d27 ("Handoff: one file per workstream", not an update) and 62f15f9c (the 6866d8a update). 8745c5d1 created the READMEs and says nothing of a stale row. |
| `2026-10-13-suite-duration.md` file `resolved (ADR 0011, 2026-10-14)`, row `open` | Holds | Both read at regenesis HEAD d02f09bc. |
| Status values in use; first-word reading handles them | Fails for one sibling | Findings: Hatchery 3 open, 10 resolved, 1 superseded; regenesis 18, 8 plus the free-text one, 1; broodforge 5, 5; dystopo 1, 17; droplet-management 3 open. All read cleanly by first word. Proposals: every `Status:` line in four repositories reads as `draft`, `under`, `promoted` or no line, except dystopo's `0027-critic-review.md` and `0028-critic-review.md`, whose line is `Status: review of an accepted ADR ...`: first word `review`, so `?` and exit 1 on every call there. Ideas: dystopo's 19 ideas carry their own grammar (`status: new | planned | done | dropped`, `pipeline:`), no block, so all 19 print `untriaged`, done ones included. Hatchery's reading gives 24 untriaged, matching the finding. |
| Merge-mode base row deletion meets the sibling's rows once, "delete-against-modify" | Holds as a conflict, wrong name | `git merge-file` of base-with-rows, project-with-own-rows, base-with-command-line: one content conflict over the table. It is a content conflict, not a delete/modify one. |
| Experiment rows 1 and 4 and the replay numbers | Not re-run | Taken from the finding; the brief rules out re-arguing union. |
| Node starts in about the time `grep` takes | Holds | `node -e 0` 0.065 s, `grep -H -m1 '^status:' docs/findings/2*.md` 0.034 s, here. |
| `grep -H -m1` is not available in the PowerShell tool's shell | Fails on this machine | `Get-Command grep` in PowerShell resolves to `C:\Program Files\Git\usr\bin\grep.exe`. The case for Node stands on `--json` and one tested reading of the block grammar, not on grep being absent. |
| Proposal 0016: `docs/` project-owned, `docs/hatchery/` base-owned, patch-only | Holds | ADR 0016 Decision items 1, 2, 5, 9 (working tree). Item 9 names 0017 as owning this question. |

## Objections

1. Serious. The `edit.unindexed` default changes every sibling's behaviour, and the proposal calls
   it free. Under "Promised surfaces" A says "the policy file, by an added key (free)"; two lines
   later it says the default "is not behaviour-preserving". ADR 0013 (The promise) defines a free
   addition as "a new key with a default that preserves old behaviour" and promises "a file or line
   written against an older base loads under a newer one with the same meaning": a sibling's
   `policy.json` without the key means "check my four indexes" today and "do not" after the
   update. Worse, the base skills stop writing rows at the same update (finding steps 3, triage
   Bounds, decant), so a sibling that keeps its tables gets tables nobody updates and a hook that
   no longer says so. Dystopo is that sibling: its `docs/ideas/README.md` Triage section says
   "Update the table in the same edit", and its Index has a Status column (`done`, `planned`) that
   its own process reads. The proposal's Sibling migration names regenesis only; ADR 0013 asks
   for each sibling. Resolve: make the switch preserving. Best: no key at all; the hook skips the
   index check for a directory whose own `README.md` carries the listing line A already adds to
   each README (a fixed marker such as a line starting `Listed on demand:`), so a README with a
   table keeps its warning, the choice lives in the project-owned file it is about (0016's
   ownership rule), there is no schema change and no older-`policy.js` hazard (objection 7).
   Second best: default `[]`, with Hatchery's own `policy.json` and the `decant` and
   `adopt-hatchery` steps writing the four patterns. Either way, the migration names all four
   siblings.

2. Serious. `docs/design/proposals/` is not a queue and its rows are not a second copy. The owner
   named three directories; the proposal adds the fourth on the "row-only" count. But the rows
   are curated prose in every repository that has them: Hatchery's 0016 row is a paragraph of the
   decision; broodforge's say "its verification table is authoritative over th[e proposal]";
   regenesis's say "proposed, awaiting Dorje". The H1 a listing prints carries none of that, and
   `--markdown` cannot regenerate it, so the exit cost line is wrong for this directory. Proposals
   are never deleted (14 added, 0 deleted), are added about once per major decision, and are
   numbered from a shared `NNNN` counter, so two parallel sessions collide on the number before
   they ever collide on the README row: A does not deliver its ten-year goal ("no file they all
   edit") for proposals, it only moves the collision. The owner reads this file; dropping its
   summaries is a change to what a human reads that the owner did not ask for. Resolve: keep
   `design/proposals/` indexed and curated like `docs/design/README.md` (the owner's own
   distinction: a README that serves as documentation), and take A for the three the owner named;
   or, if the architect still wants it in, put the question to the owner with these rows as the
   example.

3. Serious. "Every status already lives in the item ... The row is always a second copy" is false
   in the direction that matters, and A removes the only edit-time check while adding none.
   Proposals 0003 and 0005 say `Status: draft`; both were critic-reviewed on 2026-10-13
   (80d84a6) and the proposal skill has said "Set Status to `under review`" before the critic since
   b1d4c79 (2026-10-11). Their README rows ("Critic reviewed; awaits the architect's answers") are
   right and the items are wrong; `docs/HANDOFF.md` Next item 4 points at the README for exactly
   this state. Under A the listing reports both as `draft`, in parseable form, and agents will
   trust one command more than they ever trusted a table. Second: the unlisted-file warning is
   today the only thing the hook says about an idea or operator file at all (no frontmatter check
   exists for either, see the claims table); A removes it and the replacement validation ("prints
   `?` ... nothing blocks on it") runs only when someone calls the script, which for ideas is
   nobody at filing time (ideas are filed from conversation and by the triage scan, neither of
   which runs the listing). Resolve: the hook replaces the index warning, for a file in a listed-
   on-demand directory, with the script's own reading of that one file (warn when it would print
   `?`, naming the field and the allowed set), sharing the parser so the two never disagree; fix
   0003's and 0005's `Status:` lines in the same change; and either add a `review` column to the
   proposals mode (a companion `NNNN-critic-review*.md` exists) or say in the record that review
   state is not listed.

4. Serious. The script's reading misclassifies real sibling files, and the proposal says it was
   checked against "every status value". Dystopo's two critic reviews carry a `Status:` line whose
   first word is `review`, so `proposals` mode prints `?` and exits 1 on every call in that tree,
   forever, and any skill that treats exit 1 as "a bad file" reports the same two files at every
   handoff. Dystopo's 19 ideas, mostly `status: done`, print `untriaged`, and `--untriaged` hands a
   triage pass 19 finished items. The `-` rule for "a companion review file without one" does not
   cover a companion review with one. Resolve: classify a companion by name
   (`NNNN-critic-review*.md`, any `NNNN-` file that is not the proposal itself) before reading its
   `Status:`; in `ideas` mode, a file whose frontmatter carries keys outside the promised grammar
   prints them (or `?`) rather than a confident `untriaged`; and the migration says dystopo keeps
   its index or moves its grammar, by name.

5. Serious. Hidden coupling: the listing's command line becomes text in project-owned files the
   base can never update, and the proposal does not promise it. A writes the command into each
   queue README, into `docs/README.md`'s Loaded column, and (through `split.md` and the decant
   template) into every split-shape project's `docs/HANDOFF.md` map. Under ADR 0016 all of those
   are project-owned and never patched. The script's path, mode names, flags, `--json` field names
   and exit codes are therefore a surface with readers in every tree, which ADR 0013 exists to
   name; A's 0013 row promises only the frontmatter first words. A later rename (into
   `.claude/bin/`, say, or `ideas` mode renamed) leaves every sibling's README and map naming a
   command that does not exist, noticed only when someone next edits that file and the anchor
   check fires. Resolve: add the CLI to A's ADR 0013 row (path, the four modes, `--status`,
   `--untriaged`, `--json` field names, exit 0/1 meaning; the tab-separated text and `--markdown`
   not promised), and say in the record that agents parse `--json`, never the text.

6. Minor. The ADR amendment list is incomplete and partly moot. It misses `0002-hook-specs.md`
   line 193 (the spec row for the index check, which the new skip changes) and ADR 0006's living
   key table (lines 68 to 70), which must gain the new key or the README switch. ADR 0008 is
   `superseded by 0016` in the working tree, so there is nothing of it to amend. ADR 0002's cited
   lines are 52, 92 and 140 now. Resolve: correct the list.

7. Minor. Sequencing against ADR 0016 is not stated, and one hazard follows from it. Both changes
   edit `post-edit.js`, `policy.js`'s schema, `decant`, `adopt-hatchery`, `update-from-hatchery`,
   `split.md` and the rule file; "arrives with the next update in either mode" is gone under 0016
   (patch only). A sibling following A's migration text ("keep them and set `edit.unindexed: []`")
   before its `policy.js` knows the key gets a malformed policy, and the guard denies every git
   command. Who lifts it: anyone with the Edit tool, mid-session, by removing the key (no restart);
   the guard's message says "Every git command is denied until it is fixed; ask the user", and
   0016's update procedure adds the sentence naming a newer-base key as one cause. So the lift
   exists and is stated, but A's migration text does not repeat 0016's order rule (step 4 covers
   `update.exclude` only). Resolve: state that 0017 is implemented after 0016's update half lands,
   and extend 0016's step 4 sentence to every new key; objection 1's README switch removes the
   hazard outright.

8. Minor. Factual corrections. The stale-row commit bodies are af2b8d27 and 62f15f9c, not
   8745c5d1. The merge-mode case is a content conflict, not delete-against-modify. Git's
   `grep.exe` is on the PowerShell tool's PATH on this machine, so the Node rationale should rest
   on `--json` and one tested parser. "`HANDOFF.md` still links the open findings" holds for the
   single-file shape only: in the split shape the map's line is "Open findings: the Status column
   of `docs/findings/README.md`" (regenesis, 18 open findings, links none), so under A that shape
   has no list a reader sees without running the command.

9. Minor. Boilerplate exit cost, an unfalsifiable reversal and no predictions. Exit cost says
   `--markdown` restores the tables "in one base commit, no data migrates"; it cannot restore the
   curated proposals summaries or findings' One-line text beyond the H1, and each sibling's
   READMEs are its own commit under 0016. Moving parts omit the README command lines and the
   first-word promise that now binds siblings. "What would change" item 3 (a project keeps its
   tables and shows no conflict in a year of two-session use) cannot fire: no sibling runs two
   sessions on a queue, the base skills will not maintain a kept table, and it has no date. ADR
   0012 asks for predictions the trigger scan reads; A has none. Resolve: two dated predictions
   that can fail, for example "check by 2027-04-17: the listing prints no `?` in Hatchery, and no
   work merge in any sibling conflicted on a queue README" and "check by 2027-04-17: no sibling
   has re-added a per-file table to a listed-on-demand README".

10. Minor. Script details. The exit-1 `?` case has no stated stderr text; it should name the file,
    the field and the allowed set, as the fitness lines do. `--status under` for "under review" is
    the first-word rule leaking into the interface; accept the full value or the first word.
    The `proposal` skill step 3 needs no edit (already conditional). The `adr` skill's promotion
    step never touched the proposals row, which answers the proposal's own open claim and is why
    proposals rows drift only by hand.

## Refusing and failing mechanisms

| Mechanism | Who lifts, from where, mid-session | Message says so |
|---|---|---|
| `queue-list.js` exit 1 on a `?` file | Whoever fixes that file's frontmatter or `Status:` line, any session, at once; blocks nothing | Not specified (objection 10) |
| `edit.unindexed` written under an older `policy.js`: malformed policy, git-guard denies all git | Anyone with Edit, removing the key; mid-session, no restart | "ask the user"; 0016 adds the cause sentence (objection 7) |
| Harness group for the script failing: pusher check 8 refuses the push | The coder fixing the script or case; no skip entry exists outside fitness checks, which is normal for a test | The harness `FAIL` line; adequate |
| Post-edit warnings | Non-blocking; nothing to lift | n/a |

No mechanism in A refuses without a documented lift.

## Strongest case against the recommendation

The cost the owner felt is the base-against-project conflict, and ADR 0016 removes it with or
without an index. The parallel-session conflict A also removes has not happened once in 47 sibling
work merges, and when it does it is a two-row conflict an agent resolves in seconds. Against that,
A spends a script, a harness group, a policy key, eight skill edits, five ADR amendments and a
promise, removes the only browsable view of status (a forge page shows file names), and drops the
only edit-time nudge on idea files. Its single source of truth is a source that already drifts
(0003, 0005). This case is weaker than it looks for the three directories the owner named: the
owner's reason is structural, the history shows the rows are all the READMEs change, and status
already lives in the files. It is strong for `design/proposals/`.

## Strongest case for the best alternative

A', which the proposal does not consider: A for `ideas/`, `operator/` and `findings/`; the hook's
index check skipped by the README itself (the listing line A already writes is the switch), not
by a policy key; the hook warning when an edited queue file would print `?`; `design/proposals/`
kept as a curated index beside `docs/design/README.md`. It preserves every sibling's behaviour
until that sibling drops its table (ADR 0013 satisfied without argument), needs no schema change
and no update ordering, keeps the edit-time check A loses, keeps the one README whose rows are
documentation, and leaves the owner's request met in full. Its costs: a magic line the hook
greps (one harness case), and proposals keep one shared file that changes about once per decision.

## Pre-mortem

It is 2036 and the project regrets this. Every queue had one copy of status and no check at edit
or push, so items drifted the way 0003 and 0005 already had; the listing printed the wrong state
in clean JSON, every skill and coordinator trusted it, and decisions were made on proposals listed
as `draft` that had been reviewed and on ideas listed `untriaged` that were done. Siblings that
kept tables learned at an update, silently, that nothing warned or wrote rows any more, and their
tables rotted beside the files. The line that should have caught it is "Every status already
lives in the item ... The row is always a second copy" in Constraints: the evidence counted only
the row's drift (1 of 56) and never checked the item against the row the other way.

## Recommendation

Adopt A for `docs/ideas/`, `docs/operator/` and `docs/findings/`, as the owner asked, with three
changes: the switch is the directory's README (or, failing that, an `[]` default that Hatchery and
decant set), so no sibling's behaviour changes until it drops its table; the post-edit hook warns
on an edited queue file that the listing would mark `?`, using the same parser; and the listing's
CLI goes into ADR 0013's row. Keep `docs/design/proposals/README.md` as a curated index (fix the
0003 and 0005 `Status:` lines either way), unless the owner says otherwise when shown its rows.
Fix the dystopo misreadings and name every sibling in the migration. Confidence: high (about 85%)
that A's direction is right for the three named queues; medium (about 65%) that proposals belong
outside it; high that the policy-key default should not ship as written.
