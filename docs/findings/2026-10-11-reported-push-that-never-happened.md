---
date: 2026-10-11
status: resolved
area: process / verification
promoted_to: docs/reference/claude-code-on-windows.md
---

# A chained commit-and-push command reported a commit and a push that never existed

## Question

The Dystopo coordinator session, co-authoring this repository, ran one Bash tool call:
`cd <hatchery> && git add -A docs && git commit -q -F - <<EOF ... EOF && git log --oneline -1 &&
git push -q origin main && echo pushed`. The tool result showed a commit line with a short SHA and
the word `pushed`. The coordinator reported the push to the owner. Minutes later, when the owner
asked what had changed on the remote, the clone had no such commit (reflog: only the clone), the
files were uncommitted and untracked in the tree, GitHub showed no push, and the SHA did not exist
as an object. The owner's reaction: this is exactly what the process is meant to catch.

## What was tried

- `git reflog`: one entry, the clone at 16:47:41 local time, the directory's creation time.
- `git cat-file -t <sha>`: not a valid object. `git fetch origin <sha>`: no such ref on GitHub.
- `gh api repos/.../events`: push events end at the commit before the session started.
- `git status --short` (the first check had been cut to its first line by `head -1`, which hid the
  modified and untracked files and made the loss look total): every change present, uncommitted.
- A second commit, then `git pull --rebase`, `git push`, `git fetch` and a comparison of
  `git rev-parse HEAD` with `git rev-parse origin/main`: equal; the push is real.

## Result

Cause established an hour later by the Dystopo CI list: commit `44b2d0f` with the Hatchery message exists in the Dystopo repository, on `main`, pushed at 16:52, and contains the three Dystopo files that had just been edited (`docs/HANDOFF.md`, `docs/PLANNING.md`, `docs/TASKS.md`). The `cd` into the Hatchery clone at the start of the Bash call did not take effect: the shell stayed in the Dystopo checkout (the harness had been resetting the working directory between calls all day), so `git add -A docs`, `git commit` and `git push` all ran against Dystopo. Every word of the tool output was true; it was true about the other repository. The coordinator relayed it without an independent read of the remote it meant.

Rules adopted in both repositories: (1) a push, merge, tag or release is reported only after a fetch or an API read shows the remote at the expected SHA, with both SHAs quoted (the pusher agent's last check; Dystopo's `tools/push-verified.sh`); (2) a command that must run in a particular repository names it with `git -C <path>` instead of relying on `cd`, and a commit or push command prints `git rev-parse --show-toplevel` first so the report shows where it ran; (3) never truncate a status check (`git status -sb | head -1` hid the real state here).
