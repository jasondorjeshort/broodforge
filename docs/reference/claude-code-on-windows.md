---
verified: 2026-10-11
source: observed (twelve waves of parallel agents on a downstream project; Windows 10, Claude Code 2.1.x, Git Bash and Windows PowerShell 5.1)
---

# Claude Code on Windows: what holds

## Processes

- A tool call's process tree ends with the call. Emulators, servers and soak scripts are started
  detached (`Win32_Process.Create` or `Start-Process -WindowStyle Hidden`) with the PID written to a
  file, and stopped later by that PID only.
- Never stop a process by name or command-line substring: with several agents on one machine that
  kills another agent's emulator or the coordinator's live server. Stop by PID, or by serial for an
  emulator (`adb -s emulator-NNNN emu kill`).
- Each parallel agent gets its own ports and its own emulator (AVD name, console port, host port)
  assigned in the brief; a shared port range in the brief is how collisions are avoided.

## Git worktrees

- One worktree per agent under `.claude/worktrees/` [no-check: a build-phase project's directory, absent in the base]. Removal on long paths: `robocopy <empty> <dir>
  /MIR`, then `Remove-Item -Recurse -Force`, then `git worktree prune`; `git worktree unlock` first
  if the harness locked it.
- The harness creates a worktree only when the shell's cwd is inside the repo; reset the cwd before
  launching an agent after any `cd` elsewhere.

## Shells

- Bash heredocs: no apostrophes, no `\\n` sequences; use the Write or Edit tool for such text.
- Windows PowerShell 5.1: quote arguments containing dots or dashes whole when calling
  `powershell -File`; it drops empty-string arguments; `&&` and `||` do not exist; `2>&1` on a
  native command wraps stderr in error records.
- `gh ... --jq` with a backtick inside the expression breaks in PowerShell; run `gh` from bash.

## Working directory and git

- A `cd` to another repository inside a Bash call may not take effect (the harness resets the working directory between calls, and did so mid-session). Name the repository: `git -C <path> ...`, and start any commit or push command with `git rev-parse --show-toplevel` so the report shows where it ran. Finding `../findings/2026-10-11-reported-push-that-never-happened.md`.
- Prove a push: after `git push`, `git fetch` and compare `git rev-parse HEAD` with the remote ref; report both SHAs. A command's own success output is not evidence of what reached the remote.
- Never truncate a status check to its first line; `git status --short` in full.

## Shared state

- The session scratchpad is shared by every agent of the session: unique file names per agent.
- A shared download cache (environment variable pointing at one directory) is safe for concurrent
  readers; agents never delete inside it and never bump a cache version that would wipe it for the
  others.
- The Windows Recycle Bin receives files deleted through Explorer only; agent shells delete
  permanently. A folder that turns up in the bin was deleted by a person.

## Usage meters

- The status line writes the 5-hour and weekly percentages to a JSON file under `~/.claude/`; a
  PreToolUse hook on `Agent|Workflow` reads it and refuses new launches above the threshold. What
  stops at the threshold is launching new implementers; verifiers and merges of work already in
  flight may finish (owner's clarification, 2026-10-10).
