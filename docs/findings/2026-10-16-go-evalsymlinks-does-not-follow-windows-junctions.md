---
date: 2026-10-16
status: resolved
area: go / windows paths
promoted_to: docs/reference/claude-code-on-windows.md
---

# filepath.EvalSymlinks does not follow Windows directory junctions

## Question

The PreToolUse hook must decide from the canonical path of a tool's target, with every link
followed. Does `filepath.EvalSymlinks` resolve Windows directory junctions as well as symlinks?

## What was tried

Go 1.27.2, Windows 11. Paths through a symlink (`C:/Users/All Users`) and through a junction
(`C:/ProgramData/Desktop`), both pointing out of the tree the hook protects. `filepath.EvalSymlinks`
and `os.Readlink` were compared on each, and the hook's own walk was run against both as cases
in `tests/hook/cases.jsonl` (the two lines whose note begins "Windows symlink" and "Windows junction").

## Result

`filepath.EvalSymlinks` does not follow a directory junction: `os.Lstat` reports one as
`fs.ModeIrregular`, not `fs.ModeSymlink`, and the call returns the path unresolved, so a
junction out of the tree would pass an `EvalSymlinks`-based check. `os.Readlink` reads both
a symlink and a junction. `canonical` in `internal/hook/paths.go` therefore walks the path
from its root itself: any component whose `Lstat` mode is symlink or irregular is replaced by its
`Readlink` target, up to 64 links, and a link that cannot be read is an error, which the hook turns
into a deny (fail closed). Both cases deny and pass on the PC. Not tested: other irregular entries
(for example OneDrive placeholders), which the same rule would treat as links and deny if unreadable.
