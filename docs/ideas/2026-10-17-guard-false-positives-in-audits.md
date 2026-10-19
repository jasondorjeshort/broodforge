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

2026-10-19: `git replace -l`/`--list` is allowed now (the rebase --exec change). The rule-19
fix was built and withdrawn: the false positive fires only when a runner (`| xargs`, `bash -n`,
`Invoke-Command`) is in the same line, and an "inert text" exemption keyed on pipelines let real
bypasses through (`for ...; do echo '<git cmd>'; done | sh`, `tee >(sh)`, `| & iex`, write then
`sh x.sh` on one line). Those shapes are now denied cases in the harness.

Question: leave the rule-19 false positive as it is (agents split the data and the runner into two
calls), or fund a design pass on data-flow-aware parsing? Any exemption has to beat the denied cases
above, which takes tracking of compound commands, process substitution and files written then run.
I recommend leaving it: the workaround costs one extra call, and a wrong exemption lets a destructive
command through. `update-ref -d` in a scratch repository stays denied either way.
Answer:
