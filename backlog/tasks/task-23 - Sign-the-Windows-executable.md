---
id: TASK-23
title: Sign the Windows executable
status: To Do
assignee: []
created_date: '2026-11-05T09:33:00Z'
labels:
  - distribution
dependencies: []
documentation:
  - doc-3
priority: low
type: chore
ordinal: 23000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Without a signature, Windows SmartScreen warns about `Lockdeader.exe` until it builds a reputation, which scares users off. The SignPath Foundation signs open-source projects for free.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Release builds of `Lockdeader.exe` are code-signed in CI
- [ ] #2 SmartScreen shows the publisher on a fresh Windows install
<!-- AC:END -->
