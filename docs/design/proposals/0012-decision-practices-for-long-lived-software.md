# 0012. Decision practices for long-lived software

Date: 2026-10-16
Status: promoted to ADR 0012

Written by the architect from the coordinator's research
(`docs/findings/2026-10-16-decision-practices-for-long-lived-software.md`), on the owner's words:
optimise for long-term simplicity and sustainability, "actual software that does things, not
churning out code". Revised in place after the critic review (`0012-critic-review.md`); what
changed is in the last section. It applies the practices it proposes: a ten-year horizon in the
question, an exit cost and moving parts per option.

## Question

Hatchery's loop makes the few major choices carefully (proposal, critic, ADR) and the many small
ones quickly (a brief, a coder pass, one defect review). The 2026 studies in the finding say that
agent-built code is somewhat harder for agents to build on: a few percentage points on average,
up to 13 points for one model on one benchmark (CodeThread), with the loss showing in small,
task-sized changes. The one code-level signal that study found significant is behavioural, not
structural: drift in input validation and error handling at the first change, and code growth
downstream; static complexity metrics did not explain the gap. The one structural study (Needle
in the Repo) finds structural loss only through probes written for each task, and the field study
(EASE 2026) found agent-written files maintained less often, not more. So the evidence locates
the cost in the small changes the loop reviews for defects only; it does not show that widened
interfaces, accreted dependencies or shallow modules are the cause. The practices below stand
instead on SQLite's long-term support model, Go's compatibility promise, Ousterhout's account of
complexity, and Hatchery's own record: ADRs 0009 to 0011 are three compatibility incidents on
surfaces nobody had named. What practices should the base add so that a project it starts is
still doing its job in ten years with little upkeep, and where does each live: the every-turn
file, a skill or template, an agent file, or the first decanted project that has code? In ten
years the answer should look like this: each project has a short list of promised surfaces that
have not broken, a dependency list that fits on a screen, and a `git log` in which rewrites
behind the surfaces were cheap and ordinary.

## Constraints that matter here

- `CLAUDE.md` is capped at 150 lines and holds 134 (`.claude/rules/docs.md`); a downstream
  project with code holds 150 of its own, so every base line displaces one of theirs at the next
  update. Rules go there; procedures go in skills; reasoning goes in ADRs (ADR 0002).
- Agents write all the code and the reviewer is Opus (ADR 0007). A practice is real only when an
  agent can apply or check it from a terminal; a practice that lives in judgment alone drifts with
  the next model.
- The base has no application code, toolchain or test runner (ADR 0002, Not in the base), but it
  has hooks, a harness and formats that three repositories read. A check that needs a language
  waits for a project; a check on the hooks can run today.
- Fable costs two and a half times Opus. A practice that spends architect passes spends the
  owner's plan, and the cadence is the owner's to set (`CLAUDE.md`, Major decisions).
- Living docs state the practice, not the project it came from; this proposal is a closed record
  and may cite sources freely.
- The debug interface rule stands: a fitness check is a debug surface for the architecture and
  obeys the same exit codes (`debug-interface` skill).

## Options

Each option is one practice; the baseline gets the same depth so that rejecting it is a decision.
Options are kept as first written except for corrected facts and the revisions the critic won,
which the objections section records.

### A. Change nothing: the current loop is enough

What it is: the major-choice loop for the choices listed in `CLAUDE.md`, one defect review per
change, the debug interface rule, and ADR triggers that fire when someone remembers them.

How it meets the constraints: costs no lines and no passes. The reviewer already reads every
change. Ousterhout's 10 to 20 percent is approximated by whatever time the architect gets.

Cost: none now. Risks: nothing in the loop names the interface a pass must keep, counts
dependencies, or checks a structural property; the reviewer is briefed on the claim most worth
testing, and that claim is about behaviour. ADR triggers are re-read by nobody in particular (ADR
0002 has five deferred items and no owner for them). The critic's steelman is in its review.

Who else: most agent-driven projects; the studies measured them.

Exit cost: none; this is the state to leave. Moving parts: no new ones.

### B. Promised surfaces and deep modules

