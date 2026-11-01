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

Plan: tester, then the carry-back Send (`.claude/skills/carry-back/SKILL.md`). Already done on this PC:
`C:\msys64\mingw64\bin` is on the user `PATH` (`where.exe gcc` finds it) and `docs/reference/go-on-windows.md`
already says so; what remains of its per-command prefix is the line for a shell older than the change,
which it calls a gap, so leave it.
- Decision on the Open item: one sentence in `.claude/agreement/delegation.md` (nine lines today): an agent
  that meets a missing tool stops and reports it as a gap to install, never substitutes a weaker command
  for a documented check; the coordinator briefs a runner to install it or put it on `PATH` for good.
- Add a departure row to `docs/design/decisions/0024-decanted-from-hatchery.md` for that agreement file,
  then the carry-back Send. Out of scope: the agent files, the reference doc.
- Verify: the sentence is in the file, the row is in the table, `CGO_ENABLED=1 go test -race ./...` runs
  from a fresh shell with no `PATH` prefix.
