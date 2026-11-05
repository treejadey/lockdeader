---
id: TASK-14
title: Add release notes to the AppStream metainfo and validate it in CI
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
ordinal: 14000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The metainfo lists releases with only a version, date and URL. Software centres and Flathub show release notes from it, and Flathub's review checks it, so problems should show up in CI rather than at submission. CHANGELOG.md already holds the notes.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Each `<release>` in the metainfo has a description matching CHANGELOG.md
- [ ] #2 CI runs `appstreamcli validate` on the metainfo and fails on errors
<!-- AC:END -->
