**Create the local standard account `broodforge` on the PC, log in once, install Claude Code for it, and create the Task Scheduler task that runs Broodforge whether the user is logged on or not.** It waits on the owner because creating an account needs elevation and the first login and the task password are theirs to type. The reasoning is ADR 0022 (Decision, Windows); the restore side of the service step is `docs/design/decisions/0021-backup-and-restore.md`.

Status: waiting by construction. Nothing is created yet and the PC was not inspected when this was filed, so no check was made. The task action runs a Broodforge command that does not exist yet, `[no-check: not yet built]`.

Runs in an elevated PowerShell as the owner. True first: nothing else; development and scratch runs keep using the owner's own account and need none of this.

```
$pw = Read-Host -AsSecureString "Password for the broodforge account"
New-LocalUser -Name broodforge -Password $pw -PasswordNeverExpires -UserMayNotChangePassword -Description "Broodforge instance account"
Add-LocalGroupMember -Group Users -Member broodforge
```

One interactive login, once: sign out, sign in as `broodforge`, let the profile be created, then in a PowerShell as `broodforge` install Claude Code by the official Windows installer (`docs/reference/claude-code-on-windows.md`) and confirm:

```
claude --version
```

Do not run `claude` login for the instance here; credentials are delivered by `credential login <id>` (not yet built) later. Back as the owner, create the task. `<broodforge command>` is the instance start command, named by the design and not yet built:

```
$action = New-ScheduledTaskAction -Execute "<broodforge command>" -Argument "start"
$trigger = New-ScheduledTaskTrigger -AtStartup
$settings = New-ScheduledTaskSettingsSet -ExecutionTimeLimit ([TimeSpan]::Zero) -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -RestartCount 3 -RestartInterval (New-TimeSpan -Minutes 1)
$cred = Get-Credential "$env:COMPUTERNAME\broodforge"
Register-ScheduledTask -TaskName "Broodforge" -Action $action -Trigger $trigger -Settings $settings -User $cred.UserName -Password $cred.GetNetworkCredential().Password -RunLevel Limited
```

The settings matter: the defaults stop a task after 72 hours (`ExecutionTimeLimit PT72H`) and on
battery, which would kill the bot. `-User` with `-Password` is the parameter set that stores the
password and makes the task run whether the user is logged on or not; `-Principal` is a different
set and cannot be combined with them.

Check it took:

```
Get-LocalUser broodforge
Get-ScheduledTask -TaskName Broodforge | Select-Object -ExpandProperty Principal
icacls "$env:USERPROFILE" | Select-String broodforge
```

Expect the account enabled, `LogonType` `Password` (run whether logged on or not), and no `broodforge` entry on the owner's profile. Allow 30 minutes.

When done: delete this file. Spike C37 of ADR 0022 later proves a turn runs from the task with no interactive session; its result is recorded in the ADR.
