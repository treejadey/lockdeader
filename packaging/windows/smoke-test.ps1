#!/usr/bin/env pwsh
# Checks that the Windows app runs on a machine with nothing installed:
# starts it from the folder build.ps1 made, and fails if it doesn't get as
# far as its "starting Lockdeader" log line and stay up. A DLL, Qt plugin or
# QML module left out of the folder stops it before that.
#
#     pwsh packaging/windows/smoke-test.ps1 -AppDir dist\Lockdeader

[CmdletBinding()]
param(
    [Parameter(Mandatory)]
    [string]$AppDir,
    # How long the app gets to start.
    [int]$Seconds = 20
)

$ErrorActionPreference = 'Stop'

if (-not $IsWindows) {
    throw 'The Windows app can only be tested on Windows.'
}

# Windows shows a dialog, and waits, when an exe's DLL is missing. With this
# error mode, which child processes inherit, it fails straight away instead.
Add-Type -Namespace Win32 -Name ErrorMode -MemberDefinition @'
[DllImport("kernel32.dll")]
public static extern uint SetErrorMode(uint mode);
'@

$SEM_FAILCRITICALERRORS = 0x0001
$SEM_NOGPFAULTERRORBOX = 0x0002
$SEM_NOOPENFILEERRORBOX = 0x8000
[Win32.ErrorMode]::SetErrorMode($SEM_FAILCRITICALERRORS -bor $SEM_NOGPFAULTERRORBOX -bor $SEM_NOOPENFILEERRORBOX) | Out-Null

$work = Join-Path ([System.IO.Path]::GetTempPath()) "lockdeader-smoke-test-$PID"
New-Item -ItemType Directory -Path $work -Force | Out-Null

try {
    $exe = Join-Path $AppDir 'Lockdeader.exe'

    if (-not (Test-Path $exe)) {
        throw "There's no Lockdeader.exe in $AppDir"
    }

    # Its data goes here rather than the machine's own, and Qt's warnings,
    # which a windowed app otherwise keeps to itself, go to the log.
    $env:LOCKDEADER_DATA_DIR = Join-Path $work 'data'
    $env:QT_FORCE_STDERR_LOGGING = '1'

    $stdout = Join-Path $work 'stdout.txt'
    $stderr = Join-Path $work 'stderr.txt'
    $process = Start-Process -FilePath $exe -PassThru -RedirectStandardOutput $stdout -RedirectStandardError $stderr

    $exited = $process.WaitForExit($Seconds * 1000)

    if (-not $exited) {
        $process.Kill()
        $process.WaitForExit()
    }

    $log = (Get-Content $stdout -Raw -ErrorAction Ignore) + (Get-Content $stderr -Raw -ErrorAction Ignore)
    Write-Host '--- The app logged:'
    Write-Host $log
    Write-Host '---'

    if ($exited) {
        throw "Lockdeader exited with code $($process.ExitCode) within $Seconds seconds"
    }

    if ($log -notmatch 'starting Lockdeader') {
        throw "Lockdeader didn't log that it was starting, so it didn't get past creating the Qt application"
    }

    Write-Host "Lockdeader started and stayed up for $Seconds seconds"
} finally {
    Remove-Item $work -Recurse -Force -ErrorAction Ignore
}
