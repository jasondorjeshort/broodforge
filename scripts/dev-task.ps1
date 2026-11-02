# SPDX-License-Identifier: GPL-3.0-or-later
#
# The always-up task of a live Broodforge instance under the owner's own account on the PC: a
# Task Scheduler task `Broodforge-<instance>` that starts at machine boot (no logon needed), runs
# `bin\broodforge.exe --root <root> serve --discord live` through this script's `run` loop, and
# restarts the bot when it fails. The procedure is the run skill's section "2c. The always-up
# task" (.claude/skills/run/SKILL.md); the instance is 05-debug-interface-dev-instance.md's live
# form, dev-live. The dry-run `dev` instance has no task (that doc) and is refused here.
#
#   powershell -NoProfile -ExecutionPolicy Bypass -File scripts\dev-task.ps1 <command> [-Instance <name>]
#
# Commands:
#   install   Register the task (elevated shell only; prints the one-line elevated command
#             otherwise) and start it, unless a bot already answers on the instance.
#   redeploy  Build bin\, stop the bot with `stop`, `setup engine --yes` only when `setup --check
#             engine` disagrees (never the full setup line: it would touch the Discord
#             application), start the task, `status --wait --expect ok`. No elevation.
#   status    The task's registration and last run, the run loop's last log lines, the bot's
#             `status`. Exit 0 only when the task is installed and the bot answers.
#   remove    Stop the task's bot with `stop` and unregister the task (elevated shell only).
#   xml       Print the task definition install would register, checked by Task Scheduler's
#             own parser, registering nothing.
#   run       The task's action: the serve loop. Not for hand use, except as the loop's harness
#             on a scratch root: `run -Instance <scratch name> -Root <scratch root> -Discord
#             dry-run` (the one place -Discord dry-run is accepted).
#
# -Instance defaults to dev-live; `dev` is refused. -Root defaults to
# %LOCALAPPDATA%\Broodforge\<instance>. -Logon S4U (default: no stored password) or Password
# (the owner types the account password once, at install).
#
# Never stops a bot by PID or process name: only `broodforge stop` over the instance's pipe.

[CmdletBinding()]
param(
    [Parameter(Position = 0)][string]$Command = 'status',
    [string]$Instance = 'dev-live',
    [string]$Root = '',
    [string]$Discord = 'live',
    [string]$Logon = 'S4U'
)

Set-StrictMode -Version 2
$ErrorActionPreference = 'Stop'

# The run loop: wait between restarts, give up after this many failed starts in a row; a run
# that lasted at least $HealthyRun resets the count.
$RestartDelaySeconds = 60
$MaxRestarts = 99
$HealthyRun = New-TimeSpan -Minutes 10
# Task Scheduler's own restart, separate from the loop's: it only covers the loop failing to
# launch (an S4U logon refused at boot, powershell.exe missing), and stays small so a loop that
# gives up is not multiplied into thousands of Discord identifies.
$LaunchRestarts = 3

function Fail([string]$code, [string]$text) {
    [Console]::Error.WriteLine("error: ${code}: $text")
    exit 1
}

if ($Command -notin @('install', 'redeploy', 'status', 'remove', 'xml', 'run')) {
    Fail 'bad-args' "unknown command '$Command'; want install, redeploy, status, remove, xml or run"
}
# control.ValidInstanceName's rule: it names the pipe and the task.
if ($Instance -notmatch '^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$') {
    Fail 'bad-args' "instance '$Instance': want 1 to 64 of [A-Za-z0-9._-], starting with a letter or digit"
}
if ($Discord -notin @('dry-run', 'live')) { Fail 'bad-args' "-Discord '$Discord': want live" }
if ($Logon -notin @('S4U', 'Password')) { Fail 'bad-args' "-Logon '$Logon': want S4U or Password" }

$DevRoot = Join-Path $env:LOCALAPPDATA 'Broodforge\dev'
if (-not $Root) { $Root = Join-Path $env:LOCALAPPDATA "Broodforge\$Instance" }
$Root = $Root -replace '/', '\'
$RootFwd = $Root -replace '\\', '/'

