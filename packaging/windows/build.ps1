#!/usr/bin/env pwsh
# Builds the Windows app: a folder with Lockdeader.exe and every DLL, Qt
# plugin and QML module it needs, so it runs wherever the folder is put.
# Needs MSYS2's UCRT64 packages (see .github/workflows/build.yml for the
# list) and Go.
#
#     pwsh packaging/windows/build.ps1 [-Msys2Root C:\msys64] [-OutDir dist]
#
# Writes dist/Lockdeader, and tells GitHub Actions its path and the app's
# version as the step's "dir" and "version" outputs. GitHub zips the folder
# when it's uploaded as an artifact.

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
$appDir = Join-Path $OutDir 'Lockdeader'
$exe = Join-Path $appDir 'Lockdeader.exe'

if (Test-Path $appDir) {
    Remove-Item $appDir -Recurse -Force
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

# MSYS2's Qt looks for QML modules and translations under share\qt6 next to
# the app, not where windeployqt puts them, and without its QML modules the
# app can't load its window. qt.conf tells it where they are. Paths are
# relative to the app's folder.
Set-Content (Join-Path $appDir 'qt.conf') -Encoding utf8NoBOM -Value @'
[Paths]
Prefix = .
Plugins = .
QmlImports = qml
Translations = translations
'@

# windeployqt doesn't copy what Qt itself links against from MSYS2 (ICU,
# zstd, HarfBuzz, the MinGW runtime, ...), or what the plugins do (libjpeg,
# libwebp, ...). ntldd lists each file's imports; whatever it finds in
# ucrt64\bin is copied next to the app, and its own imports are then looked
# at too. Each file is read once, since starting ntldd is slow on Windows.
Write-Host 'Collecting the libraries Qt and its plugins use'
$copied = 0
$toScan = [System.Collections.Generic.Queue[string]]::new()

foreach ($binary in Get-ChildItem $appDir -Recurse -Include '*.exe', '*.dll') {
    $toScan.Enqueue($binary.FullName)
}

while ($toScan.Count -gt 0) {
    $output = ntldd $toScan.Dequeue() 2>&1

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
            $toScan.Enqueue((Resolve-Path $target).Path)
            $copied++
        }
    }
}

Write-Host "Copied $copied more DLLs"

Copy-Item LICENSE (Join-Path $appDir 'LICENSE.txt')

$size = [math]::Round((Get-ChildItem $appDir -Recurse -File | Measure-Object Length -Sum).Sum / 1MB)
Write-Host "Wrote $appDir ($size MB)"

if ($env:GITHUB_OUTPUT) {
    "dir=$(Resolve-Path $appDir)", "version=$version" | Out-File $env:GITHUB_OUTPUT -Append
}
