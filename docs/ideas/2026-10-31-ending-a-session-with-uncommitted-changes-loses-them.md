---
date: 2026-10-31
kind: bug
related: [2026-10-31-end-a-session-without-handing-off.md]  # that one is a deliberate discard; this is an accidental loss at an ordinary end
---

# Ending a session with uncommitted changes loses them

`End session` on a session whose tree has no commits past the base prints `0 commits · nothing to
push; the tree was removed` (`04-ux.md`, Ending a session; `internal/git/handoff.go`, outcome `none`)
and removes the tree, whatever the tree holds. On 2026-10-31 sessions s0008 and s0009 each wrote
an idea file and its index row, told the driver "Nothing is committed yet. Say if you want it
committed.", and were ended; both trees were gone and the work with them. Nothing warned. The
driver expected each to have landed one commit ("Each of those should have had one commit").
00-vision's line is "check in rarely; lose nothing".

Reproduce: start a session, have it edit a file without committing, `End session`; `session show`
reads `worktree removed`, the summary says 0 commits, the edit is nowhere.

What it would take, one of: the end refuses on a dirty tree with a line naming the files and the
two ways out (commit, or the discard of the related idea); the end commits the tree as one WIP
commit with the session's trailers and hands that off; the tree is kept and `status` lists it as
`kept with uncommitted changes` for the owner. The first is the smallest and keeps the driver in
charge; the second spends nothing and loses nothing but lands work nobody reviewed as a commit;
the third moves the loss to the host.

Question: which of the three the driver sees, and the line's wording. The refusal (`Could not end:
the tree has uncommitted changes to <n> files; ask the session to commit, or ...`) is the
coordinator's suggestion.
Answer:
