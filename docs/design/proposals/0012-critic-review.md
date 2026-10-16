# 0012. Critic review: decision practices for long-lived software

Date: 2026-10-16
Reviewer: critic (Opus 5.5, process proposal by model override)
Proposal: `0012-decision-practices-for-long-lived-software.md`

Summary: the direction is right and most of the practices are cheap and sound (the template
changes in F, the dependency checklist in D, naming promised surfaces in B). Three things are
wrong enough to change before an ADR: the evidence does not say what the Question says it says;
the sixteen-line `CLAUDE.md` block breaks the downstream project where the practices would first
run; and two mechanisms (the surface fingerprint and the fitness check failing a push) would
block ordinary feature work with no documented lift. Fifteen objections: 2 blocking, 7 serious,
6 minor.

## Claims checked

| Claim | Verdict | Source and what it says |
|---|---|---|
| CodeThread, 13.1 percent lower later-task resolution on agent-written code | Overstated | arxiv.org/abs/2606.21804 and /html (Patel, Hou, Purohit, Xu, Pan, He, Chen; v2 29 Sep 2026). Two-step chains (PR1 implements, PR2 is the original benchmark issue on top) over SWE-Bench Verified, Multilingual, Pro and FeatBench, four agents (Claude 4.5 Sonnet, GPT-5, GLM 4.7, MiniMax M2.5) on SWE-Agent. The 13.1 is percentage points, GLM 4.7 on SWE-Bench Pro, in the shared-instance table (45.2 to 32.0); the full-dataset figure for the same cell is 11.1. Averages by task type are 4.05 (bug fix) to 8.21 (refactor) points. The comparison is code built on (agent PR1 versus human PR1), not long agent histories; the authors say longer horizons are untested. |
| CodeThread, the cause is structural | Contradicted | The paper: "traditional static maintainability metrics generally do not explain these differences"; complexity and size metrics show "near-identical distributions". The significant signals are behavioural: input/error contract drift at PR1 (input gating, silent fallback, error handling; OR 1.83, p = 0.011), downstream lines added at PR2 (OR 1.88, p = 0.042), and task difficulty. McFadden R squared 0.069; most failures unexplained. Nothing on interfaces, dependencies or module depth. |
| Needle in the Repo: maintainability defined; probes executable and structural | Verified, narrow | arxiv.org/abs/2603.27745 (Zhu, Zhang, Wang, Yang, Qiu; 29 Mar 2026). Failure defined as code "behaviorally correct for the current task but makes future evolution harder to extend, test, or reason about safely". Each probe pairs functional tests with a structural oracle, an executable, case-specific check (for example, does a `SortView` abstraction exist; is `std::stoi` used outside the validation path). 21 authored cases in small C++ repositories, two or three per dimension; dependency control passes 4.3 percent, responsibility decomposition 15.2. The 64 of 483 "passes tests, fails the oracle" figure disagrees with the paper's own Table 2 (which sums to about 32). The oracles encode the intended solution of one task; they are not project-wide fitness functions. |
| SWE-CI: regressions under long-term maintenance | Verified; no structural attribution | arxiv.org/abs/2603.03823 (Chen, Xu, Wei, Chen, Zhao; v4 1 Apr 2026). 100 tasks from 68 Python repositories, about 71 commits each, 20 models. Most models have a zero-regression rate below 0.25; "only the two Claude-opus models exceeding 0.5". Maintainability is measured as test pass counts over time; why regressions happen is not analysed. |
| EASE 2026 study on AIDev | Cuts the other way | arxiv.org/abs/2605.06464 (Sawada et al., EASE 2026 short papers). 508 agent-created and 508 human-created files in 100 repositories, six months: agent files got about half the commits in month one, fewer fixes (11.7 versus 16.8 percent of commits) and more feature extensions; the authors conclude they "receive less frequent maintenance". The finding cites it as the "dataset-scale view" in support; it is not support. |
| sqlite.org/lts.html | Verified | Support "through the year 2050"; the C API and on-disk format kept "fully backwards compatible"; "avoid being sucked into the latest programming fashion". |
| Ousterhout, chapters 2 to 4 | Verified from secondary sources | 10 to 20 percent of development time on investment is in chapter 3; complexity as dependencies plus obscurity in chapter 2; deep modules in chapter 4. Not checked against the book text. Note: his investment is continuous, made by each developer in each change, not a scheduled pass. |
| McKinley; Lefkowitz; the 2025 post | Partly wrong | boringtechnology.club: known unknowns versus unknown unknowns is McKinley's own point. blog.glyph.im/2024/07/against-innovation-tokens.html is 2024, not 2025, and never says "known failure modes": its defensible core is operational overhead and too many tools, and its remedy is "boundary tokens" (spend one only when crossing into another technology). No 2025 revisiting post was found. |
| research.swtch.com/deps; go-proverbs.github.io | Verified with a caveat | Cox's checklist: design, code quality ("code you'd want to debug"), tests, open issues, maintenance, usage, security with untrusted input, license, own dependencies; then test, abstract, isolate, avoid, upgrade, watch. Copying is for small needs. He does not call a dependency a liability and calls reuse "an enormously positive transformation". The proverb is listed verbatim. |
| ArchUnit, import-linter, cargo-deny | Verified, one overreach | ArchUnit: Java architecture tests, Apache-2.0 (github.com/TNG/ArchUnit). import-linter: Python layer, forbidden and independence contracts, BSD-2-Clause. cargo-deny: licenses, bans (deny or allow lists, duplicate versions, wildcards), advisories, sources; no cap on the number of dependencies, so it does not do C's budget check. Ford, Parsons and Kua's book not fetched; its existence is not in doubt. |
| go.dev/doc/go1compat | Verified, cuts against B as written | The promise allows growth: "The APIs may grow, acquiring new packages and features, but not in a way that breaks existing Go 1 code." |
| `CLAUDE.md` 134 lines; block 15 plus one blank | Verified | `wc -l` gives 134; the fenced block is 15 lines; with the separator the file reaches exactly 150. |
| ADR 0002 has five deferred items | Verified | ADR 0002, Deferred, with triggers. |
| test-hooks.js contract | Partly | `--only`, `--verbose`, exit 0, 1 and 2 are there; its lines read `FAIL <id> <label>`, not `FAIL <check>: <detail>`. |

