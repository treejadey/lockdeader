---
id: TASK-9
title: Support archives that contain other archives
status: To Do
assignee: []
created_date: '2026-11-05T09:32:00Z'
labels:
  - mod-support
dependencies: []
documentation:
  - doc-2
priority: low
type: enhancement
ordinal: 9000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Some GameBanana mods (Gameplay Modifications › Randomised Abilities) ship their VPKs inside nested archives such as `rolls/*.zip`. `install.Extract` does not look inside them, so these mods currently install nothing. See doc-2, "What's on GameBanana".
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 VPKs inside zip, 7z or rar archives nested in the downloaded archive are found and offered like top-level ones
- [ ] #2 Nesting depth is bounded so a malicious archive can't recurse forever
- [ ] #3 A test covers a nested archive
<!-- AC:END -->
