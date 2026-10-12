# Findings

Results of investigations: a root cause, a benchmark, an approach that did not work and why.
Findings are evidence, not truth: read the `status` before trusting one. Files are named
`YYYY-MM-DD-<slug>.md`, carry frontmatter (`date`, `status: open | resolved | superseded`, `area`,
optional `supersedes`, `promoted_to`), and have the sections Question, What was tried, Result, Next.
Written with the `finding` skill. When a finding is resolved, its durable part is promoted to
`reference/` or `design/` and `promoted_to` records where.

Open items are marked in the Status column; `HANDOFF.md` links the open ones.

| File | Status | Area | One line |
|---|---|---|---|
| `2026-10-11-bash-heredoc-apostrophes.md` | open | tooling / Claude Code on Windows | Bash tool rejects heredocs containing an apostrophe outside double quotes. |
| `2026-10-11-background-tool-call-kills-processes.md` | resolved | tooling / Claude Code on Windows | A process started inside a tool call dies with the call; detach with a recorded PID. |
| `2026-10-11-windows-worktrees-long-paths.md` | resolved | tooling / git worktrees on Windows | `git worktree remove` fails on long paths; an empty robocopy mirror clears the tree. |
| `2026-10-11-reported-push-that-never-happened.md` | resolved | process / verification | A `cd` to another clone inside a Bash call did not take effect; the commit and push landed in the other repository and were relayed as done. Rules: prove a push by fetch; `git -C`, not `cd`. |
| `2026-10-11-shared-scratchpad-and-argument-splitting.md` | resolved | tooling / parallel agents on Windows | Parallel agents share one scratchpad; PowerShell 5.1 splits dotted arguments. |
