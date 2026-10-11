// Where an item view is scrolled to, as the first item on screen and how far
// past its top the view is. Unlike contentY, it still points at the same item
// when the window's width changes the layout, e.g. how many columns a grid
// has.

function top(view) {
    // Just inside the view's top left corner, which is in the first column.
    const index = view.indexAt(1, view.contentY + 1)
    const item = index >= 0 ? view.itemAtIndex(index) : null

    if (!item)
        return { index: -1, offset: 0 }

    return { index: index, offset: Math.round(view.contentY - item.y) }
}

function restore(view, index, offset) {
    if (index < 0 || index >= view.count)
        return

    // Back to the very top, margin and all.
    if (index === 0 && offset <= 0) {
        view.positionViewAtBeginning()

        return
    }

    view.positionViewAtIndex(index, ListView.Beginning)
    view.contentY += offset
}
