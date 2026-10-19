---
date: 2026-10-13
kind: feature
---

# Deny subagent writes to CLAUDE.md and settings through a second PreToolUse hook

A rule that denies `Write` and `Edit` on `CLAUDE.md` and `.claude/settings.json` from subagents
would fold into a second PreToolUse hook, using `agent_type` from the hook input. The new hook would
sit beside the existing ones in `.claude/hooks/`.

Question: do you still want subagents denied writes to `CLAUDE.md` and `.claude/settings.json`? `CLAUDE.md` is now a 40-line import map with the sections in `.claude/agreement/`, and model routing sends settled edits to hooks and docs to the `tester`, so a broad deny would block work the agreement assigns. I recommend not building it, or narrowing it to `.claude/settings.json` (permissions and hook wiring), which no brief should need to touch.
Answer:
