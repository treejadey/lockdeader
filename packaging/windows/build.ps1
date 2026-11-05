#!/usr/bin/env pwsh
# Builds the Windows app: a folder with Lockdeader.exe and every DLL, Qt
# plugin and QML module it needs, so it runs wherever the folder is put.
# Needs MSYS2's UCRT64 packages (see .github/workflows/build.yml for the
# list) and Go.
#
#     pwsh packaging/windows/build.ps1 [-Msys2Root C:\msys64] [-OutDir dist] [-Nightly]
#
# Writes dist/Lockdeader, and tells GitHub Actions its path and the app's
# version as the step's "dir" and "version" outputs. GitHub zips the folder
# when it's uploaded as an artifact.

[CmdletBinding()]
param(
    # Where MSYS2 is installed; its ucrt64\bin goes on the PATH.
    [string]$Msys2Root = 'C:\msys64',
    [string]$OutDir = 'dist',
    # Marks the app as a nightly build, as CI does for every build that
    # isn't from a v* tag (see version.go).
    [switch]$Nightly
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
$ldflags = '-s -w -H windowsgui'

if ($Nightly) {
    $ldflags += ' -X main.buildType=nightly'
}

go build -trimpath -ldflags $ldflags -o $exe .

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

# The licences of what's compiled into the app: the Go modules and the fonts,
# icons and audio code in the repository.
Write-Host 'Writing the licence notices'
$notices = Join-Path $appDir 'THIRD-PARTY-NOTICES.txt'
go run ./cmd/notices -o $notices

# Everything else in the folder is a DLL from an MSYS2 package: Qt and what
# it uses. Each file is traced back to where it came from (see qt.conf for
# where windeployqt puts Qt's plugins and QML modules), pacman says which
# packages own those, and each package's licence files are copied into
# licenses\.
Write-Host 'Collecting the licences of the bundled libraries'
$ucrt64 = Join-Path $Msys2Root 'ucrt64'
$pacman = Join-Path $Msys2Root 'usr' 'bin' 'pacman.exe'
$licensesDir = Join-Path $appDir 'licenses'
$sources = @()

foreach ($dll in Get-ChildItem $appDir -Recurse -File -Filter '*.dll') {
    $relative = [System.IO.Path]::GetRelativePath($appDir, $dll.FullName)
    $parts = $relative -split '[\\/]', 2

    # Where in MSYS2 a file in the folder can have come from: next to the
    # app, from ucrt64\bin; in a folder, a Qt plugin or a QML module.
    $candidates = if ($parts.Count -eq 1) {
        @(Join-Path $ucrt64 'bin' $relative)
    } else {
        @(
            (Join-Path $ucrt64 'share' 'qt6' 'plugins' $relative)
            (Join-Path $ucrt64 'share' 'qt6' 'qml' $relative)
            (Join-Path $ucrt64 'share' 'qt6' 'qml' $parts[1])
        )
    }

    $source = $candidates | Where-Object { Test-Path $_ } | Select-Object -First 1

    if (-not $source) {
        throw "Can't tell which MSYS2 package $relative came from: it's in none of $($candidates -join ', ')."
    }

    # pacman knows files by their path inside MSYS2.
    $sources += '/' + [System.IO.Path]::GetRelativePath($Msys2Root, $source).Replace('\', '/')
}

$packages = & $pacman -Qqo @sources | Sort-Object -Unique

$section = @(
    ''
    '=' * 78
    'Libraries from MSYS2'
    'https://www.msys2.org'
    ''
    'Qt, and the libraries it and the app use, are DLLs from the MSYS2 packages'
    'below, linked dynamically, so they can be replaced with other builds. Each'
    "package's licence texts are in the licenses folder next to this file, and"
    'its source code is at https://github.com/msys2/MINGW-packages and'
    'https://repo.msys2.org/mingw/sources/.'
    ''
)

foreach ($package in $packages) {
    # Not $version, which is the app's, and goes to GitHub at the end.
    $info = & $pacman -Qi $package
    $packageVersion = ($info | Select-String '^Version\s*:\s*(.+)$').Matches[0].Groups[1].Value.Trim()
    $licence = ($info | Select-String '^Licenses\s*:\s*(.+)$').Matches[0].Groups[1].Value.Trim()
    $short = $package -replace '^mingw-w64-ucrt-x86_64-', ''

    # A package's files under share/licenses, without the folders. Some, like
    # ICU, keep theirs elsewhere (share/icu/<version>/LICENSE); then any file
    # named like a licence anywhere in the package will do.
    $prefix = '/ucrt64/share/licenses/'
    $all = & $pacman -Qlq $package | Where-Object { -not $_.EndsWith('/') }
    $files = @($all | Where-Object { $_.StartsWith($prefix) } | ForEach-Object { @{ From = $_; To = $_.Substring($prefix.Length) } })

    if (-not $files) {
        $files = @($all | Where-Object { (Split-Path $_ -Leaf) -match '^(licen[cs]e|copying|notice)([-._].*)?$' } |
            ForEach-Object { @{ From = $_; To = "$short/" + (Split-Path $_ -Leaf) } })
    }

    $folders = @()

    foreach ($file in $files) {
        $target = Join-Path $licensesDir $file.To
        New-Item -ItemType Directory -Path (Split-Path $target) -Force | Out-Null
        Copy-Item (Join-Path $Msys2Root $file.From.TrimStart('/')) $target
        $folders += 'licenses/' + $file.To.Split('/')[0]
    }

    $where = if ($folders) {
        ($folders | Sort-Object -Unique) -join ', '
    } else {
        # Not a reason to stop the build, but someone should look.
        Write-Host "::warning::The MSYS2 package $package has no licence files."

        'no licence file in the package'
    }

    $section += "$package $packageVersion"
    $section += "    Licence: $licence ($where)"
}

$section | Out-File $notices -Encoding utf8NoBOM -Append

Write-Host "Collected the licences of $($packages.Count) packages"

$size = [math]::Round((Get-ChildItem $appDir -Recurse -File | Measure-Object Length -Sum).Sum / 1MB)
Write-Host "Wrote $appDir ($size MB)"

if ($env:GITHUB_OUTPUT) {
    "dir=$(Resolve-Path $appDir)", "version=$version" | Out-File $env:GITHUB_OUTPUT -Append
}
