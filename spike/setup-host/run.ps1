# SPDX-License-Identifier: GPL-3.0-or-later
#
# P20 spike (docs/design/08-implementation-plan-setup.md): claims W1, W3a, W4, W6 and the root's ACL,
# against two disposable local accounts, bf-spike and bf-spike2, on the owner's PC. README.md is the
# runbook. Run from an elevated PowerShell as
#   powershell.exe -NoProfile -NonInteractive -ExecutionPolicy Bypass -File .\run.ps1 -EnginePath <claude.exe>
# It creates and changes nothing outside the two accounts, their profiles, the task bf-spike-test and
# %ProgramData%\Broodforge-spike\ (refused when that directory already exists). It removes nothing: it
# leaves everything for probe-unelevated.ps1, and cleanup.ps1 removes it.
#
# Output: one "claim <id>: hold|refuted|untested: <evidence>" line per claim, then one "spike:" line;
# a refusal is a single "spike: refused: <reason>" line. Evidence is scrubbed (~, <profiles>,
# <host>, <owner>, %ProgramData%). The generated passwords live in this process's memory only and
# are never printed, written to a file or passed on a command line.

[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string] $EnginePath
)

Set-StrictMode -Version 2.0
$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'

$Account    = 'bf-spike'
$Account2   = 'bf-spike2'
$TaskName   = 'bf-spike-test'
$Root       = Join-Path $env:ProgramData 'Broodforge-spike'
$Marker     = Join-Path $Root 'bf-spike.marker'
$MarkerText = 'bf-spike spike root (spike/setup-host/run.ps1); cleanup.ps1 removes this directory while this file is here'
$EngineDir  = Join-Path $Root 'engine'
$EngineExe  = Join-Path $EngineDir 'claude.exe'
$Work       = Join-Path $Root 'spike'
$Probe      = Join-Path $Root 'bf-spike-acl-probe.txt'
$ProbeText  = 'bf-spike wrote this'
$Cmd        = Join-Path $env:SystemRoot 'System32\cmd.exe'
$Ping       = Join-Path $env:SystemRoot 'System32\PING.EXE'
$SidSystem  = New-Object System.Security.Principal.SecurityIdentifier 'S-1-5-18'
$SidAdmins  = New-Object System.Security.Principal.SecurityIdentifier 'S-1-5-32-544'
$ProfilesDir = [Environment]::ExpandEnvironmentVariables(
    (Get-ItemProperty 'HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\ProfileList').ProfilesDirectory)

$Claims = [ordered]@{}
foreach ($id in 'W1', 'W3a', 'W3a-env', 'W4', 'W6', 'ACL-second-account') {
    $Claims[$id] = @('untested', 'not reached')
}

function Set-Claim([string] $Id, [string] $Verdict, [string] $Evidence) {
    $Claims[$Id] = @($Verdict, $Evidence)
}

# Scrubs a line for the finding (docs rules, Personal data): no home path, host or owner name.
function Format-Evidence([string] $Text) {
    if ($null -eq $Text) { return '' }
    $t = ($Text -replace '\s*[\r\n]+\s*', '; ').Trim([char[]] @(' ', ';'))
    $t = $t -ireplace [regex]::Escape($env:USERPROFILE), '~'
    $t = $t -ireplace [regex]::Escape($ProfilesDir), '<profiles>'
    $t = $t -ireplace [regex]::Escape($env:ProgramData), '%ProgramData%'
    $t = $t -ireplace ('\b' + [regex]::Escape($env:COMPUTERNAME) + '\b'), '<host>'
    if ($env:USERNAME -notlike 'bf-spike*') {
        $t = $t -ireplace ('\b' + [regex]::Escape($env:USERNAME) + '\b'), '<owner>'
    }
    return $t
}

function Join-Evidence([System.Collections.IDictionary] $D) {
    (@($D.GetEnumerator() | ForEach-Object { '{0}={1}' -f $_.Key, $_.Value })) -join ' '
}

