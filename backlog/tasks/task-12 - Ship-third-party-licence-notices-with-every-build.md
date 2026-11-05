---
id: TASK-12
title: Ship third-party licence notices with every build
status: To Do
assignee: []
created_date: '2026-11-05T09:33:00Z'
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
- [ ] #1 A `THIRD-PARTY-NOTICES` file (or licences folder) lists every bundled component with its licence text
- [ ] #2 The Windows zip and the Flatpak both include it
- [ ] #3 Go module dependencies compiled into the binary are covered
<!-- AC:END -->
