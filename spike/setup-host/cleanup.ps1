# SPDX-License-Identifier: GPL-3.0-or-later
#
# P20 spike: removes what run.ps1 created. Run from an elevated PowerShell (README.md):
#   powershell.exe -NoProfile -NonInteractive -ExecutionPolicy Bypass -File .\cleanup.ps1
# Add -WhatIf to see what it would act on; -WhatIf also runs non-elevated (then tasks, processes
# and profiles of other accounts may be hidden from it). Safe to run again after a partial run.
#
# GUARD. Every Stop-, Unregister-, Remove- call below acts only on:
#   - an account whose name is exactly one of $SpikeAccounts ('bf-spike', 'bf-spike2'), removed by
#     the SID read from that exact name;
#   - the task named exactly $SpikeTask ('bf-spike-test') in the root folder '\';
#   - a process whose owner is exactly one of $SpikeAccounts on this computer;
#   - a profile whose folder name is bf-spike or bf-spike2 (with the ".<suffix>" Windows may add)
#     and whose SID resolves to a spike account or to no account;
#   - %ProgramData%\Broodforge-spike\ only while it holds bf-spike.marker with run.ps1's text, holds
#     nothing but the entries run.ps1 writes, and holds no link.
# Assert-SpikeName re-checks each name right before the call. No wildcard and no value from outside
# this file reaches those calls, so the account broodforge, the task Broodforge and a real
# instance's directory are never touched.
#
# Output: "cleanup: absent|removed|left <what>[: <reason>]" lines, then "cleanup: done, <n> left".

[CmdletBinding(SupportsShouldProcess = $true)]
param()

Set-StrictMode -Version 2.0
$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'

$SpikeAccounts = @('bf-spike', 'bf-spike2')
$SpikeTask     = 'bf-spike-test'
$Root          = Join-Path $env:ProgramData 'Broodforge-spike'
$Marker        = Join-Path $Root 'bf-spike.marker'
$MarkerPrefix  = 'bf-spike spike root (spike/setup-host/run.ps1)'
$RootEntries   = @('bf-spike.marker', 'engine', 'spike', 'bf-spike-acl-probe.txt')
$Left = New-Object System.Collections.Generic.List[string]

# Load the modules before any -WhatIf takes effect: an auto-load under -WhatIf prints its Set-Alias
# lines as "What if" noise.
$wi = $WhatIfPreference
$WhatIfPreference = $false
Import-Module CimCmdlets, ScheduledTasks, Microsoft.PowerShell.LocalAccounts
$WhatIfPreference = $wi

function Assert-SpikeName([string] $Name) {
    if (($SpikeAccounts -notcontains $Name) -and ($Name -ne $SpikeTask)) { throw "guard: '$Name' is not a spike name" }
}

function Write-Absent([string] $What) { Write-Output "cleanup: absent $What" }
function Write-Removed([string] $What) { Write-Output "cleanup: removed $What" }
function Write-Left([string] $What, [string] $Why) {
    $line = "left ${What}: " + ($Why -replace [regex]::Escape($env:ProgramData), '%ProgramData%')
    $Left.Add($line)
    Write-Output "cleanup: $line"
}

function Get-Inner($Exception) {
    $e = $Exception
    while ($e -is [System.Management.Automation.MethodInvocationException] -and $null -ne $e.InnerException) {
        $e = $e.InnerException
    }
    return $e
}

$id = [Security.Principal.WindowsIdentity]::GetCurrent()
$elevated = (New-Object Security.Principal.WindowsPrincipal $id).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
if (-not $elevated) {
    if (-not $WhatIfPreference) {
        Write-Output 'cleanup: refused: run from an elevated PowerShell (or add -WhatIf)'
        exit 2
    }
    Write-Output 'cleanup: not elevated; -WhatIf only, other accounts'' tasks, processes and profiles may be hidden'
}

# ---- 1. the task ----

$task = Get-ScheduledTask -TaskPath '\' -TaskName $SpikeTask -ErrorAction SilentlyContinue
if ($null -eq $task) {
    Write-Absent "task $SpikeTask"
} else {
    Assert-SpikeName $task.TaskName
    if ($PSCmdlet.ShouldProcess("task \$SpikeTask", 'Stop-ScheduledTask, Unregister-ScheduledTask')) {
        try {
            Stop-ScheduledTask -TaskPath '\' -TaskName $SpikeTask -ErrorAction SilentlyContinue
            Unregister-ScheduledTask -TaskPath '\' -TaskName $SpikeTask -Confirm:$false
            Write-Removed "task $SpikeTask"
        } catch { Write-Left "task $SpikeTask" $_.Exception.Message }
    }
}

# ---- 2. processes of the spike accounts ----

