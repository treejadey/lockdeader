---
id: TASK-31
title: Install mods whose VPKs vpk.ReadDir turns down
status: To Do
assignee: []
created_date: '2026-11-05T18:32:00Z'
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
- [ ] #1 A version 1 VPK installs, and its files are listed for clash detection
- [ ] #2 A VPK listing an empty file at offset 0xFFFFFFFF installs, and the empty file reads as empty
- [ ] #3 Bundling (deploy past 99 VPKs) works with both kinds
- [ ] #4 Entries that really point outside the data are still turned down
- [ ] #5 Tests build both kinds in code
<!-- AC:END -->
