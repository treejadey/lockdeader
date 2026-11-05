---
id: TASK-20
title: Decide how the Flatpak reaches Steam libraries on other drives
status: To Do
assignee: []
created_date: '2026-11-05T09:33:00Z'
labels:
  - distribution
dependencies: []
documentation:
  - doc-3
priority: medium
type: spike
ordinal: 20000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The Flatpak has static permissions for Steam's usual folders in the home directory, but libraries on other drives (`/mnt`, `/run/media`, ...) are unreachable. A folder picked through the portal arrives as a document portal path (`/run/user/.../doc/...`), where hard links fail and `deploy` falls back to copying. Flathub reviewers will question broad filesystem permissions.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Which locations get static permissions is decided and recorded in doc-3, with the reason next to each in the manifest
- [ ] #2 `game.Check`, `deploy.ReadGameInfo`, apply and the addons scan are verified to work through a document portal path
<!-- AC:END -->
