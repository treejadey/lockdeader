---
id: TASK-30.1
title: Read Source 2 resources and binary KV3
status: Done
assignee:
  - '@claude'
created_date: '2026-11-05T17:32:00Z'
updated_date: '2026-11-05T18:27:00Z'
labels:
  - mod-support
dependencies: []
parent_task_id: TASK-30
priority: medium
type: feature
ordinal: 31000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
TASK-30 merges sound event tables, which are compiled Source 2 resources (`.vsndevts_c`) whose DATA block is binary KV3, so Lockdeader first has to read them. Mods ship tables built by different versions of the tools, so the reader has to cope with the KV3 versions found in real mods, not only the current game's (v5, LZ4).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 A compiled resource's blocks can be listed and read by name
- [x] #2 Binary KV3 in the versions found in the game and in mods decodes to a tree of typed values (objects keep their key order)
- [x] #3 Decoding matches Source2Viewer-CLI's text output on the game's generated_vo tables and on mods' copies
- [x] #4 Damaged or unsupported data returns an error instead of panicking or reading out of bounds
- [x] #5 Tests build their data in code (no game or mod files in the repository)
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. resource/: read a compiled file's header and block list, by the layout ValveResourceFormat reads.
2. kv3/: decode binary KV3 v1-v5 (header per version, lanes and alignment, strings, types and flags, typed and auxiliary arrays, blobs; uncompressed, LZ4 incl. chained blob frames, Zstandard) into Value trees whose objects keep key order; errors via a recovered sentinel panic with bounds checks, a depth limit and a node budget.
3. kv3 text writer that lays out like Source2Viewer, to diff against it.
4. Tests: a builder that lays out each version byte by byte; damaged data; fuzz.
5. Check against Source2Viewer on the game's files and the mods' tables.

6. Added version 0 (VKV3) once the survey found it in mods' files.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
resource and kv3 packages written. Matches Source2Viewer-CLI 20.1 exactly (apart from format names in the header line, which kv3 doesn't keep a table of) on: the game's and Hatchling/Ted/Flowery's generated_vo_hero_pick_vo tables; a sample of 194 of the game's compiled files, 8 of each type (138 DATA blocks that are KV3, all 194 RED2 blocks). That sample covers KV3 v4 and v5, uncompressed, LZ4, Zstandard (RED2), and blobs with LZ4 and uncompressed. v1-v3 and Zstandard blobs are covered only by the builder tests. Fuzzed 2 min (13M inputs), no crashes.

Added version 0 (VKV3, binary and LZ4 encodings) after the survey found it in 11 mod files; all 11 match Source2Viewer. Every KV3 block decodes: 327,063 in the game's pak01 (v4, v5) and 13,518 in 125 surveyed mods (v0, v4, v5). The node budget first rejected 11 FLCI blocks with large typed arrays; it now counts all the data's bytes. Fuzzed again after the changes; no crashes.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Added package resource (the blocks of a compiled _c file) and package kv3 (binary KeyValues3 v0-v5, uncompressed, LZ4 or Zstandard, into a tree of typed values that keeps key order, plus KV3 text laid out like Source2Viewer's). This is what TASK-30 needs to read the sound event tables mods share.

Verification: output matches Source2Viewer-CLI 20.1 on a 194-file sample of the game's compiled files, the four pick tables, and the 11 version-0 mod files (apart from format names in the header line, which kv3 doesn't keep a table of). Every KV3 block in the game (327,063) and in 125 surveyed mods (13,518) decodes. Tests build every version byte by byte (build_test.go), cover damaged and cut-off data, and fail when the decoder is broken on purpose. FuzzDecode ran about 15M inputs without a crash.

Not done: writing KV3 (TASK-30), and Valve's block compression for version 0, which no Deadlock file uses. pierrec/lz4 and klauspost/compress went from indirect to direct dependencies.
<!-- SECTION:FINAL_SUMMARY:END -->
