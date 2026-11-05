---
id: TASK-12
title: Ship third-party licence notices with every build
status: In Progress
assignee:
  - '@claude'
created_date: '2026-11-05T09:33:00Z'
updated_date: '2026-11-05T10:41:00Z'
labels:
  - distribution
dependencies: []
documentation:
  - doc-3
priority: high
type: chore
ordinal: 12000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Lockdeader is GPL-3.0 and bundles software under other licences, which must travel with the app: Qt (LGPL-3.0, dynamically linked so users can swap it), the fonts (OFL), the Fluent icons (MIT) and miniaudio (public domain or MIT-0). Nothing lists them yet. Go module dependencies need the same treatment.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 A `THIRD-PARTY-NOTICES` file (or licences folder) lists every bundled component with its licence text
- [ ] #2 The Windows zip and the Flatpak both include it
- [x] #3 Go module dependencies compiled into the binary are covered
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Decision (asked 2026-11-05): generated at build time, not committed.

1. cmd/notices: `go run ./cmd/notices -o FILE` writes THIRD-PARTY-NOTICES.txt for the target GOOS/GOARCH: the Go standard library, every module `go list -deps` says is compiled into the app (with CGO_ENABLED=1), with every LICENSE/COPYING/NOTICE/PATENTS file in the module (fails if a module has none), and the vendored components (Inter, JetBrains Mono, Fluent icons, miniaudio, stb_vorbis from the end of its source file). Test for linux and windows.
2. Windows build.ps1: run it into the app folder; map every bundled DLL back to its MSYS2 file, ask pacman which packages own them, copy each package's share/licenses files into licenses/<package>/ and append a section listing package, version and licence, with where its source is (LGPL).
3. Flatpak manifest: install the generated file next to LICENSE in /app/share/licenses/<app-id>/.
4. Docs: packaging READMEs, doc-3, CLAUDE.md packaging paragraph.
5. Verify: notices for both GOOS locally, a Flatpak build, and the Windows job on CI (no pwsh/MSYS2 here).
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Built as planned. cmd/notices (go run ./cmd/notices -o FILE) writes the notices for the target GOOS: Go standard library (LICENSE/PATENTS kept in cmd/notices/golang, since Fedora moves them out of GOROOT; TestGoLicenceUpToDate compares with GOROOT where it has them), every module go list -deps says is compiled in (all LICENSE/COPYING/NOTICE/PATENTS/AUTHORS files; fails on a module without one), and the vendored fonts, icons, miniaudio and stb_vorbis. build.ps1 runs it, maps every bundled DLL back to MSYS2 (ucrt64/bin, share/qt6/plugins or share/qt6/qml), asks pacman -Qqo for the owning packages, copies their share/licenses files into licenses/ and appends a section with version, licence and source location. The Flatpak manifest installs it next to LICENSE.

Verified: go test ./... (TestNotices for linux and windows checks the right modules are in and build tools are out); local Flatpak build has /app/share/licenses/<app-id>/THIRD-PARTY-NOTICES.txt (217 KB, 33 components). build.ps1 parses in PowerShell 7 (container) and its pacman -Qi parsing and path mapping were exercised on sample output. Not yet run: the Windows CI job, so AC 2 (zip includes it) is unchecked until it has.
<!-- SECTION:NOTES:END -->
