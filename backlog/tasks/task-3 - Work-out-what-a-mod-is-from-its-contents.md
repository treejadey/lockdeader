---
id: TASK-3
title: Work out what a mod is from its contents
status: To Do
assignee: []
created_date: '2026-11-05T09:31:00Z'
labels:
  - mod-support
dependencies: []
documentation:
  - doc-2
priority: medium
type: feature
ordinal: 3000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
GameBanana's metadata is unreliable for telling a mod's hero or kind (DMM guesses from category and mod names). Once a mod is downloaded, its VPK file list says far more. This only improves installed mods, not browsing; showing it while browsing needs our own API (see the draft for it). See doc-2, section 5.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 The hero a mod changes is derived from the paths it replaces
- [ ] #2 The mod type (skin, sounds, UI, particles) is derived from file extensions
- [ ] #3 Mods that change gameplay data (`*.vdata_c`) are labelled as only working on custom servers
- [ ] #4 For archives with several VPKs, the options dialog shows what each VPK changes before the user picks
<!-- AC:END -->
