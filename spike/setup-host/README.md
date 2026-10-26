<!-- SPDX-License-Identifier: GPL-3.0-or-later -->
# spike/setup-host/

P20 of `docs/design/08-implementation-plan-setup.md`: the Windows claims P23's `setup host` and
`setup service` rest on (W1, W3a, W4, W6, and the root's ACL), run on the owner's PC against two
disposable local accounts, `bf-spike` and `bf-spike2`, never the live `broodforge` account or its
task. The finding the output fills is `docs/findings/2026-10-26-spike-windows-setup-host-claims.md`,
which also says what each claim means.

| File | What it is |
|---|---|
| `run.ps1` | Elevated. Creates the two accounts with generated passwords held in memory only, `%ProgramData%\Broodforge-spike\` with inheritance off (SYSTEM, Administrators, `bf-spike`), copies the engine to its `engine\`, registers the task `bf-spike-test`, runs each claim and prints one `claim` line each. Removes nothing. |
| `probe-unelevated.ps1` | Not elevated, as the owner. Tries to read the file `bf-spike` wrote and to list the directory; prints two `claim` lines. Reads only. |
| `cleanup.ps1` | Elevated. Removes the task, the processes, profiles and accounts of `bf-spike` and `bf-spike2`, and `%ProgramData%\Broodforge-spike\` only while it holds the spike's marker and nothing else. `-WhatIf` shows what it would do. Safe to run again. |

## Before you start

- `%ProgramData%\Broodforge-spike\` must not exist: `run.ps1` refuses rather than touch a real instance's
  directory. Nothing named `bf-spike` may exist either (`run.ps1` says so; `cleanup.ps1` clears it).
- The Secondary Logon service (`seclogon`) must not be disabled; `Start-Process -Credential` uses it.
- About three minutes. Console windows open and close while it runs: note whether one appeared on
  your desktop during the first seconds (W3a says the child runs in a console there).
- Nothing is typed into the scripts. No password is shown, and none is needed from you.

## Steps

1. An elevated PowerShell (Start, type `PowerShell`, Run as administrator; one UAC click). Point
   `-EnginePath` at your native `claude.exe`; `(Get-Command claude).Source` prints it.

   ```
   cd <repo>\spike\setup-host
   powershell.exe -NoProfile -NonInteractive -ExecutionPolicy Bypass -File .\run.ps1 -EnginePath "$env:USERPROFILE\.local\bin\claude.exe"
   ```

2. A normal PowerShell, not as administrator, while step 1's results are still in place:

   ```
   cd <repo>\spike\setup-host
   powershell.exe -NoProfile -NonInteractive -ExecutionPolicy Bypass -File .\probe-unelevated.ps1
   ```

3. Back in the elevated PowerShell:

   ```
   powershell.exe -NoProfile -NonInteractive -ExecutionPolicy Bypass -File .\cleanup.ps1
   Get-LocalUser bf-spike
   Test-Path "$env:ProgramData\Broodforge-spike"
   ```

   Expect `cleanup: done, 0 left`, an error from `Get-LocalUser`, and `False`.

## What to paste back

Every line the three scripts print (`claim ...`, `spike: ...`, `cleanup: ...`), as printed, and
whether a console window appeared in step 1. The scripts already write `~`, `<profiles>`, `<host>`,
`<owner>` and `%ProgramData%` in place of your paths and names; nothing else is needed.

A claim line reads `claim <id>: hold|refuted|untested: <evidence>`; `untested` means an earlier step
failed and the claim was not reached. A run where every claim holds looks like this (values vary):

```
claim W1: hold: registered=no password LogonType=InteractiveToken RunLevel=LeastPrivilege ExecutionTimeLimit=PT0S
claim W3a: hold: seclogon=Stopped/Manual profile-before=none exit=0 whoami=<host>\bf-spike profile-after=<profiles>\bf-spike on-disk=True status=0 loaded-now=False
claim W3a-env: hold: child USERPROFILE=<profiles>\bf-spike; APPDATA=...; TEMP=...; account profile=<profiles>\bf-spike
claim W4: hold: sha256-equal=True engine-acl=protected=True [...] start-exit=0 output=2.1.310 (Claude Code); exit=0 cfg-entries=0
claim W6: hold: noninteractive-host=True ps=5.1... new-localuser=ok register-password=ok LogonType=Password ExecutionTimeLimit=PT0S ping-owner-before=bf-spike set-localuser=ok ping-alive-after-rotate=True reregister-new-password=ok task-ran-as=<host>\bf-spike LastTaskResult=0
claim ACL-second-account: hold: root-acl=protected=True [...] write-exit=0 probe-exists=True probe-owner=<host>\bf-spike ... bf-spike2-type-exit=1 bf-spike2-type-stderr=Access is denied. ...
spike: done; next probe-unelevated.ps1 from a non-elevated PowerShell, then cleanup.ps1 elevated
claim ACL-owner-file: hold: elevated=False EnableLUA=1 open denied (UnauthorizedAccessException)
claim ACL-owner-list: hold: elevated=False EnableLUA=1 listing denied (UnauthorizedAccessException)
```

`W3a-env` refuted (the child sees your `USERPROFILE`, not the account's) is a result for P23, not
a failed run.

## When something is left behind

Whatever `run.ps1` created stays until `cleanup.ps1` removes it: the accounts `bf-spike` and
`bf-spike2`, their profiles under the profiles folder, the task `bf-spike-test`, and
`%ProgramData%\Broodforge-spike\` (the engine copy is inside it). Run `cleanup.ps1` again first; it prints
`cleanup: left <what>: <reason>` for each thing it could not remove. A loaded profile frees itself
once the account's processes end (the `ping` ends within a minute). By hand, elevated, the same
removals in the same order:

```
Get-CimInstance Win32_Process | Where-Object { (Invoke-CimMethod -InputObject $_ -MethodName GetOwner).User -in 'bf-spike','bf-spike2' } | ForEach-Object { Stop-Process -Id $_.ProcessId -Force }
Unregister-ScheduledTask -TaskPath '\' -TaskName bf-spike-test -Confirm:$false
Get-CimInstance Win32_UserProfile | Where-Object { $_.LocalPath -match '\\bf-spike2?(\..+)?$' } | Remove-CimInstance
Remove-LocalUser -Name bf-spike
Remove-LocalUser -Name bf-spike2
Remove-Item -LiteralPath "$env:ProgramData\Broodforge-spike" -Recurse -Force
```

Run the last line only when `%ProgramData%\Broodforge-spike\bf-spike.marker` exists. One trace stays by
design: registering a task with a password grants the account "Log on as a batch job", and the local
policy keeps that entry as an unresolved SID after the account is gone; it grants nothing to anyone.
