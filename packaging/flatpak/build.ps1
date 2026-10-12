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
    [string]$StateDir = '.flatpak-builder'
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