What the evidence supports: agent-built code is somewhat harder for agents to build on (a few
points on average, up to about 13 in one cell), the effect shows up in small, task-sized
changes, and functional tests alone miss some structural constraints when a probe was written to
look for them. What it does not support: that the cause is widened interfaces, accreted
dependencies or shallow modules. The strongest study names behavioural contract drift and code
growth instead. The proposal's inference has the locus right (small choices inside briefs) and
the mechanism unsupported.

## Objections

### 1. The Question and the finding state a cause the studies do not show (blocking)

Evidence: the table above. The Question says the studies place the cause in "interfaces that
widened, dependencies that accreted, modules that went shallow". CodeThread says static metrics
do not explain the gap; SWE-CI does not analyse causes; the EASE paper finds agent files
maintained less, not more; only Needle in the Repo measures structure, through oracles written
for one intended solution each, in 21 small C++ cases with an internal counting error. The
proposal's own first reversal condition ("a 13.1 percent that is one benchmark's one
configuration") is met: it is one model on one benchmark, in points, from a subset table.
Resolve: correct the Question and the finding's Result in place (the proposal skill requires it):
the locus is small changes, the measured mechanism is behavioural drift and growth, structural
loss is shown only in authored probes. Apply the proposal's own consequence: B, D and F stand on
the older sources; C waits for a project to ask for it and leaves `CLAUDE.md`.

### 2. The sixteen-line block breaks the first project where the practices would run (blocking)

