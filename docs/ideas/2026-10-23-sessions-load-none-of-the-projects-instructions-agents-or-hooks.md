---
date: 2026-10-23
kind: bug
related: [2026-10-23-engine-auto-permission-mode-as-a-third-session-mode.md, 2026-10-23-session-land-pushes-to-main-mid-session.md]
---

# Sessions load none of the project's instructions, agents or hooks

The first real session on this project (s0005, 2026-10-23) edited code, docs and the handoff,
committed, and was pushed to `main` with no subagent, no review and no cap check. The cause is
not the session's behaviour: it never saw the project's rules.

Observed (a canary project run by `claude -p` on engine 2.1.309, the owner's PC): with the flags
Broodforge passes (`--setting-sources user`, `internal/engine/claudecode.go`, `Args`; ADR 0020,
Settings sources) the project's `CLAUDE.md` is not loaded and the project's `.claude/agents/`
definitions are not launchable; the same turn without the flag loads both. Path-scoped rules
(`.claude/rules/`) were not shown loaded in either turn (the prompt touched no matching file), so
that half is unverified. Project `settings.json` entries are also ignored in a workspace not
trusted interactively, which the bot cannot accept.

So in a Broodforge session the agreement files do not exist: delegation (`delegation.md`), one
review (`one-review.md`), who commits and pushes (`git.md`), model routing, the size caps and the
post-edit, git-guard and usage-gate hooks. The only project text the session gets is
`SystemPromptAppend` (`internal/session/prompt.go`: speaker, reach, one call at a time, trailers).
ADR 0020 meant to keep repository hooks and `.mcp.json` out (a project's hooks must not run as the
bot account); `CLAUDE.md` and agents going with them looks like a side effect.

Shape to consider: do not re-enable the project settings source. Broodforge injects, from the
canonical checkout's default branch (so the text is trusted, not the session's own tree): the
project's instructions as an appended system prompt, the agent definitions through the engine's
`--agents` option, and the repo's enforcing hooks through the `--settings <file>` the bot already
controls (ADR 0020, C26: honoured beside `--setting-sources user`). Check each flag against the
engine first (`docs/reference/claude-code-flags.md`).

## The design question: two workflows, and a session needs both

This is a major design question (`major-decisions.md`; a proposal, a critic, an ADR), not a patch.
A Discord session works under two sets of instructions that have never been separated:

- Broodforge's workflow for Discord: who is speaking, the reach, one call at a time, the trailers,
  how to answer a prompt, how a venue is steered, what ends a session and what it hands off. This
  belongs to the bot and applies to every project. Today it is the short `SystemPromptAppend`.
- The repository's workflow: its `CLAUDE.md`, agreement, agents, skills, rules and hooks (here:
  delegation, one review, caps, who commits and pushes). This belongs to the project and differs
  per project; a repository with none is normal.

Neither alone is enough, and they can disagree: the repository's rule says only the runner
commits and only the pusher pushes, while Broodforge's session ends by committing and handing the
branch off itself. The open design is how the two compose: which wins on a conflict, where each is
stated (the bot's part must not be editable by the session or by the repo), and whether the
repository can say "in a Discord session, X" for itself. s0005 shows the cost of having only the
bot's part: it worked, and bypassed everything the repository would have enforced.

## Open

- Which project files are injected, and who decides: a per-project list in configuration, or
  always `CLAUDE.md` plus `.claude/agents/`? A third-party project's instructions are its own.
- Who is the coordinator in a Discord session: one agent that delegates to injected subagents
  (the design already shows `Task` lines indented in the trace), or the driver? The owner's call.
- Whether the hooks that enforce roles (commit and push roles, caps, privacy at push) can run
  inside a session at all, and what the session-side `git` is allowed to do (a departure row in
  the agreement files is Hatchery's to carry back).
- Until then: nothing from a session reaches `main` without a reviewer read and the pusher's
  checks; `docs/HANDOFF.md` records it.
