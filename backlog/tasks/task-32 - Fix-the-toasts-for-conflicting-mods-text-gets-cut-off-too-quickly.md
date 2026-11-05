---
id: TASK-32
title: 'Fix the toasts for conflicting mods: text gets cut off too quickly'
status: To Do
assignee: []
created_date: '2026-11-05T18:49:00Z'
labels:
  - mod-support
dependencies: []
priority: medium
type: bug
ordinal: 33000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Reported by the user (2026-11-05): "fix the toasts for conflicting mods, text gets cutoff too quickly, can't read the text fully, may have to show a hint text on hover".

The toasts in question are the ones clash.go announces when turning a mod on turns others off ("Turned off %s, which changes the same files", and on startup "…as another mod"), which carry Undo. `Toast.qml` shows one line, elided on the right (`Text.ElideRight`), at most 420 px wide (`maximumWidth`, also capped by the window in Main.qml); with the Undo and close buttons beside it, a long GameBanana mod name such as "Hatchling as Deadman Danny (OUTER WILDS)" leaves the important part cut off. A toast with an action goes away after 6 s (`actionDuration`). The user's idea is a hint with the full text on hover; how to fix it (hover hint, wrapping, a longer or paused timeout while hovered, shorter wording) is a UI decision to put to the user before building.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 The full text of a clash toast can be read, however long the mod names are
- [ ] #2 The toast doesn't go away while the user is reading it (e.g. while it's hovered)
- [ ] #3 Toasts with short text look as they do now
- [ ] #4 Works in both themes and at 125% display scaling
<!-- AC:END -->
