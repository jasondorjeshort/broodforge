---
date: 2026-10-21
kind: question
related: [../findings/2026-10-21-claude-code-best-practices-for-a-build-phase-go-project.md]
---

# A coder Stop hook could refuse to end a pass on a broken build

A `Stop` hook in the coder agent's frontmatter (it runs as `SubagentStop`) would run
`CGO_ENABLED=0 go build ./... && go vet ./...` and block the end of a pass on a failure. No `-race`
and no tests: `-race` needs gcc on `PATH`, and antivirus has quarantined `bin/`
(`docs/findings/2026-10-16-bullguard-sentry-quarantined-the-binary-and-its-process-tree.md`).
`.claude/agents/coder.md` is base-owned, so this is a carry-back to Hatchery.

Rejected: a settings-level Stop hook, which fires after coordinator turns that edit no Go.

Trigger: two coder reports in one wave claim done on a build that pusher check 8 then fails.

Question: wait for the trigger, or build it now?

Answer:
