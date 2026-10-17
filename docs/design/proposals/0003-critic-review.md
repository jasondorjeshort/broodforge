# Critic review of 0003: Wave workflow for the build phase

Date: 2026-10-13
Reviewer: critic (Opus 5.5)
Evidence: the downstream build-phase project the proposal draws on (Dystopo), read at its commit
`d6e3b86`: its `CLAUDE.md`, `docs/TASKS.md`, the dated log in `docs/PLANNING.md`, its `wave`
skill, ADRs 0020, 0021, 0026, 0027 and 0028, its findings, `gh pr list` on its repository. Base
state read at `f747c1f` (ADRs 0006 and 0007 are newer than the proposal).

## Part 1. Claim verification

| # | Claim in proposal | Verdict | Evidence |
|---|---|---|---|
| 1 | Twelve waves, 83 tasks, 78 PRs between 2026-10-01 and 2026-10-11 | Partly wrong | 78 PRs merged before 2026-10-12 (`gh pr list --state merged`): confirmed. First commit 2026-10-09, not 10-01. Wave 12 was launched 10-10 in two halves and its T75 and T77 merged 10-12 and 10-13, so eleven waves were closed by 10-11. |
| 2 | Long paths block `git worktree remove` on Windows | Confirmed | Finding `2026-10-11-windows-worktrees-long-paths.md` here; downstream `2026-10-12-windows-long-path-delete.md`. |
| 3 | Hooks fire inside subagents, so `git-guard` rule 3 would block an implementer's push | Confirmed, now moot | 0002 critic review claim 8; and ADR 0006 made who commits and pushes policy (`policy.examples/task-branches.json` encodes the downstream E1 to E5, A1, A2). |
| 4 | The Sonnet verifier found the Android 8 crash, the updater redirect hole, a leak | Not supported as attributed | The downstream log credits the Android 8 to 10 location crash to T81's new release-realism sweep parts ("the first sweep that would have caught" it), after it had shipped past verifiers in three alphas; the redirect allowlist is described as the implementer's design. Real verifier catches do exist (T77's FIX round: z7 contour requests answered 503 below a pack's minimum zoom). Cite those instead. |
| 5 | Verifier "never edits" | Inaccurate | Downstream ADR 0021 E3: the verifier rebases and lease-pushes the task branch when `main` moved. |
| 6 | Wave of five to ten tasks | Differs | Downstream rule is 5 to 8 implementers at once (`CLAUDE.md`, wave skill). |

## Part 2. Objections

1. **Serious. Owns lines partition files, not behaviour, and the proposal has no check for the
   difference.** Wave 12 downstream had T74 (content-versioned tile cache) and T75 (pinned offline
   areas) with disjoint Owns lines; T75 added a `keep` hook to eviction "that T74 was told about".
   T74's version change deletes old-version tiles through a path that ignores `keep`, so a pinned
   offline area loses its tiles whenever the server's content version moves; both implementers and
   both verifiers passed it, and a post-hoc critic found it (downstream ADR 0027 review). Same wave:
   T77 found and fixed a latent cross-task bug in T76's pack join (piece index from a shared cache),
   with "no new test for the other-order case". *Resolve:* the board gets an "Interacts with:" line
   per task naming the mechanism shared with another task in the wave; the verifier of the second
   PR to merge is briefed with the first PR's change to that mechanism and must run one scenario
   that crosses them.

2. **Serious. The critic loop has no place in a wave, and in practice it was skipped.** The
   downstream wave skill says "A task that faces a major choice writes a proposal first", but T75
   (a format other phones read) and T77 (a routing engine choice) wrote their proposal into the
   design doc and built in the same task; ADRs 0027 and 0028 were then written after the fact with
   "no separate critic pass", and the post-hoc review found one blocking and four serious
   objections across them. The base has the same pattern: ADR 0006 records proposal 0005 as its
   source while 0005 still says "Awaits the critic". *Resolve:* in the `wave` skill, a task facing a
   major choice stops after the proposal; the coordinator launches the critic; the implementer
   resumes after the answer (or the proposal is its own task in the wave before). The verifier
   refuses MERGE on a PR that implements a major choice without a critic review on file.

3. **Serious. The plan assumes a wave fits the budget at launch; closing it does not.** Downstream
   allows verifiers, merges, the sweep and the release past the launch line. Wave 12 launched its
   first half at 72% of the weekly budget; wave 13 now cannot launch at 95% (its handoff). Closing
   costs are a verifier per PR, fix rounds, a re-verify, a sweep and release checks, none counted
   when the wave was sized. *Resolve:* size a wave by its whole cost (implementers plus verifiers,
   one fix round each, the sweep), launch only if that fits under the hold line, and write the
   close-out allowance into the board's wave paragraph. With ADR 0006's per-meter thresholds this is
   a policy choice, not a hook change.

4. **Serious. The verifier and ADR 0007's reviewer are now two answers to one question.** The base
   added a `reviewer` (Opus, reads the diff, never runs) and "one independent review per change";
   the proposal adds a Sonnet `verify` agent that runs the change and also "reviews the code". As
   written a build-phase change gets two reviews or the coordinator must pick, which the open idea
   `2026-10-13-verify-agent-reads-reports-not-diffs.md` defers to this proposal. Downstream
   evidence: running things catches runtime faults (T77's z7 503s; device tours), reading with
   context catches design faults (objection 1), and neither role caught both. *Resolve:* in the
   build phase the verifier is the one review per PR (Sonnet, runs and reads); the Opus reviewer is
   added only for a PR that implements a major choice or carries an "Interacts with:" line. State it
   as an amendment to ADR 0007's rule.

5. **Minor. "Exempt the board from the cap" lets the resume file grow without bound.** Downstream
   `docs/TASKS.md` is 625 lines and 183 kB and the dated log 295 kB, both exempt; resume reads the
   board "from the bottom" every session. *Resolve:* make the alternative the rule: closed waves move
   to a closed record (`docs/tasks/wave-NN.md`), the board holds the open wave and owner actions.

6. **Minor. Item 2 is overtaken.** ADR 0006 moved commit and push permissions into `policy.json`,
   and `task-branches.json` encodes the wave model, including merged-PR branch deletion. Replace
   item 2 with "the build phase switches to `task-branches.json`". Downstream ADR 0021's supersede
   trigger is now met; that project can take the base guard back.

7. **Minor. Publish after every green sweep is one project's owner decision**, not a base default;
   offer it as a policy the `wave` skill reads, with "release only when the owner asks" as the base.

## Recommendation

Adopt B, revised: it is what actually ran, and its core (disjoint Owns lines, a worktree per task,
a running verifier per PR, a helper that writes the log and status together, a one-command sweep)
held for thirteen waves with one coordinator. The revision should add the two things the
downstream record shows it lacks: a check for behaviour shared across tasks in a wave (objection 1)
and a hard stop for the critic on major choices (objection 2); size waves by their close-out cost
(objection 3); and settle verifier versus reviewer in one sentence that amends ADR 0007 (objection
4). Correct claims 1, 4, 5 and 6 in Part 1 before promoting. Confidence: high on objections 1, 2 and
6 (read in code and records), moderate on 3 (two meter readings, not a cost model) and on the exact
split proposed in 4.
