---
date: 2026-10-26
status: open
area: spike / setup host (Windows)
---

# Windows `setup host` claims W1, W3a, W4, W6 and the root ACL: the spike is written, the verdicts wait for the owner's run

## Question

P23's `setup host` and `setup service` (ADR 0027, `0027-setup-commands.md`, Windows) script the
PC's account, root, engine copy and task on claims no one has run. Each one decides what P23
generates:

- W3a: `Start-Process -Credential <account> -LoadUserProfile` from an elevated PowerShell creates
  the account's profile with no interactive sign-in. Failing, the PC gains one step (the owner signs
  in once as the account).
- W4: the native `claude.exe` copied under `%ProgramData%\Broodforge-spike\engine\` runs as the account
  (`--version`) with `CLAUDE_CONFIG_DIR` in the account's profile. Failing, the native installer
  runs as the account instead.
- W6: `New-LocalUser`, `Register-ScheduledTask -User -Password -Force`, `Set-LocalUser -Password` and
  `Start-ScheduledTask` run non-interactively with the password a SecureString built in the script,
  and a rotation leaves a process already running as the account alive.
- W1: a logon task for a standard account registers with no password, and `ExecutionTimeLimit`
  zero reads back as `PT0S` (P23's `fix  service  ExecutionTimeLimit is <value>` depends on it).
- ACL (reviewer finding B1, 2026-10-26): with `%ProgramData%\Broodforge-spike\` created elevated,
  inheritance off and entries for SYSTEM, Administrators and the account only, a file the account
  writes there is unreadable by a second standard account and by the owner's non-elevated shell,
  and that shell cannot list the directory.

W3 (a password-logon task creating the profile) is refuted already (critic, Microsoft's event 1511
article: a temporary profile) and is not retried.

## What was tried

Not run yet. The owner decided on 2026-10-26 that the spike runs on their PC against disposable
accounts it creates and deletes, never `broodforge`, with no cloud machine. The scripts, in
`spike/setup-host/` (runbook `README.md`), use built-in Windows PowerShell 5.1 cmdlets only:

- `run.ps1`, elevated, as `powershell.exe -NoProfile -NonInteractive -File`: refuses when
  `%ProgramData%\Broodforge-spike\` or anything named `bf-spike` exists; creates `bf-spike` and
  `bf-spike2` (32-character CSPRNG passwords as SecureStrings, never printed); the root with a
  marker file and the ACL above (`bf-spike` Modify), `engine\` with `bf-spike` ReadAndExecute and
  the engine copied in (SHA-256 compared); W3a runs `whoami` and records the child's
  `USERPROFILE`, `APPDATA`, `TEMP` through `cmd.exe` as `bf-spike` with `-LoadUserProfile -Wait`,
  comparing `Win32_UserProfile` before and after; W4 runs `claude.exe --version` from the copy with
  the profile variables and `CLAUDE_CONFIG_DIR` set to the account's profile and
  `DISABLE_AUTOUPDATER=1`; the ACL probe file is written as `bf-spike` and read and listed as
  `bf-spike2`; W1 registers `bf-spike-test` at logon with an interactive principal and reads the XML
  back; W6 re-registers it at startup with `-User -Password -Force`, starts `ping -n 61` as
  `bf-spike`, rotates the password, checks the ping's owner, re-registers with the new password and
  runs the task, which writes its `whoami`.
- `probe-unelevated.ps1`, as the owner without elevation: the file read and the listing.
- `cleanup.ps1`, elevated: removes only names from its fixed lists and the root only with the
  marker; checked with `-WhatIf` on the PC before any run (every item `absent`, nothing touched).

Departures from P20's text in the plan, for the architect: the script is `run.ps1`, not `host.ps1`;
the passwords are generated in the script, not read from stdin, and the script runs as `-File`, so
P23's `-Command -` stdin form is not exercised (in Windows PowerShell 5.1 that form reads stdin line
by line as if typed, which P23 should test with its own template); the root is
`%ProgramData%\Broodforge-spike\` itself with a marker, not `Broodforge-spike[\]spike`, and has a SYSTEM entry;
a second account and the non-elevated probe serve the ACL claim; `W3a-env` is added because a
`-Credential` child may inherit the caller's environment; the one `claude -p` turn is not run (it
needs a login as the account, which is `setup login`'s); this file's name differs from the plan's.

The owner's run (date, Windows build, engine version): to fill.

## Result

Pending. One row per `claim` line the scripts print; the evidence is the line's text after the verdict.

| Claim | Holds when | Verdict | Evidence |
|---|---|---|---|
| W1 | registered with no password, `LogonType` `InteractiveToken`, `ExecutionTimeLimit` `PT0S` | | |
| W3a | no profile before; after the child exits it exists on disk, not temporary, and `whoami` printed `bf-spike` | | |
| W3a-env | the child's `USERPROFILE` is the account's profile, not the caller's | | |
| W4 | the copy is byte-equal, `--version` prints `<x.y.z> (Claude Code)` and exits 0 as `bf-spike` | | |
| W6 | non-interactive host; every cmdlet ran; the ping outlived the rotation; the task ran as `bf-spike` on the new password | | |
| W6 time limit | `ExecutionTimeLimit` as registered with `-User -Password` (in the W6 line) | | |
| ACL-second-account | `bf-spike2` cannot read the file `bf-spike` wrote | | |
| ACL-owner-file | the owner's non-elevated shell cannot open that file | | |
| ACL-owner-list | the owner's non-elevated shell cannot list the root | | |
| Console window | the owner saw the W3a child's console on the desktop (reported by the owner) | | |
| Cleanup | `cleanup: done, 0 left`, `Get-LocalUser bf-spike` an error, the root gone | | |

## Next

The owner runs `spike/setup-host/README.md` and pastes the lines back; the coordinator has them
filled into Result (verdict and evidence per row, the run's date, build and engine version), then
sets `status: resolved` with the durable facts promoted where P23 puts them, before P23's brief.
