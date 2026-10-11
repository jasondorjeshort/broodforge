# Handoff

Coordinator state. Subagents follow their brief, not this file.
Updated: 2026-10-11, at commit 87ab05b (plus uncommitted hooks), by main session

## Now

- ADR 0002 is implemented. Hooks `git-guard.js` and `post-edit.js` exist, `test-hooks.js` passes
  68 of 68, `onFailure: block` is on git-guard. Uncommitted: the three hook files, `usage-gate.js`
  reason text, `.claude/settings.json`, and the coder's note in `0002-hook-specs.md`.
- No application project yet. Hatchery is the project; the user has not described an app.

## Next

1. Runner commits the hook batch (one commit, trailer Opus 5.5), then pusher pushes `main`.
2. Architect folds the coder's nine spec deviations into `docs/design/decisions/0002-hook-specs.md`;
   the one with a user-visible effect is 7: `post-edit` warns on every `docs/design/proposals/*.md`
   because no index lists them. Fix by listing them in `docs/design/proposals/README.md`.
3. Start a fresh session in this directory to confirm the status line shows the 5h and 7d meters
   and that `@docs/HANDOFF.md` is imported. First real test of both.
4. User decides the first decanted project; then `decant` and `00-vision.md` via the architect.

## Waiting on user

- Nothing blocking. The first app to decant from Hatchery is the user's call, when ready.

## Open findings

- `docs/findings/2026-10-11-bash-heredoc-apostrophes.md`