What it is: each project names its promised surfaces (file formats, wire protocols, public APIs,
command-line contracts: whatever another program or a user's data depends on) in one ADR with a
compatibility promise, in SQLite's and Go's sense: additions are free, an incompatible change
(removal, rename, changed meaning, changed format of existing output) is a major choice and takes
the loop. Everything behind a surface is disposable by design. Modules are deep: a small
interface with much behind it. A brief says what its pass may add and must not change; a change
to a named interface the brief did not allow is a reviewer finding (serious), not a judgment call.

How it meets the constraints: one clause in `CLAUDE.md`, one field in the brief template, one rule
in `reviewer.md`. The reviewer checks a named interface against a diff, which an Opus pass can
do. Hatchery writes its own surfaces ADR first; a project writes one when it has a format or API.

Cost: one ADR per project at design time; a sentence per brief; surfaces and cross-module
interfaces become architect work, module-internal interfaces stay the coder's. Risks: a project
with no external users names nothing and gains nothing until it has data to keep; a surface named
too early is a promise made too early. Mitigation: the promise is scoped (a format may be marked
"until first release") and the ADR says what is not promised.

Who else: SQLite (file format, C API, support planned to 2050), Go (the compatibility promise,
which lets APIs grow and forbids breaking existing code), Ousterhout on deep modules.

Exit cost: delete the surfaces ADR and the brief field; nothing else depends on them. Moving
parts: one ADR per project, one brief field, one reviewer rule.

### C. Executable fitness checks

What it is: architecture properties checked by a program, run by the project's test command and
failing a push like a doc cap (pusher check 8 already runs the test command). The base ships the
contract, not the script: exit 0 pass, 1 fail, 2 usage error, `--only <check>` and `--verbose`
as `.claude/hooks/test-hooks.js` has, one line per failure that names the check, the detail, the
config line it came from and who may lift it; a skip entry in the config, with a reason, a date
and an idea file, lifts one check for one push and prints `SKIP` instead. The checks, in the
order a project earns them: a dependency budget (direct dependencies at most the configured
number, each with its licence; vendored copies count); module boundaries (declared import edges,
no others); a promised-surface compatibility check (the recorded fingerprint of each surface is
a subset of the current one, or the surface's ADR moved).

A failure is design work for the architect (it means a boundary or a promise is wrong, or the
change is), never a patch the coder makes to the check.

How it meets the constraints: nothing a model judges; a program says FAIL. Language-specific, so
the script is the project's; the contract and the first check (dependency budget zero on the
hooks, which need only Node built-ins) are the base's and run on Hatchery today.

Cost: one script per project, maintained like the hook harness; a config file for the budget and
the edges; a fingerprint per surface. Risks: a check that is wrong blocks pushes until an
architect looks; a budget set once is argued about forever. Mitigation: the skip entry, and
raising a budget is an ADR amendment, not an edit.

Who else: ThoughtWorks (Ford, Parsons, Kua: fitness functions), ArchUnit for the JVM,
import-linter for Python, cargo-deny for Rust (licences, bans, advisories; no count cap).

Exit cost: delete the script and its config; the test command shrinks. Moving parts: one script,
one config file, one fingerprint file per surface.

### D. Dependencies: default evaluate, with a checklist; copying in keeps the licence

As first written: boring redefined as what the Opus reviewer can verify line by line, and
dependency default no. As revised (objections 8 and 9): the default is evaluate. A dependency answers a checklist in the `proposal` skill, after Cox: what it does
that cannot be copied in under a page; licence GPL-compatible (ADR 0001); last release, open
issues and response time; maintainers; size of the transitive closure; its tests run here;
behaviour on untrusted input; who watches upstream; how it would be replaced and at what cost;
what the debug surface loses if it misbehaves. One the whole project leans on takes the proposal
loop as now; a small one answers the checklist in the brief. A thing under a page may be copied
in: it keeps its own licence and SPDX identifier, carries an origin line, lives under a path
listed in ADR 0001's components table, and counts against the budget. Reimplementing a parser, a
format, cryptography or date handling is not copying and takes the checklist like a dependency.
Boring technology stays what McKinley meant (known failure modes, operational cost of each new
tool); the reviewer's report says whether it could verify the diff line by line, as a fact about
the review, not as a test of the technology.

How it meets the constraints: no `CLAUDE.md` lines, one file in the `proposal` skill, one brief
field, one reviewer rule.

Cost: a checklist per dependency. Risks: copying a thing that later needs its upstream fixes;
agent-written replacements for library code grow the code the studies measure. Mitigation: the
origin line names the version to diff against; reimplementation is not "copying".

Who else: Go (the proverb; the standard library bias), SQLite (no dependencies), Etsy under
McKinley, Cox (copy for small needs, otherwise inspect, test, isolate, watch).

Exit cost: delete the checklist and the field; existing copied files stay as they are. Moving
parts: one checklist file, one brief field, one reviewer rule.

### E. A standing design pass (replaced by a trigger scan; objection 7)

What it is, as first written: once per build chunk the architect runs a `design-pass` skill over
every ADR's deferred items and predictions (F), the last fitness failures (C), the reviewer
findings marked interface or dependency, and the untriaged ideas, and writes at most one
proposal or idea. As revised: the inputs are scanned by the model that runs `triage` (Sonnet or Opus), at the start
of every triage: deferred items whose trigger has fired, prediction lines past their date or
with their trigger met, and `FAIL` or `SKIP` lines from fitness checks. Each hit becomes one idea
file with a `Plan:` naming the architect; the architect runs only on a hit. The `handoff` skill
runs one grep at every session end for a prediction past its date and lists `triage` in Next
when it finds one. A scheduled Fable pass is a `Question:` for the owner in `docs/ideas/`.

How it meets the constraints: a step in an existing skill, no `CLAUDE.md` lines, no new state
beyond idea files. Fable spend only on a hit; the trigger is defined (a triage run, a due
prediction, a fitness line), not "build chunk".

Cost: a few minutes of a cheaper model per triage. Risks: a trigger that needs judgment to detect
is missed by a weaker model. Mitigation: predictions and fitness lines are grep-able; only ADR
deferred items need reading.

Who else: Ousterhout (10 to 20 percent, which he means as continuous investment by whoever writes
the change: here the brief's Interface field and the coder's stop rule, at Opus price).

Exit cost: delete the step and the grep. Moving parts: one triage step, one handoff grep.

### F. Template and critic changes: horizon, exit cost, moving parts, pre-mortem, predictions

What it is: five edits to the records the loop already produces. Proposal template: the Question
asks what the chosen option should look like in ten years; each option states its exit cost (what
it takes to leave in two years: what is rewritten, what data migrates) and its moving parts (what
runs, is configured or is depended on that was not before). Critic brief: a pre-mortem, "it is
ten years on and this choice is the one the project regrets; say in one paragraph what went wrong
and which line of the proposal should have caught it", and a prediction that cannot fail is an
objection. ADR template: Consequences ends with two or three predictions, each checkable, each
carrying its own check point (a date or a trigger); superseding an ADR records their outcome
first; the scan (E) reads them.

How it meets the constraints: no `CLAUDE.md` lines; four template files and the critic brief.
Costs nothing per change that was not already being written.

Cost: a few lines per proposal and ADR. Risks: horizon and exit-cost prose becomes boilerplate;
predictions written to be safe. Mitigation: the critic's pre-mortem reads them.

Who else: pre-mortems (Klein), the "exit cost" question from boring-technology practice,
hypothesis-style ADRs.

Exit cost: revert five template edits. Moving parts: none that run.

## Recommendation

B, C, D and F into the base now, applied to Hatchery first; E as the trigger scan; A rejected
because the owner asked for practices and the loop names no surface, counts nothing and re-reads
no trigger. The reviewer gains the one practice the evidence does support: drift in input checks
and error handling, and net lines added, reported against the brief.

`CLAUDE.md` gains four lines, no section: the major-choice list gains "an incompatible change to
a promised surface (ADR 0012)", and a two-line pointer says a change is measured by the software
it leaves behind and ADR 0012 names the practices and where each lives. Every other rule lives in
the file loaded when it applies: the brief template (Interface and Dependencies fields), the
`coder` and `reviewer` agent files, the `proposal` skill (horizon, exit cost, moving parts, the
dependency checklist, the pre-mortem), the `adr` skill (predictions), `triage` (the scan),
`handoff` (the grep), `pusher` (a fitness FAIL is design work), and the test command.

Hatchery first: ADR 0013 names Hatchery's own promised surfaces (the policy file, grant files,
the idea block grammar, the `Hatchery base:` line, the usage cache, the hook seams) with what is
not promised, and ships the reference fitness check: dependency budget zero on the hooks and the
status line, in `test-hooks.js`, with the harness case that proves the skip entry.

