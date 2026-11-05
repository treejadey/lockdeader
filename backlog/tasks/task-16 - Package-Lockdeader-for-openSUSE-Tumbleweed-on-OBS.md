---
id: TASK-16
title: Package Lockdeader for openSUSE Tumbleweed on OBS
status: To Do
assignee: []
created_date: '2026-11-05T09:33:00Z'
labels:
  - distribution
dependencies: []
documentation:
  - doc-3
priority: medium
type: feature
ordinal: 16000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Linux users need a native package. A home project on openSUSE's Open Build Service needs nobody's approval and hosts a repository users can add. OBS builds offline, so Go modules must be vendored. RPM finds shared libraries itself but not the QML modules the app imports or the SVG image plugin. See doc-3, "Linux: openSUSE's Open Build Service".
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 A `lockdeader.spec` with `_service` (obs_scm and go_modules) builds the app from a release tag in an OBS home project
- [ ] #2 The package installs the binary, desktop file, icons, metainfo and licences, and requires the QML modules and the SVG plugin
- [ ] #3 `%check` runs `go test ./...` with the offscreen platform
- [ ] #4 Pushing a `v*` tag triggers a rebuild on OBS
<!-- AC:END -->
