# UX: ending a session

Date: 2026-11-01 (moved by topic from `04-ux.md` when the end gained the WIP commit and the discard;
the first section's text is the one that stood there). Status: the moved text as 04 had it; the
WIP commit, the discard and the move to `Ended sessions` are the owner's answers of 2026-10-31, their lines the owner's to edit on
sight. A sibling of `04-ux.md`, which points here from Ending a session; its conventions apply
(message text in fenced blocks exactly as posted, `<...>` a value filled in). The git side is
`03-architecture-git.md` (End of session); the CLI side `05-debug-interface-git.md`.

## Ending a session

`[End session]` on the header, `/session end`, the idle timeout if it has one (R-15), or the owner on the host. A
running turn is stopped first; the sequence is 03's (bundle, check, push or pull request) and the venue shows `Ending:
checking and handing off 3 commits...` meanwhile. Owner 10 (vision Q 2): the default is `pull-request`, so a reviewer
sees it before it lands; `push` is a per-project setting (`handoff`, 03), expected for some projects.

The summary goes to the venue and to the project channel with the transcript attached, so the
lobby shows what happened: `[#broodforge] Session s0001 ended · 3 commits · PR #42 · transcript s0001.md`; a project with `venue.summaryChannel` (03) also gets both there.

```
Session s0001 ended by Jason · 1h 12m · 5 turns · 212k in / 9.4k out · Opus
Branch broodforge/pc/s0001 -> pull request #42 <url>
Commits: a1b2c3d Add config reload · 9f8e7d6 Test last-good fallback · 5c4b3a2 Docs
Author: Jason Dorje Short. Co-authors: Sam Example, Claude Opus 5.5.
Unmerged subagent branches removed: none.
```

No commits: `Session s0001 ended by Jason · 0 commits · nothing to push; the tree was removed.`
The transcript is a Markdown file: every message with writer and time, every trace line, every permission decision and
its decider, every footer. Owner 11: it is attached to the summary in the project channel (Ticket Tool's habit) and,
when set, to the summary channel. A private channel is deleted ten minutes after the summary. Owner 12: the pull request
title and body are written by the agent in one closing turn asked to summarize its changes (readable; costs a small
turn); the summary shows the title. When the closing turn did not run (a `push` project, an auth error, or a credential
that needs login) the title is `Session <id>: <first message, truncated>` and the body is the summary.

After the summaries the venue moves to the category `Ended sessions` (`venue.endedCategory`, 03; setup's channels
step makes it beside Sessions), so the driver still reads it away from the live ones (the owner's answer, 2026-10-31).
The bot then attempts the lock the owner asked for, Send Messages denied on every overwrite but its own: Discord asks
Manage Roles for an overwrite edit, which the bot does not hold (ADR 0027, removed at the end of setup), so the lock is
refused with a `warning: discord: lock <id>: ...` line and the channel stays writable. A stray message there still gets
`This session has ended.` by direct message. Whether to drop the lock or give the bot Manage Roles back (a reversal of
ADR 0027) is the owner's pending call. The ten minutes are unchanged; removing ended channels only when the guild's cap
needs the room is a later idea. A project set up before the category existed has the key empty: its venue stays where it
is and the bot's stderr gets one `warning: discord: moving the venue of <id>: ...` line naming `setup channels`.

A session interrupted by a restart ends the same way, from its header's `End session`, `/session end` or
the host (ADR 0026): the summary's counts are the session's whole, the attached transcript holds only
what came after the restart and opens with the interrupted line. Nothing else works there: a message
or any other action gets the `interrupted refusal` row below (the owner's wording, 2026-10-24), in place
of `This session has ended.`, which is for a session that is ending or ended. Two summaries only such a
session can show (owner-approved 2026-10-24): when its tree is gone but its commits are still
on the host, `Branch broodforge/pc/s0001 -> not handed off: its tree is missing on the host; the owner can
recover refs/sessions/s0001 from the canonical.`; when the owner had already run `worktree handoff` and it
pushed nothing, `Branch broodforge/pc/s0001 -> handed off earlier on the host; nothing more to push.`

## Uncommitted changes at the end

An end never loses the tree's work (00-vision: "lose nothing"). Before the bundle, a tree with
uncommitted changes (a changed tracked file, staged or not, or an untracked file git does not
ignore) is committed as one commit on the session branch, `WIP: uncommitted changes committed at
end`, its author and committer the driver and its trailers the ones the check requires of a commit
made then (`03-architecture-git.md`, Attribution), so it passes the check like the session's own
commits. The commit is made through the spawn backend, as the bundle is; the bot's git never opens
the tree (ADR 0022). It is handed off like any other commit by the project's `handoff` (`push` or
`pull-request`; ADR 0028's `main` landing is not built yet, P34), and the push scan runs on it. The
Ending line counts it. The summary's branch line becomes the owner's line (2026-10-31):

```
1 commit · pushed to broodforge/pc/s0001 (WIP: uncommitted changes committed at end)
```

with the session's whole count (`3 commits · ...` when two came before it); on a `pull-request`
project the request's line keeps its form and ends with the same parenthesis. A refused handoff
keeps the WIP commit in the tree, so the next end hands it off without a second one. On an interrupted session a refused WIP commit's text names `broodforge session end <id>` as the owner's retry, never `worktree handoff`, which makes no WIP commit and would remove the tree with its uncommitted files (`04-ux.md`, Failure flows). No refusal
on a dirty tree: the owner's answer is that a refusal makes the human the safety mechanism.

## Discarding

`[End and discard]` sits beside `[End session]` on the header, open to the same `answer` list. It
asks once in the venue (the owner's line, 2026-10-31; `1 commit stays` in the singular):

```
Discard session s0001? Its 3 commits stay recoverable.  [Discard]
```

`[Discard]`, or `session end <id> --discard` on the host, ends the session without a handoff: a
running turn stopped, a dirty tree WIP-committed first (above) so nothing is unrecoverable, the
commits fetched into the canonical as every end does and kept there on `refs/aborted/<id>`, the tree
and `refs/sessions/<id>` removed, nothing pushed. The summary's branch line becomes the owner's
line, then the ref (`not 04`, the owner's to word):

```
3 commits · discarded; recoverable on request
Kept as refs/aborted/s0001 in the project's canonical.
```

A discard of a session with no commits and a clean tree reads as the plain end's `0 commits ·
nothing to push; the tree was removed.` The transcript still goes to the lobby with the summary (it
happened; the work did not land). The ref is kept until a later `session prune`; a later `session
restore <id>` brings it back, so the summary names the ref, not a command. A session interrupted by a
restart (ADR 0026) is discarded the same way. A standing confirm loses its button when the session
ends. Audit: `session-discarded` with the commit count, the ref and the WIP commit, beside the end's
`session-ended` with outcome `discarded`.
