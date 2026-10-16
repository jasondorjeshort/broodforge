# Critic review of 0014: handoff layout

Date: 2026-10-16
Reviewer: critic (Fable 5.1)
Read: `CLAUDE.md`, the proposal, ADRs 0002 (layout and hook specs), 0012, 0013, `post-edit.js`,
`pusher.md`, the `handoff` and `update-from-hatchery` skills, the finding
`docs/findings/2026-10-16-handoff-layout-practices.md`, and the downstream record: its ADR 0017, its
`docs/HANDOFF.md`, every file in its `docs/handoff/`, its `post-edit.js`, pusher and skills at af2b8d27.

## Claims, verified

1. Nested imports resolve. Confirmed by experiment (below): `CLAUDE.md` to `docs/a.md` to a second file
   loaded, depth two. The docs (code.claude.com/docs/en/memory, "Import additional files", fetched
   2026-10-16) say "a maximum depth of four hops", not five; the proposal and ADR 0017 both say five.
2. Relative path inside `docs/HANDOFF.md` resolves from the project root. **Fails.** The docs state:
   "Relative paths resolve relative to the file containing the import, not the working directory."
   Experiment in the scratchpad, Claude Code 2.1.296, `claude -p --model haiku`, no tools used:
   `CLAUDE.md` holds `@docs/a.md`; `docs/a.md` holds `@docs/b.md` and `@c.md`; `docs/b.md` carries
   MANGO, `docs/docs/b.md` carries QUARTZ, `docs/c.md` carries COBALT. The session reported ZEBRA (a),
   QUARTZ (`docs/docs/b.md`) and COBALT (`docs/c.md`); MANGO never appeared. Its transcript shows one
   `Contents of <path> (` header per loaded file, including `docs\docs\b.md`. So a map at
   `docs/HANDOFF.md` that writes `@docs/handoff/live.md` loads `docs/docs/handoff/live.md`, which does
   not exist, and nothing is reported: no warning in the `-p` output, no line in the transcript.
   Downstream in practice: the one session launched after af2b8d27 landed (transcript
   `8ed4c14a`, 2026-10-16 12:03Z; the commit is 11:23Z) carries exactly four `Contents of` headers
   (MEMORY.md, rules README, CLAUDE.md, `docs\HANDOFF.md`) and no `docs\handoff\*.md`; the string
   `LIVE: 2785d79` from its `live.md` occurs zero times in the whole transcript. The downstream
   handoff has loaded nothing since the split landed. The correct spelling from `docs/HANDOFF.md` is
   `@handoff/<file>.md` (the experiment's `@c.md` case).
3. Code spans and fences are skipped. Confirmed from the docs page, same section.
4. Globs: confirmed by experiment that `@docs/glob*.md` and `@docs/` load nothing (a sibling plain
   import in the same file loaded), and the docs document no glob or directory form.
5. Subagents receive the resolved imports. Confirmed by observation: this review was written by a
   subagent whose context carries `docs/HANDOFF.md` under a `Contents of ... (project instructions)`
   header with no Read. The sub-agents page adds that the built-in Explore and Plan agents skip it.
6. Index check. Confirmed in `post-edit.js` lines 79 to 89: for `docs/handoff/x.md` the loop runs one
   depth and reads only `docs/handoff/README.md`; a missing row draws "is not listed". Hence three
   files per stream downstream.
7. Bare glob. Confirmed: in Hatchery `wc -l docs/handoff/*.md` prints
   `wc: 'docs/handoff/*.md': No such file or directory`, exit 1. `git ls-files -- 'docs/handoff/*.md'`
   prints nothing, exit 0. Downstream `decant` step 11, `adopt-hatchery` step 10 and
   `update-from-hatchery` step 6 all use the bare form.
8. Counts. Confirmed exactly: 85 commits since 2026-10-11, 24 touching `docs/HANDOFF.md`, 10 touching
   nothing else. Downstream, the week to 2026-10-16: 439 commits, 49 touching the handoff (its ADR says
   438 and 48, written before its own commit); eight committer names.
9. Routine rebase conflicts. Not verifiable from this clone: conflicts leave no record, and the local
   reflog shows 5 `rebase (start)` entries against 86 `pull` entries, so most pulls on this machine
   fast-forwarded. Taken as the owner's observation across several machines; the proposal should say so.

Not verified by anyone: whether nested imports re-expand after compaction (the docs say only that the
project-root `CLAUDE.md` is re-read); the finding could not either.

## Where the finding changes an option's standing

- Its inference holds: every account of a split names parallel writers on the same lines; the
  single-writer harness kept one progress file; it found no single-writer project that split and no
  team that split and went back. So the proposal's first "what would change" bullet does not fire and
  B does not become the default. D's reverse trigger has no precedent to lean on.
- Pattern 4 (claim files, directory near-empty at rest) and the TRAVEL case (one file per owner item,
  cleared by `git rm`) both avoid fixed shared files; ADR 0017 and D keep two or one. See objection 6.
- The `git merge-file --union` test (a deleted answer returns silently) rules out the one cheap fix for
  shape A that the proposal did not list; worth one line in A so nobody proposes it later.
- Only Claude Code and Copilot CLI inline `@` imports; a Codex or Cursor reader sees a map of inert
  lines. The default being the single file covers decanted projects; the split should say it is
  Claude-Code-only.
- Four hops, not five (claim 1).

## Mechanisms that refuse, deny or fail closed

- Pusher check 7 on a stream file over 40 lines: fails the push with `FAIL <file>: N lines, cap 40`;
  lift is a trim or split by whoever edits the file, from the working tree, mid-session; the pusher text
  says "the fix is a split or a trim, not a cap change". Documented.
- `post-edit.js` cap and index warnings: advisory, nothing to lift.
- `update-from-hatchery` step 6 line counts: reported as failures, not refusals.
- The import chain itself fails open and silent: a wrong path, a fifth hop, a renamed file, or a
  reader that does not inline imports leaves every session with an empty handoff and no message. No
  mechanism in the proposal detects it; the "one `if`" tests presence of an `@` line, not that it
  resolves. That is objection 2.

## Objections

1. **Blocking. Claim 2 fails; B and the split shape of D are misspelled, and the downstream shape has
   loaded nothing since it landed.** Evidence: claim 2 above. The proposal's own exit for this case
   (line 197: "C is the recommendation") is wrong too, because the outcome is a spelling fix, not a
   dead mechanism. Resolves: every `@docs/handoff/<file>.md` in the proposal becomes
   `@handoff/<file>.md` with one sentence saying why (relative to the importing file); the response to
   this objection records the experiment; the coordinator tells the downstream project today, outside
   this loop, that its map, README, docs rule and `handoff` skill must change the spelling, and that its
   `hatchery.md` stream item "carry the handoff split back" would carry the defect into the base.
2. **Serious. The guaranteed read becomes a two-link chain with no agent-observable check.** ADR 0002
   chose the import because it "needs no code and no state"; the split adds a second hop, a relative
   resolution rule and a four-hop limit, and the failure is silent (above). D's map-index `if` would
   have passed the downstream map today. Resolves: the check resolves each `@` target of
   `docs/HANDOFF.md` against `docs/` (the directory of the importing file, as Claude Code does) and warns
   when the file is absent, with the resolved path in the message; one harness case feeds a map holding
   `@docs/handoff/x.md` beside an existing `docs/handoff/x.md` and expects the warning naming
   `docs/docs/handoff/x.md`; the `handoff` skill states the one human check (`/context` lists the
   stream files under Memory files) and the one agent check (the harness case). Line 88 of the
   proposal should have led here.
3. **Serious. No total cap.** ADR 0002's affordability argument is about the sum loaded per launch; D
   caps per file and bounds the count by "aim under 15" and a close discipline. Downstream today: map 13
   lines plus 79 in six files, 92 per launch against 40, with one week of streams. Resolves: pusher check
   7 and `post-edit.js` also sum `docs/HANDOFF.md` plus `docs/handoff/*.md` against one total (120 is
   three old handoffs); same two enforcement points, so ADR 0002's "third point" rule is not tripped.
   The "what would change" bullet about six streams for a month then has a mechanism instead of a watch.
4. **Serious. The switch trigger cannot fire in the base's own model, and nothing reads it.** The
   pusher stops when the branch is behind or diverged and never rebases (check 3), so "the pre-push
   rebase has conflicted on `docs/HANDOFF.md` twice" describes the downstream project's process, not the
   base's; the base has no parallel-session procedure at all, which the proposal should name as the gap
   it is. "A second coordinator session commits in the same working week" is not visible in git (the
   trailer names the model, not the session). The reverse trigger "one session for a month" is equally
   unobservable. Resolves: state the trigger in what the base emits ("the pusher has stopped on behind or
   diverged twice in a week, or a merge or rebase has conflicted on `docs/HANDOFF.md`"), write it as a
   Deferred item with trigger in ADR 0014 so the `triage` scan reads it, and say in one line what a
   base project does when the pusher stops on diverged (today: nothing is written).
5. **Serious. `docs/handoff/README.md` is a mixed base-and-project file, and shipping it would flip
   every project into the split shape.** D makes the map the index yet keeps a README holding the format
   plus "a project adds fixed files ... and lists them in its own README": format is the base's, rows are
   the project's, exactly the `docs/ideas/README.md` problem the downstream project is already stuck on
   (its `hatchery.md` stream, item 3). And since D detects the shape by the existence of `docs/handoff/`,
   the base can never put a template README there. Resolves: the format lives in the `handoff` skill as a
   second file beside `SKILL.md` (templates beside skills is the pattern), `docs/handoff/README.md` is a
   two-line project-owned index that the map supersedes, or the docs rule exempts `docs/handoff/` from the
   per-directory README rule and the directory ships no README at all. Pick one and say which.
6. **Minor. `waiting.md` is a shared hot spot by design and duplicates the ideas queue.** Every stream
   that raises an owner question edits one fixed file; ADR 0017 already names `live.md` as the next
   conflict and the finding's union test shows a deleted answer returning. The ideas queue already holds
   `Question:` blocks with blank `Answer:` for the owner (ADR 0007), so the owner has two places to look
   either way. Resolves: state the trigger (the first conflict on `waiting.md`) and the fallback (a
   `## Waiting` section per stream, the map pointing at `docs/ideas/` Question blocks), or argue in one
   sentence why one file for the owner is worth the hot spot.
7. **Minor. Factual: "five hops" in claim 1 and in ADR 0017; the docs say four.** Resolves: correct in
   place; the downstream project is told with objection 1.
8. **Minor. Compaction is unverified for the second hop.** The constraints list "survives compaction"
   as a property of the import; the docs confirm only that the root `CLAUDE.md` is re-read. Resolves:
   mark it unverified in the ADR, or test it once (`/compact` in a scratch session, then ask for a nested
   token) and record the result in the finding.
9. **Minor. Two-shape prose in seven skills is inventory prose that drifts.** "or
   `docs/handoff/waiting.md` in the split shape" in `finding`, `proposal`, `commit`, `brief` and the
   rule is the kind of enumerated branching `.claude/rules/docs.md` warns against; every later edit to
   one skill must remember the parenthesis. Resolves: the skills say "the handoff's Waiting on user" and
   the `handoff` skill alone defines where that is in each shape; the split's procedure sits in a sibling
   file the skill loads when `docs/handoff/` exists.
10. **Minor. Claim 9 is attributed as fact.** The 438/48 figures hold, but the conflict rate is the
    owner's report; this clone's reflog cannot confirm it (5 rebase starts). Resolves: write "the
    owner's observation" in the Question paragraph.

## The strongest case against D, and for the alternatives

For B as made (one shape, everywhere): the base's tooling and skills then describe one layout, not two,
and the finding shows no project that regretted a split. Against it: the finding also shows no
single-writer project that made one, a fresh decant pays four files and a directory to say
"decanting", every subagent launch pays the sum of the streams, and the shape as made loads nothing
today, which no one noticed in a day of use because the failure is silent. One shape bought at the
price of a second mechanism is not simpler. For C alone (reject the split; only fix the hash): the base
has never run two sessions, its pusher refuses to push when behind, so documenting a shape for a
parallel model the base does not have is design ahead of evidence; the downstream project can keep its
divergence under the update skill's existing exception ("the project has recorded a reason to differ").
Against C alone: the exception is a hand resolution on every base update for five `.claude/` files,
which is the divergence the proposal rightly wants closed, and the tooling half of D (two globs and a
resolving index check) costs ten lines while the directory is absent. The honest middle: tooling
accepts both shapes unconditionally; the common-path skill text stays single-shape; the split's
procedure is one sibling file in the `handoff` skill; the trigger is one the base can observe.

## Pre-mortem

Ten years on, the regret is a year in which every session of a split project started with an empty
handoff and nobody knew: Claude Code had changed how it resolved or limited nested imports, or the
project had moved to a tool that reads `CLAUDE.md` but not `@` lines, and the map of bare lines was
inert prose. Sessions asked the owner what was in flight, briefs grew to carry what the handoff had
carried, and the shape was blamed when the mechanism had failed. The line that should have caught it is
88 to 89: "A stream file whose `@` line is missing is silently never loaded; ADR 0017 defers the check",
followed at 124 to 129 by a check that tests the presence of a line rather than that it resolves. The
proposal was written on the day the downstream project was already in that state, with the `@` lines
present and the files present, and its proposed detector would have passed. The question paragraph's
own test, "a file git merges and `CLAUDE.md` imports, with nothing running to keep it true", was the
right test; D accepted a second import hop without asking what keeps the second hop true.

## Recommendation

Adopt D with the corrections above, not as written: the import spelling is `@handoff/<file>.md`; the
map-index check resolves targets the way Claude Code does and a harness case proves it; a total cap
joins the per-file cap in the two existing enforcement points; the trigger is restated in terms the
pusher emits and recorded as a Deferred item the scan reads; the split's format and procedure live in
the `handoff` skill, not in a README the base cannot ship; and the downstream project is told today
that its handoff has not loaded since 07:23 and that its carry-back would bring the defect with it.
Confidence: high on claim 2 and the downstream state (a controlled experiment plus a transcript with no
stream headers and no stream text); high that the default stays the single file with the hash dropped
(the finding and Hatchery's own 10 handoff-only commits both point there); moderate on documenting the
split in the base at all rather than leaving it to a project ADR, because the base has no parallel
model to attach the trigger to yet, which is objection 4's real content.
