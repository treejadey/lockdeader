---
id: TASK-5
title: Offer to go back to the last setup that ran without a crash
status: To Do
assignee: []
created_date: '2026-11-05T09:31:00Z'
labels:
  - mod-support
dependencies: []
documentation:
  - doc-2
priority: low
type: feature
ordinal: 5000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
A mod can crash the game. Launching vanilla is already instant (Play Without Mods), but the user then has to guess which mod did it. Recording the setup applied at each launch lets the app offer the last one that worked. See doc-2, "Mods that crash the game".
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 The set of applied mods is recorded at each Play With Mods launch
- [ ] #2 If the game exits abnormally soon after starting, the app offers to go back to the last setup that ran without a crash
- [ ] #3 The offer shows which mods changed since that setup
- [ ] #4 Going back can be undone
<!-- AC:END -->
