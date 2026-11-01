---
date: 2026-10-31
kind: feature
related: [2026-10-29-a-session-remembers-across-weeks.md]  # that one keeps the agent's grasp across a gap; this one moves the session back to an earlier turn
---

# Rewind and fork a session at a turn

Recorded from the owner's conversation of 2026-10-31 as a moonshot, not sanity checked: nobody has
read the anchor against the claim yet.

Every turn ends with a ref under the session's namespace in the canonical, beside the
`refs/sessions/<id>` tip. That tip is not written every turn: it is force-fetched from a bundle at
`worktree check`, at the handoff and at a reviewer's launch (`internal/git/create.go`,
`internal/git/handoff.go` `Refresh`), and the bot never opens the session's tree, so a per-turn ref
and a rewind each need a backend step that bundles and resets the worktree. A "rewind to turn 3" button in the venue
resets the worktree to that turn's ref and forks the engine transcript at that turn, since the
engine resumes by session id and can fork a resumed session. A second person can fork the same
point into a new session to race an alternative. For the vibe programmer of `00-vision.md` this
is the undo button they have no other way to reach: they cannot run `git reset`, and today a
wrong turn is only recoverable by asking the agent to undo its own work.

Anchor: `internal/git/create.go` (`SessionTip`, the one ref per session today);
`internal/engine/claudecode.go` (`--resume` is the only resume shape the adapter knows);
the button would be a new custom id in `internal/discord/customid.go`.

## Open

- A ref per turn or a tag of the commit the turn ended on; a turn that ended without a commit has
  only a dirty tree to snapshot.
- Whether a fork is a new session (its own worktree, branch and venue) or the same session moved
  back, and what the audit says about the turns it discards.
- Whether the engine's fork keeps the permission answers given after the fork point.

Question: Does the owner want a spike on engine forking before any design of the per-turn ref and rewind button?
Recommend a spike first (one tester, the `spike` pattern in `docs/findings/`): the installed engine has a
`--fork-session` flag ("when resuming, create a new session ID"), but the adapter passes only `--resume`
(`internal/engine/claudecode.go`), so whether forking works under `-p` and keeps the transcript is unproved.
Build the ref and button only after the finding says it does.

Answer:
