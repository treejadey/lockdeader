---
id: TASK-28
title: Publish a nightly release from every successful main build
status: Done
assignee:
  - '@claude'
created_date: '2026-11-05T14:20:00Z'
updated_date: '2026-11-05T16:05:00Z'
labels:
  - distribution
dependencies:
  - TASK-11
priority: high
type: feature
ordinal: 28000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
People who want the newest Lockdeader without building it need a download that tracks main. Copied from Chatterino2 (decided 2026-11-05): after every push to main whose build, tests and Windows smoke test pass, a pre-release called "Nightly Release" under the tag `nightly-build` gets that build's files, replacing the previous ones; its notes list the commits since the last v* tag (the 5 newest, the rest folded away, like Chatterino's .CI/format-recent-changes.py); and the `nightly-build` tag is force-moved to the commit. Chatterino uses ncipollo/release-action with allowUpdates, replacesArtifacts and prerelease. Like Chatterino's NIGHTLY_BUILD, builds that aren't from a v* tag say they're nightlies in the app.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Each successful push to main replaces the files of the "Nightly Release" pre-release (tag `nightly-build`) with that build's Windows zip and Flatpak, under names that don't change between builds
- [x] #2 The release notes list the commits since the last v* tag, newest first, with all but the 5 newest folded away
- [x] #3 The `nightly-build` tag points at the commit the files were built from
- [x] #4 Builds not made from a v* tag show "Nightly" after the version in the window title and Settings › Advanced; tag builds and local builds look as now
- [x] #5 The Flatpak shows the commit it was built from in Settings › Advanced, as the Windows build does
- [x] #6 Tested on GitHub before it reaches main, with the test release and tag removed afterwards
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. App: buildType and buildCommit set by -ldflags -X (empty for local builds). Nightly builds show "<version> Nightly" in the title and buildInfo; buildCommit fills in the commit when Go has no VCS stamp (the Flatpak).
2. packaging/windows/build.ps1 -Nightly adds the -X flags. packaging/flatpak/build.ps1 -Nightly writes a git-ignored packaging/flatpak/build-flags that the manifest's go build reads (flatpak-builder can't pass variables in); always writes the commit.
3. build.yml: pass -Nightly unless the ref is a v* tag (Chatterino's "auto"); concurrency group per ref with cancel-in-progress, as Chatterino.
4. nightly job (Chatterino's create-release): on push to main, after all build jobs; checkout with fetch-depth 0 and persist-credentials; download artifacts; rename to Lockdeader-nightly-*; .sha256 files; packaging/nightly-notes.ps1 (port of format-recent-changes.py); ncipollo/release-action pinned as Chatterino (allowUpdates, replacesArtifacts, artifactErrorsFailBuild, prerelease, name Nightly Release, tag nightly-build); git tag -f nightly-build and push -f.
5. Docs: doc-3, CLAUDE.md packaging, CHANGELOG.
6. Test on the branch with a temporary commit that runs the nightly job there under a test tag, then drop the commit and delete the test release and tag.
<!-- SECTION:PLAN:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Copied Chatterino2's nightly. The nightly job (push to main, after all builds pass) makes Lockdeader-nightly-windows-x64.zip and Lockdeader-nightly-linux-x86_64.flatpak, writes notes with packaging/nightly-notes.ps1 (port of format-recent-changes.py: commits since the last v* tag, 5 shown, the rest in <details>), updates the "Nightly Release" pre-release with ncipollo/release-action pinned as Chatterino (allowUpdates, replacesArtifacts, prerelease, tag nightly-build), then force-moves the nightly-build tag. Workflow-level concurrency per ref (cancel-in-progress) keeps an older build from replacing a newer nightly.

In the app: builds not from a v* tag pass -Nightly to the build scripts, which set buildType with -ldflags -X; displayVersion() adds " Nightly" in the title, startup log and Settings › Advanced. The Flatpak gets buildCommit the same way through a git-ignored packaging/flatpak/build-flags file (flatpak-builder can't pass values in), and vcsRevision falls back to it.

Tested: unit tests (TestDisplayVersion, TestVCSRevisionFallback); local Flatpak build has the commit in its binary; a nightly-test branch run (temporary commit pointing the job at nightly-build-test) created the pre-release with both files and notes, a rerun replaced the files, the tag followed the commit, and its smoke test reported "0.2.0 Nightly". Test release, tag and branch deleted. The first real nightly appears with the first push to main that includes this.
<!-- SECTION:FINAL_SUMMARY:END -->
