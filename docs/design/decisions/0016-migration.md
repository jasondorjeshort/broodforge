# 0016, sibling: the migration, Hatchery first and then each sibling

Companion to [ADR 0016](0016-base-owned-and-project-owned-files.md). Order of implementation:
the coder brief, the tester brief, Hatchery's own migration with the subtree move last (the owner
may still veto the move), then each sibling at its next update. Sibling names are allowed here: a
decision is a closed record.

## 1. Coder brief (code, Hatchery)

`ownership.json` and `ownership.js` with its four flags ([the map](0016-ownership-map.md)); group
`O` and case `A1` in `test-hooks.js`; `imports.js`; in `policy.js` the `update.exclude` key, the
`docs/hatchery/*` default in `edit.anchorExempt`, the two `CLAUDE.md` caps beside `docCap` with the
`CLAUDE.md`-by-exact-name lift, and `--claude-check` ([the procedure](0016-update-procedure.md),
Checks); `post-edit.js` calling the parser for `CLAUDE.md`, `.claude/agreement/*.md` and
`.claude/project/*.md` and dropping its 150 constant; the guard's malformed-policy sentence.
Verify: `node .claude/hooks/test-hooks.js` green with `O` and `A` present; `--check` empty on the
real tree; `--command --ref HEAD` prints a line that O2 accepts.

## 2. Tester brief (skills, templates, rules; Hatchery)

- `update-from-hatchery`: the steps of [the procedure](0016-update-procedure.md); one mode; step 1's
  precondition is the relation, not where the skill was read from; the first-update order; the
  report's sections and their actions.
- `decant`: step 4 writes the project `CLAUDE.md` from the template below; step 5 sets `usage.pool`
  to `null` unless the machine has the switcher; step 9 keeps `docs/hatchery/` as shipped, empties
  `docs/design/decisions/` to its README, keeps `proposals/README.md`, writes
  `NNNN-decanted-from-hatchery.md` (`0001`) from the provenance template with the base line
  (`git rev-parse hatchery/main` at decant) and an empty departure table; no routing slot.
- `adopt-hatchery`: step 3's checkout list is the base-owned set with `docs/hatchery` and without
  `LICENSE`/`LICENSES` (the licence stays step 8's question; a decline writes `update.exclude`);
  step 6 writes the project `CLAUDE.md` from the template, folding the project's every-turn rules
  into `## Project rules` and each kept contradiction into a replacement file plus a departure row,
  no routing slot; step 7 copies no record by name, the skeleton (handoff, queue READMEs,
  `docs/design/README.md` with an empty "Decisions so far", `proposals/README.md`,
  `decisions/README.md`) only where absent; step 10 writes the provenance ADR from the template.
- The provenance template: `.claude/skills/adopt-hatchery/template-adr.md` generalised to both
  cases (title slot `Adopted from` or `Decanted from`, the base line, the licence decision, the
  departure table with its header row, the replaced sections), read by `decant` step 9 too.
- `adr` step 1: one test (no `hatchery` remote: `docs/hatchery/decisions/`, from `0016` up; else
  `docs/design/decisions/`, from `0001` or the next free number, proposals included); `triage` and
  `handoff` grep the same directory the test names.
- `.claude/rules/docs.md`: `paths:` gains `.claude/agreement/**` and `.claude/project/**`; the
  indexes paragraph says `docs/hatchery/` has its own three READMEs and a project never edits them;
  `docs/design/README.md lists every ADR` becomes "every ADR of the tree's own directory".
- Pusher check 7: the `--claude-check` line and its FAIL texts.
- Every `git show hatchery/main:<path>` in a skill, a template or an ADR's living line becomes
  `git show <sha>:<path>` after `S=$(git rev-parse hatchery/main)`, with a `finding` recording the
  MSYS rewrite first and the reference entry's Shells section citing it.
- `README.md`'s two prompts: the update prompt says the skill is read from `<sha>`.

The project `CLAUDE.md` template (at most 40 lines; `#` title, then exactly):

```
# <Project> working agreement

<Decanted from|Adopted from> Hatchery; provenance and departures: `docs/design/decisions/<NNNN>-<...>.md`.
Where work stands right now (coordinator state; subagents follow their brief, not this):

@docs/HANDOFF.md

## Phase

<one or two lines>

Branch: <name>

@.claude/agreement/this-agreement.md
@.claude/agreement/where-things-live.md
@.claude/agreement/delegation.md
@.claude/agreement/debug-interface.md
@.claude/agreement/major-decisions.md
@.claude/agreement/usage-gating.md
@.claude/agreement/model-routing.md
@.claude/agreement/one-review.md
@.claude/agreement/git.md
@.claude/agreement/licensing.md

## Project rules

<every-turn rules that contradict no section; a contradiction is a replacement file>
```

A replacement file `.claude/project/<section>.md`, one line is enough: "Replaces
`.claude/agreement/<section>.md` for this project: <the rule>. Decided in ADR <NNNN>." The
replaced path is in backticks so the loader does not import it.

## 3. Hatchery's own migration (the architect reviews, the tester executes)

1. `.claude/agreement/<section>.md`, ten files, each the `##` section of today's `CLAUDE.md`
   verbatim from its heading, with project values generalised: the thresholds are "the lines in
   `.claude/hooks/policy.json` (`node .claude/hooks/policy.js` prints them)"; the branch is "the
   `Branch:` line of `CLAUDE.md`"; the licence section says `LICENSE`, GPL-3.0-or-later as the base
   default, and cites `docs/hatchery/decisions/0001-licensing.md`; the decisions directory is "the
   tree's (`adr` skill, step 1)"; routing stays base text (the models are pinned in base-owned agent
   files). `this-agreement.md` adds that the agreement files are edited in Hatchery only, and that
   in a project a fix to a base-owned file is a departure: a table row and a carry-back idea, same
   sitting. The ten total at most 150 lines (`A1`).
