# 0014. Handoff layout: one file, one file per workstream, or a default with a switch

Date: 2026-10-16
Status: promoted to ADR 0014 (`../decisions/0014-handoff-layout.md`)

## Question

ADR 0002 made `docs/HANDOFF.md` one 40-line file with four fixed sections, imported by `CLAUDE.md`
into every session and subagent, rewritten by the `handoff` skill, capped by the post-edit hook and
the pusher. A downstream build-phase project (`../regenesis`, its ADR 0017, commit af2b8d27, the
owner's decision with no proposal or critic) replaced it with a near-static map that imports one
file per workstream from `docs/handoff/`, dropped the commit hash from the Updated line, and edits
the stream file in the same commit as the work. Its evidence: 438 commits in a week, 48 touching
the handoff; several parallel sessions each running `git pull --rebase --autostash` before every
push, conflicting on the handoff routinely (the owner's observation across machines; this clone's
reflog cannot confirm the rate, objection 10); a 15-minute suite per push, which the hash forced to run
twice because a handoff naming a commit can only be written after that commit lands. Hatchery's own
figures (counted 2026-10-16): 85 commits since 2026-10-11, 24 touching `docs/HANDOFF.md`, 10 of them
touching nothing else; one session at a time; no suite. The owner asks whether the split is the
better design and whether the base should adapt or reject it. The base is decanted into projects,
so the real question is what default a new project gets, what a project may vary without an
incompatible change, and what the base's tooling must accept. In ten years the chosen shape should
still be a file git merges and `CLAUDE.md` imports, with nothing running to keep it true.

## Constraints that matter here

- Every line of the handoff is loaded into every session and every subagent launch (ADR 0002,
  Session continuity); the 40-line cap is what makes the import affordable.
- The import is the one guaranteed read: no brief has to name it, and it survives compaction.
- Enforcement is a path check in three places: `.claude/hooks/post-edit.js` (`HANDOFF_CAP`, the
  `INDEX_EXEMPT` set), `pusher.md` check 7, and the harness cases `P02`, `G16`. ADR 0002 said a
  third enforcement point moves the numbers into a shared file; this proposal must not add one.
- `update-from-hatchery` keeps the project's handoff (project wins) and the base's `.claude/`
  (base wins). A project whose handoff shape the base tooling does not accept conflicts on every
  update and keeps its own skill copies; that is where `../regenesis` is today (its ADR 0017,
  Consequences, "Hatchery divergence").
- ADR 0012: an incompatible change to a promised surface is a major choice; ADR 0013 lists the
  surfaces and does not name the handoff. Three skills (`decant`, `adopt-hatchery`,
  `update-from-hatchery`) and the `finding`, `proposal`, `commit` and `brief` skills name the file.
- Living docs state the practice, not the sibling (`.claude/rules/docs.md`): whatever is adopted is
  written for a project that has never heard of the one that made it.
- The finding (`docs/findings/2026-10-16-handoff-layout-practices.md`) surveys how other projects
  avoid conflicts on shared status files. Every split it found answers parallel writers on the
  same lines; it found no single-writer project that split and no team that split and went back;
  changelog fragments assume a tool that reads at release time, which the handoff has not. Only
  Claude Code and Copilot CLI inline `@` imports; `merge=union` brings a deleted line back.
- Claude Code resolves a relative `@` import from the directory of the importing file, to four
  hops (memory docs; the critic's experiment, review claim 2). So the map at `docs/HANDOFF.md`
  writes `@handoff/<file>.md`; `@docs/handoff/<file>.md` loads `docs/docs/handoff/<file>.md`,
  which does not exist, and nothing reports it.

## Options

### A. Keep the single file as is

One 40-line file, four sections, `Updated: <date>, at commit <sha>, by <who>`, rewritten at
wrap-up and committed on its own because the sha is not known until the work commit exists.
Loaded per launch: at most 40 lines. Conflict surface: the whole file, every parallel session;
with one session, none. Staleness: the sha on the Updated line against `HEAD`; the line is odd in
that it names the commit before the one carrying it, so it is wrong by one the moment it lands.
Enforcement: unchanged. `decant`, `adopt-hatchery`, `update-from-hatchery`: unchanged. The
downstream project stays divergent: every base update conflicts on the handoff skill, the docs
rule, `post-edit.js`, `pusher.md` and the harness, and its patch keeps its own (ADR 0017), so the
base's fixes to those files never reach it cleanly. Not a fix for A: a `merge=union` attribute
on the file, because a deleted answer returns silently when the other side edits beside it (the
finding's `git merge-file` test).
Exit cost: nothing to leave; the next shape rewrites one file and the skill, as the downstream
project did in one commit of 27 files.
Moving parts: none new.

### B. The per-workstream split as made downstream

`docs/HANDOFF.md` is a map: title, coordinator-state line, two sentences of organisation, a
pointer to `docs/findings/README.md` for open findings, one bare `@handoff/<file>.md` line per
file (as made downstream the lines read `@docs/handoff/<file>.md`, which resolve to
`docs/docs/handoff/` and have loaded nothing since they landed; objection 1). `docs/handoff/`
holds `README.md` (format, index, not imported), two fixed files `live.md`
(deployed and pushed-not-deployed state) and `waiting.md` (questions for the owner), and one file
per workstream named for its source (a plan, an ADR rollout, an idea's Plan block). Each file:
`# <Stream>`, `Updated: <date> by <who>` without a hash, Now, Next, at most 40 lines. A session
edits only its stream's file, in the commit of the work; opening a stream adds the file, its `@`
line and its README row; closing deletes all three.
Loaded per launch: the map plus every stream file; downstream today six files, 91 lines, against
40 before; the per-file cap bounds each, nothing bounds the count. A new decanted project opens
with a map, `waiting.md`, `live.md` saying "nothing" and one stream saying "decanting": four files
to say what one line said. Conflict surface: a stream file conflicts only when two sessions share
a stream, which is the signal that matters; the map conflicts on open and close; `live.md` is the
new hot spot (ADR 0017 names it). Staleness: per file, `git log -1 --format=%cs -- <file>`; a
stream whose date has not moved in a week is abandoned (deferred to triage downstream).
Enforcement that moves: `post-edit.js` caps `docs/handoff/*.md` at 40 (the `isHandoff` test in
af2b8d27), pusher check 7 adds the glob to its loop, harness cases `P02b`, `P02c`; the index check
still wants every stream named in `docs/handoff/README.md`, which is why opening touches three
files. `decant` step 6 and `adopt-hatchery` step 7 create the directory and the four files;
`update-from-hatchery` adds `docs/handoff/` to project-wins and `docs/handoff/*.md` to its `wc -l`
line (a bare glob with no match fails in bash without `nullglob`; the line must go through
`git ls-files`). `finding` and `proposal` point at `waiting.md` and the findings index instead of
sections. A stream file whose `@` line is missing, or whose `@` line does not resolve, is silently
never loaded; ADR 0017 defers the check that would catch the first and nothing catches the second.
Claude Code only: a reader that does not inline `@` imports sees a map of inert lines.
Exit cost: fold every stream back into one file and delete the directory; the skill and rule
revert to A; one commit, no data lost because the content is prose. The import mechanism is the
risk: if nested imports stop resolving, every project on B has an empty handoff until it folds.
Moving parts: a directory with a format README and two fixed files in every project from day one;
an open-and-close discipline the per-file cap does not enforce; a map whose lines must stay bare;
a dependence on import depth and on the code-span rule (Claims 1 to 3).

### C. The single file, hash dropped, edit folded into the work commit

A's file with `Updated: <date> by <who>`; the sha comes from `git log -1 --format=%h --
docs/HANDOFF.md`, which is the same information and never off by one. The `handoff` skill step 5
and the `commit` skill say: when the session goes on, the handoff edit rides in the next work
commit (the runner's brief lists it); only the wrap-up handoff, which has nothing to ride with, is
its own commit. Loaded per launch: at most 40 lines. Conflict surface: unchanged from A for
parallel sessions; the downstream ADR 0017 is right that this removes the second push and keeps
the rebase conflicts. For Hatchery the gain is the honest line and fewer handoff-only commits (at
most the 10 counted, of which the wrap-ups remain). Staleness: `git log -1 -- docs/HANDOFF.md`
against `HEAD`, plus the date. Enforcement: unchanged. `decant`, `adopt-hatchery`: unchanged (they
call the skill). `update-from-hatchery`: unchanged. The downstream divergence stays as in A.
Exit cost: as A.
Moving parts: none new; one line of the template changes and one sentence in two skills.

### D. The single file by default; the split as a documented shape a project switches to at a trigger

Default: C. The split: B refined, documented as the second shape in a sibling file of the
`handoff` skill (`split.md`), which the skill follows when `docs/handoff/` exists (no policy key,
no flag; the tree says which shape it is in, and the base ships nothing under that path, so no
project is in the split until it puts a file there). The base tooling accepts both shapes
unconditionally, which costs nothing while the directory is absent. As revised after review
(objections 2 to 6, 9); the first draft is in the review's quotations.

The split shape, as the base documents it: the map as in B, aim under 15 lines, its stream lines
spelled `@handoff/<file>.md`; no `README.md` in `docs/handoff/` (the format is the skill's, the
rows are the map's, and a README there would be a mixed base-and-project file); no fixed files
(every fixed file is a hot spot by construction; the owner's questions already have one file per
item in `docs/ideas/` `Question:` blocks, and the map points there; a stream may carry a
`Waiting:` line for a question not yet filed); one file per stream, Now and Next, `Updated:
<date> by <who>`, cap 40 and aim under 15. Open findings are the status column of
`docs/findings/README.md`, pointed to from the map. A project that needs a fixed file (deployed
state, say) adds it by its own ADR, imports it like a stream, and owns its conflicts. The map is
the index of the directory: `post-edit.js` reads the `@handoff/<name>` lines of `docs/HANDOFF.md`
as the index for `docs/handoff/*.md`, so opening a stream touches two files and a stream without
its line draws a warning in the same turn. The chain is checked for resolution, not presence: the
post-edit hook and pusher check 7 resolve every `@` line of `CLAUDE.md`, `docs/HANDOFF.md` and
`docs/handoff/*.md` from the importing file's directory and name the resolved path of a missing
target; the human check is `/context`, which lists each loaded file under Memory files. A total
cap of 120 lines (three old handoffs) over the map plus the streams sits in the same two
enforcement points as the per-file cap, so ADR 0002's "third point" rule is not tripped.

The trigger to switch, in what the base emits: the pusher has stopped on behind or diverged
(check 3) twice in a working week, or a merge or rebase has conflicted on `docs/HANDOFF.md` twice;
the coordinator notes each in the handoff's Now, and ADR 0014 carries the trigger as a Deferred
item the `triage` scan reads. The base has no parallel-session procedure today; that is a gap,
named as such and deferred, with one line in `commit` for the case the pusher stops on diverged.
The reverse trigger: one session again for a month; fold back. Switching touches:
`docs/HANDOFF.md` rewritten as the map; one stream file per item in flight; a project ADR of one
page (its own numbering, 0020 and up) that cites the trigger evidence, as ADR 0017 does. Nothing
in `.claude/` and nothing in the policy file, because the base already accepts the shape; a base
update is clean in both shapes. Folding back is the inverse, one commit.

Loaded per launch: 40 lines in the default; in the split, at most 120. Conflict surface: as C by
default, as B after the switch, and the switch happens exactly when the conflict surface starts
to matter. Staleness: the `handoff` skill's step 1 gathers `git log -1 --format='%h %cs' --
docs/HANDOFF.md docs/handoff/` in either shape; in the split, triage may list a stream whose file
is older than the working week (ADR 0017's deferred item, written into `triage` as one line when
the first stale stream is found by hand). Enforcement that moves: `post-edit.js` caps
`docs/handoff/*.md` at 40 and the total at 120, treats the map's `@` lines as the directory's
index, and resolves `@` lines; pusher check 7 gains the glob, the total and the resolution loop;
harness cases for each behaviour in their own scratch root; `.claude/rules/docs.md` states the
caps, the README exception and the import rule; the hook specs table gains the rows. `decant` and
`adopt-hatchery`: unchanged (the default is the single file; their verify lines stay `wc -l
CLAUDE.md docs/HANDOFF.md`). `update-from-hatchery`: `docs/handoff/` joins project-wins, and the
post-merge cap check runs the pusher's check 7 lines rather than a bare glob. `finding`,
`proposal`, `brief`, `commit`: they say "the handoff's Waiting on user" or "the handoff" and the
`handoff` skill alone says where that is in each shape; no two-shape parentheses. The downstream
project's next update then brings skills and hooks that accept its shape; its divergence on base
files closes, and only its fixed files and its spelling fix remain its own work.
Exit cost: dropping the split shape later deletes `split.md`, one sentence in the skill, the
`docs/handoff/` branches in the hook, the harness cases and the glob in check 7; a project on the
split folds back as in B. Dropping the single file later is B's path. Nothing migrates but prose.
Moving parts: one more skill file; path tests and an `@`-line resolver in a hook and a check that
already test paths; no new file in a project that has not switched.

Also considered, not developed: one file per session (sessions switch streams; ADR 0017's
rejection holds); no handoff, `git log` plus `docs/ideas/` (loses the guaranteed read); a glob
import `@docs/handoff/*.md` with no map (imports take paths, not globs, Claim 4); one file with a
section per stream (hunks three lines apart still collide in a 40-line file, and the header is
shared). Promising the split's import depth by making `CLAUDE.md` import the stream files directly
(no map): keeps depth at one but puts a project's stream list into the base-owned half of
`CLAUDE.md`, which every update then fights over.

## Recommendation

D. The downstream pain had three causes, and Hatchery has one of them: the hash on the Updated
line, which forces a separate commit and is wrong by one anyway. Parallel sessions and a per-push
suite are the other two, and they are properties of a project in its build phase, not of the base
or of a freshly decanted project with one stream called "decanting". So the default stays one file
(C's fix included) and the base commits to accepting the split, so that a project reaches for it at
a stated trigger without leaving the base. The single most useful carry-back from af2b8d27 is the
tooling, not the default; the decant and adopt changes do not carry back, and the spelling does
not either. What the base gains today, before it has a parallel-session model of its own: the
sibling's divergence on five base files closes at its next update, so the base's fixes to the
hook, the pusher, the harness, the rule and the skill reach it; an import-resolution check that
guards the base's own `@docs/HANDOFF.md` line as much as a map; and the honest Updated line.
Waiting for the base to run parallel sessions before accepting a shape a sibling already needs
would be design behind evidence, not ahead of it. First step: the ADR, then a `tester` applies it
from the ADR alone, and the next sibling update is the test of the claim that its divergence
closes.

The promise (ADR 0013). The handoff's content is not a surface: no program reads it and no sibling
reads another's. The paths the base tooling caps and the import lines are: a project that switched
shapes relies on the next base update still capping `docs/handoff/*.md`, still reading the map as
the directory's index and still importing `@docs/HANDOFF.md`. ADR 0013 gains one row, "The
handoff paths: `@docs/HANDOFF.md` in `CLAUDE.md`; `docs/HANDOFF.md` and `docs/handoff/*.md` capped
at 40 each and 120 together; the map's `@handoff/<file>.md` lines as the directory's index;
carried by `CLAUDE.md`, `post-edit.js`, `pusher.md`; readers: every project in either shape;
record: ADR 0014". Removing either shape from the tooling is then a major choice, which is the
right weight: it strands projects. Adding a shape is free. ADR 0013 is still `proposed` and so
takes the row cheaply. A project switching shapes is not a surface change under this row; it is a
project decision recorded in its own ADR.

## What would change this recommendation

- The finding shows single-writer projects also adopt one-file-per-item status files for a reason
  other than conflicts (say, the context cost of an unrelated stream): then B becomes the default.
- Claim 1 fails (nested imports do not resolve at all, or Claude Code stops inlining them): the
  split shape is dead and is dropped from the skill; C stands; every project on the split folds.
  Claim 2 failed as written (imports resolve relative to the importing file): the outcome is a
  spelling, `@handoff/<file>.md`, and a resolution check, not a dead mechanism; the downstream
  project is told its streams have not been loading. Corrected in place.
- Hatchery itself takes a second parallel coordinator (the sibling sync, a scheduled design pass
  that commits): then Hatchery crosses its own trigger and switches, and the default is re-asked.
- A split project hits the 120-line total within a month of switching and closes nothing: the
  context cost is real and the shape, not the cap, is re-asked.
- Three decanted projects switch within their first month: the default was wrong; move to B.

## Claims to verify

1. Claude Code resolves `@` imports inside an imported file, to a depth of at least two (the
   memory docs say four hops; the first draft said five). Check: https://code.claude.com/docs/en/memory,
   "imports"; and in `../regenesis`, `/memory` lists `docs/handoff/*.md` as loaded, or a fresh
   session states the content of `docs/handoff/live.md` without reading it. Verified by the
   critic's experiment: depth two loads.
2. A relative import path inside `docs/HANDOFF.md` such as `@docs/handoff/live.md` resolves from
   the project root, not from `docs/`. Failed: the docs say relative paths resolve relative to the
   file containing the import, and the critic's experiment (Claude Code 2.1.296) loaded
   `docs/docs/b.md` for `@docs/b.md` written in `docs/a.md`, with no warning anywhere. The
   downstream map has loaded nothing since it landed (its one later transcript carries no stream
   header and no stream text). The correct spelling from `docs/HANDOFF.md` is `@handoff/<file>.md`.
3. An `@` inside a code span or fenced block is not resolved (memory docs), so the prose lines in
   the map and the backticked paths in this proposal and the skills import nothing.
4. Imports take a path, not a glob; `@docs/handoff/*.md` imports nothing. Memory docs.
5. Subagents receive `CLAUDE.md` with its imports resolved on every launch (ADR 0002 cites the
   sub-agents and memory docs); the per-launch cost in B and D-split is therefore the sum of the
   stream files. Check: a subagent asked for the handoff content with no Read tool.
6. `post-edit.js` index check: a `docs/handoff/x.md` not named in `docs/handoff/README.md` draws
   the "not listed" warning today (hook specs, the index row), which is why af2b8d27 touches three
   files per stream; D's one `if` redirects it to the map. Check: `0002-hook-specs.md` line 185
   and `post-edit.js`.
7. In bash without `nullglob`, `wc -l docs/handoff/*.md` with no such directory prints an error and
   exits non-zero; af2b8d27's verify lines in `decant`, `adopt-hatchery` and `update-from-hatchery`
   would fail in a project that has not switched. Check: run it in Hatchery.
8. Hatchery's counts: 85 commits since 2026-10-11, 24 touching `docs/HANDOFF.md`, 10 touching only
   it (the brief said 16; the critic recounts, `git log --since=2026-10-11 -- docs/HANDOFF.md`).
   Downstream's 438 and 48 are its ADR's figures. Verified exactly; downstream 439 and 49 by the
   day of the review.
9. A rebase conflict on a 40-line file whose header and top bullets both change in two sessions
   is routine rather than rare: ADR 0017's claim, plausible from git's three-line hunk context.
   Not verifiable from this clone (5 rebase starts against 86 pulls in the reflog); taken as the
   owner's observation across several machines.

Claims 3 to 7 verified by the critic as stated (code spans skipped; no glob or directory import;
subagents receive resolved imports, except the built-in Explore and Plan agents; the index check
reads `docs/handoff/README.md`; the bare glob exits 1 and `git ls-files` exits 0). Unverified by
anyone: whether nested imports re-expand after compaction; the docs say only that the root
`CLAUDE.md` is re-read. Carried as unverified in ADR 0014.

## Critic's objections and responses

1. Blocking: claim 2 fails; the split as spelled loads nothing. Accepted. The spelling is
   `@handoff/<file>.md` throughout (B, D, the constraints); the experiment is recorded in claim 2;
   the "what would change" bullet now says the outcome is a spelling and a check, not a dead
   mechanism. Telling the downstream project is the coordinator's, outside this record; ADR 0014
   says how it migrates and that its carry-back item would have carried the defect.
2. Serious: a two-link chain with no agent-observable check. Accepted with change: the check
   resolves each `@` line of `CLAUDE.md`, `docs/HANDOFF.md` and `docs/handoff/*.md` against the
   importing file's directory and names the resolved path when it is absent; it runs in the
   post-edit hook (the same turn) and in pusher check 7 (fails the push, which the critic did not
   ask for and which is the fail-closed point the hook cannot be); a harness case feeds a map
   holding `@docs/handoff/x.md` beside an existing `docs/handoff/x.md` and expects the warning
   naming `docs/docs/handoff/x.md`; the `handoff` skill states the `/context` check. The base
   gains the check on its own `@docs/HANDOFF.md` line.
3. Serious: no total cap. Accepted: 120 lines over `docs/HANDOFF.md` plus `docs/handoff/*.md`, in
   the hook and check 7 only; the six-streams bullet became the 120-line bullet.
4. Serious: the trigger cannot fire in the base's model and nothing reads it. Accepted with change:
   the trigger is restated in what the pusher emits (stopped on behind or diverged twice in a
   working week, or a merge or rebase conflicted on `docs/HANDOFF.md` twice), carried as a
   Deferred item in ADR 0014 for the scan, with the evidence noted in the handoff's Now; the base's
   lack of a parallel-session procedure is named as a gap and deferred with its own trigger; the
   `commit` skill gains one line for the pusher stopping on diverged (the committing agent
   rebases, a handoff conflict is resolved by rewriting the file, a fresh pusher follows). On the
   critic's moderate confidence about documenting the split at all: kept, for the three gains the
   recommendation names; the alternative leaves the sibling resolving five base files by hand on
   every update, which is the cost the proposal exists to remove.
5. Serious: `docs/handoff/README.md` is a mixed file and shipping it flips every project. Accepted
   with change, the critic's third option: no README in `docs/handoff/`; the format lives in
   `.claude/skills/handoff/split.md`; the map is the index; `.claude/rules/docs.md` exempts the
   directory from the per-directory README rule; the base ships nothing under `docs/handoff/`.
6. Minor: `waiting.md` is a hot spot by design and duplicates the ideas queue. Accepted with
   change: the base's split shape has no fixed files; the owner's questions are `docs/ideas/`
   `Question:` blocks (one file per item, as the TRAVEL case) plus a `Waiting:` line in the stream
   that raised one; a project that wants a fixed file adds it by its own ADR and owns the hot spot.
7. Minor: five hops. Accepted; four, corrected in claim 1 and the constraints.
8. Minor: compaction unverified for the second hop. Accepted: marked unverified here and in ADR
   0014's Context, with a Deferred item naming the check (a split project's first session after
   `/compact` asks for a stream-only token).
9. Minor: two-shape prose in seven skills drifts. Accepted: the skills say "the handoff's Waiting
   on user" or "the handoff"; the `handoff` skill alone says where that is per shape; the split's
   procedure is `split.md`, loaded when `docs/handoff/` exists.
10. Minor: claim 9 attributed as fact. Accepted; "the owner's observation" in the Question and in
    claim 9.

No objection rejected. The critic's recommendation (D with the corrections) is what ADR 0014
records.