# The dev instance is dry-run and has no task (05-debug-interface-dev-instance.md); its skill
# steps rely on no bot answering there. Dry-run is the scratch-root harness of `run` only.
if ($Instance -eq 'dev' -or $Root.TrimEnd('\') -ieq $DevRoot) {
    Fail 'dry-run-not-supported' "the dev instance has no always-up task; this script serves live instances (default dev-live)"
}
if ($Discord -eq 'dry-run' -and ($Command -ne 'run' -or -not $PSBoundParameters.ContainsKey('Root'))) {
    Fail 'dry-run-not-supported' "-Discord dry-run is only the run loop's harness on a scratch root (run -Root <scratch root>)"
}

$Repo = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$Exe = Join-Path $Repo 'bin\broodforge.exe'
$Script = Join-Path $Repo 'scripts\dev-task.ps1'
$TaskName = "Broodforge-$Instance"
$TaskLog = Join-Path $Root 'task.log'

# Every broodforge call below reads the root from the environment, as the run skill's step 0
# sets it; the service instance's key override never applies here.
$env:BROODFORGE_ROOT = $RootFwd
Remove-Item Env:BROODFORGE_KEY_FILE -ErrorAction SilentlyContinue

# --- helpers ---------------------------------------------------------------------------------

# Quote one argument for a Windows command line (CommandLineToArgvW rules).
function Format-Arg([string]$a) {
    if ($a -eq '') { return '""' }
    if ($a -notmatch '[\s"]') { return $a }
    $q = $a -replace '(\\*)"', '$1$1\"'
    $q = $q -replace '(\\+)$', '$1$1'
    return '"' + $q + '"'
}

# Run a program to completion with stdin closed; returns Code, Out, Err. No PowerShell stream
# wrapping of stderr (2>&1 on a native command makes error records in Windows PowerShell 5.1).
function Invoke-Native([string]$file, [string[]]$argv, [hashtable]$extraEnv = @{}) {
    $psi = New-Object System.Diagnostics.ProcessStartInfo
    $psi.FileName = $file
    $psi.Arguments = (($argv | ForEach-Object { Format-Arg $_ }) -join ' ')
    $psi.UseShellExecute = $false
    $psi.RedirectStandardOutput = $true
    $psi.RedirectStandardError = $true
    $psi.RedirectStandardInput = $true
    $psi.CreateNoWindow = $true
    $psi.WorkingDirectory = $Repo
    foreach ($k in $extraEnv.Keys) { $psi.EnvironmentVariables[$k] = $extraEnv[$k] }
    $p = [System.Diagnostics.Process]::Start($psi)
    $p.StandardInput.Close()
    $o = $p.StandardOutput.ReadToEndAsync()
    $e = $p.StandardError.ReadToEndAsync()
    $p.WaitForExit()
    return [pscustomobject]@{ Code = $p.ExitCode; Out = $o.Result; Err = $e.Result }
}

function Invoke-Bf([string[]]$argv) {
    if (-not (Test-Path -LiteralPath $Exe)) { Fail 'no-binary' "$Exe is missing; build it: CGO_ENABLED=0 go build -o bin/ ./cmd/broodforge" }
    return Invoke-Native $Exe $argv
}

# Print a command's output, leaving out setup's key line and spawn preflight's account line
# (the run skill: the key stays out of reports; the account line names the machine's user).
function Write-Result($r) {
    foreach ($text in @($r.Out, $r.Err)) {
        foreach ($line in ($text -split "`r?`n")) {
            if ($line -eq '') { continue }
            if ($line -like 'ok  key  *' -or $line -like 'ok  check  spawn preflight  ok  account  *') { continue }
            Write-Output $line
        }
    }
}

function Test-Elevated {
    $id = [Security.Principal.WindowsIdentity]::GetCurrent()
    return ([Security.Principal.WindowsPrincipal]$id).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

# The one line the owner pastes into a PowerShell window; it asks UAC for the owner's consent.
function Get-ElevatedLine([string]$cmd) {
    $extra = " -Instance $Instance"
    if ($Root -ne (Join-Path $env:LOCALAPPDATA "Broodforge\$Instance")) { $extra += " -Root ""$Root""" }
    if ($Logon -ne 'S4U') { $extra += " -Logon $Logon" }
    return "Start-Process powershell -Verb RunAs -ArgumentList '-NoProfile -ExecutionPolicy Bypass -NoExit -File ""$Script"" $cmd$extra'"
}

function Get-Task {
    return Get-ScheduledTask -TaskName $TaskName -TaskPath '\' -ErrorAction SilentlyContinue
}

# What answers on the instance's pipe: 'none', 'setup' (a `setup` run holds it: `status` prints
# `config: setup`) or 'bot'. A missing binary reads 'none', so the run loop counts it as a
# failed start instead of ending.
function Get-PipeState {
    if (-not (Test-Path -LiteralPath $Exe)) { return 'none' }
    $r = Invoke-Native $Exe @('status')
    if ($r.Code -ne 0) { return 'none' }
    if ($r.Out -match '(?m)^config: setup\s*$') { return 'setup' }
    return 'bot'
}

function Test-BotAnswers { return (Get-PipeState) -ne 'none' }

# Poll until nothing answers (true) or the timeout passes (false).
function Wait-BotGone([int]$seconds) {
    $deadline = (Get-Date).AddSeconds($seconds)
    while ((Get-Date) -lt $deadline) {
        if (-not (Test-BotAnswers)) { return $true }
        Start-Sleep -Milliseconds 500
    }
    return -not (Test-BotAnswers)
}

function Wait-TaskIdle([int]$seconds) {
    $deadline = (Get-Date).AddSeconds($seconds)
    while ((Get-Date) -lt $deadline) {
        $t = Get-Task
        if (-not $t -or $t.State -ne 'Running') { return $true }
        Start-Sleep -Milliseconds 500
    }
    return $false
}

# Stop the bot that answers on the instance, through its pipe only.
function Stop-Bot {
    $r = Invoke-Bf @('stop')
    Write-Result $r
    if ($r.Code -ne 0) { Fail 'stop-failed' "stop exited $($r.Code); the bot is left as it is" }
    if (-not (Wait-BotGone 40)) { Fail 'stop-failed' "the bot still answers 40 s after stop" }
}

function Write-TaskLogTail([int]$n) {
    if (Test-Path -LiteralPath $TaskLog) {
        Get-Content -LiteralPath $TaskLog -Tail $n | ForEach-Object { Write-Output "log   $_" }
    }
}

# The installed task's -Discord and -Root, read back from its action.
function Get-TaskTarget($t) {
    $argText = [string](@($t.Actions)[0].Arguments)
    $mode = ''; $root = ''
    if ($argText -match '-Discord\s+(\S+)') { $mode = $Matches[1] }
    if ($argText -match '-Root\s+"([^"]+)"') { $root = $Matches[1] }
    return [pscustomobject]@{ Mode = $mode; Root = $root }
}

# --- the task definition ---------------------------------------------------------------------

function Get-TaskXml([string]$sid) {
    $e = { param($s) [Security.SecurityElement]::Escape($s) }
    $ps = Join-Path $env:SystemRoot 'System32\WindowsPowerShell\v1.0\powershell.exe'
    $taskArgs = "-NoProfile -NonInteractive -ExecutionPolicy Bypass -WindowStyle Hidden -File ""$Script"" run -Instance $Instance -Root ""$Root"" -Discord $Discord"
    if ($Logon -eq 'S4U') { $lt = 'S4U' } else { $lt = 'Password' }
    return @"
<Task version="1.4" xmlns="http://schemas.microsoft.com/windows/2004/02/mit/task">
  <RegistrationInfo>
    <Description>$(& $e "Broodforge instance $Instance, always up: serve --discord $Discord on $Root, started at boot by scripts\dev-task.ps1 run.")</Description>
  </RegistrationInfo>
  <Triggers>
    <BootTrigger>
      <Enabled>true</Enabled>
    </BootTrigger>
  </Triggers>
  <Principals>
    <Principal id="Author">
      <UserId>$sid</UserId>
      <LogonType>$lt</LogonType>
      <RunLevel>LeastPrivilege</RunLevel>
    </Principal>
  </Principals>
  <Settings>
    <MultipleInstancesPolicy>IgnoreNew</MultipleInstancesPolicy>
    <DisallowStartIfOnBatteries>false</DisallowStartIfOnBatteries>
    <StopIfGoingOnBatteries>false</StopIfGoingOnBatteries>
    <AllowHardTerminate>true</AllowHardTerminate>
    <StartWhenAvailable>true</StartWhenAvailable>
    <RunOnlyIfNetworkAvailable>false</RunOnlyIfNetworkAvailable>
    <IdleSettings>
      <StopOnIdleEnd>false</StopOnIdleEnd>
      <RestartOnIdle>false</RestartOnIdle>
    </IdleSettings>
    <AllowStartOnDemand>true</AllowStartOnDemand>
    <Enabled>true</Enabled>
    <Hidden>true</Hidden>
    <RunOnlyIfIdle>false</RunOnlyIfIdle>
    <WakeToRun>false</WakeToRun>
    <ExecutionTimeLimit>PT0S</ExecutionTimeLimit>
    <Priority>7</Priority>
    <RestartOnFailure>
      <Interval>PT1M</Interval>
      <Count>$LaunchRestarts</Count>
    </RestartOnFailure>
  </Settings>
  <Actions Context="Author">
    <Exec>
      <Command>$(& $e $ps)</Command>
      <Arguments>$(& $e $taskArgs)</Arguments>
      <WorkingDirectory>$(& $e $Repo)</WorkingDirectory>
    </Exec>
  </Actions>
</Task>
"@
}

# SYSTEM and Administrators full control; the owner read and execute, so `redeploy` starts the
# task from a normal shell. Changing or deleting the task stays elevated.
function Get-TaskSddl([string]$sid) { return "D:(A;;FA;;;SY)(A;;FA;;;BA)(A;;GRGX;;;$sid)" }

function Get-SchedulerFolder {
    $svc = New-Object -ComObject Schedule.Service
    $svc.Connect()
    return @($svc, $svc.GetFolder('\'))
}

# --- commands --------------------------------------------------------------------------------

function Invoke-Xml {
    $sid = [Security.Principal.WindowsIdentity]::GetCurrent().User.Value
    $svc, $null = Get-SchedulerFolder
    $def = $svc.NewTask(0)
    try { $def.XmlText = (Get-TaskXml $sid) } catch { Fail 'bad-xml' $_.Exception.Message }
    Write-Output $def.XmlText
    Write-Output "sddl  $(Get-TaskSddl $sid)"
}

function Invoke-Status {
    $t = Get-Task
    $code = 0
    if (-not $t) {
        Write-Output "task  $TaskName  not-installed"
        $code = 1
    } else {
        $info = Get-ScheduledTaskInfo -TaskName $TaskName -TaskPath '\'
        $action = @($t.Actions)[0]
        Write-Output ("task  $TaskName  {0}  {1}  last-run {2}  last-result 0x{3:X}" -f $t.State, $t.Principal.LogonType, $info.LastRunTime, $info.LastTaskResult)
        Write-Output "action  $($action.Execute) $($action.Arguments)"
        Write-TaskLogTail 3
    }
    $r = Invoke-Bf @('status')
    Write-Result $r
    if ($r.Code -ne 0) { $code = 1 }
    exit $code
}

function Invoke-Install {
    if (-not (Test-Path -LiteralPath $Exe)) { Fail 'no-binary' "$Exe is missing; build it first: CGO_ENABLED=0 go build -o bin/ ./cmd/broodforge" }
    if (-not (Test-Path -LiteralPath (Join-Path $Root 'config'))) { Fail 'no-config' "$Root has no config; make the live root first with setup" }
    if (-not (Test-Elevated)) {
        Write-Output "needs-elevation  registering a boot task needs an elevated shell; paste this into a PowerShell window as the owner (UAC asks once):"
        Write-Output (Get-ElevatedLine 'install')
        exit 1
    }
    $id = [Security.Principal.WindowsIdentity]::GetCurrent()
    $sid = $id.User.Value
    Write-Output "user  $($id.Name)  (the task runs as this account; it must be the owner's own)"

    # Another task that starts the same root would race this one for the pipe at boot.
    foreach ($other in @(Get-ScheduledTask -ErrorAction SilentlyContinue)) {
        if (-not $other -or $other.TaskName -eq $TaskName) { continue }
        foreach ($a in @($other.Actions)) {
            $argText = ''
            if ($a.PSObject.Properties['Arguments'] -and $a.Arguments) { $argText = [string]$a.Arguments }
            if ($argText -match ('Broodforge[\\/]' + [regex]::Escape($Instance) + '(\s|"|$)')) {
                Write-Output "warn  task $($other.TaskPath)$($other.TaskName) also starts $Instance; disable it: Disable-ScheduledTask -TaskName '$($other.TaskName)'"
            }
        }
    }

    $svc, $folder = Get-SchedulerFolder
    $xml = Get-TaskXml $sid
    $sddl = Get-TaskSddl $sid
    if ($Logon -eq 'S4U') {
        $null = $folder.RegisterTask($TaskName, $xml, 6, $id.Name, $null, 2, $sddl)
    } else {
        $sec = Read-Host -AsSecureString "Password of $($id.Name) (kept by Task Scheduler only)"
        $plain = (New-Object System.Net.NetworkCredential('', $sec)).Password
        $null = $folder.RegisterTask($TaskName, $xml, 6, $id.Name, $plain, 1, $sddl)
        $plain = $null
    }
    $t = Get-Task
    Write-Output ("ok  task  $TaskName  {0}  at boot  no time limit  launch restarts {1}x every 1 min  serve restarts up to {2} in a row every {3} s" -f `
            $t.Principal.LogonType, $LaunchRestarts, $MaxRestarts, $RestartDelaySeconds)

    if (Test-BotAnswers) {
        Write-Output "skip  start  a bot already answers on $Instance (started by hand); the task is not started beside it"
        Write-Output "next  from a normal shell: powershell -NoProfile -ExecutionPolicy Bypass -File ""$Script"" redeploy -Instance $Instance"
        exit 0
    }
    Start-ScheduledTask -TaskName $TaskName -TaskPath '\'
    $r = Invoke-Bf @('status', '--wait', '30', '--expect', 'ok')
    Write-Result $r
    if ($r.Code -ne 0) {
        Write-TaskLogTail 5
        if ($Logon -eq 'S4U') { Write-Output "hint  if the log shows the bot cannot start under S4U, rerun install with -Logon Password" }
        exit 1
    }
    Write-Output "ok  start  $TaskName"
    Write-Output "next  check from a normal (not elevated) shell: powershell -NoProfile -ExecutionPolicy Bypass -File ""$Script"" status -Instance $Instance"
}

function Invoke-Redeploy {
    $t = Get-Task
    if (-not $t) { Fail 'not-installed' "task $TaskName is not installed; the owner runs: $(Get-ElevatedLine 'install')" }
    # The task must serve what this redeploy refreshes: the same mode and root.
    $target = Get-TaskTarget $t
    if ($target.Mode -ne $Discord -or $target.Root -ine $Root) {
        Fail 'task-mismatch' "task $TaskName runs -Discord '$($target.Mode)' -Root '$($target.Root)', not '$Discord' '$Root'; the owner reinstalls it"
    }

    # Build first: go replaces bin\broodforge.exe under a running bot, and a failed build leaves
    # the bot up.
    $go = Get-Command go -CommandType Application -ErrorAction SilentlyContinue | Select-Object -First 1
    if (-not $go) { Fail 'no-go' 'go is not on PATH (docs/reference/go-on-windows.md)' }
    $r = Invoke-Native $go.Source @('build', '-o', 'bin/', './cmd/broodforge') @{ CGO_ENABLED = '0' }
    Write-Result $r
    if ($r.Code -ne 0) { Fail 'build-failed' "go build exited $($r.Code); the bot is left as it is" }
    Write-Output "ok  build  bin/broodforge.exe"

    if (Test-BotAnswers) { Stop-Bot }
    # A clean stop ends the loop at once. A task still running with no bot is a loop between
    # restarts: it starts the new build itself within $RestartDelaySeconds s.
    $loopWaiting = -not (Wait-TaskIdle 30)

    # Never the full setup line on a live root (it would touch the Discord application and
    # token); only the engine step, and only when it disagrees. `--check` reports the
    # disagreement as `fix  engine  config has version ...` and `error: check-failed: engine`.
    $r = Invoke-Bf @('setup', '--check', 'engine')
    if ($r.Code -ne 0 -and ($r.Out + $r.Err) -match '(?m)^fix  engine  |error: mismatch-engine') {
        Write-Result $r
        $r = Invoke-Bf @('setup', 'engine', '--yes', '--wait', '1')
        Write-Result $r
        if ($r.Code -ne 0) { Fail 'setup-failed' "setup engine exited $($r.Code); the bot is not started" }
    } elseif ($r.Code -ne 0) {
        Write-Result $r
        Fail 'setup-failed' "setup --check engine exited $($r.Code); the bot is not started"
    }

    $wait = 30
    if ($loopWaiting) {
        Write-Output "note  task $TaskName is between restarts (see $TaskLog); its loop starts the new build itself"
        $wait = $RestartDelaySeconds + 30
    } else {
        try { Start-ScheduledTask -TaskName $TaskName -TaskPath '\' }
        catch { Fail 'start-failed' "$($_.Exception.Message.Trim()); from an elevated shell: Start-ScheduledTask -TaskName $TaskName" }
    }
    $r = Invoke-Bf @('status', '--wait', [string]$wait, '--expect', 'ok')
    Write-Result $r
    if ($r.Code -ne 0) {
        Write-TaskLogTail 5
        exit 1
    }
    Write-Output "ok  start  $TaskName"
}

function Invoke-Remove {
    $t = Get-Task
    if (-not $t) {
        Write-Output "task  $TaskName  not-installed"
        exit 0
    }
    if (-not (Test-Elevated)) {
        Write-Output "needs-elevation  removing the task needs an elevated shell; paste this into a PowerShell window as the owner:"
        Write-Output (Get-ElevatedLine 'remove')
        exit 1
    }
    # Only the task's own bot is stopped: a bot that answers while the task is not running was
    # started by hand and is left alone.
    if ($t.State -eq 'Running' -and (Test-BotAnswers)) { Stop-Bot }
    if (-not (Wait-TaskIdle 30)) {
        Fail 'task-busy' ("task $TaskName still runs with no bot answering: its loop is between restarts (see $TaskLog). " +
            "End the loop in this elevated shell with Stop-ScheduledTask -TaskName $TaskName, then run remove again")
    }
    Unregister-ScheduledTask -TaskName $TaskName -TaskPath '\' -Confirm:$false
    Write-Output "ok  removed  $TaskName"
}

# The task's action. One bot at a time: when a bot already answers, the loop leaves it alone
# and ends; while a `setup` run holds the pipe it waits. A clean exit (0, `stop`) ends the task;
# any other exit, or a start that throws (a missing binary), restarts after
# $RestartDelaySeconds, at most $MaxRestarts failed starts in a row. A start refused because the
# owner's Claude Code updated (start-engine, engine.version) gets `setup engine --yes` once per
# run, the fix the run skill applies, then an immediate retry.
function Invoke-Run {
    function Write-Log([string]$m) {
        try { [IO.File]::AppendAllText($TaskLog, ("{0}  {1}`n" -f (Get-Date -Format 'yyyy-MM-ddTHH:mm:ss'), $m)) } catch { }
    }
    $pidFile = Join-Path $Root 'serve.pid'
    $fails = 0
    $engineFixed = $false
    $setupLogged = $false
    while ($true) {
        $began = Get-Date
        try {
            $state = Get-PipeState
            if ($state -eq 'setup') {
                if (-not $setupLogged) { Write-Log 'a setup run holds the pipe; waiting'; $setupLogged = $true }
                Start-Sleep -Seconds 5
                continue
            }
            $setupLogged = $false
            if ($state -eq 'bot') {
                Write-Log "a bot already answers on $Instance; the task leaves it alone and ends"
                exit 0
            }
            # A retry within the same second gets its own pair rather than overwriting the last.
            $stamp = Get-Date -Format 'yyyyMMdd-HHmmss'
            $n = 1
            while (Test-Path -LiteralPath (Join-Path $Root "serve-$stamp.log")) { $n++; $stamp = (Get-Date -Format 'yyyyMMdd-HHmmss') + "-$n" }
            $out = Join-Path $Root "serve-$stamp.log"
            $err = Join-Path $Root "serve-$stamp.err"
            $argLine = ((@('--root', $Root, 'serve', '--discord', $Discord) | ForEach-Object { Format-Arg $_ }) -join ' ')
            $p = Start-Process -FilePath $Exe -ArgumentList $argLine -WorkingDirectory $Repo -WindowStyle Hidden `
                -RedirectStandardOutput $out -RedirectStandardError $err -PassThru
            $null = $p.Handle # keeps ExitCode readable after the exit
            [IO.File]::WriteAllText($pidFile, [string]$p.Id)
            Write-Log "started  pid $($p.Id)  serve-$stamp.log"
            $p.WaitForExit()
            $code = $p.ExitCode
            Write-Log ("exited  {0}  after {1:N0} s" -f $code, ((Get-Date) - $began).TotalSeconds)
            if ((Test-Path -LiteralPath $pidFile) -and ((Get-Content -LiteralPath $pidFile -Raw).Trim() -eq [string]$p.Id)) {
                Remove-Item -LiteralPath $pidFile
            }
            if ($code -eq 0) {
                Write-Log 'clean stop; the task ends'
                exit 0
            }
            $text = ''
            foreach ($f in @($out, $err)) { if (Test-Path -LiteralPath $f) { $text += (Get-Content -LiteralPath $f -Raw) } }
            if (-not $engineFixed -and $text -match 'start-engine: engine\.version is') {
                $engineFixed = $true
                $r = Invoke-Native $Exe @('setup', 'engine', '--yes', '--wait', '1')
                Write-Log ("setup engine --yes  exit {0}  {1}" -f $r.Code, (($r.Out + $r.Err) -replace "`r?`n", ' ').Trim())
                if ($r.Code -eq 0) { continue }
            }
        } catch {
            Write-Log ("start failed  {0}" -f $_.Exception.Message.Trim())
        }
        if (((Get-Date) - $began) -ge $HealthyRun) { $fails = 0 }
        $fails++
        if ($fails -ge $MaxRestarts) {
            Write-Log "$fails failed starts in a row; giving up"
            exit 1
        }
        Write-Log "restart in $RestartDelaySeconds s ($fails of $MaxRestarts)"
        Start-Sleep -Seconds $RestartDelaySeconds
    }
}

switch ($Command) {
    'install' { Invoke-Install }
    'redeploy' { Invoke-Redeploy }
    'status' { Invoke-Status }
    'remove' { Invoke-Remove }
    'xml' { Invoke-Xml }
    'run' { Invoke-Run }
}
