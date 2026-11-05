---
id: TASK-6
title: Show when installed mods have new releases on GameBanana
status: To Do
assignee: []
created_date: '2026-11-05T09:31:00Z'
updated_date: '2026-11-05T16:21:00Z'
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
- [ ] #1 On startup, every installed mod from GameBanana has its files looked up in the background, without slowing startup; a failed lookup is just tried again next time
- [ ] #2 A mod with files on GameBanana that aren't installed and were uploaded after its newest installed file is marked as having a new release, or how many new releases
- [ ] #3 Opening the mod's files to choose from highlights those new files as recently uploaded
- [ ] #4 Installing a new file goes through the existing configure flow (download, MD5 check, only then replace), keeping the mod on or off and its place in the load order, and the mark goes once the mod has its newest file
- [ ] #5 Mods not from GameBanana (local mods) are never marked
- [ ] #6 Tests cover new, older and already-installed files, and a failed lookup
<!-- AC:END -->
