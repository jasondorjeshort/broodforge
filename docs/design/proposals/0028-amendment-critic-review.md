# Critic review of ADR 0028 as amended 2026-11-02

Date: 2026-11-02. Reviewer: critic (Fable 5.1). Subject: the `Amendments 2026-11-02` entry of
`../decisions/0028-landing-a-session-on-the-default-branch.md` (Decisions 1, 2, 5 as it changes
them) and the companion `../decisions/0028-land-refusals-and-push-forms.md`. Sources: the two
files; `0028-critic-review.md` (its objections are not repeated; the proposal answered them);
`../decisions/0022-linux-spawn-and-handoff.md` (Handoff, Amendments 2026-10-31);
`../04-ux-session-end.md`; `../03-architecture-git.md` (Landing); `../05-debug-interface-git.md`
(Landing, The fetch at session start); `../04-ux.md` (Landing, the canonical-diverged row);
`../08-implementation-plan-landing.md` (P36); the four idea files of 2026-10-23, 2026-10-29 and
2026-11-02 the amendment touches; and the code at `83f1e60`: `internal/git/handoff.go` (`Run`,
`runMain`, `FallsBack`, `FallbackLine`, `Land`, `fastForwardOnto`, `landPush`, `token`),
`internal/session/service.go` (`Land`, `landGate`, `LandGate`), `internal/session/actions.go`
(`End`, `end`, `endRefusal`, `endWith`, `checkAnswerList`, `acting`, `endInfo`),
`internal/session/workflow.go` (`ReviewFact`), `internal/policy/roles.go` (`MayLand`,
`Permitted`), `internal/cli/session.go` and `internal/cli/worktree.go` (`sessionEnd`,
`runHandoff`, `hostGate`), `internal/config/load.go` (`landWarnings`), `tests/repro/land.sh`.
Nothing was run; the repro script's asserted strings are read as the recorded behaviour.

## Verdict in one paragraph

The amendment is right in its mechanism and I would keep all three of its moves: every land refusal
at the end but `land-canonical-diverged` falls back to `handoffFallback`; the end is judged by the
ender, not the driver; an end with nobody behind it falls back. It is the only reading under which
the end's WIP commit of 2026-11-01 and `"handoff": "main"` can coexist (a dirty-tree end would
otherwise be unendable except by discard), it keeps "nothing is lost", and the code does what the
text says. Three things need fixing before the texts go to the owner, none of them the mechanism.
The fallback line is built by wrapping the mid-session refusal, so after an end that removed the
tree it tells a closed session to rebase its tree and push again (`FallbackLine`, asserted verbatim
in `tests/repro/land.sh` step 4). The amendment quietly removes the one path by which a finished,
reviewed, unattended session reached `main` (under Decision 5 as written the idle end was judged by
the driver's roles; now it is judged by nobody's), and it neither says so nor gives the real reason,
which is that the bot stores no roles. And the venue's `[End session]` now does two different things
depending on who presses it, with no word beforehand. One open point the brief asked me to take a
position on (`worktree handoff` landing with the role unchecked): keep it, and resolve the
inconsistency by bringing the host's `session end` in line with the host convention, not the other
way round; reasoning below.

## Claims verified

