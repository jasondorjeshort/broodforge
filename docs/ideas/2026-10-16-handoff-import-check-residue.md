---
date: 2026-10-16
kind: chore
related: [docs/design/decisions/0014-handoff-layout.md]
---
# Handoff import check: three residues of ADR 0014's own text

ADR 0014 item 4 specifies the hook and pusher checks that resolve `@` imports. Three things in that
text, now built as written in `.claude/hooks/post-edit.js` and `.claude/agents/pusher.md`, are left
for the architect to settle.

- Every `@` target is joined onto the importing file's directory. The documented `@~/...` and
  absolute forms therefore warn in the hook and fail in pusher line 3 although Claude Code accepts
  them. Nothing here uses either form today.
- The hook treats any run of three or more backticks as a fence token, so a four-backtick block
  holding an inner three-backtick line closes early. Pusher line 3 skips no fences at all. Hook and
  pusher disagree on fenced `@` lines.
- ADR 0014 item 6's verbatim line 2 (the 120 total) is denied by `git-guard`: an unexpanded
  argument in a line that runs git. `.claude/agents/pusher.md` now carries lines 2 and 3 in a
  rewritten form (the `git ls-files` output written to a `mktemp` file, then looped), which the
  guard allows and which also handles a filename with a space; the ADR's own lines 2 and 3 split
  on spaces. The amendment should take the pusher's text.

## Open

- Does the ADR keep the join-everything rule and say so, or resolve `~` and absolute targets?
- One fence rule for hook and pusher: which, and does the pusher gain one?
