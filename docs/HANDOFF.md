# Handoff

Coordinator state. Subagents follow their brief, not this file.
Updated: 2026-10-11, at commit e9e7abf (plus uncommitted work), by architect

## Now

- Implementing ADR 0002 (repository layout). The architect has written the ADR, the hook spec, the
  `docs/` tree, `.claude/rules/`, nine skills, the agent and settings updates, and the `CLAUDE.md`
  and `README.md` edits. All of it is uncommitted in the working tree.
- The two Node hooks do not exist yet; `.claude/settings.json` already points at them (every
  Bash and Write/Edit call shows a non-blocking hook error until they land). `"onFailure": "block"`
  on git-guard is deliberately absent until the script exists; the coder adds it as its last step.

## Next

1. Coordinator reviews the architect's output, then the runner commits it in steps: ADR and
   proposal; docs tree and rules; skills; agents and settings; CLAUDE.md, README, finding.
2. Coder writes `.claude/hooks/git-guard.js` and `.claude/hooks/post-edit.js` and edits the
   usage-gate reason text, to `docs/design/decisions/0002-hook-specs.md`; runs the test tables.
3. Runner confirms the pusher doc-size check passes (`pusher.md` check 7) on the current tree.
4. Pusher pushes `main` on the user's request.
5. Optional: test once whether `SessionStart` fires inside subagents (critic claim 5); only matters
   if the deferred SessionStart hook is revived.

## Waiting on user

none

## Open findings

- `docs/findings/2026-10-11-bash-heredoc-apostrophes.md`
