---
date: 2026-10-23
kind: chore
related: [2026-10-23-sessions-load-none-of-the-projects-instructions-agents-or-hooks.md]
---

# The cli tests fail when they run inside a Broodforge session

Found by the first live session (s0005, which noticed it and worked around it) and by a reviewer:
`TestProofHookPreToolUse` (`internal/cli/hook_test.go`, around line 46) expects the hook to report
`BROODFORGE_SESSION` as not set and gets `defer` instead, because a developer working inside a
Discord session has the variable set in the environment. The session ran its tests with the
variable unset in the command; any agent or person who forgets sees a red suite for no reason.

Shape to consider: the test (and any other that reads the `BROODFORGE_*` environment) unsets what it
depends on with `t.Setenv(name, "")` or runs the hook with a built environment, so the suite is the
same inside and outside a session. A fitness check: `BROODFORGE_SESSION=s0001 go test ./...` stays in
the pusher's test line or a note in `.claude/rules/go.md`.

## Open

- Which other tests read the process environment (grep for `Getenv` and `BROODFORGE_` under
  `internal/**/*_test.go`).
