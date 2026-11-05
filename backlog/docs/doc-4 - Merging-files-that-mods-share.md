---
id: doc-4
title: Merging files that mods share
type: specification
created_date: '2026-11-05T18:27:00Z'
updated_date: '2026-11-05T18:47:00Z'
---
Which files that several mods ship could be merged rather than kept apart (TASK-1 turns all but one off), and what merging them takes. Written 2026-11-05 for TASK-30, from a survey of 125 mods. Raw data was kept outside the repository; the findings below are what matters.

## How often mods clash, and over what

The sample was 138 mods from GameBanana (most downloaded, newest, and the top and newest of every Mod and Sound category). Of those, 125 were read: 143 VPKs and 21,069 distinct paths.

- 2,142 paths are shipped by two or more mods with different contents.
- 188 pairs of mods clash, and 87 of the 125 mods are in at least one clash.
- **78 of the 188 pairs (41%) clash only over table files**: sound event tables, `scripts/*.vdata_c`, panorama styles and layouts, and localization. The rest clash over a hero's own assets (sounds, card images, models, particles). Those are real conflicts; two copies of one model can't be combined.
- The table files most often behind a clash, in order: `panorama/styles/citadel_hero_tags.vcss_c` and `citadel_hero_abilities.vcss_c` (14 pairs each), `scripts/abilities.vdata_c` (10), `soundevents/player.vsndevts_c` (9), `panorama/layout/citadel_hud_top_bar.vxml_c` (6), `soundevents/music.vsndevts_c` (6), `soundevents/generated_vo_hero_pick_vo.vsndevts_c` (5).

## What can be merged

| Files | What they are | Merge? |
|---|---|---|
| `soundevents/**/*.vsndevts_c` | Compiled KV3 (`kv3`): a map of independent sound events, e.g. `Generated.<Hero>.Hero.Pick.VO` or `<hero>_<line>_3d` | **Yes, per event.** Voice mods change only their own hero's events. Real conflicts are rare: `Player.Bonk` (two mods), `drifter_ping_rat_strat_ping_2d` (two mods), and one skin that rewrites all 1,736 Ratking voice events. |
| `scripts/*.vdata_c` (abilities, heroes, generic data) | Compiled KV3: a map with one entry per ability or hero, plus `_include` and editor-only `_editor` blocks | **Only entries a mod adds.** Mods that change entries often carry old balance values in the same entries (601444 mixes its own fields with values from an older patch), so taking a mod's entry would revert the game's changes. `_editor` blocks must be ignored when comparing: some copies have them and some don't. |
| `panorama/styles/citadel_hero_tags.vcss_c`, `citadel_hero_abilities.vcss_c` | Compiled panorama CSS: one rule per hero, keyed by selector | **Yes, per rule, later.** Skins change only their own hero's rule, and none overlap. This needs a reader for compiled panorama CSS (resource version 3, not KV3). |
| HUD panorama: `hud.vcss_c`, `citadel_hud_top_bar.vcss_c`, `unit_status_v2.vcss_c`, `panorama/layout/*.vxml_c` | Style sheets and layout trees | **No.** HUD mods rewrite 40–100% of the rules, overlapping by 110–500 rules, and the rules depend on each other. Layouts are trees of inserted elements. |
| `resource/localization/**/*.txt` | KeyValues 1 text, `lang { Tokens { … } }` | **Yes, per token, later.** Mods change different tokens, except two gameplay modes that add the same `upgrade_*` ones. This needs a KV1 reader. It's unverified whether the game reads localization from a mod's VPK at all: the game's own copies are loose files. |
| Everything else: `sounds/**.vsnd_c`, textures, models, particles, animation | Assets | **No.** Whole files; a clash is real. |

## How to merge a table

These rules follow from what the mods' copies actually look like.

- **Compare against the game's current copy**, read from `pak01` at every apply, so a game update is picked up. In the survey, 10 or more mods shipped out-of-date copies of tables, missing whole heroes' events or rules. Example: Flowery - Apollo ships a pick table of 116 events where the game has 180.
- **Compare numbers at the precision KV3 text keeps (six decimals)**, not bit for bit. Mods appear to be made by decompiling the game's file to text, editing it and compiling it again, which turns the game's 5.5773921012878418 (5.577392 as a single-precision float) into exactly 5.577392. Compared bit for bit, mod 664127 changes 173 of 180 pick events; compared as text, it changes 6.
- **An event the mod's copy lacks comes from the game.** Missing events are almost always staleness, not deletion.
- **An event the mod adds is taken**, whatever it refers to. QOL Lock (650634) adds 3,092 events that play the game's own sounds. Events only a stale copy has are old names nothing plays, so taking them does no harm.
- **An event two mods change differently is a clash**, as now.
- **A changed event is the mod's only if it refers to a sound file the mod ships** (the "own sounds" rule). An out-of-date copy also has events that differ from the game's in ways the mod didn't mean, and taking those reverts the game's changes. For example, mod 664127 changes 3 Nano events on purpose, but its copy also has old versions of Artist's pick event and Ratking's and Viscous's reveal events.

