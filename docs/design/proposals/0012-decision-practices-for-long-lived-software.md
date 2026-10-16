# 0012. Decision practices for long-lived software

Date: 2026-10-16
Status: under review

Written by the architect from the coordinator's research
(`docs/findings/2026-10-16-decision-practices-for-long-lived-software.md`), on the owner's words:
optimise for long-term simplicity and sustainability, "actual software that does things, not
churning out code". This proposal applies the practices it proposes: its question carries the
ten-year horizon, and every option states its exit cost and moving parts.

## Question

Hatchery's loop makes the few major choices carefully (proposal, critic, ADR) and the many small
ones quickly (a brief, a coder pass, one defect review). The 2026 studies in the finding say the
cost of agent-built software is paid later, in work that builds on it, and that the cause is
structural (interfaces that widened, dependencies that accreted, modules that went shallow), none
of it a defect on the day it landed. What practices should the base add so that a project it
starts is still doing its job in ten years with little upkeep, and where does each live: the
every-turn file, a skill or template, an agent file, or the first decanted project that has code?
In ten years the answer should look like this: each project has a short list of promised surfaces
that have not broken, a dependency list that fits on a screen, and a `git log` in which rewrites
behind the surfaces were cheap and ordinary.

## Constraints that matter here

- `CLAUDE.md` is capped at 150 lines and holds 134 (`.claude/rules/docs.md`): sixteen lines,
  including the blank one that separates sections. Rules go there; procedures go in skills;
  reasoning goes in ADRs (ADR 0002).
- Agents write all the code and the reviewer is Opus (ADR 0007). A practice is real only when an
  agent can apply or check it from a terminal; a practice that lives in judgment alone drifts with
  the next model.
- The base has no application code, toolchain or test runner (ADR 0002, Not in the base). A check
  that needs a language waits for a project; the base can ship its contract and its slot.
- Fable costs two and a half times Opus. A practice that spends architect passes spends the
  owner's plan, and the cadence is the owner's to set (`CLAUDE.md`, Major decisions).
- Living docs state the practice, not the project it came from; this proposal is a closed record
  and may cite sources freely.
- The debug interface rule stands: a fitness check is a debug surface for the architecture and
  obeys the same exit codes (`debug-interface` skill).

## Options

Each option is one practice. The recommendation combines them; the baseline is given the same
depth so that rejecting it is a decision, not an omission.

### A. Change nothing: the current loop is enough

What it is: the major-choice loop for the choices listed in `CLAUDE.md`, one defect review per
change, the debug interface rule, and ADR triggers that fire when someone remembers them.

How it meets the constraints: costs no lines and no passes. The reviewer already reads every
change. Ousterhout's 10 to 20 percent is approximated by whatever time the architect gets.

Cost: none now. Risks: the finding's point 1 is exactly the gap. Nothing in the loop names the
interface a pass must keep, counts dependencies, or checks a structural property; the reviewer
is briefed on the claim most worth testing, and that claim is about behaviour. ADR triggers are
re-read by nobody in particular (ADR 0002 has five deferred items and no owner for them).

Who else: most agent-driven projects; the studies measured them.

Exit cost: none; this is the state to leave. Moving parts: no new ones.

### B. Promised surfaces and deep modules

