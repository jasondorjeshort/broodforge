# Findings

Results of investigations: a root cause, a benchmark, an approach that did not work and why.
Findings are evidence, not truth: read the `status` before trusting one. Files are named
`YYYY-MM-DD-<slug>.md`, carry frontmatter (`date`, `status: open | resolved | superseded`, `area`,
optional `supersedes`, `promoted_to`), and have the sections Question, What was tried, Result, Next.
Written with the `finding` skill. When a finding is resolved, its durable part is promoted to
`reference/` or `design/` and `promoted_to` records where.

Status is read from the file, nowhere else, by its first word. The open ones:
`node .claude/hooks/queue-list.js findings --status open`; `HANDOFF.md` links them. A file that
lists as `?` names the field to fix on stderr.

Listed on demand: node .claude/hooks/queue-list.js findings
