---
paths:
  - ".claude/hooks/**"
---

# Conventions for .claude/hooks/

- Edit a hook file with the Edit or Write tool, never through Bash (sed, a heredoc, a script). A hook
  that fails to parse denies every Bash call, including the one that would repair it.
- After any edit run `node .claude/hooks/test-hooks.js`; do not leave the tree with a failing hook.
- A new denial or allowance gets a case in `test-hooks.js` in the same change.
