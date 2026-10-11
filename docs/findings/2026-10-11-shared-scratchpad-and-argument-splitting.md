---
date: 2026-10-11
status: resolved
area: tooling / parallel agents on Windows
promoted_to: docs/reference/claude-code-on-windows.md
---

# Parallel agents share one scratchpad directory, and PowerShell 5.1 splits dotted arguments

## Question

Two defects seen while eight implementers ran in parallel in Dystopo wave 11: one agent's commit
message files were overwritten by another agent's, and a Gradle property passed as
`-PdystopoVersion=0.1.4-alpha` arrived split at the dot.

## What was tried

- Both agents wrote `msg1.txt` and `msg2.txt` into the session scratchpad directory; the second
  write replaced the first. Unique names (task id in the file name) fixed it.
- `powershell -File script.ps1 -PdystopoVersion=0.1.4-alpha`: Windows PowerShell 5.1 passed the
  value without the part after the dot. Quoting the whole argument (`"-PdystopoVersion=0.1.4-alpha"`)
  passed it intact. The same shell drops an empty argument (`-Osm ''`) entirely, which is why the
  Dystopo scripts use `-NoOsm` switches rather than empty strings.
- Bash heredocs in the same environment turned `\\n` into a literal backslash-n inside a YAML file
  and refused apostrophes (finding `2026-10-11-bash-heredoc-apostrophes.md`); the Write and Edit
  tools avoided both.

## Result

Agents running in parallel must use unique file names in shared scratch space. Arguments containing
dots or dashes passed to `powershell -File` are quoted whole. Files that contain apostrophes,
backticks or backslash sequences are written with the Write or Edit tool, never through a heredoc.
