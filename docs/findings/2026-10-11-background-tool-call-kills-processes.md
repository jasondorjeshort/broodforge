---
date: 2026-10-11
status: resolved
area: tooling / Claude Code on Windows
promoted_to: docs/reference/claude-code-on-windows.md
---

# A long-lived process started inside a background tool call dies when that call ends

## Question

Agents that boot an Android emulator (or any server) from a Bash or PowerShell tool call with
`run_in_background` found the process gone as soon as the call finished, and a foreground call
that launched it hung until the tool timeout. Dystopo T72 and T81 both hit this.

## What was tried

- `tools/android-emu.ps1` launched from a background PowerShell tool call: the emulator was gone
  from `adb devices` right after the call ended.
- The same launcher in a foreground call: the call did not return until the emulator exited or
  the timeout fired, because the script waited on its child.
- Launching through `Win32_Process.Create` (WMI) from a short foreground call: the emulator
  outlived the call and was still present minutes later. `Start-Process -WindowStyle Hidden` with
  the PID recorded behaved the same.

## Result

A tool call's process tree is torn down when the call returns. Anything that must outlive a call
(emulators, servers, soak scripts) has to be detached: `Win32_Process.Create` or `Start-Process`,
with the PID written to a file so the agent can stop exactly that process later. The pattern now
lives in Dystopo's `tools/server-soak.ps1` (writes `soak-script.pid`) and in the emulator launcher
notes in its `CLAUDE.md`.
