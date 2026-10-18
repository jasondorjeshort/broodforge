---
date: 2026-10-18
kind: bug
---

# `git rebase --exec` runs a shell command the guard never reads

`.claude/hooks/git-guard.js`. `git rebase --exec <cmd>` and `-x <cmd>` hand `<cmd>` to a shell
after each replayed commit; the guard reads the rebase, not the command inside it. So `git rebase
-x "git push --force origin main" origin/main` passes every rule the caller is allowed a rebase
under: with the base policy the runner and the main session; with the task-branches example every
subagent on its task branch, and, since the first carry-back (ADR 0019) added `git.integrate`,
on `main` too in that example. Predates that change: the reviewer of the landing ran the old and
new guard on it and both allowed it. The base default exposes it only to callers that may already
push with force to that ref by an explicit refspec, so it is a gap in the reading, not a new
power; the first downstream project that lets more callers rebase widens it.

Fix: the guard reads the argument of `--exec`, `-x` and `--exec=<cmd>` as a command line of its
own (the parser already splits shell text for `sh -c` and heredocs; the hook specs, rule 11 and
the "quoted text is data" note, say where) and applies every rule to it with the same caller and
branch; a harness case per shell. Rule number and wording: the architect, in the hook specs.