What the owner decides: whether a scheduled architect design pass is wanted in addition to the
scan (`docs/ideas/2026-10-16-scheduled-architect-design-pass.md`), and each project's surfaces
ADR where it names a format or command users touch (`proposed` until answered). Nothing in the
base's own change alters what humans see.

First implementation step: ADR 0012 (the edit list) and ADR 0013 (Hatchery's surfaces and the
reference check); then a `tester` brief for the edits and a `coder` brief for the check, both
verified by `node .claude/hooks/test-hooks.js` and `wc -l CLAUDE.md`.

## What would change this recommendation

- A reviewer model that catches unnamed dependencies and drift without the brief fields: the
  fields go, the reviewer rule stays.
- Three skip entries for the same check with no ADR amendment behind any of them: the check is
  wrong; it is removed, not tuned.
- A brief corpus in which "Interface: unconstrained" is the majority answer: the field is
  boilerplate; it moves from the template to the `brief` skill's guidance.
- Four lines needed in `CLAUDE.md` for something more urgent: the pointer collapses to one line.
- The owner answers yes to the scheduled pass: E's first shape returns as a skill, cadence theirs;
  a project meant to be thrown away records an opt-out ADR and the base does not change.

## Claims to verify

Checked by the critic against primary sources (`0012-critic-review.md`, Claims checked). Wrong or
overstated as first written, corrected above and in the finding: the CodeThread figure and its
mechanism, the EASE study's direction, the Lefkowitz date and point, Cox on liability, cargo-deny's
count cap, the harness line format. The rest verified, with Go's promise cutting against B as
first written.