| Claim | Finding | How |
|---|---|---|
| 1. A fallback on `land-unreviewed` publishes the unreviewed tip to the session branch exactly as a `push` project does, and nothing else reaches `main` | Holds. `Land` runs step 1 (`Fetch`: the canonical's tracking ref and default branch move to the remote's, nothing public), `collectLocked`, `vet`, then the gate; a gate refusal returns before `landPush`. `runMain` copies the `Handoff`, sets `Mode` to `Fallback` (`push` when empty) and calls `run`, the same function a `push` project's end calls, pushing `refs/sessions/<id>:refs/heads/<branch>`; the WIP commit made before `runMain` is in it. One difference: `run` calls `Collect`, so the tree is bundled a second time (objection 10). | `handoff.go` `Run`, `runMain`, `Land`, `run` |
| 2. The idle end on a `main` project can never land `main` | Holds. `end()` builds `ender{actor: actor}` with no roles; `endWith` calls `landGate(l, "", true)`; `LandGate` with `checkRoles` true calls `MayLand(nil, land, groups)`, and `Permitted` with no held roles returns false for every list (a `driver` entry is skipped with `isDriver` false; a group or raw id needs a held role). So the first refusal met is `land-not-allowed` and `FallsBack` makes it a fallback. The overnight idea as filed is not defeated (its accept is a human's click), but the idle end pre-empts it: once the fallback has pushed, the tree and `refs/sessions/<id>` are gone (`run` calls `remove`) and no bot path lands that branch later; the channel must run with `idleTimeoutMinutes` 0 or take the grant idea. Objection 2. | `actions.go`, `roles.go`, the two idea files |
| 3. The ender's roles, not the driver's: who can end, and can a land-role member land another's unreviewed work by ending | Who can end: anyone `acting` admits whose roles pass the channel's `answer` list (`checkAnswerList`; the venue always passes the member's roles, `heldRoles(in.Roles)` in `adapter.go`; the host without `--roles` is unchecked). A member in the `answer` list who holds the land role and never wrote in the session can end it and the land goes through when the fact holds. The fact (`ReviewFact`) is per session: the newest `ok` reviewer row of this session naming the tip. It does gate the unreviewed case (an unreviewed tip is `land-unreviewed`, a fallback); it does not gate who lands reviewed work, which is the role model's intent. The inconsistency is with the mid-session door, where only the driver may land (`04-ux.md` Landing; Decision 5): objection 4. | `actions.go`, `adapter.go`, `workflow.go` |
| 4. `land-canonical-diverged` final at the end; is the lift reachable while a session runs | Holds and is reachable. `endRefusal` returns true for the code; the session goes to `WaitingForInput`, the idle timer re-armed unless the end was the idle timer's (then it waits for activity). The lift is the owner's shell on the host in the canonical (`reset --keep <remote>/<default>`, or a push of its commits), which nothing locks while a session runs (a concurrent land of another session of the same project could race step 5; rare). The venue text says only "the owner reconciles it": no command, no word that the session stayed open or that End works again; the companion and 05 name `branch -f` first, which git refuses on the checked-out branch the canonical always has (the owner corrected this to `reset --keep` on 2026-10-31, `04-ux.md` row canonical-diverged). Objection 6. | `actions.go`, 04, 05, companion |
| 5. The companion's warning and every refusal text's lift under the amendment | The `config check` warning ("every session end there will fall back") now agrees with Decision 1; `landWarnings` prints it. The `land-not-allowed` row's lift names the amendment and the ender. The `land-unreviewed` row says "every dirty-tree end is this code", which is wrong in gate order when the ender has no land role (`land-not-allowed` is met first); objection 5. Every fallback line carries the mid-session lift verbatim; objection 1. | companion, `load.go`, `handoff.go` |

Further facts the reading produced:

| Id | Finding | How |
|---|---|---|
| F1 | `git.Error` has `Code`, `Message`, `Commits`, `Hits`, `Checked`: no field separates the reason from the lift, although the companion's push-word deny is specified as `<text>; <lift>`. `FallbackLine` therefore wraps the whole message. | `git.go`, `handoff.go` |
| F2 | Absent `--roles` is read two ways inside one `End`: `checkAnswerList` treats it as unchecked (passes), `landGate(l, by.roles, true)` treats it as none (refuses). `session land <id>` without `--roles` is unchecked (`req.Roles != ""`), as `05` documents for every session command; `land.sh` step 1 reaches `land-unreviewed` with no `--roles`. `session end <id> --roles <id>` with no `--as` lands as actor `host` with no `claimed`. | `actions.go`, `service.go`, `land.sh` |
| F3 | Two lists decide an end's outcome and must stay complementary: `git.FallsBack` (four codes) and `session.endRefusal` (`handoff-refused-*` or `land-canonical-diverged`). A `land-*` code in neither falls to `closeSession(... "failed: <code>")`: the session closes, tree kept. The handoff's `token` error at a `main` end (`handoff-failed-push`, a missing credential) takes that path today, as on a `push` project. | `handoff.go`, `actions.go` |
| F4 | The amendment's rationale for the diverged exception (the canonical's own commits would be published under the session's name) applies to the ahead canonical too, which the start admits (`ahead <n>`): a tip descending from local-only commits passes `fastForwardOnto` (the remote's branch is an ancestor) and lands them on `main`. Accepted for `push` projects on 2026-10-30; new for `main`. | `fastForwardOnto`, 0022 companion step 0 |
| F5 | Stale comments: `Outcome.Fallback` and `Run`'s doc still name "land-not-fast-forward, land-rejected" as the two fallback codes. | `handoff.go` |

## Refusals and lifts at a `main` end under the amendment

| Mechanism | Who lifts it, from where, while a session runs | Does the text say so? |
|---|---|---|
| `land-not-allowed` (fallback) | To land instead: the owner adds `roles.land` (`config reload`, the host), or a member who holds the role ends the session (the venue). After the fallback: nobody in the bot; the branch is merged by hand. | The line names the role, not whose (the ender's) nor that another member could have ended it. |
| `land-unreviewed` (fallback) | Before the end: the session commits (the WIP is made only at the end, so a dirty tree cannot be reviewed first), launches the reviewer, then ends. After: nobody in the bot. | The line tells the closed session to "commit, launch the reviewer, then push again" (objection 1). |
| `land-not-fast-forward` (fallback) | Before: the driver or the runner rebases in the tree, reviews again, ends. After: by hand. | The line says "rebase this tree" after the tree was removed (objection 1). |
| `land-rejected` (fallback) | A race: nobody now; the branch is pushed. Protection or a checkout remote: the owner, the remote's configuration or `handoff`, from the host, any time. | "push again" is wrong after the end; the owner's part and the appended checkout warning are right. |
| `land-canonical-diverged` (final, session open) | The owner, on the host, in the canonical (`reset --keep`), while the session runs; then End again. | Not said: no command, no "end again", no notice-channel post as the start's refusal has (objection 6). |
| `handoff-refused-*` (final) | As today: the session amends and ends again. | The existing texts. |
| `handoff-failed-push` from a missing credential (closes the session, tree kept) | The owner: `setup git-credential`; then `worktree handoff <id>`. | Unchanged from a `push` project; not the amendment's. |
| An unlisted `land-*` code (F3) | Nobody: the session closes "failed". | A design default, not a text (objection 8). |

## The open point: `worktree handoff` lands with the role unchecked

Position: keep it, and fix the inconsistency on the other side. The host convention since 05 is that
`--roles` absent leaves the role unchecked (every session command, `checkAnswerList`, `session
land`); the role gate is about Discord members, and on the host there is no member, only the owner's
shell (or, on the PC, whatever rule 8 lets through, which is the same for every host door). A typed
`--roles` on the host proves nothing (any id in the list passes), so requiring it would add ceremony,
not a check; and `worktree handoff` is the one door that lands a session the service no longer holds
(`session land` needs `s.lookup` to find a live session), so a refusal there would leave a dead
session's reviewed work with no land at all. The amendment's `session end <id>` falling back without
`--roles` is therefore the odd one out: inside one command (F2) absent roles are unchecked for the
`answer` list and none for the land. The honest rule is: the idle end is the only roleless end; on
the host, roles are a reproduction aid, never a gate. I would write that sentence into the
amendment, let the host's `session end` land as `session land` and `worktree handoff` do (actor
`host`, `claimed` only when `--as` is given), and change `land.sh` step 5 to an idle end (a short
`--idle-timeout-seconds`) so the test still proves the roleless fallback. The alternative, uniform
in the other direction (absent `--roles` is nobody everywhere; `session land <id>` refuses,
`worktree handoff` gains `--roles` and falls back without it), is coherent too and would make the
audit carry an explicit assertion; its costs are a changed `session land`, a flag that proves
nothing, and a dead-session land that depends on typing it. Either way the ADR must say which
reading of absent `--roles` holds, since today both are in the code. Confidence moderate: the fork
is a judgment about whether the host is trusted, which the owner already answered yes for `session
land`.

## Objections

1. **Serious. The fallback line carries a lift that is unreachable after the end.** `FallbackLine`
   strips `Not landed: ` and the final period from the refusal's whole message and appends `Handed
   off as <branch> instead.`; the message carries the mid-session lift. `land.sh` step 4 asserts the
   result: `... rebase this tree onto origin/main (the driver or the runner), review again, then push
   again. Handed off as <branch> instead.` A done fallback removes the tree and closes the session
   (`run` calls `remove`; `closeSession`), so "this tree", "push again" and, for `land-unreviewed`,
   "commit, launch the reviewer" name nothing that exists; a driver reading the summary is told to do
   the impossible, and the actual lift (the branch is on the remote: merge or rebase it from a
   checkout, or start a session from it) is nowhere. The wording is the owner's; the structure (F1:
   one string for reason and lift, though the companion already distinguishes `<text>; <lift>` for
   the push word) is the design's. Resolves by: the refusal carries reason and lift apart (a `Lift`
   field on `git.Error`, or the `Land*` constructors take both); `FallbackLine` uses the reason alone
   and adds the end-time lift; the companion's table gains an at-the-end lift per row; the repro's
   expected strings follow.

2. **Serious. The amendment removes the unattended land and does not say so, or why.** Decision 5
   as written judged every land by the driver's roles, so an idle end of a reviewed session landed;
   the amendment judges the idle end by nobody's, so it always falls back, and after the fallback
   the tree and `refs/sessions/<id>` are gone and no bot path lands that branch (claim 2). On the
   owner's instance that is "a branch to merge by hand" on every idle end, the shape the ADR's
   Context says the owner's projects want to avoid, and it pre-empts the overnight idea's accept
   button unless the channel's idle timeout is off. The choice is still right as the default: the
   bot stores no roles (`Permitted` is "a pure function of the roles a person holds at that
   moment", fed per interaction by `heldRoles`), so the idle end has nothing to judge by without a
   Discord member fetch or a grant recorded at start; the grant idea
   (`../../ideas/2026-11-02-an-unattended-session-lands-on-idle.md`) is that alternative. Resolves
   by: the amendment states the consequence (an idle end on `main` always falls back and its work is
   then the owner's to merge), the reason (no stored roles), the recorded alternative with its
   trigger (the owner wants unattended landing), and the interim advice (`idleTimeoutMinutes` 0 on
   `main` channels); the owner is told, since it is their loop.

3. **Serious. The End button now does two things and says neither beforehand.** Any `answer`-list
   member may end (claim 3). Under the amendment the same `[End session]` lands `main` when the
   ender holds the land role and pushes a branch when not; there is no confirm (only discard has
   one) and the `Ending:` line says "checking and handing off N commits" either way (`ending`). A
   land-role member tidying a colleague's venue lands its work; a driver without the role, expecting
   their `main` project to land, gets a branch and a line addressed to a closed session (objection
   1). Both are correct by the rules and both surprise; before the amendment the driver's own role
   decided, which the driver knows. Resolves by: the `Ending:` line (the owner's wording) names the
   target as computed from the ender's roles before the handoff runs ("landing on `main` if a
   reviewer read the tip, else as `<branch>`"), or an End on a `main` project by a land-role ender
   confirms as discard does. The owner picks the form; computing it from the ender's roles is the
   design's part.

4. **Minor. The two doors judge different people, and P36 inherits the split.** Mid-session,
   `/session land` is the driver's (`04-ux.md` Landing: "by a driver with a role"; Decision 5); at
   the end, any `answer`-list member's. So a land-role member who is not the driver cannot land
   mid-session but can land by ending, and P36's plan text already wavers ("the caller's roles").
   The amendment has made P36's decision without naming it. Resolves by: one sentence in the
   amendment: a land at either door is judged by the member who asked for it, admitted by the
   `answer` list; or the reason the doors differ.

5. **Minor. Two companion sentences are wrong in gate order.** The `land-unreviewed` row: "every
   dirty-tree end is this code"; it is `land-not-allowed` when the ender has no role, which the
   fallback line then reports (the order the amendment itself states). The `land-not-allowed` row's
   When column still says "the driver". Resolves by: "every dirty-tree end by a land-role ender",
   and "the driver (mid-session) or the ender (at the end)".

6. **Minor. The one final refusal is the one with the weakest text and a wrong command.** The
   `land-canonical-diverged` venue line says "the owner reconciles it" and stops: no command, no
   word that the session stayed open, no "end again", and on an idle-reason end nobody is in the
   venue to read it while the session sits open with its timer unarmed. The companion row and 05
   (The fetch at session start) name `branch -f` first, which git refuses on the checked-out branch
   the canonical always has; the owner's own answer of 2026-10-31 is `reset --keep`. The start's
   refusal posts to the notice channel; the land's does not. Resolves by: the text says the session
   is open and End works after the owner's fix; `branch -f` goes from the companion and 05; P36
   gives the land's diverged refusal the start's notice-channel post.

7. **Minor. Stale code comments (F5).** `Outcome.Fallback` and `Run`'s doc name two fallback codes;
   `FallsBack` names four. Resolves by: the two comments point at `FallsBack`.

8. **Minor. An unlisted land code closes the session (F3).** `FallsBack` and `endRefusal` are two
   lists that must stay complementary; a code in neither (the checks idea's `land-checks-failed`, or
   any later one) makes the end a hard close with the tree kept. Resolves by: `endRefusal` treats any
   `land-*` code `FallsBack` does not name as final (open), or both read one table.

9. **Minor. The diverged rationale is not applied to the ahead canonical (F4).** A `main` land of a
   tip built on the canonical's unpushed commits lands them on `main` through the fast-forward check,
   and the repro suite's own habit of committing a fixture into the canonical is the known case.
   Resolves by: the amendment says the ahead case is accepted for `main` as it was for `push`, or the
   land warns in its outcome line when `rev-list --count <remote>/<default>..<base>` is non-zero.

10. **Minor. The fallback bundles twice.** `Land` collects under the lock and releases it; `run`
    collects again (`Collect` takes the lock itself), so a fallback end spawns `worktree-bundle` and
    the scan twice and reads the tip anew. At the end the turn is stopped first, so the tips agree;
    the cost is one bundle. Resolves by: `runMain` hands the `Collected` to `run`, or the double
    bundle is noted as accepted.

## Pre-mortem

Ten years on, the regret is that "fall back and say so; nothing is lost" became "fall back and
nobody reads why". The owner had said it for two refusals, conflicts and rejected pushes, and the
amendment generalised it to every refusal in one sitting because the WIP commit left no other way to
end a dirty tree. From then on a `main` project's end fell back far more often than it landed: on
every idle end (nobody's roles), every dirty-tree end by anyone but a land-role ender, every end by
a member who did not know which button they held. Each produced a session branch and a venue line
that began `Not landed on main:` and went on to tell a closed session to rebase a tree that no
longer existed; people learned to skip the line, the branches piled up on the remote, and the
owner's own projects, the ones the ADR was written for, were back to merging by hand. The single
refusal that stayed final stranded idle sessions open overnight with a line in an empty venue that
named no command. The sentence that should have caught this is in the brief, not the ADR: the owner
agreed to the fallback "for conflicts and rejected pushes, not in so many words for the role and
review refusals". An amendment that widens an owner's answer should say it has, and say what the
owner now gets instead of a land.

## Recommendation

Accept the amendment's mechanism as built: every land refusal at the end but
`land-canonical-diverged` falls back; the ender is judged, not the driver; nothing is lost. Before
the texts go to the owner, split reason from lift so the fallback line stops addressing a closed
session (1); write the unattended consequence and its reason into the amendment and point at the
grant idea, telling the owner (2); have the `Ending:` line say what this ender's End will do (3). On
the open point, keep `worktree handoff` landing with the role unchecked and make the host's
`session end` agree with it, the idle end being the only roleless end; state the one reading of
absent `--roles` in the ADR. The minor items are an afternoon of tester work. Confidence: high on
objections 1, 2, 5, 6 and 7 (each read in the code or the repro's asserted strings); moderate on 3
(a venue judgment the owner may weigh differently) and on the open-point position (a trust call the
owner has answered once already, for `session land`).
