---
date: 2026-10-12
status: resolved
area: tooling / deleting worktrees on Windows
promoted_to: docs/reference/claude-code-on-windows.md
---

# Deleting a long-path worktree on Windows: plain `Remove-Item` fails, the `\\?\` prefix works

## Question

`git worktree remove` fails on Windows when Gradle or Android build output nests below the
260-character limit (finding `2026-10-11-windows-worktrees-long-paths.md`, which found the robocopy
mirror). Dystopo's `CLAUDE.md` said to delete with `Remove-Item -LiteralPath \?\<path> -Recurse
-Force`, and its coordinator reported on 2026-10-12 that `Remove-Item` fails on long Gradle paths while
`[System.IO.Directory]::Delete('\\?\' + path, true)` works. What exactly works?

## What was tried

Dystopo task T84, on its development machine (Windows 11, `LongPathsEnabled` = 0, Windows PowerShell
5.1.26100): a tree of eight 45-character directories (a 504-character path) with one file, created
through the `\\?\` prefix, then deleted five ways.

| Command (PowerShell) | Result |
|---|---|
| `Remove-Item -LiteralPath $path -Recurse -Force` | fails: `DirectoryNotFoundException`, "Could not find a part of the path" |
| `Remove-Item -LiteralPath ('\?\' + $path) -Recurse -Force` (one backslash, as Dystopo's `CLAUDE.md` printed it) | fails: `DriveNotFoundException`, "A drive with the name '\?\C' does not exist" |
| `Remove-Item -LiteralPath ('\\?\' + $path) -Recurse -Force` | works, including a read-only file in the tree |
| `[System.IO.Directory]::Delete('\\?\' + $path, $true)` | works when no file is read-only; fails on a read-only file ("Access to the path ... is denied") |
| `robocopy <empty> <dir> /MIR`, then delete | works (the earlier finding; not re-run) (unsafe with junctions: `2026-10-14-robocopy-mirror-purges-through-junctions.md`) |

## Result

On Windows PowerShell 5.1 without long paths enabled, a recursive delete of a long-path tree works only
through the `\\?\` prefix. With it, both `Remove-Item -LiteralPath ('\\?\' + $path) -Recurse -Force`
and `[System.IO.Directory]::Delete('\\?\' + $path, $true)` work; `Remove-Item -Force` also clears
read-only attributes, `Directory.Delete` does not. The coordinator's observation fits a prefix that had lost a
backslash: the `CLAUDE.md` line had been edited through a shell that collapsed `\\` to `\`. Write the
prefix in single quotes in PowerShell and never pass it through Git Bash. After the delete,
`git worktree prune`; a worktree the harness locked needs `git worktree unlock` first.

Promoted 2026-10-14 to `docs/reference/claude-code-on-windows.md`, Git worktrees. Dystopo carries
the same facts in its `docs/reference/windows-and-shells.md`.
