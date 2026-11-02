---
date: 2026-11-02
kind: question
related: [2026-10-23-session-land-pushes-to-main-mid-session.md]
---

# `worktree handoff` lands `main` with the role unchecked

Not sanity checked.

ADR 0028 as amended 2026-11-02 names four doors to a land and says who is judged at each: the
driver mid-session, the ender at the end, `--as --roles` for `session land` on the host. The fifth
command that lands is `worktree handoff <id>` (`internal/cli/worktree.go`, `hostGate`): on a `main`
project it runs the land, needs the channel to have a `roles.land` list and the review fact for the
tip, and checks nobody's role. The amendment names `session end` on the host (no roles, so a
fallback) but not this command, so the two host doors disagree: `session end s0001` falls back to
the session branch, `worktree handoff s0001` lands `main`.

The host is the owner's shell, so the hole is narrow; the inconsistency is the point. Either the
host is trusted at every door (then `session end` should land too, and the amendment's "an end
with no roles behind it falls back" needs a host exception), or `worktree handoff` takes `--as
--roles` like `session land` and refuses without them.

Anchor: `internal/cli/worktree.go` (`hostGate`); ADR 0028 Amendments 2026-11-02; P34's coder
report (left as "the architect's call").

## Open

- Which way; then one line in the ADR's amendment and the matching test in `internal/session/land_test.go`.
