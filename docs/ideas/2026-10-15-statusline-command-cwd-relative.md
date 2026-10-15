---
date: 2026-10-15
kind: bug
---

# The status line command is still relative to the working directory

Anchor: `.claude/settings.json`, `statusLine.command` (`node .claude/statusline.js`).

The hook commands in the same file run as `node "${CLAUDE_PROJECT_DIR:-.}/.claude/hooks/x.js"`, so
they work when the shell's working directory is not the project root. The status line command
did not get the same change. If Claude Code runs it from another directory, it fails to find the
script, the line goes blank and the usage cache stops updating, which the usage gate reads. The
reviewer of the hook-path change noticed this; nobody has reproduced it.

## Open

- Does Claude Code run the status line command from the project root, or from the shell's
  working directory? Is `CLAUDE_PROJECT_DIR` set in its environment?
- The same review read in the Claude Code binary (untested) that Claude Code substitutes a plain
  `${CLAUDE_PROJECT_DIR}` itself, which would work under both Git Bash and PowerShell. The
  `${CLAUDE_PROJECT_DIR:-.}` form relies on the shell to expand it, and expands to nothing under
  a hook set to `"shell": "powershell"`. Whether the hooks should switch forms belongs to the
  same check.
