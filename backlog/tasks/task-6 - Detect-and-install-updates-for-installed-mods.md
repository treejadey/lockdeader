---
id: TASK-6
title: Show when installed mods have new releases on GameBanana
status: Done
assignee:
  - '@claude'
created_date: '2026-11-05T09:31:00Z'
updated_date: '2026-11-07T15:15:00Z'
labels:
  - mod-support
dependencies: []
documentation:
  - doc-2
  - doc-1
priority: medium
type: feature
ordinal: 6000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Requested 2026-11-05: on startup, check every installed mod's GameBanana page for files uploaded since the ones installed. A mod with new releases says so on the installed page, and when its files are opened to choose from (its options dialog, ModConfigDialog/FilePickerList), the new files are highlighted as recently uploaded, so picking the new version is easy. The user picks; nothing updates by itself. Context: installed.lookUp/lookUpAll (configure.go) already fetches each installed mod's files when the installed page is shown, and mod_files stores each installed archive's GameBanana file id, MD5 and upload time, so a file is new when it isn't installed and was uploaded after the newest installed file. DMM compares the mod's modified date instead, which changes for any edit (doc-1, "Downloads and updates"). How the mark and the highlight look is a UI decision to ask about when the task starts.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 On startup, every installed mod from GameBanana has its files looked up in the background, without slowing startup; a failed lookup is just tried again next time
- [x] #2 A mod with files on GameBanana that aren't installed and were uploaded since the user last installed any of its files (not merely after its newest installed file) is marked as having a new release, or how many new releases
- [x] #3 Opening the mod's files to choose from highlights those new files as recently uploaded
- [x] #4 Installing a new file goes through the existing configure flow (download, MD5 check, only then replace), keeping the mod on or off and its place in the load order, and the mark goes once the mod has its newest file
- [x] #5 Mods not from GameBanana (local mods) are never marked
- [x] #6 Tests cover new, older and already-installed files, and a failed lookup
- [x] #7 Each new file's highlight can be dismissed, which lasts across restarts; once every new file of a mod is dismissed, its mark goes
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Decided with the user (2026-11-07): an accent marker beside the clash and merge markers (a solid accent badge over pictures), with a tooltip giving the count, which opens Change Configuration when clicked; in the file list, an accent "New" pill with a dismiss button before a new file's upload time. A file is new when it isn't installed, was uploaded after the user last installed any of the mod's files (mods.installed_at, mod_files.downloaded_at), and hasn't been dismissed.
1. appdb: schema v3, dismissed_files (GameBanana file id, mod id, deleted with the mod); Mod.Dismissed; DismissFile.
2. configure.go: option.isNew by that rule; "isNew" in the configuration; DismissNew.
3. installed.go: a "newReleases" role, set in show(); a "dismissNew" key, {key, id}.
4. main.go: look up every installed mod's files once startup is done.
5. QML: NewReleaseNotice in InstalledGrid and InstalledList, opening the configuration; the pill and its dismiss button in FilePickerList, through ModConfigDialog.newDismissed to installed.dismissNew.
6. Tests in appdb and installed_test.go; check at 1.25 scale in both themes.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Checked (2026-11-07) in the nested KWin session at 1.25 scale, in both themes (dark forced in a throwaway build), on a scratch copy of the real database with every installed file's upload time set to 1970: the startup lookups against GameBanana marked 5 of 9 mods (those with other files), in the grid and the list, and Green Abrams' Configuration dialog showed the New pill on its newer file. The marker over a dark picture is fainter in the light theme, as ClashNotice's is: it uses the same see-through badge. Clicking the marker wasn't exercised with a real click; it calls the same function as the menu's Change Configuration….

Changed with the user (2026-11-07): over pictures the marker is a solid accent badge with the icon in textOnAccent, as the see-through one was faint over dark pictures in the light style. Checked in both themes at 1.25 scale: it shows on dark and light pictures alike.

Changed with the user (2026-11-07): new now means uploaded since the user last installed any of the mod's files, not after its newest installed file, so files that were already there when it was installed aren't marked; and each New pill has a dismiss button (dismissed_files, schema v3), with the mod's marker going once all are dismissed. Checked in the light theme at 1.25 scale through the real QML write (installed.dismissNew): the pill and Green Abrams' marker went, the row was in dismissed_files, and another mod kept its marker. The pill's colours are the same in both themes, as before.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
An installed mod with files on GameBanana uploaded since the user last installed any of its files shows an accent arrow-up marker: after its name in the list, and as a solid accent badge over its picture in the grid. Its tooltip gives the count, and clicking it opens Change Configuration. There, each such file has an accent "New" pill before its upload time, with a dismiss button. Dismissing is saved (schema v3, dismissed_files, deleted with the mod), and once all of a mod's new files are dismissed its marker goes. Picking a new file goes through the existing configure flow, keeping the mod on or off and in its place, and as that's a new download, files uploaded before it stop being new.

The installed mods' files are looked up on startup in the background, two at a time; a failed lookup marks nothing and is tried again when the installed page is next shown, or on the next start. Local mods are never marked.

Tests: appdb TestDismissFile; TestNewReleases (older files, one newer than the installed file but already there at install, new ones; picking the newest clears the mark and keeps the mod off and in place), TestDismissNew, TestNewReleasesFailedLookup, TestNewReleasesLocalMod. go test ./..., go vet, gofmt and the translations check pass. Screenshots at 1.25 scale in both themes for the markers and pill; dismissing checked in the light theme.
<!-- SECTION:FINAL_SUMMARY:END -->
