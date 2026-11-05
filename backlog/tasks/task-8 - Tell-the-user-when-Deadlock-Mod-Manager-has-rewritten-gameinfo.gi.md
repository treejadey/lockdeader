---
id: TASK-8
title: Tell the user when Deadlock Mod Manager has rewritten gameinfo.gi
status: To Do
assignee: []
created_date: '2026-11-05T09:31:00Z'
labels:
  - mod-support
dependencies: []
documentation:
  - doc-2
priority: low
type: enhancement
ordinal: 8000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
DMM replaces the whole `SearchPaths` block in `gameinfo.gi`. Lockdeader still inserts its own line above DMM's `Game citadel`, so it keeps working, and `deploy.State.DMM` already detects DMM's `// Deadlock Mod Manager` markers, but nothing tells the user that two tools are editing the file, which matters when they run both.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 When `deploy.State.DMM` is true, the user sees a notice explaining that DMM also edits the game's configuration
- [ ] #2 The notice follows docs/interface-guidelines.md and offers a useful action (e.g. taking over DMM's mods)
<!-- AC:END -->
