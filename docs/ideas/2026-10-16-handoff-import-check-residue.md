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

Plan: `architect`, an amendment of ADR 0014 (`adr` skill, "Amend without superseding"). The code has settled all three: `.claude/hooks/imports.js` serves both the post-edit hook and pusher check 7 (`policy.js --claude-check`), treats `~/`, absolute and climbing targets as outside the root (an info line, not a warning), and has one fence rule (three or more backticks or tildes, any indent); pusher check 7 carries the `mktemp` form of lines 2 and 3. The amendment rewrites items 4 and 6 of `docs/hatchery/decisions/0014-handoff-layout.md` to say what the code does, from the `imports.js` header and `.claude/agents/pusher.md` check 7, and deletes this file. Verify: `node .claude/hooks/test-hooks.js`; the ADR stays under 300 lines. Out of scope: any code change.
