#!/usr/bin/env pwsh
# Builds the Flatpak and bundles it into one file that `flatpak install`
# takes. Needs flatpak and flatpak-builder.
#
#     pwsh packaging/flatpak/build.ps1 [-OutDir dist] [-StateDir .flatpak-builder]
#
# Writes dist/Lockdeader-<version>-linux-x86_64.flatpak, and tells GitHub
# Actions its path as the step's "bundle" output. The repository it's
# bundled from is left in dist/repo.

[CmdletBinding()]
param(
    [string]$OutDir = 'dist',
    # flatpak-builder's cache: downloads, built stages and ccache's
    # compiled objects, which make the next build much faster.
    [string]$StateDir = '.flatpak-builder',
    # Marks the app as a nightly build, as CI does for every build that
    # isn't from a v* tag (see version.go).
    [switch]$Nightly
)

$ErrorActionPreference = 'Stop'
$PSNativeCommandUseErrorActionPreference = $true

if (-not $IsLinux) {
    throw 'The Flatpak can only be built on Linux.'
}

$root = Resolve-Path (Join-Path $PSScriptRoot '..' '..')
Set-Location $root

$appId = 'io.github.treejadey.Lockdeader'
$manifest = Join-Path 'packaging' 'flatpak' "$appId.yml"
$version = (Get-Content VERSION.txt -Raw).Trim()
$flathub = 'https://dl.flathub.org/repo/flathub.flatpakrepo'

New-Item -ItemType Directory -Path $OutDir -Force | Out-Null
$repo = Join-Path $OutDir 'repo'
$bundle = Join-Path (Resolve-Path $OutDir) "Lockdeader-$version-linux-x86_64.flatpak"

Write-Host "Building Lockdeader $version"

# What the manifest's go build passes to -ldflags (see version.go): the
# commit, which Go can't read from the copy flatpak-builder builds, and
# whether it's a nightly. Written every time, so a stale one never lingers.
$flags = @()

# Left out, rather than failing the build, outside a Git checkout.
$commit = try { git rev-parse HEAD 2>$null } catch { $null }

if ($commit) {
    $flags += "-X main.buildCommit=$commit"
}

if ($Nightly) {
    $flags += '-X main.buildType=nightly'
}

Set-Content (Join-Path 'packaging' 'flatpak' 'build-flags') -Encoding utf8NoBOM -NoNewline -Value ($flags -join ' ')

# The runtime and SDK come from Flathub, into the user's installation.
flatpak remote-add --user --if-not-exists flathub $flathub

# ccache keeps miqt's compiled C++ between builds; Go's cgo finds it in
# place of gcc.
flatpak-builder --user --install-deps-from=flathub --force-clean --ccache `
    --state-dir=$StateDir --repo=$repo flatpak-build $manifest

# One file, which also says where to get the runtime from.
flatpak build-bundle --runtime-repo=$flathub $repo $bundle $appId

$size = [math]::Round((Get-Item $bundle).Length / 1MB)
Write-Host "Wrote $bundle ($size MB)"

if ($env:GITHUB_OUTPUT) {
    "bundle=$bundle" | Out-File $env:GITHUB_OUTPUT -Append
}
