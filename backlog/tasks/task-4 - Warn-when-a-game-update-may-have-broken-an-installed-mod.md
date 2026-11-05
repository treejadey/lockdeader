---
id: TASK-4
title: Warn when a game update may have broken an installed mod
status: To Do
assignee: []
created_date: '2026-11-05T09:31:00Z'
labels:
  - mod-support
dependencies: []
documentation:
  - doc-2
priority: medium
type: feature
ordinal: 4000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Breakage after a Deadlock update currently goes unnoticed: a mod that replaces files Valve changed may look wrong or crash, and a mod whose files Valve removed silently does nothing. Comparing the base game's file list (path and CRC32 for every file in `citadel/pak01_dir.vpk`; only the directory is read) between builds catches both. See doc-2, section 4.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 A snapshot of the base game's file list is kept per game build
- [ ] #2 When the build changes, mods replacing files whose CRC changed get "This update changed files this mod replaces; it may be broken"
- [ ] #3 Mods whose replaced files no longer exist in the base game get "This mod no longer does anything"
- [ ] #4 Tests cover changed, removed and unchanged paths
<!-- AC:END -->
