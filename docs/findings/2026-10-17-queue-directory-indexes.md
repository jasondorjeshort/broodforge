---
date: 2026-10-17
status: resolved
area: process / indexes of queue directories
promoted_to: docs/design/decisions/0017-queue-directories-list-on-demand.md
---

# One-file-per-item projects keep no hand index: a tool lists the files when asked; git only narrows the conflict

## Question

The owner: in `docs/ideas/`, `docs/operator/` and `docs/findings/` "we most likely should not put an
index of the files into the readme. The entire point of using files for this is to not have
conflicts on that one file constantly"; `reference/` and `design/` may differ. How do projects with
one file per item avoid a hand-kept index, and how do readers and tools find items and status
without one? Evidence, no design. Extends `2026-10-16-handoff-layout-practices.md` and
`2026-10-17-base-repo-defaults-and-upstream-merges.md`. "Inference" is mine, "Observed" is local.

## What was tried

2026-10-17: WebSearch, WebFetch; `curl` and `gh api` of the sources named; GitLab REST API. git
2.56.0.windows.2 in `scratchpad/idx/`: `git apply --3way` onto a file staged with `git add`, no
commit (`git-guard` denies this agent `commit`). Sibling merges replayed with `git merge-tree
--write-tree` in `git clone --bare --shared` copies (today's merge-ort). Hatchery at 0339741.

### 1. Changelog fragment tools: per item, the list, any index

- towncrier (towncrier.readthedocs.io/en/stable/tutorial.html, cli.html): `<issue>.<type>` files;
  `build` "will remove the news files (with `git rm`)"; `--draft` prints; `check` wants a fragment.
- scriv (scriv.readthedocs.io/en/latest/commands.html): `changelog.d/` files named by date-time, user
  and branch, "collected in chronological order"; `collect` deletes them; `print` writes to stdout.
- reno (opendev.org/openstack/reno `reno/main.py`): `reno list` "list notes files based on query
  arguments"; `report` walks history; `cache` writes a generated cache (committed: not checked).
  changesets (github.com/changesets/changesets `docs/command-line-options.md`): `status`, JSON out.
  None of the four describes an index file.
- RuboCop: PR 3594 (github.com/rubocop/rubocop/pull/3594, 2016-10-10) set `CHANGELOG.md
  merge=union` as it "is very often conflict", noting "GitHub doesn't support the feature. So, We
  can't merge a PR at GitHub Web if CHANGELOG.md conflicts"; PR 8930 (2020-10-22) added
  `changelog/{type}_{description}.md`, "unique to avoid conflicts" (CONTRIBUTING.md). Observed:
  union line kept, 82 entries today.

### 2. Decision records and RFCs: hand-edited, generated or absent

- adr-tools (github.com/npryce/adr-tools `src/adr-list`, `src/_adr_generate_toc`; last commit
  2020-03-30): `adr list` is `find | grep | sort`; `generate toc` prints titles to stdout; status
  is each file's `## Status` section. adr-log (github.com/adr/adr-log): stdout, or `-i` injects the
  log between `<!-- adrlog -->` and `<!-- adrlogstop -->`, a committed generated index. MADR
  (adr.github.io/madr/): `status:` in front matter, no index prescribed.
- log4brains (github.com/thomvaill/log4brains; last commit 2024-12-17): the site lists the files;
  its `docs/adr/README.md` lists none ("Please use this link to browse them"). Its ADR 20201016:
  `NNNN-` names conflicted "when two developers have created a new ADR"; now `YYYYMMDD-slug`.
- PEPs (github.com/python/peps `pep_sphinx_extensions/pep_zero_generator/`): "Automatically create
  PEP 0 (the PEP index)" from each file's header at build; raises on "not a valid Status value";
  writes `api/peps.json`. Observed: no `pep-0000` file in `peps/`; peps.python.org serves the JSON.
- KEPs (github.com/kubernetes/enhancements): `kep.yaml` per KEP directory, `status:
  provisional|implementable|implemented|deferred|rejected|withdrawn|replaced`; `kepctl query --sig
  --status` will "Query the local filesystem"; `kepify` writes `keps.json`; kubernetes.dev/resources/keps
  tabulates the repo and warns of empty cells where "the KEP may not have updated information".
  CNCF, 2021-04-12 (cncf.io/blog/2021/04/12/enhancing-the-kubernetes-enhancements-process/):
  tracking left "manual updates to spreadsheets" for files, "via Git commits".
- Rust RFCs (github.com/rust-lang/rfcs `generate-book.py`): "auto-generates the mdBook SUMMARY.md
  file based on the layout on the filesystem" at deploy; no status in files; "Active RFC List" at
  rfcbot.rs. git-issue (github.com/dspinellis/git-issue `git-issue.sh`): a `tags` file per issue;
  `git issue list` is `find issues -name tags | xargs grep -Flx open`. Neither keeps an index.

### 3. Listings derived at read or build time

- Hugo (gohugo.io/templates/section/, gohugo.io/functions/collections/where/): `range .Pages`, and
  `where .Site.RegularPages "Params.color" "ne" nil`; Jekyll (jekyllrb.com/docs/collections/) loops.
- Obsidian Dataview (blacksmithgu.github.io/obsidian-dataview/): "a live index and query engine" over
  front matter (`TABLE ... FROM "<folder>" WHERE ...`). Dataview Serializer
  (github.com/dsebastien/obsidian-dataview-serializer) writes results into notes so they "appear on
  Obsidian Publish websites, which is not the case with the Dataview plugin". Inference: a read-time
  view reaches only readers inside the tool; materializing it makes an index file again.
- Terminal, observed: `grep -H -m1 '^status:' docs/findings/2*.md` lists 13 findings with status in
  0.05 s; one `awk` pass printing `file | status | H1` 0.04 s; a per-file shell loop 1.6 s. The
  `triage` skill step 2 already derives idea state from the files: the same grep for a final block
  finds 24 untriaged, 8 dated 2026-10-16, as the handoff says. No index column holds triage state.
- No agent tool found auto-loads a README index (`2026-10-16-handoff-split-default.md`): an index
  costs a read, a listing a grep. docs.github.com (about-readmes) names only `.github`, root, `docs`.

### 4. Git: experiment and hosting limits

Base patch applied with `git apply --3way` onto a project README holding prose, `## Index` and five
rows; files in `scratchpad/idx/in/`, results in `run/*.out`.

| Base change / project change | Default | `merge=union` |
|---|---|---|
| append a row / append a row | conflict | clean, both rows |
| edit one prose line / own rows replace the base's (decant) | clean | not run |
| append a row / own rows replace the base's | conflict | clean; all five base rows return |
| delete the last row / append after it | conflict | clean; the deleted row returns |
| delete a row / own rows replace the base's | conflict | not run |
| append at the end / insert in sorted middle | clean | not run |
| append above a marker line / append below it | clean | not run |

`git apply --3way` honoured a `merge=union` in a worktree-only `.gitattributes`; `--3way --union`
and `--3way --ours` (git-scm.com/docs/git-apply) gave the same and the project side, for every hunk
of the patch. Inference: date-named files sort last, so two new rows always meet at one place.

Hosting: GitLab issue 18830 (2017, closed 2023): GitLab "use[s] informations like 'union'" from
`.gitattributes` in MRs, not custom drivers. Bitbucket Cloud BCLOUD-20234 (2020, unresolved):
"doesn't allow the use of GIT native or custom merge drivers". GitHub: only RuboCop's 2016 line.

### 5. Hatchery and siblings, measured

- Row-only commits of all commits to each README: ideas 16 of 17, findings 13 of 14 (the other is
  the creating one), reference 3 of 5, `docs/design/README.md` 12 of 19, proposals 11 of 13. Files
  added and deleted in 101 commits: ideas 39 and 7, findings 13 and 0, proposals 14 and 0,
  decisions 16 and 0, reference 2 and 0, operator none.
- Replayed merges: `../regenesis` (50 merges): 3 of 5 base-update merges conflict on index READMEs;
  of 45 work merges none does, one conflicts on `docs/HANDOFF.md`, six on `FIDELITY_LEDGER.md`, a
  hand-edited table every pass updates (138 commits). `../broodforge`: all 3 base merges conflict on
  `docs/ideas/README.md` (the latest on six READMEs); its 2 work merges on none.
- Status in two places: Hatchery's 13 findings match their Status column, and each of the 13
  commits that changed a `status:` line touched the README. Siblings: 1 of 56 differs,
  `../regenesis` `2026-10-13-suite-duration.md`, `status: resolved (ADR 0011, 2026-10-14)` in the
  file and `open` in the row; resolving commit d2a28872 left the README alone, so a skill reading
  the Status column reports it open. The free-text value also fails an exact match. The `handoff`
  skill step 1 reads "the Open column", which the README does not have (it is `Status`).
  `docs/reference/README.md` repeats each file's `verified:` date in a Verified column.

## Result

| Practice | Who | Evidence on conflicts | Cost | Fit: the three queues | `reference/`, `design/` |
|---|---|---|---|---|---|
| No index; a command lists the directory | towncrier, scriv, reno, changesets, adr-tools, kepctl, git-issue | No shared file, so none by construction; RuboCop added fragments after union | Every reader runs a command; a forge shows names, not status | Fits the files: idea state is already derived this way; the hook's index check and the `finding` (step 3), `decant`, `handoff` and `triage` skills read or write rows today | Reference adds 2 files in 101 commits; little to gain |
| Index generated at build, not committed | PEP 0, Rust RFC book, KEP site, log4brains, Hugo, Jekyll | Nothing committed, nothing to conflict; PEP build rejects a bad Status | A build step and a place to view it | No build or site here; a script printing the listing is the terminal form | Many-writer decision repos all generate (PEPs, RFCs, KEPs); 16 ADRs here in 7 days |
| Generated index committed between markers | adr-log, Dataview Serializer | None found | Still a both-sides edit, resolved by regenerating (inference) | No: same conflict as a hand index | Possible for the ADR list |
| Status only in the item file | PEP header, `kep.yaml`, MADR, findings front matter | Drift where it lives twice: 1 of 56 sibling findings | A grep per query; free text breaks exact matching | Findings and ideas already carry it; the Status column is the second copy | The Verified column is a second `verified:` |
| Names that cannot collide | scriv, log4brains, reno, RuboCop | log4brains dropped `NNNN` after branch conflicts | None | Already dated slugs | `NNNN` ADR and proposal numbers are the collision log4brains named |
| `union` (attribute or `apply --union`) | RuboCop since 2016 | Experiment: clean | Returns deleted rows; imports base rows into a decanted index; GitHub (2016) and Bitbucket Cloud do not honour it | No: queue rows are deleted when done | Append-only ADR lists only |
| Marker line or sorted insertion | `docs/design/README.md` for decanted projects (proposal 0016) | Experiment: clean | Each side must own a section; dated rows meet at the end | No: the rows are each repository's own state (0016, critic objection 3) | Already used for the ADR list |

Inference: proposal 0016 makes the queue READMEs project-owned and unpatched, which ends the
base-merge conflicts measured above with or without an index; what an index still costs is the
parallel-work conflict (none in 47 sibling work merges so far; the ledger table shows the shape
once work runs in parallel), the second copy of status, and an edit to one file per add or delete.
The README counts bear the owner out for `reference/` and `docs/design/README.md` (prose in 2 of 5
and 7 of 19 commits); `design/proposals/README.md` (11 of 13 row-only) behaves like a queue index.

Not verified: GitHub's current web merge with `merge=union`; whether `reno cache` output is
committed; how GitHub renders a subdirectory README; whether replayed conflicts match the originals.

## Next

The architect's proposal on these directories, if the owner wants one: a listing per queue (command
or script), what the post-edit index check and the four skills named above do instead, and
whether `design/` numbering is in scope.
