---
name: brief
description: Write a complete, short brief for launching a subagent (architect, critic, coder, tester, runner, pusher). Use when delegating work, launching an agent, or when a previous agent report shows the brief was missing context.
---

# Brief

The coordinator launches agents several times an hour. A complete brief is the difference between
a result and a second round. Agent rules live in `.claude/agents/<name>.md` and are not restated;
the brief supplies what only this task knows.

## Steps

1. Pick the agent by task (routing table in `CLAUDE.md`). If the task mixes kinds (design plus
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
  files with apostrophes".
- tester: name the behaviours to cover and the test command; say whether failing tests are
  expected.
- runner: give exact commands or the commit brief from the `commit` skill; nothing to interpret.
- pusher: use the one-line brief from the `commit` skill, with the user's request quoted.

## Rules

- Under 40 lines. If the brief needs more, the task is too big; split it.
- Do not paste `docs/HANDOFF.md` or design docs into the brief; they are imported or indexed.
- Say which model attribution the commit will carry if the agent's output will be committed by
  someone else.
