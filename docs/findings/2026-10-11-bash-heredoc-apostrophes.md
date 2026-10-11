---
date: 2026-10-11
status: open
area: tooling / Claude Code Bash tool on Windows
---

# Bash heredocs containing an apostrophe fail on this Windows setup

## Question

Why do Bash tool calls that write files with a heredoc fail intermittently, and what is the
reliable way to write multi-line text from an agent?

## What was tried

Both the coordinator and the critic hit the same failure on 2026-10-11 while writing Markdown
through the Bash tool (Git Bash under Claude Code 2.1.296, Windows 10 Pro 19045):

```
cat > file.md <<'EOF'
... the critic's objections ...
EOF
```

Any heredoc body containing an apostrophe that is not inside double quotes (for example
"critic's" or "doesn't") was rejected before execution with:

```
unexpected EOF while looking for matching `'`
```

The quoted delimiter (`<<'EOF'`) did not help; the failure is in how the command text is handled
before bash sees it, not in bash's own heredoc parsing. Text without apostrophes went through.
Single-quoted arguments on a single line (`printf '%s' 'x'`) work; the problem is specific to
apostrophes inside a heredoc body.

## Result

Unresolved root cause (tool-side pre-processing is suspected, not confirmed). Working rules:

- Use the Write or Edit tool for any file whose content contains an apostrophe or backtick.
- If Bash is required, split the write into appends of apostrophe-free chunks, or write the
  apostrophe as `'"'"'` inside a double-quoted segment, which is unreadable and not recommended.
- Skills and agent briefs that tell an agent to create a file should say "use the Write tool".

## Next

- Reproduce in a minimal form on Linux to confirm it is Windows-specific.
- Check the Claude Code changelog for a fix; if fixed, set `status: resolved` and promote the
  surviving rule ("Write tool for prose") to `docs/reference/claude-code-tools.md`.
