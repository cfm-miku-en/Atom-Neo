#Requires -Version 5.1
<#
.SYNOPSIS
    Installs the atom command for the current user.

.DESCRIPTION
    Builds atom.exe (or uses one you already built), copies it under
    %LOCALAPPDATA%\Atom-Neo\bin, and puts that folder on your user PATH.
    No administrator rights are needed and nothing outside your profile
    is touched.

.PARAMETER InstallDir
    Where to install. Defaults to %LOCALAPPDATA%\Atom-Neo.

.PARAMETER NoPath
    Install the binary but leave PATH alone.

.EXAMPLE
    powershell -ExecutionPolicy Bypass -File install.ps1
#>
[CmdletBinding()]
param(
    [string]$InstallDir = (Join-Path $env:LOCALAPPDATA 'Atom-Neo'),
    [switch]$NoPath
)

$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $MyInvocation.MyCommand.Path
$binDir = Join-Path $InstallDir 'bin'
$target = Join-Path $binDir 'atom.exe'

function Write-Step($text) { Write-Host "  $text" }

Write-Host ''
Write-Host 'Installing Atom-Neo' -ForegroundColor Cyan
Write-Host ''

# Prefer building from source so the install matches this checkout; fall back to
# a binary that is already sitting here.
$source = $null
if (Get-Command go -ErrorAction SilentlyContinue) {
    Write-Step 'Building from source...'
    Push-Location $root
    try {
        & go build -ldflags '-s -w' -o (Join-Path $root 'atom.exe') ./src
        if ($LASTEXITCODE -ne 0) { throw 'go build failed' }
    } finally {
        Pop-Location
    }
    $source = Join-Path $root 'atom.exe'
} else {
    foreach ($candidate in @('atom.exe', 'atom3.exe')) {
        $path = Join-Path $root $candidate
        if (Test-Path $path) { $source = $path; break }
    }
    if (-not $source) {
        throw 'Go is not installed and no prebuilt atom.exe was found. Install Go, or build atom.exe first.'
    }
    Write-Step "Go not found, using $(Split-Path -Leaf $source)"
}

if (-not (Test-Path $binDir)) {
    New-Item -ItemType Directory -Path $binDir -Force | Out-Null
}

Copy-Item -Path $source -Destination $target -Force
Write-Step "Installed to $target"

if ($NoPath) {
    Write-Step 'Left PATH unchanged (-NoPath)'
} else {
    # Read the User scope specifically. Reading $env:Path instead would merge in
    # the machine PATH and then write the whole thing back into the user scope,
    # which is a common way to end up with a duplicated, bloated PATH.
    $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')

    $entries = @()
    if ($userPath) {
        $entries = $userPath -split ';' | Where-Object { $_ -ne '' }
    }

    if ($entries -contains $binDir) {
        Write-Step 'Already on your PATH'
    } else {
        $entries += $binDir
        [Environment]::SetEnvironmentVariable('Path', ($entries -join ';'), 'User')
        Write-Step 'Added to your PATH'
    }

    # Make it work in this session too, without waiting for a new terminal.
    if (($env:Path -split ';') -notcontains $binDir) {
        $env:Path = "$env:Path;$binDir"
    }
}

Write-Host ''

# The redirect happens inside cmd rather than PowerShell: redirecting a native
# command's stderr in PS 5.1 wraps every line in an error record, and the
# banner is written to stderr.
$banner = cmd /c "`"$target`" 2>&1" | Select-Object -First 1
if ($banner) {
    Write-Host "  $banner" -ForegroundColor DarkGray
}
Write-Host ''
Write-Host 'Done. Open a new terminal, then try:' -ForegroundColor Green
Write-Host '    atom repl'
Write-Host '    atom run'
Write-Host ''
