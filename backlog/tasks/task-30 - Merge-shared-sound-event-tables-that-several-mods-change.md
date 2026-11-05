---
id: TASK-30
title: Merge shared sound event tables that several mods change
status: To Do
assignee: []
created_date: '2026-11-05T17:29:00Z'
updated_date: '2026-11-05T18:47:00Z'
labels:
  - mod-support
dependencies:
  - TASK-1
documentation:
  - doc-2
  - doc-4
priority: medium
type: feature
ordinal: 30000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Voice mods for different heroes often clash only over shared, generated sound event tables such as `soundevents/generated_vo_hero_pick_vo.vsndevts_c`, `generated_vo_hero_death_vo.vsndevts_c` and `vo/hero_vote_reveal.vsndevts_c`. These hold one event per hero for every hero (the voice line files and their `vsnd_duration`), so every mod that touches its own hero ships a whole copy. Under TASK-1 this turns off mods that would work together, e.g. "Hatchling as Deadman Danny" (GameBanana 730794), "Ted Over Rem Mod" (731690) and "Flowery - Apollo" (705731).

Findings from comparing those mods with the game's own copy (pak01) on 2026-11-05, decompiled with Source2Viewer-CLI:
- Hatchling changes only `vsnd_duration` of `Generated.Deadpack.Hero.Pick.VO` and `.Remove.VO`; Ted only of `Generated.Familiar.Hero.Avoided.VO` and `.Remove.VO`. All other events are byte-identical to the game's, so the two merge without conflict.
- Flowery ships an old copy of the table (116 of 180 events): no Reveal, Debut or Voted events for any hero, none at all for Artist, Baba, Chessmaster, Deadpack, Nurse and Ratking, and old `context_name` values ("fencer" for "hero_fencer"). When it wins, those heroes lose their lines even with no other mod on. A merge that takes only what a mod changed from the game's copy fixes this too.
- The file is a Source 2 resource with a RED2 block (the compiler's input dependencies, identical between the game's copy and Hatchling's) and a DATA block of binary KV3 v5, LZ4-compressed in 16 KB frames, with two buffers (strings and values).

Unknowns: whether the game accepts a table written in an older or uncompressed KV3 version, and whether a mod that adds voice line files the game's table doesn't list (none of these three does) needs anything beyond the event's `vsnd_files`. TASK-10 plans a generated VPK for contested files; a merged table belongs in the same kind of VPK.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Mods whose only shared files are sound event tables can be on together, unless an event is a real conflict (#2), and each keeps the events that use its own sounds
- [ ] #2 An event that two mods that are on both change, each with sound files it ships, and differently, is still a clash as under TASK-1
- [ ] #3 In a table that two or more mods that are on share: a changed event comes from a mod only if it refers to a sound file that mod ships; events a mod adds are taken; everything else comes from the game's current copy, so an out-of-date copy no longer removes or reverts other heroes' events. Numbers compare at KV3 text precision (6 decimals)
- [ ] #4 A table that only one mod that's on has is used as the mod ships it, as now
- [ ] #5 A mod whose changes a merge leaves out shows an information marker on its card and list row, where ClashWarning sits, with a tooltip naming the mods it's merged with (proposed: "Some of its changes are left out so it works alongside %1"); a mod the merge leaves nothing out of shows nothing
- [ ] #6 The merged table is rebuilt when the mods that are on change or the game updates its own copy
- [ ] #7 Tests cover merging, a real conflict, an out-of-date mod table, a lone mod's table and a table that can't be read
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Checked in the game with Hatchling, Ted and Flowery on together: each hero's pick line plays
<!-- DOD:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Survey of 125 mods (2026-11-05) in doc-4, Merging files that mods share: sound event tables are mergeable per event; vdata only for added entries; per-hero panorama CSS and localization later, with their own readers; HUD panorama and assets not. Compare numbers at KV3 text precision (6 decimals), since mods recompile decompiled tables. Open: telling a mod's own changes from a stale copy's.

Decided with the user 2026-11-05: the own sounds rule (doc-4) applies only when two or more mods that are on share a table; a lone mod's table is used as it is. A merge that leaves out some of a mod's changes shows a quiet marker on that mod (tooltip, no toast or info bar), only when something is actually left out. Wording of the tooltip still to be confirmed when it's built.
<!-- SECTION:NOTES:END -->
