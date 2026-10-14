---
date: 2026-10-14
status: resolved
area: tooling / deleting worktrees on Windows
supersedes: 2026-10-11-windows-worktrees-long-paths.md
promoted_to: docs/reference/claude-code-on-windows.md
---

# An empty robocopy mirror purges through a junction into its target; `\\?\` Remove-Item does not

## Question

The Windows reference prescribed `robocopy <empty> <dir> /MIR` to clear a long-path worktree, and
the 0005 critic review (objection 1a) proposed adding `/XJ` so the mirror would not purge through
a junction, noting that whether it does was untested. A downstream incident had a junction from a
worktree into the main checkout, and a recursive delete of the worktree emptied the main
checkout's `out/`. Which removal commands reach through a junction, and which do not?

## What was tried

The reviewer, 2026-10-14, Windows 11 build 26300, Windows PowerShell 5.1.26100, `LongPathsEnabled` = 0,
on throwaway targets: a worktree-shaped directory holding a junction (`mklink /J`) to a separate
directory with files in it, removed by five delete attempts and one link removal with `cmd /c rmdir`,
then the target inspected.

| Command (PowerShell) | Link | Target |
|---|---|---|
| `robocopy <empty> <wt> /MIR /XJ` | removed | purged |
| `robocopy <empty> <wt> /MIR /XJD` | removed | purged |
| `robocopy <empty> <wt> /MIR` | removed | purged |
| `Remove-Item -LiteralPath $path -Recurse -Force` (plain path) | removed | intact |
| `Remove-Item -LiteralPath ('\\?\' + $path) -Recurse -Force` | removed | intact |
| `cmd /c rmdir` on the `\\?\`-prefixed junction name | removed | intact |

Also tried: `Get-ChildItem -Recurse -Force -Attributes ReparsePoint` on a plain path, over a tree
with a junction at a 420-character path: `DirectoryNotFoundException`, the junction not listed.
The same listing on the `\\?\`-prefixed path found it.

## Result

Robocopy's empty mirror purged a junction's target with `/XJ`, `/XJD` or no flag (`/XJF` untested),
so the route the reference carried and the fix the critic proposed both reach through a link.
(Inference, not observed: the flags change what is copied, not what `/MIR` purges.) `Remove-Item -Recurse -Force`, plain or `\\?\`-prefixed,
removed the junction and left its target intact; symbolic links were not tested. `cmd /c rmdir`
on the prefixed junction name removes the link alone. Which tool deleted through the junction in
the downstream incident is not established; robocopy is the one route tested that does.

Both the listing and the `rmdir` need the `\\?\` form on a long path; a plain path misses the
link, and a link missed is a link a robocopy mirror would follow (symlinks untested; `Remove-Item`
left junction targets intact). Promoted to
`docs/reference/claude-code-on-windows.md`, Git worktrees: the robocopy route is dropped, the
`\\?\` `Remove-Item` is the removal, links are listed and removed first in the prefixed form.
