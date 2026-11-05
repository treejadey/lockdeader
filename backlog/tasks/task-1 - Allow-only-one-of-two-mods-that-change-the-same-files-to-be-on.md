---
id: TASK-1
title: Allow only one of two mods that change the same files to be on
status: Done
assignee:
  - '@claude'
created_date: '2026-11-05T09:31:00Z'
updated_date: '2026-11-05T10:29:00Z'
labels:
  - mod-support
dependencies: []
documentation:
  - doc-2
  - doc-1
priority: high
type: feature
ordinal: 1000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Today, when two enabled mods replace the same files, the one higher in the load order silently wins every shared file. When they overlap only partly the result is a broken mix (one mod's model with the other's materials), and the user has no idea why. The first step is to make conflicts visible and impossible to enable together; per-file resolution comes later. Reading a VPK's file list with `vpk.ReadDir` only reads the directory, so it is instant. This also guarantees the slot allocator (`deploy/alloc.go`) never has to split a bundle. DMM's noise list applies: `addoninfo.txt`, `modinfo.json`, readmes, `panorama/image_compiler.vdata_c` and `materials/default/default_*` do not count as conflicts. See doc-2, section 3.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Each installed VPK's file list is read from the store at most once per run and kept in memory by SHA-256 (store files never change), with no schema change
- [x] #2 Two mods conflict when they replace any of the same files, ignoring the noise files listed in the description
- [x] #3 Turning on a mod that conflicts with an enabled one turns the other off, and a toast names the mod turned off and offers Undo; the same applies to a new install, a taken-over mod and a configuration change
- [x] #4 On startup, enabled mods that clash are resolved: the one that wins in the load order stays on, the others are turned off, with a toast offering Undo
- [x] #5 The installed page shows a warning icon on every installed mod that clashes with another, in the list and the grid, with a tooltip naming the mods it clashes with
- [x] #6 Tests cover conflicting, non-conflicting and noise-only overlaps
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Decisions (asked 2026-11-05): turn the others off with Undo; fix existing clashes on startup keeping the load-order winner; Fluent warning icon + tooltip on row and card; file lists in memory, not the DB.

1. deploy: export PathSet and ReadPaths (reuse the allocator's readPaths); add Clashes([]PathSet) [][]int, built from a path index so it is linear in paths, ignoring noise files (addoninfo.txt, modinfo.json, readme*, panorama/image_compiler.vdata_c, materials/default/default_*). Unit tests.
2. installed.go: cache of PathSet by SHA-256 from the store; a mod's set is the union of its enabled VPKs. New "clashes" role (names joined), recomputed after load, install, uninstall/undo, configure, take-over.
3. turnOffClashing(row): after a mod is turned on (SetEnabled, finished install, configure, AddInstalled), turn off enabled mods that clash; undoable toast. Generalise Undo/messageGone to cover this pending toggle as well as the pending uninstall.
4. ResolveClashes(): called from main.go after QML loads; walks enabled mods in load order, turns off the losers, undoable toast.
5. QML: warning_24_regular.svg from Fluent; icon after the name in InstalledList, badge bottom-left of the picture in InstalledGrid; tooltip "Changes the same files as %1". othermods sets the role to "".
6. Go tests for the rule, startup resolution and undo; extract translations; vet, gofmt, go test; check QML loads headless.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Built as planned. deploy: PathSet/ReadPaths exported, Clashes() in deploy/clash.go (path index, noise list). App: clash.go (cache by SHA-256, clashes role, settle/turnOffClashing, ResolveClashes, Undo via pendingToggle); wired into SetEnabled, finish, AddInstalled, Configure, Uninstall/putBack; main.go calls ResolveClashes after Main.qml loads. QML: ClashWarning.qml + Fluent warning_24_regular.svg in InstalledList (after the name) and InstalledGrid (badge bottom-left); othermods sets the role to "".

Verified: go test ./... passes (deploy TestClashes/TestReadPaths*, main TestTurningOn*, TestAddedModTurnsOffClashing, TestNoiseDoesntClash, TestResolveClashesOnStartup, TestVPKPathsReadOnce). Headless run on a seeded scratch data dir + fake game: startup turned off the later of two clashing mods, deploy placed 2 VPKs, no QML warnings about the new role. AC 5 (icon/tooltip look) not yet checked on screen: offscreen does not draw MultiEffect icons.

Seen during the smoke run: "Toast.qml:84 Binding loop detected for property width" when the startup toast shows. Not investigated; may predate this change.

Added TestInstallTurnsOffClashing (a downloaded install turns off the mod it clashes with). The configuration-change path calls the same settle() as install and take-over, but has no test of its own.

Fixed the Toast binding loop seen in the smoke run (it predates this task; the startup toast exposed it): the label's rightPadding changed with hasAction, and Text only recomputes implicitWidth when it is read, so the width binding changed it while evaluating. The padding is now constant and root.rightPadding makes up the difference, with the same final widths. Checked with a QML harness (both text renderers) and a headless app run: no warning.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Only one of two mods that change the same files can now be on.

- deploy.Clashes finds clashing mods from their VPK file lists in one pass, ignoring files many mods carry (addoninfo.txt, modinfo.json, readmes, panorama/image_compiler.vdata_c, materials/default/default_*). deploy.ReadPaths is exported from the allocator.
- clash.go: file lists are read from the store once per run, cached by SHA-256 (no schema change). Turning a mod on, installing, taking over or reconfiguring one that is on turns the clashing ones off with an Undo toast; on startup ResolveClashes keeps the load-order winner of mods already on together.
- ClashWarning.qml (Fluent warning icon, tooltip naming the other mods) after the name in the list and on a badge styled like the unchecked TickBox on cards.
- Also fixed a pre-existing Toast binding loop the startup toast exposed (label padding now constant).

Verified: go test ./... (7 new app tests, 4 deploy tests), headless runs on seeded data with a fake game folder, and the user checked the icon in both styles on screen.

Not covered by its own test: the configuration-change path (shares settle() with install and take-over). Follow-up already planned: TASK-10 (per-file resolution).
<!-- SECTION:FINAL_SUMMARY:END -->
