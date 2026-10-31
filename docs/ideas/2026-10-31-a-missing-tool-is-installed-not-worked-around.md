---
date: 2026-10-31
kind: chore
---

# A missing tool is installed, not worked around

The pusher's pre-push run on 2026-10-31 found no `gcc` on `PATH`, so it dropped `-race` and ran
`CGO_ENABLED=0 go test ./...` instead of the `.claude/rules/go.md` test line, then reported the
gap. The compiler was installed all along (`C:\msys64\mingw64\bin\gcc.exe`, MSYS2 MinGW64 16.2.0;
`docs/reference/go-on-windows.md` prescribed a per-command `PATH` prefix). The owner's rule, in
their words: "We should (as policy) install software rather than try to work around its absence."

Anchor: `.claude/agreement/delegation.md` or the agent definitions of `runner` and `pusher`
(base-owned; a carry-back, no departure here since no base file changes in this project). The
rule: an agent that meets a missing tool stops and reports it as a gap to install, never substitutes
a weaker command for a documented check; the coordinator briefs a runner to install it or put its
directory on the user `PATH` for good, verifies, and the reference doc drops the workaround. The
same sitting on this PC: `C:\msys64\mingw64\bin` appended to the user `PATH`, `go-on-windows.md`
updated.

## Open

- Whether the rule sits in `delegation.md` (one sentence) or in each agent file's rules.
