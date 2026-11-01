---
date: 2026-10-31
kind: feature
related: [2026-10-29-broodforge-works-on-itself-overnight.md]  # that one has a discard button on an unattended session's result; this is the driver's own discard of a live session
---

# End a session without handing off

Ending a session always runs the handoff (`docs/design/04-ux.md`, Ending a session; `internal/git/handoff.go`
`Run`): bundle, checks, then the project's `handoff` (`push` or `pull-request`). A driver who does not
want the work has no way to say so: the branch lands anyway and they delete it by hand in the
checkout, or they leave the session open for good (this channel's idle timeout is "never"), tree kept.
`worktree forget|adopt` are a later pass (`05-debug-interface-git.md`, Surface), so the host has no
discard either. The owner, 2026-10-31, on the first `dev-live` sessions: "what if I don't like the
changes I made and just leave it dangling or close it in frustration?"

What it would take: a second end, `session end <id> --discard` on the CLI and a venue action, that
stops the turn, writes the summary with `discarded` in place of the push line, removes the tree and
`refs/sessions/<id>` without a push, and audits `session-discarded` with the commit count. The commits
are gone with the tree, so it confirms once in the venue. Gate: the driver (ending is already theirs).

The owner, 2026-10-31, in session s0008 (its edit was lost at the end, see
`2026-10-31-ending-a-session-with-uncommitted-changes-loses-them.md`; refiled here): "another button
'abort session' that discards the commits (well they can still stay recoverable on a tree
somewhere)". So: the tree is removed but the commits stay reachable on an unpushed ref such as
`refs/aborted/<id>`, a later `session restore <id>` brings them back, the confirm guards a reversible
step, and the audit line names the ref.

## Open

- Whether a discarded session's transcript still goes to the lobby (it did happen; the work did not).
- How long an aborted session's ref is kept, and who prunes it.
- Whether an interrupted session (ADR 0026) can be discarded the same way.

Question: the venue shape and words are the owner's. A second button beside `[End session]` (`[End and
discard]`, with one confirm), or a choice inside the end flow (`End session` → `Hand off` / `Discard`)?
And the summary line's wording for a discard.
Answer: 2026-10-31, two buttons: `[End session]` and `[End and discard]` side by side, the discard
behind one confirm, "to throw away a session without saving changes" (the owner's words). The
commits stay recoverable on `refs/aborted/<id>`; the dirty tree is WIP-committed first (the related
idea's answer), so nothing is unrecoverable. `session restore <id>` is a later pass ("recovering
sessions may also be a thing one day"), so the first pass writes the ref and the audit line and the
summary names the ref, not the command. Lines, the owner's to edit on sight: confirm `Discard session
<id>? Its <n> commits stay recoverable.`; summary `<n> commits Â· discarded; recoverable on request`.
Settled by the coordinator: the transcript still goes to the lobby; the ref is kept until a later
`session prune`; an interrupted session can be discarded the same way. Now a Plan: coder, one pass
(`internal/git/handoff.go`, `internal/discord/customid.go` and the header, `04-ux.md` Ending a session,
`05-debug-interface-git.md` `session end --discard`, the golden for the end).
