<!-- SPDX-License-Identifier: GPL-3.0-or-later -->
# spike/setup-host/

P20 of `docs/design/08-implementation-plan-setup.md`: the Windows claims P23's `setup host` and
`setup service` rest on (W1, W3a, W4, W6, and the root's ACL), run on the owner's PC against two
disposable local accounts, `bf-spike` and `bf-spike2`, never the live `broodforge` account or its
task. The finding the output fills is `docs/findings/2026-10-26-spike-windows-setup-host-claims.md`,
which also says what each claim means.

| File | What it is |
|---|---|
| `run.ps1` | Elevated. Creates the two accounts with generated passwords held in memory only, `%ProgramData%\Broodforge-spike\` with inheritance off (SYSTEM, Administrators, `bf-spike`), copies the engine to its `engine\`, enables the `Microsoft-Windows-TaskScheduler/Operational` log when it is off (noting so in `bf-spike-tslog.txt` in that directory), registers the task `bf-spike-test`, runs each claim and prints one `claim` line each, then the `w6` detail lines. Removes nothing. |
| `probe-unelevated.ps1` | Not elevated, as the owner. Tries to read the file `bf-spike` wrote and to list the directory; prints two `claim` lines. Reads only. |
| `cleanup.ps1` | Elevated. Removes the task, the processes, profiles and accounts of `bf-spike` and `bf-spike2`, and `%ProgramData%\Broodforge-spike\` only while it holds the spike's marker and nothing else. Before that, when the directory holds `bf-spike-tslog.txt`, it disables the TaskScheduler log again and reads its state back; while the log still reads enabled it keeps the directory. `-WhatIf` shows what it would do. Safe to run again. |

## Before you start

- `%ProgramData%\Broodforge-spike\` must not exist: `run.ps1` refuses rather than touch a real instance's
  directory. Nothing named `bf-spike` may exist either (`run.ps1` says so; `cleanup.ps1` clears it).
- The Secondary Logon service (`seclogon`) must not be disabled; `Start-Process -Credential` uses it.
- About five minutes: W6 tries the task's start up to three ways and waits up to 30 s after each.
  Console windows open and close while it runs: note whether one appeared on your desktop during
  the first seconds (W3a says the child runs in a console there).
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

   Expect `cleanup: done, 0 left`, an error from `Get-LocalUser`, and `False`. When `run.ps1` found
   the TaskScheduler log disabled, `cleanup.ps1` also prints `cleanup: removed enable of
   Microsoft-Windows-TaskScheduler/Operational (disabled again, reads enabled=False)`.

## What to paste back

Every line the three scripts print (`claim ...`, `w6 ...`, `spike: ...`, `cleanup: ...`), as
printed, and whether a console window appeared in step 1. The `w6` lines carry the TaskScheduler
log's own messages; the scripts already write `~`, `<profiles>`, `<host>`, `<owner>`,
`%ProgramData%` and `S-1-5-21-<machine>` in place of your paths, names and machine SID; nothing else
is needed.

A claim line reads `claim <id>: hold|refuted|untested: <evidence>`; `untested` means an earlier step
failed and the claim was not reached, or that an earlier start attempt already worked.

W6 now ends in up to three start attempts, one line each, in order, stopping at the first that
produces the task's output file; the `W6` line says which (`started-by`):

- `W6-a`: `Start-ScheduledTask` on the task as W6 registered it (`AtStartup` trigger).
- `W6-b`: the task re-registered with the same password, action and settings (which carry
  `-AllowStartIfOnBatteries -DontStopIfGoingOnBatteries`) and a one-time trigger a year ahead, then
  `Start-ScheduledTask`.
- `W6-c`: `schtasks /run /tn \bf-spike-test` on the task as last registered.

Each prints `method`, `prepare` (`W6-b` only: `registered`, or the error, which leaves `W6-b`
untested), `state-before`, `start` (`ok`, `schtasks`'s exit and text, or the error), `task-ran-as`,
`LastTaskResult` in hex and decimal (`0x00041303` is "has not run"), `LastRunTime` and
`state-after`. The output file is credited by its creation time to the last attempt that started
before it, not to the attempt that happened to be waiting: a `late-output=` at the end means it
appeared only after that attempt's 30 s (after `W6-c`, the script waits 30 s more for one); an
attempt that started after the file was created reads `untested: the output file (created ...)
predates this start`; `not tried: the output file existed before the first start` means it was
there before `W6-a`. The `W6` line holds only when an attempt (`started-by=W6-...`) produced output
as `bf-spike`, and then also carries `output-created`. After the claims come the `w6` lines: `w6 ts-log` (was the log enabled, did
`wevtutil` enable it), `w6 readback` after the second registration and after `W6-b`'s (action,
trigger, principal, state, settings), `w6 battery`, and after each attempt the log's events naming
the task that are new since the last attempt (`w6 event after W6-a: id=<n> time=<hh:mm:ss>
<message>`, or `w6 events after W6-a: none ...`), and once more `after the final wait`.

A run where every claim holds looks like this (values vary):

```
claim W1: hold: registered=no password LogonType=InteractiveToken RunLevel=LeastPrivilege ExecutionTimeLimit=PT0S
claim W3a: hold: seclogon=Stopped/Manual profile-before=none exit=0 whoami=<host>\bf-spike profile-after=<profiles>\bf-spike on-disk=True status=0 loaded-now=False
claim W3a-env: hold: child USERPROFILE=<profiles>\bf-spike; APPDATA=...; TEMP=...; account profile=<profiles>\bf-spike
claim W4: hold: sha256-equal=True engine-acl=protected=True [...] start-exit=0 output=2.1.310 (Claude Code); exit=0 cfg-entries=0
claim W6: hold: noninteractive-host=True ps=5.1... new-localuser=ok register-password=ok LogonType=Password ExecutionTimeLimit=PT0S ping-owner-before=bf-spike set-localuser=ok ping-alive-after-rotate=True reregister-new-password=ok started-by=W6-a task-ran-as=<host>\bf-spike output-created=...
claim W6-a: hold: method=Start-ScheduledTask (AtStartup trigger as registered) state-before=Ready start=ok task-ran-as=<host>\bf-spike LastTaskResult=0x00000000 (0) LastRunTime=... state-after=Ready
claim W6-b: untested: not needed: W6-a produced the output file
claim W6-c: untested: not needed: W6-a produced the output file
claim ACL-second-account: hold: root-acl=protected=True [...] write-exit=0 probe-exists=True probe-owner=<host>\bf-spike ... bf-spike2-type-exit=1 bf-spike2-type-stderr=Access is denied. ...
w6 ts-log: was-enabled=False record=bf-spike-tslog.txt wevtutil-exit=0 enabled-now=True
w6 readback after the second registration: action=[C:\WINDOWS\system32\cmd.exe /d /c whoami > "%ProgramData%\Broodforge-spike\spike\task-out.txt" 2>&1] trigger=[MSFT_TaskBootTrigger Enabled=True StartBoundary=] principal=bf-spike/Password/Limited state=Ready Enabled=True AllowDemandStart=True DisallowStartIfOnBatteries=False ...
w6 battery: no Win32_Battery instance (no battery)
w6 event after W6-a: id=<n> time=... <the log's message naming "\bf-spike-test">
...
spike: done; next probe-unelevated.ps1 from a non-elevated PowerShell, then cleanup.ps1 elevated
claim ACL-owner-file: hold: elevated=False EnableLUA=1 open denied (UnauthorizedAccessException)
claim ACL-owner-list: hold: elevated=False EnableLUA=1 listing denied (UnauthorizedAccessException)
```

`W3a-env` refuted (the child sees your `USERPROFILE`, not the account's) is a result for P23, not
a failed run.

## When something is left behind

Whatever `run.ps1` created stays until `cleanup.ps1` removes it: the accounts `bf-spike` and
`bf-spike2`, their profiles under the profiles folder, the task `bf-spike-test`,
`%ProgramData%\Broodforge-spike\` (the engine copy is inside it), and the TaskScheduler log's enabled
state when `run.ps1` turned it on. Run `cleanup.ps1` again first; it prints
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

Run the last line only when `%ProgramData%\Broodforge-spike\bf-spike.marker` exists. Before it, and
only when `%ProgramData%\Broodforge-spike\bf-spike-tslog.txt` exists (`run.ps1` found the TaskScheduler
log disabled and enabled it), disable the log again and check that the second line prints `False`:

```
wevtutil sl Microsoft-Windows-TaskScheduler/Operational /e:false
(Get-WinEvent -ListLog Microsoft-Windows-TaskScheduler/Operational).IsEnabled
```

One trace stays by
design: registering a task with a password grants the account "Log on as a batch job", and the local
policy keeps that entry as an unresolved SID after the account is gone; it grants nothing to anyone.
