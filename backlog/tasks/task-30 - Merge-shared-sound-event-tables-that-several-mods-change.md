---
id: TASK-30
title: Merge shared sound event tables that several mods change
status: To Do
assignee: []
created_date: '2026-11-05T17:29:00Z'
updated_date: '2026-11-05T17:29:00Z'
labels:
  - mod-support
dependencies:
  - TASK-1
documentation:
  - doc-2
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
- [ ] #1 Mods whose only shared files are sound event tables, and which change different events in them, can be on together, and each keeps its own events in the game
- [ ] #2 An event changed differently by two mods that are on is still a clash, as under TASK-1
- [ ] #3 Events a mod's copy leaves out or leaves unchanged from the game's copy come from the game's current copy, so an out-of-date table no longer removes other heroes' lines
- [ ] #4 The merged table is rebuilt when the mods that are on change or the game updates its own copy
- [ ] #5 Tests cover merging, a real conflict, an out-of-date mod table and a table that can't be read
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Checked in the game with Hatchling, Ted and Flowery on together: each hero's pick line plays
<!-- DOD:END -->
