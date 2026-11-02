---
date: 2026-11-02
kind: bug
related: [2026-10-31-broodforge-deploys-broodforge.md]
---

# `redeploy` cannot replace the binary under the running bot

`scripts/dev-task.ps1` `redeploy` builds first, on the comment's assumption that go replaces
`bin\broodforge.exe` under a running bot, and stops the bot only after the build. On the PC, with
the dev-live bot up from the boot task, the build fails:

```
The process cannot access the file because it is being used by another process
```

Observed 2026-11-02 by the P34 coder running `CGO_ENABLED=0 go build -o bin/ ./cmd/broodforge` from
the checkout while the task's bot served; it built into a copy of the tree instead. Whether the
lock is the running process alone, the S4U task's account, or the antivirus of
`docs/reference/go-on-windows.md` is not known; nobody has reproduced it through `redeploy` itself.

Anchor: `scripts/dev-task.ps1` (the `redeploy` branch, the build before `Stop-Bot`); the run skill's
step 2c; `docs/operator/` boot-task step.

## Open

- Reproduce through `redeploy` while the task's bot is up; record the finding.
- The fix shape: build under a staging name beside the binary and rename it after `Stop-Bot`, so a
  failed build still leaves the bot up; or stop first and accept the gap.
