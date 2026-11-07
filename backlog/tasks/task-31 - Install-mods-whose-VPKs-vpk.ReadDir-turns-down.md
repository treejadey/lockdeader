---
id: TASK-31
title: Install mods whose VPKs vpk.ReadDir turns down
status: Done
assignee:
  - '@claude'
created_date: '2026-11-05T18:32:00Z'
updated_date: '2026-11-07T14:29:00Z'
labels:
  - mod-support
dependencies: []
documentation:
  - doc-4
priority: high
type: bug
ordinal: 32000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
In the survey of 125 mods for doc-4 (2026-11-05), Lockdeader couldn't install 20 of the 132 archives tried, because `vpk.ReadDir` (which install.Extract uses to check each VPK) turns down two kinds of VPK that the game loads:
- Version 1 VPKs (`vpk: not a version 2 VPK`): 15 archives, including recent uploads, e.g. GameBanana mod 736394 (Roster Tools, pak57_dir.vpk), 610813, 638547, 660322, 735723, and older sound mods (79267, 80204, 81950, 81969, 81982, 82032, 82339, 82963, 83629). Version 1 has a 12-byte header with no data section size or checksums; data follows the tree.
- An empty file listed at offset 0xFFFFFFFF with length 0 (`vpk: the file's directory is damaged`): 5 skins (628247 50 Cent Warden, 631628, 653713, 655733, 710434). In each it's an empty `readme.txt` in the data section (archive 0x7FFF), and the bounds check `offset+length > dataSize` fails on it. Source2Viewer's --vpk_verify passes on 631628.
Other users of vpk.Dir have to cope too: clash detection (deploy.ReadPaths), and bundling past 99 VPKs, which streams files out of stored VPKs with Dir.Open.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 A version 1 VPK installs, and its files are listed for clash detection
- [x] #2 A VPK listing an empty file at offset 0xFFFFFFFF installs, and the empty file reads as empty
- [x] #3 Bundling (deploy past 99 VPKs) works with both kinds
- [x] #4 Entries that really point outside the data are still turned down
- [x] #5 Tests build both kinds in code
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. vpk.ReadDir: read the 12-byte version 1 header (magic, version, tree size); data follows the tree and runs to the end of the file, so check that the furthest entry's last byte is there by reading it. Version 2 keeps checking against the header's data size. Other versions are still ErrVersion.
2. readEntry: an entry with no data (length 0) has its offset set to 0, so a placeholder offset such as 0xFFFFFFFF passes the bounds check and reads as empty through Open and ReadFile.
3. vpk tests: a version 1 VPK built in code (entries, contents, ReadFile, data cut off), an empty file at 0xFFFFFFFF in both versions, version 3 turned down.
4. deploy tests: ReadPaths lists a version 1 VPK; bundling past 99 includes a version 1 VPK and one with the empty file.
5. install test: Extract accepts both kinds.
6. go test ./..., go vet, gofmt, GOOS=windows go vet for vpk/deploy/install.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Already in a user's store: THAT BIRD THAT I HATE - Grey Talon Sound Mod (sha256 cffff1ce…) is a version 1 VPK installed before the version check existed. deploy.ReadPaths fails on it, so clash detection counts it as having no files (clash.go vpkPaths logs a warning).

Checked against the real store (2026-11-07): all 13 VPKs read, the version 1 Grey Talon sound mod (cffff1ce…) included, every file matching its CRC, and deploy.ReadPaths lists them. Clash detection picks up that mod's files on the next run, as file lists are read once per run.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
vpk.ReadDir now reads version 1 VPKs (12-byte header, data after the tree up to the end of the file). Since that header has no data size, it reads the byte where the furthest file ends, so a cut-off file is still ErrCorrupt. An entry with no data has its offset set to 0, so the empty readme.txt some skins list at 0xFFFFFFFF passes and reads as empty. Entries with data outside the file are still turned down in both versions, and versions other than 1 and 2 are still ErrVersion. Nothing else changed: install, clash detection and bundling all go through ReadDir and Dir.Open/ReadFile, and bundles are still written as version 2.

Tests (all build their VPKs in code): vpk TestReadDirVersion1 and TestReadDirEmptyFileAnywhere (both versions, plus data at 0xFFFFFFFF turned down); deploy TestReadPathsVersion1 and TestApplyBundlesOddVPKs; install TestExtractOddVPKs. All five fail on the old reader. go test ./..., go vet, gofmt and GOOS=windows go vet for vpk/deploy/install pass.

Not verified: loading these mods in the game. They go in as the same files as before, and the task's survey says the game loads them.
<!-- SECTION:FINAL_SUMMARY:END -->