What it is: each project names its promised surfaces (file formats, wire protocols, public APIs,
command-line contracts: whatever another program or a user's data depends on) in one ADR with a
compatibility promise, in SQLite's sense. Everything behind a surface is disposable by design.
A change to a surface is a major choice and takes the loop. Modules are deep: a small interface,
reviewed by the architect when designed and by a human when it is a surface, with much behind it.
A brief names the interface its pass must not widen; a widening the brief did not name is a
reviewer finding (serious), not a judgment call.

How it meets the constraints: two lines in `CLAUDE.md`, one field in the brief template, one rule
in `reviewer.md`. The reviewer checks a named interface against a diff, which an Opus pass can
do. The surfaces ADR waits for a project.

Cost: one ADR per project at design time; a sentence per brief; interface design becomes an
architect task where today it is the coder's smallest reasonable choice. Risks: a project with no
external users names nothing and gains nothing until it has data to keep; a surface named too
early is a promise made too early. Mitigation: the promise is scoped (a format may be marked
"until first release") and the ADR says what is not promised.

Who else: SQLite (file format, C API, support planned to 2050), Go (the compatibility promise),
Ousterhout on deep modules.

Exit cost: delete the surfaces ADR and the brief field; nothing else depends on them. Moving
parts: one ADR per project, one brief field, one reviewer rule.

### C. Executable fitness checks

What it is: architecture properties checked by a program, run by the project's test command and
failing a push like a doc cap (pusher check 8 already runs the test command). The base ships the
contract, not the script: one program, exit 0 pass, 1 fail with one line per failure
`FAIL <check>: <detail>`, 2 usage error, with `--only <check>` and `--verbose`, as
`.claude/hooks/test-hooks.js` has. The first three checks, written when a project has code:

- dependency budget: direct dependencies at most the number in the project's fitness config, and
  every one listed with its license;
- module boundaries: the allowed import edges are declared and the actual graph has no others;
- promised-surface change without a version bump: a recorded fingerprint of each surface (a
  schema, a help text, a header) must match, or the surface's version line in the ADR moved.

A failure is design work for the architect (it means a boundary or a promise is wrong, or the
change is), never a patch the coder makes to the check.

How it meets the constraints: Needle in the Repo's own method, executable structural probes,
turned into the project's suite. Nothing a model judges; a program says FAIL. Language-specific,
so the script is the project's; the contract and the three checks are the base's.

Cost: one script per project, maintained like the hook harness; a config file for the budget and
the edges; a fingerprint per surface. Risks: a check that is wrong blocks pushes until an
architect looks; a budget set once is argued about forever. Mitigation: the check names its
config line in the failure, and raising a budget is an ADR amendment, not an edit.

Who else: ThoughtWorks (Ford, Parsons, Kua: fitness functions), ArchUnit for the JVM,
import-linter for Python, cargo-deny for Rust.

Exit cost: delete the script and its config; the test command shrinks. Moving parts: one script,
one config file, one fingerprint file per surface.

### D. Boring technology as reviewability, and dependency default no

What it is: boring is redefined for an agent loop. The sound part of the boring-technology
argument is known failure modes; the agent-era test is whether the Opus reviewer can verify,
line by line, what was generated in that technology. A language or framework it cannot check is
not boring here whatever its age. A dependency is a liability (Cox): the default is no, a small
thing is copied in with its license header, and one that stays answers a checklist in the
`proposal` skill: what it does that cannot be copied in under a page; license GPL-compatible
(ADR 0001); last release and response to issues; maintainer count; size of the transitive
closure; its tests run here; the reviewer knows it; how it would be replaced and what that costs;
what the debug surface loses if it misbehaves. A dependency the whole project leans on takes the
proposal loop as now; a small one answers the checklist in the brief, and a new dependency the
brief did not name is a reviewer finding.

How it meets the constraints: two lines in `CLAUDE.md`, one file in the `proposal` skill, one
brief field, one reviewer rule. Reviewability is checkable: the reviewer says so in its report.

Cost: a checklist per dependency; some things that would be a one-line install become a copied
file to maintain. Risks: copying a thing that later needs its upstream fixes; "the reviewer can
check it" is a claim about a model and moves as models do. Mitigation: a copied file records its
origin and version in its header; the reviewability claim is re-read at the design pass (E).

Who else: Go (the proverb; the standard library bias), SQLite (no dependencies), Etsy under
McKinley, every project that has had a left-pad week.

Exit cost: delete the checklist and the field; existing copied files stay as they are. Moving
parts: one checklist file, one brief field, one reviewer rule.

### E. A standing design pass

What it is: Ousterhout's continuous investment made into a scheduled pass rather than a phase.
Once per build chunk, before the chunk's briefs, the architect runs a `design-pass` skill: read
every ADR's deferred items and predictions (F), the last test run's fitness failures (C), the
reviewer findings marked interface or dependency since the last pass, and the untriaged ideas;
write at most one proposal or one idea with a `Plan:`, and report which triggers fired and which
predictions failed. A failed prediction is a finding; a fired trigger is a proposal.

How it meets the constraints: a skill (procedure), two lines in `CLAUDE.md` (the rule and the
cadence), no new state: the inputs already exist in the tree. It is Fable spend; the cadence is
the owner's number, and this proposal recommends one but does not set it.

Cost: one architect pass per chunk, bounded by its output (one artefact). Risks: a pass that
finds nothing is pure cost; a pass that finds too much becomes a redesign. Mitigation: the output
cap, and the reversal condition below (three passes with nothing to report).

Who else: Ousterhout (10 to 20 percent), any team with an architecture review cadence.

Exit cost: delete the skill and the two lines. Moving parts: one skill, one handoff line per
chunk.

### F. Template and critic changes: horizon, exit cost, moving parts, pre-mortem, predictions

What it is: five edits to the records the loop already produces. Proposal template: the Question
asks what the chosen option should look like in ten years; each option states its exit cost (what
it takes to leave in two years: what is rewritten, what data migrates) and its moving parts (what
runs, is configured or is depended on that was not before). Critic brief: a pre-mortem, "it is
ten years on and this choice is the one the project regrets; say in one paragraph what went wrong
and which line of the proposal should have caught it". ADR template: Consequences ends with two
or three predictions, each checkable, about what will be true in a year if the decision is right;
the design pass (E) checks them.

How it meets the constraints: no `CLAUDE.md` lines; four template files and the critic brief.
Costs nothing per change that was not already being written. The predictions are what make E
checkable rather than a re-read.

Cost: a few lines per proposal and ADR. Risks: horizon and exit-cost prose becomes boilerplate;
predictions written to be safe. Mitigation: the critic's pre-mortem reads them, and a prediction
that cannot fail is an objection.

Who else: pre-mortems (Klein), the "exit cost" question from boring-technology practice,
hypothesis-style ADRs.

Exit cost: revert five template edits. Moving parts: none that run.

## Recommendation

B, D and F into the base now; C and E into the base as contract and skill, with their first
real run at the first decanted project that has code; A rejected because the evidence places
the cost exactly where the current loop does not look. Where each line goes:

`CLAUDE.md`, a new section between Major decisions and Usage gating, fifteen lines plus the
separating blank, so the file ends at 150; a later addition anywhere displaces something:

```
## Built to last

The measure of a change is the software it leaves behind, not how fast it landed (ADR 0012).

- Each project names its promised surfaces (the file formats, APIs and commands others depend
  on) in one ADR with a compatibility promise; everything behind them may be rewritten, and a
  change to a surface is a major choice. A brief names the interface its pass must not widen;
  a widening the brief did not name is a reviewer finding.
- A dependency is a liability, default no: the checklist in the `proposal` skill decides, and a
  small thing is copied in with its license rather than depended on.
- Boring means the Opus reviewer can verify what was generated, line by line; age is no test.
- Fitness checks run with the test command and fail a push like a cap; a failure is design work
  for the architect, not a patch for the coder.
- One architect design pass per build chunk, before its briefs: it re-reads ADR triggers and
  predictions and fitness failures, and writes at most one proposal or idea.
```

Skills, templates and agent files (settled edits; `tester` work once the ADR exists):

- `.claude/skills/proposal/template.md`: Question becomes "what is being decided and why now, and
  what the chosen option should look like in ten years"; each option gains the lines "Exit cost:
  ..." and "Moving parts: ...". `SKILL.md` step 2 names both as required.
- `.claude/skills/proposal/brief-critic.md`: the pre-mortem paragraph from F, after the lift
  question.
- `.claude/skills/proposal/dependency.md` (new): the checklist from D, one question per line, and
  the rule that the answers go in the brief for a small dependency and in the proposal for one the
  project leans on. `SKILL.md` points at it.
- `.claude/skills/adr/template.md`: Consequences gains "Predictions: two or three checkable
  statements about what is true in a year if this is right; the design pass checks them."
- `.claude/skills/brief/template.md`: after Scope, "Interface: <the module interface or promised
  surface this pass must not widen; or none>. Dependencies: <none, or each new one with the
  checklist answered>." The `brief` skill's coder note says both fields are required.
- `.claude/agents/reviewer.md`: one rule: a widened interface the brief did not name, a new
  dependency the brief did not name, or a promised-surface change with no ADR, is a serious
  finding; the report says whether the reviewer could verify the generated code line by line.
- `.claude/skills/design-pass/SKILL.md` (new): the procedure from E; inputs, the output cap, the
  report shape. `architect.md` lists it.
- `docs/design/README.md`: the `03-architecture.md` row gains "promised surfaces, with the
  compatibility promise, and the fitness checks" so a decanted project knows the slot.

Waits for the first decanted project with code: the surfaces ADR, the fitness script and its
config (the contract in C is in the ADR for it to build from), the dependency budget number, and
the first design pass. The base ships none of these as files; the post-edit hook has nothing to
check until they exist.

What the owner decides: the design-pass cadence is Fable spend; one per chunk is the
recommendation and the ADR records the owner's answer. Nothing here changes what humans see.

First implementation step: the critic pass on this proposal, then ADR 0012 with the `CLAUDE.md`
block and the edit list above as its Decision, then one `tester` brief that makes the edits and
runs `node .claude/hooks/test-hooks.js` (the post-edit cap check must still pass at 150 lines).

## What would change this recommendation

- The critic finds the 2026 studies do not say what the finding says (a retracted preprint, a
  13.1 percent that is one benchmark's one configuration): B, D and F stand on the older sources
  alone; C loses its strongest argument and waits for a project to ask for it.
- Three design passes in a row with nothing to report: E's cadence drops to on-trigger only.
- A fitness check that has blocked three pushes with no design change behind any of them: the
  check is wrong; C's contract gets a disable-with-reason line.
- A reviewer model that catches unnamed interface widening without the brief field: the field
  goes, the reviewer rule stays.
- Sixteen lines needed in `CLAUDE.md` for something more urgent: the section collapses to its
  first line and a pointer to the ADR; the rules stay in the ADR and the skills.
- A decanted project meant to be thrown away: it records an opt-out ADR; the base does not change.

## Claims to verify

- CodeThread (arxiv 2606.21804, summary at emergentmind.com/papers/2606.21804): later-task
  resolution up to 13.1 percent lower on agent-written than on human-written code; check the
  figure, the benchmark and whether the effect is on agent-written code or agent-written history.
- Needle in the Repo (arxiv.org/pdf/2603.27745): the maintainability definition and that its
  probes are executable; check they are structural rather than test-based.
- SWE-CI: regressions under long-term maintenance; find the paper and the figure.
- EASE 2026 study on the AIDev dataset: what it measured; the coordinator noted no figure.
- sqlite.org/lts.html: support planned to 2050; the file format and C API as the promised
  surfaces; the warning against fashion.
- Ousterhout, A Philosophy of Software Design: deep modules, tactical versus strategic, 10 to
  20 percent, complexity as dependencies plus obscurity (chapters 2, 3, 4).
- boringtechnology.club (McKinley), Lefkowitz's Against Innovation Tokens (blog.glyph.im) and the
  2025 revisiting post: that the defensible claim is known failure modes, not a token count.
- research.swtch.com/deps (Cox) and go-proverbs.github.io: the evaluation list and the proverb.
- Fitness functions: Ford, Parsons, Kua, Building Evolutionary Architectures; ArchUnit,
  import-linter and cargo-deny exist and do what C says.
- Go's compatibility promise (go.dev/doc/go1compat) as a named-surfaces precedent.
- `CLAUDE.md` is 134 lines and the block above is fifteen plus one blank (`wc -l`).

## Critic's objections and responses

(none yet)
