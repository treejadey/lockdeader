---
id: TASK-15
title: Find the oldest Qt version Lockdeader builds and runs with
status: To Do
assignee: []
created_date: '2026-11-05T09:33:00Z'
labels:
  - distribution
dependencies: []
documentation:
  - doc-3
priority: medium
type: spike
ordinal: 15000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The app is developed against Qt 6.11. Which distribution releases OBS can build for (Leap, Fedora, Ubuntu) depends on how old a Qt still works, and nobody has tried.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 The oldest working Qt 6 minor version is found by building and running against older ones
- [ ] #2 The result, and what breaks below it, is recorded in doc-3 and the README
<!-- AC:END -->
