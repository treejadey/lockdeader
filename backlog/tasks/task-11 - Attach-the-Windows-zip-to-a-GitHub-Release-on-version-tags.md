---
id: TASK-11
title: Attach the Windows zip to a GitHub Release on version tags
status: Done
assignee:
  - '@claude'
created_date: '2026-11-05T09:33:00Z'
updated_date: '2026-11-05T16:05:00Z'
labels:
  - distribution
dependencies: []
documentation:
  - doc-3
priority: high
type: chore
ordinal: 11000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
CI already builds the Windows app folder on every push and uploads it as a workflow artifact, but artifacts expire and need a GitHub login. What users download should come from a tagged release (doc-3, Principles). To release, `VERSION.txt` is changed and the commit tagged `v` plus that version.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 A `v*` tag zips the Windows app folder as `Lockdeader-<version>-windows-x64.zip` and attaches it to the release for that tag, or to a new draft release when there is none
- [x] #2 The Flatpak bundle is attached to the same release
- [x] #3 The release job fails if the tag does not match `VERSION.txt`
- [x] #4 Tested with real tags on the repository, which are removed afterwards
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. release job in build.yml: on v* tags, after windows, windows-smoke-test and flatpak; contents: write only there. First step fails unless the tag is v + VERSION.txt.
2. Download both artifacts, zip the Windows folder (contents at the top), write <file>.sha256 for each.
3. gh release upload --clobber to the tag's release if it exists, else gh release create --draft --verify-tag --generate-notes with the files.
4. Docs: doc-3 release steps, packaging/windows/README.md, CHANGELOG.
5. Test on a throwaway branch: bump VERSION.txt, push a matching test tag (draft gets files, checksums verify), push a mismatched tag (job fails); delete tags and drafts.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Dropped the .sha256 files (asked 2026-11-05): GitHub shows each release asset's SHA-256 next to it.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
v* tags now publish their builds. The release job (after the Windows build, its smoke test and the Flatpak pass) zips the Windows folder with packaging/release-files.ps1 and attaches Lockdeader-<version>-windows-x64.zip and the Flatpak to the tag's release with gh release upload --clobber, or creates a draft (--generate-notes) when there's none. No .sha256 files: GitHub shows each asset's SHA-256. A separate Version job checks the tag against VERSION.txt before anything builds.

Tested 2026-11-05 with real tags: v0.2.99 (matching) created a draft with both files, and on a rerun updated the files of that existing draft; v0.2.98 (mismatched) failed in the Version job in 10 s with every other job skipped. The smoke test's app reported 0.2.99 without "Nightly". All test tags and releases were deleted (git push --delete hit a GitHub server error for two tags; the REST API deleted them).
<!-- SECTION:FINAL_SUMMARY:END -->
