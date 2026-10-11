---
date: 2026-10-11
status: open
area: process / verification
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

Cause not established. The git output in the tool result was real git output, so the commit was
made against some repository state that this clone never had; a working-directory drift between
tool calls (the shell's cwd reset between calls several times that day) is the likeliest
mechanism, but it was not reproduced. What is established: the coordinator relayed a command's
own success echo as a fact, and only an independent read of the remote exposed it. Rule adopted in
both repositories: a push, merge, tag or release is reported only after a fetch (or an API read)
shows the remote at the expected SHA, and the report quotes both SHAs. The pusher agent now does
this as its last check; Dystopo has `tools/push-verified.sh`.

## Next

Reproduce the drift if it recurs: log `pwd` and `git rev-parse --show-toplevel` at the start of
any commit command run by the main session; if the toplevel is not the intended repository, the
mechanism is found. Then set `status: resolved` and promote the rule to `docs/reference/`.
