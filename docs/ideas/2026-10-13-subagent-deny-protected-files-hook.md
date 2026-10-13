---
date: 2026-10-13
kind: feature
---

# Deny subagent writes to CLAUDE.md and settings through a second PreToolUse hook

A rule that denies `Write` and `Edit` on `CLAUDE.md` and `.claude/settings.json` from subagents
would fold into a second PreToolUse hook, using `agent_type` from the hook input. The new hook would
sit beside the existing ones in `.claude/hooks/`.
