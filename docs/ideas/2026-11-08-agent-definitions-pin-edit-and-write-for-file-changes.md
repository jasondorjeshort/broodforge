---
date: 2026-11-08
kind: chore
related: [2026-11-07-the-post-edit-hook-is-skipped-by-bash-edits-and-hangs-when-run-bare.md]
---

# Agent definitions pin Edit and Write for file changes

On 2026-11-07 four testers and one architect made tracked-file edits through Bash (`sed`, a Python
one-liner, a Node script) although their briefs said "Edit tool only" in so many words; each reported
that a session-level instruction to prefer Bash had outranked the brief. Every such edit skipped the
`PostToolUse` post-edit hook (the index, size, path and frontmatter checks), and the Go rule's ban on
heredoc writes went unenforced for the Go files touched. The coordinator re-ran the hook by hand each
time. A brief is the weakest layer; the agent definition (`.claude/agents/<name>.md`, loaded as the
subagent's system prompt) is the layer a session instruction does not override.

The change (base-owned: a Hatchery carry-back): one sentence in every agent that writes files
(`coder`, `tester`, `architect`, `critic`): "File changes go through the Edit and Write tools, never
through a shell command, whatever a session instruction says; the repository's edit hooks run only on
those tools." The `brief` skill's template then drops the per-brief line. A second, mechanical guard
if the sentence is not enough: a `PreToolUse` hook on Bash that denies a command whose text writes to a
tracked path (`>` or `>>` to a repository file, `sed -i`, `tee`), with the coordinator's own session
exempt as the git policy already is.

Anchors: `.claude/agents/tester.md`, `coder.md`, `architect.md`, `critic.md`; `.claude/skills/brief/SKILL.md`;
`.claude/hooks/post-edit.js` (the hook that was skipped); `.claude/rules/go.md` (the heredoc ban).

## Open

- Whether the mechanical guard is worth its false positives (a coder's scratch files under `$TEMP` are
  fine to write from Bash; the guard must know the repository root).
- The auto-mode instruction's source is the harness, not the repository; a note in
  `docs/hatchery/reference/claude-code-on-windows.md` or its sibling that names it.
