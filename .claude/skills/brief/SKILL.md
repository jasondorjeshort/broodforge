---
name: brief
description: Write a complete, short brief for launching a subagent (architect, critic, coder, tester, reviewer, runner, pusher). Use when delegating work, launching an agent, or when a previous agent report shows the brief was missing context.
---

# Brief

The coordinator launches agents several times an hour. A complete brief is the difference between
a result and a second round. Agent rules live in `.claude/agents/<name>.md` and are not restated;
the brief supplies what only this task knows.

## Steps

1. Pick the agent by task (routing table in `.claude/agreement/model-routing.md`). If the task mixes kinds (design plus
   code), split it into two briefs in sequence.
2. Fill `template.md` in this directory. Every field is required; write "none" rather than
   leaving one out.
3. Files to read are given as index paths (`docs/reference/README.md`, `docs/design/README.md`),
   not as a list of everything; name specific files only when the agent must read them in order.
4. "Already ruled out" prevents re-investigation: list what was tried and why it failed, with the
   finding path when there is one.
5. Scope boundary names what the agent must not touch, in paths.
6. Verification is a command or an observable, never "make sure it works".
7. Report format tells the agent what the coordinator needs back, so the report is short and the
   next brief can be written from it.

## Per-agent notes (one line each; the rest is in the agent file)

- architect: say whether the output is a design doc, an ADR, a proposal, or a review; give the
  line cap.
- critic: use `brief-critic.md` from the `proposal` skill.
- coder: name the design section and the debug-interface section; say "use the Write tool for
  files with apostrophes"; fill Interface and Dependencies; a surface or cross-module interface
  is the architect's to design, a module-internal one the coder's.
- tester: name the behaviours to cover and the test command; say whether failing tests are
  expected.
- reviewer: name the claim most worth testing (a destructive step, a safety property); experiments
  in the scratchpad are expected and it may run them unasked, but naming the claim focuses the review.
- runner: give exact commands or the commit brief from the `commit` skill; nothing to interpret.
- pusher: use the one-line brief from the `commit` skill, with the user's request quoted.

## Rules

- A brief that carries a claim about what a destructive or safety-relevant command does (a delete,
  a removal procedure, a guard) says "test it in isolation or drop it". An untested claim marked
  "untested" is not a resolution: a later reader takes the procedure as safe.
- Route by what the task needs, not by what it touches. A settled edit from a complete brief, even
  to docs, skills or hooks, is `tester` or `coder` work; the `architect` is for the decision itself
  (a proposal, an ADR, answers to the critic, a design doc, a plan). Fable costs two and a half
  times Opus and five times Sonnet, and a settled edit does not need it.
- Under 40 lines. If the brief needs more, the task is too big; split it.
- Do not paste the handoff or design docs into the brief; they are imported or indexed.
- Say which model attribution the commit will carry if the agent's output will be committed by
  someone else.
- Paths in a brief use forward slashes (`C:/Users/...`) even on Windows: the Bash tool is Git Bash
  and strips the backslashes from an unquoted path. A command meant for another repository says
  `git -C <path>`, never `cd` (`docs/reference/claude-code-on-windows.md`, Working directory and git).
- Name only tools the agent has (`.claude/agents/<name>.md`): the runner has no Write tool, so a
  file it needs is written by the coordinator and the brief gives the path.
