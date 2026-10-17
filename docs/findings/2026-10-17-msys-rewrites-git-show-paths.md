---
date: 2026-10-17
status: resolved
area: tooling / Git Bash on Windows
promoted_to: docs/reference/claude-code-on-windows.md
---

# Git Bash rewrites the path in `git show hatchery/main:<path>`; name the commit by sha

## Question

The update skill read base files with `git show hatchery/main:<path>`. Does that work in the Bash
tool on Windows?

## What was tried

The critic of ADR 0016 ran `git show hatchery/main:<path>` in Git Bash (Windows 11), quoted and
unquoted, then `S=$(git rev-parse hatchery/main)` with `git show $S:<path>`, and the same command
with `MSYS_NO_PATHCONV=1`. The reviewer separately ran `git diff ... | git apply --3way` in
PowerShell 5.1 and in Bash.

## Result

- `git show hatchery/main:<path>` fails in Git Bash, quoted or not: MSYS reads the argument as a
  path list and rewrites it before git sees it.
- `git show <sha>:<path>` after `S=$(git rev-parse hatchery/main)` works, as does
  `MSYS_NO_PATHCONV=1 git show hatchery/main:<path>`. A Node child process is not rewritten, which is
  why the update helper reads the base through `child_process`.
- Skills and templates therefore name the sha, never `hatchery/main:`, in a `git show`.
- Related, same session: in PowerShell 5.1 `git diff ... | git apply --3way` fails every file with
  `patch does not apply` and leaves `git status` empty (the pipe re-encodes the patch); in Bash it
  applies. The update command is Bash-only.

Both facts are in `docs/reference/claude-code-on-windows.md`, Shells.
