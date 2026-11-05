---
id: TASK-30
title: Merge shared sound event tables that several mods change
status: Done
assignee:
  - '@claude'
created_date: '2026-11-05T17:29:00Z'
updated_date: '2026-11-05T20:07:00Z'
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
- [x] #1 Mods whose only shared files are sound event tables can be on together, unless an event is a real conflict (#2), and each keeps the events that use its own sounds
- [x] #2 An event that two mods that are on both change, each with sound files it ships, and differently, is still a clash as under TASK-1
- [x] #3 In a table that two or more mods that are on share: a changed event comes from a mod only if it refers to a sound file that mod ships; events a mod adds are taken; everything else comes from the game's current copy, so an out-of-date copy no longer removes or reverts other heroes' events. Numbers compare at KV3 text precision (6 decimals)
- [x] #4 A table that only one mod that's on has is used as the mod ships it, as now
- [x] #5 A mod whose changes a merge leaves out shows a muted information marker (MergeNotice) on its card and list row, with the tooltip "Some of its sound changes are left out so it works alongside %1" naming the mods it's merged with; a mod the merge leaves nothing out of shows nothing
- [x] #6 The merged table is rebuilt when the mods that are on change or the game updates its own copy
- [x] #7 Tests cover merging, a real conflict, an out-of-date mod table, a lone mod's table and a table that can't be read
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 Checked in the game with Hatchling, Ted and Flowery on together: each hero's pick line plays
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Write binary KV3 (kv3.Encode, v5 uncompressed like the game's own uncompressed files) and compiled files (resource.Write). Check in the game that it accepts a table Lockdeader writes.
2. Read the game's own copy of a table out of pak01's chunk archives (vpk can't yet).
3. The merge: per event, numbers compared at 6 decimals, the own sounds rule only when a table is shared, added events taken, real conflicts found, and which mods had changes left out.
4. Clash detection: for sound event tables, look inside (cached per VPK, as file lists are) and clash only on real event conflicts.
5. Deploy: merged tables in a generated VPK that loads ahead of the mods', in the manifest; rebuilt when the enabled mods or the game's copy change.
6. The marker on mods whose changes a merge leaves out (role, QML like ClashWarning, tooltip wording confirmed with the user, translations).
7. Check in the game with Hatchling, Ted and Flowery on together.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Survey of 125 mods (2026-11-05) in doc-4, Merging files that mods share: sound event tables are mergeable per event; vdata only for added entries; per-hero panorama CSS and localization later, with their own readers; HUD panorama and assets not. Compare numbers at KV3 text precision (6 decimals), since mods recompile decompiled tables. Open: telling a mod's own changes from a stale copy's.

Decided with the user 2026-11-05: the own sounds rule (doc-4) applies only when two or more mods that are on share a table; a lone mod's table is used as it is. A merge that leaves out some of a mod's changes shows a quiet marker on that mod (tooltip, no toast or info bar), only when something is actually left out. Wording of the tooltip still to be confirmed when it's built.

Step 1 done: kv3.Encode and resource.Write. Re-encoding the game's pick table decodes identically (ours and Source2Viewer's), and resource.Write reproduces an unchanged file byte for byte. In-game test (2026-11-05): the game's pick table re-encoded by Lockdeader, with Abrams' Pick/Remove/Avoided events pointed at Haze's files, in addons/pak50_dir.vpk via Play With Mods: Abrams said Haze's lines. So the game accepts KV3 v5 uncompressed tables Lockdeader writes, with the game's own RED2 block.

Step 2 done: vpk.Dir.ReadFile reads a file whole, from the base game's chunk archives too, checked against its CRC (the CRCs of all files in 20 mod VPKs checked were right). On the game: pak01's directory (158,644 entries) reads in about 100 ms, and all 160 sound event tables read and check in about 140 ms.

Step 3 done: package merge, SoundEvents. On the real mods (Hatchling hatchdanielupdated, Ted ted_2, Flowery Flowery_apollo) over the game's tables: no conflicts; the merged pick table has all 180 of the game's events with exactly Hatchling's 2 Deadpack, Ted's 2 Familiar and Flowery's 3 Fencer events changed, plus Flowery's stale Generated.Boho.Hero.Pick.VO added (an old hero name, harmless). Left out: Hatchling and Ted nothing; Flowery 112 pick events, 37 death events and Vo.VoteReveal, all its out-of-date versions of other heroes' events, so Flowery would show the marker.

Step 4 done: deploy.Clashes takes a mergeable callback; installedMods.merges answers it for .vsndevts_c by running merge.SoundEventConflicts on the pair over the game's copy (deploy.Base, which keeps pak01's directory until an update replaces it), cached per game version and the mods' VPKs. No game folder, or a copy that can't be read: they clash, as before. Conflicts now need both versions to use their mod's own sounds (two stale copies adding the same old event differently aren't a clash). vpk.Dir.ReadFile now opens the _dir.vpk by path for data kept in it, so no file stays open (an open pak01 would get in Steam's way on Windows). Switched off in the app until step 5: main.go doesn't set installed.gameFolder, since without merged tables in the game, letting such mods be on together would lose one mod's changes.

Step 5 done: deploy.MergeTables merges every .vsndevts_c that two or more mods that are on have different copies of, over the game's (deploy.Base), into one VPK; deployer.withMergedTables keeps it in the store by SHA-256 and puts it first in the load order (pak01), so it wins those tables and nothing else. Tables that can't be merged, a VPK that can't be read, or no game: logged, and left to the load order. Clash detection's merging (step 4) is switched on in main.go with it. Dry run with the real Hatchling, Ted and Flowery VPKs into a fake game folder linking the real pak01: pak01 held the merged death, pick and vote-reveal tables, then the three mods; a second apply placed nothing; left out: Flowery only.

Step 7 (in-game check) done 2026-11-05: Hatchling (hatchdanielupdated), Ted (ted_over_rem_2_) and Flowery (flowery_apollo) installed through the app all stayed on together; with Play With Mods, Rem and Apollo played their mods' lines, and Deadman Danny played Hatchling's lines for the 3 of 10 select lines (02, 03, 07) and 2 of 10 unselect lines it replaces, the rest vanilla, as the mod is made. The merged pak01 held Hatchling's versions of Danny's events. Without merging, Flowery's out-of-date table would leave Danny no pick line at all.

Step 6 done: MergeNotice (Fluent info_24_regular in Theme.subtle, on the same badge as ClashWarning over pictures) beside ClashWarning in InstalledGrid and InstalledList, from the mergedWith role; tooltip, as the user chose: "Some of its sound changes are left out so it works alongside %1". The deployer's merge returns what it left out per mod (deploy.Merged.LeftOut), and onMerged (installed.SetMerged) sets the role after each apply, even while the game runs; no game folder clears it. Headless run (offscreen, scratch data dir, fake game folder linking the real pak01) with the user's mods, in grid and list view: no QML warnings, 3 tables merged. Evidence for the criteria: #1 and #2 TestSharedTablesMerge and the in-game check; #3 merge tests and the real Hatchling/Ted/Flowery merge; #4 TestMergeTablesNothingToMerge; #6 every apply merges afresh over a newly read pak01 (TestBase covers noticing an update) and the same mods give the same VPK; #7 merge, deploy and root tests. #5 waits on the user seeing the marker in the app.

Follow-up from the user's look at the app (2026-11-05): (1) bug: installedMods worked out clashes while loading, before main.go gave it the game folder, so mods sharing tables showed as clashing after a restart; the folder now comes with newInstalledMods (TestSharedTablesMergeOnStartup). (2) The user found the gold clash warning on every clashing mod confusing for non-technical users, as clashing is a handled state. Decided: only a mod that's off because mods that are on change the same files shows a marker, ClashNotice (was ClashWarning), the same muted info icon as MergeNotice, tooltip "Can’t be on at the same time as %1: they change the same files"; a mod that's on shows nothing. Recomputed in installedMods.changed after every change to which mods are on. warning_24_regular.svg removed (unused). doc-2 updated.

The user checked the markers in the app (2026-11-05): only Flowery shows MergeNotice; no clash markers on mods that merge.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Mods that clash only over shared sound event tables can now be on together. Each apply merges the tables that two or more mods that are on have different copies of, over the game's own copy, into one VPK loaded first (pak01), so it wins those tables and nothing else; a lone mod's table, or identical copies, are used as they are. In a shared table, a changed event comes from a mod only if it uses a sound file the mod ships (so out-of-date copies, common on GameBanana, can't revert or remove the game's events), added events are taken, and numbers compare to six decimals, since mods recompile tables decompiled to text. Clash detection asks the same merge per pair, so only real conflicts (two mods changing one event with their own sounds) still turn a mod off.

Built in steps: kv3.Encode and resource.Write (TASK-30.1 read the formats); vpk.Dir.ReadFile reads the game's chunks; package merge; deploy.Base, deploy.Clashes' mergeable callback and installedMods.merges; deploy.MergeTables and deployer.withMergedTables (kept in the store by SHA-256, so an unchanged apply rewrites nothing); MergeNotice (muted info marker, tooltip "Some of its sound changes are left out so it works alongside %1") from the deployer's onMerged. From the user's review, clashes became a calm ClashNotice on the mod that's off only, no more gold warnings, and a startup ordering bug that showed merged mods as clashing was fixed.

Verified: unit tests across kv3, resource, vpk, merge, deploy and the app (each failing when its logic is broken on purpose); go vet for Linux and Windows; a dry run into a fake game folder with the real mods; in-game tests: a table written by Lockdeader was loaded (Abrams with Haze's lines), and Hatchling, Ted and Flowery on together each played their lines; the user checked the markers in the app.

Follow-ups, not done: other table kinds from doc-4 (entries mods add to vdata, per-hero panorama CSS, localization) would be new tasks; TASK-31 (VPKs vpk.ReadDir turns down) affects which mods can be read for clashes and merging; the first clash check after startup reads pak01's directory on the UI thread (about 0.1 s).
<!-- SECTION:FINAL_SUMMARY:END -->
