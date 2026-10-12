import QtQuick
import QtQuick.Controls.Fusion

import "ScrollPosition.js" as ScrollPosition

Item {
    id: root

    property alias model: grid.model
    readonly property alias contentY: grid.contentY
    readonly property alias count: grid.count
    property bool loading: false
    property string error: ""
    property string errorDetail: ""
    property int cardWidth: 240
    property int cardHeight: Math.round(cardWidth * 9 / 16)
    // Under the picture, for the name and date.
    property int textHeight: 48
    // The edges of the grid get at least half of it.
    property int cardSpacing: 12
    // In pixels per mouse wheel notch.
    property real wheelStep: 100
    // 0 means none is selected.
    property int selectedModId: 0

    signal retry()
    signal modClicked(int modId)

    function showRow(row) {
        wheel.stop()
        grid.positionViewAtIndex(row, GridView.Contain)
    }

    // The first mod on screen, and how far past its top the view is.
    function topMod() {
        const top = ScrollPosition.top(grid)
        const item = top.index >= 0 ? grid.itemAtIndex(top.index) : null

        return { modId: item ? item.modId : 0, offset: top.offset }
    }

    function restoreTop(row, offset) {
        wheel.stop()
        ScrollPosition.restore(grid, row, offset)
    }

    GridView {
        id: grid

        // Same as GridView's own column count, which ignores the margins.
        readonly property int columns: Math.max(1, Math.floor(width / cellWidth))

        anchors {
            top: parent.top
            bottom: parent.bottom
            left: parent.left
            right: scrollBar.left
        }
        topMargin: root.cardSpacing / 2
        clip: true
        boundsBehavior: Flickable.StopAtBounds

        // Build cards (and start their image downloads) a screen ahead of the
        // viewport. This also makes the view ask the model for the next page
        // about a screen before reaching the end, so scrolling never stalls.
        cacheBuffer: height

        // Each cell is a card plus half a gap on every side. Cell size never
        // depends on the content, so nothing moves while thumbnails load.
        cellWidth: root.cardWidth + root.cardSpacing
        cellHeight: root.cardHeight + root.textHeight + root.cardSpacing

        delegate: ModCard {
            required views
            required likes

            width: grid.cellWidth
            height: grid.cellHeight
            tileWidth: root.cardWidth
            tileHeight: root.cardHeight
            textHeight: root.textHeight
            // GridView lays columns out from the left; shift every card by half
            // the unused width so the block of columns is centred.
            horizontalOffset: Math.floor((grid.width - grid.columns * grid.cellWidth) / 2)
            selected: modId === root.selectedModId
            onClicked: root.modClicked(modId)
        }

        footer: LoadMoreFooter {
            width: grid.width
            loading: root.loading && grid.count > 0
            error: grid.count > 0 ? root.error : ""
            errorDetail: grid.count > 0 ? root.errorDetail : ""
            onRetry: root.retry()
        }

        ScrollBar.vertical: scrollBar

        // Emptied for a new listing, e.g. another category: start at the top.
        onCountChanged: {
            if (count === 0) {
                wheel.stop()
                positionViewAtBeginning()
            }
        }
    }

    SmoothWheel {
        id: wheel

        anchors.fill: grid
        flickable: grid
        step: root.wheelStep
    }

    ScrollBar {
        id: scrollBar

        anchors {
            top: parent.top
            bottom: parent.bottom
            right: parent.right
        }
        policy: ScrollBar.AlwaysOn
        onPressedChanged: if (pressed) wheel.stop()
    }
}
