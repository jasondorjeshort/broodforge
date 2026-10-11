---
name: debug-interface
description: Design and verify the agent-usable debug surface for one feature (CLI, HTTP endpoint, REPL command, or scripted harness). Use when designing a feature, briefing the coder on a feature, or checking whether a feature counts as done.
---

# Debug interface

Agents cannot click a GUI. A feature is done when an agent can drive it, observe its state, and
reproduce a bug from a terminal. The surface is designed before the feature, recorded in
`docs/design/05-debug-interface.md`, and verified by running it.

## Design (architect, or coordinator when briefing)

For the feature, write a section in `docs/design/05-debug-interface.md` with:

1. Surface: one of CLI entry point, HTTP endpoint, REPL command, scripted harness. Prefer the one
   the feature already needs for production; add a harness only when none exists.
2. Drive: the exact commands or requests that exercise every behaviour in the design, with
   arguments and example input.
3. Observe: how state is read back (a `status`/`dump` command, a GET endpoint, a log at a known
   path, a structured output flag such as `--json`). Output must be stable text an agent can grep.
4. Reproduce: how a failing case is captured as a command line or a fixture file so a bug report is
   a command, not a description.
5. Isolation: how to run against a scratch state (temp dir, in-memory store, env var) so runs do
   not touch real data.

## Build and verify (coder, tester)

- Build the surface with the feature, not after. Exit codes: 0 success, 1 failure with a
  one-line reason on stderr, 2 usage error.
- Run every command in the Drive list and paste the output in the report. A command that is
  documented but not run is a bug.
- When debugging, reproduce through this surface first, fix, then show the same command passing.
  Report the root cause.

## Checklist for "done"

- Section exists in `05-debug-interface.md` and matches what was built.
- Each design behaviour has at least one Drive command.
- An agent with only Bash can run the Drive list on a clean checkout, following the doc alone.
- A failing case can be reproduced by one command.
