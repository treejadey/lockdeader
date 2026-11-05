---
id: TASK-13
title: Add app icon PNGs and the Windows icon and version resource
status: To Do
assignee: []
created_date: '2026-11-05T09:33:00Z'
labels:
  - distribution
dependencies: []
documentation:
  - doc-3
  - docs/interface-guidelines.md
priority: medium
type: chore
ordinal: 13000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Only the scalable `assets/logo.svg` exists. The interface guidelines ask for the icon drawn for each size rather than scaled, and on Windows the `.exe` has no icon, file version or description, so Explorer and the taskbar show a generic one.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 PNGs at 16, 24, 32, 48, 64 and 128 px, each drawn for its size, are installed under the app ID
- [ ] #2 A Windows `.ico` built from them is embedded in `Lockdeader.exe` through a `.syso` resource (e.g. `goversioninfo`), which also sets the file version and description
- [ ] #3 The Windows build still works with plain `go build`
<!-- AC:END -->
