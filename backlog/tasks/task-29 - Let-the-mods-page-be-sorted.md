---
id: TASK-29
title: Let the mods page be sorted
status: To Do
assignee: []
created_date: '2026-11-05T16:21:00Z'
labels:
  - ui
dependencies: []
priority: medium
type: feature
ordinal: 29000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The mods ("store") page always lists mods in GameBanana's default order, newest first, so there's no way to find the most popular or recently updated mods. banana.ListModsOptions already takes a Sort (banana/mods.go: SortNewest, SortLatestUpdated, SortMostLiked, SortMostViewed, SortMostDownloaded); modmodel.go just never sets it. Check whether the API keeps the sort when a name search or a category is applied. How the choice is offered (control, wording, where it sits) is a UI decision to ask about when the task starts (docs/interface-guidelines.md).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 The mods page offers each sort order the GameBanana client supports, and changing it reloads the list in that order
- [ ] #2 The chosen order applies to every category and to searches, wherever the API allows it
- [ ] #3 The chosen order is remembered across restarts, like the page's other state (appdb.UIState)
- [ ] #4 Tests cover the sort reaching the API request
<!-- AC:END -->
