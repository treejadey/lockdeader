---
id: DRAFT-1
title: Run our own API that indexes mod contents
status: Draft
assignee: []
created_date: '2026-11-05T09:34:00Z'
labels:
  - mod-support
dependencies: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Insights from mod contents (hero, type, gameplay data, conflicts, breakage after game updates) only help while browsing if someone has already downloaded and indexed the mods. A small service could index the VPK file lists of GameBanana's Deadlock files (paths and CRCs, which is small), serve per-mod data, compare file lists against each game build to flag likely-broken mods, and show conflicts against a user's installed mods before they download. DMM's backend does something similar (`/api/v2/vpk-analyse-hashes`). It could also cache in front of GameBanana's servers to lighten their load. See doc-2, "Future: our own API".
<!-- SECTION:DESCRIPTION:END -->
