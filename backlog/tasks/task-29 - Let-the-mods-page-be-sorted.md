---
id: TASK-29
title: Let the mods page be sorted
status: Done
assignee:
  - '@claude'
created_date: '2026-11-05T16:21:00Z'
updated_date: '2026-11-07T15:35:00Z'
labels:
  - ui
dependencies: []
priority: medium
type: feature
ordinal: 29000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The mods ("store") page always lists mods in GameBanana's default order, newest first, so there's no way to find the most popular or recently updated mods. banana.ListModsOptions already takes a Sort (banana/mods.go: SortNewest, SortLatestUpdated, SortMostLiked, SortMostViewed, SortMostDownloaded); modmodel.go just never sets it. Check whether the API keeps the sort when a name search or a category is applied. How the choice is offered (control, wording, where it sits) is a UI decision to ask about when the task starts (docs/interface-guidelines.md).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 The mods page offers each sort order the GameBanana client supports, except Recently Updated, which hides never-updated mods (decided with the user), and changing it reloads the list in that order
- [x] #2 The chosen order applies to every category and to searches, wherever the API allows it
- [x] #3 The chosen order is remembered across restarts, like the page's other state (appdb.UIState)
- [x] #4 Tests cover the sort reaching the API request
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Decided with the user (2026-11-07): a drop-down in the mods page's top bar, beside the grid/list switcher, offering Recent Activity (GameBanana's default, what the page shows now: new uploads and updates mixed), Newest, Most Liked, Most Viewed and Most Downloaded. Not Recently Updated: it hides mods never updated (2,318 of 5,802), and Recent Activity already brings updates up. Checked against the API (2026-11-07): every order applies with a category, a name search and in Sounds.
1. modmodel.go: a sort, sent with each page; SetSort reloads.
2. search.go: "sort" (QML writes it) and "sorts" ({id, name} for the drop-down, names through tr); unknown ids are ignored; changing it closes the open mod like a new search.
3. appdb.UIState.ModsSort (ui.mods_sort), the uiState bridge, and main.go restoring it before the category, so the saved scroll position is looked for in the same order.
4. QML: a DropDown component (button showing the current choice and a chevron, opening a PopupMenu with the current item ticked) in TopBar, shown on the mods page.
5. Tests: the sort reaches the request (search bridge and model), with a category and a search; UIState round trip; an unknown sort is ignored.
6. Check at 1.25 scale in both themes.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Checked (2026-11-07) in the nested KWin session at 1.25 scale, both themes (dark forced in a throwaway build): the drop-down sits left of the grid/list switcher, or of the volume on Sounds; the menu ticks only the current order; picking Most Liked reloaded the list (log: sort=Generic_MostLiked), it carried over to Sounds, and the next start opened in it. The window's minimum width went from about 1113 to 1224 logical px, mostly because the Sounds page's volume control, beside the drop-down there, is wide. At that width the search field moves left of centre to clear the controls.

Changed with the user (2026-11-07): the drop-down follows Carbon's dropdown rather than the pill of the grid/list switcher: a field like SearchBox (top corners 4 px, bottom line, 2 px foam while open or focused), the chevron turning up while open, and the list under it at the same width, square, 40 px options with inset separators, the current one highlighted and ticked on the right; arrow keys, Enter and Esc work in it. Checked in both themes at 1.25 scale. Window minimum width now about 1234.

The chevron's side has 12 px of padding rather than 16, as its glyph has room around it in its icon (asked by the user, 2026-11-07).
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
The mods page's top bar has a sort drop-down (new DropDown component: a pill showing the order, opening a PopupMenu with the current one ticked) offering Recent Activity (GameBanana's own order, the default, as before), Newest, Most Liked, Most Viewed and Most Downloaded. Not Recently Updated, which leaves out every mod that was never updated. The order is sent with every page of every listing, so it applies in any category, in Sounds and with a search (checked against the live API), and changing it reloads the list and closes the open mod, as a new search does. It's remembered across restarts (appdb.UIState.ModsSort, ui.mods_sort) and restored before the category, so the saved scroll position is looked for in the same order; an unknown value falls back to GameBanana's order.

The search field stays centred while there's room and moves left to clear the controls on the right when there isn't. The window's minimum width grows by about 111 px (to about 1224).

Tests: TestModSearchSort (the sort in the request, with a category and a search; an unknown one ignored; GameBanana's own sends none; remembered), the UIState round trip. go test ./..., go vet, gofmt and the translations check pass.
<!-- SECTION:FINAL_SUMMARY:END -->
