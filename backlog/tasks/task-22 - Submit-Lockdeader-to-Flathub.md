---
id: TASK-22
title: Submit Lockdeader to Flathub
status: To Do
assignee: []
created_date: '2026-11-05T09:33:00Z'
labels:
  - distribution
dependencies:
  - TASK-11
  - TASK-12
  - TASK-13
  - TASK-14
  - TASK-19
  - TASK-20
  - TASK-21
documentation:
  - doc-3
priority: low
type: chore
ordinal: 22000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Flathub is where most Linux desktop users look for apps. Reviewers check the app ID and our right to it, the metainfo, that every permission is needed, and that the name and description don't suggest the app is Valve's. `--talk-name=org.freedesktop.Flatpak` will be questioned: it is needed to tell whether the game runs, as Chatterino's manifest does for OBS.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Every permission in the manifest has a comment saying why it is needed
- [ ] #2 The manifest is submitted as a pull request to flathub/flathub
- [ ] #3 Once accepted, `x-checker-data` updates the Flathub repository from our tags
<!-- AC:END -->
