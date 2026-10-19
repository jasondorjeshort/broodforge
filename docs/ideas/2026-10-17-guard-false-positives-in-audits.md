---
date: 2026-10-17
kind: bug
related: [../findings/2026-10-17-history-rewrite-of-three-repos.md]
---

# The git-guard denies read-only and scratch-repo commands during history audits

`.claude/hooks/git-guard.js` denied these while agents audited a rewritten history, none of which
changes a repository the work depends on:

- `git replace -l`, which only lists replace refs, read as a history rewrite.
- `git update-ref -d` in a scratch repository made for the audit.
- Command lines whose heredoc or echo text contained `git ... $VAR`, denied as an "unexpanded
  argument" although the text was data, not a command.

Found during the 2026-10-17 history rewrite; the audits worked around each denial.

Suggested: allow `git replace` with `-l`/`--list`; decide whether `update-ref -d` is allowed inside a
repository whose path is under the scratchpad; make the unexpanded-argument rule skip quoted and
heredoc text, as the other rules do (`docs/reference/claude-code-on-windows.md`, Shells).

Plan: `coder`, in `.claude/hooks/git-guard.js` and `docs/hatchery/decisions/0002-hook-specs.md`. (1) The rewrite rule (line 526: `filter-branch`, `filter-repo`, `replace`, `update-ref`) allows `git replace` with only `-l` or `--list` and an optional pattern. (2) The unexpanded-argument rule (n 19, line 589) skips heredoc bodies and quoted data as the other rules do (the "quoted text is data" note). Harness cases for each, allowed and still denied (`git replace <a> <b>`, `git push origin $B`). Verify: `node .claude/hooks/test-hooks.js`. Out of scope: `update-ref -d` in a scratch repository stays denied; an exemption there needs the guard to know the scratchpad, a separate decision.
