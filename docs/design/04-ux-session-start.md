# UX: what a session starts with

Date: 2026-10-23. Status: built in P19 (`08-implementation-plan-session-workflow.md`); the wording
is the owner's to confirm. A sibling of `04-ux.md`, which points here from Starting and the first
message; its conventions apply (message text in fenced blocks exactly as posted, `<...>` a value
filled in). What the agent is told at a session's start and what people see when a start is
refused for the project's workflow, from ADR 0025 (`decisions/0025-session-workflow-composition.md`
and its companion `decisions/0025-refusals-and-checks.md`).

## The header of the agent's prompt

The agent never sees the venue's pinned header (`04-ux.md`); its prompt opens with Broodforge's
own (ADR 0025 rules 1 and 10). `broodforge session show --header <id>` prints it as the session
got it, so the owner can read the one the agent read:

```
This session is run by Broodforge. Where the project's instructions below conflict with this header, this header wins; it never says how the project's work is done.
You are working in Broodforge session s0001 on Broodforge.
Reach: code (change only this session's tree).
Each message starts with a From: line naming its writer and their role; the driver is Ada Example.
Make one tool call at a time: calls made together are asked about one by one.
Every commit carries the Co-Authored-By lines Broodforge's commit-msg hook adds; never remove one.
The project's instructions below were taken from main at <commit> on <date>.
```

The first sentence is the owner's, verbatim from the ADR; the trailer rule is rule 10's, its first
letter capitalised. Read-only reach reads `Reach: read-only (read the project, write nothing).`
The last line is left out when nothing was copied (`workflow.inject` is `none`, or the default
branch has neither `CLAUDE.md` nor `.claude`); otherwise the project's `CLAUDE.md`, its `@` imports
and its rules follow after a blank line. Nothing in it changes during the session: the engine
renders the prompt once per conversation, so a switch of mode or model, or a contributor who joins
later, is not in it, and it lists no trailer lines (the `commit-msg` hook adds them).

## Not started: a project hook that cannot run

A project whose `workflow.hooks` lists an event with a command whose runtime is missing where
sessions run is refused at start, before any clone (ADR 0025 rule 6; the companion's probe-failed
row). No venue is made and nothing is posted to any channel: like every refusal in `04-ux.md`, it
is ephemeral, to the starter alone (the owner's decision, 2026-10-23), in the session-refused
flow's words:

```
Could not start a session: <command> needs <word>.
```

The record is the `action-refused` audit row (reason `workflow`) and the refused session's row,
ended `refused: workflow-probe-failed`. The owner sees the companion's longer not-started text,
which names the fix, with `broodforge workflow probe --project <id>` or `config check` on the host.