$found = 0
foreach ($proc in @(Get-CimInstance -ClassName Win32_Process)) {
    try { $o = Invoke-CimMethod -InputObject $proc -MethodName GetOwner -WhatIf:$false } catch { continue }
    if ($null -eq $o -or $o.ReturnValue -ne 0) { continue }
    if ($o.Domain -ne $env:COMPUTERNAME -or $SpikeAccounts -notcontains $o.User) { continue }
    $found++
    Assert-SpikeName $o.User
    $what = 'process {0} ({1}) of {2}' -f $proc.ProcessId, $proc.Name, $o.User
    if ($PSCmdlet.ShouldProcess($what, 'Stop-Process')) {
        try { Stop-Process -Id $proc.ProcessId -Force; Write-Removed $what } catch { Write-Left $what $_.Exception.Message }
    }
}
if ($found -eq 0) { Write-Absent 'processes of bf-spike, bf-spike2' }

# ---- 3. profiles (Remove-CimInstance of Win32_UserProfile drops the folder and the hive) ----

$found = 0
foreach ($p in @(Get-CimInstance -ClassName Win32_UserProfile | Where-Object { $_.LocalPath })) {
    $leaf = Split-Path -Leaf $p.LocalPath
    if (-not ($leaf -match '^(bf-spike2?)(\.[^\\]+)?$')) { continue }
    $name = $Matches[1]
    $owner = try { (New-Object System.Security.Principal.SecurityIdentifier $p.SID).Translate(
        [System.Security.Principal.NTAccount]).Value } catch { $null }
    $what = "profile $leaf"
    if ($null -ne $owner -and $SpikeAccounts -notcontains ($owner -split '\\')[-1]) {
        Write-Left $what 'its SID belongs to another account'
        continue
    }
    $found++
    Assert-SpikeName $name
    if ($p.Loaded) { Write-Left $what 'loaded (a process of the account still runs; run cleanup.ps1 again)'; continue }
    if ($PSCmdlet.ShouldProcess($what, 'Remove-CimInstance Win32_UserProfile')) {
        try { Remove-CimInstance -InputObject $p; Write-Removed $what } catch { Write-Left $what $_.Exception.Message }
    }
}
if ($found -eq 0) { Write-Absent 'profiles of bf-spike, bf-spike2' }

# ---- 4. accounts ----

foreach ($name in $SpikeAccounts) {
    $u = Get-LocalUser -Name $name -ErrorAction SilentlyContinue
    if ($null -eq $u) { Write-Absent "account $name"; continue }
    Assert-SpikeName $u.Name
    if ($PSCmdlet.ShouldProcess("account $($u.Name)", 'Remove-LocalUser')) {
        try { Remove-LocalUser -SID $u.SID; Write-Removed "account $($u.Name)" } catch { Write-Left "account $name" $_.Exception.Message }
    }
}

# ---- 5. %ProgramData%\Broodforge-spike\, only the spike's ----

$rootName = '%ProgramData%\Broodforge-spike'
if (-not (Test-Path -LiteralPath $Root)) {
    Write-Absent $rootName
} else {
    $first = $null
    $why = $null
    try { $first = @([System.IO.File]::ReadAllLines($Marker))[0] }
    catch {
        $e = Get-Inner $_.Exception
        if ($e -is [System.IO.FileNotFoundException]) { $why = 'no bf-spike.marker; not the spike''s directory, not touched' }
        else { $why = 'cannot read bf-spike.marker: ' + $e.Message }
    }
    if ($null -eq $why -and ($null -eq $first -or -not $first.StartsWith($MarkerPrefix))) {
        $why = 'bf-spike.marker does not carry run.ps1''s text; not touched'
    }
    if ($null -eq $why) {
        $extra = @(Get-ChildItem -LiteralPath $Root -Force | Where-Object { $RootEntries -notcontains $_.Name } | ForEach-Object { $_.Name })
        if ($extra.Count -gt 0) { $why = 'holds entries run.ps1 does not write (' + ($extra -join ', ') + '); not touched' }
    }
    if ($null -eq $why) {
        $links = @(Get-ChildItem -LiteralPath $Root -Recurse -Force -Attributes ReparsePoint)
        if ($links.Count -gt 0) { $why = 'holds a link or junction; not touched' }
    }
    if ($null -ne $why) {
        Write-Left $rootName $why
    } elseif ($PSCmdlet.ShouldProcess($rootName, 'Remove-Item -Recurse')) {
        try { Remove-Item -LiteralPath $Root -Recurse -Force; Write-Removed $rootName } catch { Write-Left $rootName $_.Exception.Message }
    }
}

Write-Output ('cleanup: done, {0} left' -f $Left.Count)
if ($Left.Count -gt 0) { exit 1 }
