---
id: TASK-26
title: Upgrade miqt to v0.15.0 for Qt 6.12
status: In Progress
assignee:
  - '@claude'
created_date: '2026-11-05T11:05:00Z'
updated_date: '2026-11-05T11:05:00Z'
labels:
  - distribution
dependencies: []
priority: high
type: chore
ordinal: 26000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
MSYS2 moved to Qt 6.12.0 (seen 2026-11-05), whose headers make the QAccessible*Interface classes non-copyable, so miqt v0.14.0's gen_qaccessible.cpp no longer compiles and every Windows CI build fails (run 42013677048). miqt v0.15.0 fixes it (mappu/miqt#359, #360). CLAUDE.md documents several v0.14.0-specific workarounds (roleNames override, translate override, heap-corrupting list-of-pairs setters) that need checking against the new version.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 go.mod requires miqt v0.15.0 and the app builds and passes go test ./... against Qt 6.11 locally
- [ ] #2 The Windows CI job builds against MSYS2's Qt 6.12 and its smoke test passes
- [ ] #3 CLAUDE.md's notes on miqt v0.14.0 workarounds are checked against v0.15.0 and updated where they changed
<!-- AC:END -->
