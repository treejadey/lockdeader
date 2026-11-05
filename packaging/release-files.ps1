#!/usr/bin/env pwsh
# Turns a workflow run's artifacts into the files a release offers: the
# Windows app's folder zipped, the Flatpak bundle, and next to each a .sha256
# file that `sha256sum -c` checks a download with.
#
#     pwsh packaging/release-files.ps1 -ArtifactsDir artifacts -Label 0.3.0 [-OutDir release]
#
# The label goes into the files' names: the version for a release, "nightly"
# for the nightly, whose files keep their names from one build to the next.

[CmdletBinding()]
param(
    # Where actions/download-artifact put the run's artifacts.
    [Parameter(Mandatory)][string]$ArtifactsDir,
    [Parameter(Mandatory)][string]$Label,
    [string]$OutDir = 'release'
)

$ErrorActionPreference = 'Stop'
$PSNativeCommandUseErrorActionPreference = $true

# Exactly one of each, or something's wrong with the run.
function Find-One($what, $items) {
    $items = @($items)

    if ($items.Count -ne 1) {
        throw "Found $($items.Count) $what in $ArtifactsDir, not one: $($items -join ', ')"
    }

    $items[0]
}

$windows = Find-One 'Windows app folders' (Get-ChildItem $ArtifactsDir -Directory -Filter 'Lockdeader-*-windows-x64')
$flatpak = Find-One 'Flatpak bundles' (Get-ChildItem $ArtifactsDir -Recurse -File -Filter '*.flatpak')

New-Item -ItemType Directory -Path $OutDir -Force | Out-Null

# The app's files at the top of the zip, as in the one GitHub makes of the
# artifact.
$zip = Join-Path $OutDir "Lockdeader-$Label-windows-x64.zip"
Compress-Archive -Path (Join-Path $windows.FullName '*') -DestinationPath $zip -CompressionLevel Optimal -Force

$bundle = Join-Path $OutDir "Lockdeader-$Label-linux-x86_64.flatpak"
Copy-Item $flatpak.FullName $bundle

foreach ($file in $zip, $bundle) {
    $hash = (Get-FileHash $file -Algorithm SHA256).Hash.ToLowerInvariant()
    $name = Split-Path $file -Leaf

    # sha256sum's own format, with a Unix line ending wherever it's written.
    Set-Content "$file.sha256" -Encoding utf8NoBOM -NoNewline -Value "$hash  $name`n"
}

Get-ChildItem $OutDir | ForEach-Object { Write-Host "$($_.Name) ($([math]::Round($_.Length / 1MB, 1)) MB)" }
