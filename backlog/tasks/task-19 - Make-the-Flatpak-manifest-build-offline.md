---
id: TASK-19
title: Make the Flatpak manifest build offline
status: To Do
assignee: []
created_date: '2026-11-05T09:33:00Z'
labels:
  - distribution
dependencies: []
documentation:
  - doc-3
priority: medium
type: chore
ordinal: 19000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
`packaging/flatpak/io.github.treejadey.Lockdeader.yml` lets Go download modules during the build, which Flathub forbids. The modules must be listed as sources (flatpak-builder-tools' Go module tool) or come vendored in a release tarball.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 The manifest builds with the network turned off during the build
- [ ] #2 Regenerating the module sources after a `go.mod` change is a documented single command
<!-- AC:END -->
