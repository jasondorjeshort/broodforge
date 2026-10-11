---
name: handoff
description: Rewrite docs/HANDOFF.md so the next session can resume without asking. Use when wrapping up, when the usage gate reports HOLD or STOP, before a long pause, after finishing or re-planning a chunk of work, or when asked for a handoff.
---

# Handoff

`docs/HANDOFF.md` is imported into every session and every subagent by `CLAUDE.md`, so it is the
one file that is guaranteed to be read. It is rewritten, not appended. Hard cap 40 lines; the
pusher refuses a push above it.

## Steps

1. Gather: `git status -sb`, `git log --oneline -8`, the Open column of `docs/findings/README.md`,
   and what this session knows that is not yet in a file.
2. Rewrite the file with the Write tool, keeping exactly these sections:

   ```
   # Handoff

   Coordinator state. Subagents follow their brief, not this file.
   Updated: YYYY-MM-DD, at commit <short sha>, by <main session | agent name>

   ## Now
   1-3 bullets: what is in flight, which agent had it, exact state (files, failing test, blocker).

   ## Next
   Ordered, 3-7 items, each actionable in a single brief. Name the agent when it is obvious.

   ## Waiting on user
   Questions needing a human decision (front end, wording, scope), or "none".

   ## Open findings
   Paths of findings with status open, or "none".
   ```

3. Anything that does not fit: later work to `docs/backlog.md` (Ideas or Deferred with a trigger);
   evidence to a finding (`finding` skill); durable facts to `docs/reference/`. The handoff links,
   it never explains.
4. Check `wc -l docs/HANDOFF.md` is 40 or less.
5. Commit it. Below 50% usage brief the runner (`commit` skill). At HOLD or STOP the main session
   commits itself: `git add docs/HANDOFF.md` plus any WIP files, with a message starting `WIP:` if
   work is unfinished, ending with the Co-Authored-By line for the model doing the work.
6. Report in one paragraph what moved to backlog or findings.

## Rules

- Never write procedure, rationale, or history here. Git log has the history; ADRs have the why.
- If the file is missing (a decanted project may have deleted it), recreate it from the template
  rather than skipping the step.
