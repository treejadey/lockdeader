---
id: TASK-17
title: Build packages for other distributions on OBS
status: To Do
assignee: []
created_date: '2026-11-05T09:33:00Z'
labels:
  - distribution
dependencies:
  - TASK-15
  - TASK-16
documentation:
  - doc-3
priority: low
type: feature
ordinal: 17000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Once the Tumbleweed package works, the same spec can cover Fedora (and Leap) with a few conditionals, as far as their Qt is new enough. Debian and Ubuntu need their own packaging; only add them once someone asks.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 The OBS project builds for every Fedora and Leap release whose Qt is new enough
- [ ] #2 The README says which distributions have packages and how to add the repository
<!-- AC:END -->
