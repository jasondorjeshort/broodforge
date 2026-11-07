---
date: 2026-11-07
kind: chore
---

# The post-edit hook is skipped by Bash edits and hangs when run bare

Two ways the repository's own edit check went missing in one sitting:

- Auto mode tells every agent to make file changes with Bash (`sed`, heredocs, a Node script)
  rather than the Edit or Write tool. `.claude/hooks/post-edit.js` is a `PostToolUse` hook on Edit
  and Write, so an edit made through Bash is never checked: no index row, no size cap, no path
  check, no frontmatter check. A tester did exactly that today and reported the hook silent.
- A brief's verify line of the form `node .claude/hooks/post-edit.js <file>` is wrong: the hook
  reads the engine's JSON from stdin and ignores its arguments. With stdin at EOF (the main
  session's Bash tool) it exits 0 having checked nothing; with stdin open (an agent's shell) it
  waits for ever, and two agents hung on it today. The working form is
  `printf '{"tool_input":{"file_path":"<abs path>"}}' | node .claude/hooks/post-edit.js`.

Anchors: `.claude/hooks/post-edit.js` (stdin at its end), `.claude/skills/brief/SKILL.md` (the
verify line), `.claude/rules/go.md` (already forbids heredoc writes for Go; the docs rule does not
say the same for docs).

## Open

- Whether the hook should also accept a file argument and check it at once (a base change, so a
  carry-back), or the brief skill should carry the stdin form; or both.
- Whether the docs rule should forbid Bash-made edits to tracked docs outright, as the Go rule
  does, so the hook always runs; and whether a `PostToolUse` matcher on Bash can catch a write.
