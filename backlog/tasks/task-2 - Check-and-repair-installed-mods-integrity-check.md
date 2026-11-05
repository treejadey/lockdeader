---
id: TASK-2
title: Check and repair installed mods (integrity check)
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
ordinal: 2000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
A store file can be damaged or deleted (disk errors, antivirus, the user cleaning up). Like Steam's "Verify integrity of game files", the user should be able to confirm every installed mod is intact and get damaged ones repaired without reinstalling. The database already holds what is needed: each VPK's SHA-256 in `mod_vpks` (with an unused `verified_at` column for this), and the GameBanana file id and MD5 of the archive it came from in `mod_files`. Since the game folder is rebuilt from the store, repairing the store repairs the game too. Hashing every VPK takes a while for big setups. See doc-2, section 6.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 A check action hashes each stored VPK against its recorded SHA-256 and records each pass in `mod_vpks.verified_at`
- [ ] #2 A missing or mismatched VPK is repaired by re-downloading its archive, checking the archive MD5 and the VPK SHA-256 before it goes back into the store
- [ ] #3 When the archive is gone from GameBanana, the user is told and offered the mod's current files instead
- [ ] #4 The check runs in the background with visible progress
- [ ] #5 After a repair the game folder is applied again
<!-- AC:END -->
