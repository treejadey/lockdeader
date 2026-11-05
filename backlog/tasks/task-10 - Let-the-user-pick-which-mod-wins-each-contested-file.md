---
id: TASK-10
title: Let the user pick which mod wins each contested file
status: To Do
assignee: []
created_date: '2026-11-05T09:32:00Z'
labels:
  - mod-support
dependencies:
  - TASK-1
documentation:
  - doc-2
priority: low
type: feature
ordinal: 10000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
After TASK-1, two mods that change the same files can't both be on. Many such pairs overlap only partly (e.g. a model mod and a texture mod for the same hero), and users would want both, choosing per file. The plan is to generate a small `pak01_dir.vpk` (lowest number wins) holding only the contested files, from whichever mod the user picked for each; it is a few MB and is rebuilt only when conflicts or choices change. The VPK writer (`vpk.Write`) already exists. See doc-2, section 3, "Later: per-file resolution".
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Conflicting mods can both be on, with each contested file taken from the mod the user picked
- [ ] #2 Default choices follow load order, so users only touch the files they want to change
- [ ] #3 The user is warned about partial overlaps, where mod A wins some shared files and loses others
- [ ] #4 The generated VPK is only rebuilt when conflicts or choices change
<!-- AC:END -->
