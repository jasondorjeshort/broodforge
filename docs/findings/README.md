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
| `2026-10-11-windows-worktrees-long-paths.md` | superseded | tooling / git worktrees on Windows | `git worktree remove` fails on long paths; an empty robocopy mirror clears the tree; superseded by the robocopy finding of 2026-10-14 |
| `2026-10-11-reported-push-that-never-happened.md` | resolved | process / verification | A `cd` to another clone inside a Bash call did not take effect; the commit and push landed in the other repository and were relayed as done. Rules: prove a push by fetch; `git -C`, not `cd`. |
| `2026-10-11-shared-scratchpad-and-argument-splitting.md` | resolved | tooling / parallel agents on Windows | Parallel agents share one scratchpad; PowerShell 5.1 splits dotted arguments. |
| `2026-10-12-windows-long-path-delete.md` | resolved | tooling / deleting worktrees on Windows | `Remove-Item` on Gradle build trees needs the `\\?\` prefix; plain paths fail past 260 characters. |
| `2026-10-13-usage-gate-deny-line-trap.md` | resolved | process / usage gate | The deny line was a ceiling the owner never asked for and nothing could lift it mid-session; deny is now optional, the base ships none, and the critic brief asks who lifts every refusing mechanism. |
| `2026-10-16-account-switcher-practices.md` | resolved | process / usage gate under an account switcher | How people run Claude Code across accounts: switch at session boundaries, let the switcher hold the cross-account usage, pin projects to accounts; nothing upstream to adopt as a hook. Facts promoted to `reference/claude-swap.md`; decision in proposal 0011. |
| `2026-10-14-robocopy-mirror-purges-through-junctions.md` | resolved | tooling / deleting worktrees on Windows | An empty robocopy mirror purges a junction's target with `/XJ`, `/XJD` or no flag; `\\?\` `Remove-Item` and `cmd /c rmdir` leave it intact; a plain-path listing misses a long-path link. Supersedes the 2026-10-11 worktrees finding. |
