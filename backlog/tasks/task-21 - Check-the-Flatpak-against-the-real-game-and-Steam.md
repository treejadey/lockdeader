---
id: TASK-21
title: Check the Flatpak against the real game and Steam
status: To Do
assignee: []
created_date: '2026-11-05T09:33:00Z'
labels:
  - distribution
dependencies: []
documentation:
  - doc-3
priority: medium
type: task
ordinal: 21000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Two Flatpak paths were written without the real game to test against. Inside a Flatpak, `deploy.Running` runs `flatpak-spawn --host pgrep -xi deadlock.exe`, which assumes the process name under Proton is `deadlock.exe` (the `/proc` code also accepts the first word of the command line, which `pgrep -x` won't match). Launching uses the `steam://` URL through the OpenURI portal, which may behave differently for native and Flatpak Steam.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 With Deadlock running under Proton, the Flatpak's game status shows it running, and shows it stopped after it closes
- [ ] #2 Play With Mods starts the game from the Flatpak with native Steam and with Flatpak Steam
- [ ] #3 Any fix needed is made, or the findings are recorded in doc-3
<!-- AC:END -->