2. Hatchery's `CLAUDE.md` to the template; its `## Project rules` holds nothing today. The hook
   warns and check 7 fails until both this step and step 1 are in the same commit.
3. The reference entry `docs/reference/agreement-loading.md` (moved into the subtree in step 4):
   the sentinel procedure and the compaction check, run once before the commit with the results
   dated `verified:`.
4. The subtree, last, after the owner has seen it: `git mv docs/design/decisions/<every NNNN-*.md>
   docs/hatchery/decisions/` and `git mv docs/design/decisions/README.md
   docs/hatchery/decisions/README.md` (the "Decisions so far" list moves into it; relative links
   between records survive); a fresh `docs/design/decisions/README.md` with the template and an
   empty list; `git mv docs/reference/claude-code-on-windows.md docs/hatchery/reference/` with a
   `README.md`; `docs/hatchery/README.md` (what the subtree is, one line per direct entry, the
   never-edit rule); `docs/design/README.md` keeps its directory table and points at the subtree
   for Hatchery's records; `docs/README.md` gains a `hatchery/` row; `docs/reference/README.md`
   loses the Windows row; this record and its three siblings move with the rest.
5. The citation pass, living files only (closed records keep their text under `anchorExempt`):
   `grep -rnE 'docs/design/decisions/(000[1-9]|001[0-9])-|docs/reference/claude-code-on-windows'
   --exclude-dir=.git .` minus `docs/design/proposals docs/findings docs/ideas` and the moved
   records themselves. Known sites: the hook headers and `POLICY_DOC` in `usage-gate.js`, the
   guard's deny texts, `policy.json`'s `$comment`, the `adopt-hatchery` skill and its template, the
   harness, `.claude/rules/docs.md` (the Windows pointer), `README.md`, `docs/README.md`, the
   proposals README's promoted rows and template line, `docs/design/README.md`. Each becomes
   `docs/hatchery/decisions/NNNN-...` or `docs/hatchery/reference/...`. The post-edit hook's
   missing-path warning is the check: none after the pass.
6. Verify: harness green; `node .claude/hooks/ownership.js --check` empty; `--claude-check` clean;
   `grep -rn "^<<<<<<<" .` empty; a `decant` dry run in a scratch clone produces a tree whose
   `--command` from Hatchery's HEAD is a no-op apply.

## 4. Each sibling, at its next update, in the procedure's first-update order

Common to all four: step 4 of the order writes `CLAUDE.md` to the template with its sections as
import lines or replacement files; the departure table from its base-owned edits (`git diff
--name-only <FROM> HEAD -- .claude`) and replaced sections; its copies of Hatchery's records
removed from `docs/design/decisions/` and `docs/reference/` (a project-edited copy's `git diff
<FROM> HEAD -- <path>` goes to a carry-back idea first), their rows dropped from
`docs/design/README.md` and `decisions/README.md`, a `hatchery/` row in `docs/README.md`; the
project-owned citations rewritten (`grep -rnE 'docs/design/decisions/(000[1-9]|001[0-9])-'
--exclude-dir=.claude`, read against the project's own numbers, since an adopted project's `0005`
may be its own); a supersession of a Hatchery record the project had written into its copy
(a different licence superseding 0001) becomes a citation line in the project's ADR.

| Sibling | Relation and `FROM` | Specific to it |
|---|---|---|
| `../broodforge` | decanted; `git merge-base HEAD <sha>` once (48a43e4 on 2026-10-17); writes `docs/design/decisions/0024-decanted-from-hatchery.md` with the line | seven edited base-owned files under `.claude/` become departure rows or are given up to base; four edited record copies under `docs/` go to carry-back ideas before removal; `CLAUDE.md` 138 lines with about one project line, so the template fits; three citations (2 in `CLAUDE.md`, 1 in `README.md`); its own ADRs 0020 to 0023 keep their numbers |
| `../regenesis` | adopted; base line in its adoption ADR; branch `master` | replaces `git.md` and `licensing.md` (`.claude/project/git.md`, `licensing.md`); its hooks paragraph to `docs/reference/`; two-machines and live-data rules stay in `## Project rules` (about 50 lines, then a trim or `edit.capExempt` naming `CLAUDE.md`); 13 edited base-owned files become rows; its own 0003 to 0018 and its citations of them are untouched; its copies of Hatchery's 0001, 0002 and the hook specs go |
| `../droplet-management` | adopted at 9d90716, never updated; base line in its adoption ADR | two citations (`CLAUDE.md`, the adoption ADR, the latter a closed record and left); `CLAUDE.md` 150 lines with about 76 project lines: the rules go to `## Project rules`, `docs/reference/` or `.claude/rules/`, or the lift; its own 0003 keeps its number |
| `../dystopo` | adopted; base line; patch mode already | carries none of the base's rule text, so most sections become replacement files and its Rules section stays; 11 edited base-owned files become rows; one citation in a closed record, left; its own 0020 to 0035 keep their numbers; updates cleanly only after the patch-update ideas in Hatchery's queue are fixed |

Each sibling's update commit carries the migration; its report names the departure rows written,
the excludes, every path dispositioned, and the `--report` sections. A sibling that refuses the
`CLAUDE.md` rewrite takes the lift (`edit.capExempt: ["CLAUDE.md"]`) and still needs the import
lines, since `--claude-check` has no lift for a section neither imported nor replaced.
