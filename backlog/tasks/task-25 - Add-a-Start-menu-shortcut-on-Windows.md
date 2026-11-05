---
id: TASK-25
title: Add a Start menu shortcut on Windows
status: To Do
assignee: []
created_date: '2026-11-05T09:33:00Z'
labels:
  - distribution
dependencies: []
documentation:
  - doc-3
  - docs/interface-guidelines.md
priority: low
type: feature
ordinal: 25000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The interface guidelines (Desktop integration) say the Windows install adds a Start menu shortcut named "Lockdeader" with the AppUserModelID "Lockdeader" that `notification/` uses, so notifications and the taskbar group under the app. The Windows build is a portable zip with no installer, so there is no shortcut. Either an installer, or the app creating its own shortcut on first run, would close the gap.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 After first run or install, a Start menu shortcut named "Lockdeader" exists with AppUserModelID "Lockdeader"
- [ ] #2 Notifications and the taskbar entry group under that shortcut
<!-- AC:END -->
