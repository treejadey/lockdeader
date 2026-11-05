---
id: TASK-6
title: Detect and install updates for installed mods
status: To Do
assignee: []
created_date: '2026-11-05T09:31:00Z'
labels:
  - mod-support
dependencies: []
documentation:
  - doc-2
  - doc-1
priority: medium
type: feature
ordinal: 6000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Users have no way to tell that an installed mod has a newer version on GameBanana. DMM compares timestamps (`_tsDateAdded`, or `mdate` without saved file ids), which is fragile. Lockdeader already stores each installed archive's GameBanana file id and MD5 (`mod_files`), so it can compare against `DownloadPage` directly. See doc-2 section 7 and doc-1 "Downloads and updates".
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 A mod counts as having an update when a picked file is gone from its GameBanana files or a newer file replaces it, decided by file id and MD5, not timestamps
- [ ] #2 The installed page shows which mods have updates
- [ ] #3 Updating downloads the new file, verifies its MD5, and only then replaces the old VPKs, keeping the mod's on/off state and load order
<!-- AC:END -->
