---
id: TASK-11
title: Attach the Windows zip to a GitHub Release on version tags
status: In Progress
assignee:
  - '@claude'
created_date: '2026-11-05T09:33:00Z'
updated_date: '2026-11-05T14:20:00Z'
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
- [ ] #1 A `v*` tag zips the Windows app folder as `Lockdeader-<version>-windows-x64.zip` and attaches it, with a `.sha256` file, to the release for that tag, or to a new draft release when there is none
- [ ] #2 The Flatpak bundle, with its `.sha256` file, is attached to the same release
- [ ] #3 The release job fails if the tag does not match `VERSION.txt`
- [ ] #4 Tested with real tags on the repository, which are removed afterwards
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. release job in build.yml: on v* tags, after windows, windows-smoke-test and flatpak; contents: write only there. First step fails unless the tag is v + VERSION.txt.
2. Download both artifacts, zip the Windows folder (contents at the top), write <file>.sha256 for each.
3. gh release upload --clobber to the tag's release if it exists, else gh release create --draft --verify-tag --generate-notes with the files.
4. Docs: doc-3 release steps, packaging/windows/README.md, CHANGELOG.
5. Test on a throwaway branch: bump VERSION.txt, push a matching test tag (draft gets files, checksums verify), push a mismatched tag (job fails); delete tags and drafts.
<!-- SECTION:PLAN:END -->
