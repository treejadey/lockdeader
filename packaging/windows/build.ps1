#!/usr/bin/env pwsh
# Builds the Windows zip: Lockdeader.exe with every DLL, Qt plugin and QML
# module it needs, so it runs where it's unzipped. Needs MSYS2's UCRT64
# packages (see .github/workflows/build.yml for the list) and Go.
#
#     pwsh packaging/windows/build.ps1 [-Msys2Root C:\msys64] [-OutDir dist]
#
# Writes dist/Lockdeader-<version>-windows-x64.zip, and tells GitHub Actions
# its path as the step's "zip" output.

[CmdletBinding()]
param(
    # Where MSYS2 is installed; its ucrt64\bin goes on the PATH.
    [string]$Msys2Root = 'C:\msys64',
    [string]$OutDir = 'dist'
)

$ErrorActionPreference = 'Stop'
$PSNativeCommandUseErrorActionPreference = $true

if (-not $IsWindows) {
    throw 'The Windows zip can only be built on Windows.'
}

$root = Resolve-Path (Join-Path $PSScriptRoot '..' '..')
Set-Location $root

$ucrt64Bin = Join-Path $Msys2Root 'ucrt64' 'bin'

if (-not (Test-Path $ucrt64Bin)) {
    throw "MSYS2's UCRT64 environment isn't at $ucrt64Bin. Pass -Msys2Root."
}

$env:PATH = "$ucrt64Bin;$env:PATH"

$version = (Get-Content VERSION.txt -Raw).Trim()
$staging = Join-Path $OutDir 'staging'
$appDir = Join-Path $staging 'Lockdeader'
$exe = Join-Path $appDir 'Lockdeader.exe'

if (Test-Path $staging) {
    Remove-Item $staging -Recurse -Force
}

New-Item -ItemType Directory -Path $appDir -Force | Out-Null

Write-Host "Building Lockdeader $version"

# No console window, and no symbol table or DWARF, which the zip doesn't
# need.
$env:CGO_ENABLED = '1'
go build -trimpath -ldflags '-s -w -H windowsgui' -o $exe .

# Qt's DLLs, plugins and the QML modules the app imports (found by scanning
# qml/), and Qt's own translations, which translations.go looks for next to
# the app. The software OpenGL renderer and the D3D shader compiler are left
# out: Qt Quick draws through Direct3D 11 here, with shaders compiled ahead
# of time.
Write-Host 'Collecting Qt'
windeployqt6 --release --qmldir qml --no-opengl-sw --no-system-d3d-compiler `
    --exclude-plugins qicns,qtga,qtiff,qwbmp $exe

# windeployqt doesn't copy what Qt itself links against from MSYS2 (ICU,
# zstd, HarfBuzz, the MinGW runtime, ...), or what the plugins do (libjpeg,
# libwebp, ...). ntldd lists each file's imports; whatever it finds in
# ucrt64\bin is copied next to the app, until nothing new turns up.
Write-Host 'Collecting the libraries Qt and its plugins use'
$copied = 0

do {
    $added = 0
    $binaries = Get-ChildItem $appDir -Recurse -Include '*.exe', '*.dll'

    foreach ($binary in $binaries) {
        $output = ntldd $binary.FullName 2>&1

        foreach ($line in $output) {
            if ("$line" -notmatch '^\s*(\S+)\s+=>\s+(.+?)\s+\(0x[0-9a-fA-F]+\)\s*$') {
                continue
            }

            $name, $path = $Matches[1], $Matches[2]

            if (-not $path.StartsWith($ucrt64Bin, [System.StringComparison]::OrdinalIgnoreCase)) {
                continue
            }

            $target = Join-Path $appDir $name

            if (-not (Test-Path $target)) {
                Copy-Item $path $target
                $added++
            }
        }
    }

    $copied += $added
} while ($added -gt 0)

Write-Host "Copied $copied more DLLs"

Copy-Item LICENSE (Join-Path $appDir 'LICENSE.txt')

# The zip holds a Lockdeader folder, so unzipping doesn't scatter files.
$zip = Join-Path (Resolve-Path $OutDir) "Lockdeader-$version-windows-x64.zip"

if (Test-Path $zip) {
    Remove-Item $zip
}

Add-Type -AssemblyName System.IO.Compression.FileSystem
[System.IO.Compression.ZipFile]::CreateFromDirectory((Resolve-Path $appDir), $zip,
    [System.IO.Compression.CompressionLevel]::Optimal, $true)

Remove-Item $staging -Recurse -Force

$size = [math]::Round((Get-Item $zip).Length / 1MB)
Write-Host "Wrote $zip ($size MB)"

if ($env:GITHUB_OUTPUT) {
    "zip=$zip" | Out-File $env:GITHUB_OUTPUT -Append
}
