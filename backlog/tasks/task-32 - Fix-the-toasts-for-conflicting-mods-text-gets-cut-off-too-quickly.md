---
id: TASK-32
title: 'Fix the toasts for conflicting mods: text gets cut off too quickly'
status: Done
assignee:
  - '@claude'
created_date: '2026-11-05T18:49:00Z'
updated_date: '2026-11-07T14:40:00Z'
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
- [x] #1 The full text of a clash toast can be read, however long the mod names are
- [x] #2 The toast doesn't go away while the user is reading it (e.g. while it's hovered)
- [x] #3 Toasts with short text look as they do now
- [x] #4 Works in both themes and at 125% display scaling
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Decided with the user (2026-11-07): wider, then wrap.
1. Toast.qml: maximumWidth 600 (Main.qml caps it at the window, less 32 px). The label wraps instead of eliding; the toast grows taller to fit, with the buttons centred beside the text. A toast whose text fits on one line looks as it does now.
2. Toast.qml: a HoverHandler stops the timer while the pointer is over the toast, and restarts it when the pointer leaves; show() doesn't start the timer while the toast is hovered.
3. docs/interface-guidelines.md: say a toast stays while the pointer is over it.
4. Check in the nested KWin session at 1.25 scale, in both themes, with a short toast and a long clash toast.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Checked (2026-11-07): qmltestrunner tests in a scratch copy of qml/ (not in the repo, which has no QML test setup) cover going by itself, staying while hovered, a new message arriving while hovered, short text on one line at the old size (36 px with Undo), and long text wrapping at 600 px and at a 300 px cap. Screenshots in the nested KWin session at 1.25 scale, in both themes (dark forced in a throwaway build, since Qt ignores a requested colour scheme there): the long clash toast wraps to two lines with even padding, and the shorter ones look as before. Right after show() the toast's size lags one message behind until the next frame; the old toast does the same, so it isn't from this change.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Toasts are now up to 600 px wide (capped by the window, less 32 px) and wrap text that's longer, instead of cutting it off at 420 px. A wrapped toast with an action keeps the padding a single line has beside its buttons, so a clash toast with a long mod name shows all of its text. While the pointer is over a toast, its timer is stopped; it restarts in full when the pointer leaves, and a message shown while hovered waits too. Toasts that fit on one line look as before. The interface guidelines say so as well. Verified with QML tests (qmltestrunner, kept out of the repo) and screenshots at 1.25 scale in both themes.
<!-- SECTION:FINAL_SUMMARY:END -->