Evidence: the build-phase sibling with code (`../regenesis`, Go and Python, live data) has a
`CLAUDE.md` of exactly 150 lines today, its own sections included (`wc -l`). The
`update-from-hatchery` skill verifies `wc -l CLAUDE.md` at most 150. Its next update would land at
about 166 and fail verification, forcing that project to delete sixteen lines of its own rules to
take sixteen of the base's. The base's header says projects edit "the project-specific parts";
a base at 150 leaves them no room. Filling the cap to the last line is also the tactical move the
proposal argues against: ADR 0007 already raised the cap once because the base was over it.
Most bullets have a home that is loaded at the moment of use: the brief template (interface and
dependency fields), `reviewer.md` (the rule), the `proposal` skill (checklist, horizon),
`architect.md` and the handoff skill (the pass), the test command (fitness checks). `CLAUDE.md`
is read every turn by the runner and pusher, which need none of it.
Resolve: at most four lines in `CLAUDE.md`: one principle sentence with the pointer to ADR 0012
and the files that carry the practice, and if wanted one clause in Major decisions ("an
incompatible change to a promised surface"). Everything else lives where it is applied. Path-scoped
rules are not the answer while the base has no code globs; a project adds one with its language.

### 3. The practice the evidence does support is missing (serious)

Evidence: CodeThread's one significant code-level signal is drift in the input and error
contract (validation removed or moved, silent fallbacks, changed error handling) plus downstream
code growth. Both are reviewable on a diff by the Opus reviewer today, at no extra pass. The
proposal dismisses the current reviewer as checking "behaviour", which is where the study points.
Resolve: the reviewer rule gains "a changed input check, a new silent fallback or a swallowed
error the brief did not name is a serious finding", and the reviewer reports net lines added
against the brief's expectation. This is cheaper than C and better grounded.

### 4. "A change to a surface is a major choice" and the fingerprint check block every feature (serious)

Evidence: B and the third fitness check treat any change as the trigger: any change runs the
full loop; a fingerprint "must match" or the version moves. Go's promise, cited as precedent,
allows growth. The debug-interface rule requires every feature to add a CLI, HTTP or REPL entry
point, the `debug-interface` skill says to prefer "the one the feature already needs for
production", and its output "must be stable text". So a command-line contract named as a surface
changes with every feature: the help-text fingerprint fails every feature push, and each one is a
Fable proposal. Agents will either loop on the architect or learn to bump the version line by
rote, which is the mechanical compliance the brief warns about.
Resolve: the trigger is an incompatible change (removal, rename, changed meaning, changed
format of existing output); additions are allowed and the check compares compatibility (old
fingerprint is a subset of the new), not equality. Debug surfaces are not promised unless the
surfaces ADR names them.

### 5. The fitness check refuses a push with no documented lift (serious)

Evidence: C fails the push through pusher check 8 ("It must pass"). The only lift written down is
an ADR amendment, which is architect work: Fable, and impossible from HOLD because the gate
refuses the launch. The disable-with-reason line is deferred until three wrongful blocks, and no
file counts blocks. The `FAIL <check>: <detail>` line names the config line but not who may lift
it. In practice the only in-session lift is the owner running `git push` from a terminal, which
bypasses every check, not just the wrong one. `pusher.md` is not in the edit list, so the pusher
reports a generic test failure.
Resolve: ship the skip in the contract now: a `skip: <check> <reason> <date> <idea file>` entry in
the fitness config that the coordinator may add for one push (recorded in the commit message,
with an idea file for the architect), the architect amends the config by ADR afterwards, and the
`FAIL` line ends with "lift: skip entry in <config> (coordinator), amendment (architect)". Pusher
check 8 says a fitness failure is reported as design work and not retried.

### 6. The base has promised surfaces and code now; the proposal waits for someone else (serious)

Evidence: Hatchery ships hooks with a test harness, and downstream projects depend on its formats:
`.claude/hooks/policy.json` (ADR 0006), the three-block idea grammar that ADR 0007 says "must stay
identical across the three repositories", the `Hatchery base: <sha>` line (ADRs 0009 and 0010),
and the usage cache file older siblings still read (ADR 0011 kept it for that reason). ADRs 0009
to 0011 are compatibility incidents of exactly the kind B addresses. The hooks use only Node
built-ins, so a dependency budget of zero is a check the base could run today. Deferring B and C
to "the first decanted project with code" ships an untested contract to a project to debug.
Resolve: the first implementation step is Hatchery's own surfaces ADR (the four surfaces above,
with what is not promised), and one reference fitness check in `test-hooks.js` (hooks require
only Node built-ins) that exercises C's contract, its skip line and its failure message.

### 7. Fable spend is under-flagged and the cadence is set before the owner answers (serious)