### What the own sounds rule does to the surveyed mods

This was measured on all 118 sound event tables in the survey, against the game's copies. 68 of the 70 tables mods ship exist in the game.

- **Kept:** 4,911 changed events refer to the mod's own sounds and are taken.
- **Dropped:** 3,136 changed events don't. Almost all of them are stale:
  - 83629 (Yes King, Abrams voice) ships an old copy of Holliday's voice table, not Abrams', with 1,463 changed events. Today it reverts Holliday's lines.
  - 85147 (Frank) has 1,075 changed events, and 664127 has 377 in `generated_vo_misc`.
  - Smaller stale copies have old `base` templates (`Base.Movement` where the game now has `Base.Movement.Slide`), old `context_name` values, and lists of the game's files from an earlier patch.
- **Lost with them:** deliberate edits that involve no file of the mod's own. These are a few dozen events in about a dozen mods:
  - **Template tweaks.** 8 of the 55 sound mods set `Base.Ability`'s pitch range to 1.0–1.0 in `soundevents/base/citadel_base_events.vsndevts_c`, so the game stops pitching their ability sounds at random between 0.85 and 1.5. Under the rule their sounds get the random pitch back.
  - **Muted lines.** 736655 (Cockroach King) empties `vsnd_files` of 10 Ratking voice lines. Under the rule those lines play again.
  - **Volume and timing changes to the game's own sounds.** These are mixed in with stale changes in the same copies and can't be told apart, e.g. 87879's 31 music events and 664127's 15 Nano events.
- **Stale fields still come through own events, as they do today:** the rule takes an own event whole. 383 own events have an old `context_name` (306 of them Frank's), as Flowery's Apollo events do ("fencer" for "hero_fencer"). Taking only the file fields of own events wouldn't fix this without a loss: 664127 deliberately adds a `volume` to 1,410 of its own events.
- **"Trust copies that have every event the game has" doesn't work as an alternative.** It would keep the template tweaks and muted lines, but also let through stale changes: the old `Stinger.Rejuvinator.Descent` in two music mods, and 664127's 377 `generated_vo_misc` events.
- **Decided (2026-11-05): the rule applies only when two or more mods that are on share a table.** A lone mod's table is used as it ships, deliberate edits and staleness alike, so turning merging on makes no mod worse than before. Applying it always would also have fixed lone stale mods (Flowery alone still removes 65 of the pick table's 180 events), at the cost of their deliberate edits.
- **Decided (2026-11-05): a mod whose changes a merge leaves out shows a quiet marker**, an information icon on its card and list row where `ClashWarning` sits, whose tooltip names the mods it's merged with. There's no toast or info bar. The marker shows only when something of the mod is actually left out, so two voice mods for different heroes show nothing.

## Formats seen

- **Sound event tables** are resource version 1 with KV3 in `DATA`. The game's copies are KV3 v5 with LZ4, and one uses Zstandard. Mods' copies are v5 with LZ4 (113 files) or Zstandard (1), or v4, uncompressed (1) or with LZ4 (3).
- **`vdata_c`** is resource version 0 with an `FLCI` block, KV3 v5, uncompressed, LZ4 or Zstandard.
- **Across all compiled files,** the game has KV3 v4 and v5. Mods also have v0 (`VKV\x03`, 11 particle and animation files from two mods, all uncompressed). `kv3.Decode` reads all of them and matches Source2Viewer's output.
- **Panorama files aren't KV3:**
  - `vcss_c` is resource version 3.
  - `vjs_c` and `vts_c` are version 4, with source text in `DATA`.
  - `vxml_c` is version 3, with a binary `DATA` block and a `LaCo` block.
- **Writing a table back** isn't done yet. Whether the game accepts a table written in another KV3 version or without compression is untested.

## Found along the way

- **Lockdeader can't install 20 of the 132 archives tried,** because `vpk.ReadDir` turns down two kinds of VPK. Neither is tracked yet.
  - **Version-1 VPKs (15 archives).** These are not only old sound mods; 6 are recent uploads, e.g. 736394, Roster Tools.
  - **An empty file listed at offset `0xFFFFFFFF` (5 archives, all skins).** The file is an empty `readme.txt` with length 0, and it fails the bounds check. Example: 628247, 50 Cent Warden.
- **The noise list needs nothing new.** DeadlockForge's `deadlockforge/made_with_dlf.txt` is in 10 mods, with the same contents each time, so ignoring identical copies already covers it. One mod ships its whole `node_modules/` folder, but no other mod has those files.
- **During the survey the local game was one patch behind live** (6848 against 6851). Some differences a few mods' tables appear to have from the local game are that patch, not the mods.
