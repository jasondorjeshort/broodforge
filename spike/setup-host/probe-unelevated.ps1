# SPDX-License-Identifier: GPL-3.0-or-later
#
# P20 spike, the owner's half of the ACL claim: run after run.ps1, from a non-elevated PowerShell as
# the owner, before cleanup.ps1. Reads only; creates, changes and removes nothing.
#   powershell.exe -NoProfile -NonInteractive -ExecutionPolicy Bypass -File .\probe-unelevated.ps1
# Output: "claim ACL-owner-file: ..." and "claim ACL-owner-list: ..." lines, hold meaning denied.

[CmdletBinding()]
param()

Set-StrictMode -Version 2.0
$ErrorActionPreference = 'Stop'

$Root  = Join-Path $env:ProgramData 'Broodforge-spike'
$Probe = Join-Path $Root 'bf-spike-acl-probe.txt'

function Get-Inner($Exception) {
    $e = $Exception
    while ($e -is [System.Management.Automation.MethodInvocationException] -and $null -ne $e.InnerException) {
        $e = $e.InnerException
    }
    return $e
}

$id = [Security.Principal.WindowsIdentity]::GetCurrent()
$elevated = (New-Object Security.Principal.WindowsPrincipal $id).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
if ($elevated) {
    Write-Output 'spike: refused: this probe must run from a non-elevated PowerShell'
    exit 2
}
$lua = try { (Get-ItemProperty 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System').EnableLUA } catch { 'unknown' }
$ctx = 'elevated=False EnableLUA={0}' -f $lua

if (-not (Test-Path -LiteralPath $Root)) {
    Write-Output ('claim ACL-owner-file: untested: {0} no %ProgramData%\Broodforge-spike (run.ps1 not run?)' -f $ctx)
    Write-Output ('claim ACL-owner-list: untested: {0} no %ProgramData%\Broodforge-spike (run.ps1 not run?)' -f $ctx)
    exit 1
}

try {
    $text = [System.IO.File]::ReadAllText($Probe)
    Write-Output ('claim ACL-owner-file: refuted: {0} read {1} characters of the probe' -f $ctx, $text.Length)
} catch {
    $e = Get-Inner $_.Exception
    if ($e -is [System.UnauthorizedAccessException]) {
        Write-Output ('claim ACL-owner-file: hold: {0} open denied ({1})' -f $ctx, $e.GetType().Name)
    } else {
        Write-Output ('claim ACL-owner-file: untested: {0} {1}: {2}' -f $ctx, $e.GetType().Name, ($e.Message -replace [regex]::Escape($env:ProgramData), '%ProgramData%'))
    }
}

try {
    $n = @([System.IO.Directory]::GetFileSystemEntries($Root)).Count
    Write-Output ('claim ACL-owner-list: refuted: {0} listed {1} entries' -f $ctx, $n)
} catch {
    $e = Get-Inner $_.Exception
    if ($e -is [System.UnauthorizedAccessException]) {
        Write-Output ('claim ACL-owner-list: hold: {0} listing denied ({1})' -f $ctx, $e.GetType().Name)
    } else {
        Write-Output ('claim ACL-owner-list: untested: {0} {1}: {2}' -f $ctx, $e.GetType().Name, ($e.Message -replace [regex]::Escape($env:ProgramData), '%ProgramData%'))
    }
}
