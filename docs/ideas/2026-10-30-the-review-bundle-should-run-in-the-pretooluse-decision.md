---
date: 2026-10-30
kind: feature
---

# The review bundle should run in the PreToolUse decision, not on the tool-use event

P31 bundles the session tree into `refs/sessions/<id>` when the bot reads the reviewer call's
`tool-use` block (`internal/session/turn.go`, `refresh`). The engine does not wait for that read:
the hook is answered on another goroutine and the stream read lags by seconds. A commit that lands
after the reviewer has read the tree but before `bundle create` resolves the branch is recorded as
reviewed. A `worktree check` running at the same time widens it (both write `<sess>/out.bundle`);
that clash usually fails verify and leaves `commitError`, which fails safe.

Bundling inside the bot's PreToolUse decision for the reviewer call gives a real happens-before:
the reviewer cannot start until the sha is pinned. That moves the rule from ADR 0025's amendment
(rule 7, "at the reviewer's launch") to the hook path, so it is the ADR's line to change, then a
pass. `spawn.WorktreeBundle` (`internal/spawn/worktree.go`) would also need a context so the hook
decision cannot hang. Found by the P31 review (`238ea6d`).