## Critic's objections and responses

1. The Question and the finding state a cause the studies do not show (blocking). Accepted: the
   critic read the papers, the coordinator had a summary; both corrected in place, and the
   practices stand on SQLite, Go, Ousterhout and Hatchery's own incidents. Not accepted: that C
   "waits for a project"; objection 6 shows the base has the checkable property today, and C
   stands on fitness-function practice, not on CodeThread.
2. Sixteen lines break the first project where the practices would run (blocking). Verified:
   `wc -l ../regenesis/CLAUDE.md` is 150. Accepted: four lines, a clause and a pointer; every
   other rule in the file loaded when it applies. The sibling still trims four at its update.
3. The supported practice is missing (serious). Accepted: reviewer rule on changed input checks,
   silent fallbacks and swallowed errors the brief did not name; net lines added in the report.
4. Any surface change as a major choice blocks every feature (serious). Accepted: the trigger is
   an incompatible change; additions are free; the compatibility check compares subset, not
   equality; a debug surface is promised only when the surfaces ADR names it.
5. The fitness check refuses with no lift (serious). Accepted: a skip entry in the config (reason,
   date, idea file), added by the coordinator for one push and named in the commit message; the
   `FAIL` line names the lift; pusher check 8 reports it as design work; no check without a skip.
6. The base has surfaces and code now (serious). Accepted: ADR 0013 names Hatchery's surfaces and
   the reference check runs in `test-hooks.js` with the case that proves the skip entry.
7. Fable spend under-flagged, cadence set before the owner answers, "build chunk" undefined
   (serious). Accepted: E becomes the trigger scan in `triage`; the scheduled pass is a
   `Question:` idea; surfaces and cross-module interfaces are the architect's, module-internal
   ones the coder's, whose Interface field is the continuous investment.
8. Default no moves code into agent-written copies and clashes with licensing (serious).
   Accepted: default evaluate; copying under a page keeps its licence and SPDX identifier with an
   origin line, under a path in ADR 0001's living table, counted against the budget;
   reimplementation is not copying; Cox's security and watch questions added.
9. Boring as reviewability is a model judgment that pre-decides language choices (serious).
   Accepted with change: the definition goes; the reviewer's report keeps one yes/no line,
   "verified line by line", naming the hunks it could not verify. Attribution and date fixed.
10. The brief field is ambiguous (minor). Accepted: "Interface: may add X; must not change Y; or
    unconstrained because <reason>".
11. An accepted serious finding is unrecorded; a coder mid-pass has no path (minor). Accepted: the
    commit message names it; the coder stops and reports a needed dependency or interface change.
12. Reversal conditions need counts no file keeps (minor). Accepted: with E gone the pass tally
    is moot; a skip entry is an idea file, so the count of wrongful blocks is a listing.
13. One-year predictions do not fit ADRs that turn over in days (minor). Accepted: each
    prediction carries its own date or trigger; superseding checks them first.
14. "Nothing here changes what humans see" is true for the base only (minor). Accepted: a surfaces
    ADR naming something users touch is `proposed` until the owner answers; ADR 0013 is one.
15. Small factual slips (minor). Accepted: corrected in place above and in the finding.
