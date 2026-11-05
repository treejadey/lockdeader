---
id: TASK-11
title: Attach the Windows zip to a GitHub Release on version tags
status: To Do
assignee: []
created_date: '2026-11-05T09:33:00Z'
labels:
  - distribution
dependencies: []
documentation:
  - doc-3
priority: high
type: chore
ordinal: 11000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
CI already builds the Windows app folder on every push and uploads it as a workflow artifact, but artifacts expire and need a GitHub login. What users download should come from a tagged release (doc-3, Principles). To release, `VERSION.txt` is changed and the commit tagged `v` plus that version.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 A `v*` tag zips the Windows app folder as `Lockdeader-<version>-windows-x64.zip` and attaches it to a GitHub Release with a `.sha256` file
- [ ] #2 The Flatpak bundle is attached to the same release
- [ ] #3 The release job fails if the tag does not match `VERSION.txt`
<!-- AC:END -->
