#Requires -Version 5.1
<#
.SYNOPSIS
    Removes the atom command installed by install.ps1.

.PARAMETER InstallDir
    Where atom was installed. Defaults to %LOCALAPPDATA%\Atom-Neo.

.EXAMPLE
    powershell -ExecutionPolicy Bypass -File uninstall.ps1
#>
[CmdletBinding()]
param(
    [string]$InstallDir = (Join-Path $env:LOCALAPPDATA 'Atom-Neo')
)

$ErrorActionPreference = 'Stop'
$binDir = Join-Path $InstallDir 'bin'

Write-Host ''
Write-Host 'Removing Atom-Neo' -ForegroundColor Cyan
Write-Host ''

# Only the entry this installer added is removed; the rest of PATH is rewritten
# exactly as it was.
$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')

$entries = @()
if ($userPath) {
    $entries = $userPath -split ';' | Where-Object { $_ -ne '' }
}

# Membership is checked explicitly so PATH is only ever written when this entry
# is genuinely there. Comparing joined strings instead would rewrite PATH just
# because an empty segment got dropped.
if ($entries -contains $binDir) {
    $remaining = $entries | Where-Object { $_ -ne $binDir }
    [Environment]::SetEnvironmentVariable('Path', ($remaining -join ';'), 'User')
    Write-Host '  Removed from your PATH'
} else {
    Write-Host '  Was not on your PATH'
}

if (Test-Path $InstallDir) {
    Remove-Item -Path $InstallDir -Recurse -Force
    Write-Host "  Deleted $InstallDir"
} else {
    Write-Host "  Nothing installed at $InstallDir"
}

Write-Host ''
Write-Host 'Done. Projects and atom_modules folders were left alone.' -ForegroundColor Green
Write-Host ''