Evidence: the proposal flags only E. E's cost is stated as one pass "bounded by its output (one
artefact)", but an artefact that is a proposal pulls in a critic, the architect's answers and an
ADR: up to four Fable passes. B makes interface design "an architect task where today it is the
coder's smallest reasonable choice": Fable per feature, not flagged. C sends every failure to
the architect. The `CLAUDE.md` block states "One architect design pass per build chunk" as a rule,
while the text says the cadence is the owner's and "the ADR records the owner's answer"; the
first implementation step has no point where the owner is asked. "Build chunk" is defined nowhere
in the tree (it appears once, informally, in the handoff skill's description), so the owner cannot
price the recommendation. Ousterhout's 10 to 20 percent is continuous investment by whoever
writes the change, which maps to the coder's brief at Opus price, not a batched Fable pass.
Resolve: put a Question to the owner in the ADR or `docs/HANDOFF.md` before the rule lands, with
the cost per pass stated as a range. Offer the cheaper shape as the default: the `handoff` or
`triage` skill (Sonnet or Opus) lists fired ADR triggers, unchecked predictions, fitness failures
and interface or dependency findings; the architect runs only when that list is non-empty. Name
which B decisions are architect work (surfaces, cross-module interfaces) and which stay the
coder's (module-internal).

### 8. "Default no" moves code from tested upstream to agent-written copies, and clashes with licensing (serious)

Evidence: the studies measure agent-written code as the hazard; "default no" plus "copy it in"
means agents reimplement or paste what a library did, which grows exactly that code. The budget
check counts direct dependencies, so copying evades it while keeping the liability: Goodhart in
one step. Cox recommends copying for small needs and otherwise inspection, tests and isolation;
he does not call dependencies liabilities. Licensing: `CLAUDE.md` says every source file starts
with a GPL or LGPL SPDX header and `coder.md` says "Add the SPDX header to every new source file";
a copied MIT or Apache-2.0 file must keep its own notice (Apache also its NOTICE), and ADR 0001's
component table has no row for third-party code. An agent following both rules either mislabels
the copy or breaks the rule.
Resolve: the default is "evaluate", not "no"; copying is limited to under a page with an origin
header in the original's SPDX identifier, under a vendored path listed in ADR 0001; the budget
counts vendored copies; reimplementing a parser, a format, crypto or date handling is not
"copying" and takes the checklist like a dependency. Add Cox's security and watch questions.

### 9. "Boring means the Opus reviewer can verify it" is a model judgment that pre-decides language choices (serious)

Evidence: the reviewer is asked to report whether it "could verify the generated code line by
line", a yes/no self-assessment that models make poorly and will answer "yes" by rote. It
contradicts C's own principle ("Nothing a model judges; a program says FAIL"). It quietly sets a
criterion for every future language and framework proposal in every project (hidden coupling),
biased toward whatever has the most training data, and names a model in a rule that outlives the
routing table. The attribution is off: Glyph's defensible point is operational overhead and tool
boundaries, not known failure modes.
Resolve: drop the bullet from `CLAUDE.md`. In the proposal template ask for machine
checkability (static checks, toolchain stability, the number of technology boundaries a choice
adds, after Glyph), and have the reviewer list specific lines it could not verify rather than a
yes or no.

### 10. The brief field is ambiguous and often unknowable at brief time (minor)

Evidence: "Interface: <the interface this pass must not widen; or none>": "none" can mean nothing
is protected or nothing may widen, and the reviewer rule depends on which. A feature brief widens
by design; the coordinator writes briefs without reading code, so it often cannot name the
interface. The brief skill already says every field is filled, so "none" becomes the default.
Resolve: "Interface: may add <X>; must not change <Y>; or unconstrained (say why)".

### 11. A serious reviewer finding is overridable but the override is unrecorded; a coder mid-pass has no path (minor)

Evidence: `CLAUDE.md` sends only overridden blocking findings to the user, so the coordinator may
accept a serious "unnamed widening" or "unnamed dependency" alone, with no record, and the next
design pass cannot see it. A coder that finds it needs a dependency mid-pass has no instruction:
add it and be flagged, or stop.
Resolve: an accepted serious finding is named in the commit message; the coder stops and reports
a needed dependency or widening rather than adding it.

### 12. The reversal conditions need counts no file keeps (minor)

Evidence: "three design passes with nothing to report" and "three pushes blocked with no design
change" need a tally across sessions; E claims "no new state", and the handoff is rewritten each
session under a 40-line cap. Also a pass told it may write one artefact rarely reports nothing.
Resolve: one idea file per condition holding the tally, or a dated line in ADR 0012's
Consequences, appended by the pass.

### 13. Predictions "in a year" do not fit a base whose ADRs turn over in days (minor)

Evidence: ADRs 0008, 0009 and 0010 were superseded or amended within three days of each other.
A one-year prediction will be checked against a superseded record or never.
Resolve: each prediction carries its own check point (a date or a trigger), and superseding an
ADR checks its predictions first.

### 14. "Nothing here changes what humans see" is true for the base, not for each project (minor)

Evidence: the surfaces ADR names file formats and command-line contracts that users touch, and
B itself says a surface is "reviewed ... by a human". Under `CLAUDE.md`, that is the owner's call.
Resolve: say that each project's surfaces ADR, and any compatibility promise, goes to the owner.

### 15. Small factual slips (minor)

Evidence: cargo-deny has no dependency count limit; test-hooks.js prints `FAIL <id> <label>`;
Cox does not say "liability"; Glyph's post is 2024 and the 2025 post was not found; the finding
cites the EASE study as support.
Resolve: correct in place.

## Who can lift each refusal

| Mechanism | Refuses what | Lift as proposed | Says so in its message | Gap |
|---|---|---|---|---|
| Fitness check (pusher check 8) | The push | ADR amendment by the architect; owner pushing by hand | No | Objection 5 |
| Surface fingerprint | The push | Moving the ADR's version line (architect) | No | Objections 4, 5 |
| Reviewer serious finding | Nothing directly; the coordinator routes it | Coordinator, alone, unrecorded | Not applicable | Objection 11 |
| Dependency default no | The coder adding one | A brief with the checklist answered (coordinator) or the proposal loop | Not applicable | Objection 11 |
| `CLAUDE.md` cap after the update | The update's verification | Deleting project rules | Yes (post-edit) | Objection 2 |

## The strongest case against, and the best alternative

Option A, steelmanned: the loop already treats an API surface and any dependency the project leans
on as major choices, reviews every change with Opus, runs tests before every push, and fixes its
own process "in the same sitting" when friction shows. The best-measured long-horizon result
(SWE-CI) is about regressions, which tests catch, and the Opus models were the only ones above
0.5 zero-regression; the large field study found agent files needed less upkeep. No Hatchery
project has yet shown structural decay, and the base's habit is to defer with a named trigger
(ADR 0002) rather than build ahead of evidence. The owner has asked for practices, so A is out,
but its argument limits how much weight the new practices should carry before a project shows
the problem.

Process weight is the real risk. Each new field (Interface, Dependencies, Exit cost, Moving parts,
Predictions, line-by-line verifiability) will be filled on every brief and proposal; fields that
cannot fail get boilerplate. The ones that work are those a program or a specific question
checks: an unnamed dependency in a diff, a removed validation, an incompatible change to a named
format. The rest should be few.

Best alternative: a smaller set, tested on the base first. F as proposed with dated predictions.
D's checklist in the `proposal` skill, default "evaluate", with the vendoring row in ADR 0001. B
written first for Hatchery's own surfaces, scoped to incompatible change. One reviewer rule that
covers unnamed dependencies, incompatible surface changes and input or error contract drift. C as
a contract with its skip line, proved by one check in `test-hooks.js`. E replaced by a cheap
trigger scan with the architect called only when it fires, the scheduled cadence put to the owner
as a Question. `CLAUDE.md` gains at most four lines.

## Recommendation

Accept the proposal's direction and most of its parts, but not as written. Fix the two blocking
items first: correct the evidence (the cause is not shown to be structural; the one significant
signal is behavioural contract drift, which the reviewer can check today) and shrink the
`CLAUDE.md` block to at most four lines, because the downstream project with code is already at
150. Then scope surfaces to incompatible change, give the fitness check a lift that works during a
session, put the design-pass cost to the owner as a Question with a trigger-based default, and
apply B and C to Hatchery itself before any project. I am confident (about 90 percent) that the
evidence correction and the `CLAUDE.md` size must change, and moderately confident (about 70
percent) that the trimmed, trigger-based set above gets the owner's long-term aim at lower cost
and with less rote compliance than the proposal as written.

Sources: arxiv.org/abs/2606.21804 and arxiv.org/html/2606.21804; arxiv.org/abs/2603.27745 and
arxiv.org/html/2603.27745; arxiv.org/abs/2603.03823 and arxiv.org/html/2603.03823;
arxiv.org/html/2605.06464; sqlite.org/lts.html; research.swtch.com/deps; go-proverbs.github.io;
go.dev/doc/go1compat; boringtechnology.club; blog.glyph.im/2024/07/against-innovation-tokens.html;
github.com/TNG/ArchUnit; import-linter.readthedocs.io; embarkstudios.github.io/cargo-deny (and
its bans configuration page); goodreads.com notes on A Philosophy of Software Design (secondary).
