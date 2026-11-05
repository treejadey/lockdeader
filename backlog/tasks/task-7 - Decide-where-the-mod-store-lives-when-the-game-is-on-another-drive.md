---
id: TASK-7
title: Decide where the mod store lives when the game is on another drive
status: To Do
assignee: []
created_date: '2026-11-05T09:31:00Z'
labels:
  - mod-support
dependencies: []
documentation:
  - doc-2
priority: medium
type: spike
ordinal: 7000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Mods are hard-linked from the store (`Cache/Mods` in the data dir) into the game folder, which only works on the same volume. On Windows the Steam library is often on another drive (`D:\SteamLibrary`) while the data dir is on `C:`, so many Windows users fall back to copying, which takes the space of every enabled mod twice. Option: keep the store on the game's volume (e.g. `<library>\Lockdeader\`), at the cost of one more place Lockdeader keeps files. exFAT can't hard-link at all, so copying stays as a fallback either way. See doc-2, "Putting files into our folder, and Windows".
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 The decision and its reasoning are recorded in doc-2
- [ ] #2 Follow-up implementation tasks are created if the store moves
<!-- AC:END -->
