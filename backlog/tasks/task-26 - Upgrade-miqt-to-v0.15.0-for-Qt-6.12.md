---
id: TASK-26
title: Upgrade miqt to v0.15.0 for Qt 6.12
status: Done
assignee:
  - '@claude'
created_date: '2026-11-05T11:05:00Z'
updated_date: '2026-11-05T13:32:00Z'
labels:
  - distribution
dependencies: []
priority: high
type: chore
ordinal: 26000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
MSYS2 moved to Qt 6.12.0 (seen 2026-11-05), whose headers make the QAccessible*Interface classes non-copyable, so miqt v0.14.0's gen_qaccessible.cpp no longer compiles and every Windows CI build fails (run 42013677048). miqt v0.15.0 fixes it (mappu/miqt#359, #360). CLAUDE.md documents several v0.14.0-specific workarounds (roleNames override, translate override, heap-corrupting list-of-pairs setters) that need checking against the new version.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 go.mod requires miqt v0.15.0 and the app builds and passes go test ./... against Qt 6.11 locally
- [x] #2 The Windows CI job builds against MSYS2's Qt 6.12 and its smoke test passes
- [x] #3 CLAUDE.md's notes on miqt v0.14.0 workarounds are checked against v0.15.0 and updated where they changed
<!-- AC:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Upgraded miqt v0.14.0 → v0.15.0, which supports Qt 6.12 (MSYS2 moved to it, breaking every Windows build in gen_qaccessible.cpp; mappu/miqt#359, #360).

- go.mod/go.sum bumped. go test ./... passes locally against Qt 6.11; the app starts and loads QML headless, behaving the same as a v0.14.0 build on the same data.
- CI run 42034966312: Windows build against Qt 6.12, its clean-machine smoke test and the Flatpak all pass.
- v0.15.0 fixes the virtual-override use-after-free (mappu/miqt#338) behind the roleNames and translate workarounds; CLAUDE.md, modmodel.go and translator.cpp now say so. The workarounds still work and were kept. The list-of-pairs setter bug is still in v0.15.0 (8 bytes allocated per 24-byte pair), so that warning stays.
- Also unpinned the miqt version in TestNotices.

Possible follow-up (not created): replace the roleNames/translate workarounds with plain overrides now that v0.15.0 allows it.
<!-- SECTION:FINAL_SUMMARY:END -->
