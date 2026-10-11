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
Lessons that are too specific for this file go in `docs/design/decisions/` as an ADR if they
are a choice, or in the relevant skill if they are a procedure.

## Where things live

`docs/README.md` is the map. Rules live here. Procedures are skills in `.claude/skills/`; load one
with the Skill tool when its description fits the moment. Path-scoped conventions are in
`.claude/rules/`. What we build and why is in `docs/design/`; durable facts in `docs/reference/`;
investigation results in `docs/findings/`; now and next in `docs/HANDOFF.md`; later in
`docs/backlog.md`. Read a directory's `README.md` before its files. Reasoning: ADR 0002.

## Delegation: agents do all coding and debugging

The main session is a coordinator. It plans, briefs, reviews results, and talks to the user.
It does not write or debug application code itself. All coding and all debugging go to subagents
defined in `.claude/agents/`. Brief each one fully (`brief` skill): goal, files to read, what is
already ruled out, scope boundaries, how to verify.

## Debug interface is part of every feature

Agents cannot click a GUI. Every feature ships with an agent-usable way to exercise it: a CLI
entry point, an HTTP endpoint, a REPL command, or a scripted harness. "Done" means an agent can
drive the feature, observe its state, and reproduce a bug from a terminal. The debug interface is
designed up front in `docs/design/05-debug-interface.md` (`debug-interface` skill), not bolted on.

## Major decisions get a proposal, a critic, and an ADR

A project may span several languages and platforms; the targets are listed in
`docs/design/06-platforms.md`.
Choosing the language, toolkit, and API for each part matters more than almost anything else,
so major choices are planned heavily and never made in passing. A major choice is any of:
language, platform UI toolkit, cross-platform strategy, data storage, network protocol or API
surface, build system, or a dependency the whole project will lean on.

Process (`proposal` and `adr` skills):
1. `architect` writes a proposal: the options considered, the constraints that matter here,
   a recommendation, and what would change the recommendation. Saved under
   `docs/design/proposals/NNNN-title.md`.
2. `critic` reads it and argues against it, verifying claims and steelmanning the alternatives.
3. `architect` revises, answers each objection in writing, and records the result as an ADR in
   `docs/design/decisions/`. Unresolved disagreement goes to the user with both positions stated.
4. Only then does implementation start.

Technical design decisions do not need the user's sign-off; the coordinator makes the call
after step 3 and reports it. Decisions about what humans see and touch (the front end, flows,
wording, visuals) do need the user, because the end goal is a front end that humans use and
their feedback is the ground truth there. Flag those explicitly and do not guess.

## Usage gating (hard rule)

Plan limits are the 5-hour session meter and the weekly meter. Check them with `/usage`.
The status line (`.claude/statusline.js`) shows both and flags HOLD at 50% and STOP at 80%.
A `PreToolUse` hook (`.claude/hooks/usage-gate.js`) refuses to launch new subagents once the
higher meter reaches 50%. The hook reads `~/.claude/usage-state.json`, which the status line writes.
The gate blocks subagent launches only, never Bash: wrap-up commits (`docs/HANDOFF.md`, WIP) are
made by the main session itself at any usage level.

| Usage (max of session, weekly) | Behavior |
|---|---|
| below 50% | Normal operation. |
| 50% to 80% | No new work. No new subagents or tasks. Finish what is in flight, run the `handoff` skill, commit from the main session, then stop and report. |
| 80% and above | Stop. Run the `handoff` skill, commit in-progress work from the main session with a clear WIP message, end the turn. |

Status line usage numbers exist only on claude.ai Pro and Max plans and only after the first
response of a session. If they read `n/a`, run `/usage` by hand before launching agents.

## Model routing

Pick the subagent by task; each one pins its own model in `.claude/agents/`.

| Task | Agent | Model | Why |
|---|---|---|---|
| Design docs, architecture, ADRs, planning, organizing work, final review | `architect` | Fable 5.1 | Strongest reasoning; worth the cost where judgment matters most. |
| Adversarial second opinion on a proposal or major technical choice | `critic` | Fable 5.1 | Must be able to find what the architect missed; a weaker critic rubber-stamps. |
| Implementation, debugging, non-trivial refactors, building debug interfaces | `coder` | Opus 5.5 | Best coding model at Opus price. Effort pinned to high because its default is medium. |
| Writing tests, fixing failing tests, small well-specified code changes | `tester` | Sonnet 5.5 | Fast and capable for scoped work at a fifth of Fable's cost. |
| Running test suites, lint, format, commits, mechanical file ops | `runner` | Haiku 5.5 | Cheapest by far; these tasks need reliability, not depth. |
| Pre-push sanity checks and the push itself | `pusher` | Sonnet 5.5 | Needs enough judgment to read test output and spot secrets, not enough to justify Opus. |

Price reference (per million tokens, input/output): Fable 5.1 $10/$50, Opus 5.5 $4/$20,
Sonnet 5.5 $2/$10, Haiku 5.5 $0.10/$0.50.

## Git

- Branch: `main`. Commit small and often, as work lands. Below 50% usage the `runner` commits
  normal work from a brief (`commit` skill); the `coder` and `tester` agents leave a clean,
  described working tree for it. Wrap-up commits (`docs/HANDOFF.md`, WIP) are made by the main
  session itself at any usage level.
- Amend only when necessary (fixing a message, folding a forgotten file into the commit it belongs
  to) and only on commits that have not been pushed. Never amend a pushed commit.
- Only the `pusher` agent (or the user) pushes. It runs the checks in its agent file first and
  never force-pushes. Launch it fresh for each push, with the user's request quoted in the brief;
  a retry sent to a resumed pusher arrives as an agent message, which the permission system treats
  as untrusted, and the push gets blocked. If a push is blocked anyway, tell the user and let them
  run `git push` themselves; never route around the block. Procedure, including the fix for a
  mismatched stored GitHub credential: `commit` skill, Pushing section.
- The `git-guard` hook denies destructive git for everyone, agents included (force push, `reset --hard`,
  `clean`, whole-tree `checkout`/`restore`, `branch -D`, `rebase -i`, stash drop/clear, history rewrites),
  plus `git push` from any agent but the `pusher` and `git commit` from any agent but the `runner`.
- Commit messages end with `Co-Authored-By: <model that did the work> <noreply@anthropic.com>`,
  for example `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`.

## Licensing

Fully open source. Default license is GPL-3.0-or-later (`LICENSE`). A component that is meant to be
linked into other people's software may be LGPL-3.0-or-later instead; such a component carries its
own `LICENSE` file in its directory and is listed in `docs/design/decisions/0001-licensing.md`.
Every source file starts with an SPDX header: `SPDX-License-Identifier: GPL-3.0-or-later` (or `LGPL-3.0-or-later`).
Dependencies must be GPL-compatible. See the ADR for the reasoning.
