# Working agreement

Read this before doing anything. It governs how Claude works in this repo.
This file comes from the Hatchery base repository; a project decanted from it keeps the rules and
edits the project-specific parts (title, platforms, thresholds, routing) as needed.

Where work stands right now (coordinator state; subagents follow their brief, not this):

@docs/HANDOFF.md

## Phase

Design first. No application code until the design docs in `docs/design/` are written and agreed.

## This agreement is part of the work

Designing the base is itself the project. When a step is awkward, an agent brief is missing
something, a hook misfires, or a better way turns up, fix the protocol in the same sitting:
edit this file, the agent definitions, the skills, or the hooks, and commit the change with the
reason in the commit message. Do not carry process friction forward as tribal knowledge.
This file states rules. A lesson too specific for it goes in `docs/design/decisions/` as an ADR if
it is a choice, or in the relevant skill if it is a procedure; how a mechanism works goes in
`docs/reference/` or the ADR that chose it, with a pointer from here.

## Where things live

`docs/README.md` is the map. Rules live here. Procedures are skills in `.claude/skills/`; load one
with the Skill tool when its description fits the moment. Path-scoped conventions are in
`.claude/rules/`. What we build and why is in `docs/design/`; durable facts in `docs/reference/`;
investigation results in `docs/findings/`; now and next in `docs/HANDOFF.md`; later, or not yet
decided, one dated file each in `docs/ideas/`; steps only the owner can take in `docs/operator/`.
Read a directory's `README.md` before its files. Reasoning: ADRs 0002 and 0007.

## Delegation: agents do all coding and debugging

The main session is a coordinator. Its own work is reading, deciding, briefing (`brief` skill:
goal, files to read, what is already ruled out, scope boundaries, how to verify), reviewing
reports, editing the coordination files (`docs/HANDOFF.md`, the blocks in `docs/ideas/`, a task
board, this file), and the wrap-up commits below. Everything that executes the project goes to the
subagents in `.claude/agents/`, even when it is one command: installs, pulls, builds, test runs,
servers, devices, and every fix. When the gate blocks launches, the coordinator writes the handoff and stops; it
does not do the delegated work itself.

## Debug interface is part of every feature

Agents cannot click a GUI. Every feature ships with an agent-usable way to exercise it: a CLI
entry point, an HTTP endpoint, a REPL command, or a scripted harness. "Done" means an agent can
drive the feature, observe its state, and reproduce a bug from a terminal. The debug interface is
designed up front in `docs/design/05-debug-interface.md` (`debug-interface` skill), not bolted on.

## Major decisions get a proposal, a critic, and an ADR

A major choice is any of: language, platform UI toolkit, cross-platform strategy, data storage,
network protocol or API surface, build system, or a dependency the whole project will lean on.
The targets are listed in `docs/design/06-platforms.md`. A major choice is never made in passing:
the `architect` writes a proposal in `docs/design/proposals/`, the `critic` argues against it, the
architect answers each objection in writing and records an ADR in `docs/design/decisions/`, and
only then does implementation start (`proposal` and `adr` skills). Unresolved disagreement goes
to the user with both positions stated.

Technical decisions do not need the user's sign-off; the coordinator makes the call once the
objections are answered and the ADR is recorded, and reports it. Decisions about what humans see and touch (the front end, flows, wording,
visuals) do need the user, because their feedback is the ground truth there. Flag those
explicitly and do not guess.

## Usage gating (hard rule)

Plan limits are the 5-hour session meter and the weekly meter (`/usage`). The status line shows
both and flags HOLD and STOP at the thresholds in `.claude/hooks/policy.json` (a project sets its
own there, never in the JS); the `usage-gate` hook refuses new subagent launches from HOLD. The
gate blocks launches only, never Bash. How the meters reach the hook, staleness, and the owner's
override: ADR 0006 and `docs/design/decisions/0002-hook-specs.md`.

| Usage (the worse meter, each judged against its own thresholds) | Behavior |
|---|---|
| below HOLD (base: 50%) | Normal operation. |
| HOLD to STOP (base: 50% to 100%) | No new work, subagents or tasks, unless the owner's override file lifts HOLD (ADR 0006). Otherwise let in-flight agents finish, run the `handoff` skill, commit from the main session, then stop and report. |
| STOP and above (base: 100%, a ceiling only a project that forbids overage lowers) | Stop. Run the `handoff` skill, commit in-progress work from the main session with a clear WIP message, end the turn. |

Meters show only on Pro and Max plans, after the first response; on `n/a`, run `/usage` first.

## Model routing

Pick the subagent by task; each one pins its own model in `.claude/agents/`.

| Task | Agent | Model | Why |
|---|---|---|---|
| Design docs, architecture, ADRs, planning, organizing work, design review | `architect` | Fable 5.1 | Strongest reasoning; worth the cost where judgment matters most. |
| Adversarial second opinion on a proposal or major technical choice | `critic` | Fable 5.1 | Must be able to find what the architect missed; a weaker critic rubber-stamps. |
| Implementation, debugging, non-trivial refactors, building debug interfaces | `coder` | Opus 5.5 | Best coding model at Opus price. Effort pinned to high because its default is medium. |
| Writing tests, fixing failing tests, small well-specified code changes | `tester` | Sonnet 5.5 | Fast and capable for scoped work at a fifth of Fable's cost. |
| Running test suites, lint, format, commits, mechanical file ops | `runner` | Haiku 5.5 | Cheapest by far; these tasks need reliability, not depth. |
| Independent review of a diff or unpushed commit; reports, does not fix | `reviewer` | Opus 5.5 | Must read code as well as the coder wrote it; a weaker reviewer rubber-stamps. |
| Pre-push sanity checks and the push itself | `pusher` | Sonnet 5.5 | Needs enough judgment to read test output and spot secrets, not enough to justify Opus. |

## One independent review per change

Every change is read once by an agent that did not write it before it is pushed: `reviewer` for
code and docs, `critic` for a proposal. The coordinator reads the report, not the diff, and
routes fixes back to the writer; a fix to an unpushed commit is amended into it. One review, not
three: the writer's own check and the pusher's mechanical checks are not reviews, and a second
reading of the same diff by the same model finds what the first did. The user reviews what humans
see and touch, and anything the reviewer marked blocking that the coordinator overrode.

## Git

- Branch: `main`. Commit small and often, as work lands. Below HOLD the `runner` commits from a
  brief (`commit` skill); `coder` and `tester` leave a clean, described working tree for it.
  Wrap-up commits (`docs/HANDOFF.md`, WIP) are made by the main session at any usage level.
- Amend only to fold a review fix, a forgotten file or a message fix into the unpushed commit it
  belongs to. Never amend a pushed commit.
- The `pusher` pushes, launched fresh for each push with the user's request quoted in the brief
  (a resumed pusher's retry gets blocked; the `commit` skill says why); the main session may push
  when the user asked for the push in so many words. Never force-push. A push that is blocked
  anyway goes back to the user to run `git push` themselves; never route around the block.
- The `git-guard` hook denies destructive git for everyone, agents included, in Bash and
  PowerShell alike. Which agents may commit and push is policy in `.claude/hooks/policy.json`
  (the base: `git commit` only from the `runner`, `git push` only from the `pusher`, the main
  session exempt from both); a project with its own branch or push model edits the policy file,
  never the hook. The denied list and each hook's fail mode: ADR 0006 and the hook specs.
- Commit messages end with `Co-Authored-By: <model that did the work> <noreply@anthropic.com>`,
  for example `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`.

## Licensing

Fully open source. Default license is GPL-3.0-or-later (`LICENSE`). A component that is meant to be
linked into other people's software may be LGPL-3.0-or-later instead; such a component carries its
own `LICENSE` file in its directory and is listed in `docs/design/decisions/0001-licensing.md`.
Every source file starts with an SPDX header: `SPDX-License-Identifier: GPL-3.0-or-later` (or `LGPL-3.0-or-later`).
Dependencies must be GPL-compatible. See the ADR for the reasoning.