function Write-Claims {
    foreach ($k in $Claims.Keys) {
        Write-Output ('claim {0}: {1}: {2}' -f $k, $Claims[$k][0], (Format-Evidence $Claims[$k][1]))
    }
}

function Stop-Spike([string] $Reason) {
    Write-Output ('spike: refused: ' + (Format-Evidence $Reason))
    exit 2
}

function Test-Elevated {
    $id = [Security.Principal.WindowsIdentity]::GetCurrent()
    (New-Object Security.Principal.WindowsPrincipal $id).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

# 32 characters from a CSPRNG, one of each class first, as a SecureString built in memory.
function New-SpikePassword {
    $sets = @('ABCDEFGHJKLMNPQRSTUVWXYZ', 'abcdefghijkmnopqrstuvwxyz', '23456789', '-_.+!#*')
    $all = -join $sets
    $bytes = New-Object byte[] 32
    $rng = [System.Security.Cryptography.RandomNumberGenerator]::Create()
    $rng.GetBytes($bytes)
    $rng.Dispose()
    $pw = New-Object System.Security.SecureString
    for ($i = 0; $i -lt 32; $i++) {
        $set = if ($i -lt $sets.Count) { $sets[$i] } else { $all }
        $pw.AppendChar($set[$bytes[$i] % $set.Length])
    }
    $pw.MakeReadOnly()
    return $pw
}

function New-SpikeCredential([string] $Name, [securestring] $Password) {
    New-Object System.Management.Automation.PSCredential ("$env:COMPUTERNAME\$Name", $Password)
}

function Add-ToUsersGroup($User) {
    try { Add-LocalGroupMember -SID 'S-1-5-32-545' -Member $User }
    catch { if ($_.FullyQualifiedErrorId -notlike 'MemberExists*') { throw } }
}

# Inheritance off, inherited entries dropped, then SYSTEM and Administrators full and $Sid at $Rights.
function Set-SpikeAcl([string] $Path, $Sid, [string] $Rights) {
    $acl = Get-Acl -LiteralPath $Path
    $acl.SetAccessRuleProtection($true, $false)
    foreach ($r in @($acl.GetAccessRules($true, $false, [System.Security.Principal.SecurityIdentifier]))) {
        [void] $acl.RemoveAccessRuleSpecific($r)
    }
    $inherit = [System.Security.AccessControl.InheritanceFlags] 'ContainerInherit, ObjectInherit'
    $prop = [System.Security.AccessControl.PropagationFlags]::None
    $allow = [System.Security.AccessControl.AccessControlType]::Allow
    foreach ($pair in @(@($SidSystem, 'FullControl'), @($SidAdmins, 'FullControl'), @($Sid, $Rights))) {
        $acl.AddAccessRule((New-Object System.Security.AccessControl.FileSystemAccessRule (
            $pair[0], [System.Security.AccessControl.FileSystemRights] $pair[1], $inherit, $prop, $allow)))
    }
    Set-Acl -LiteralPath $Path -AclObject $acl
}

function Get-AclSummary([string] $Path) {
    $acl = Get-Acl -LiteralPath $Path
    $rules = @($acl.Access | ForEach-Object {
        '{0}:{1}{2}' -f $_.IdentityReference, $_.FileSystemRights, $(if ($_.IsInherited) { '(inherited)' } else { '' })
    })
    'protected={0} [{1}]' -f $acl.AreAccessRulesProtected, ($rules -join ', ')
}

function Get-SpikeProfile($Sid) {
    Get-CimInstance -ClassName Win32_UserProfile -Filter ("SID='{0}'" -f $Sid.Value)
}

function Read-Out([string] $Path) {
    if (-not (Test-Path -LiteralPath $Path)) { return '(no output file)' }
    $t = Get-Content -LiteralPath $Path -Raw
    if ($null -eq $t) { return '(empty)' }
    return $t.Trim()
}

function Write-Batch([string] $Path, [string] $Text) {
    [System.IO.File]::WriteAllText($Path, ($Text -replace "`r?`n", "`r`n"), [System.Text.Encoding]::ASCII)
}

# cmd.exe as another account through Start-Process -Credential (CreateProcessWithLogonW), -Wait.
function Invoke-As {
    param([pscredential] $Credential, [string[]] $Arguments, [switch] $WithProfile, [string] $StdOut, [string] $StdErr)
    $sp = @{
        FilePath = $Cmd; ArgumentList = $Arguments; Credential = $Credential
        WorkingDirectory = $env:SystemRoot; Wait = $true; PassThru = $true
    }
    if ($WithProfile) { $sp['LoadUserProfile'] = $true }
    if ($StdOut) { $sp['RedirectStandardOutput'] = $StdOut; $sp['RedirectStandardError'] = $StdErr }
    $p = Start-Process @sp
    try { return [string] $p.ExitCode } catch { return 'unknown' }
}

# Invoke-As with stdout and stderr captured by this (elevated) process, for an account that cannot
# write in the root; when redirection under -Credential fails, the exit code alone.
function Invoke-AsCaptured([pscredential] $Credential, [string[]] $Arguments, [string] $Name) {
    $o = Join-Path $Work "$Name-out.txt"
    $e = Join-Path $Work "$Name-err.txt"
    try {
        $code = Invoke-As -Credential $Credential -Arguments $Arguments -StdOut $o -StdErr $e
        return @($code, (Read-Out $o), (Read-Out $e))
    } catch {
        $why = 'redirect failed: ' + $_.Exception.Message
        $code = Invoke-As -Credential $Credential -Arguments $Arguments
        return @($code, '(not captured)', $why)
    }
}

function Get-TaskXmlValue([xml] $Xml, [string] $Name) {
    $n = $Xml.GetElementsByTagName($Name)
    if ($n.Count -gt 0) { return $n[0].InnerText }
    return '(absent)'
}

function Get-ProcessOwner([int] $ProcessId) {
    $p = Get-CimInstance -ClassName Win32_Process -Filter ("ProcessId={0}" -f $ProcessId)
    if ($null -eq $p) { return $null }
    $o = Invoke-CimMethod -InputObject $p -MethodName GetOwner
    if ($o.ReturnValue -ne 0) { return '(unknown)' }
    return $o.User
}

# ---- preflight: refuse anything left over, and any directory that is not the spike's ----

if (-not (Test-Elevated)) { Stop-Spike 'not elevated; run from an elevated PowerShell' }
if (-not (Test-Path -LiteralPath $EnginePath -PathType Leaf)) { Stop-Spike "no engine at $EnginePath" }
foreach ($n in $Account, $Account2) {
    if (Get-LocalUser -Name $n -ErrorAction SilentlyContinue) { Stop-Spike "account $n exists; run cleanup.ps1 first" }
}
if (Get-ScheduledTask -TaskPath '\' -TaskName $TaskName -ErrorAction SilentlyContinue) {
    Stop-Spike "task $TaskName exists; run cleanup.ps1 first"
}
if (Test-Path -LiteralPath $Root) {
    if (Test-Path -LiteralPath $Marker) { Stop-Spike "$Root is left from an earlier run; run cleanup.ps1 first" }
    Stop-Spike "$Root exists without the spike's marker (a real instance's directory?); the spike does not touch it"
}
$stale = @(Get-CimInstance -ClassName Win32_UserProfile | Where-Object {
    $_.LocalPath -and ((Split-Path -Leaf $_.LocalPath) -match '^bf-spike2?(\..+)?$') })
if ($stale.Count -gt 0) { Stop-Spike 'a bf-spike profile folder is left over; run cleanup.ps1 first' }

$args0 = [Environment]::GetCommandLineArgs()
$nonInteractive = @($args0 | Where-Object { $_ -match '^-noni' }).Count -gt 0
$w6 = [ordered]@{ 'noninteractive-host' = $nonInteractive; ps = $PSVersionTable.PSVersion.ToString() }
$userId = "$env:COMPUTERNAME\$Account"
$pw1 = New-SpikePassword
$pw2 = New-SpikePassword
$pwB = New-SpikePassword
$cred1 = New-SpikeCredential $Account $pw1
$credB = New-SpikeCredential $Account2 $pwB

# ---- 1. accounts (W6: New-LocalUser with an in-memory SecureString, no prompt) ----

try {
    $user = New-LocalUser -Name $Account -Password $pw1 -PasswordNeverExpires -UserMayNotChangePassword `
        -AccountNeverExpires -Description 'Broodforge P20 spike; cleanup.ps1 removes it'
    Add-ToUsersGroup $user
    $user2 = New-LocalUser -Name $Account2 -Password $pwB -PasswordNeverExpires -UserMayNotChangePassword `
        -AccountNeverExpires -Description 'Broodforge P20 spike; cleanup.ps1 removes it'
    Add-ToUsersGroup $user2
    $sid1 = $user.SID
    $w6['new-localuser'] = 'ok'
} catch {
    $w6['new-localuser'] = 'failed: ' + $_.Exception.Message
    Set-Claim 'W6' 'refuted' (Join-Evidence $w6)
    Write-Claims
    Write-Output 'spike: stopped after the accounts; run cleanup.ps1'
    exit 1
}

# ---- 2. the root, its ACL, the engine copy and the batch files ----

try {
    New-Item -ItemType Directory -Path $Root | Out-Null
    [System.IO.File]::WriteAllText($Marker, $MarkerText)
    Set-SpikeAcl $Root $sid1 'Modify'
    New-Item -ItemType Directory -Path $Work | Out-Null
    New-Item -ItemType Directory -Path $EngineDir | Out-Null
    Set-SpikeAcl $EngineDir $sid1 'ReadAndExecute'
    Copy-Item -LiteralPath $EnginePath -Destination $EngineExe
    $rootAcl = Get-AclSummary $Root
    $engineAcl = Get-AclSummary $EngineDir
} catch {
    $msg = 'root setup failed: ' + $_.Exception.Message
    foreach ($id in @($Claims.Keys)) { if ($id -ne 'W6') { Set-Claim $id 'untested' $msg } }
    Set-Claim 'W6' 'untested' ((Join-Evidence $w6) + ' ' + $msg)
    Write-Claims
    Write-Output 'spike: stopped after the root; run cleanup.ps1'
    exit 1
}

Write-Batch (Join-Path $Work 'w3a.cmd') @'
@echo off
whoami > "%~dp0w3a-out.txt" 2>&1
>> "%~dp0w3a-out.txt" echo USERPROFILE=%USERPROFILE%
>> "%~dp0w3a-out.txt" echo APPDATA=%APPDATA%
>> "%~dp0w3a-out.txt" echo TEMP=%TEMP%
'@

Write-Batch (Join-Path $Work 'acl-write.cmd') (@'
@echo off
> "__PROBE__" echo __TEXT__
>> "%~dp0acl-write-out.txt" echo exit=%ERRORLEVEL%
'@).Replace('__PROBE__', $Probe).Replace('__TEXT__', $ProbeText)

# ---- 3. W3a: Start-Process -Credential -LoadUserProfile -Wait creates the profile ----

$profilePath = $null
$w3a = [ordered]@{}
try { $s = Get-Service -Name seclogon; $w3a['seclogon'] = '{0}/{1}' -f $s.Status, $s.StartType } catch { $w3a['seclogon'] = 'unknown' }
$before = Get-SpikeProfile $sid1
$w3a['profile-before'] = if ($null -eq $before) { 'none' } else { $before.LocalPath }
try {
    $w3a['exit'] = Invoke-As -Credential $cred1 -Arguments '/d', '/c', ('"{0}"' -f (Join-Path $Work 'w3a.cmd')) -WithProfile
    $out = Read-Out (Join-Path $Work 'w3a-out.txt')
    $lines = @($out -split "`r?`n")
    $whoami = $lines[0].Trim()
    $childProfile = (@($lines | Where-Object { $_ -like 'USERPROFILE=*' }) + @('USERPROFILE=')).Item(0).Substring(12).Trim()
    $after = Get-SpikeProfile $sid1
    $w3a['whoami'] = $whoami
    if ($null -eq $after) {
        $w3a['profile-after'] = 'none'
        Set-Claim 'W3a' 'refuted' (Join-Evidence $w3a)
    } else {
        $profilePath = $after.LocalPath
        $temporary = ($after.Status -band 1) -ne 0
        $w3a['profile-after'] = $after.LocalPath
        $w3a['on-disk'] = Test-Path -LiteralPath $after.LocalPath
        $w3a['status'] = $after.Status
        $w3a['loaded-now'] = $after.Loaded
        $ok = ($null -eq $before) -and $w3a['on-disk'] -and (-not $temporary) -and ($whoami -like "*\$Account")
        Set-Claim 'W3a' $(if ($ok) { 'hold' } else { 'refuted' }) (Join-Evidence $w3a)
    }
    $envEv = 'child USERPROFILE={0}; APPDATA={1}; TEMP={2}; account profile={3}' -f $childProfile,
        ((@($lines | Where-Object { $_ -like 'APPDATA=*' }) + @('APPDATA=')).Item(0).Substring(8)),
        ((@($lines | Where-Object { $_ -like 'TEMP=*' }) + @('TEMP=')).Item(0).Substring(5)),
        $(if ($profilePath) { $profilePath } else { 'none' })
    $envHolds = $profilePath -and ($childProfile -ieq $profilePath)
    Set-Claim 'W3a-env' $(if ($envHolds) { 'hold' } else { 'refuted' }) $envEv
} catch {
    $w3a['start-process'] = 'failed: ' + $_.Exception.Message
    Set-Claim 'W3a' 'refuted' (Join-Evidence $w3a)
    Set-Claim 'W3a-env' 'untested' 'no child ran'
}

# ---- 4. W4: the engine copy runs as bf-spike, CLAUDE_CONFIG_DIR in the account's profile ----

if ($null -eq $profilePath) {
    Set-Claim 'W4' 'untested' 'no profile for bf-spike (W3a)'
} else {
    $cfg = Join-Path $profilePath 'bf-spike-claude'
    $w4 = @'
@echo off
set "USERPROFILE=__PROFILE__"
set "APPDATA=__PROFILE__\AppData\Roaming"
set "LOCALAPPDATA=__PROFILE__\AppData\Local"
set "TEMP=__PROFILE__\AppData\Local\Temp"
set "TMP=__PROFILE__\AppData\Local\Temp"
set "CLAUDE_CONFIG_DIR=__CFG__"
set "DISABLE_AUTOUPDATER=1"
if not exist "%TEMP%" mkdir "%TEMP%"
if not exist "%CLAUDE_CONFIG_DIR%" mkdir "%CLAUDE_CONFIG_DIR%"
"__ENGINE__" --version > "%~dp0w4-out.txt" 2>&1
>> "%~dp0w4-out.txt" echo exit=%ERRORLEVEL%
'@
    Write-Batch (Join-Path $Work 'w4.cmd') $w4.Replace('__PROFILE__', $profilePath).Replace('__CFG__', $cfg).Replace('__ENGINE__', $EngineExe)
    $ev = [ordered]@{}
    try {
        $ev['sha256-equal'] = (Get-FileHash -LiteralPath $EnginePath).Hash -eq (Get-FileHash -LiteralPath $EngineExe).Hash
        $ev['engine-acl'] = $engineAcl
        $ev['start-exit'] = Invoke-As -Credential $cred1 -Arguments '/d', '/c', ('"{0}"' -f (Join-Path $Work 'w4.cmd')) -WithProfile
        $out = Read-Out (Join-Path $Work 'w4-out.txt')
        $ev['output'] = $out
        $ev['cfg-entries'] = if (Test-Path -LiteralPath $cfg) { @(Get-ChildItem -LiteralPath $cfg -Force).Count } else { 'no dir' }
        $ok = ($out -match '\d+\.\d+\.\d+ \(Claude Code\)') -and ($out -match 'exit=0') -and $ev['sha256-equal']
        Set-Claim 'W4' $(if ($ok) { 'hold' } else { 'refuted' }) (Join-Evidence $ev)
    } catch {
        $ev['failed'] = $_.Exception.Message
        Set-Claim 'W4' 'refuted' (Join-Evidence $ev)
    }
}

# ---- 5. ACL: a file bf-spike writes in the root is unreadable by bf-spike2 ----

$acl = [ordered]@{ 'root-acl' = $rootAcl }
try {
    $acl['write-exit'] = Invoke-As -Credential $cred1 -Arguments '/d', '/c', ('"{0}"' -f (Join-Path $Work 'acl-write.cmd'))
    $acl['probe-exists'] = Test-Path -LiteralPath $Probe
    if (-not $acl['probe-exists']) {
        Set-Claim 'ACL-second-account' 'untested' ((Join-Evidence $acl) + ' bf-spike wrote no probe: ' + (Read-Out (Join-Path $Work 'acl-write-out.txt')))
    } else {
        $acl['probe-owner'] = (Get-Acl -LiteralPath $Probe).Owner
        $acl['probe-acl'] = Get-AclSummary $Probe
        $r = Invoke-AsCaptured $credB @('/d', '/c', ('type "{0}"' -f $Probe)) 'b-type'
        $acl['bf-spike2-type-exit'] = $r[0]
        $typeOut = $r[1]
        $acl['bf-spike2-type-stderr'] = $r[2]
        $r = Invoke-AsCaptured $credB @('/d', '/c', ('dir /b "{0}"' -f $Root)) 'b-dir'
        $acl['bf-spike2-dir-exit'] = $r[0]
        $acl['bf-spike2-dir-stderr'] = $r[2]
        $read = $typeOut -like "*$ProbeText*"
        $acl['bf-spike2-read-content'] = $read
        $ok = (-not $read) -and ($acl['bf-spike2-type-exit'] -ne '0')
        Set-Claim 'ACL-second-account' $(if ($ok) { 'hold' } else { 'refuted' }) (Join-Evidence $acl)
    }
} catch {
    $acl['failed'] = $_.Exception.Message
    Set-Claim 'ACL-second-account' 'untested' (Join-Evidence $acl)
}

# ---- 6. W1: a logon task for bf-spike registers with no password; ExecutionTimeLimit read back ----

$taskOut = Join-Path $Work 'task-out.txt'
$action = New-ScheduledTaskAction -Execute $Cmd -Argument ('/d /c whoami > "{0}" 2>&1' -f $taskOut)
$settings = New-ScheduledTaskSettingsSet -ExecutionTimeLimit ([TimeSpan]::Zero) -AllowStartIfOnBatteries `
    -DontStopIfGoingOnBatteries -RestartCount 3 -RestartInterval (New-TimeSpan -Minutes 1)
$w1 = [ordered]@{}
try {
    $principal = New-ScheduledTaskPrincipal -UserId $userId -LogonType Interactive -RunLevel Limited
    Register-ScheduledTask -TaskName $TaskName -TaskPath '\' -Action $action -Settings $settings `
        -Trigger (New-ScheduledTaskTrigger -AtLogOn -User $userId) -Principal $principal | Out-Null
    $x = [xml] (Export-ScheduledTask -TaskName $TaskName -TaskPath '\')
    $w1['registered'] = 'no password'
    $w1['LogonType'] = Get-TaskXmlValue $x 'LogonType'
    $w1['RunLevel'] = Get-TaskXmlValue $x 'RunLevel'
    $w1['ExecutionTimeLimit'] = Get-TaskXmlValue $x 'ExecutionTimeLimit'
    $ok = ($w1['LogonType'] -eq 'InteractiveToken') -and ($w1['ExecutionTimeLimit'] -eq 'PT0S')
    Set-Claim 'W1' $(if ($ok) { 'hold' } else { 'refuted' }) (Join-Evidence $w1)
} catch {
    $w1['register'] = 'failed: ' + $_.Exception.Message
    Set-Claim 'W1' 'refuted' (Join-Evidence $w1)
}

# ---- 7. W6: -User -Password -Force, rotation under a running process, Start-ScheduledTask ----

$pingProc = $null
try {
    Register-ScheduledTask -TaskName $TaskName -TaskPath '\' -Action $action -Settings $settings `
        -Trigger (New-ScheduledTaskTrigger -AtStartup) -User $userId `
        -Password $cred1.GetNetworkCredential().Password -RunLevel Limited -Force | Out-Null
    $x = [xml] (Export-ScheduledTask -TaskName $TaskName -TaskPath '\')
    $w6['register-password'] = 'ok'
    $w6['LogonType'] = Get-TaskXmlValue $x 'LogonType'
    $w6['ExecutionTimeLimit'] = Get-TaskXmlValue $x 'ExecutionTimeLimit'

    $pingProc = Start-Process -FilePath $Ping -ArgumentList '-n', '61', '127.0.0.1' -Credential $cred1 `
        -WorkingDirectory $env:SystemRoot -WindowStyle Minimized -PassThru
    Start-Sleep -Seconds 2
    $w6['ping-owner-before'] = Get-ProcessOwner $pingProc.Id

    Set-LocalUser -SID $sid1 -Password $pw2
    $w6['set-localuser'] = 'ok'
    Start-Sleep -Seconds 5
    $ownerAfter = Get-ProcessOwner $pingProc.Id
    $w6['ping-alive-after-rotate'] = ($ownerAfter -eq $Account)

    $cred2 = New-SpikeCredential $Account $pw2
    Register-ScheduledTask -TaskName $TaskName -TaskPath '\' -Action $action -Settings $settings `
        -Trigger (New-ScheduledTaskTrigger -AtStartup) -User $userId `
        -Password $cred2.GetNetworkCredential().Password -RunLevel Limited -Force | Out-Null
    $w6['reregister-new-password'] = 'ok'

    Start-ScheduledTask -TaskName $TaskName -TaskPath '\'
    $deadline = (Get-Date).AddSeconds(60)
    while (-not (Test-Path -LiteralPath $taskOut) -and (Get-Date) -lt $deadline) { Start-Sleep -Seconds 2 }
    Start-Sleep -Seconds 2
    $w6['task-ran-as'] = Read-Out $taskOut
    $w6['LastTaskResult'] = (Get-ScheduledTaskInfo -TaskName $TaskName -TaskPath '\').LastTaskResult

    $ok = $nonInteractive -and $w6['ping-alive-after-rotate'] -and ($w6['task-ran-as'] -like "*\$Account")
    Set-Claim 'W6' $(if ($ok) { 'hold' } else { 'refuted' }) (Join-Evidence $w6)
} catch {
    $w6['failed'] = $_.Exception.Message
    Set-Claim 'W6' 'refuted' (Join-Evidence $w6)
} finally {
    if ($null -ne $pingProc -and (Get-ProcessOwner $pingProc.Id) -eq $Account) {
        Stop-Process -Id $pingProc.Id -Force -ErrorAction SilentlyContinue
    }
}

foreach ($s in $pw1, $pw2, $pwB) { $s.Dispose() }
Write-Claims
Write-Output 'spike: done; next probe-unelevated.ps1 from a non-elevated PowerShell, then cleanup.ps1 elevated'
