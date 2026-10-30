---
date: 2026-10-29
kind: feature
related: [2026-10-24-a-turn-ending-on-a-tool-call-could-prompt-the-engine-for-a-closing-message.md]  # that one writes for the venue; this one writes for the next resume
---

# A session remembers across weeks

A moonshot the owner judged far off at filing (2026-10-29). The engine writes a per-session handoff
file into the worktree at the end of every turn (what was asked, what landed, what is open), and a
session resumed after days or weeks starts from that file rather than from a cold prompt and the
engine's own resume. This is the "check in rarely, lose nothing" principle of `00-vision.md`
applied to the agent's memory, not only to the branch: today the branch survives any gap, the
agent's grasp of it does not survive a lost or expired engine session.

Anchor: the system prompt the session appends in `internal/session/prompt.go`
(`SystemPromptAppend`), which would ask for the file and feed it back; the file would live in the
worktree beside the code and travel with the branch (`internal/git/handoff.go` decides what the
handoff carries).

## Open

- Whether the file is committed on the session branch (it reaches the pull request) or kept
  untracked in the worktree (it dies with the worktree).
- One engine prompt per turn to write it costs tokens on every turn; the related closing-message idea
  has the same cost and the two could share the one prompt.
- Resume already exists in the engine; measure first how often it fails after a long gap.

Waits: the first usable version (`00-vision.md`, "The first usable version") has shipped; the owner, 2026-10-29.
