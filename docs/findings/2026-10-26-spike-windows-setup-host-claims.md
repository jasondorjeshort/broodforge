---
date: 2026-10-26
status: open
area: spike / setup host (Windows)
---

# Windows `setup host` claims W1, W3a, W4, W6 and the root ACL: all hold but W6's last step, which is open

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

The owner's run, 2026-10-26: `run.ps1` elevated (about a minute; a second `run.ps1` refused until
cleanup, as designed), `probe-unelevated.ps1` not elevated, `cleanup.ps1` elevated. Windows
PowerShell 5.1.26100.9711 (from the W6 line); engine `claude.exe` 2.1.310 (from the W4 line); the
Windows edition and build string were not pasted. Host names and profile roots in the lines below
are the scrubbed `<host>` and `<profiles>` the owner pasted.

## Result

Seven rows hold, W6 is refuted at its last step only, and the console-window row was not reported.
The evidence column is the claim line's text after the verdict.

| Claim | Holds when | Verdict | Evidence |
|---|---|---|---|
| W1 | registered with no password, `LogonType` `InteractiveToken`, `ExecutionTimeLimit` `PT0S` | hold | `registered=no password LogonType=InteractiveToken RunLevel=(absent) ExecutionTimeLimit=PT0S` |
| W3a | no profile before; after the child exits it exists on disk, not temporary, and `whoami` printed `bf-spike` | hold | `seclogon=Stopped/Manual profile-before=none exit=0 whoami=<host>\bf-spike profile-after=<profiles>\bf-spike on-disk=True status=0 loaded-now=False` |
| W3a-env | the child's `USERPROFILE` is the account's profile, not the caller's | hold | `USERPROFILE=<profiles>\bf-spike; APPDATA=<profiles>\bf-spike\AppData\Roaming; TEMP=<profiles>\bf-spike\AppData\Local\Temp; account profile=<profiles>\bf-spike` |
| W4 | the copy is byte-equal, `--version` prints `<x.y.z> (Claude Code)` and exits 0 as `bf-spike` | hold | `sha256-equal=True`, engine ACL protected (SYSTEM and Administrators FullControl, `bf-spike` ReadAndExecute), `start-exit=0 output=2.1.310 (Claude Code); exit=0 cfg-entries=0` |
| W6 | non-interactive host; every cmdlet ran; the ping outlived the rotation; the task ran as `bf-spike` on the new password | refuted at the last step (open) | `noninteractive-host=True ps=5.1.26100.9711 new-localuser=ok register-password=ok LogonType=Password ExecutionTimeLimit=PT0S ping-owner-before=bf-spike set-localuser=ok ping-alive-after-rotate=True reregister-new-password=ok task-ran-as=(no output file) LastTaskResult=267011` |
| W6 time limit | `ExecutionTimeLimit` as registered with `-User -Password` (in the W6 line) | hold | `ExecutionTimeLimit=PT0S` in the W6 line (`LogonType=Password`) |
| ACL-second-account | `bf-spike2` cannot read the file `bf-spike` wrote | hold | root ACL protected (SYSTEM, Administrators FullControl, `bf-spike` Modify); `write-exit=0`, probe file owned by `bf-spike`, inherited ACL; `bf-spike2` `type` exit 1 `Access is denied.`, `dir` exit 1 `File Not Found`, `read-content=False` |
| ACL-owner-file | the owner's non-elevated shell cannot open that file | hold | `elevated=False EnableLUA=1 open denied (UnauthorizedAccessException)` |
| ACL-owner-list | the owner's non-elevated shell cannot list the root | hold | `elevated=False EnableLUA=1 listing denied (UnauthorizedAccessException)` |
| Console window | the owner saw the W3a child's console on the desktop (reported by the owner) | not reported | the owner did not say whether a console window appeared in the first seconds |
| Cleanup | `cleanup: done, 0 left`, `Get-LocalUser bf-spike` an error, the root gone | hold | `cleanup: done, 0 left`; `Get-LocalUser bf-spike` not found; the directory gone |

### What W6 shows

Held: `New-LocalUser` and `Register-ScheduledTask -User -Password -Force` ran from a non-interactive
elevated host with the password built in the script (the registered `LogonType` reads `Password`,
`ExecutionTimeLimit` `PT0S`); `Set-LocalUser -Password` ran while a process started as the account
was running, and that process stayed alive and owned by `bf-spike` (`ping-alive-after-rotate=True`);
re-registering with the new password succeeded.

Failed: only the last step. `run.ps1` (the W6 block, lines 405-445) then ran `Start-ScheduledTask`
and waited 60 s for the task's output file; none appeared. `LastTaskResult` 267011 is `0x41303`,
`SCHED_S_TASK_HAS_NOT_RUN`: the task never started. A bad password or a missing batch-logon right
would be expected to give a different code, so those are not what the evidence points to.

The cause is not established. Candidates, none tested: the task's `AtStartup` trigger (the task has
no logon or time trigger, and `Start-ScheduledTask` may not start an `AtStartup` task registered
this way); `Start-ScheduledTask` called from a non-interactive elevated host; the action or its
output path as registered (not read back in this run); the task's settings or state (for example
disabled, or a start condition). Because the verdict reads one step that did not run, it says
nothing about whether the task would run as `bf-spike` on the new password.

Consequence for P23 until this closes: the password, rotation and re-registration parts of
`setup service` stand on the evidence above; "`Start-ScheduledTask` starts the service task" does
not, and P23 should not script it as proven.

## Next

1. A coder pass on the W6 last step: read back the registered task (action, trigger, settings,
   state) after the second `Register-ScheduledTask`, and try the start with a trigger that
   `Start-ScheduledTask` is known to honour, then with `schtasks /run`, from the same
   non-interactive elevated host, using disposable accounts and `cleanup.ps1` as before. Record the
   result as a new row or finding, not by editing the evidence above.
2. The owner says whether a console window appeared on the desktop during the W3a child (not
   reported yet); the row stays `not reported` until then.
3. When W6 closes and the console row is filled, set `status: resolved` and promote the durable
   facts where P23 puts them, before P23's brief.
