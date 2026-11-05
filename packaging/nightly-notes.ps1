#!/usr/bin/env pwsh
# Writes the nightly release's notes: the commits since the last v* tag,
# newest first, with all but the five newest folded away. A port of
# Chatterino's .CI/format-recent-changes.py.
#
#     pwsh packaging/nightly-notes.ps1 > notes.md
#
# Needs the history and tags, e.g. actions/checkout with fetch-depth: 0.

$ErrorActionPreference = 'Stop'

# git describe fails when there's no v* tag; then a few recent commits do.
$lastTag = git describe --tags --abbrev=0 --match 'v*' 2>$null

$log = if ($LASTEXITCODE -eq 0 -and $lastTag) {
    git log "$lastTag..HEAD" '--pretty=format:%cI|%an|%s' --no-merges
} else {
    git log -n10 '--pretty=format:%cI|%an|%s' --no-merges
}

if ($LASTEXITCODE -ne 0) {
    throw 'git log failed'
}

$changes = foreach ($line in $log) {
    if (-not $line.Trim()) {
        continue
    }

    $date, $author, $subject = $line -split '\|', 3

    if ($author -eq 'dependabot[bot]') {
        continue
    }

    $when = [DateTimeOffset]::Parse($date).ToUniversalTime()

    [pscustomobject]@{ When = $when; Line = "- [$($when.ToString('yyyy-MM-dd'))] $subject" }
}

$changes = @($changes | Sort-Object When -Descending)

if ($changes.Count -eq 0) {
    'No changes since last release.'
}

$changes | Select-Object -First 5 | ForEach-Object Line

if ($changes.Count -gt 5) {
    "<details><summary>More Changes</summary>`n"
    $changes | Select-Object -Skip 5 | ForEach-Object Line
    '</details>'
}
